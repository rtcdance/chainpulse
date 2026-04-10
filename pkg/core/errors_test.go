package core

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestErrorVariables tests error variable definitions
func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrInvalidBlockNumber)
	assert.NotNil(t, ErrInvalidTransactionHash)
	assert.NotNil(t, ErrInvalidContractAddress)
	assert.NotNil(t, ErrInvalidEventName)
	assert.NotNil(t, ErrInvalidAddress)
	assert.NotNil(t, ErrInvalidBlockHash)
	assert.NotNil(t, ErrInvalidLogIndex)
	assert.NotNil(t, ErrInvalidEventData)
	assert.NotNil(t, ErrInvalidTimestamp)
}

// TestErrorCodeConstants tests error code constants
func TestErrorCodeConstants(t *testing.T) {
	assert.Equal(t, "VALIDATION_ERROR", ErrorCodeValidation)
	assert.Equal(t, "NOT_FOUND", ErrorCodeNotFound)
	assert.Equal(t, "DUPLICATE", ErrorCodeDuplicate)
	assert.Equal(t, "DATABASE_ERROR", ErrorCodeDatabaseError)
	assert.Equal(t, "CACHE_ERROR", ErrorCodeCacheError)
	assert.Equal(t, "MQ_ERROR", ErrorCodeMQError)
	assert.Equal(t, "NETWORK_ERROR", ErrorCodeNetworkError)
	assert.Equal(t, "TIMEOUT", ErrorCodeTimeout)
	assert.Equal(t, "INTERNAL_ERROR", ErrorCodeInternalError)
	assert.Equal(t, "CONFIG_ERROR", ErrorCodeConfigError)
}

// TestNewSystemError tests SystemError creation
func TestNewSystemError(t *testing.T) {
	originalErr := errors.New("original error")
	sysErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Network failed", originalErr)

	assert.Equal(t, ErrorTypeTransient, sysErr.Type)
	assert.Equal(t, ErrorCodeNetworkError, sysErr.Code)
	assert.Equal(t, "Network failed", sysErr.Message)
	assert.Equal(t, originalErr, sysErr.Err)
	assert.NotNil(t, sysErr.Details)
}

// TestSystemErrorError tests SystemError.Error() method
func TestSystemErrorError(t *testing.T) {
	sysErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Connection failed", nil)
	errorStr := sysErr.Error()

	assert.Contains(t, errorStr, "transient")
	assert.Contains(t, errorStr, "NETWORK_ERROR")
	assert.Contains(t, errorStr, "Connection failed")
}

// TestSystemErrorErrorWithWrappedError tests SystemError.Error() with wrapped error
func TestSystemErrorErrorWithWrappedError(t *testing.T) {
	originalErr := errors.New("connection refused")
	sysErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Connection failed", originalErr)
	errorStr := sysErr.Error()

	assert.Contains(t, errorStr, "connection refused")
}

// TestSystemErrorWithDetail tests WithDetail method
func TestSystemErrorWithDetail(t *testing.T) {
	sysErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
	_ = sysErr.WithDetail("service", "api-gateway")
	_ = sysErr.WithDetail("port", 8080)

	assert.Equal(t, "api-gateway", sysErr.Details["service"])
	assert.Equal(t, 8080, sysErr.Details["port"])
}

// TestSystemErrorIsTransient tests IsTransient method
func TestSystemErrorIsTransient(t *testing.T) {
	transientErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
	permanentErr := NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "Error", nil)

	assert.True(t, transientErr.IsTransient())
	assert.False(t, permanentErr.IsTransient())
}

// TestSystemErrorIsPermanent tests IsPermanent method
func TestSystemErrorIsPermanent(t *testing.T) {
	transientErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
	permanentErr := NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "Error", nil)

	assert.False(t, transientErr.IsPermanent())
	assert.True(t, permanentErr.IsPermanent())
}

// TestSystemErrorIsCritical tests IsCritical method
func TestSystemErrorIsCritical(t *testing.T) {
	criticalErr := NewSystemError(ErrorTypeCritical, ErrorCodeInternalError, "Error", nil)
	transientErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)

	assert.True(t, criticalErr.IsCritical())
	assert.False(t, transientErr.IsCritical())
}

// TestClassifyErrorNil tests ClassifyError with nil error
func TestClassifyErrorNil(t *testing.T) {
	errorType := ClassifyError(nil)
	assert.Equal(t, ErrorTypePermanent, errorType)
}

// TestClassifyErrorSystemError tests ClassifyError with SystemError
func TestClassifyErrorSystemError(t *testing.T) {
	sysErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
	errorType := ClassifyError(sysErr)
	assert.Equal(t, ErrorTypeTransient, errorType)
}

// TestClassifyErrorContextDeadlineExceeded tests ClassifyError with context deadline
func TestClassifyErrorContextDeadlineExceeded(t *testing.T) {
	errorType := ClassifyError(context.DeadlineExceeded)
	assert.Equal(t, ErrorTypeTransient, errorType)
}

// TestClassifyErrorContextCanceled tests ClassifyError with context canceled
func TestClassifyErrorContextCanceled(t *testing.T) {
	errorType := ClassifyError(context.Canceled)
	assert.Equal(t, ErrorTypePermanent, errorType)
}

// TestClassifyErrorSyscall tests ClassifyError with syscall errors
func TestClassifyErrorSyscall(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{"Connection Refused", syscall.ECONNREFUSED, ErrorTypeTransient},
		{"Connection Reset", syscall.ECONNRESET, ErrorTypeTransient},
		{"Timeout", syscall.ETIMEDOUT, ErrorTypeTransient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := ClassifyError(tt.err)
			assert.Equal(t, tt.expected, errorType)
		})
	}
}

// TestClassifyErrorGeneric tests ClassifyError with generic error
func TestClassifyErrorGeneric(t *testing.T) {
	err := errors.New("generic error")
	errorType := ClassifyError(err)
	assert.Equal(t, ErrorTypePermanent, errorType)
}

// TestDefaultRetryConfig tests DefaultRetryConfig
func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 10*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
}

// TestRetryWithBackoffSuccess tests RetryWithBackoff with successful operation
func TestRetryWithBackoffSuccess(t *testing.T) {
	config := DefaultRetryConfig()
	attempts := 0

	err := RetryWithBackoff(context.Background(), config, func() error {
		attempts++
		if attempts < 2 {
			return NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

// TestRetryWithBackoffMaxRetriesExceeded tests RetryWithBackoff with max retries exceeded
func TestRetryWithBackoffMaxRetriesExceeded(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}
	attempts := 0

	err := RetryWithBackoff(context.Background(), config, func() error {
		attempts++
		return NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
	})

	assert.Error(t, err)
	assert.Equal(t, 2, attempts)
}

// TestRetryWithBackoffPermanentError tests RetryWithBackoff with permanent error
func TestRetryWithBackoffPermanentError(t *testing.T) {
	config := DefaultRetryConfig()
	attempts := 0

	err := RetryWithBackoff(context.Background(), config, func() error {
		attempts++
		return NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "Validation failed", nil)
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}

// TestRetryWithBackoffContextCanceled tests RetryWithBackoff with context canceled
func TestRetryWithBackoffContextCanceled(t *testing.T) {
	config := DefaultRetryConfig()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RetryWithBackoff(ctx, config, func() error {
		return NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestIsContextError tests IsContextError
func TestIsContextError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"Context Canceled", context.Canceled, true},
		{"Context Deadline Exceeded", context.DeadlineExceeded, true},
		{"Generic Error", errors.New("error"), false},
		{"Nil Error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsContextError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWrapError tests WrapError
func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError(originalErr, "additional context")

	assert.NotNil(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "additional context")
	assert.Contains(t, wrappedErr.Error(), "original error")
}

// TestWrapErrorNil tests WrapError with nil error
func TestWrapErrorNil(t *testing.T) {
	wrappedErr := WrapError(nil, "additional context")
	assert.Nil(t, wrappedErr)
}

// TestConfigurationConstants tests configuration constants
func TestConfigurationConstants(t *testing.T) {
	assert.Equal(t, 10, DefaultWorkerPoolSize)
	assert.Equal(t, 100, DefaultBatchSize)
	assert.Equal(t, 3, DefaultMaxRetries)
	assert.Equal(t, 100, DefaultRetryBackoff)
	assert.Equal(t, 3600, DefaultCacheTTL)
	assert.Equal(t, 8080, DefaultAPIPort)
}

// TestRetryWithBackoffExponentialBackoff tests exponential backoff calculation
func TestRetryWithBackoffExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}
	attempts := 0

	err := RetryWithBackoff(context.Background(), config, func() error {
		attempts++
		if attempts < 4 {
			return NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil)
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 4, attempts)
}

// TestSystemErrorChaining tests error chaining with WithDetail
func TestSystemErrorChaining(t *testing.T) {
	sysErr := NewSystemError(ErrorTypeTransient, ErrorCodeNetworkError, "Error", nil).
		WithDetail("service", "api").
		WithDetail("port", 8080).
		WithDetail("retry", 3)

	assert.Equal(t, "api", sysErr.Details["service"])
	assert.Equal(t, 8080, sysErr.Details["port"])
	assert.Equal(t, 3, sysErr.Details["retry"])
}
