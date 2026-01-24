package cache

import "fmt"

// CacheError represents a cache-related error
type CacheError struct {
	Code    string
	Message string
}

// Error implements the error interface
func (ce *CacheError) Error() string {
	return fmt.Sprintf("[%s] %s", ce.Code, ce.Message)
}

// Error constructors

// ErrInvalidConfig creates an invalid configuration error
func ErrInvalidConfig(message string) error {
	return &CacheError{
		Code:    "INVALID_CONFIG",
		Message: message,
	}
}

// ErrConnectionFailed creates a connection failed error
func ErrConnectionFailed(message string) error {
	return &CacheError{
		Code:    "CONNECTION_FAILED",
		Message: message,
	}
}

// ErrKeyNotFound creates a key not found error
func ErrKeyNotFound(key string) error {
	return &CacheError{
		Code:    "KEY_NOT_FOUND",
		Message: fmt.Sprintf("key not found: %s", key),
	}
}

// ErrSerializationFailed creates a serialization failed error
func ErrSerializationFailed(message string) error {
	return &CacheError{
		Code:    "SERIALIZATION_FAILED",
		Message: message,
	}
}

// ErrOperationFailed creates an operation failed error
func ErrOperationFailed(operation, message string) error {
	return &CacheError{
		Code:    "OPERATION_FAILED",
		Message: fmt.Sprintf("%s failed: %s", operation, message),
	}
}

// ErrCacheFull creates a cache full error
func ErrCacheFull() error {
	return &CacheError{
		Code:    "CACHE_FULL",
		Message: "cache is full",
	}
}

// ErrInvalidTTL creates an invalid TTL error
func ErrInvalidTTL() error {
	return &CacheError{
		Code:    "INVALID_TTL",
		Message: "TTL must be positive",
	}
}
