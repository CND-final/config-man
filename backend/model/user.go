package model

type UserRole string

const (
	RoleSystemAdmin    UserRole = "system_admin"
	RoleProjectAdmin   UserRole = "project_admin"
	RoleUserGroupAdmin UserRole = "group_admin"
	RoleDeveloper      UserRole = "developer"
	RoleReviewer       UserRole = "reviewer"
	RoleViewer         UserRole = "viewer"
)

type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Role  UserRole `json:"role"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
