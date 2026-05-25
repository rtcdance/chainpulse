package api

import (
	"fmt"
	"math"
	"testing"
)

func TestLooksLikeHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"valid_hash", "0x" + "0000000000000000000000000000000000000000000000000000000000000001", true},
		{"too_short", "0x1234", false},
		{"no_0x_prefix", "123456789012345678901234567890123456789012345678901234567890123456", false},
		{"empty", "", false},
		{"exact_length_no_prefix", "123456789012345678901234567890123456789012345678901234567890123456", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := looksLikeHash(tc.value)
			if got != tc.want {
				t.Errorf("looksLikeHash(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestSafeUint64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value uint64
		want  int64
	}{
		{"small", 42, 42},
		{"zero", 0, 0},
		{"max_int64", math.MaxInt64, math.MaxInt64},
		{"overflow", math.MaxInt64 + 1, math.MaxInt64},
		{"max_uint64", math.MaxUint64, math.MaxInt64},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := safeUint64ToInt64(tc.value)
			if got != tc.want {
				t.Errorf("safeUint64ToInt64(%d) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseLastEventID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		want    uint64
		wantErr bool
	}{
		{"valid", "42", 42, false},
		{"zero", "0", 0, false},
		{"large", "18446744073709551615", 18446744073709551615, false},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseLastEventID(tc.id)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseLastEventID(%q) error = %v, wantErr %v", tc.id, err, tc.wantErr)
			}
			if err == nil && got != tc.want {
				t.Errorf("parseLastEventID(%q) = %d, want %d", tc.id, got, tc.want)
			}
		})
	}
}

func TestValidateQueryDepth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		maxDepth int
		wantErr  bool
	}{
		{"simple", "{ events { id } }", 5, false},
		{"deep", "{{{{{{ events }}}}}}", 5, true},
		{"exact_limit", "{{{ events }}}", 3, false},
		{"just_over", "{{{{ events }}}}", 3, true},
		{"empty", "", 5, false},
		{"unbalanced_close", "}}}", 5, false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateQueryDepth(tc.query, tc.maxDepth)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateQueryDepth(%q, %d) error = %v, wantErr %v", tc.query, tc.maxDepth, err, tc.wantErr)
			}
		})
	}
}

func TestValidateQueryDepth_EarlyExit(t *testing.T) {
	t.Parallel()

	err := validateQueryDepth("{{{{{{", 5)
	if err == nil {
		t.Error("expected error for depth exceeding limit early")
	}
	if err.Error() != fmt.Sprintf("query exceeds maximum depth of %d", 5) {
		t.Errorf("error message = %q", err.Error())
	}
}
