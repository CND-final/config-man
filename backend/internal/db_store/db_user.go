package dbstore

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"config-man/backend/model"
)

// HasUsers reports whether the users table contains at least one row.
// Used on startup to decide whether to insert seed users.
func (s *Store) HasUsers(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users LIMIT 1)`).Scan(&exists)
	return exists, err
}

// SaveSeedUsers inserts the initial demo users with NULL password_hash so that
// the demo-password fallback path in Login remains active for them.
// ON CONFLICT (id) DO NOTHING ensures re-runs are idempotent.
func (s *Store) SaveSeedUsers(ctx context.Context, users []model.User) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		for _, u := range users {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO users (id, email, name, role, password_hash, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, NULL, $5, $6)
				 ON CONFLICT (id) DO NOTHING`,
				u.ID, u.Email, u.Name, string(u.Role), now, now,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveUser inserts a new user with a bcrypt password_hash.
// Timestamps are computed here; the application layer assembles all other fields.
func (s *Store) SaveUser(ctx context.Context, user model.User) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, role, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.Name, string(user.Role), user.PasswordHash, now, now,
	)
	return err
}

// FindUserByEmail looks up a user by email (case-insensitive) and populates
// PasswordHash so the caller (Login) can perform bcrypt comparison.
// Returns (user, true, nil) on success; (zero, false, nil) if not found.
// Unexpected DB errors are logged and returned to the caller.
func (s *Store) FindUserByEmail(ctx context.Context, email string) (model.User, bool, error) {
	var u model.User
	var hashNull sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, role, password_hash
		 FROM users WHERE lower(email) = lower($1)`,
		email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &hashNull)
	if err == sql.ErrNoRows {
		return model.User{}, false, nil
	}
	if err != nil {
		slog.Error("db: FindUserByEmail failed", "error", err)
		return model.User{}, false, err
	}
	if hashNull.Valid {
		u.PasswordHash = hashNull.String
	}
	return u, true, nil
}

// FindUserByID looks up a user by their ID for auth-token validation.
// Does NOT select password_hash; it is not needed outside of Login.
// Unexpected DB errors are logged and returned to the caller.
func (s *Store) FindUserByID(ctx context.Context, id string) (model.User, bool, error) {
	var u model.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, role FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role)
	if err == sql.ErrNoRows {
		return model.User{}, false, nil
	}
	if err != nil {
		slog.Error("db: FindUserByID failed", "error", err)
		return model.User{}, false, err
	}
	return u, true, nil
}

// ListUsers returns all users sorted by name, without password_hash.
func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, name, role FROM users ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
