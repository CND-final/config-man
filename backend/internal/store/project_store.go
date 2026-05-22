package store

import (
	"context"

	"config-man/backend/model"
)

func (s *Store) ListProjects() []model.Project {
	projects, err := s.db.ListProjects(context.Background())
	if err != nil {
		return nil
	}
	return projects
}

func (s *Store) FindProject(projectID string) (model.Project, bool) {
	project, ok, err := s.db.FindProject(context.Background(), projectID)
	if err != nil {
		return model.Project{}, false
	}
	return project, ok
}

func (s *Store) ProjectNameExists(name string) bool {
	exists, err := s.db.ProjectNameExists(context.Background(), name)
	return err == nil && exists
}

func (s *Store) SaveProject(project model.Project, audit model.AuditLog) error {
	return s.db.SaveProject(context.Background(), project, audit)
}
