package model

import "time"

type ProjectEnvironment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

type Project struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Description   string               `json:"description,omitempty"`
	RepoURL       string               `json:"repoUrl,omitempty"`
	OwnerName     string               `json:"ownerName"`
	DefaultFormat string               `json:"defaultFormat"`
	Environments  []ProjectEnvironment `json:"environments"`
	ConfigCount   int                  `json:"configCount"`
	CreatedAt     time.Time            `json:"createdAt"`
	UpdatedAt     time.Time            `json:"updatedAt"`
}
