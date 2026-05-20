package processor

import (
	"strings"

	"config-man/backend/internal/apperror"
	appctx "config-man/backend/internal/context"
	"config-man/backend/model"
)

const demoPassword = "password"

func (p *Processor) Login(req model.LoginRequest) (model.AuthResponse, *apperror.AppError) {
	user, ok := p.store.FindUserByEmail(req.Email)
	if !ok || req.Password != demoPassword {
		return model.AuthResponse{}, apperror.Unauthorized("Invalid email or password")
	}
	return model.AuthResponse{Token: user.ID, User: user}, nil
}

func (p *Processor) AuthenticateToken(token string) (appctx.RequestContext, *apperror.AppError) {
	user, ok := p.store.FindUserByID(strings.TrimSpace(token))
	if !ok {
		return appctx.RequestContext{}, apperror.Unauthorized("Missing or invalid login token")
	}
	return appctx.RequestContext{Actor: user}, nil
}

func (p *Processor) BaseTemplate(_ appctx.RequestContext) model.Template {
	return model.BaseTemplate()
}
