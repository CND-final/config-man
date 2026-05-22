package store

import (
	"context"

	"config-man/backend/model"
)

func (s *Store) ListTemplates(ownerUserID string) []model.Template {
	templates, err := s.db.ListTemplates(context.Background(), ownerUserID)
	if err != nil {
		return nil
	}
	return templates
}

func (s *Store) FindTemplate(ownerUserID, templateID string) (model.Template, bool) {
	template, ok, err := s.db.FindTemplate(context.Background(), ownerUserID, templateID)
	if err != nil {
		return model.Template{}, false
	}
	return template, ok
}

func (s *Store) SaveTemplate(template model.Template, audit model.AuditLog) error {
	return s.db.SaveTemplate(context.Background(), template, audit)
}
