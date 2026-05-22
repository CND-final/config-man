package dbstore

import (
	"context"
	"database/sql"

	"config-man/backend/model"
)

func (s *Store) SaveSeedData(ctx context.Context, projects map[string]*model.Project, configs map[string]*model.ConfigEntry, reviews map[string]*model.ReviewRequest, templates map[string]*model.Template, versions []model.ConfigVersion, revisions []model.ConfigRevision, audits []model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
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
