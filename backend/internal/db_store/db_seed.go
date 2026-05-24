package dbstore

import (
	"context"
	"database/sql"
	"time"

	"config-man/backend/model"
)

func (s *Store) SaveSeedData(ctx context.Context, groups map[string]*model.Group, projects map[string]*model.Project, configs map[string]*model.ConfigEntry, reviews map[string]*model.ReviewRequest, templates map[string]*model.Template, versions []model.ConfigVersion, revisions []model.ConfigRevision, audits []model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		for _, group := range groups {
			if _, err := tx.ExecContext(ctx, `INSERT INTO groups (id, name) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`, group.ID, group.Name); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM group_members WHERE group_id = $1`, group.ID); err != nil {
				return err
			}
			now := projectSeedTime(projects)
			for _, member := range group.Members {
				role := member.GroupRole
				if role == "" {
					role = model.RoleGroupMember
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO group_members (group_id, user_id, role, created_at) VALUES ($1,$2,$3,$4)`, group.ID, member.ID, string(role), now); err != nil {
					return err
				}
			}
		}
		for _, project := range projects {
			if err := upsertProjectTx(ctx, tx, *project); err != nil {
				return err
			}
		}
		for _, template := range templates {
			if err := upsertTemplateTx(ctx, tx, *template); err != nil {
				return err
			}
		}
		for _, entry := range configs {
			if err := upsertConfigEntryTx(ctx, tx, *entry); err != nil {
				return err
			}
		}
		for _, version := range versions {
			if err := insertConfigVersionSeedTx(ctx, tx, version); err != nil {
				return err
			}
		}
		for _, revision := range revisions {
			if err := insertConfigRevisionSeedTx(ctx, tx, revision); err != nil {
				return err
			}
		}
		for _, review := range reviews {
			if err := upsertReviewRequestTx(ctx, tx, *review); err != nil {
				return err
			}
		}
		for _, audit := range audits {
			if err := insertAuditSeedTx(ctx, tx, audit); err != nil {
				return err
			}
		}
		return nil
	})
}

func projectSeedTime(projects map[string]*model.Project) time.Time {
	for _, project := range projects {
		if !project.CreatedAt.IsZero() {
			return project.CreatedAt
		}
	}
	return time.Now().UTC()
}
