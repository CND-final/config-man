package model

import "time"

type LibraryScope string

const (
	ScopeGlobal  LibraryScope = "global"
	ScopeGroup   LibraryScope = "group"
	ScopeProject LibraryScope = "project"
)

type SharedConfigEntry struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Environment string `json:"environment"`
	IsSensitive bool   `json:"isSensitive"`
}

type SharedConfig struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	Scope                LibraryScope        `json:"scope"`
	ScopeID              string              `json:"scopeId,omitempty"`
	ScopeName            string              `json:"scopeName,omitempty"`
	Format               string              `json:"format"`
	Entries              []SharedConfigEntry `json:"entries"`
	InheritedBy          int                 `json:"inheritedBy"`
	ProdEnvironmentCount int                 `json:"prodEnvironmentCount"`
	AffectedProjects     []string            `json:"affectedProjects"`
	UpdatedBy            string              `json:"updatedBy"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type SharedConfigUpdateRequest struct {
	ID               string       `json:"id"`
	SharedConfigID   string       `json:"sharedConfigId"`
	SharedConfigName string       `json:"sharedConfigName"`
	Scope            LibraryScope `json:"scope"`
	ScopeID          string       `json:"scopeId,omitempty"`
	Requester        string       `json:"requester"`
	Status           string       `json:"status"`
	Reason           string       `json:"reason"`
	ProposedConfig   SharedConfig `json:"proposedConfig"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

func SeedSharedConfigs(now time.Time) []SharedConfig {
	return []SharedConfig{
		{
			ID:          "global-runtime-defaults",
			Name:        "Global Runtime Defaults",
			Description: "Company-wide runtime defaults intended to be inherited by every service.",
			Scope:       ScopeGlobal,
			ScopeName:   "Global",
			Format:      "yaml",
			Entries: []SharedConfigEntry{
				{Key: "logging.level.root", Value: "DEBUG", ValueType: "string", Environment: "dev"},
				{Key: "logging.level.root", Value: "INFO", ValueType: "string", Environment: "staging"},
				{Key: "logging.level.root", Value: "INFO", ValueType: "string", Environment: "prod"},
				{Key: "observability.metrics.enabled", Value: "false", ValueType: "boolean", Environment: "dev"},
				{Key: "observability.metrics.enabled", Value: "true", ValueType: "boolean", Environment: "staging"},
				{Key: "observability.metrics.enabled", Value: "true", ValueType: "boolean", Environment: "prod"},
				{Key: "security.headers.enabled", Value: "false", ValueType: "boolean", Environment: "dev"},
				{Key: "security.headers.enabled", Value: "true", ValueType: "boolean", Environment: "staging"},
				{Key: "security.headers.enabled", Value: "true", ValueType: "boolean", Environment: "prod"},
			},
			InheritedBy:      2,
			AffectedProjects: []string{"customer-portal", "billing-api"},
			UpdatedBy:        "Alice Lin",
			UpdatedAt:        now,
		},
		{
			ID:          "platform-team-defaults",
			Name:        "Platform Team Defaults",
			Description: "Shared Platform Team settings for service endpoints and release controls.",
			Scope:       ScopeGroup,
			ScopeID:     "platform-team",
			ScopeName:   "Platform Team",
			Format:      "yaml",
			Entries: []SharedConfigEntry{
				{Key: "platform.api.baseUrl", Value: "https://api.example.com", ValueType: "string", Environment: "prod"},
				{Key: "release.approval.required", Value: "true", ValueType: "boolean", Environment: "prod"},
			},
			InheritedBy:      1,
			AffectedProjects: []string{"customer-portal"},
			UpdatedBy:        "Grace Huang",
			UpdatedAt:        now,
		},
		{
			ID:          "commerce-team-defaults",
			Name:        "Commerce Team Defaults",
			Description: "Shared Commerce Team settings for payment and reconciliation services.",
			Scope:       ScopeGroup,
			ScopeID:     "commerce-team",
			ScopeName:   "Commerce Team",
			Format:      "yaml",
			Entries: []SharedConfigEntry{
				{Key: "payment.reconciliation.enabled", Value: "true", ValueType: "boolean", Environment: "prod"},
				{Key: "release.approval.required", Value: "true", ValueType: "boolean", Environment: "prod"},
			},
			InheritedBy:      1,
			AffectedProjects: []string{"billing-api"},
			UpdatedBy:        "Grace Huang",
			UpdatedAt:        now,
		},
	}
}
