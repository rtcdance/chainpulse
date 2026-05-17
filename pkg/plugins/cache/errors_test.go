package cache

import (
	"errors"
	"strings"
	"testing"
)

func TestCacheError_Unwrap(t *testing.T) {
	inner := errors.New("connection refused")
	ce := ErrConnectionFailed("redis:6379", inner)

	// errors.Is should traverse through CacheError
	if !errors.Is(ce, inner) {
		t.Error("errors.Is should find wrapped error through CacheError.Unwrap()")
	}

	// Verify Unwrap returns the inner error
	if unwrapped := ce.(*CacheError).Unwrap(); unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestCacheError_UnwrapNil(t *testing.T) {
	ce := ErrKeyNotFound("mykey")
	if err := ce.(*CacheError).Unwrap(); err != nil {
		t.Errorf("Unwrap() should return nil when no inner error, got %v", err)
	}
}

func TestCacheError_ErrorWithWrap(t *testing.T) {
	inner := errors.New("timeout")
	ce := ErrSerializationFailed("json marshal", inner)
	msg := ce.Error()
	if msg == "" {
		t.Error("Error() should not be empty")
	}
	// Should contain both the code and inner error message
	if !contains(msg, "SERIALIZATION_FAILED") {
		t.Errorf("Error() should contain code, got: %s", msg)
	}
	if !contains(msg, "timeout") {
		t.Errorf("Error() should contain inner error, got: %s", msg)
	}
}

func TestCacheError_ErrorWithoutWrap(t *testing.T) {
	ce := ErrKeyNotFound("mykey")
	msg := ce.Error()
	if !contains(msg, "KEY_NOT_FOUND") {
		t.Errorf("Error() should contain code, got: %s", msg)
	}
	if contains(msg, "mykey") == false {
		t.Errorf("Error() should contain key name, got: %s", msg)
	}
}

func TestCacheError_ErrorsIsWithSystemError(t *testing.T) {
	inner := errors.New("dial tcp: connection refused")
	wrapped := ErrOperationFailed("SET", "write error", inner)

	// Should be able to find the inner error
	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is should traverse CacheError chain")
	}

	// Should NOT match an unrelated error
	unrelated := errors.New("something else")
	if errors.Is(wrapped, unrelated) {
		t.Error("errors.Is should not match unrelated error")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
