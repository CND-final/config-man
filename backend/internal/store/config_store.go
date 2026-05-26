package store

import (
	"context"

	"config-man/backend/model"
)

func (s *Store) ListConfigs(projectID string) []model.Config {
	configs, err := s.db.ListConfigs(context.Background(), projectID)
	if err != nil {
		return nil
	}
	return configs
}

func (s *Store) FindConfig(projectID, configID string) (model.Config, bool) {
	config, ok, err := s.db.FindConfig(context.Background(), projectID, configID)
	if err != nil {
		return model.Config{}, false
	}
	return config, ok
}

func (s *Store) ConfigNameExists(projectID, name string) bool {
	exists, err := s.db.ConfigNameExists(context.Background(), projectID, name)
	return err == nil && exists
}

func (s *Store) SaveConfig(config model.Config, audit model.AuditLog) error {
	return s.db.SaveConfig(context.Background(), config, audit)
}
