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

// CreateUser persists a newly registered user (with bcrypt hash already set)
// and records an audit log in the same transaction.
func (s *Store) CreateUser(user model.User, audit model.AuditLog) error {
	return s.db.SaveUser(context.Background(), user, audit)
}
