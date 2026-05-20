package processor

import (
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func newVersion(configID string, oldValue *string, newValue, changedBy, reason string) model.ConfigVersion {
	return model.ConfigVersion{
		ID:           util.NewID("ver"),
		ConfigID:     configID,
		OldValue:     oldValue,
		NewValue:     newValue,
		ChangedBy:    changedBy,
		ChangeReason: reason,
		CreatedAt:    time.Now().UTC(),
	}
}

func newAudit(actor, action, resourceType, resourceID, projectID string, metadata map[string]any) model.AuditLog {
	return model.AuditLog{
		ID:           util.NewID("aud"),
		Actor:        actor,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ProjectID:    projectID,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}
}
