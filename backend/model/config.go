package model

import "time"

type ConfigEntry struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Environment string    `json:"environment"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"valueType"`
	IsSensitive bool      `json:"isSensitive"`
	UpdatedBy   string    `json:"updatedBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ConfigVersion struct {
	ID           string    `json:"id"`
	ConfigID     string    `json:"configId"`
	OldValue     *string   `json:"oldValue"`
	NewValue     string    `json:"newValue"`
	ChangedBy    string    `json:"changedBy"`
	ChangeReason string    `json:"changeReason"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AuditLog struct {
	ID           string         `json:"id"`
	Actor        string         `json:"actor"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId,omitempty"`
	ProjectID    string         `json:"projectId,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}
