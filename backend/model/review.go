package model

import "time"

type ReviewConfigChange struct {
	ConfigEntryID string  `json:"configEntryId,omitempty"`
	ConfigID      string  `json:"configId"`
	Key           string  `json:"key"`
	OldValue      *string `json:"oldValue,omitempty"`
	Value         string  `json:"value"`
	ValueType     string  `json:"valueType"`
	IsSensitive   bool    `json:"isSensitive"`
	Environment   string  `json:"environment"`
	Branch        string  `json:"branch"`
}

type ReviewRequest struct {
	ID              string               `json:"id"`
	ProjectID       string               `json:"projectId"`
	ProjectName     string               `json:"projectName"`
	Environment     string               `json:"environment"`
	Branch          string               `json:"branch"`
	ConfigKey       string               `json:"configKey,omitempty"`
	Requester       string               `json:"requester"`
	Reviewer        string               `json:"reviewer,omitempty"`
	Status          string               `json:"status"`
	Reason          string               `json:"reason"`
	Comment         string               `json:"comment,omitempty"`
	ProposedChanges []ReviewConfigChange `json:"proposedChanges,omitempty"`
	CreatedAt       time.Time            `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
}
