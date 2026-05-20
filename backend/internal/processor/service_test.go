package processor

import (
	"testing"

	"config-man/backend/internal/store"
	"config-man/backend/model"
)

func TestLoginSuccess(t *testing.T) {
	proc := NewInMemory()

	resp, err := proc.Login(model.LoginRequest{
		Email:    "admin@config-man.local",
		Password: "password",
	})

	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.User.Email != "admin@config-man.local" {
		t.Fatalf("user email = %q, want admin@config-man.local", resp.User.Email)
	}
	if resp.User.Role != model.RoleSystemAdmin {
		t.Fatalf("user role = %s, want system_admin", resp.User.Role)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	proc := NewInMemory()

	_, err := proc.Login(model.LoginRequest{
		Email:    "admin@config-man.local",
		Password: "wrongpassword",
	})

	if err == nil {
		t.Fatal("expected login to fail with invalid password")
	}
	if err.Kind != "unauthorized" {
		t.Fatalf("error code = %s, want unauthorized", err.Kind)
	}
}

func TestLoginInvalidEmail(t *testing.T) {
	proc := NewInMemory()

	_, err := proc.Login(model.LoginRequest{
		Email:    "nonexistent@config-man.local",
		Password: "password",
	})

	if err == nil {
		t.Fatal("expected login to fail with nonexistent email")
	}
	if err.Kind != "unauthorized" {
		t.Fatalf("error code = %s, want unauthorized", err.Kind)
	}
}

func TestLoginEmailTrimmed(t *testing.T) {
	proc := NewInMemory()

	resp, err := proc.Login(model.LoginRequest{
		Email:    "  admin@config-man.local  ",
		Password: "password",
	})

	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.User.Email != "admin@config-man.local" {
		t.Fatalf("email should be trimmed: %q", resp.User.Email)
	}
}

func TestAuthenticateTokenSuccess(t *testing.T) {
	proc := NewInMemory()

	ctx, err := proc.AuthenticateToken("alice")

	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}
	if ctx.Actor.ID != "alice" {
		t.Fatalf("actor id = %q, want alice", ctx.Actor.ID)
	}
	if ctx.Actor.Email != "admin@config-man.local" {
		t.Fatalf("actor email = %q, want admin@config-man.local", ctx.Actor.Email)
	}
}

func TestAuthenticateTokenInvalid(t *testing.T) {
	proc := NewInMemory()

	_, err := proc.AuthenticateToken("nonexistent")

	if err == nil {
		t.Fatal("expected authentication to fail with invalid token")
	}
	if err.Kind != "unauthorized" {
		t.Fatalf("error code = %s, want unauthorized", err.Kind)
	}
}

func TestAuthenticateTokenTrimmed(t *testing.T) {
	proc := NewInMemory()

	ctx, err := proc.AuthenticateToken("  alice  ")

	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}
	if ctx.Actor.ID != "alice" {
		t.Fatalf("token should be trimmed: %q", ctx.Actor.ID)
	}
}

func TestAuthenticateEmptyToken(t *testing.T) {
	proc := NewInMemory()

	_, err := proc.AuthenticateToken("")

	if err == nil {
		t.Fatal("expected authentication to fail with empty token")
	}
	if err.Kind != "unauthorized" {
		t.Fatalf("error code = %s, want unauthorized", err.Kind)
	}
}

func TestBaseTemplateReturnsValidTemplate(t *testing.T) {
	proc := NewInMemory()

	ctx, _ := proc.AuthenticateToken("alice")
	tmpl := proc.BaseTemplate(ctx)

	if tmpl.Name == "" {
		t.Fatal("template name should not be empty")
	}
	if len(tmpl.Entries) == 0 {
		t.Fatal("template should have entries")
	}
}

func TestBaseTemplateRequiredEntries(t *testing.T) {
	proc := NewInMemory()

	ctx, _ := proc.AuthenticateToken("alice")
	tmpl := proc.BaseTemplate(ctx)

	requiredKeys := map[string]bool{}
	for _, entry := range tmpl.Entries {
		if entry.Required {
			requiredKeys[entry.Key] = true
		}
	}

	expectedRequired := []string{"app.timezone", "log.level", "api.baseUrl", "database.url"}
	for _, key := range expectedRequired {
		if !requiredKeys[key] {
			t.Fatalf("expected %q to be required", key)
		}
	}
}

func TestNewProcessorNilStore(t *testing.T) {
	_, err := NewProcessor(nil)

	if err == nil {
		t.Fatal("expected error with nil store")
	}
}

func TestNewProcessorValidStore(t *testing.T) {
	s := store.NewStore()
	proc, err := NewProcessor(s)

	if err != nil {
		t.Fatalf("processor creation failed: %v", err)
	}
	if proc == nil {
		t.Fatal("processor should not be nil")
	}
}

func TestNewInMemoryCreatesStore(t *testing.T) {
	proc := NewInMemory()

	if proc == nil {
		t.Fatal("in-memory processor should not be nil")
	}

	resp, err := proc.Login(model.LoginRequest{
		Email:    "admin@config-man.local",
		Password: "password",
	})

	if err != nil {
		t.Fatalf("login should work with in-memory processor: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("token should be set")
	}
}
