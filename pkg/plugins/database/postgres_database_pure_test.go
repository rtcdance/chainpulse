package database

import "testing"

func TestOrDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s, def string
		want   string
	}{
		{"returns_s_when_non_empty", "hello", "default", "hello"},
		{"returns_def_when_empty", "", "default", "default"},
		{"returns_def_when_empty_and_same", "", "", ""},
		{"returns_s_with_numbers", "123", "0", "123"},
		{"returns_s_with_special_chars", "!@#", "default", "!@#"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := orDefault(tt.s, tt.def)
			if got != tt.want {
				t.Errorf("orDefault(%q, %q) = %q, want %q", tt.s, tt.def, got, tt.want)
			}
		})
	}
}
