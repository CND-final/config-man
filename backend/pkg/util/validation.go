package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"config-man/backend/model"
)

func ValidateEntries(projectID string, environments []string, entries []model.ValidationEntry) model.ValidationResult {
	errors := make([]model.ValidationIssue, 0)
	warnings := make([]model.ValidationIssue, 0)

	requiredEntries := make([]model.TemplateEntry, 0)
	for _, entry := range model.BaseTemplate().Entries {
		if entry.Required {
			requiredEntries = append(requiredEntries, entry)
		}
	}

	for _, environment := range environments {
		envEntries := make([]model.ValidationEntry, 0)
		for _, entry := range entries {
			if entry.Environment == environment {
				envEntries = append(envEntries, entry)
			}
		}

		keys := map[string]bool{}
		counts := map[string]int{}
		for _, entry := range envEntries {
			keys[entry.Key] = true
			counts[entry.Key]++
		}

		for _, templateEntry := range requiredEntries {
			if !keys[templateEntry.Key] {
				errors = append(errors, model.ValidationIssue{
					Environment: environment,
					Key:         templateEntry.Key,
					Code:        "missing_required_key",
					Message:     fmt.Sprintf("Missing required config key %q", templateEntry.Key),
				})
			}
		}

		for key, count := range counts {
			if count > 1 {
				errors = append(errors, model.ValidationIssue{
					Environment: environment,
					Key:         key,
					Code:        "duplicate_key",
					Message:     fmt.Sprintf("Duplicate config key %q in %q", key, environment),
				})
			}
		}

		for _, entry := range envEntries {
			if !valueMatchesType(entry.Value, entry.ValueType) {
				errors = append(errors, model.ValidationIssue{
					Environment: environment,
					Key:         entry.Key,
					Code:        "invalid_value_type",
					Message:     fmt.Sprintf("Value does not match type %q", entry.ValueType),
				})
			}
			if LooksSensitive(entry.Key) && !entry.IsSensitive {
				warnings = append(warnings, model.ValidationIssue{
					Environment: environment,
					Key:         entry.Key,
					Code:        "sensitive_key_not_marked",
					Message:     fmt.Sprintf("Key %q looks sensitive but is not marked sensitive", entry.Key),
				})
			}
		}
	}

	return model.ValidationResult{
		ProjectID:    projectID,
		Environments: environments,
		Valid:        len(errors) == 0,
		Errors:       errors,
		Warnings:     warnings,
	}
}

func valueMatchesType(value, valueType string) bool {
	switch NormalizeValueType(valueType) {
	case "number":
		_, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return strings.TrimSpace(value) != "" && err == nil
	case "boolean":
		return strings.EqualFold(value, "true") || strings.EqualFold(value, "false")
	case "json":
		var parsed any
		return json.Unmarshal([]byte(value), &parsed) == nil
	case "string":
		return true
	default:
		return false
	}
}
