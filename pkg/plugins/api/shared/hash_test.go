package shared

import (
	"testing"
)

func TestHashCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("produces_hash", func(t *testing.T) {
		t.Parallel()
		h := HashCacheKey("test-key")
		if len(h) != 32 {
			t.Errorf("hash length = %d, want 32 (16 bytes hex)", len(h))
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		h1 := HashCacheKey("same-input")
		h2 := HashCacheKey("same-input")
		if h1 != h2 {
			t.Error("HashCacheKey should be deterministic")
		}
	})

	t.Run("different_inputs", func(t *testing.T) {
		t.Parallel()
		h1 := HashCacheKey("input-a")
		h2 := HashCacheKey("input-b")
		if h1 == h2 {
			t.Error("different inputs should produce different hashes")
		}
	})

	t.Run("empty_string", func(t *testing.T) {
		t.Parallel()
		h := HashCacheKey("")
		if len(h) != 32 {
			t.Errorf("hash length = %d, want 32", len(h))
		}
	})
}
