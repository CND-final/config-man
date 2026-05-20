package util

import (
	"testing"

	"config-man/backend/model"
)

func TestValidateEntriesRequiredKeys(t *testing.T) {
	result := ValidateEntries("project-1", []string{"prod"}, []model.ValidationEntry{})

	if result.Valid {
		t.Fatal("expected validation to fail with missing required keys")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected validation errors")
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "missing_required_key" && issue.Environment == "prod" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing_required_key error, got %v", result.Errors)
	}
}

func TestValidateEntriesDuplicateKeys(t *testing.T) {
	entries := []model.ValidationEntry{
		{Key: "app.name", Value: "app1", ValueType: "string", Environment: "dev"},
		{Key: "app.name", Value: "app2", ValueType: "string", Environment: "dev"},
	}

	result := ValidateEntries("project-1", []string{"dev"}, entries)

	if result.Valid {
		t.Fatal("expected validation to fail with duplicate keys")
	}

	found := false
	for _, issue := range result.Errors {
		if issue.Code == "duplicate_key" && issue.Key == "app.name" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate_key error for app.name")
	}
}

func TestValidateEntriesInvalidValueType(t *testing.T) {
	entries := []model.ValidationEntry{
		{Key: "app.name", Value: "valid-string", ValueType: "string", Environment: "dev"},
		{Key: "timeout", Value: "not-a-number", ValueType: "number", Environment: "dev"},
		{Key: "enabled", Value: "maybe", ValueType: "boolean", Environment: "dev"},
	}

	result := ValidateEntries("project-1", []string{"dev"}, entries)

	if result.Valid {
		t.Fatal("expected validation to fail with invalid types")
	}

	errorCount := 0
	for _, issue := range result.Errors {
		if issue.Code == "invalid_value_type" {
			errorCount++
		}
	}
	if errorCount != 2 {
		t.Fatalf("expected 2 invalid_value_type errors, got %d", errorCount)
	}
}

func TestValidateEntriesSensitiveKeyWarning(t *testing.T) {
	entries := []model.ValidationEntry{
		{Key: "database_password", Value: "secret123", ValueType: "string", Environment: "prod", IsSensitive: false},
		{Key: "api_token", Value: "token123", ValueType: "string", Environment: "prod", IsSensitive: false},
		{Key: "app.name", Value: "myapp", ValueType: "string", Environment: "prod", IsSensitive: false},
	}

	result := ValidateEntries("project-1", []string{"prod"}, entries)

	warningCount := 0
	for _, issue := range result.Warnings {
		if issue.Code == "sensitive_key_not_marked" {
			warningCount++
		}
	}
	if warningCount != 2 {
		t.Fatalf("expected 2 sensitive_key_not_marked warnings, got %d", warningCount)
	}
}

func TestValidateEntriesMultipleEnvironments(t *testing.T) {
	entries := []model.ValidationEntry{
		{Key: "app.name", Value: "app-dev", ValueType: "string", Environment: "dev"},
		{Key: "app.name", Value: "app-prod", ValueType: "string", Environment: "prod"},
	}

	result := ValidateEntries("project-1", []string{"dev", "prod"}, entries)

	if len(result.Environments) != 2 {
		t.Fatalf("environment count = %d, want 2", len(result.Environments))
	}
	if result.ProjectID != "project-1" {
		t.Fatalf("project ID = %q, want project-1", result.ProjectID)
	}
}

func TestValidateEntriesValidConfig(t *testing.T) {
	entries := []model.ValidationEntry{
		{Key: "app.timezone", Value: "UTC", ValueType: "string", Environment: "prod"},
		{Key: "log.level", Value: "info", ValueType: "string", Environment: "prod"},
		{Key: "api.baseUrl", Value: "https://api.example.com", ValueType: "string", Environment: "prod"},
		{Key: "database.url", Value: "postgres://localhost:5432/db", ValueType: "string", Environment: "prod", IsSensitive: true},
	}

	result := ValidateEntries("project-1", []string{"prod"}, entries)

	if !result.Valid {
		t.Fatalf("expected valid config, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(result.Errors))
	}
}

func TestValueMatchesTypeNumber(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{value: "42", want: true},
		{value: "3.14", want: true},
		{value: "-100", want: true},
		{value: "0", want: true},
		{value: "", want: false},
		{value: " ", want: false},
		{value: "abc", want: false},
		{value: "  123  ", want: true},
	}

	for _, tc := range cases {
		if got := valueMatchesType(tc.value, "number"); got != tc.want {
			t.Errorf("valueMatchesType(%q, number) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestValueMatchesTypeBoolean(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{value: "true", want: true},
		{value: "True", want: true},
		{value: "TRUE", want: true},
		{value: "false", want: true},
		{value: "False", want: true},
		{value: "FALSE", want: true},
		{value: "yes", want: false},
		{value: "1", want: false},
		{value: "", want: false},
	}

	for _, tc := range cases {
		if got := valueMatchesType(tc.value, "boolean"); got != tc.want {
			t.Errorf("valueMatchesType(%q, boolean) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestValueMatchesTypeJSON(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{value: `{"key":"value"}`, want: true},
		{value: `[1,2,3]`, want: true},
		{value: `{"nested":{"child":true}}`, want: true},
		{value: `[]`, want: true},
		{value: `{}`, want: true},
		{value: `{invalid}`, want: false},
		{value: `[1,2,`, want: false},
		{value: `plain string`, want: false},
	}

	for _, tc := range cases {
		if got := valueMatchesType(tc.value, "json"); got != tc.want {
			t.Errorf("valueMatchesType(%q, json) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestValueMatchesTypeString(t *testing.T) {
	cases := []struct {
		value string
	}{
		{value: "anything"},
		{value: ""},
		{value: " spaces "},
		{value: "123"},
		{value: "true"},
	}

	for _, tc := range cases {
		if got := valueMatchesType(tc.value, "string"); !got {
			t.Errorf("valueMatchesType(%q, string) = false, want true", tc.value)
		}
	}
}

func TestValueMatchesTypeUnknown(t *testing.T) {
	if got := valueMatchesType("value", "unknown"); got {
		t.Errorf("valueMatchesType(value, unknown) = true, want false")
	}
}
