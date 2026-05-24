package dbstore

import (
	"context"
	"database/sql"

	"config-man/backend/model"
)

func (s *Store) SaveNotifications(ctx context.Context, notifications []model.Notification) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, notification := range notifications {
			if _, err := tx.ExecContext(ctx, `INSERT INTO app_notifications (id, user_id, title, message, read, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, notification.ID, notification.UserID, notification.Title, notification.Message, notification.Read, notification.CreatedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListNotifications(ctx context.Context, userID string) ([]model.Notification, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, title, message, read, created_at FROM app_notifications WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := make([]model.Notification, 0)
	for rows.Next() {
		notification := model.Notification{}
		if err := rows.Scan(&notification.ID, &notification.UserID, &notification.Title, &notification.Message, &notification.Read, &notification.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}
