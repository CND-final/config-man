package processor

import (
	appctx "config-man/backend/internal/context"
	"fmt"
	"strings"
	"time"

	"config-man/backend/internal/apperror"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) ListProjects(_ appctx.RequestContext) []model.Project {
	return p.store.ListProjects()
}

func (p *Processor) CreateProject(ctx appctx.RequestContext, req model.CreateProjectRequest) (model.Project, *apperror.AppError) {
	if !util.CanRegisterProject(ctx.Actor) {
		return model.Project{}, apperror.Forbidden("Only system_admin or project_admin can register projects")
	}

	name := strings.TrimSpace(req.Name)
	owner := strings.TrimSpace(req.OwnerName)
	if len(name) < 2 || len(owner) < 2 {
		return model.Project{}, apperror.BadRequest("name and ownerName are required")
	}

	format := strings.TrimSpace(req.DefaultFormat)
	if format == "" {
		format = "yaml"
	}
	if !util.IsSupportedConfigFormat(format) {
		return model.Project{}, apperror.BadRequest("defaultFormat must be properties, json, or yaml")
	}
	if p.store.ProjectNameExists(name) {
		return model.Project{}, apperror.Conflict(fmt.Sprintf("Project %q already exists", name))
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
	})); err != nil {
		return model.Project{}, apperror.Internal("database persistence failed: " + err.Error())
	}
	return project, nil
}

func (p *Processor) GetProject(_ appctx.RequestContext, projectID string) (model.Project, *apperror.AppError) {
	project, err := p.requireProject(projectID)
	if err != nil {
		return model.Project{}, err
	}
	return project, nil
}

func (p *Processor) requireProject(projectID string) (model.Project, *apperror.AppError) {
	project, ok := p.store.FindProject(projectID)
	if !ok {
		return model.Project{}, apperror.NotFound(fmt.Sprintf("Project %q not found", projectID))
	}
	return project, nil
}

func (p *Processor) requireEnvironment(projectID, environment string) *apperror.AppError {
	project, err := p.requireProject(projectID)
	if err != nil {
		return err
	}
	for _, env := range project.Environments {
		if env.Name == environment {
			return nil
		}
	}
	return apperror.NotFound(fmt.Sprintf("Environment %q not found for project %q", environment, projectID))
}
