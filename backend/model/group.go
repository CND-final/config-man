package model

type GroupRole string

const (
	RoleGroupAdmin  GroupRole = "group_admin"
	RoleGroupMember GroupRole = "member"
)

type Group struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Members     []GroupMember `json:"members"`
	MemberCount int           `json:"memberCount"`
}

type GroupMember struct {
	User
	GroupRole GroupRole `json:"groupRole"`
}
