package model

import (
	"regexp"
	"strings"
	"time"
)

type Config struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"projectId"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	SourceType  string        `json:"sourceType"`
	SourceID    string        `json:"sourceId,omitempty"`
	Prefix      string        `json:"prefix"`
	SortOrder   int           `json:"sortOrder"`
	Entries     []ConfigEntry `json:"entries,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type ConfigEntry struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"projectId"`
	Environment     string    `json:"environment"`
	Branch          string    `json:"branch"`
	ConfigID        string    `json:"configId"`
	Key             string    `json:"key"`
	Value           string    `json:"value"`
	ValueType       string    `json:"valueType"`
	IsSensitive     bool      `json:"isSensitive"`
	UpdatedBy       string    `json:"updatedBy"`
	Inherited       bool      `json:"inherited,omitempty"`
	Overridden      bool      `json:"overridden,omitempty"`
	SourceType      string    `json:"sourceType,omitempty"`
	SourceID        string    `json:"sourceId,omitempty"`
	SharedValue     string    `json:"sharedValue"`
	SharedSensitive bool      `json:"sharedSensitive,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ConfigVersion struct {
	ID            string    `json:"id"`
	ConfigEntryID string    `json:"configEntryId"`
	OldValue      *string   `json:"oldValue"`
	NewValue      string    `json:"newValue"`
	ChangedBy     string    `json:"changedBy"`
	ChangeReason  string    `json:"changeReason"`
	CreatedAt     time.Time `json:"createdAt"`
}

type ConfigRevisionEntry struct {
	ConfigID    string `json:"configId,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	IsSensitive bool   `json:"isSensitive"`
}

type ConfigRevision struct {
	ID           string                `json:"id"`
	ProjectID    string                `json:"projectId"`
	Environment  string                `json:"environment"`
	Branch       string                `json:"branch"`
	Entries      []ConfigRevisionEntry `json:"entries"`
	ChangedBy    string                `json:"changedBy"`
	ChangeReason string                `json:"changeReason"`
	CreatedAt    time.Time             `json:"createdAt"`
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

func ConfigID(projectID, name string) string {
	return "cfg-" + slug(projectID) + "-" + slug(NormalizeConfigName(name))
}

func NormalizeConfigName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".properties") {
		return name
	}
	return name + ".yaml"
}

func ConfigPrefix(name string) string {
	name = NormalizeConfigName(name)
	lower := strings.ToLower(name)
	for _, suffix := range []string{".properties", ".yaml", ".yml", ".json"} {
		lower = strings.TrimSuffix(lower, suffix)
	}
	return strings.TrimSpace(lower)
}

func standardConfig(projectID, name, description, prefix string, sortOrder int, now time.Time) Config {
	return Config{
		ID:          ConfigID(projectID, name),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		SourceType:  "standard",
		Prefix:      prefix,
		SortOrder:   sortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "default"
	}
	return value
}
