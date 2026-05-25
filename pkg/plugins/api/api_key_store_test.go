package api

import (
	"testing"
)

func TestKeyHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		plainKey    string
		wantEmpty   bool
	}{
		{"valid cp_ prefix", "cp_abc123", false},
		{"empty string", "", true},
		{"no cp_ prefix", "normal_key_abc", true},
		{"cp_ only", "cp_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := KeyHash(tt.plainKey)
			if tt.wantEmpty && result != "" {
				t.Errorf("KeyHash(%q) = %q, want empty", tt.plainKey, result)
			}
			if !tt.wantEmpty && result == "" {
				t.Errorf("KeyHash(%q) should not be empty", tt.plainKey)
			}
		})
	}
}

func TestKeyHash_Deterministic(t *testing.T) {
	t.Parallel()
	h1 := KeyHash("cp_my_secret_key")
	h2 := KeyHash("cp_my_secret_key")
	if h1 != h2 {
		t.Error("KeyHash should be deterministic")
	}
	if h1 == "" {
		t.Error("KeyHash for valid cp_ key should not be empty")
	}
	if len(h1) != 64 {
		t.Errorf("KeyHash length = %d, want 64 (sha256 hex)", len(h1))
	}
}

func TestNewAPIKeyStore(t *testing.T) {
	t.Parallel()
	store := NewAPIKeyStore(nil, nil, nil)
	if store == nil {
		t.Fatal("NewAPIKeyStore should return a non-nil store")
	}
	if store.db != nil {
		t.Error("expected nil db")
	}
	if store.done == nil {
		t.Error("expected done channel to be initialized")
	}
}