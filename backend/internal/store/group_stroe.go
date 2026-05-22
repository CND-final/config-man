package store

import (
	"sort"

	"config-man/backend/model"
)

func (s *Store) ListGroups() []model.Group {
	groups := make([]model.Group, 0, len(s.groups))
	for _, group := range s.groups {
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return groups
}

func (s *Store) FindGroup(groupID string) (model.Group, bool) {
	group, ok := s.groups[groupID]
	if !ok {
		return model.Group{}, false
	}
	return *group, true
}
