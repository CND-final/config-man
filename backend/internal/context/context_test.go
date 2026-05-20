package context

import (
	stdctx "context"
	"strings"
	"testing"

	"config-man/backend/pkg/config"
)

func TestNewConfigManContextRequiresDatabaseURL(t *testing.T) {
	_, err := NewConfigManContext(stdctx.Background(), config.Config{})
	if err == nil {
		t.Fatal("expected missing DATABASE_URL to fail context initialization")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %q", err.Error())
	}
}
