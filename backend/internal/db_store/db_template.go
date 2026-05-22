package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"

	"config-man/backend/model"
)

func (s *Store) SaveTemplate(ctx context.Context, template model.Template, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := upsertTemplateTx(ctx, tx, template); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func upsertTemplateTx(ctx context.Context, tx *sql.Tx, template model.Template) error {
	variables, err := json.Marshal(template.Variables)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO custom_templates (id, owner_user_id, name, description, format, body, variables, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET owner_user_id = EXCLUDED.owner_user_id, name = EXCLUDED.name, description = EXCLUDED.description, format = EXCLUDED.format, body = EXCLUDED.body, variables = EXCLUDED.variables, updated_at = EXCLUDED.updated_at`, template.ID, template.OwnerUserID, template.Name, template.Description, template.Format, template.Body, variables, template.CreatedAt, template.UpdatedAt)
	return err
}

func (s *Store) ListTemplates(ctx context.Context, ownerUserID string) ([]model.Template, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, owner_user_id, name, description, format, body, variables, created_at, updated_at FROM custom_templates WHERE owner_user_id = $1 ORDER BY created_at DESC`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]model.Template, 0)
	for rows.Next() {
		template := model.Template{IsCustom: true}
		var variablesBytes []byte
		if err := rows.Scan(&template.ID, &template.OwnerUserID, &template.Name, &template.Description, &template.Format, &template.Body, &variablesBytes, &template.CreatedAt, &template.UpdatedAt); err != nil {
			return nil, err
		}
		if len(variablesBytes) > 0 {
			_ = json.Unmarshal(variablesBytes, &template.Variables)
		}
		templates = append(templates, template)
	}
	return templates, rows.Err()
}

func (s *Store) FindTemplate(ctx context.Context, ownerUserID, templateID string) (model.Template, bool, error) {
	template := model.Template{IsCustom: true}
	var variablesBytes []byte
	err := s.db.QueryRowContext(ctx, `SELECT id, owner_user_id, name, description, format, body, variables, created_at, updated_at FROM custom_templates WHERE owner_user_id = $1 AND id = $2`, ownerUserID, templateID).Scan(&template.ID, &template.OwnerUserID, &template.Name, &template.Description, &template.Format, &template.Body, &variablesBytes, &template.CreatedAt, &template.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.Template{}, false, nil
	}
	if err != nil {
		return model.Template{}, false, err
	}
	if len(variablesBytes) > 0 {
		_ = json.Unmarshal(variablesBytes, &template.Variables)
	}
	return template, true, nil
}
