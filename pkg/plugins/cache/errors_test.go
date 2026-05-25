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

	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is should traverse CacheError chain")
	}

	unrelated := errors.New("something else")
	if errors.Is(wrapped, unrelated) {
		t.Error("errors.Is should not match unrelated error")
	}
}

func TestErrInvalidConfig(t *testing.T) {
	err := ErrInvalidConfig("RedisAddr is required")
	if err == nil {
		t.Fatal("ErrInvalidConfig should not return nil")
	}
	ce := err.(*CacheError)
	if ce.Code != "INVALID_CONFIG" {
		t.Errorf("expected code INVALID_CONFIG, got %s", ce.Code)
	}
	if ce.Message != "RedisAddr is required" {
		t.Errorf("expected message 'RedisAddr is required', got %q", ce.Message)
	}
}

func TestErrCacheFull(t *testing.T) {
	err := ErrCacheFull()
	if err == nil {
		t.Fatal("ErrCacheFull should not return nil")
	}
	ce := err.(*CacheError)
	if ce.Code != "CACHE_FULL" {
		t.Errorf("expected code CACHE_FULL, got %s", ce.Code)
	}
	if ce.Message != "cache is full" {
		t.Errorf("expected message 'cache is full', got %q", ce.Message)
	}
}

func TestErrInvalidTTL(t *testing.T) {
	err := ErrInvalidTTL()
	if err == nil {
		t.Fatal("ErrInvalidTTL should not return nil")
	}
	ce := err.(*CacheError)
	if ce.Code != "INVALID_TTL" {
		t.Errorf("expected code INVALID_TTL, got %s", ce.Code)
	}
	if ce.Message != "TTL must be positive" {
		t.Errorf("expected message 'TTL must be positive', got %q", ce.Message)
	}
}

func TestErrConnectionFailed_NoWrap(t *testing.T) {
	err := ErrConnectionFailed("redis:6379")
	if err == nil {
		t.Fatal("ErrConnectionFailed should not return nil")
	}
	ce := err.(*CacheError)
	if ce.Err != nil {
		t.Error("expected no inner error")
	}
}

func TestErrSerializationFailed_NoWrap(t *testing.T) {
	err := ErrSerializationFailed("json marshal")
	if err == nil {
		t.Fatal("ErrSerializationFailed should not return nil")
	}
	ce := err.(*CacheError)
	if ce.Err != nil {
		t.Error("expected no inner error")
	}
}

func TestErrOperationFailed_NoWrap(t *testing.T) {
	err := ErrOperationFailed("GET", "key not found")
	if err == nil {
		t.Fatal("ErrOperationFailed should not return nil")
	}
	ce := err.(*CacheError)
	if ce.Err != nil {
		t.Error("expected no inner error")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
