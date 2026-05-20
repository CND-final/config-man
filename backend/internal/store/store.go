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
	versions     []model.ConfigVersion
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

func (s *Store) SaveConfig(entry model.ConfigEntry, version model.ConfigVersion, audit model.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyEntry := entry
	s.configs[copyEntry.ID] = &copyEntry
	s.appendVersionLocked(version)
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
	s.appendAuditLocked(audit)
	return deleted, true, s.persistLocked()
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
