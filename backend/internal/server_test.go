package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"config-man/backend/internal/processor"
	"config-man/backend/internal/store"
	"config-man/backend/model"
	"config-man/backend/pkg/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	db := openServerTestDB(t)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if _, err := store.NewStoreWithDB(ctx, db); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	resetServerTestDB(t, db)

	dataStore, err := store.NewStoreWithDB(ctx, db)
	if err != nil {
		t.Fatalf("initialize store: %v", err)
	}
	proc, err := processor.NewProcessor(dataStore)
	if err != nil {
		t.Fatalf("initialize processor: %v", err)
	}
	log := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	server, err := NewServer(proc, log, config.Config{})
	if err != nil {
		t.Fatalf("initialize server: %v", err)
	}
	return server.Handler()
}

func openServerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed server test")
	}
	adminDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := adminDB.PingContext(ctx); err != nil {
		_ = adminDB.Close()
		t.Fatalf("ping database: %v", err)
	}

	schemaName := fmt.Sprintf("test_%d_%d", os.Getpid(), time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, `CREATE SCHEMA `+quoteIdentifier(schemaName)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create isolated schema: %v", err)
	}

	isolatedURL, err := isolatedDatabaseURL(databaseURL, schemaName)
	if err != nil {
		_ = adminDB.Close()
		t.Fatalf("build isolated database url: %v", err)
	}

	db, err := sql.Open("pgx", isolatedURL)
	if err != nil {
		_ = adminDB.Close()
		t.Fatalf("open isolated database: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		_ = adminDB.Close()
		t.Fatalf("ping isolated database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = adminDB.ExecContext(cleanupCtx, `DROP SCHEMA IF EXISTS `+quoteIdentifier(schemaName)+` CASCADE`)
		_ = adminDB.Close()
	})
	return db
}

func resetServerTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		TRUNCATE TABLE
			audit_logs,
			app_notifications,
			shared_config_update_requests,
			shared_config_entries,
			shared_configs,
			review_requests,
			config_revisions,
			config_versions,
			config_entries,
			project_members,
			project_environments,
			projects,
			custom_templates,
			group_members,
			groups
		CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func request(t *testing.T, handler http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func decodeBody[T any](t *testing.T, res *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(res.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response body: %v body=%s", err, res.Body.String())
	}
	return value
}

func TestLoginAndMe(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPost, "/api/v1/auth/login", "", map[string]any{"email": "admin@config-man.local", "password": "password"})
	if res.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[map[string]any](t, res)
	if payload["token"] != "alice" {
		t.Fatalf("token = %v", payload["token"])
	}

	res = request(t, handler, http.MethodGet, "/api/v1/auth/me", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestProtectedRoutesRequireAuthentication(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodGet, "/api/v1/projects", "", nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("protected route status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestCreateProjectCreatesDefaultEnvironments(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{"name": "billing-service", "groupId": "platform-team"})
	if res.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body=%s", res.Code, res.Body.String())
	}
	project := decodeBody[model.Project](t, res)
	if len(project.Environments) != 3 {
		t.Fatalf("environment count = %d", len(project.Environments))
	}
	if project.Environments[0].Name != "dev" || project.Environments[2].Name != "prod" {
		t.Fatalf("unexpected environments: %#v", project.Environments)
	}
}

func TestCreateGroupAndManageMembers(t *testing.T) {
	handler := newTestHandler(t)

	res := request(t, handler, http.MethodGet, "/api/v1/users", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list users status = %d body=%s", res.Code, res.Body.String())
	}
	users := decodeBody[struct {
		Users []model.User `json:"users"`
	}](t, res)
	if len(users.Users) == 0 {
		t.Fatalf("expected demo users")
	}

	res = request(t, handler, http.MethodPost, "/api/v1/groups", "alice", map[string]any{
		"name":      "Platform Owners",
		"memberIds": []string{"paul"},
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create group status = %d body=%s", res.Code, res.Body.String())
	}
	group := decodeBody[model.Group](t, res)
	if group.ID == "" || group.Name != "Platform Owners" || !groupHasUser(group, "paul") {
		t.Fatalf("unexpected created group: %#v", group)
	}

	res = request(t, handler, http.MethodPost, "/api/v1/groups/"+group.ID+"/members", "alice", map[string]any{"userId": "nora"})
	if res.Code != http.StatusOK {
		t.Fatalf("add group member status = %d body=%s", res.Code, res.Body.String())
	}
	group = decodeBody[model.Group](t, res)
	if !groupHasUser(group, "paul") || !groupHasUser(group, "nora") {
		t.Fatalf("group members not updated: %#v", group)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/groups/"+group.ID, "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("get group status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Group model.Group `json:"group"`
	}](t, res)
	if payload.Group.MemberCount != 2 || !groupHasUser(payload.Group, "nora") {
		t.Fatalf("unexpected group detail: %#v", payload.Group)
	}

	res = request(t, handler, http.MethodDelete, "/api/v1/groups/"+group.ID+"/members/nora", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("remove group member status = %d body=%s", res.Code, res.Body.String())
	}
	group = decodeBody[model.Group](t, res)
	if groupHasUser(group, "nora") {
		t.Fatalf("member was not removed: %#v", group)
	}
}

func TestProjectMembershipControlsProjectAccess(t *testing.T) {
	handler := newTestHandler(t)

	res := request(t, handler, http.MethodGet, "/api/v1/projects", "vincent", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list projects as viewer status = %d body=%s", res.Code, res.Body.String())
	}
	viewerProjects := decodeBody[[]model.Project](t, res)
	if !projectListHasID(viewerProjects, "customer-portal") {
		t.Fatalf("viewer should see member project: %#v", viewerProjects)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects", "grace", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list projects as unassigned group-admin status = %d body=%s", res.Code, res.Body.String())
	}
	groupAdminProjects := decodeBody[[]model.Project](t, res)
	if len(groupAdminProjects) != 0 {
		t.Fatalf("unassigned group-admin should not see projects: %#v", groupAdminProjects)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/members", "nora", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list members as developer status = %d body=%s", res.Code, res.Body.String())
	}
	membersPayload := decodeBody[struct {
		Members []model.ProjectMember `json:"members"`
	}](t, res)
	if !projectMembersHaveRole(membersPayload.Members, "paul", model.RoleProjectMemberAdmin) {
		t.Fatalf("seeded project_admin missing: %#v", membersPayload.Members)
	}

	res = request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/members", "nora", map[string]any{
		"members": []map[string]any{{"userId": "paul", "projectRole": "project_admin"}},
	})
	if res.Code != http.StatusForbidden {
		t.Fatalf("developer updated project members status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/members", "paul", map[string]any{
		"members": []map[string]any{
			{"userId": "paul", "projectRole": "project_admin"},
			{"userId": "grace", "projectRole": "viewer"},
		},
	})
	if res.Code != http.StatusOK {
		t.Fatalf("project admin updated members status = %d body=%s", res.Code, res.Body.String())
	}
	membersPayload = decodeBody[struct {
		Members []model.ProjectMember `json:"members"`
	}](t, res)
	if !projectMembersHaveRole(membersPayload.Members, "grace", model.RoleProjectViewer) {
		t.Fatalf("grace viewer membership missing after update: %#v", membersPayload.Members)
	}
}

func projectListHasID(projects []model.Project, projectID string) bool {
	for _, project := range projects {
		if project.ID == projectID {
			return true
		}
	}
	return false
}

func projectMembersHaveRole(members []model.ProjectMember, userID string, role model.ProjectRole) bool {
	for _, member := range members {
		if member.ID == userID && member.ProjectRole == role {
			return true
		}
	}
	return false
}

func TestListTemplatesIncludesBaseInfrastructureTemplates(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodGet, "/api/v1/templates", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list templates status = %d body=%s", res.Code, res.Body.String())
	}
	templates := decodeBody[[]model.Template](t, res)

	spring, ok := findTemplateByID(templates, "spring-boot-base-template")
	if !ok || spring.Format != "yaml" || !templateHasVariable(spring, "DB_SECRET") {
		t.Fatalf("spring boot template invalid or missing: %#v", templates)
	}

	logging, ok := findTemplateByID(templates, "global-logging-template")
	if !ok || logging.Format != "yaml" || !templateHasVariable(logging, "LOG_LEVEL_ROOT") || !templateHasVariable(logging, "APP_NAME") {
		t.Fatalf("global logging template invalid or missing: %#v", templates)
	}
	if templateListHasID(templates, "group-base-template") {
		t.Fatalf("old group base template should not be exposed: %#v", templates)
	}
	if !strings.Contains(logging.Body, "max-size: 10MB") || strings.Contains(logging.Body, "#") {
		t.Fatalf("global logging template body was not updated or still has comments: %q", logging.Body)
	}
}

func TestCreateTemplateIsPrivateToActor(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPost, "/api/v1/templates", "alice", map[string]any{
		"name":        "Alice Spring Template",
		"description": "private template",
		"format":      "yaml",
		"body":        "server:\n  port: ${APP_PORT}\n",
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create template status = %d body=%s", res.Code, res.Body.String())
	}
	created := decodeBody[model.Template](t, res)
	if !created.IsCustom || created.OwnerUserID != "alice" || !templateHasVariable(created, "APP_PORT") {
		t.Fatalf("unexpected custom template: %#v", created)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/templates", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list alice templates status = %d body=%s", res.Code, res.Body.String())
	}
	aliceTemplates := decodeBody[[]model.Template](t, res)
	if !templateListHasID(aliceTemplates, created.ID) {
		t.Fatalf("alice cannot see own template: %#v", aliceTemplates)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/templates", "paul", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list paul templates status = %d body=%s", res.Code, res.Body.String())
	}
	paulTemplates := decodeBody[[]model.Template](t, res)
	if templateListHasID(paulTemplates, created.ID) {
		t.Fatalf("paul should not see alice template: %#v", paulTemplates)
	}

	res = request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
		"name":       "templated-service",
		"templateId": created.ID,
		"groupId":    "platform-team",
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create project with own template status = %d body=%s", res.Code, res.Body.String())
	}
	project := decodeBody[model.Project](t, res)
	if project.TemplateID != created.ID {
		t.Fatalf("template id = %q want %q", project.TemplateID, created.ID)
	}

	res = request(t, handler, http.MethodPost, "/api/v1/projects", "paul", map[string]any{
		"name":       "forbidden-template-service",
		"templateId": created.ID,
		"groupId":    "platform-team",
	})
	if res.Code != http.StatusNotFound {
		t.Fatalf("create project with another user template status = %d body=%s", res.Code, res.Body.String())
	}
}

func groupHasUser(group model.Group, userID string) bool {
	for _, member := range group.Members {
		if member.ID == userID {
			return true
		}
	}
	return false
}

func findTemplateByID(templates []model.Template, id string) (model.Template, bool) {
	for _, template := range templates {
		if template.ID == id {
			return template, true
		}
	}
	return model.Template{}, false
}

func templateHasVariable(template model.Template, name string) bool {
	for _, variable := range template.Variables {
		if variable.Name == name {
			return true
		}
	}
	return false
}

func templateListHasID(templates []model.Template, id string) bool {
	for _, template := range templates {
		if template.ID == id {
			return true
		}
	}
	return false
}

func TestSensitiveConfigIsMaskedByDefault(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/configs?env=prod", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list configs status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Entries []model.ConfigEntry `json:"entries"`
	}](t, res)
	for _, entry := range payload.Entries {
		if entry.Key == "database.url" && entry.Value != "******" {
			t.Fatalf("sensitive value was not masked: %q", entry.Value)
		}
	}
}

func TestConfigVersionHistoryAndRollback(t *testing.T) {
	handler := newTestHandler(t)
	nextValue := "warn"
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level", "alice", map[string]any{
		"value":        nextValue,
		"changeReason": "test update",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("update config status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level/versions", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list versions status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Versions []model.ConfigVersion `json:"versions"`
	}](t, res)
	if len(payload.Versions) < 2 {
		t.Fatalf("version count = %d", len(payload.Versions))
	}
	if payload.Versions[0].OldValue == nil || *payload.Versions[0].OldValue != "info" || payload.Versions[0].NewValue != nextValue {
		t.Fatalf("unexpected latest version: %#v", payload.Versions[0])
	}

	res = request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level/rollback", "alice", map[string]any{
		"versionId":    payload.Versions[0].ID,
		"changeReason": "test rollback",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("rollback status = %d body=%s", res.Code, res.Body.String())
	}
	entry := decodeBody[model.ConfigEntry](t, res)
	if entry.Value != "info" {
		t.Fatalf("rollback value = %q", entry.Value)
	}
}

func TestConfigHistoryRevisionsAndRollback(t *testing.T) {
	handler := newTestHandler(t)
	nextValue := "warn"
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level", "alice", map[string]any{
		"value":        nextValue,
		"changeReason": "revision update",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("update config status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/config-history?env=staging", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list config history status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Revisions []model.ConfigRevision `json:"revisions"`
	}](t, res)
	if len(payload.Revisions) < 2 {
		t.Fatalf("revision count = %d", len(payload.Revisions))
	}
	if !revisionHasValue(payload.Revisions[0], "log.level", nextValue) {
		t.Fatalf("latest revision did not include updated log.level: %#v", payload.Revisions[0])
	}

	res = request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/config-history/rollback", "alice", map[string]any{
		"environment":  "staging",
		"revisionId":   payload.Revisions[1].ID,
		"changeReason": "revision rollback",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("rollback revision status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/configs?env=staging", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list configs status = %d body=%s", res.Code, res.Body.String())
	}
	configs := decodeBody[struct {
		Entries []model.ConfigEntry `json:"entries"`
	}](t, res)
	found := false
	for _, entry := range configs.Entries {
		if entry.Key == "log.level" {
			found = true
			if entry.Value != "info" {
				t.Fatalf("rollback revision value = %q", entry.Value)
			}
		}
	}
	if !found {
		t.Fatalf("log.level config missing after rollback")
	}
}

func revisionHasValue(revision model.ConfigRevision, key, value string) bool {
	for _, entry := range revision.Entries {
		if entry.Key == key && entry.Value == value {
			return true
		}
	}
	return false
}

func TestUpdateConfigKey(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level", "alice", map[string]any{
		"key":          "logging.level",
		"changeReason": "rename key",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("update key status = %d body=%s", res.Code, res.Body.String())
	}
	entry := decodeBody[model.ConfigEntry](t, res)
	if entry.Key != "logging.level" {
		t.Fatalf("updated key = %q", entry.Key)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/configs?env=staging", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list configs status = %d body=%s", res.Code, res.Body.String())
	}
	configs := decodeBody[struct {
		Entries []model.ConfigEntry `json:"entries"`
	}](t, res)
	if !configEntriesHaveKey(configs.Entries, "logging.level") || configEntriesHaveKey(configs.Entries, "log.level") {
		t.Fatalf("unexpected keys after rename: %#v", configs.Entries)
	}
}

func configEntriesHaveKey(entries []model.ConfigEntry, key string) bool {
	for _, entry := range entries {
		if entry.Key == key {
			return true
		}
	}
	return false
}

func TestDeveloperCannotModifyProdConfig(t *testing.T) {
	handler := newTestHandler(t)
	value := "https://api2.example.com"
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-prod-api-baseurl", "nora", map[string]any{"value": value})
	if res.Code != http.StatusForbidden {
		t.Fatalf("update prod as developer status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestImportJSONConfig(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/configs/import", "alice", map[string]any{"environment": "dev", "format": "json", "content": `{"feature":{"checkout":true},"limit":3}`})
	if res.Code != http.StatusCreated {
		t.Fatalf("import status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[map[string]any](t, res)
	if payload["imported"].(float64) != 2 {
		t.Fatalf("imported = %v", payload["imported"])
	}
}

func TestExtractConfigDoesNotPersist(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/configs/extract", "alice", map[string]any{
		"environment": "dev",
		"format":      "json",
		"content":     `{"feature":{"preview":true},"limit":3}`,
	})
	if res.Code != http.StatusOK {
		t.Fatalf("extract status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Entries    []model.ConfigRevisionEntry `json:"entries"`
		EntryCount int                         `json:"entryCount"`
		Created    int                         `json:"created"`
	}](t, res)
	if payload.EntryCount != 2 || len(payload.Entries) != 2 || payload.Created != 2 {
		t.Fatalf("unexpected extract payload: %#v", payload)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/configs?env=dev", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list configs status = %d body=%s", res.Code, res.Body.String())
	}
	configs := decodeBody[struct {
		Entries []model.ConfigEntry `json:"entries"`
	}](t, res)
	for _, entry := range configs.Entries {
		if entry.Key == "feature.preview" || entry.Key == "limit" {
			t.Fatalf("extract persisted config unexpectedly: %#v", entry)
		}
	}
}

func TestGlobalSharedConfigPermissions(t *testing.T) {
	handler := newTestHandler(t)

	res := request(t, handler, http.MethodGet, "/api/v1/shared-configs", "nora", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list shared configs status = %d body=%s", res.Code, res.Body.String())
	}
	items := decodeBody[[]model.SharedConfig](t, res)
	if len(items) == 0 {
		t.Fatalf("expected seeded shared configs")
	}

	body := map[string]any{
		"name":        "Global Test Defaults",
		"description": "test shared config",
		"format":      "yaml",
		"entries": []map[string]any{{
			"key":         "feature.test",
			"value":       "true",
			"valueType":   "boolean",
			"environment": "prod",
		}},
	}
	res = request(t, handler, http.MethodPost, "/api/v1/shared-configs", "nora", body)
	if res.Code != http.StatusForbidden {
		t.Fatalf("developer create shared config status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodPost, "/api/v1/shared-configs", "alice", body)
	if res.Code != http.StatusCreated {
		t.Fatalf("system_admin create shared config status = %d body=%s", res.Code, res.Body.String())
	}
	created := decodeBody[model.SharedConfig](t, res)
	if created.Scope != model.ScopeGlobal {
		t.Fatalf("created scope = %q", created.Scope)
	}

	res = request(t, handler, http.MethodPost, "/api/v1/shared-configs/"+created.ID+"/submit-update", "nora", map[string]any{
		"reason": "request safer default",
		"entries": []map[string]any{{
			"key":         "feature.test",
			"value":       "false",
			"valueType":   "boolean",
			"environment": "prod",
		}},
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("submit shared config update status = %d body=%s", res.Code, res.Body.String())
	}
	updateRequest := decodeBody[model.SharedConfigUpdateRequest](t, res)
	if updateRequest.Status != "pending" || updateRequest.SharedConfigID != created.ID {
		t.Fatalf("unexpected update request: %#v", updateRequest)
	}

	res = request(t, handler, http.MethodPut, "/api/v1/shared-configs/"+created.ID, "nora", map[string]any{
		"name":         created.Name,
		"format":       created.Format,
		"changeReason": "apply directly",
		"entries":      body["entries"],
	})
	if res.Code != http.StatusForbidden {
		t.Fatalf("developer update shared config status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodPut, "/api/v1/shared-configs/global-runtime-defaults", "alice", map[string]any{
		"name":         "Global Runtime Defaults",
		"description":  "Updated defaults",
		"format":       "yaml",
		"changeReason": "tighten global runtime defaults",
		"entries": []map[string]any{{
			"key":         "logging.level.root",
			"value":       "WARN",
			"valueType":   "string",
			"environment": "prod",
		}},
	})
	if res.Code != http.StatusOK {
		t.Fatalf("system_admin update seeded shared config status = %d body=%s", res.Code, res.Body.String())
	}
	updated := decodeBody[model.SharedConfig](t, res)
	if updated.ProdEnvironmentCount != 1 {
		t.Fatalf("prod impact = %d", updated.ProdEnvironmentCount)
	}

	res = request(t, handler, http.MethodGet, "/api/v1/notifications", "nora", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list notifications status = %d body=%s", res.Code, res.Body.String())
	}
	notifications := decodeBody[[]model.Notification](t, res)
	if len(notifications) == 0 {
		t.Fatalf("expected notification for affected project member")
	}

	res = request(t, handler, http.MethodDelete, "/api/v1/shared-configs/"+created.ID, "nora", nil)
	if res.Code != http.StatusForbidden {
		t.Fatalf("developer delete shared config status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodDelete, "/api/v1/shared-configs/"+created.ID, "alice", nil)
	if res.Code != http.StatusNoContent {
		t.Fatalf("system_admin delete shared config status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestReviewRequestLifecycle(t *testing.T) {
	handler := newTestHandler(t)
	res := request(t, handler, http.MethodPost, "/api/v1/review-requests", "nora", map[string]any{"projectId": "customer-portal", "environment": "prod", "configKey": "database.url", "reason": "Rotate database credential"})
	if res.Code != http.StatusCreated {
		t.Fatalf("create review status = %d body=%s", res.Code, res.Body.String())
	}
	created := decodeBody[model.ReviewRequest](t, res)

	res = request(t, handler, http.MethodPut, "/api/v1/review-requests/"+created.ID+"/approve", "rachel", map[string]any{"comment": "looks good"})
	if res.Code != http.StatusOK {
		t.Fatalf("approve status = %d body=%s", res.Code, res.Body.String())
	}
	approved := decodeBody[model.ReviewRequest](t, res)
	if approved.Status != "approved" || approved.Reviewer != "Rachel Kao" {
		t.Fatalf("unexpected approved request: %#v", approved)
	}
}

func isolatedDatabaseURL(baseURL, schemaName string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("options", "-c search_path="+schemaName+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
