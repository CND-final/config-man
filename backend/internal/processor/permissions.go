package processor

import (
	"strings"

	"config-man/backend/model"
)

const demoPassword = "password"

func canWriteEnvironment(user model.User, environment string) bool {
	if user.Role == model.RoleSystemAdmin || user.Role == model.RoleProjectAdmin {
		return true
	}
	return user.Role == model.RoleDeveloper && strings.ToLower(environment) != "prod"
}

func canRevealSensitive(user model.User) bool {
	return user.Role == model.RoleSystemAdmin || user.Role == model.RoleProjectAdmin || user.Role == model.RoleDeveloper
}

func canCreateReview(user model.User) bool {
	return user.Role == model.RoleSystemAdmin ||
		user.Role == model.RoleProjectAdmin ||
		user.Role == model.RoleDeveloper ||
		user.Role == model.RoleReviewer
}

func canReview(user model.User) bool {
	return user.Role == model.RoleSystemAdmin || user.Role == model.RoleReviewer
}

func normalizeValueType(valueType string) string {
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

func isSupportedConfigFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "properties", "json", "yaml":
		return true
	default:
		return false
	}
}
