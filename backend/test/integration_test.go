package test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	app "config-man/backend/internal"
	"config-man/backend/internal/processor"
	"config-man/backend/internal/store"
	"config-man/backend/model"
	"config-man/backend/pkg/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const dbTimeout = 5 * time.Second

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration tests")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("ping database: %v", err)
	}
	return db
}

func prepareDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := store.NewStoreWithDB(context.Background(), db); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	resetDatabase(t, db)
}

func resetDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	if _, err := db.ExecContext(ctx, `
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
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func newIntegrationHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()

	dataStore, err := store.NewStoreWithDB(context.Background(), db)
	if err != nil {
		t.Fatalf("initialize store: %v", err)
	}
	proc, err := processor.NewProcessor(dataStore)
	if err != nil {
		t.Fatalf("initialize processor: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server, err := app.NewServer(proc, log, config.Config{Host: "127.0.0.1", Port: "0"})
	if err != nil {
		t.Fatalf("initialize server: %v", err)
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

func TestIntegrationProjectPersistsAfterReload(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)
	res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
		"name":    "integration-billing",
		"groupId": "platform-team",
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body=%s", res.Code, res.Body.String())
	}
	created := decodeBody[model.Project](t, res)
	if created.ID == "" {
		t.Fatalf("project id missing: %#v", created)
	}

	reloaded := newIntegrationHandler(t, db)
	res = request(t, reloaded, http.MethodGet, "/api/v1/projects/"+created.ID, "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("get project status = %d body=%s", res.Code, res.Body.String())
	}
	loaded := decodeBody[model.Project](t, res)
	if loaded.Name != created.Name {
		t.Fatalf("loaded project name = %q want %q", loaded.Name, created.Name)
	}
}

func TestIntegrationCustomTemplatePersistsAfterReload(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)
	res := request(t, handler, http.MethodPost, "/api/v1/templates", "alice", map[string]any{
		"name":   "integration-template",
		"format": "yaml",
		"body":   "service:\n  port: ${APP_PORT}\n",
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("create template status = %d body=%s", res.Code, res.Body.String())
	}
	created := decodeBody[model.Template](t, res)

	reloaded := newIntegrationHandler(t, db)
	res = request(t, reloaded, http.MethodGet, "/api/v1/templates", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list templates status = %d body=%s", res.Code, res.Body.String())
	}
	templates := decodeBody[[]model.Template](t, res)
	if !templateListHasID(templates, created.ID) {
		t.Fatalf("custom template missing after reload: %#v", templates)
	}
}

func TestIntegrationConfigVersionPersistsAfterReload(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)
	nextValue := "warn"
	res := request(t, handler, http.MethodPut, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level", "alice", map[string]any{
		"value":        nextValue,
		"changeReason": "integration update",
	})
	if res.Code != http.StatusOK {
		t.Fatalf("update config status = %d body=%s", res.Code, res.Body.String())
	}

	reloaded := newIntegrationHandler(t, db)
	res = request(t, reloaded, http.MethodGet, "/api/v1/projects/customer-portal/configs/cfg-staging-log-level/versions", "alice", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list versions status = %d body=%s", res.Code, res.Body.String())
	}
	payload := decodeBody[struct {
		Versions []model.ConfigVersion `json:"versions"`
	}](t, res)
	if !versionsContainValue(payload.Versions, nextValue) {
		t.Fatalf("expected version with new value %q, got %#v", nextValue, payload.Versions)
	}
}

func templateListHasID(templates []model.Template, id string) bool {
	for _, template := range templates {
		if template.ID == id {
			return true
		}
	}
	return false
}

func versionsContainValue(versions []model.ConfigVersion, value string) bool {
	for _, version := range versions {
		if version.NewValue == value {
			return true
		}
	}
	return false
}
