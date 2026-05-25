package store

import (
	"context"

	"config-man/backend/model"
)

// ListUsers returns all users sorted by name. DB errors return nil.
func (s *Store) ListUsers() []model.User {
	users, err := s.db.ListUsers(context.Background())
	if err != nil {
		return nil
	}
	return users
}

// CreateUser persists a newly registered user (with bcrypt hash already set).
func (s *Store) CreateUser(user model.User) error {
	return s.db.SaveUser(context.Background(), user)
}
