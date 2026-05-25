package util

import (
	"strings"

	"config-man/backend/model"
)

func CanRegisterUser(user model.User) bool {
	return user.Role == model.RoleSystemAdmin
}

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

func GroupMemberExists(members []model.GroupMember, userID string) bool {
	for _, member := range members {
		if member.ID == userID {
			return true
		}
	}
	return false
}

func GroupHasMember(group model.Group, userID string) bool {
	return GroupMemberExists(group.Members, userID)
}

func CanReadGroup(user model.User, group model.Group) bool {
	return user.Role == model.RoleSystemAdmin || GroupHasMember(group, user.ID)
}

func CanManageGroup(user model.User, group model.Group) bool {
	if user.Role == model.RoleSystemAdmin {
		return true
	}
	for _, member := range group.Members {
		if member.ID != user.ID {
			continue
		}
		if member.GroupRole == model.RoleGroupAdmin || user.Role == model.RoleUserGroupAdmin {
			return true
		}
	}
	return false
}

func ValidGroupRole(role model.GroupRole) bool {
	return role == model.RoleGroupAdmin || role == model.RoleGroupMember
}

func ProjectRoleForUser(user model.User, members []model.ProjectMember) (model.ProjectRole, bool) {
	for _, member := range members {
		if member.ID == user.ID {
			return member.ProjectRole, true
		}
	}
	return "", false
}

func CanReadProject(user model.User, members []model.ProjectMember) bool {
	if user.Role == model.RoleSystemAdmin {
		return true
	}
	_, ok := ProjectRoleForUser(user, members)
	return ok
}

func CanRevealProjectSensitive(user model.User, members []model.ProjectMember) bool {
	if user.Role == model.RoleSystemAdmin {
		return true
	}
	role, ok := ProjectRoleForUser(user, members)
	if !ok {
		return false
	}
	return role == model.RoleProjectMemberAdmin || role == model.RoleProjectDeveloper
}

func CanWriteProjectEnvironment(user model.User, members []model.ProjectMember, environment string) bool {
	if user.Role == model.RoleSystemAdmin {
		return true
	}
	role, ok := ProjectRoleForUser(user, members)
	if !ok {
		return false
	}
	if role == model.RoleProjectMemberAdmin {
		return true
	}
	return role == model.RoleProjectDeveloper && strings.ToLower(environment) != "prod"
}

func CanCreateProjectReview(user model.User, members []model.ProjectMember) bool {
	if user.Role == model.RoleSystemAdmin {
		return true
	}
	role, ok := ProjectRoleForUser(user, members)
	if !ok {
		return false
	}
	return role == model.RoleProjectMemberAdmin || role == model.RoleProjectDeveloper || role == model.RoleProjectReviewer
}

func CanReviewProject(user model.User, members []model.ProjectMember) bool {
	if user.Role == model.RoleSystemAdmin {
		return true
	}
	role, ok := ProjectRoleForUser(user, members)
	return ok && role == model.RoleProjectReviewer
}

func ValidProjectRole(role model.ProjectRole) bool {
	switch role {
	case model.RoleProjectMemberAdmin, model.RoleProjectDeveloper, model.RoleProjectReviewer, model.RoleProjectViewer:
		return true
	default:
		return false
	}
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
	case "yaml", "yml":
		return "yaml"
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
