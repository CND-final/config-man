package model

import (
	"regexp"
	"strings"
	"time"
)

type ConfigFile struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SourceType  string    `json:"sourceType"`
	SourceID    string    `json:"sourceId,omitempty"`
	Prefix      string    `json:"prefix"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func StandardConfigFiles(projectID string, now time.Time) []ConfigFile {
	return []ConfigFile{
		standardConfigFile(projectID, "application.yaml", "Application settings", "application", 1, now),
		standardConfigFile(projectID, "redis.yaml", "Redis settings", "redis", 2, now),
		standardConfigFile(projectID, "security.json", "Security settings", "security", 3, now),
	}
}

func StandardConfigFileForKey(projectID, key string, now time.Time) ConfigFile {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, file := range StandardConfigFiles(projectID, now) {
		switch file.Name {
		case "redis.yaml":
			if key == "redis" || strings.HasPrefix(key, "redis.") || strings.Contains(key, ".redis.") {
				return file
			}
		case "security.json":
			if keyHasSecurityPrefix(key) {
				return file
			}
		}
	}
	return StandardConfigFiles(projectID, now)[0]
}

func ConfigFileID(projectID, name string) string {
	return "cfgfile-" + slug(projectID) + "-" + slug(NormalizeConfigFileName(name))
}

func NormalizeConfigFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".properties") {
		return name
	}
	return name + ".yaml"
}

func ConfigFilePrefix(name string) string {
	name = NormalizeConfigFileName(name)
	lower := strings.ToLower(name)
	for _, suffix := range []string{".properties", ".yaml", ".yml", ".json"} {
		lower = strings.TrimSuffix(lower, suffix)
	}
	return strings.TrimSpace(lower)
}

func standardConfigFile(projectID, name, description, prefix string, sortOrder int, now time.Time) ConfigFile {
	return ConfigFile{
		ID:          ConfigFileID(projectID, name),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		SourceType:  "standard",
		Prefix:      prefix,
		SortOrder:   sortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func keyHasSecurityPrefix(key string) bool {
	prefixes := []string{"security", "auth", "jwt", "oauth", "cors", "tls", "ssl", "saml", "session"}
	for _, prefix := range prefixes {
		if key == prefix || strings.HasPrefix(key, prefix+".") || strings.Contains(key, "."+prefix+".") {
			return true
		}
	}
	return false
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "default"
	}
	return value
}
