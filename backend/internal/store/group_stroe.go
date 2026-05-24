package store

import (
	"context"

	"config-man/backend/model"
)

func (s *Store) ListGroups() []model.Group {
	groups, err := s.db.ListGroups(context.Background())
	if err != nil {
		return nil
	}
	for index := range groups {
		s.hydrateGroup(&groups[index])
	}
	return groups
}

func (s *Store) FindGroup(groupID string) (model.Group, bool) {
	group, ok, err := s.db.FindGroup(context.Background(), groupID)
	if err != nil || !ok {
		return model.Group{}, false
	}
	s.hydrateGroup(&group)
	return group, true
}

func (s *Store) GroupNameExists(name string) bool {
	exists, err := s.db.GroupNameExists(context.Background(), name)
	return err == nil && exists
}

func (s *Store) SaveGroup(group model.Group, audit model.AuditLog) error {
	return s.db.SaveGroup(context.Background(), group, audit)
}
func (s *Store) DeleteGroup(groupID string, audit model.AuditLog) error {
	return s.db.DeleteGroup(context.Background(), groupID, audit)
}

func (s *Store) AddGroupMember(groupID, userID string, role model.GroupRole, audit model.AuditLog) error {
	return s.db.AddGroupMember(context.Background(), groupID, userID, role, audit)
}

func (s *Store) RemoveGroupMember(groupID, userID string, audit model.AuditLog) error {
	return s.db.RemoveGroupMember(context.Background(), groupID, userID, audit)
}

func (s *Store) hydrateGroup(group *model.Group) {
	for index := range group.Members {
		member := &group.Members[index]
		user, ok := s.FindUserByID(member.ID)
		if !ok {
			continue
		}
		member.User = user
	}
	for index := range group.Projects {
		s.hydrateProjectMembers(&group.Projects[index])
	}
	group.MemberCount = len(group.Members)
	group.ProjectCount = len(group.Projects)
}
