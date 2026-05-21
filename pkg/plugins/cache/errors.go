package cache

import (
	"fmt"
)

// CacheError represents a cache-related error
type CacheError struct {
	Code    string
	Message string
	Err     error // underlying error, if any
}

// Error implements the error interface
func (ce *CacheError) Error() string {
	if ce.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", ce.Code, ce.Message, ce.Err)
	}
	return fmt.Sprintf("[%s] %s", ce.Code, ce.Message)
}

// Unwrap returns the underlying error, enabling errors.Is() and errors.As()
// to traverse the error chain through CacheError.
func (ce *CacheError) Unwrap() error { return ce.Err }

// Error constructors

// ErrInvalidConfig creates an invalid configuration error
func ErrInvalidConfig(message string) error {
	return &CacheError{
		Code:    "INVALID_CONFIG",
		Message: message,
	}
}

// ErrConnectionFailed creates a connection failed error
func ErrConnectionFailed(message string, wrapErrs ...error) error {
	ce := &CacheError{
		Code:    "CONNECTION_FAILED",
		Message: message,
	}
	if len(wrapErrs) > 0 {
		ce.Err = wrapErrs[0]
	}
	return ce
}

// ErrKeyNotFound creates a key not found error
func ErrKeyNotFound(key string) error {
	return &CacheError{
		Code:    "KEY_NOT_FOUND",
		Message: fmt.Sprintf("key not found: %s", key),
	}
}

// ErrSerializationFailed creates a serialization failed error
func ErrSerializationFailed(message string, wrapErrs ...error) error {
	ce := &CacheError{
		Code:    "SERIALIZATION_FAILED",
		Message: message,
	}
	if len(wrapErrs) > 0 {
		ce.Err = wrapErrs[0]
	}
	return ce
}

// ErrOperationFailed creates an operation failed error
func ErrOperationFailed(operation, message string, wrapErrs ...error) error {
	ce := &CacheError{
		Code:    "OPERATION_FAILED",
		Message: fmt.Sprintf("%s failed: %s", operation, message),
	}
	if len(wrapErrs) > 0 {
		ce.Err = wrapErrs[0]
	}
	return ce
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
