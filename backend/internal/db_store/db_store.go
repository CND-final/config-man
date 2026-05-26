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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(742983621)); err != nil {
		return err
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		)`,
		`ALTER TABLE groups DROP COLUMN IF EXISTS description`,
		`ALTER TABLE groups DROP COLUMN IF EXISTS created_at`,
		`ALTER TABLE groups DROP COLUMN IF EXISTS updated_at`,
		`CREATE TABLE IF NOT EXISTS group_members (
			group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY(group_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			repo_url TEXT NOT NULL DEFAULT '',
			template_id TEXT NOT NULL DEFAULT '',
			group_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS template_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS group_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE projects DROP COLUMN IF EXISTS owner_name`,
		`ALTER TABLE projects DROP COLUMN IF EXISTS default_format`,
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
		`CREATE TABLE IF NOT EXISTS project_members (
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY(project_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS shared_configs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL,
			scope_id TEXT NOT NULL DEFAULT '',
			scope_name TEXT NOT NULL DEFAULT '',
			format TEXT NOT NULL DEFAULT 'yaml',
			inherited_by INTEGER NOT NULL DEFAULT 0,
			affected_projects JSONB NOT NULL DEFAULT '[]'::jsonb,
			updated_by TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS shared_config_entries (
			shared_config_id TEXT NOT NULL REFERENCES shared_configs(id) ON DELETE CASCADE,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			value_type TEXT NOT NULL DEFAULT 'string',
			environment TEXT NOT NULL DEFAULT '',
			is_sensitive BOOLEAN NOT NULL DEFAULT false,
			sort_order INTEGER NOT NULL,
			PRIMARY KEY(shared_config_id, key, environment)
		)`,
		`CREATE TABLE IF NOT EXISTS shared_config_update_requests (
			id TEXT PRIMARY KEY,
			shared_config_id TEXT NOT NULL REFERENCES shared_configs(id) ON DELETE CASCADE,
			shared_config_name TEXT NOT NULL,
			scope TEXT NOT NULL,
			scope_id TEXT NOT NULL DEFAULT '',
			requester TEXT NOT NULL,
			status TEXT NOT NULL,
			reason TEXT NOT NULL,
			proposed_config JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS configs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			source_type TEXT NOT NULL DEFAULT 'custom',
			source_id TEXT NOT NULL DEFAULT '',
			prefix TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 100,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE(project_id, name)
		)`,
		`DO $$
		BEGIN
			IF to_regclass('public.config_files') IS NOT NULL THEN
				INSERT INTO configs (id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at)
				SELECT id, project_id, name, description, source_type, source_id, prefix, sort_order, created_at, updated_at
				FROM config_files
				ON CONFLICT (id) DO NOTHING;
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS config_entries (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			environment TEXT NOT NULL,
			config_id TEXT NOT NULL DEFAULT '',
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			value_type TEXT NOT NULL,
			is_sensitive BOOLEAN NOT NULL DEFAULT false,
			updated_by TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE(project_id, environment, key)
		)`,
		`ALTER TABLE config_entries ADD COLUMN IF NOT EXISTS config_id TEXT NOT NULL DEFAULT ''`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'config_entries' AND column_name = 'config_file_id'
			) THEN
				UPDATE config_entries SET config_id = config_file_id WHERE config_id = '' AND config_file_id <> '';
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS config_versions (
			id TEXT PRIMARY KEY,
			config_entry_id TEXT NOT NULL DEFAULT '',
			old_value TEXT,
			new_value TEXT NOT NULL,
			changed_by TEXT NOT NULL,
			change_reason TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`ALTER TABLE config_versions ADD COLUMN IF NOT EXISTS config_entry_id TEXT NOT NULL DEFAULT ''`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'config_versions' AND column_name = 'config_id'
			) THEN
				UPDATE config_versions SET config_entry_id = config_id WHERE config_entry_id = '' AND config_id <> '';
			END IF;
		END $$`,
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
		`CREATE TABLE IF NOT EXISTS app_notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			read BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL
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
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
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
