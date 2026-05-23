package store

import (
	"sort"

	"config-man/backend/model"
)

func (s *Store) ListUsers() []model.User {
	users := make([]model.User, len(s.users))
	copy(users, s.users)
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})
	return users
}
