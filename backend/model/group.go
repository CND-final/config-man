package model

type GroupRole string

const (
	RoleGroupAdmin  GroupRole = "group_admin"
	RoleGroupMember GroupRole = "member"
)

type Group struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Members      []GroupMember `json:"members"`
	Projects     []Project     `json:"projects"`
	MemberCount  int           `json:"memberCount"`
	ProjectCount int           `json:"projectCount"`
}

type GroupMember struct {
	User
	GroupRole GroupRole `json:"groupRole"`
}
