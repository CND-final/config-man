package store

import (
	"strings"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

type seedData struct {
	groups      map[string]*model.Group
	projects    map[string]*model.Project
	configFiles map[string]*model.Config
	configs     map[string]*model.ConfigEntry
	reviews     map[string]*model.ReviewRequest
	templates   map[string]*model.Template
	versions    []model.ConfigVersion
	revisions   []model.ConfigRevision
	audits      []model.AuditLog
}

func (s *Store) seedUsers() {
	users := []model.User{
		{ID: "alice", Email: "admin@config-man.local", Name: "Alice Lin", Role: model.RoleSystemAdmin},
		{ID: "paul", Email: "project-admin@config-man.local", Name: "Paul Wu", Role: model.RoleProjectAdmin},
		{ID: "grace", Email: "group-admin@config-man.local", Name: "Grace Huang", Role: model.RoleUserGroupAdmin},
		{ID: "nora", Email: "developer@config-man.local", Name: "Nora Chen", Role: model.RoleDeveloper},
		{ID: "rachel", Email: "reviewer@config-man.local", Name: "Rachel Kao", Role: model.RoleReviewer},
		{ID: "vincent", Email: "viewer@config-man.local", Name: "Vincent Lee", Role: model.RoleViewer},
		{ID: "oliver", Email: "platform-viewer@config-man.local", Name: "Oliver Chen", Role: model.RoleViewer},
		{ID: "ethan", Email: "commerce-developer@config-man.local", Name: "Ethan Liu", Role: model.RoleDeveloper},
		{ID: "sophia", Email: "commerce-reviewer@config-man.local", Name: "Sophia Wang", Role: model.RoleReviewer},
		{ID: "mia", Email: "commerce-viewer@config-man.local", Name: "Mia Tsai", Role: model.RoleViewer},
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
		groups:      make(map[string]*model.Group),
		projects:    make(map[string]*model.Project),
		configFiles: make(map[string]*model.Config),
		configs:     make(map[string]*model.ConfigEntry),
		reviews:     make(map[string]*model.ReviewRequest),
		templates:   make(map[string]*model.Template),
	}

	platformGroup := &model.Group{
		ID:   "platform-team",
		Name: "Platform Team",
		Members: []model.GroupMember{
			{User: model.User{ID: "paul"}, GroupRole: model.RoleGroupAdmin},
			{User: model.User{ID: "nora"}, GroupRole: model.RoleGroupMember},
			{User: model.User{ID: "rachel"}, GroupRole: model.RoleGroupMember},
			{User: model.User{ID: "vincent"}, GroupRole: model.RoleGroupMember},
			{User: model.User{ID: "oliver"}, GroupRole: model.RoleGroupMember},
		},
	}
	commerceGroup := &model.Group{
		ID:   "commerce-team",
		Name: "Commerce Team",
		Members: []model.GroupMember{
			{User: model.User{ID: "grace"}, GroupRole: model.RoleGroupAdmin},
			{User: model.User{ID: "ethan"}, GroupRole: model.RoleGroupMember},
			{User: model.User{ID: "sophia"}, GroupRole: model.RoleGroupMember},
			{User: model.User{ID: "mia"}, GroupRole: model.RoleGroupMember},
		},
	}
	seed.groups[platformGroup.ID] = platformGroup
	seed.groups[commerceGroup.ID] = commerceGroup

	customerPortal := &model.Project{
		ID:          "customer-portal",
		Name:        "customer-portal",
		Description: "Customer-facing web portal owned by Platform Team",
		RepoURL:     "https://git.example.com/platform/customer-portal",
		GroupID:     platformGroup.ID,
		Environments: []model.ProjectEnvironment{
			{ID: "env-customer-dev", Name: "dev", SortOrder: 1},
			{ID: "env-customer-staging", Name: "staging", SortOrder: 2},
			{ID: "env-customer-prod", Name: "prod", SortOrder: 3},
		},
		Members: []model.ProjectMember{
			{User: model.User{ID: "paul"}, ProjectRole: model.RoleProjectMemberAdmin},
			{User: model.User{ID: "nora"}, ProjectRole: model.RoleProjectDeveloper},
			{User: model.User{ID: "rachel"}, ProjectRole: model.RoleProjectReviewer},
			{User: model.User{ID: "vincent"}, ProjectRole: model.RoleProjectViewer},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	billingAPI := &model.Project{
		ID:          "billing-api",
		Name:        "billing-api",
		Description: "Billing service owned by Commerce Team",
		RepoURL:     "https://git.example.com/commerce/billing-api",
		GroupID:     commerceGroup.ID,
		Environments: []model.ProjectEnvironment{
			{ID: "env-billing-dev", Name: "dev", SortOrder: 1},
			{ID: "env-billing-staging", Name: "staging", SortOrder: 2},
			{ID: "env-billing-prod", Name: "prod", SortOrder: 3},
		},
		Members: []model.ProjectMember{
			{User: model.User{ID: "grace"}, ProjectRole: model.RoleProjectMemberAdmin},
			{User: model.User{ID: "ethan"}, ProjectRole: model.RoleProjectDeveloper},
			{User: model.User{ID: "sophia"}, ProjectRole: model.RoleProjectReviewer},
			{User: model.User{ID: "mia"}, ProjectRole: model.RoleProjectViewer},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	seed.projects[customerPortal.ID] = customerPortal
	seed.projects[billingAPI.ID] = billingAPI

	addConfig := func(project *model.Project, name, description, sourceType, sourceID string, sortOrder int) string {
		config := model.Config{
			ID:          model.ConfigID(project.ID, name),
			ProjectID:   project.ID,
			Name:        model.NormalizeConfigName(name),
			Description: description,
			SourceType:  sourceType,
			SourceID:    sourceID,
			Prefix:      model.ConfigPrefix(name),
			SortOrder:   sortOrder,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		seed.configFiles[config.ID] = &config
		return config.ID
	}
	customerApp := addConfig(customerPortal, "application.yaml", "Application settings", "seed", "", 1)
	customerRuntime := addConfig(customerPortal, "runtime-defaults.yaml", "Shared global runtime defaults", "shared-config", "global-runtime-defaults", 2)
	customerSecurity := addConfig(customerPortal, "security.json", "Security settings", "seed", "", 3)
	billingService := addConfig(billingAPI, "service.yaml", "Service settings", "seed", "", 1)
	billingRuntime := addConfig(billingAPI, "runtime-defaults.yaml", "Shared global runtime defaults", "shared-config", "global-runtime-defaults", 2)
	billingPayment := addConfig(billingAPI, "payment.yaml", "Payment provider settings", "seed", "", 3)

	addEntry := func(project *model.Project, configID, environment, key, value, valueType string, sensitive bool) {
		entry := model.ConfigEntry{
			ID:          util.NewID("cfg"),
			ProjectID:   project.ID,
			Environment: environment,
			ConfigID:    configID,
			Key:         key,
			Value:       value,
			ValueType:   valueType,
			IsSensitive: sensitive,
			UpdatedBy:   "seed-admin",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		seed.configs[entry.ID] = &entry
		seed.versions = append(seed.versions, model.ConfigVersion{
			ID:            util.NewID("ver"),
			ConfigEntryID: entry.ID,
			NewValue:      entry.Value,
			ChangedBy:     "seed-admin",
			ChangeReason:  "seed demo config",
			CreatedAt:     now,
		})
	}
	addEntriesForAllEnvs := func(project *model.Project, configID, key, dev, staging, prod, valueType string, sensitive bool) {
		addEntry(project, configID, "dev", key, dev, valueType, sensitive)
		addEntry(project, configID, "staging", key, staging, valueType, sensitive)
		addEntry(project, configID, "prod", key, prod, valueType, sensitive)
	}

	addEntriesForAllEnvs(customerPortal, customerApp, "api.baseUrl", "https://dev-api.example.com", "https://staging-api.example.com", "https://api.example.com", "string", false)
	addEntriesForAllEnvs(customerPortal, customerApp, "log.level", "debug", "info", "info", "string", false)
	addEntriesForAllEnvs(customerPortal, customerApp, "feature.checkout.enabled", "true", "true", "true", "boolean", false)

	addEntriesForAllEnvs(customerPortal, customerRuntime, "logging.level.root", "DEBUG", "INFO", "INFO", "string", false)
	addEntriesForAllEnvs(customerPortal, customerRuntime, "observability.metrics.enabled", "false", "true", "true", "boolean", false)
	addEntriesForAllEnvs(customerPortal, customerRuntime, "security.headers.enabled", "false", "true", "true", "boolean", false)

	addEntriesForAllEnvs(customerPortal, customerSecurity, "database.url", "postgresql://dev-user:dev-secret@dev-db:5432/app", "postgresql://staging-user:staging-secret@staging-db:5432/app", "postgresql://prod-user:prod-secret@prod-db:5432/app", "string", true)
	addEntriesForAllEnvs(customerPortal, customerSecurity, "auth.jwt.issuer", "customer-portal-dev", "customer-portal-staging", "customer-portal", "string", false)

	addEntriesForAllEnvs(billingAPI, billingService, "service.port", "8081", "8081", "8080", "number", false)
	addEntriesForAllEnvs(billingAPI, billingService, "log.level", "debug", "info", "warn", "string", false)
	addEntriesForAllEnvs(billingAPI, billingService, "worker.concurrency", "2", "4", "8", "number", false)

	addEntriesForAllEnvs(billingAPI, billingRuntime, "logging.level.root", "DEBUG", "INFO", "INFO", "string", false)
	addEntriesForAllEnvs(billingAPI, billingRuntime, "observability.metrics.enabled", "false", "true", "true", "boolean", false)
	addEntriesForAllEnvs(billingAPI, billingRuntime, "security.headers.enabled", "false", "true", "true", "boolean", false)

	addEntriesForAllEnvs(billingAPI, billingPayment, "payment.provider", "stripe-sandbox", "stripe-sandbox", "stripe", "string", false)
	addEntriesForAllEnvs(billingAPI, billingPayment, "payment.apiKey", "sk_test_dev_demo_secret", "sk_test_staging_demo_secret", "sk_live_demo_secret", "string", true)
	addEntriesForAllEnvs(billingAPI, billingPayment, "payment.webhook.enabled", "false", "true", "true", "boolean", false)

	for _, project := range []*model.Project{customerPortal, billingAPI} {
		for _, env := range project.Environments {
			seed.revisions = append(seed.revisions, configRevisionFromEntries(project.ID, env.Name, "seed-admin", "seed demo config", entriesForProjectEnv(seed.configs, project.ID, env.Name), ""))
		}
		seed.audits = append(seed.audits, model.AuditLog{
			ID:           util.NewID("aud"),
			Actor:        "seed-admin",
			Action:       "seed_demo_project",
			ResourceType: "project",
			ResourceID:   project.ID,
			ProjectID:    project.ID,
			Metadata:     map[string]any{"projectName": project.Name, "groupId": project.GroupID},
			CreatedAt:    now,
		})
	}

	review := &model.ReviewRequest{
		ID:          "seed-prod-database-review",
		ProjectID:   customerPortal.ID,
		ProjectName: customerPortal.Name,
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
	return seed
}

func entriesForProjectEnv(entries map[string]*model.ConfigEntry, projectID, environment string) []model.ConfigEntry {
	out := make([]model.ConfigEntry, 0)
	for _, entry := range entries {
		if entry.ProjectID == projectID && entry.Environment == environment {
			out = append(out, *entry)
		}
	}
	return out
}
