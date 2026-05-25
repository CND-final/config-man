package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"config-man/backend/model"
)

func (s *Store) SaveConfigChanges(ctx context.Context, entries []model.ConfigEntry, versions []model.ConfigVersion, revision model.ConfigRevision, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, entry := range entries {
			if err := upsertConfigEntryTx(ctx, tx, entry); err != nil {
				return err
			}
		}
		for _, version := range versions {
			if err := insertConfigVersionTx(ctx, tx, version); err != nil {
				return err
			}
		}
		if err := insertConfigRevisionTx(ctx, tx, revision); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) DeleteConfig(ctx context.Context, configID string, revision model.ConfigRevision, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := deleteConfigEntryTx(ctx, tx, configID); err != nil {
			return err
		}
		if err := insertConfigRevisionTx(ctx, tx, revision); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) RestoreConfig(ctx context.Context, upserts []model.ConfigEntry, deleteIDs []string, versions []model.ConfigVersion, revision model.ConfigRevision, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, configID := range deleteIDs {
			if err := deleteConfigEntryTx(ctx, tx, configID); err != nil {
				return err
			}
		}
		for _, entry := range upserts {
			if err := upsertConfigEntryTx(ctx, tx, entry); err != nil {
				return err
			}
		}
		for _, version := range versions {
			if err := insertConfigVersionTx(ctx, tx, version); err != nil {
				return err
			}
		}
		if err := insertConfigRevisionTx(ctx, tx, revision); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func upsertConfigEntryTx(ctx context.Context, tx *sql.Tx, entry model.ConfigEntry) error {
	if entry.ID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO config_entries (id, project_id, environment, config_file_id, key, value, value_type, is_sensitive, updated_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO UPDATE SET project_id = EXCLUDED.project_id, environment = EXCLUDED.environment, config_file_id = EXCLUDED.config_file_id, key = EXCLUDED.key, value = EXCLUDED.value, value_type = EXCLUDED.value_type, is_sensitive = EXCLUDED.is_sensitive, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at`, entry.ID, entry.ProjectID, entry.Environment, entry.ConfigFileID, entry.Key, entry.Value, entry.ValueType, entry.IsSensitive, entry.UpdatedBy, entry.CreatedAt, entry.UpdatedAt)
	return err
}

func deleteConfigEntryTx(ctx context.Context, tx *sql.Tx, configID string) error {
	if configID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM config_entries WHERE id = $1`, configID)
	return err
}

func insertConfigVersionTx(ctx context.Context, tx *sql.Tx, version model.ConfigVersion) error {
	if version.ID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO config_versions (id, config_id, old_value, new_value, changed_by, change_reason, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, version.ID, version.ConfigID, version.OldValue, version.NewValue, version.ChangedBy, version.ChangeReason, version.CreatedAt)
	return err
}

func insertConfigVersionSeedTx(ctx context.Context, tx *sql.Tx, version model.ConfigVersion) error {
	if version.ID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO config_versions (id, config_id, old_value, new_value, changed_by, change_reason, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`, version.ID, version.ConfigID, version.OldValue, version.NewValue, version.ChangedBy, version.ChangeReason, version.CreatedAt)
	return err
}

func insertConfigRevisionTx(ctx context.Context, tx *sql.Tx, revision model.ConfigRevision) error {
	if revision.ID == "" {
		return nil
	}
	entries, err := json.Marshal(revision.Entries)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (id, project_id, environment, entries, changed_by, change_reason, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, configRevisionTable), revision.ID, revision.ProjectID, revision.Environment, entries, revision.ChangedBy, revision.ChangeReason, revision.CreatedAt)
	return err
}

func insertConfigRevisionSeedTx(ctx context.Context, tx *sql.Tx, revision model.ConfigRevision) error {
	if revision.ID == "" {
		return nil
	}
	entries, err := json.Marshal(revision.Entries)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (id, project_id, environment, entries, changed_by, change_reason, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`, configRevisionTable), revision.ID, revision.ProjectID, revision.Environment, entries, revision.ChangedBy, revision.ChangeReason, revision.CreatedAt)
	return err
}

func (s *Store) ListConfigEntries(ctx context.Context, projectID, environment string) ([]model.ConfigEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, environment, config_file_id, key, value, value_type, is_sensitive, updated_by, created_at, updated_at FROM config_entries WHERE project_id = $1 AND environment = $2 ORDER BY key ASC`, projectID, environment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]model.ConfigEntry, 0)
	for rows.Next() {
		entry := model.ConfigEntry{}
		if err := rows.Scan(&entry.ID, &entry.ProjectID, &entry.Environment, &entry.ConfigFileID, &entry.Key, &entry.Value, &entry.ValueType, &entry.IsSensitive, &entry.UpdatedBy, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) FindConfig(ctx context.Context, projectID, configID string) (model.ConfigEntry, bool, error) {
	entry := model.ConfigEntry{}
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, environment, config_file_id, key, value, value_type, is_sensitive, updated_by, created_at, updated_at FROM config_entries WHERE project_id = $1 AND id = $2`, projectID, configID).Scan(&entry.ID, &entry.ProjectID, &entry.Environment, &entry.ConfigFileID, &entry.Key, &entry.Value, &entry.ValueType, &entry.IsSensitive, &entry.UpdatedBy, &entry.CreatedAt, &entry.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.ConfigEntry{}, false, nil
	}
	if err != nil {
		return model.ConfigEntry{}, false, err
	}
	return entry, true, nil
}

func (s *Store) FindConfigByKey(ctx context.Context, projectID, environment, key string) (model.ConfigEntry, bool, error) {
	entry := model.ConfigEntry{}
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, environment, config_file_id, key, value, value_type, is_sensitive, updated_by, created_at, updated_at FROM config_entries WHERE project_id = $1 AND environment = $2 AND key = $3`, projectID, environment, key).Scan(&entry.ID, &entry.ProjectID, &entry.Environment, &entry.ConfigFileID, &entry.Key, &entry.Value, &entry.ValueType, &entry.IsSensitive, &entry.UpdatedBy, &entry.CreatedAt, &entry.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.ConfigEntry{}, false, nil
	}
	if err != nil {
		return model.ConfigEntry{}, false, err
	}
	return entry, true, nil
}

func (s *Store) ListConfigVersions(ctx context.Context, configID string) ([]model.ConfigVersion, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, config_id, old_value, new_value, changed_by, change_reason, created_at FROM config_versions WHERE config_id = $1 ORDER BY created_at DESC`, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]model.ConfigVersion, 0)
	for rows.Next() {
		var oldValue sql.NullString
		version := model.ConfigVersion{}
		if err := rows.Scan(&version.ID, &version.ConfigID, &oldValue, &version.NewValue, &version.ChangedBy, &version.ChangeReason, &version.CreatedAt); err != nil {
			return nil, err
		}
		if oldValue.Valid {
			value := oldValue.String
			version.OldValue = &value
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) ListConfigRevisions(ctx context.Context, projectID, environment string) ([]model.ConfigRevision, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, project_id, environment, entries, changed_by, change_reason, created_at FROM %s WHERE project_id = $1 AND environment = $2 ORDER BY created_at DESC`, configRevisionTable), projectID, environment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	revisions := make([]model.ConfigRevision, 0)
	for rows.Next() {
		revision := model.ConfigRevision{}
		var entriesBytes []byte
		if err := rows.Scan(&revision.ID, &revision.ProjectID, &revision.Environment, &entriesBytes, &revision.ChangedBy, &revision.ChangeReason, &revision.CreatedAt); err != nil {
			return nil, err
		}
		if len(entriesBytes) > 0 {
			_ = json.Unmarshal(entriesBytes, &revision.Entries)
		}
		revisions = append(revisions, revision)
	}
	return revisions, rows.Err()
}
