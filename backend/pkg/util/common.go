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
