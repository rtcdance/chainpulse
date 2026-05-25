package database

import (
	"math"
	"testing"
)

func TestSanitizeMongoPoolSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
		want uint64
	}{
		{"zero", 0, defaultMaxMongoPool},
		{"negative", -1, defaultMaxMongoPool},
		{"below_min", 1, defaultMinMongoPool},
		{"at_min", defaultMinMongoPool, defaultMinMongoPool},
		{"normal", 50, 50},
		{"max_int32", math.MaxInt32, math.MaxInt32},
		{"exceed_int32", math.MaxInt32 + 1, math.MaxInt32},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeMongoPoolSize(tc.size); got != tc.want {
				t.Errorf("sanitizeMongoPoolSize(%d) = %d, want %d", tc.size, got, tc.want)
			}
		})
	}
}

func TestSanitizePostgresPoolSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
		want int
	}{
		{"zero", 0, defaultMaxPoolSize},
		{"negative", -5, defaultMaxPoolSize},
		{"below_min", 1, defaultMinPoolSize},
		{"at_min", defaultMinPoolSize, defaultMinPoolSize},
		{"normal", 20, 20},
		{"large", 500, 500},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizePostgresPoolSize(tc.size); got != tc.want {
				t.Errorf("sanitizePostgresPoolSize(%d) = %d, want %d", tc.size, got, tc.want)
			}
		})
	}
}
