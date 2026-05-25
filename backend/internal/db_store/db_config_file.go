package dbstore

import (
	"context"
	"database/sql"
	"time"

	"config-man/backend/model"
)

func (s *Store) SaveConfigFile(ctx context.Context, file model.ConfigFile, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := upsertConfigFileTx(ctx, tx, file); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) UpsertConfigFiles(ctx context.Context, files []model.ConfigFile) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, file := range files {
			if err := upsertConfigFileTx(ctx, tx, file); err != nil {
				return err
			}
		}
		return nil
	})
}

func upsertConfigFileTx(ctx context.Context, tx *sql.Tx, file model.ConfigFile) error {
	if file.ID == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO config_files (id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET project_id = EXCLUDED.project_id, name = EXCLUDED.name, description = EXCLUDED.description, source_type = EXCLUDED.source_type, source_id = EXCLUDED.source_id, prefix = EXCLUDED.prefix, sort_order = EXCLUDED.sort_order, updated_at = EXCLUDED.updated_at`, file.ID, file.ProjectID, file.Name, file.Description, file.SourceType, file.SourceID, file.Prefix, file.SortOrder, file.CreatedAt, file.UpdatedAt)
	return err
}

func (s *Store) ListConfigFiles(ctx context.Context, projectID string) ([]model.ConfigFile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at FROM config_files WHERE project_id = $1 ORDER BY sort_order ASC, name ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]model.ConfigFile, 0)
	for rows.Next() {
		file := model.ConfigFile{}
		if err := rows.Scan(&file.ID, &file.ProjectID, &file.Name, &file.Description, &file.SourceType, &file.SourceID, &file.Prefix, &file.SortOrder, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Store) FindConfigFile(ctx context.Context, projectID, fileID string) (model.ConfigFile, bool, error) {
	file := model.ConfigFile{}
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at FROM config_files WHERE project_id = $1 AND id = $2`, projectID, fileID).Scan(&file.ID, &file.ProjectID, &file.Name, &file.Description, &file.SourceType, &file.SourceID, &file.Prefix, &file.SortOrder, &file.CreatedAt, &file.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.ConfigFile{}, false, nil
	}
	if err != nil {
		return model.ConfigFile{}, false, err
	}
	return file, true, nil
}

func (s *Store) ConfigFileNameExists(ctx context.Context, projectID, name string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM config_files WHERE project_id = $1 AND LOWER(name) = LOWER($2))`, projectID, name).Scan(&exists)
	return exists, err
}

func (s *Store) AssignMissingConfigFileIDs(ctx context.Context, projectID string) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, key FROM config_entries WHERE project_id = $1 AND config_file_id = ''`, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type assignment struct{ id, fileID string }
	assignments := make([]assignment, 0)
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return err
		}
		assignments = append(assignments, assignment{id: id, fileID: model.StandardConfigFileForKey(projectID, key, time.Now().UTC()).ID})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(assignments) == 0 {
		return nil
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, item := range assignments {
			if _, err := tx.ExecContext(ctx, `UPDATE config_entries SET config_file_id = $1 WHERE id = $2`, item.fileID, item.id); err != nil {
				return err
			}
		}
		return nil
	})
}
