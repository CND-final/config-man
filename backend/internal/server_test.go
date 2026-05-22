package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Fatalf("ping database: %v", err)
	}
	return db
}

func resetServerTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		TRUNCATE TABLE
			audit_logs,
			review_requests,
			config_revisions,
			config_versions,
			config_entries,
			project_environments,
			projects,
			custom_templates
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
	res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{"name": "billing-service", "ownerName": "Billing Team"})
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
		"ownerName":  "Platform Team",
		"templateId": created.ID,
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
		"ownerName":  "Platform Team",
		"templateId": created.ID,
	})
	if res.Code != http.StatusNotFound {
		t.Fatalf("create project with another user template status = %d body=%s", res.Code, res.Body.String())
	}
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
