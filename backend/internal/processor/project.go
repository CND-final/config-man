package processor

import (
	appctx "config-man/backend/internal/context"
	"fmt"
	"strings"
	"time"

	"config-man/backend/internal/logger"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) ListProjects(ctx appctx.RequestContext) []model.Project {
	projects := p.store.ListProjects()
	if ctx.Actor.Role == model.RoleSystemAdmin {
		logger.Project.Info("projects listed")
		return projects
	}
	visible := make([]model.Project, 0)
	for _, project := range projects {
		if util.CanReadProject(ctx.Actor, project.Members) {
			visible = append(visible, project)
		}
	}
	logger.Project.Info("projects listed")
	return visible
}

func (p *Processor) CreateProject(ctx appctx.RequestContext, req model.CreateProjectRequest) (model.Project, *model.ErrorDetail) {
	name := strings.TrimSpace(req.Name)
	logger.Project.Info("create project requested")

	if !util.CanRegisterProject(ctx.Actor) {
		logger.Project.Warn("create project denied")
		return model.Project{}, model.Forbidden("Only system_admin, group_admin, or project_admin can register projects")
	}

	if len(name) < 2 {
		logger.Project.Warn("create project invalid")
		return model.Project{}, model.InvalidInput("name is required")
	}
	if p.store.ProjectNameExists(name) {
		logger.Project.Warn("create project conflict")
		return model.Project{}, model.Conflict(fmt.Sprintf("Project %q already exists", name))
	}

	templateID := strings.TrimSpace(req.TemplateID)
	if templateID != "" {
		if _, ok := p.findAccessibleTemplate(ctx, templateID); !ok {
			logger.Project.Warn("create project invalid")
			return model.Project{}, model.NotFound(fmt.Sprintf("Template %q not found", templateID))
		}
	}
	groupID := strings.TrimSpace(req.GroupID)
	if groupID == "" {
		logger.Project.Warn("create project invalid")
		return model.Project{}, model.InvalidInput("groupId is required")
	}
	group, groupErr := p.requireGroup(groupID)
	if groupErr != nil {
		logger.Project.Warn("create project invalid")
		return model.Project{}, groupErr
	}
	if !util.CanManageGroup(ctx.Actor, group) {
		logger.Project.Warn("create project denied")
		return model.Project{}, model.Forbidden("You cannot assign projects to this group")
	}

	envs := util.NormalizeEnvironments(req.Environments)
	now := time.Now().UTC()
	project := model.Project{
		ID:           util.NewID("prj"),
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		RepoURL:      strings.TrimSpace(req.RepoURL),
		TemplateID:   templateID,
		GroupID:      groupID,
		Environments: make([]model.ProjectEnvironment, 0, len(envs)),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	for index, env := range envs {
		project.Environments = append(project.Environments, model.ProjectEnvironment{
			ID:        util.NewID("env"),
			Name:      env,
			SortOrder: index + 1,
		})
	}
	admin, adminErr := p.defaultProjectAdmin(ctx, group)
	if adminErr != nil {
		logger.Project.Warn("create project invalid")
		return model.Project{}, adminErr
	}
	project.Members = []model.ProjectMember{{User: admin, ProjectRole: model.RoleProjectMemberAdmin}}
	project.MemberCount = len(project.Members)

	if err := p.store.SaveProject(project, newAudit(ctx.ActorName(), "project.create", "project", project.ID, project.ID, map[string]any{
		"name":         project.Name,
		"environments": envs,
		"templateId":   project.TemplateID,
		"groupId":      project.GroupID,
	})); err != nil {
		logger.Project.Error("create project persistence failed")
		return model.Project{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Project.Info("project created")
	return project, nil
}

func (p *Processor) GetProject(ctx appctx.RequestContext, projectID string) (model.Project, *model.ErrorDetail) {
	project, err := p.requireProject(projectID)
	if err != nil {
		logger.Project.Warn("get project failed")
		return model.Project{}, err
	}
	if !util.CanReadProject(ctx.Actor, project.Members) {
		logger.Project.Warn("get project denied")
		return model.Project{}, model.Forbidden("You cannot view this project")
	}
	logger.Project.Info("project loaded")
	return project, nil
}

func (p *Processor) requireProject(projectID string) (model.Project, *model.ErrorDetail) {
	project, ok := p.store.FindProject(projectID)
	if !ok {
		return model.Project{}, model.NotFound(fmt.Sprintf("Project %q not found", projectID))
	}
	return project, nil
}

func (p *Processor) requireEnvironment(projectID, environment string) *model.ErrorDetail {
	project, err := p.requireProject(projectID)
	if err != nil {
		return err
	}
	for _, env := range project.Environments {
		if env.Name == environment {
			return nil
		}
	}
	return model.NotFound(fmt.Sprintf("Environment %q not found for project %q", environment, projectID))
}

func (p *Processor) requireReadableProject(ctx appctx.RequestContext, projectID string) (model.Project, *model.ErrorDetail) {
	project, err := p.requireProject(projectID)
	if err != nil {
		return model.Project{}, err
	}
	if !util.CanReadProject(ctx.Actor, project.Members) {
		return model.Project{}, model.Forbidden("You cannot view this project")
	}
	return project, nil
}

func (p *Processor) requireWritableProjectEnvironment(ctx appctx.RequestContext, projectID, environment string) (model.Project, *model.ErrorDetail) {
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		return model.Project{}, err
	}
	if !util.CanWriteProjectEnvironment(ctx.Actor, project.Members, environment) {
		return model.Project{}, model.Forbidden("You cannot modify this project environment")
	}
	return project, nil
}

func (p *Processor) ListProjectMembers(ctx appctx.RequestContext, projectID string) ([]model.ProjectMember, *model.ErrorDetail) {
	project, err := p.requireProject(projectID)
	if err != nil {
		return nil, err
	}
	if !util.CanReadProject(ctx.Actor, project.Members) {
		return nil, model.Forbidden("You cannot view this project")
	}
	return project.Members, nil
}

func (p *Processor) UpdateProjectMembers(ctx appctx.RequestContext, projectID string, req model.UpdateProjectMembersRequest) ([]model.ProjectMember, *model.ErrorDetail) {
	project, err := p.requireProject(projectID)
	if err != nil {
		return nil, err
	}
	group, groupErr := p.requireGroup(project.GroupID)
	if groupErr != nil {
		return nil, groupErr
	}
	if ctx.Actor.Role != model.RoleSystemAdmin {
		role, ok := util.ProjectRoleForUser(ctx.Actor, project.Members)
		canManageProject := ok && role == model.RoleProjectMemberAdmin
		if !canManageProject && !util.CanManageGroup(ctx.Actor, group) {
			return nil, model.Forbidden("Only system_admin, group_admin, or project_admin can edit project members")
		}
	}

	members, appErr := p.projectMembersFromRequests(req.Members, group)
	if appErr != nil {
		return nil, appErr
	}
	if !projectHasProjectAdmin(members) {
		return nil, model.InvalidInput("At least one project_admin is required")
	}
	if err := p.store.SaveProjectMembers(project.ID, members, newAudit(ctx.ActorName(), "project_members.update", "project", project.ID, project.ID, map[string]any{"memberCount": len(members)})); err != nil {
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}
	updated, _ := p.store.FindProject(project.ID)
	return updated.Members, nil
}

func (p *Processor) projectMembersFromRequests(requests []model.ProjectMemberRequest, group model.Group) ([]model.ProjectMember, *model.ErrorDetail) {
	seen := map[string]bool{}
	groupMemberIDs := groupMemberIDSet(group)
	members := make([]model.ProjectMember, 0, len(requests))
	for _, request := range requests {
		userID := strings.TrimSpace(request.UserID)
		if userID == "" || seen[userID] {
			continue
		}
		if !groupMemberIDs[userID] {
			return nil, model.InvalidInput(fmt.Sprintf("User %q is not a member of project group %q", userID, group.ID))
		}
		role := request.ProjectRole
		if role == "" {
			role = model.RoleProjectViewer
		}
		if !util.ValidProjectRole(role) {
			return nil, model.InvalidInput("projectRole must be project_admin, developer, reviewer, or viewer")
		}
		user, ok := p.store.FindUserByID(userID)
		if !ok {
			return nil, model.NotFound(fmt.Sprintf("User %q not found", userID))
		}
		seen[userID] = true
		members = append(members, model.ProjectMember{User: user, ProjectRole: role})
	}
	return members, nil
}

func (p *Processor) defaultProjectAdmin(ctx appctx.RequestContext, group model.Group) (model.User, *model.ErrorDetail) {
	for _, member := range group.Members {
		if member.ID == ctx.Actor.ID {
			user, ok := p.store.FindUserByID(member.ID)
			if ok {
				return user, nil
			}
		}
	}
	for _, member := range group.Members {
		if member.GroupRole == model.RoleGroupAdmin {
			user, ok := p.store.FindUserByID(member.ID)
			if ok {
				return user, nil
			}
		}
	}
	for _, member := range group.Members {
		user, ok := p.store.FindUserByID(member.ID)
		if ok {
			return user, nil
		}
	}
	return model.User{}, model.InvalidInput("Project group must have at least one member")
}

func groupMemberIDSet(group model.Group) map[string]bool {
	ids := make(map[string]bool, len(group.Members))
	for _, member := range group.Members {
		ids[member.ID] = true
	}
	return ids
}

func projectHasProjectAdmin(members []model.ProjectMember) bool {
	for _, member := range members {
		if member.ProjectRole == model.RoleProjectMemberAdmin {
			return true
		}
	}
	return false
}
