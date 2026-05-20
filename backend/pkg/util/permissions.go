package util

import (
	"strings"

	"config-man/backend/model"
)

func CanRegisterProject(user model.User) bool {
	return user.Role == model.RoleSystemAdmin || user.Role == model.RoleProjectAdmin
}

func CanWriteEnvironment(user model.User, environment string) bool {
	if user.Role == model.RoleSystemAdmin || user.Role == model.RoleProjectAdmin {
		return true
	}
	return user.Role == model.RoleDeveloper && strings.ToLower(environment) != "prod"
}

func CanRevealSensitive(user model.User) bool {
	return user.Role == model.RoleSystemAdmin || user.Role == model.RoleProjectAdmin || user.Role == model.RoleDeveloper
}

func CanCreateReview(user model.User) bool {
	return user.Role == model.RoleSystemAdmin ||
		user.Role == model.RoleProjectAdmin ||
		user.Role == model.RoleDeveloper ||
		user.Role == model.RoleReviewer
}

func CanReview(user model.User) bool {
	return user.Role == model.RoleSystemAdmin || user.Role == model.RoleReviewer
}

func NormalizeValueType(valueType string) string {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "", "string":
		return "string"
	case "number":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "json":
		return "json"
	default:
		return ""
	}
}

func IsSupportedConfigFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "properties", "json", "yaml":
		return true
	default:
		return false
	}
}
