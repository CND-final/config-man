package model

import "time"

type ReviewRequest struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	ProjectName string    `json:"projectName"`
	Environment string    `json:"environment"`
	ConfigKey   string    `json:"configKey,omitempty"`
	Requester   string    `json:"requester"`
	Reviewer    string    `json:"reviewer,omitempty"`
	Status      string    `json:"status"`
	Reason      string    `json:"reason"`
	Comment     string    `json:"comment,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
