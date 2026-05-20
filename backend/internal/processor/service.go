package processor

import (
	"strings"

	appctx "config-man/backend/internal/context"
	"config-man/backend/internal/logger"
	"config-man/backend/model"
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

func (p *Processor) BaseTemplate(ctx appctx.RequestContext) model.Template {
	logger.Template.Info("base template loaded",
		"operation", "template.base",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
	)
	return model.BaseTemplate()
}
