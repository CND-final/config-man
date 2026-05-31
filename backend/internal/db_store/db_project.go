package dbstore

import (
	"context"
	"database/sql"
	"time"

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
	if _, err := tx.ExecContext(ctx, `INSERT INTO projects (id, name, description, repo_url, template_id, group_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, repo_url = EXCLUDED.repo_url, template_id = EXCLUDED.template_id, group_id = EXCLUDED.group_id, updated_at = EXCLUDED.updated_at`, project.ID, project.Name, project.Description, project.RepoURL, project.TemplateID, project.GroupID, project.CreatedAt, project.UpdatedAt); err != nil {
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_branches WHERE project_id = $1`, project.ID); err != nil {
		return err
	}
	for _, branch := range project.Branches {
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_branches (id, project_id, name, sort_order) VALUES ($1,$2,$3,$4)`, branch.ID, project.ID, branch.Name, branch.SortOrder); err != nil {
			return err
		}
	}
	if project.Members != nil {
		if err := replaceProjectMembersTx(ctx, tx, project.ID, project.Members); err != nil {
			return err
		}
	}
	return nil
}

func replaceProjectMembersTx(ctx context.Context, tx *sql.Tx, projectID string, members []model.ProjectMember) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_members WHERE project_id = $1`, projectID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, member := range members {
		role := member.ProjectRole
		if role == "" {
			role = model.RoleProjectViewer
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO project_members (project_id, user_id, role, created_at) VALUES ($1,$2,$3,$4)`, projectID, member.ID, string(role), now); err != nil {
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
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, repo_url, template_id, group_id, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		project := model.Project{}
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.RepoURL, &project.TemplateID, &project.GroupID, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, err
		}
		project.Environments, err = s.listProjectEnvironments(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		project.Branches, err = s.listProjectBranches(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		project.Members, err = s.listProjectMembers(ctx, project.ID)
		if err != nil {
			return nil, err
		}
		project.MemberCount = len(project.Members)
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
	err := s.db.QueryRowContext(ctx, `SELECT id, name, description, repo_url, template_id, group_id, created_at, updated_at FROM projects WHERE id = $1`, projectID).Scan(&project.ID, &project.Name, &project.Description, &project.RepoURL, &project.TemplateID, &project.GroupID, &project.CreatedAt, &project.UpdatedAt)
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
	project.Branches, err = s.listProjectBranches(ctx, project.ID)
	if err != nil {
		return model.Project{}, false, err
	}
	project.Members, err = s.listProjectMembers(ctx, project.ID)
	if err != nil {
		return model.Project{}, false, err
	}
	project.MemberCount = len(project.Members)
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

func (s *Store) listProjectBranches(ctx context.Context, projectID string) ([]model.ProjectBranch, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, sort_order FROM project_branches WHERE project_id = $1 ORDER BY sort_order ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	branches := make([]model.ProjectBranch, 0)
	for rows.Next() {
		branch := model.ProjectBranch{}
		if err := rows.Scan(&branch.ID, &branch.Name, &branch.SortOrder); err != nil {
			return nil, err
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		branches = append(branches, model.ProjectBranch{ID: projectID + "-default", Name: "default", SortOrder: 1})
	}
	return branches, nil
}

func (s *Store) listProjectMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, role FROM project_members WHERE project_id = $1 ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]model.ProjectMember, 0)
	for rows.Next() {
		member := model.ProjectMember{}
		if err := rows.Scan(&member.ID, &member.ProjectRole); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (s *Store) SaveProjectMembers(ctx context.Context, projectID string, members []model.ProjectMember, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := replaceProjectMembersTx(ctx, tx, projectID, members); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) configCount(ctx context.Context, projectID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM config_entries WHERE project_id = $1`, projectID).Scan(&count)
	return count, err
}
