package store

import (
	"context"

	"config-man/backend/model"
)

func (s *Store) SaveNotifications(notifications []model.Notification) error {
	return s.db.SaveNotifications(context.Background(), notifications)
}

func (s *Store) ListNotifications(userID string) []model.Notification {
	notifications, err := s.db.ListNotifications(context.Background(), userID)
	if err != nil {
		return nil
	}
	return notifications
}
