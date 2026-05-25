package processor

import (
	"strings"

	"golang.org/x/crypto/bcrypt"

	appctx "config-man/backend/internal/context"
	"config-man/backend/internal/logger"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

var validUserRoles = map[model.UserRole]bool{
	model.RoleSystemAdmin:    true,
	model.RoleProjectAdmin:   true,
	model.RoleUserGroupAdmin: true,
	model.RoleDeveloper:      true,
	model.RoleReviewer:       true,
	model.RoleViewer:         true,
}

func (p *Processor) RegisterUser(ctx appctx.RequestContext, req model.CreateUserRequest) (model.User, *model.ErrorDetail) {
	logger.Auth.Info("register user requested")

	// a. Permission check
	if !util.CanRegisterUser(ctx.Actor) {
		return model.User{}, model.Forbidden("Only system_admin can register users")
	}

	// b. Validate fields
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)
	role := model.UserRole(strings.TrimSpace(req.Role))

	if email == "" || !strings.Contains(email, "@") {
		return model.User{}, model.InvalidInput("valid email is required")
	}
	if len(name) < 2 {
		return model.User{}, model.InvalidInput("name must be at least 2 characters")
	}
	if !validUserRoles[role] {
		return model.User{}, model.InvalidInput("role must be one of: system_admin, project_admin, group_admin, developer, reviewer, viewer")
	}
	if len(req.Password) < 8 {
		return model.User{}, model.InvalidInput("password must be at least 8 characters")
	}

	// c. Email uniqueness — processor-layer check for a friendly error message;
	// the DB UNIQUE constraint is the final safety net for concurrent requests.
	if _, exists := p.store.FindUserByEmail(email); exists {
		return model.User{}, model.InvalidInput("email already registered")
	}

	// d. Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Auth.Error("bcrypt hash failed during registration")
		return model.User{}, model.InternalError("failed to process password")
	}

	// e. Assemble user (PasswordHash carries json:"-" so it is never serialized;
	// created_at/updated_at are stamped by the DB layer)
	user := model.User{
		ID:           util.NewID("usr"),
		Email:        email,
		Name:         name,
		Role:         role,
		PasswordHash: string(hash),
	}

	// f. Persist user + audit log in one transaction
	audit := newAudit(ctx.ActorName(), "user.register", "user", user.ID, "", map[string]any{
		"email": user.Email,
		"name":  user.Name,
		"role":  string(user.Role),
	})
	if err := p.store.CreateUser(user, audit); err != nil {
		logger.Auth.Error("register user persistence failed")
		return model.User{}, model.InternalError("database persistence failed: " + err.Error())
	}

	// g. Return user; PasswordHash is omitted from JSON via json:"-"
	logger.Auth.Info("user registered")
	return user, nil
}
