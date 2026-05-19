package processor

import (
	"config-man/backend/model"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const maskedValue = "******"

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

func (s *Store) CreateProject(req model.CreateProjectRequest, actor string) (model.Project, *AppError) {
	name := strings.TrimSpace(req.Name)
	owner := strings.TrimSpace(req.OwnerName)
	if len(name) < 2 || len(owner) < 2 {
		return model.Project{}, badRequest("name and ownerName are required")
	}

	format := strings.TrimSpace(req.DefaultFormat)
	if format == "" {
		format = "yaml"
	}
	if !isSupportedConfigFormat(format) {
		return model.Project{}, badRequest("defaultFormat must be properties, json, or yaml")
	}

	envs := normalizeEnvironments(req.Environments)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, project := range s.projects {
		if strings.EqualFold(project.Name, name) {
			return model.Project{}, conflict(fmt.Sprintf("model.Project %q already exists", name))
		}
	}

	project := &model.Project{
		ID:            newID("prj"),
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		RepoURL:       strings.TrimSpace(req.RepoURL),
		OwnerName:     owner,
		DefaultFormat: format,
		Environments:  make([]model.ProjectEnvironment, 0, len(envs)),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	for index, env := range envs {
		project.Environments = append(project.Environments, model.ProjectEnvironment{
			ID:        newID("env"),
			Name:      env,
			SortOrder: index + 1,
		})
	}

	s.projects[project.ID] = project
	s.auditLocked(actor, "project.create", "project", project.ID, project.ID, map[string]any{
		"name":         project.Name,
		"environments": envs,
	})
	if err := s.persistLocked(); err != nil {
		return model.Project{}, err
	}

	return cloneProject(*project), nil
}

func (s *Store) GetProject(projectID string) (model.Project, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return model.Project{}, notFound(fmt.Sprintf("model.Project %q not found", projectID))
	}
	copyProject := cloneProject(*project)
	copyProject.ConfigCount = s.configCountLocked(projectID)
	return copyProject, nil
}

func (s *Store) EnvironmentExists(projectID, environment string) *AppError {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ensureEnvironmentLocked(projectID, environment)
}

func (s *Store) ListConfigs(user model.User, projectID, environment string, revealSensitive bool) (map[string]any, *AppError) {
	if strings.TrimSpace(environment) == "" {
		return nil, badRequest(`Query parameter "env" is required`)
	}
	if revealSensitive && !canRevealSensitive(user) {
		return nil, forbidden("Role cannot reveal sensitive values")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.ensureEnvironmentLocked(projectID, environment); err != nil {
		return nil, err
	}

	entries := make([]model.ConfigEntry, 0)
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && entry.Environment == environment {
			serialized := *entry
			if serialized.IsSensitive && !revealSensitive {
				serialized.Value = maskedValue
			}
			entries = append(entries, serialized)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	return map[string]any{
		"projectId":    projectID,
		"environment":  environment,
		"entries":      entries,
		"entryCount":   len(entries),
		"maskedValues": !revealSensitive,
	}, nil
}

func (s *Store) CreateConfig(user model.User, projectID string, req model.CreateConfigRequest) (model.ConfigEntry, *AppError) {
	environment := strings.TrimSpace(req.Environment)
	key := strings.TrimSpace(req.Key)
	if environment == "" || key == "" {
		return model.ConfigEntry{}, badRequest("environment and key are required")
	}
	if !canWriteEnvironment(user, environment) {
		return model.ConfigEntry{}, forbidden(fmt.Sprintf("Role %q cannot modify %q config", user.Role, environment))
	}

	valueType := normalizeValueType(req.ValueType)
	if valueType == "" {
		return model.ConfigEntry{}, badRequest("valueType must be string, number, boolean, or json")
	}
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureEnvironmentLocked(projectID, environment); err != nil {
		return model.ConfigEntry{}, err
	}
	if existing := s.findConfigByKeyLocked(projectID, environment, key); existing != nil {
		return model.ConfigEntry{}, conflict(fmt.Sprintf("Config key %q already exists in %q", key, environment))
	}

	entry := &model.ConfigEntry{
		ID:          newID("cfg"),
		ProjectID:   projectID,
		Environment: environment,
		Key:         key,
		Value:       req.Value,
		ValueType:   valueType,
		IsSensitive: req.IsSensitive,
		UpdatedBy:   user.Name,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.configs[entry.ID] = entry
	s.versionLocked(entry.ID, nil, entry.Value, user.Name, fallback(req.ChangeReason, "create config"))
	s.auditLocked(user.Name, "config.create", "config_entry", entry.ID, projectID, map[string]any{
		"environment": entry.Environment,
		"key":         entry.Key,
		"isSensitive": entry.IsSensitive,
	})
	if err := s.persistLocked(); err != nil {
		return model.ConfigEntry{}, err
	}
	return *entry, nil
}

func (s *Store) UpdateConfig(user model.User, projectID, configID string, req model.UpdateConfigRequest) (model.ConfigEntry, *AppError) {
	if req.Value == nil && req.ValueType == nil && req.IsSensitive == nil {
		return model.ConfigEntry{}, badRequest("No config fields provided for update")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.configs[configID]
	if !ok || entry.ProjectID != projectID {
		return model.ConfigEntry{}, notFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	if !canWriteEnvironment(user, entry.Environment) {
		return model.ConfigEntry{}, forbidden(fmt.Sprintf("Role %q cannot modify %q config", user.Role, entry.Environment))
	}

	oldValue := entry.Value
	if req.Value != nil {
		entry.Value = *req.Value
	}
	if req.ValueType != nil {
		valueType := normalizeValueType(*req.ValueType)
		if valueType == "" {
			return model.ConfigEntry{}, badRequest("valueType must be string, number, boolean, or json")
		}
		entry.ValueType = valueType
	}
	if req.IsSensitive != nil {
		entry.IsSensitive = *req.IsSensitive
	}
	entry.UpdatedBy = user.Name
	entry.UpdatedAt = time.Now().UTC()

	s.versionLocked(entry.ID, &oldValue, entry.Value, user.Name, fallback(req.ChangeReason, "update config"))
	s.auditLocked(user.Name, "config.update", "config_entry", entry.ID, projectID, map[string]any{
		"environment": entry.Environment,
		"key":         entry.Key,
		"valueType":   entry.ValueType,
		"isSensitive": entry.IsSensitive,
	})
	if err := s.persistLocked(); err != nil {
		return model.ConfigEntry{}, err
	}

	return *entry, nil
}

func (s *Store) DeleteConfig(user model.User, projectID, configID string) (map[string]any, *AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.configs[configID]
	if !ok || entry.ProjectID != projectID {
		return nil, notFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	if !canWriteEnvironment(user, entry.Environment) {
		return nil, forbidden(fmt.Sprintf("Role %q cannot modify %q config", user.Role, entry.Environment))
	}

	deleted := *entry
	delete(s.configs, configID)
	s.auditLocked(user.Name, "config.delete", "config_entry", configID, projectID, map[string]any{
		"environment": deleted.Environment,
		"key":         deleted.Key,
	})
	if err := s.persistLocked(); err != nil {
		return nil, err
	}

	return map[string]any{"deleted": true, "config": deleted}, nil
}

func (s *Store) ImportConfigs(user model.User, projectID string, req model.ImportConfigRequest) (map[string]any, *AppError) {
	environment := strings.TrimSpace(req.Environment)
	if environment == "" {
		return nil, badRequest("environment is required")
	}
	if !canWriteEnvironment(user, environment) {
		return nil, forbidden(fmt.Sprintf("Role %q cannot modify %q config", user.Role, environment))
	}
	if !isSupportedConfigFormat(req.Format) {
		return nil, badRequest("format must be json, yaml, or properties")
	}

	parsed, parseErr := parseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		return nil, badRequest(parseErr.Error())
	}
	if len(parsed) == 0 {
		return nil, badRequest("No config entries found in file content")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureEnvironmentLocked(projectID, environment); err != nil {
		return nil, err
	}

	created, updated, unchanged := 0, 0, 0
	for _, parsedEntry := range parsed {
		existing := s.findConfigByKeyLocked(projectID, environment, parsedEntry.Key)
		if existing == nil {
			now := time.Now().UTC()
			entry := &model.ConfigEntry{
				ID:          newID("cfg"),
				ProjectID:   projectID,
				Environment: environment,
				Key:         parsedEntry.Key,
				Value:       parsedEntry.Value,
				ValueType:   parsedEntry.ValueType,
				IsSensitive: looksSensitive(parsedEntry.Key),
				UpdatedBy:   user.Name,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			s.configs[entry.ID] = entry
			s.versionLocked(entry.ID, nil, entry.Value, user.Name, fallback(req.ChangeReason, "import config file"))
			created++
			continue
		}

		if existing.Value == parsedEntry.Value && existing.ValueType == parsedEntry.ValueType {
			unchanged++
			continue
		}

		oldValue := existing.Value
		existing.Value = parsedEntry.Value
		existing.ValueType = parsedEntry.ValueType
		existing.IsSensitive = existing.IsSensitive || looksSensitive(parsedEntry.Key)
		existing.UpdatedBy = user.Name
		existing.UpdatedAt = time.Now().UTC()
		s.versionLocked(existing.ID, &oldValue, existing.Value, user.Name, fallback(req.ChangeReason, "import config file"))
		updated++
	}

	s.auditLocked(user.Name, "config.import", "config_file", "", projectID, map[string]any{
		"environment": environment,
		"format":      req.Format,
		"created":     created,
		"updated":     updated,
		"unchanged":   unchanged,
	})
	if err := s.persistLocked(); err != nil {
		return nil, err
	}

	return map[string]any{
		"projectId":    projectID,
		"environment":  environment,
		"format":       req.Format,
		"imported":     len(parsed),
		"created":      created,
		"updated":      updated,
		"unchanged":    unchanged,
		"changeReason": fallback(req.ChangeReason, "import config file"),
	}, nil
}

func (s *Store) ListReviewRequests(projectID string, filters model.ReviewFilters) ([]model.ReviewRequest, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if projectID != "" {
		if _, ok := s.projects[projectID]; !ok {
			return nil, notFound(fmt.Sprintf("model.Project %q not found", projectID))
		}
	}

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
	return requests, nil
}

func (s *Store) CreateReviewRequest(user model.User, req model.CreateReviewRequest) (model.ReviewRequest, *AppError) {
	if !canCreateReview(user) {
		return model.ReviewRequest{}, forbidden(fmt.Sprintf("Role %q is not allowed", user.Role))
	}
	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Environment) == "" || strings.TrimSpace(req.Reason) == "" {
		return model.ReviewRequest{}, badRequest("projectId, environment, and reason are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureEnvironmentLocked(req.ProjectID, req.Environment); err != nil {
		return model.ReviewRequest{}, err
	}
	project := s.projects[req.ProjectID]
	now := time.Now().UTC()
	request := &model.ReviewRequest{
		ID:          newID("rev"),
		ProjectID:   req.ProjectID,
		ProjectName: project.Name,
		Environment: req.Environment,
		ConfigKey:   strings.TrimSpace(req.ConfigKey),
		Requester:   user.Name,
		Status:      "pending",
		Reason:      strings.TrimSpace(req.Reason),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.reviews[request.ID] = request
	s.auditLocked(user.Name, "review_request.create", "change_request", request.ID, request.ProjectID, map[string]any{
		"environment": request.Environment,
		"configKey":   request.ConfigKey,
		"reason":      request.Reason,
	})
	if err := s.persistLocked(); err != nil {
		return model.ReviewRequest{}, err
	}
	return *request, nil
}

func (s *Store) SetReviewStatus(user model.User, requestID, status, comment string) (model.ReviewRequest, *AppError) {
	if !canReview(user) {
		return model.ReviewRequest{}, forbidden(fmt.Sprintf("Role %q is not allowed", user.Role))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	request, ok := s.reviews[requestID]
	if !ok {
		return model.ReviewRequest{}, notFound(fmt.Sprintf("Review request %q not found", requestID))
	}
	request.Status = status
	request.Reviewer = user.Name
	request.Comment = strings.TrimSpace(comment)
	request.UpdatedAt = time.Now().UTC()
	s.auditLocked(user.Name, "review_request."+status, "change_request", request.ID, request.ProjectID, map[string]any{
		"comment": request.Comment,
	})
	if err := s.persistLocked(); err != nil {
		return model.ReviewRequest{}, err
	}
	return *request, nil
}

func (s *Store) ValidateProject(projectID string, req model.ValidateProjectRequest) (model.ValidationResult, *AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return model.ValidationResult{}, notFound(fmt.Sprintf("model.Project %q not found", projectID))
	}

	targetEnvs := make([]string, 0)
	if strings.TrimSpace(req.Environment) != "" {
		if err := s.ensureEnvironmentLocked(projectID, req.Environment); err != nil {
			return model.ValidationResult{}, err
		}
		targetEnvs = append(targetEnvs, req.Environment)
	} else {
		envs := append([]model.ProjectEnvironment(nil), project.Environments...)
		sort.Slice(envs, func(i, j int) bool { return envs[i].SortOrder < envs[j].SortOrder })
		for _, env := range envs {
			targetEnvs = append(targetEnvs, env.Name)
		}
	}

	entries := make([]model.ValidationEntry, 0)
	for _, entry := range s.configs {
		if entry.ProjectID == projectID && contains(targetEnvs, entry.Environment) {
			entries = append(entries, model.ValidationEntry{
				Environment: entry.Environment,
				Key:         entry.Key,
				Value:       entry.Value,
				ValueType:   entry.ValueType,
				IsSensitive: entry.IsSensitive,
			})
		}
	}
	for _, draft := range req.DraftEntries {
		if contains(targetEnvs, draft.Environment) {
			entries = append(entries, model.ValidationEntry{
				Environment: draft.Environment,
				Key:         draft.Key,
				Value:       draft.Value,
				ValueType:   fallback(draft.ValueType, "string"),
				IsSensitive: draft.IsSensitive,
			})
		}
	}

	return validateEntries(projectID, targetEnvs, entries), nil
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

func (s *Store) ensureEnvironmentLocked(projectID, environment string) *AppError {
	project, ok := s.projects[projectID]
	if !ok {
		return notFound(fmt.Sprintf("model.Project %q not found", projectID))
	}
	for _, env := range project.Environments {
		if env.Name == environment {
			return nil
		}
	}
	return notFound(fmt.Sprintf("Environment %q not found for project %q", environment, projectID))
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
		ID:           newID("ver"),
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
		ID:           newID("aud"),
		Actor:        actor,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ProjectID:    projectID,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	})
}

func cloneProject(project model.Project) model.Project {
	project.Environments = append([]model.ProjectEnvironment(nil), project.Environments...)
	return project
}

func normalizeEnvironments(environments []string) []string {
	if len(environments) == 0 {
		return []string{"dev", "staging", "prod"}
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(environments))
	for _, env := range environments {
		name := strings.ToLower(strings.TrimSpace(env))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return []string{"dev", "staging", "prod"}
	}
	return normalized
}

func newID(prefix string) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
