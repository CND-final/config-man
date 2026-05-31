package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (s *Store) ListConfigEntries(projectID, environment, branch string) []model.ConfigEntry {
	entries, err := s.db.ListConfigEntries(context.Background(), projectID, environment, branch)
	if err != nil {
		return nil
	}
	return entries
}

func (s *Store) FindConfigEntry(projectID, configID string) (model.ConfigEntry, bool) {
	entry, ok, err := s.db.FindConfigEntry(context.Background(), projectID, configID)
	if err != nil {
		return model.ConfigEntry{}, false
	}
	return entry, ok
}

func (s *Store) FindConfigEntryByKey(projectID, environment, branch, key string) (model.ConfigEntry, bool) {
	entry, ok, err := s.db.FindConfigEntryByKey(context.Background(), projectID, environment, branch, key)
	if err != nil {
		return model.ConfigEntry{}, false
	}
	return entry, ok
}

func (s *Store) ListConfigVersions(configID string) []model.ConfigVersion {
	versions, err := s.db.ListConfigVersions(context.Background(), configID)
	if err != nil {
		return nil
	}
	return versions
}

func (s *Store) SaveConfigEntries(entries []model.ConfigEntry, versions []model.ConfigVersion, audit model.AuditLog) error {
	revision, err := s.revisionForConfigChanges(entries, versions, audit)
	if err != nil {
		return err
	}
	return s.db.SaveConfigChanges(context.Background(), entries, versions, revision, audit)
}

func (s *Store) DeleteConfigEntry(projectID, configID string, audit model.AuditLog) (model.ConfigEntry, bool, error) {
	deleted, ok, err := s.db.FindConfigEntry(context.Background(), projectID, configID)
	if err != nil || !ok {
		return model.ConfigEntry{}, ok, err
	}
	entries, err := s.db.ListConfigEntries(context.Background(), deleted.ProjectID, deleted.Environment, util.NormalizeBranch(deleted.Branch))
	if err != nil {
		return model.ConfigEntry{}, false, err
	}
	revision := configRevisionFromEntries(deleted.ProjectID, deleted.Environment, audit.Actor, audit.Action, entries, configID)
	return deleted, true, s.db.DeleteConfigEntry(context.Background(), configID, revision, audit)
}

func (s *Store) ListConfigRevisions(projectID, environment, branch string) []model.ConfigRevision {
	revisions, err := s.db.ListConfigRevisions(context.Background(), projectID, environment, branch)
	if err != nil {
		return nil
	}
	return revisions
}

func (s *Store) RestoreConfigRevision(projectID, environment string, revision model.ConfigRevision, actor, reason string, audit model.AuditLog) error {
	currentEntries, err := s.db.ListConfigEntries(context.Background(), projectID, environment, util.NormalizeBranch(revision.Branch))
	if err != nil {
		return err
	}
	upserts, deleteIDs, versions := buildRestoreChanges(projectID, environment, revision, currentEntries, actor, reason)
	nextRevision := configRevisionFromRevision(projectID, environment, actor, reason, revision)
	return s.restoreConfigDB(projectID, environment, upserts, deleteIDs, versions, nextRevision, audit)
}

func (s *Store) restoreConfigDB(projectID, environment string, upserts []model.ConfigEntry, deleteIDs []string, versions []model.ConfigVersion, revision model.ConfigRevision, audit model.AuditLog) error {
	if projectID == "" || environment == "" {
		return fmt.Errorf("projectID and environment are required")
	}
	return s.db.RestoreConfig(context.Background(), upserts, deleteIDs, versions, revision, audit)
}

func (s *Store) revisionForConfigChanges(entries []model.ConfigEntry, versions []model.ConfigVersion, audit model.AuditLog) (model.ConfigRevision, error) {
	if len(entries) == 0 {
		return model.ConfigRevision{}, nil
	}
	projectID := entries[0].ProjectID
	environment := entries[0].Environment
	currentEntries, err := s.db.ListConfigEntries(context.Background(), projectID, environment, util.NormalizeBranch(entries[0].Branch))
	if err != nil {
		return model.ConfigRevision{}, err
	}
	changedBy := audit.Actor
	changeReason := audit.Action
	if len(versions) > 0 {
		changedBy = versions[0].ChangedBy
		changeReason = versions[0].ChangeReason
	}
	return configRevisionFromEntries(projectID, environment, changedBy, changeReason, mergeChangedEntries(currentEntries, entries), ""), nil
}

func mergeChangedEntries(currentEntries, changedEntries []model.ConfigEntry) []model.ConfigEntry {
	changedByID := make(map[string]model.ConfigEntry, len(changedEntries))
	for _, entry := range changedEntries {
		changedByID[entry.ID] = entry
	}

	merged := make([]model.ConfigEntry, 0, len(currentEntries)+len(changedEntries))
	seen := make(map[string]bool, len(currentEntries)+len(changedEntries))
	for _, entry := range currentEntries {
		if changed, ok := changedByID[entry.ID]; ok {
			merged = append(merged, changed)
			seen[entry.ID] = true
			continue
		}
		merged = append(merged, entry)
		seen[entry.ID] = true
	}
	for _, entry := range changedEntries {
		if !seen[entry.ID] {
			merged = append(merged, entry)
			seen[entry.ID] = true
		}
	}
	return merged
}

func configRevisionFromEntries(projectID, environment, changedBy, reason string, entries []model.ConfigEntry, excludeConfigID string) model.ConfigRevision {
	revisionEntries := make([]model.ConfigRevisionEntry, 0, len(entries))
	for _, entry := range entries {
		if excludeConfigID != "" && entry.ID == excludeConfigID {
			continue
		}
		revisionEntries = append(revisionEntries, model.ConfigRevisionEntry{
			ConfigID:    entry.ConfigID,
			Branch:      entry.Branch,
			Key:         entry.Key,
			Value:       entry.Value,
			ValueType:   entry.ValueType,
			IsSensitive: entry.IsSensitive,
		})
	}
	sort.Slice(revisionEntries, func(i, j int) bool {
		if revisionEntries[i].ConfigID != revisionEntries[j].ConfigID {
			return revisionEntries[i].ConfigID < revisionEntries[j].ConfigID
		}
		return revisionEntries[i].Key < revisionEntries[j].Key
	})
	return model.ConfigRevision{
		ID:           util.NewID("rev"),
		ProjectID:    projectID,
		Environment:  environment,
		Branch:       branchFromEntries(entries),
		Entries:      revisionEntries,
		ChangedBy:    changedBy,
		ChangeReason: reason,
		CreatedAt:    time.Now().UTC(),
	}
}

func branchFromEntries(entries []model.ConfigEntry) string {
	if len(entries) == 0 {
		return "default"
	}
	return util.NormalizeBranch(entries[0].Branch)
}

func configRevisionFromRevision(projectID, environment, changedBy, reason string, sourceRevision model.ConfigRevision) model.ConfigRevision {
	revision := model.ConfigRevision{
		ID:           util.NewID("rev"),
		ProjectID:    projectID,
		Environment:  environment,
		Branch:       util.NormalizeBranch(sourceRevision.Branch),
		Entries:      append([]model.ConfigRevisionEntry(nil), sourceRevision.Entries...),
		ChangedBy:    changedBy,
		ChangeReason: reason,
		CreatedAt:    time.Now().UTC(),
	}
	sort.Slice(revision.Entries, func(i, j int) bool {
		if revision.Entries[i].ConfigID != revision.Entries[j].ConfigID {
			return revision.Entries[i].ConfigID < revision.Entries[j].ConfigID
		}
		return revision.Entries[i].Key < revision.Entries[j].Key
	})
	return revision
}

func buildRestoreChanges(projectID, environment string, revision model.ConfigRevision, currentEntries []model.ConfigEntry, actor, reason string) ([]model.ConfigEntry, []string, []model.ConfigVersion) {
	currentByKey := make(map[string]model.ConfigEntry, len(currentEntries))
	for _, entry := range currentEntries {
		currentByKey[configEntryRevisionKey(entry.ConfigID, entry.Key)] = entry
	}

	now := time.Now().UTC()
	upserts := make([]model.ConfigEntry, 0, len(revision.Entries))
	versions := make([]model.ConfigVersion, 0)
	restoredKeys := make(map[string]bool, len(revision.Entries))
	for _, revisionEntry := range revision.Entries {
		identity := configEntryRevisionKey(revisionEntry.ConfigID, revisionEntry.Key)
		restoredKeys[identity] = true
		entry, ok := currentByKey[identity]
		if !ok {
			newEntry := model.ConfigEntry{
				ID:          util.NewID("cfg"),
				ProjectID:   projectID,
				Environment: environment,
				ConfigID:    revisionEntry.ConfigID,
				Key:         revisionEntry.Key,
				Value:       revisionEntry.Value,
				ValueType:   revisionEntry.ValueType,
				IsSensitive: revisionEntry.IsSensitive,
				UpdatedBy:   actor,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			upserts = append(upserts, newEntry)
			versions = append(versions, newConfigVersion(newEntry.ID, nil, newEntry.Value, actor, reason))
			continue
		}

		oldValue := entry.Value
		entry.Branch = util.NormalizeBranch(revision.Branch)
		if revisionEntry.ConfigID != "" {
			entry.ConfigID = revisionEntry.ConfigID
		}
		if entry.Value != revisionEntry.Value || entry.ValueType != revisionEntry.ValueType || entry.IsSensitive != revisionEntry.IsSensitive {
			entry.Value = revisionEntry.Value
			entry.ValueType = revisionEntry.ValueType
			entry.IsSensitive = revisionEntry.IsSensitive
			entry.UpdatedBy = actor
			entry.UpdatedAt = now
			upserts = append(upserts, entry)
			versions = append(versions, newConfigVersion(entry.ID, &oldValue, entry.Value, actor, reason))
		}
	}

	deleteIDs := make([]string, 0)
	for _, entry := range currentEntries {
		if !restoredKeys[configEntryRevisionKey(entry.ConfigID, entry.Key)] {
			deleteIDs = append(deleteIDs, entry.ID)
		}
	}
	return upserts, deleteIDs, versions
}

func configEntryRevisionKey(configID, key string) string {
	return configID + "\x00" + key
}

func newConfigVersion(configEntryID string, oldValue *string, newValue, changedBy, reason string) model.ConfigVersion {
	return model.ConfigVersion{
		ID:            util.NewID("ver"),
		ConfigEntryID: configEntryID,
		OldValue:      oldValue,
		NewValue:      newValue,
		ChangedBy:     changedBy,
		ChangeReason:  reason,
		CreatedAt:     time.Now().UTC(),
	}
}
