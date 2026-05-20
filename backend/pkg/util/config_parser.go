package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ParsedConfigEntry struct {
	Key       string
	Value     string
	ValueType string
}

func ParseConfigFile(format, content string) ([]ParsedConfigEntry, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return parseJSONConfig(content)
	case "properties":
		return parsePropertiesConfig(content), nil
	case "yaml":
		return parseSimpleYAMLConfig(content), nil
	default:
		return nil, fmt.Errorf("format must be json, yaml, or properties")
	}
}

func parseJSONConfig(content string) ([]ParsedConfigEntry, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(content))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	flattened := flattenObject(value, "")
	entries := make([]ParsedConfigEntry, 0, len(flattened))
	for _, item := range flattened {
		entries = append(entries, ParsedConfigEntry{
			Key:       item.key,
			Value:     stringifyValue(item.value),
			ValueType: inferValueType(item.value),
		})
	}
	return entries, nil
}

func parsePropertiesConfig(content string) []ParsedConfigEntry {
	entries := make([]ParsedConfigEntry, 0)
	for _, rawLine := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		separator := strings.Index(line, "=")
		if separator < 0 {
			separator = strings.Index(line, ":")
		}
		if separator < 0 {
			continue
		}
		key := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		if key == "" {
			continue
		}
		entries = append(entries, ParsedConfigEntry{
			Key:       key,
			Value:     value,
			ValueType: inferValueType(value),
		})
	}
	return entries
}

func parseSimpleYAMLConfig(content string) []ParsedConfigEntry {
	type stackItem struct {
		indent int
		path   []string
	}

	entries := make([]ParsedConfigEntry, 0)
	stack := []stackItem{{indent: -1, path: nil}}

	for _, rawLine := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(rawLine) == "" || strings.HasPrefix(strings.TrimSpace(rawLine), "#") {
			continue
		}

		indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))
		line := strings.TrimSpace(rawLine)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		rawValue := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}

		for len(stack) > 1 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}

		parentPath := append([]string(nil), stack[len(stack)-1].path...)
		path := append(parentPath, key)
		if rawValue == "" {
			stack = append(stack, stackItem{indent: indent, path: path})
			continue
		}

		value := strings.Trim(rawValue, `'"`)
		entries = append(entries, ParsedConfigEntry{
			Key:       strings.Join(path, "."),
			Value:     value,
			ValueType: inferValueType(value),
		})
	}

	return entries
}

type flattenedValue struct {
	key   string
	value any
}

func flattenObject(value any, prefix string) []flattenedValue {
	switch typed := value.(type) {
	case map[string]any:
		out := make([]flattenedValue, 0)
		for key, nested := range typed {
			nextPrefix := key
			if prefix != "" {
				nextPrefix = prefix + "." + key
			}
			out = append(out, flattenObject(nested, nextPrefix)...)
		}
		return out
	default:
		if prefix == "" {
			return nil
		}
		return []flattenedValue{{key: prefix, value: value}}
	}
}

func stringifyValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case map[string]any, []any:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(bytes)
	default:
		return fmt.Sprint(typed)
	}
}

func inferValueType(value any) string {
	switch typed := value.(type) {
	case bool:
		return "boolean"
	case int, int64, float64:
		return "number"
	case json.Number:
		if _, err := typed.Float64(); err == nil {
			return "number"
		}
	case map[string]any, []any:
		return "json"
	}

	stringValue := strings.TrimSpace(stringifyValue(value))
	if strings.EqualFold(stringValue, "true") || strings.EqualFold(stringValue, "false") {
		return "boolean"
	}
	if stringValue != "" {
		if _, err := strconv.ParseFloat(stringValue, 64); err == nil {
			return "number"
		}
	}
	if (strings.HasPrefix(stringValue, "{") && strings.HasSuffix(stringValue, "}")) ||
		(strings.HasPrefix(stringValue, "[") && strings.HasSuffix(stringValue, "]")) {
		return "json"
	}
	return "string"
}
