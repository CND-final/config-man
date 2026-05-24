package store

import (
	"context"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
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

func (s *Store) ListSharedConfigs() []model.SharedConfig {
	items, err := s.db.ListSharedConfigs(context.Background())
	if err != nil {
		return nil
	}
	return items
}

func (s *Store) FindSharedConfig(id string) (model.SharedConfig, bool) {
	item, ok, err := s.db.FindSharedConfig(context.Background(), id)
	if err != nil {
		return model.SharedConfig{}, false
	}
	return item, ok
}

func (s *Store) SaveSharedConfig(item model.SharedConfig, audit model.AuditLog) error {
	return s.db.SaveSharedConfig(context.Background(), item, audit)
}

func (s *Store) DeleteSharedConfig(id string, audit model.AuditLog) error {
	return s.db.DeleteSharedConfig(context.Background(), id, audit)
}

func (s *Store) SaveSharedConfigUpdateRequest(request model.SharedConfigUpdateRequest, audit model.AuditLog) error {
	return s.db.SaveSharedConfigUpdateRequest(context.Background(), request, audit)
}

func (s *Store) ensureDefaultSharedConfigs() {
	hasItems, err := s.db.HasSharedConfigs(context.Background())
	if err != nil || hasItems {
		return
	}
	now := time.Now().UTC()
	for _, item := range model.SeedSharedConfigs(now) {
		_ = s.SaveSharedConfig(item, model.AuditLog{
			ID:           util.NewID("aud"),
			Actor:        "system",
			Action:       "shared_config.seed",
			ResourceType: "shared_config",
			ResourceID:   item.ID,
			Metadata:     map[string]any{"scope": item.Scope, "entryCount": len(item.Entries)},
			CreatedAt:    now,
		})
	}
}
