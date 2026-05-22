package dbstore

import (
	"context"
	"database/sql"

	"config-man/backend/model"
)

func (s *Store) SaveProject(ctx context.Context, project model.Project, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := upsertProjectTx(ctx, tx, project); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func upsertProjectTx(ctx context.Context, tx *sql.Tx, project model.Project) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO projects (id, name, description, repo_url, owner_name, default_format, template_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, repo_url = EXCLUDED.repo_url, owner_name = EXCLUDED.owner_name, default_format = EXCLUDED.default_format, template_id = EXCLUDED.template_id, updated_at = EXCLUDED.updated_at`, project.ID, project.Name, project.Description, project.RepoURL, project.OwnerName, project.DefaultFormat, project.TemplateID, project.CreatedAt, project.UpdatedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_environments WHERE project_id = $1`, project.ID); err != nil {
		return err
	}
	for _, env := range project.Environments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_environments (id, project_id, name, sort_order) VALUES ($1,$2,$3,$4)`, env.ID, project.ID, env.Name, env.SortOrder); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) HasProjects(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, repo_url, owner_name, default_format, template_id, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		project := model.Project{}
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.RepoURL, &project.OwnerName, &project.DefaultFormat, &project.TemplateID, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, err
		}
		project.Environments, err = s.listProjectEnvironments(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		project.ConfigCount, err = s.configCount(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (s *Store) FindProject(ctx context.Context, projectID string) (model.Project, bool, error) {
	project := model.Project{}
	err := s.db.QueryRowContext(ctx, `SELECT id, name, description, repo_url, owner_name, default_format, template_id, created_at, updated_at FROM projects WHERE id = $1`, projectID).Scan(&project.ID, &project.Name, &project.Description, &project.RepoURL, &project.OwnerName, &project.DefaultFormat, &project.TemplateID, &project.CreatedAt, &project.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.Project{}, false, nil
	}
	if err != nil {
		return model.Project{}, false, err
	}
	project.Environments, err = s.listProjectEnvironments(ctx, project.ID)
	if err != nil {
		return model.Project{}, false, err
	}
	project.ConfigCount, err = s.configCount(ctx, project.ID)
	if err != nil {
		return model.Project{}, false, err
	}
	return project, true, nil
}

func (s *Store) ProjectNameExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE lower(name) = lower($1))`, name).Scan(&exists)
	return exists, err
}

func (s *Store) listProjectEnvironments(ctx context.Context, projectID string) ([]model.ProjectEnvironment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, sort_order FROM project_environments WHERE project_id = $1 ORDER BY sort_order ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	envs := make([]model.ProjectEnvironment, 0)
	for rows.Next() {
		env := model.ProjectEnvironment{}
		if err := rows.Scan(&env.ID, &env.Name, &env.SortOrder); err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, rows.Err()
}

func (s *Store) configCount(ctx context.Context, projectID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM config_entries WHERE project_id = $1`, projectID).Scan(&count)
	return count, err
}
