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
	for index := range projects {
		s.hydrateProjectMembers(&projects[index])
	}
	return projects
}

func (s *Store) FindProject(projectID string) (model.Project, bool) {
	project, ok, err := s.db.FindProject(context.Background(), projectID)
	if err != nil {
		return model.Project{}, false
	}
	if ok {
		s.hydrateProjectMembers(&project)
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

func (s *Store) SaveProjectMembers(projectID string, members []model.ProjectMember, audit model.AuditLog) error {
	return s.db.SaveProjectMembers(context.Background(), projectID, members, audit)
}

func (s *Store) hydrateProjectMembers(project *model.Project) {
	for index := range project.Members {
		member := &project.Members[index]
		user, ok := s.FindUserByID(member.ID)
		if !ok {
			continue
		}
		member.User = user
	}
	project.MemberCount = len(project.Members)
}
