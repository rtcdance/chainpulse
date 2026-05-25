package api

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()

	lengths := []int{16, 32, 64}
	for _, l := range lengths {
		l := l
		t.Run("length_"+string(rune('0'+l/10))+string(rune('0'+l%10)), func(t *testing.T) {
			t.Parallel()
			key, err := generateAPIKey(l)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(key) != l {
				t.Errorf("key length = %d, want %d", len(key), l)
			}
			for _, c := range key {
				if !strings.ContainsRune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", c) {
					t.Errorf("invalid character %c in key", c)
				}
			}
		})
	}
}

func TestHashKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"normal", "my-secret-key"},
		{"empty", ""},
		{"long", strings.Repeat("x", 256)},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			hash := hashKey(tc.key)
			if len(hash) != 64 {
				t.Errorf("hash length = %d, want 64 (SHA256 hex)", len(hash))
			}
		})
	}
}

func TestHashKeyDeterministic(t *testing.T) {
	t.Parallel()

	key := "test-key"
	h1 := hashKey(key)
	h2 := hashKey(key)
	if h1 != h2 {
		t.Error("hashKey should be deterministic")
	}
}

func TestGenerateID(t *testing.T) {
	t.Parallel()

	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generated ID should not be empty")
	}
	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}
}
