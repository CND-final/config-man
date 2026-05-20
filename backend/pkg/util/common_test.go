package util

import (
	"testing"
)

func TestNormalizeEnvironments(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "empty-returns-default",
			input: []string{},
			want:  []string{"dev", "staging", "prod"},
		},
		{
			name:  "lowercase-normalizes",
			input: []string{"Dev", "STAGING", "Prod"},
			want:  []string{"dev", "staging", "prod"},
		},
		{
			name:  "removes-whitespace",
			input: []string{"  dev  ", "staging", "  prod  "},
			want:  []string{"dev", "staging", "prod"},
		},
		{
			name:  "filters-empty-strings",
			input: []string{"dev", "", "staging", "  ", "prod"},
			want:  []string{"dev", "staging", "prod"},
		},
		{
			name:  "removes-duplicates",
			input: []string{"dev", "dev", "prod", "dev"},
			want:  []string{"dev", "prod"},
		},
		{
			name:  "duplicates-different-case",
			input: []string{"dev", "Dev", "DEV"},
			want:  []string{"dev"},
		},
		{
			name:  "single-environment",
			input: []string{"production"},
			want:  []string{"production"},
		},
		{
			name:  "all-whitespace-returns-default",
			input: []string{"  ", "", "   "},
			want:  []string{"dev", "staging", "prod"},
		},
		{
			name:  "custom-environments-preserved",
			input: []string{"local", "test", "canary", "stable"},
			want:  []string{"local", "test", "canary", "stable"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeEnvironments(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("length = %d, want %d", len(got), len(tc.want))
			}
			for i, v := range got {
				if v != tc.want[i] {
					t.Errorf("index %d = %q, want %q", i, v, tc.want[i])
				}
			}
		})
	}
}

func TestNewID(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
	}{
		{name: "proj", prefix: "proj"},
		{name: "cfg", prefix: "cfg"},
		{name: "rev", prefix: "rev"},
		{name: "empty", prefix: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id1 := NewID(tc.prefix)
			id2 := NewID(tc.prefix)

			if id1 == "" {
				t.Fatal("id should not be empty")
			}

			if id1 == id2 {
				t.Errorf("ids should be unique: %q, %q", id1, id2)
			}

			if tc.prefix != "" && id1[:len(tc.prefix)+1] != tc.prefix+"-" {
				t.Errorf("id %q should start with %q-", id1, tc.prefix)
			}
		})
	}
}

func TestNewIDFormat(t *testing.T) {
	id := NewID("test")

	if len(id) < len("test-") {
		t.Fatalf("id length = %d, should be at least %d", len(id), len("test-"))
	}

	if id[:5] != "test-" {
		t.Fatalf("id should start with 'test-', got %q", id[:5])
	}
}

func TestFallback(t *testing.T) {
	cases := []struct {
		name         string
		value        string
		defaultValue string
		want         string
	}{
		{name: "uses-value", value: "hello", defaultValue: "default", want: "hello"},
		{name: "uses-default-empty", value: "", defaultValue: "default", want: "default"},
		{name: "uses-default-whitespace", value: "   ", defaultValue: "default", want: "default"},
		{name: "uses-default-tabs", value: "\t\t", defaultValue: "default", want: "default"},
		{name: "preserves-leading-whitespace", value: "  value", defaultValue: "default", want: "  value"},
		{name: "empty-default", value: "", defaultValue: "", want: ""},
		{name: "zero-value", value: "0", defaultValue: "default", want: "0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Fallback(tc.value, tc.defaultValue)
			if got != tc.want {
				t.Errorf("Fallback(%q, %q) = %q, want %q", tc.value, tc.defaultValue, got, tc.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	cases := []struct {
		name   string
		values []string
		target string
		want   bool
	}{
		{name: "found", values: []string{"a", "b", "c"}, target: "b", want: true},
		{name: "not-found", values: []string{"a", "b", "c"}, target: "d", want: false},
		{name: "empty-slice", values: []string{}, target: "a", want: false},
		{name: "first-element", values: []string{"a", "b", "c"}, target: "a", want: true},
		{name: "last-element", values: []string{"a", "b", "c"}, target: "c", want: true},
		{name: "case-sensitive", values: []string{"Hello", "World"}, target: "hello", want: false},
		{name: "exact-match", values: []string{"hello", "world"}, target: "hello", want: true},
		{name: "empty-string-match", values: []string{"", "a", "b"}, target: "", want: true},
		{name: "whitespace-no-trim", values: []string{" a ", "b"}, target: "a", want: false},
		{name: "whitespace-exact", values: []string{" a ", "b"}, target: " a ", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Contains(tc.values, tc.target)
			if got != tc.want {
				t.Errorf("Contains(%v, %q) = %v, want %v", tc.values, tc.target, got, tc.want)
			}
		})
	}
}
