package util

import (
	"testing"
)

func TestLooksSensitivePassword(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{key: "password", want: true},
		{key: "PASSWORD", want: true},
		{key: "db_password", want: true},
		{key: "admin.password", want: true},
		{key: "user_password_hash", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := LooksSensitive(tc.key); got != tc.want {
				t.Errorf("LooksSensitive(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestLooksSensitiveSecret(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{key: "secret", want: true},
		{key: "SECRET", want: true},
		{key: "api_secret", want: true},
		{key: "jwt.secret", want: true},
		{key: "stripe_secret_key", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := LooksSensitive(tc.key); got != tc.want {
				t.Errorf("LooksSensitive(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestLooksSensitiveToken(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{key: "token", want: true},
		{key: "TOKEN", want: true},
		{key: "auth_token", want: true},
		{key: "refresh_token", want: true},
		{key: "github.token", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := LooksSensitive(tc.key); got != tc.want {
				t.Errorf("LooksSensitive(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestLooksSensitiveCredential(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{key: "credential", want: true},
		{key: "CREDENTIAL", want: true},
		{key: "aws_credentials", want: true},
		{key: "credentials.path", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := LooksSensitive(tc.key); got != tc.want {
				t.Errorf("LooksSensitive(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestLooksSensitiveDatabaseURL(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{key: "database.url", want: true},
		{key: "DATABASE.URL", want: true},
		{key: "db.url", want: true},
		{key: "DB.URL", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := LooksSensitive(tc.key); got != tc.want {
				t.Errorf("LooksSensitive(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestLooksSensitiveNonSensitive(t *testing.T) {
	cases := []struct {
		key string
	}{
		{key: "app.name"},
		{key: "app.version"},
		{key: "log.level"},
		{key: "max_connections"},
		{key: "timeout_ms"},
		{key: "feature_enabled"},
		{key: "user_email"},
		{key: "project_id"},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := LooksSensitive(tc.key); got {
				t.Errorf("LooksSensitive(%q) = true, want false", tc.key)
			}
		})
	}
}
