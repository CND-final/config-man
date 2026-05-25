package store

import (
	"context"
	"time"

	"config-man/backend/model"
)

func (s *Store) ListConfigFiles(projectID string) []model.ConfigFile {
	s.ensureStandardConfigFiles(projectID)
	_ = s.db.AssignMissingConfigFileIDs(context.Background(), projectID)
	files, err := s.db.ListConfigFiles(context.Background(), projectID)
	if err != nil {
		return nil
	}
	return files
}

func (s *Store) FindConfigFile(projectID, fileID string) (model.ConfigFile, bool) {
	s.ensureStandardConfigFiles(projectID)
	file, ok, err := s.db.FindConfigFile(context.Background(), projectID, fileID)
	if err != nil {
		return model.ConfigFile{}, false
	}
	return file, ok
}

func (s *Store) ConfigFileNameExists(projectID, name string) bool {
	exists, err := s.db.ConfigFileNameExists(context.Background(), projectID, name)
	return err == nil && exists
}

func (s *Store) SaveConfigFile(file model.ConfigFile, audit model.AuditLog) error {
	return s.db.SaveConfigFile(context.Background(), file, audit)
}

func (s *Store) ensureStandardConfigFiles(projectID string) {
	if projectID == "" {
		return
	}
	now := time.Now().UTC()
	_ = s.db.UpsertConfigFiles(context.Background(), model.StandardConfigFiles(projectID, now))
}

func (s *Store) EnsureProjectConfigFiles(projectID string) {
	s.ensureStandardConfigFiles(projectID)
	_ = s.db.AssignMissingConfigFileIDs(context.Background(), projectID)
}
