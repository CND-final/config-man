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
	logger.Project.Info("projects listed")
	return projects
}

func (p *Processor) CreateProject(ctx appctx.RequestContext, req model.CreateProjectRequest) (model.Project, *model.ErrorDetail) {
	name := strings.TrimSpace(req.Name)
	owner := strings.TrimSpace(req.OwnerName)
	logger.Project.Info("create project requested")

	if !util.CanRegisterProject(ctx.Actor) {
		logger.Project.Warn("create project denied")
		return model.Project{}, model.Forbidden("Only system_admin or project_admin can register projects")
	}

	if len(name) < 2 || len(owner) < 2 {
		logger.Project.Warn("create project invalid")
		return model.Project{}, model.InvalidInput("name and ownerName are required")
	}

	format := strings.TrimSpace(req.DefaultFormat)
	if format == "" {
		format = "yaml"
	}
	if !util.IsSupportedConfigFormat(format) {
		logger.Project.Warn("create project invalid")
		return model.Project{}, model.InvalidInput("defaultFormat must be properties, json, or yaml")
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

	envs := util.NormalizeEnvironments(req.Environments)
	now := time.Now().UTC()
	project := model.Project{
		ID:            util.NewID("prj"),
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		RepoURL:       strings.TrimSpace(req.RepoURL),
		OwnerName:     owner,
		DefaultFormat: format,
		TemplateID:    templateID,
		Environments:  make([]model.ProjectEnvironment, 0, len(envs)),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	for index, env := range envs {
		project.Environments = append(project.Environments, model.ProjectEnvironment{
			ID:        util.NewID("env"),
			Name:      env,
			SortOrder: index + 1,
		})
	}

	if err := p.store.SaveProject(project, newAudit(ctx.ActorName(), "project.create", "project", project.ID, project.ID, map[string]any{
		"name":         project.Name,
		"environments": envs,
		"templateId":   project.TemplateID,
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
