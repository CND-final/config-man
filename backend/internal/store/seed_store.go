package store

import (
	"strings"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

type seedData struct {
	projects  map[string]*model.Project
	configs   map[string]*model.ConfigEntry
	reviews   map[string]*model.ReviewRequest
	templates map[string]*model.Template
	versions  []model.ConfigVersion
	revisions []model.ConfigRevision
	audits    []model.AuditLog
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

func demoSeedData() seedData {
	now := time.Now().UTC()
	seed := seedData{
		projects:  make(map[string]*model.Project),
		configs:   make(map[string]*model.ConfigEntry),
		reviews:   make(map[string]*model.ReviewRequest),
		templates: make(map[string]*model.Template),
	}

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
	seed.projects[project.ID] = project

	entries := []model.ConfigEntry{
		{ID: "cfg-dev-api-baseurl", ProjectID: project.ID, Environment: "dev", Key: "api.baseUrl", Value: "https://dev-api.example.com", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-dev-log-level", ProjectID: project.ID, Environment: "dev", Key: "log.level", Value: "debug", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-staging-api-baseurl", ProjectID: project.ID, Environment: "staging", Key: "api.baseUrl", Value: "https://staging-api.example.com", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-staging-log-level", ProjectID: project.ID, Environment: "staging", Key: "log.level", Value: "info", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-prod-api-baseurl", ProjectID: project.ID, Environment: "prod", Key: "api.baseUrl", Value: "https://api.example.com", ValueType: "string", UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
		{ID: "cfg-prod-database-url", ProjectID: project.ID, Environment: "prod", Key: "database.url", Value: "postgresql://prod-user:secret@prod-db:5432/app", ValueType: "string", IsSensitive: true, UpdatedBy: "seed-admin", CreatedAt: now, UpdatedAt: now},
	}
	for index := range entries {
		entry := entries[index]
		seed.configs[entry.ID] = &entry
		seed.versions = append(seed.versions, model.ConfigVersion{
			ID:           util.NewID("ver"),
			ConfigID:     entry.ID,
			NewValue:     entry.Value,
			ChangedBy:    "seed-admin",
			ChangeReason: "seed demo config",
			CreatedAt:    now,
		})
	}
	for _, env := range project.Environments {
		seed.revisions = append(seed.revisions, configRevisionFromEntries(project.ID, env.Name, "seed-admin", "seed demo config", entries, ""))
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
	seed.reviews[review.ID] = review
	seed.audits = append(seed.audits, model.AuditLog{
		ID:           util.NewID("aud"),
		Actor:        "seed-admin",
		Action:       "seed_demo_project",
		ResourceType: "project",
		ResourceID:   project.ID,
		ProjectID:    project.ID,
		Metadata:     map[string]any{"projectName": project.Name},
		CreatedAt:    now,
	})
	return seed
}
