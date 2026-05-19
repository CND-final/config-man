package processor

import (
	"net/http"
	"strings"

	"config-man/backend/model"
)

func (p *Processor) Login(req model.LoginRequest) (model.AuthResponse, *AppError) {
	user, ok := p.store.FindUserByEmail(req.Email)
	if !ok || req.Password != demoPassword {
		return model.AuthResponse{}, unauthorized("Invalid email or password")
	}
	return model.AuthResponse{Token: user.ID, User: user}, nil
}

func (p *Processor) RequireUser(r *http.Request) (model.User, *AppError) {
	token := strings.TrimSpace(r.Header.Get("X-Actor"))
	if token == "" {
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			token = strings.TrimSpace(authorization[7:])
		} else {
			token = authorization
		}
	}

	user, ok := p.store.FindUserByID(token)
	if !ok {
		return model.User{}, unauthorized("Missing or invalid login token")
	}
	return user, nil
}

func (p *Processor) BaseTemplate() model.Template {
	return model.BaseTemplate()
}

func (p *Processor) ListProjects() []model.Project {
	return p.store.ListProjects()
}

func (p *Processor) CreateProject(req model.CreateProjectRequest, actor string) (model.Project, *AppError) {
	return p.store.CreateProject(req, actor)
}

func (p *Processor) GetProject(projectID string) (model.Project, *AppError) {
	return p.store.GetProject(projectID)
}

func (p *Processor) ListConfigs(user model.User, projectID, environment string, revealSensitive bool) (map[string]any, *AppError) {
	return p.store.ListConfigs(user, projectID, environment, revealSensitive)
}

func (p *Processor) CreateConfig(user model.User, projectID string, req model.CreateConfigRequest) (model.ConfigEntry, *AppError) {
	return p.store.CreateConfig(user, projectID, req)
}

func (p *Processor) UpdateConfig(user model.User, projectID, configID string, req model.UpdateConfigRequest) (model.ConfigEntry, *AppError) {
	return p.store.UpdateConfig(user, projectID, configID, req)
}

func (p *Processor) DeleteConfig(user model.User, projectID, configID string) (map[string]any, *AppError) {
	return p.store.DeleteConfig(user, projectID, configID)
}

func (p *Processor) ImportConfigs(user model.User, projectID string, req model.ImportConfigRequest) (map[string]any, *AppError) {
	return p.store.ImportConfigs(user, projectID, req)
}

func (p *Processor) ListReviewRequests(projectID string, filters model.ReviewFilters) ([]model.ReviewRequest, *AppError) {
	return p.store.ListReviewRequests(projectID, filters)
}

func (p *Processor) CreateReviewRequest(user model.User, req model.CreateReviewRequest) (model.ReviewRequest, *AppError) {
	return p.store.CreateReviewRequest(user, req)
}

func (p *Processor) SetReviewStatus(user model.User, requestID, status, comment string) (model.ReviewRequest, *AppError) {
	return p.store.SetReviewStatus(user, requestID, status, comment)
}

func (p *Processor) ValidateProject(projectID string, req model.ValidateProjectRequest) (model.ValidationResult, *AppError) {
	return p.store.ValidateProject(projectID, req)
}
