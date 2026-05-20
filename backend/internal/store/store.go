package store

import (
	"database/sql"
	"sort"
	"strings"
	"sync"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

type Store struct {
	mu           sync.RWMutex
	db           *sql.DB
	users        []model.User
	usersByID    map[string]model.User
	usersByEmail map[string]model.User
	projects     map[string]*model.Project
	configs      map[string]*model.ConfigEntry
	reviews      map[string]*model.ReviewRequest
	templates    map[string]*model.Template
	versions     []model.ConfigVersion
	snapshots    []model.ConfigSnapshot
	audits       []model.AuditLog
}

func NewStore() *Store {
	store := newStoreBase(nil)
	store.seedDemoData()
	return store
}

func newStoreBase(db *sql.DB) *Store {
	store := &Store{
		db:           db,
		usersByID:    make(map[string]model.User),
		usersByEmail: make(map[string]model.User),
		projects:     make(map[string]*model.Project),
		configs:      make(map[string]*model.ConfigEntry),
		reviews:      make(map[string]*model.ReviewRequest),
		templates:    make(map[string]*model.Template),
	}
	store.seedUsers()
	return store
}

func (s *Store) FindUserByEmail(email string) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	return user, ok
}

func (s *Store) FindUserByID(id string) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByID[id]
	return user, ok
}

func (s *Store) ListProjects() []model.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]model.Project, 0, len(s.projects))
	for _, project := range s.projects {
		copyProject := cloneProject(*project)
		copyProject.ConfigCount = s.configCountLocked(project.ID)
		projects = append(projects, copyProject)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].CreatedAt.After(projects[j].CreatedAt)
	})
	return projects
}

func (s *Store) FindProject(projectID string) (model.Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return model.Project{}, false
	}
	copyProject := cloneProject(*project)
	copyProject.ConfigCount = s.configCountLocked(projectID)
	return copyProject, true
}

func (s *Store) ProjectNameExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, project := range s.projects {
		if strings.EqualFold(project.Name, name) {
			return true
		}
	}
	return false
}

func (s *Store) SaveProject(project model.Project, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyProject := cloneProject(project)
	s.projects[copyProject.ID] = &copyProject
	s.appendAuditLocked(audit)
	return s.persistLocked()
}

func (s *Store) ListTemplates(ownerUserID string) []model.Template {
	s.mu.RLock()
	defer s.mu.RUnlock()

	templates := make([]model.Template, 0)
	for _, template := range s.templates {
		if template.OwnerUserID == ownerUserID {
			templates = append(templates, cloneTemplate(*template))
		}
	}
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].CreatedAt.After(templates[j].CreatedAt)
	})
	return templates
}

func (s *Store) FindTemplate(ownerUserID, templateID string) (model.Template, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	template, ok := s.templates[templateID]
	if !ok || template.OwnerUserID != ownerUserID {
		return model.Template{}, false
	}
	return cloneTemplate(*template), true
}

func (s *Store) SaveTemplate(template model.Template, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyTemplate := cloneTemplate(template)
	s.templates[copyTemplate.ID] = &copyTemplate
	s.appendAuditLocked(audit)
	return s.persistLocked()
}

func (s *Store) ListConfigEntries(projectID, environment string) []model.ConfigEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]model.ConfigEntry, 0)
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && entry.Environment == environment {
			entries = append(entries, *entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}

func (s *Store) FindConfig(projectID, configID string) (model.ConfigEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.configs[configID]
	if !ok || entry.ProjectID != projectID {
		return model.ConfigEntry{}, false
	}
	return *entry, true
}

func (s *Store) FindConfigByKey(projectID, environment, key string) (model.ConfigEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry := s.findConfigByKeyLocked(projectID, environment, key)
	if entry == nil {
		return model.ConfigEntry{}, false
	}
	return *entry, true
}

func (s *Store) ListConfigVersions(configID string) []model.ConfigVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versions := make([]model.ConfigVersion, 0)
	for _, version := range s.versions {
		if version.ConfigID == configID {
			versions = append(versions, cloneVersion(version))
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})
	return versions
}

func (s *Store) SaveConfig(entry model.ConfigEntry, version model.ConfigVersion, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyEntry := entry
	s.configs[copyEntry.ID] = &copyEntry
	s.appendVersionLocked(version)
	s.snapshotLocked(copyEntry.ProjectID, copyEntry.Environment, version.ChangedBy, version.ChangeReason)
	s.appendAuditLocked(audit)
	return s.persistLocked()
}

func (s *Store) SaveConfigBatch(entries []model.ConfigEntry, versions []model.ConfigVersion, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range entries {
		entry := entries[index]
		s.configs[entry.ID] = &entry
	}
	for _, version := range versions {
		s.appendVersionLocked(version)
	}
	if len(entries) > 0 {
		changedBy := audit.Actor
		changeReason := audit.Action
		if len(versions) > 0 {
			changedBy = versions[0].ChangedBy
			changeReason = versions[0].ChangeReason
		}
		s.snapshotLocked(entries[0].ProjectID, entries[0].Environment, changedBy, changeReason)
	}
	s.appendAuditLocked(audit)
	return s.persistLocked()
}

func (s *Store) DeleteConfig(projectID, configID string, audit model.AuditLog) (model.ConfigEntry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.configs[configID]
	if !ok || entry.ProjectID != projectID {
		return model.ConfigEntry{}, false, nil
	}
	deleted := *entry
	delete(s.configs, configID)
	s.snapshotLocked(deleted.ProjectID, deleted.Environment, audit.Actor, audit.Action)
	s.appendAuditLocked(audit)
	return deleted, true, s.persistLocked()
}

func (s *Store) ListConfigSnapshots(projectID, environment string) []model.ConfigSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshots := make([]model.ConfigSnapshot, 0)
	for _, snapshot := range s.snapshots {
		if snapshot.ProjectID == projectID && snapshot.Environment == environment {
			snapshots = append(snapshots, cloneSnapshot(snapshot))
		}
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})
	return snapshots
}

func (s *Store) RestoreConfigSnapshot(projectID, environment string, snapshot model.ConfigSnapshot, actor, reason string, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentByKey := make(map[string]*model.ConfigEntry)
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && entry.Environment == environment {
			currentByKey[entry.Key] = entry
		}
	}

	now := time.Now().UTC()
	restoredKeys := make(map[string]bool, len(snapshot.Entries))
	for _, snapshotEntry := range snapshot.Entries {
		restoredKeys[snapshotEntry.Key] = true
		entry := currentByKey[snapshotEntry.Key]
		if entry == nil {
			newEntry := model.ConfigEntry{
				ID:          util.NewID("cfg"),
				ProjectID:   projectID,
				Environment: environment,
				Key:         snapshotEntry.Key,
				Value:       snapshotEntry.Value,
				ValueType:   snapshotEntry.ValueType,
				IsSensitive: snapshotEntry.IsSensitive,
				UpdatedBy:   actor,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			s.configs[newEntry.ID] = &newEntry
			s.versionLocked(newEntry.ID, nil, newEntry.Value, actor, reason)
			continue
		}

		oldValue := entry.Value
		if entry.Value != snapshotEntry.Value || entry.ValueType != snapshotEntry.ValueType || entry.IsSensitive != snapshotEntry.IsSensitive {
			entry.Value = snapshotEntry.Value
			entry.ValueType = snapshotEntry.ValueType
			entry.IsSensitive = snapshotEntry.IsSensitive
			entry.UpdatedBy = actor
			entry.UpdatedAt = now
			s.versionLocked(entry.ID, &oldValue, entry.Value, actor, reason)
		}
	}

	for key, entry := range currentByKey {
		if !restoredKeys[key] {
			delete(s.configs, entry.ID)
		}
	}

	s.snapshotLocked(projectID, environment, actor, reason)
	s.appendAuditLocked(audit)
	return s.persistLocked()
}

func (s *Store) ListReviewRequests(projectID string, filters model.ReviewFilters) []model.ReviewRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests := make([]model.ReviewRequest, 0, len(s.reviews))
	for _, request := range s.reviews {
		if projectID != "" && request.ProjectID != projectID {
			continue
		}
		if filters.Environment != "" && request.Environment != filters.Environment {
			continue
		}
		if filters.ConfigKey != "" && request.ConfigKey != filters.ConfigKey {
			continue
		}
		if filters.Status != "" && request.Status != filters.Status {
			continue
		}
		requests = append(requests, *request)
	}
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].CreatedAt.After(requests[j].CreatedAt)
	})
	return requests
}

func (s *Store) FindReviewRequest(requestID string) (model.ReviewRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	request, ok := s.reviews[requestID]
	if !ok {
		return model.ReviewRequest{}, false
	}
	return *request, true
}

func (s *Store) SaveReviewRequest(request model.ReviewRequest, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyRequest := request
	s.reviews[copyRequest.ID] = &copyRequest
	s.appendAuditLocked(audit)
	return s.persistLocked()
}

func (s *Store) ValidationEntries(projectID string, environments []string) []model.ValidationEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]model.ValidationEntry, 0)
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && util.Contains(environments, entry.Environment) {
			entries = append(entries, model.ValidationEntry{
				Environment: entry.Environment,
				Key:         entry.Key,
				Value:       entry.Value,
				ValueType:   entry.ValueType,
				IsSensitive: entry.IsSensitive,
			})
		}
	}
	return entries
}

func (s *Store) seedUsers() {
	users := []model.User{
		{ID: "alice", Email: "admin@config-man.local", Name: "Alice Lin", Role: model.RoleSystemAdmin},
		{ID: "paul", Email: "project-admin@config-man.local", Name: "Paul Wu", Role: model.RoleProjectAdmin},
		{ID: "nora", Email: "developer@config-man.local", Name: "Nora Chen", Role: model.RoleDeveloper},
		{ID: "rachel", Email: "reviewer@config-man.local", Name: "Rachel Kao", Role: model.RoleReviewer},
		{ID: "vincent", Email: "viewer@config-man.local", Name: "Vincent Lee", Role: model.RoleViewer},
	}
	for _, user := range users {
		s.users = append(s.users, user)
		s.usersByID[user.ID] = user
		s.usersByEmail[strings.ToLower(user.Email)] = user
	}
}

func (s *Store) seedDemoData() {
	now := time.Now().UTC()
	project := &model.Project{
		ID:            "customer-portal",
		Name:          "customer-portal",
		Description:   "Demo project for config-man phase 1",
		RepoURL:       "https://git.example.com/platform/customer-portal",
		OwnerName:     "Platform Team",
		DefaultFormat: "yaml",
		Environments: []model.ProjectEnvironment{
			{ID: "env-customer-dev", Name: "dev", SortOrder: 1},
			{ID: "env-customer-staging", Name: "staging", SortOrder: 2},
			{ID: "env-customer-prod", Name: "prod", SortOrder: 3},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.projects[project.ID] = project

	entries := []model.ConfigEntry{
		{ID: "cfg-dev-api-baseurl", ProjectID: project.ID, Environment: "dev", Key: "api.baseUrl", Value: "https://dev-api.example.com", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-dev-log-level", ProjectID: project.ID, Environment: "dev", Key: "log.level", Value: "debug", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-staging-api-baseurl", ProjectID: project.ID, Environment: "staging", Key: "api.baseUrl", Value: "https://staging-api.example.com", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-staging-log-level", ProjectID: project.ID, Environment: "staging", Key: "log.level", Value: "info", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-prod-api-baseurl", ProjectID: project.ID, Environment: "prod", Key: "api.baseUrl", Value: "https://api.example.com", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-prod-database-url", ProjectID: project.ID, Environment: "prod", Key: "database.url", Value: "postgresql://prod-user:secret@prod-db:5432/app", ValueType: "string", IsSensitive: true, UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
	}
	for index := range entries {
		entry := &entries[index]
		s.configs[entry.ID] = entry
		s.versionLocked(entry.ID, nil, entry.Value, "seed-admin", "seed demo config")
	}
	for _, env := range project.Environments {
		s.snapshotLocked(project.ID, env.Name, "seed-admin", "seed demo config")
	}

	review := &model.ReviewRequest{
		ID:          "seed-prod-database-review",
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Environment: "prod",
		ConfigKey:   "database.url",
		Requester:   "Nora Chen",
		Reviewer:    "Rachel Kao",
		Status:      "approved",
		Reason:      "Rotate production database credential before release",
		Comment:     "Approved for demo seed data",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.reviews[review.ID] = review
	s.auditLocked("seed-admin", "seed_demo_project", "project", project.ID, project.ID, map[string]any{
		"projectName": project.Name,
	})
}

func (s *Store) seedCurrentSnapshots() {
	for _, project := range s.projects {
		for _, env := range project.Environments {
			s.snapshotLocked(project.ID, env.Name, "system", "initialize config history")
		}
	}
}

func (s *Store) configCountLocked(projectID string) int {
	count := 0
	for _, entry := range s.configs {
		if entry.ProjectID == projectID {
			count++
		}
	}
	return count
}

func (s *Store) findConfigByKeyLocked(projectID, environment, key string) *model.ConfigEntry {
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && entry.Environment == environment && entry.Key == key {
			return entry
		}
	}
	return nil
}

func (s *Store) versionLocked(configID string, oldValue *string, newValue, changedBy, reason string) {
	s.versions = append(s.versions, model.ConfigVersion{
		ID:           util.NewID("ver"),
		ConfigID:     configID,
		OldValue:     oldValue,
		NewValue:     newValue,
		ChangedBy:    changedBy,
		ChangeReason: reason,
		CreatedAt:    time.Now().UTC(),
	})
}

func (s *Store) snapshotLocked(projectID, environment, changedBy, reason string) {
	entries := make([]model.ConfigSnapshotEntry, 0)
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && entry.Environment == environment {
			entries = append(entries, model.ConfigSnapshotEntry{
				Key:         entry.Key,
				Value:       entry.Value,
				ValueType:   entry.ValueType,
				IsSensitive: entry.IsSensitive,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	s.snapshots = append(s.snapshots, model.ConfigSnapshot{
		ID:           util.NewID("snap"),
		ProjectID:    projectID,
		Environment:  environment,
		Entries:      entries,
		ChangedBy:    changedBy,
		ChangeReason: reason,
		CreatedAt:    time.Now().UTC(),
	})
}

func (s *Store) auditLocked(actor, action, resourceType, resourceID, projectID string, metadata map[string]any) {
	s.audits = append(s.audits, model.AuditLog{
		ID:           util.NewID("aud"),
		Actor:        actor,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ProjectID:    projectID,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	})
}

func (s *Store) appendVersionLocked(version model.ConfigVersion) {
	if version.ID != "" {
		s.versions = append(s.versions, version)
	}
}

func (s *Store) appendAuditLocked(audit model.AuditLog) {
	if audit.ID != "" {
		s.audits = append(s.audits, audit)
	}
}

func cloneProject(project model.Project) model.Project {
	project.Environments = append([]model.ProjectEnvironment(nil), project.Environments...)
	return project
}

func cloneTemplate(template model.Template) model.Template {
	template.Variables = append([]model.TemplateVariable(nil), template.Variables...)
	template.Entries = append([]model.TemplateEntry(nil), template.Entries...)
	return template
}

func cloneVersion(version model.ConfigVersion) model.ConfigVersion {
	if version.OldValue != nil {
		oldValue := *version.OldValue
		version.OldValue = &oldValue
	}
	return version
}

func cloneSnapshot(snapshot model.ConfigSnapshot) model.ConfigSnapshot {
	snapshot.Entries = append([]model.ConfigSnapshotEntry(nil), snapshot.Entries...)
	return snapshot
}
