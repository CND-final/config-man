package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"

	"config-man/backend/model"
)

func insertAuditTx(ctx context.Context, tx *sql.Tx, audit model.AuditLog) error {
	if audit.ID == "" {
		return nil
	}
	metadata, err := encodeMetadata(audit.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_logs (id, actor, action, resource_type, resource_id, project_id, metadata, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, audit.ID, audit.Actor, audit.Action, audit.ResourceType, audit.ResourceID, audit.ProjectID, metadata, audit.CreatedAt)
	return err
}

func insertAuditSeedTx(ctx context.Context, tx *sql.Tx, audit model.AuditLog) error {
	if audit.ID == "" {
		return nil
	}
	metadata, err := encodeMetadata(audit.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_logs (id, actor, action, resource_type, resource_id, project_id, metadata, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`, audit.ID, audit.Actor, audit.Action, audit.ResourceType, audit.ResourceID, audit.ProjectID, metadata, audit.CreatedAt)
	return err
}

func encodeMetadata(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(metadata)
}
