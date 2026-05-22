package dbstore

import (
	"context"
	"database/sql"
	"fmt"
)

const configRevisionTable = "config_revisions"

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) InitSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			repo_url TEXT NOT NULL DEFAULT '',
			owner_name TEXT NOT NULL,
			default_format TEXT NOT NULL DEFAULT 'yaml',
			template_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS template_id TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS custom_templates (
			id TEXT PRIMARY KEY,
			owner_user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			format TEXT NOT NULL,
			body TEXT NOT NULL,
			variables JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_environments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			sort_order INTEGER NOT NULL,
			UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS config_entries (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			environment TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			value_type TEXT NOT NULL,
			is_sensitive BOOLEAN NOT NULL DEFAULT false,
			updated_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE(project_id, environment, key)
		)`,
		`CREATE TABLE IF NOT EXISTS config_versions (
			id TEXT PRIMARY KEY,
			config_id TEXT NOT NULL,
			old_value TEXT,
			new_value TEXT NOT NULL,
			changed_by TEXT NOT NULL,
			change_reason TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			environment TEXT NOT NULL,
			entries JSONB NOT NULL DEFAULT '[]'::jsonb,
			changed_by TEXT NOT NULL,
			change_reason TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`, configRevisionTable),
		`CREATE TABLE IF NOT EXISTS review_requests (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			project_name TEXT NOT NULL,
			environment TEXT NOT NULL,
			config_key TEXT NOT NULL DEFAULT '',
			requester TEXT NOT NULL,
			reviewer TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			reason TEXT NOT NULL,
			comment TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL DEFAULT '',
			project_id TEXT NOT NULL DEFAULT '',
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL
		)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
