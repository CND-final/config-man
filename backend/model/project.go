package model

import "time"

type ProjectRole string

const (
	RoleProjectMemberAdmin ProjectRole = "project_admin"
	RoleProjectDeveloper   ProjectRole = "developer"
	RoleProjectReviewer    ProjectRole = "reviewer"
	RoleProjectViewer      ProjectRole = "viewer"
)

type ProjectEnvironment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

type ProjectMember struct {
	User
	ProjectRole ProjectRole `json:"projectRole"`
}

type Project struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	RepoURL      string               `json:"repoUrl,omitempty"`
	TemplateID   string               `json:"templateId,omitempty"`
	GroupID      string               `json:"groupId,omitempty"`
	Environments []ProjectEnvironment `json:"environments"`
	Members      []ProjectMember      `json:"members,omitempty"`
	MemberCount  int                  `json:"memberCount"`
	ConfigCount  int                  `json:"configCount"`
	CreatedAt    time.Time            `json:"createdAt"`
	UpdatedAt    time.Time            `json:"updatedAt"`
}
