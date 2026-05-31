package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func NormalizeEnvironments(environments []string) []string {
	if len(environments) == 0 {
		return []string{"dev", "staging", "prod"}
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(environments))
	for _, env := range environments {
		name := strings.ToLower(strings.TrimSpace(env))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return []string{"dev", "staging", "prod"}
	}
	return normalized
}

func NormalizeBranches(branches []string) []string {
	if len(branches) == 0 {
		return []string{"default"}
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(branches))
	for _, branch := range branches {
		name := strings.ToLower(strings.TrimSpace(branch))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return []string{"default"}
	}
	return normalized
}

func NormalizeBranch(branch string) string {
	branch = strings.ToLower(strings.TrimSpace(branch))
	if branch == "" {
		return "default"
	}
	return branch
}

func NewID(prefix string) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf)
}

func Fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func Contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
