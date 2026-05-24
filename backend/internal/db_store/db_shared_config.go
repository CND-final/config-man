package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"

	"config-man/backend/model"
)

func (s *Store) HasSharedConfigs(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shared_configs`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) ListSharedConfigs(ctx context.Context) ([]model.SharedConfig, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, scope, scope_id, scope_name, format, inherited_by, affected_projects, updated_by, created_at, updated_at FROM shared_configs ORDER BY scope ASC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.SharedConfig, 0)
	for rows.Next() {
		item, err := scanSharedConfig(rows)
		if err != nil {
			return nil, err
		}
		item.Entries, err = s.listSharedConfigEntries(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FindSharedConfig(ctx context.Context, id string) (model.SharedConfig, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, description, scope, scope_id, scope_name, format, inherited_by, affected_projects, updated_by, created_at, updated_at FROM shared_configs WHERE id = $1`, id)
	item, err := scanSharedConfig(row)
	if err == sql.ErrNoRows {
		return model.SharedConfig{}, false, nil
	}
	if err != nil {
		return model.SharedConfig{}, false, err
	}
	item.Entries, err = s.listSharedConfigEntries(ctx, item.ID)
	if err != nil {
		return model.SharedConfig{}, false, err
	}
	return item, true, nil
}

func scanSharedConfig(scanner interface{ Scan(dest ...any) error }) (model.SharedConfig, error) {
	item := model.SharedConfig{}
	var affected []byte
	if err := scanner.Scan(&item.ID, &item.Name, &item.Description, &item.Scope, &item.ScopeID, &item.ScopeName, &item.Format, &item.InheritedBy, &affected, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.SharedConfig{}, err
	}
	_ = json.Unmarshal(affected, &item.AffectedProjects)
	return item, nil
}

func (s *Store) listSharedConfigEntries(ctx context.Context, id string) ([]model.SharedConfigEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value, value_type, environment, is_sensitive FROM shared_config_entries WHERE shared_config_id = $1 ORDER BY sort_order ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]model.SharedConfigEntry, 0)
	for rows.Next() {
		entry := model.SharedConfigEntry{}
		if err := rows.Scan(&entry.Key, &entry.Value, &entry.ValueType, &entry.Environment, &entry.IsSensitive); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) SaveSharedConfig(ctx context.Context, item model.SharedConfig, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := upsertSharedConfigTx(ctx, tx, item); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func upsertSharedConfigTx(ctx context.Context, tx *sql.Tx, item model.SharedConfig) error {
	affected, err := json.Marshal(item.AffectedProjects)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO shared_configs (id, name, description, scope, scope_id, scope_name, format, inherited_by, affected_projects, updated_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, scope = EXCLUDED.scope, scope_id = EXCLUDED.scope_id, scope_name = EXCLUDED.scope_name, format = EXCLUDED.format, inherited_by = EXCLUDED.inherited_by, affected_projects = EXCLUDED.affected_projects, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at`, item.ID, item.Name, item.Description, string(item.Scope), item.ScopeID, item.ScopeName, item.Format, item.InheritedBy, affected, item.UpdatedBy, item.CreatedAt, item.UpdatedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM shared_config_entries WHERE shared_config_id = $1`, item.ID); err != nil {
		return err
	}
	for index, entry := range item.Entries {
		if _, err := tx.ExecContext(ctx, `INSERT INTO shared_config_entries (shared_config_id, key, value, value_type, environment, is_sensitive, sort_order) VALUES ($1,$2,$3,$4,$5,$6,$7)`, item.ID, entry.Key, entry.Value, entry.ValueType, entry.Environment, entry.IsSensitive, index+1); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteSharedConfig(ctx context.Context, id string, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM shared_configs WHERE id = $1`, id); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) SaveSharedConfigUpdateRequest(ctx context.Context, request model.SharedConfigUpdateRequest, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		proposed, err := json.Marshal(request.ProposedConfig)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO shared_config_update_requests (id, shared_config_id, shared_config_name, scope, scope_id, requester, status, reason, proposed_config, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, request.ID, request.SharedConfigID, request.SharedConfigName, string(request.Scope), request.ScopeID, request.Requester, request.Status, request.Reason, proposed, request.CreatedAt, request.UpdatedAt); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}
