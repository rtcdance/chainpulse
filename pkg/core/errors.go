package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

// Error variables for validation
var (
	ErrInvalidBlockNumber     = errors.New("invalid block number")
	ErrInvalidTransactionHash = errors.New("invalid transaction hash")
	ErrInvalidContractAddress = errors.New("invalid contract address")
	ErrInvalidEventName       = errors.New("invalid event name")
	ErrInvalidAddress         = errors.New("invalid address")
	ErrInvalidBlockHash       = errors.New("invalid block hash")
	ErrInvalidLogIndex        = errors.New("invalid log index")
	ErrInvalidEventData       = errors.New("invalid event data")
	ErrInvalidTimestamp       = errors.New("invalid timestamp")
)

// Constants for configuration
const (
	DefaultWorkerPoolSize = 10
	DefaultBatchSize      = 100
	DefaultMaxRetries     = 3
	DefaultRetryBackoff   = 100  // milliseconds
	DefaultCacheTTL       = 3600 // seconds (1 hour)
	DefaultAPIPort        = 8080
)

// Constants for error codes
const (
	ErrorCodeValidation    = "VALIDATION_ERROR"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeDuplicate     = "DUPLICATE"
	ErrorCodeDatabaseError = "DATABASE_ERROR"
	ErrorCodeCacheError    = "CACHE_ERROR"
	ErrorCodeMQError       = "MQ_ERROR"
	ErrorCodeNetworkError  = "NETWORK_ERROR"
	ErrorCodeTimeout       = "TIMEOUT"
	ErrorCodeInternalError = "INTERNAL_ERROR"
	ErrorCodeConfigError   = "CONFIG_ERROR"
)

// NewSystemError creates a new system error
func NewSystemError(errorType ErrorType, code, message string, err error) *SystemError {
	return &SystemError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// Error implements the error interface
func (e *SystemError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (%v)", e.Type, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// WithDetail adds a detail to the error
func (e *SystemError) WithDetail(key string, value interface{}) *SystemError {
	e.Details[key] = value
	return e
}

// IsTransient checks if error is transient
func (e *SystemError) IsTransient() bool {
	return e.Type == ErrorTypeTransient
}

// IsPermanent checks if error is permanent
func (e *SystemError) IsPermanent() bool {
	return e.Type == ErrorTypePermanent
}

// IsCritical checks if error is critical
func (e *SystemError) IsCritical() bool {
	return e.Type == ErrorTypeCritical
}

// ClassifyError classifies an error as transient, permanent, or critical
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypePermanent
	}

	// Check if it's a SystemError first
	if sysErr, ok := err.(*SystemError); ok {
		return sysErr.Type
	}

	// Check for network errors (transient)
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return ErrorTypeTransient
		}
	}

	// Note: Temporary() is deprecated, but we keep timeout check above

	// Check for syscall errors (transient)
	if err == syscall.ECONNREFUSED || err == syscall.ECONNRESET || err == syscall.ETIMEDOUT {
		return ErrorTypeTransient
	}

	// Check for context deadline exceeded (transient)
	if err == context.DeadlineExceeded {
		return ErrorTypeTransient
	}

	// Default to permanent
	return ErrorTypePermanent
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryWithBackoff retries an operation with exponential backoff
func RetryWithBackoff(ctx context.Context, config RetryConfig, operation func() error) error {
	var lastErr error
	delay := config.InitialDelay

	// MaxRetries represents the total number of attempts allowed
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is transient
		if ClassifyError(err) != ErrorTypeTransient {
			return err
		}

		// Check if this is the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Wait before retrying
		select {
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			nextDelay := time.Duration(float64(delay) * config.Multiplier)
			if nextDelay > config.MaxDelay {
				nextDelay = config.MaxDelay
			}
			delay = nextDelay
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return NewSystemError(
		ErrorTypeTransient,
		ErrorCodeNetworkError,
		"max retries exceeded",
		lastErr,
	)
}

// IsContextError checks if error is a context error
func IsContextError(err error) bool {
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	return false
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
