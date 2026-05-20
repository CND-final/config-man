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
