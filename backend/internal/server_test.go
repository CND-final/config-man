package app

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"config-man/backend/internal/processor"
	"config-man/backend/model"
	"config-man/backend/pkg/config"
)

func newTestHandler() http.Handler {
	log := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	server, err := NewServer(processor.NewInMemory(), log, config.Config{})
	if err != nil {
		panic(err)
	}
	return server.Handler()
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
	handler := newTestHandler()
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
	handler := newTestHandler()
	res := request(t, handler, http.MethodGet, "/api/v1/projects", "", nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("protected route status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestCreateProjectCreatesDefaultEnvironments(t *testing.T) {
	handler := newTestHandler()
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

func TestListTemplatesIncludesSpringBootVariables(t *testing.T) {
	handler := newTestHandler()
	res := request(t, handler, http.MethodGet, "/api/v1/templates", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list templates status = %d body=%s", res.Code, res.Body.String())
	}
	templates := decodeBody[[]model.Template](t, res)
	for _, template := range templates {
		if template.ID == "spring-boot-base-template" {
			if template.Format != "yaml" || len(template.Variables) == 0 {
				t.Fatalf("unexpected spring template: %#v", template)
			}
			if !templateHasVariable(template, "DB_SECRET") {
				t.Fatalf("DB_SECRET variable missing: %#v", template.Variables)
			}
			return
		}
	}
	t.Fatalf("spring boot template missing: %#v", templates)
}

func TestCreateTemplateIsPrivateToActor(t *testing.T) {
	handler := newTestHandler()
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
	handler := newTestHandler()
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
	handler := newTestHandler()
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

func TestConfigHistorySnapshotsAndRollback(t *testing.T) {
	handler := newTestHandler()
	nextValue := "warn"
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level", "alice", map[string]any{
		"value":        nextValue,
		"changeReason": "snapshot update",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("update config status = %d body=%s", res.Code, res.Body.String())
	}

	res = request(t, handler, http.MethodGet, "/api/v1/projects/customer-portal/config-history?env=staging", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list config history status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Snapshots []model.ConfigSnapshot `json:"snapshots"`
	}](t, res)
	if len(payload.Snapshots) < 2 {
		t.Fatalf("snapshot count = %d", len(payload.Snapshots))
	}
	if !snapshotHasValue(payload.Snapshots[0], "log.level", nextValue) {
		t.Fatalf("latest snapshot did not include updated log.level: %#v", payload.Snapshots[0])
	}

	res = request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/config-history/rollback", "alice", map[string]any{
		"environment":  "staging",
		"snapshotId":   payload.Snapshots[1].ID,
		"changeReason": "snapshot rollback",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("rollback snapshot status = %d body=%s", res.Code, res.Body.String())
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
				t.Fatalf("rollback snapshot value = %q", entry.Value)
			}
		}
	}
	if !found {
		t.Fatalf("log.level config missing after rollback")
	}
}

func snapshotHasValue(snapshot model.ConfigSnapshot, key, value string) bool {
	for _, entry := range snapshot.Entries {
		if entry.Key == key && entry.Value == value {
			return true
		}
	}
	return false
}

func TestUpdateConfigKey(t *testing.T) {
	handler := newTestHandler()
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
	handler := newTestHandler()
	value := "https://api2.example.com"
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-prod-api-baseurl", "nora", map[string]any{"value": value})
	if res.Code != http.StatusForbidden {
		t.Fatalf("update prod as developer status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestImportJSONConfig(t *testing.T) {
	handler := newTestHandler()
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
	handler := newTestHandler()
	res := request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/configs/extract", "alice", map[string]any{
		"environment": "dev",
		"format":      "json",
		"content":     `{"feature":{"preview":true},"limit":3}`,
	})
	if res.Code != http.StatusOK {
		t.Fatalf("extract status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Entries    []model.ConfigSnapshotEntry `json:"entries"`
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
	handler := newTestHandler()
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

func TestValidationDetectsMissingRequiredKeys(t *testing.T) {
	handler := newTestHandler()
	res := request(t, handler, http.MethodPost, "/api/v1/projects/customer-portal/validate", "alice", map[string]any{"environment": "prod"})
	if res.Code != http.StatusOK {
		t.Fatalf("validate status = %d body=%s", res.Code, res.Body.String())
	}
	result := decodeBody[model.ValidationResult](t, res)
	if result.Valid {
		t.Fatalf("expected prod validation to fail because app.timezone and log.level are missing")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected validation errors")
	}
}
