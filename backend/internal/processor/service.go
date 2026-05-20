package processor

import (
	"strings"
	"time"

	appctx "config-man/backend/internal/context"
	"config-man/backend/internal/logger"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

const demoPassword = "password"

func (p *Processor) Login(req model.LoginRequest) (model.AuthResponse, *model.ErrorDetail) {
	email := strings.TrimSpace(req.Email)
	logger.Auth.Info("login attempt", "operation", "auth.login", "email", email)

	user, ok := p.store.FindUserByEmail(email)
	if !ok || req.Password != demoPassword {
		logger.Auth.Warn("login failed", "operation", "auth.login", "email", email)
		return model.AuthResponse{}, model.Unauthorized("Invalid email or password")
	}

	logger.Auth.Info("login succeeded",
		"operation", "auth.login",
		logger.FieldUserID, user.ID,
		logger.FieldRole, string(user.Role),
	)
	return model.AuthResponse{Token: user.ID, User: user}, nil
}

func (p *Processor) AuthenticateToken(token string) (appctx.RequestContext, *model.ErrorDetail) {
	token = strings.TrimSpace(token)
	user, ok := p.store.FindUserByID(token)
	if !ok {
		logger.Auth.Warn("authentication failed", "operation", "auth.authenticate")
		return appctx.RequestContext{}, model.Unauthorized("Missing or invalid login token")
	}

	logger.Auth.Info("authentication succeeded",
		"operation", "auth.authenticate",
		logger.FieldUserID, user.ID,
		logger.FieldRole, string(user.Role),
	)
	return appctx.RequestContext{Actor: user}, nil
}

func (p *Processor) Templates(ctx appctx.RequestContext) []model.Template {
	logger.Template.Info("templates listed",
		"operation", "template.list",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
	)
	templates := model.InfrastructureTemplates()
	templates = append(templates, p.store.ListTemplates(ctx.Actor.ID)...)
	return templates
}

func (p *Processor) BaseTemplate(ctx appctx.RequestContext) model.Template {
	logger.Template.Info("base template loaded",
		"operation", "template.base",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
	)
	return model.BaseTemplate()
}

func (p *Processor) CreateTemplate(ctx appctx.RequestContext, req model.CreateTemplateRequest) (model.Template, *model.ErrorDetail) {
	name := strings.TrimSpace(req.Name)
	format := strings.ToLower(strings.TrimSpace(req.Format))
	body := req.Body
	logger.Template.Info("create template requested",
		"operation", "template.create",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		"template_name", name,
		"format", format,
	)

	if len(name) < 2 {
		return model.Template{}, model.InvalidInput("template name is required")
	}
	if format == "" {
		format = "yaml"
	}
	if !util.IsSupportedConfigFormat(format) {
		return model.Template{}, model.InvalidInput("format must be properties, json, or yaml")
	}
	if strings.TrimSpace(body) == "" {
		return model.Template{}, model.InvalidInput("template body is required")
	}

	now := time.Now().UTC()
	template := model.Template{
		ID:          util.NewID("tpl"),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Format:      format,
		Body:        body,
		Variables:   model.ExtractTemplateVariables(body),
		OwnerUserID: ctx.Actor.ID,
		IsCustom:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := p.store.SaveTemplate(template, newAudit(ctx.ActorName(), "template.create", "template", template.ID, "", map[string]any{
		"name":   template.Name,
		"format": template.Format,
	})); err != nil {
		logger.Template.Error("create template persistence failed", "operation", "template.create", "error", err)
		return model.Template{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Template.Info("template created",
		"operation", "template.create",
		logger.FieldUserID, ctx.Actor.ID,
		"template_id", template.ID,
		"variable_count", len(template.Variables),
	)
	return template, nil
}

func (p *Processor) findAccessibleTemplate(ctx appctx.RequestContext, templateID string) (model.Template, bool) {
	for _, template := range model.InfrastructureTemplates() {
		if template.ID == templateID && template.Body != "" {
			return template, true
		}
	}
	return p.store.FindTemplate(ctx.Actor.ID, templateID)
}
