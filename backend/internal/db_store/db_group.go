package dbstore

import (
	"context"
	"database/sql"
	"time"

	"config-man/backend/model"
)

func (s *Store) ListGroups(ctx context.Context) ([]model.Group, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM groups ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]model.Group, 0)
	for rows.Next() {
		group := model.Group{}
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, err
		}
		group.Members, err = s.listGroupMembers(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		group.MemberCount = len(group.Members)
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (s *Store) FindGroup(ctx context.Context, groupID string) (model.Group, bool, error) {
	group := model.Group{}
	err := s.db.QueryRowContext(ctx, `SELECT id, name FROM groups WHERE id = $1`, groupID).Scan(&group.ID, &group.Name)
	if err == sql.ErrNoRows {
		return model.Group{}, false, nil
	}
	if err != nil {
		return model.Group{}, false, err
	}
	group.Members, err = s.listGroupMembers(ctx, group.ID)
	if err != nil {
		return model.Group{}, false, err
	}
	group.MemberCount = len(group.Members)
	return group, true, nil
}

func (s *Store) GroupNameExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM groups WHERE lower(name) = lower($1))`, name).Scan(&exists)
	return exists, err
}

func (s *Store) SaveGroup(ctx context.Context, group model.Group, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO groups (id, name) VALUES ($1,$2)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`, group.ID, group.Name); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM group_members WHERE group_id = $1`, group.ID); err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, member := range group.Members {
			role := member.GroupRole
			if role == "" {
				role = model.RoleGroupMember
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO group_members (group_id, user_id, role, created_at) VALUES ($1,$2,$3,$4)`, group.ID, member.ID, string(role), now); err != nil {
				return err
			}
		}
		return insertAuditTx(ctx, tx, audit)
	})
}
func (s *Store) DeleteGroup(ctx context.Context, groupID string, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM groups WHERE id = $1`, groupID); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) AddGroupMember(ctx context.Context, groupID, userID string, role model.GroupRole, audit model.AuditLog) error {
	if role == "" {
		role = model.RoleGroupMember
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO group_members (group_id, user_id, role, created_at) VALUES ($1,$2,$3,$4)
			ON CONFLICT (group_id, user_id) DO UPDATE SET role = EXCLUDED.role`, groupID, userID, string(role), time.Now().UTC()); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) RemoveGroupMember(ctx context.Context, groupID, userID string, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func (s *Store) listGroupMembers(ctx context.Context, groupID string) ([]model.GroupMember, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, role FROM group_members WHERE group_id = $1 ORDER BY created_at ASC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]model.GroupMember, 0)
	for rows.Next() {
		member := model.GroupMember{}
		if err := rows.Scan(&member.ID, &member.GroupRole); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}
