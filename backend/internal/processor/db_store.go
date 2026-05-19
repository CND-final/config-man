package processor

import (
	"context"
	"database/sql"
	"encoding/json"

	"config-man/backend/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewStoreWithDatabase(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	store := newStoreBase(db)
	if err := store.initSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.loadSnapshot(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if len(store.projects) == 0 {
		store.seedDemoData()
		if err := store.persistSnapshotLocked(ctx); err != nil {
			db.Close()
			return nil, err
		}
	}
	return store, nil
}

func (s *Store) persistLocked() *AppError {
	if s.db == nil {
		return nil
	}
	if err := s.persistSnapshotLocked(context.Background()); err != nil {
		return internalError("database persistence failed: " + err.Error())
	}
	return nil
}

func (s *Store) initSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			repo_url TEXT NOT NULL DEFAULT '',
			owner_name TEXT NOT NULL,
			default_format TEXT NOT NULL DEFAULT 'yaml',
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

func (s *Store) loadSnapshot(ctx context.Context) error {
	if err := s.loadProjects(ctx); err != nil {
		return err
	}
	if err := s.loadProjectEnvironments(ctx); err != nil {
		return err
	}
	if err := s.loadConfigEntries(ctx); err != nil {
		return err
	}
	if err := s.loadConfigVersions(ctx); err != nil {
		return err
	}
	if err := s.loadReviewRequests(ctx); err != nil {
		return err
	}
	return s.loadAuditLogs(ctx)
}

func (s *Store) loadProjects(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, repo_url, owner_name, default_format, created_at, updated_at FROM projects`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		project := &model.Project{}
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.RepoURL, &project.OwnerName, &project.DefaultFormat, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return err
		}
		s.projects[project.ID] = project
	}
	return rows.Err()
}

func (s *Store) loadProjectEnvironments(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, name, sort_order FROM project_environments ORDER BY sort_order ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var projectID string
		env := model.ProjectEnvironment{}
		if err := rows.Scan(&env.ID, &projectID, &env.Name, &env.SortOrder); err != nil {
			return err
		}
		if project := s.projects[projectID]; project != nil {
			project.Environments = append(project.Environments, env)
		}
	}
	return rows.Err()
}

func (s *Store) loadConfigEntries(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, environment, key, value, value_type, is_sensitive, updated_by, created_at, updated_at FROM config_entries`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		entry := &model.ConfigEntry{}
		if err := rows.Scan(&entry.ID, &entry.ProjectID, &entry.Environment, &entry.Key, &entry.Value, &entry.ValueType, &entry.IsSensitive, &entry.UpdatedBy, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return err
		}
		s.configs[entry.ID] = entry
	}
	return rows.Err()
}

func (s *Store) loadConfigVersions(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, config_id, old_value, new_value, changed_by, change_reason, created_at FROM config_versions ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var oldValue sql.NullString
		version := model.ConfigVersion{}
		if err := rows.Scan(&version.ID, &version.ConfigID, &oldValue, &version.NewValue, &version.ChangedBy, &version.ChangeReason, &version.CreatedAt); err != nil {
			return err
		}
		if oldValue.Valid {
			value := oldValue.String
			version.OldValue = &value
		}
		s.versions = append(s.versions, version)
	}
	return rows.Err()
}

func (s *Store) loadReviewRequests(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, project_name, environment, config_key, requester, reviewer, status, reason, comment, created_at, updated_at FROM review_requests`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		review := &model.ReviewRequest{}
		if err := rows.Scan(&review.ID, &review.ProjectID, &review.ProjectName, &review.Environment, &review.ConfigKey, &review.Requester, &review.Reviewer, &review.Status, &review.Reason, &review.Comment, &review.CreatedAt, &review.UpdatedAt); err != nil {
			return err
		}
		s.reviews[review.ID] = review
	}
	return rows.Err()
}

func (s *Store) loadAuditLogs(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, actor, action, resource_type, resource_id, project_id, metadata, created_at FROM audit_logs ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		audit := model.AuditLog{}
		var metadataBytes []byte
		if err := rows.Scan(&audit.ID, &audit.Actor, &audit.Action, &audit.ResourceType, &audit.ResourceID, &audit.ProjectID, &metadataBytes, &audit.CreatedAt); err != nil {
			return err
		}
		if len(metadataBytes) > 0 {
			_ = json.Unmarshal(metadataBytes, &audit.Metadata)
		}
		s.audits = append(s.audits, audit)
	}
	return rows.Err()
}

func (s *Store) persistSnapshotLocked(ctx context.Context) error {
	if s.db == nil {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		`DELETE FROM audit_logs`,
		`DELETE FROM review_requests`,
		`DELETE FROM config_versions`,
		`DELETE FROM config_entries`,
		`DELETE FROM project_environments`,
		`DELETE FROM projects`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	for _, project := range s.projects {
		if _, err := tx.ExecContext(ctx, `INSERT INTO projects (id, name, description, repo_url, owner_name, default_format, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, project.ID, project.Name, project.Description, project.RepoURL, project.OwnerName, project.DefaultFormat, project.CreatedAt, project.UpdatedAt); err != nil {
			return err
		}
		for _, env := range project.Environments {
			if _, err := tx.ExecContext(ctx, `INSERT INTO project_environments (id, project_id, name, sort_order) VALUES ($1,$2,$3,$4)`, env.ID, project.ID, env.Name, env.SortOrder); err != nil {
				return err
			}
		}
	}

	for _, entry := range s.configs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO config_entries (id, project_id, environment, key, value, value_type, is_sensitive, updated_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, entry.ID, entry.ProjectID, entry.Environment, entry.Key, entry.Value, entry.ValueType, entry.IsSensitive, entry.UpdatedBy, entry.CreatedAt, entry.UpdatedAt); err != nil {
			return err
		}
	}

	for _, version := range s.versions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO config_versions (id, config_id, old_value, new_value, changed_by, change_reason, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, version.ID, version.ConfigID, version.OldValue, version.NewValue, version.ChangedBy, version.ChangeReason, version.CreatedAt); err != nil {
			return err
		}
	}

	for _, review := range s.reviews {
		if _, err := tx.ExecContext(ctx, `INSERT INTO review_requests (id, project_id, project_name, environment, config_key, requester, reviewer, status, reason, comment, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, review.ID, review.ProjectID, review.ProjectName, review.Environment, review.ConfigKey, review.Requester, review.Reviewer, review.Status, review.Reason, review.Comment, review.CreatedAt, review.UpdatedAt); err != nil {
			return err
		}
	}

	for _, audit := range s.audits {
		metadata := []byte(`{}`)
		if audit.Metadata != nil {
			encoded, err := json.Marshal(audit.Metadata)
			if err != nil {
				return err
			}
			metadata = encoded
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (id, actor, action, resource_type, resource_id, project_id, metadata, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, audit.ID, audit.Actor, audit.Action, audit.ResourceType, audit.ResourceID, audit.ProjectID, metadata, audit.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}
