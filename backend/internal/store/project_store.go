package store

import (
	"context"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
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

func (s *Store) ensureDefaultProjectMembers() {
	projects := s.ListProjects()
	for _, project := range projects {
		if len(project.Members) > 0 {
			continue
		}
		members := s.defaultProjectMembers(project)
		if len(members) == 0 {
			continue
		}
		_ = s.SaveProjectMembers(project.ID, members, model.AuditLog{
			ID:           util.NewID("aud"),
			Actor:        "system",
			Action:       "project_members.backfill",
			ResourceType: "project",
			ResourceID:   project.ID,
			ProjectID:    project.ID,
			Metadata:     map[string]any{"memberCount": len(members)},
			CreatedAt:    time.Now().UTC(),
		})
	}
}

func (s *Store) defaultProjectMembers(project model.Project) []model.ProjectMember {
	if project.ID == "customer-portal" {
		return []model.ProjectMember{
			{User: model.User{ID: "paul"}, ProjectRole: model.RoleProjectMemberAdmin},
			{User: model.User{ID: "nora"}, ProjectRole: model.RoleProjectDeveloper},
			{User: model.User{ID: "rachel"}, ProjectRole: model.RoleProjectReviewer},
			{User: model.User{ID: "vincent"}, ProjectRole: model.RoleProjectViewer},
		}
	}
	return nil
}
