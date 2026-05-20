package util

import (
	"encoding/json"
	"testing"
)

func entriesByKey(entries []ParsedConfigEntry) map[string]ParsedConfigEntry {
	out := make(map[string]ParsedConfigEntry, len(entries))
	for _, entry := range entries {
		out[entry.Key] = entry
	}
	return out
}

func TestParseConfigFileJSON(t *testing.T) {
	content := `{
  "feature": {"checkout": true, "limit": 3},
  "names": ["alice", "bob"],
  "metadata": {"owner": "team", "flags": {"beta": false}},
  "pi": 3.14,
  "nullValue": null
}`

	entries, err := ParseConfigFile("json", content)
	if err != nil {
		t.Fatalf("parse json config: %v", err)
	}

	got := entriesByKey(entries)
	want := map[string]ParsedConfigEntry{
		"feature.checkout":    {Key: "feature.checkout", Value: "true", ValueType: "boolean"},
		"feature.limit":       {Key: "feature.limit", Value: "3", ValueType: "number"},
		"names":               {Key: "names", Value: `["alice","bob"]`, ValueType: "json"},
		"metadata.owner":      {Key: "metadata.owner", Value: "team", ValueType: "string"},
		"metadata.flags.beta": {Key: "metadata.flags.beta", Value: "false", ValueType: "boolean"},
		"pi":                  {Key: "pi", Value: "3.14", ValueType: "number"},
		"nullValue":           {Key: "nullValue", Value: "", ValueType: "string"},
	}

	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d", len(got), len(want))
	}
	for key, expected := range want {
		entry, ok := got[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		if entry.Value != expected.Value {
			t.Fatalf("value for %q = %q, want %q", key, entry.Value, expected.Value)
		}
		if entry.ValueType != expected.ValueType {
			t.Fatalf("value type for %q = %q, want %q", key, entry.ValueType, expected.ValueType)
		}
	}
}

func TestParseConfigFileJSONInvalid(t *testing.T) {
	if _, err := ParseConfigFile("json", "{"); err == nil {
		t.Fatal("expected invalid json to return error")
	}
}

func TestParseConfigFileJSONRootArray(t *testing.T) {
	entries, err := ParseConfigFile("json", `[1,2,3]`)
	if err != nil {
		t.Fatalf("parse json array: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entry count = %d, want 0", len(entries))
	}
}

func TestParseConfigFileProperties(t *testing.T) {
	content := `
# comment
! ignore
foo=bar
baz: 123
spaced = true
=invalid
noSeparator
`
	entries, err := ParseConfigFile("properties", content)
	if err != nil {
		t.Fatalf("parse properties config: %v", err)
	}

	got := entriesByKey(entries)
	want := map[string]ParsedConfigEntry{
		"foo":    {Key: "foo", Value: "bar", ValueType: "string"},
		"baz":    {Key: "baz", Value: "123", ValueType: "number"},
		"spaced": {Key: "spaced", Value: "true", ValueType: "boolean"},
	}

	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d", len(got), len(want))
	}
	for key, expected := range want {
		entry, ok := got[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		if entry.Value != expected.Value {
			t.Fatalf("value for %q = %q, want %q", key, entry.Value, expected.Value)
		}
		if entry.ValueType != expected.ValueType {
			t.Fatalf("value type for %q = %q, want %q", key, entry.ValueType, expected.ValueType)
		}
	}
}

func TestParseConfigFilePropertiesWindowsNewlines(t *testing.T) {
	entries, err := ParseConfigFile("properties", "a=1\r\nb=true\r\n")
	if err != nil {
		t.Fatalf("parse properties config: %v", err)
	}

	got := entriesByKey(entries)
	if len(got) != 2 {
		t.Fatalf("entry count = %d, want 2", len(got))
	}
	if got["a"].Value != "1" || got["a"].ValueType != "number" {
		t.Fatalf("a entry = %#v", got["a"])
	}
	if got["b"].Value != "true" || got["b"].ValueType != "boolean" {
		t.Fatalf("b entry = %#v", got["b"])
	}
}

func TestParseConfigFileYAML(t *testing.T) {
	content := `
# comment
app:
  name: "Config Man"
  enabled: true
  retries: 2
  nested:
    key: 'value'
  empty:
root: 3
`
	entries, err := ParseConfigFile("yaml", content)
	if err != nil {
		t.Fatalf("parse yaml config: %v", err)
	}

	got := entriesByKey(entries)
	want := map[string]ParsedConfigEntry{
		"app.name":       {Key: "app.name", Value: "Config Man", ValueType: "string"},
		"app.enabled":    {Key: "app.enabled", Value: "true", ValueType: "boolean"},
		"app.retries":    {Key: "app.retries", Value: "2", ValueType: "number"},
		"app.nested.key": {Key: "app.nested.key", Value: "value", ValueType: "string"},
		"root":           {Key: "root", Value: "3", ValueType: "number"},
	}

	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d", len(got), len(want))
	}
	for key, expected := range want {
		entry, ok := got[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		if entry.Value != expected.Value {
			t.Fatalf("value for %q = %q, want %q", key, entry.Value, expected.Value)
		}
		if entry.ValueType != expected.ValueType {
			t.Fatalf("value type for %q = %q, want %q", key, entry.ValueType, expected.ValueType)
		}
	}

	if _, ok := got["app.empty"]; ok {
		t.Fatal("unexpected entry for empty yaml key")
	}
}

func TestParseConfigFileYAMLIndentation(t *testing.T) {
	content := `
root:
  first: 1
  nested:
    child: yes
sibling: 2
invalid line
`
	entries, err := ParseConfigFile("yaml", content)
	if err != nil {
		t.Fatalf("parse yaml config: %v", err)
	}

	got := entriesByKey(entries)
	want := map[string]ParsedConfigEntry{
		"root.first":        {Key: "root.first", Value: "1", ValueType: "number"},
		"root.nested.child": {Key: "root.nested.child", Value: "yes", ValueType: "string"},
		"sibling":           {Key: "sibling", Value: "2", ValueType: "number"},
	}
	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d", len(got), len(want))
	}
	for key, expected := range want {
		entry, ok := got[key]
		if !ok {
			t.Fatalf("missing key %q", key)
		}
		if entry.Value != expected.Value {
			t.Fatalf("value for %q = %q, want %q", key, entry.Value, expected.Value)
		}
		if entry.ValueType != expected.ValueType {
			t.Fatalf("value type for %q = %q, want %q", key, entry.ValueType, expected.ValueType)
		}
	}
	if _, ok := got["root.nested"]; ok {
		t.Fatal("unexpected entry for empty yaml object key")
	}
}

func TestInferValueType(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "bool", value: true, want: "boolean"},
		{name: "json-number", value: json.Number("12.5"), want: "number"},
		{name: "map", value: map[string]any{"a": 1}, want: "json"},
		{name: "string-bool", value: "true", want: "boolean"},
		{name: "string-number", value: " 42 ", want: "number"},
		{name: "string-json-object", value: `{"a":1}`, want: "json"},
		{name: "string-json-array", value: `[1,2]`, want: "json"},
		{name: "string-empty", value: "", want: "string"},
		{name: "string-default", value: "hello", want: "string"},
		{name: "json-number-invalid", value: json.Number("1e"), want: "string"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferValueType(tc.value); got != tc.want {
				t.Fatalf("inferValueType(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseConfigFileFormatNormalization(t *testing.T) {
	entries, err := ParseConfigFile(" JSON ", `{"a":1}`)
	if err != nil {
		t.Fatalf("parse json config: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	if entries[0].Key != "a" || entries[0].Value != "1" || entries[0].ValueType != "number" {
		t.Fatalf("entry = %#v", entries[0])
	}
}

func TestParseConfigFileUnsupportedFormat(t *testing.T) {
	if _, err := ParseConfigFile("toml", ""); err == nil {
		t.Fatal("expected unsupported format to return error")
	}
}
