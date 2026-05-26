package dbstore

import (
	"context"
	"database/sql"

	"config-man/backend/model"
)

func (s *Store) SaveConfig(ctx context.Context, config model.Config, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := upsertConfigTx(ctx, tx, config); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) UpsertConfigs(ctx context.Context, configs []model.Config) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, config := range configs {
			if err := upsertConfigTx(ctx, tx, config); err != nil {
				return err
			}
		}
		return nil
	})
}

func upsertConfigTx(ctx context.Context, tx *sql.Tx, config model.Config) error {
	if config.ID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO configs (id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET project_id = EXCLUDED.project_id, name = EXCLUDED.name, description = EXCLUDED.description, source_type = EXCLUDED.source_type, source_id = EXCLUDED.source_id, prefix = EXCLUDED.prefix, sort_order = EXCLUDED.sort_order, updated_at = EXCLUDED.updated_at`, config.ID, config.ProjectID, config.Name, config.Description, config.SourceType, config.SourceID, config.Prefix, config.SortOrder, config.CreatedAt, config.UpdatedAt)
	return err
}

func (s *Store) ListConfigs(ctx context.Context, projectID string) ([]model.Config, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at FROM configs WHERE project_id = $1 ORDER BY sort_order ASC, name ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make([]model.Config, 0)
	for rows.Next() {
		config := model.Config{}
		if err := rows.Scan(&config.ID, &config.ProjectID, &config.Name, &config.Description, &config.SourceType, &config.SourceID, &config.Prefix, &config.SortOrder, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, rows.Err()
}

func (s *Store) FindConfig(ctx context.Context, projectID, configID string) (model.Config, bool, error) {
	config := model.Config{}
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at FROM configs WHERE project_id = $1 AND id = $2`, projectID, configID).Scan(&config.ID, &config.ProjectID, &config.Name, &config.Description, &config.SourceType, &config.SourceID, &config.Prefix, &config.SortOrder, &config.CreatedAt, &config.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.Config{}, false, nil
	}
	if err != nil {
		return model.Config{}, false, err
	}
	return config, true, nil
}

func (s *Store) ConfigNameExists(ctx context.Context, projectID, name string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM configs WHERE project_id = $1 AND LOWER(name) = LOWER($2))`, projectID, name).Scan(&exists)
	return exists, err
}
