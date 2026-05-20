package util

import (
	"testing"

	"config-man/backend/model"
)

func TestCanRegisterProject(t *testing.T) {
	cases := []struct {
		name string
		role model.UserRole
		want bool
	}{
		{name: "system-admin", role: model.RoleSystemAdmin, want: true},
		{name: "project-admin", role: model.RoleProjectAdmin, want: true},
		{name: "developer", role: model.RoleDeveloper, want: false},
		{name: "reviewer", role: model.RoleReviewer, want: false},
		{name: "viewer", role: model.RoleViewer, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Role: tc.role}
			if got := CanRegisterProject(user); got != tc.want {
				t.Errorf("CanRegisterProject(%s) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestCanWriteEnvironment(t *testing.T) {
	cases := []struct {
		name        string
		role        model.UserRole
		environment string
		want        bool
	}{
		{name: "admin-any-env", role: model.RoleSystemAdmin, environment: "prod", want: true},
		{name: "project-admin-any-env", role: model.RoleProjectAdmin, environment: "dev", want: true},
		{name: "developer-dev", role: model.RoleDeveloper, environment: "dev", want: true},
		{name: "developer-staging", role: model.RoleDeveloper, environment: "staging", want: true},
		{name: "developer-prod", role: model.RoleDeveloper, environment: "prod", want: false},
		{name: "developer-PROD-uppercase", role: model.RoleDeveloper, environment: "PROD", want: false},
		{name: "developer-Prod-mixed", role: model.RoleDeveloper, environment: "Prod", want: false},
		{name: "reviewer-any-env", role: model.RoleReviewer, environment: "prod", want: false},
		{name: "viewer-any-env", role: model.RoleViewer, environment: "dev", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Role: tc.role}
			if got := CanWriteEnvironment(user, tc.environment); got != tc.want {
				t.Errorf("CanWriteEnvironment(%s, %q) = %v, want %v", tc.role, tc.environment, got, tc.want)
			}
		})
	}
}

func TestCanRevealSensitive(t *testing.T) {
	cases := []struct {
		name string
		role model.UserRole
		want bool
	}{
		{name: "system-admin", role: model.RoleSystemAdmin, want: true},
		{name: "project-admin", role: model.RoleProjectAdmin, want: true},
		{name: "developer", role: model.RoleDeveloper, want: true},
		{name: "reviewer", role: model.RoleReviewer, want: false},
		{name: "viewer", role: model.RoleViewer, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Role: tc.role}
			if got := CanRevealSensitive(user); got != tc.want {
				t.Errorf("CanRevealSensitive(%s) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestCanCreateReview(t *testing.T) {
	cases := []struct {
		name string
		role model.UserRole
		want bool
	}{
		{name: "system-admin", role: model.RoleSystemAdmin, want: true},
		{name: "project-admin", role: model.RoleProjectAdmin, want: true},
		{name: "developer", role: model.RoleDeveloper, want: true},
		{name: "reviewer", role: model.RoleReviewer, want: true},
		{name: "viewer", role: model.RoleViewer, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Role: tc.role}
			if got := CanCreateReview(user); got != tc.want {
				t.Errorf("CanCreateReview(%s) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestCanReview(t *testing.T) {
	cases := []struct {
		name string
		role model.UserRole
		want bool
	}{
		{name: "system-admin", role: model.RoleSystemAdmin, want: true},
		{name: "project-admin", role: model.RoleProjectAdmin, want: false},
		{name: "developer", role: model.RoleDeveloper, want: false},
		{name: "reviewer", role: model.RoleReviewer, want: true},
		{name: "viewer", role: model.RoleViewer, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Role: tc.role}
			if got := CanReview(user); got != tc.want {
				t.Errorf("CanReview(%s) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestNormalizeValueType(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "string", want: "string"},
		{input: "number", want: "number"},
		{input: "boolean", want: "boolean"},
		{input: "bool", want: "boolean"},
		{input: "json", want: "json"},
		{input: "", want: "string"},
		{input: "  STRING  ", want: "string"},
		{input: "BOOLEAN", want: "boolean"},
		{input: "JSON", want: "json"},
		{input: "invalid", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := NormalizeValueType(tc.input); got != tc.want {
				t.Errorf("NormalizeValueType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsSupportedConfigFormat(t *testing.T) {
	cases := []struct {
		format string
		want   bool
	}{
		{format: "json", want: true},
		{format: "JSON", want: true},
		{format: "yaml", want: true},
		{format: "YAML", want: true},
		{format: "yml", want: false},
		{format: "properties", want: true},
		{format: "PROPERTIES", want: true},
		{format: "toml", want: false},
		{format: "xml", want: false},
		{format: "  json  ", want: true},
		{format: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			if got := IsSupportedConfigFormat(tc.format); got != tc.want {
				t.Errorf("IsSupportedConfigFormat(%q) = %v, want %v", tc.format, got, tc.want)
			}
		})
	}
}
