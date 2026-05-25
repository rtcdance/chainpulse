package config

import (
	"testing"
)

func TestIsSensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		sensitive bool
	}{
		{"password", "db_password", true},
		{"token", "api_token", true},
		{"secret", "client_secret", true},
		{"api_key", "my_api_key", true},
		{"private", "private_key", true},
		{"credential", "credential_path", true},
		{"normal_key", "chain_id", false},
		{"empty", "", false},
		{"lowercase_password", "password", true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isSensitive(tc.key)
			if got != tc.sensitive {
				t.Errorf("isSensitive(%q) = %v, want %v", tc.key, got, tc.sensitive)
			}
		})
	}
}

func TestContainsSubstring(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       string
		substr  string
		contain bool
	}{
		{"contains", "hello world", "world", true},
		{"not_contains", "hello world", "foo", false},
		{"exact_match", "hello", "hello", true},
		{"empty_substr", "hello", "", true},
		{"empty_string", "", "hello", false},
		{"short_string", "hi", "world", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := containsSubstring(tc.s, tc.substr)
			if got != tc.contain {
				t.Errorf("containsSubstring(%q, %q) = %v, want %v", tc.s, tc.substr, got, tc.contain)
			}
		})
	}
}
