package core

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"net"
	"sync"
	"syscall"
	"time"
)

// ErrorTypeConnection represents a connection error
const ErrorTypeConnection = "connection"

// ErrorTypeTimeout represents a timeout error
const ErrorTypeTimeout = "timeout"

// ErrorTypeUnknown represents an unknown error type
const ErrorTypeUnknown = "unknown"

// MQErrorHandler handles errors and recovery for MQ operations
type MQErrorHandler struct {
	mu                   sync.Mutex
	logger               Logger
	metricsCollector     MetricsCollector
	maxRetries           int
	baseRetryDelay       time.Duration
	maxRetryDelay        time.Duration
	timeoutDuration      time.Duration
	permanentErrorCodes  map[string]bool
	transientErrorCodes  map[string]bool
	consecutiveErrors    int64
	lastErrorTime        time.Time
	degradedMode         bool
	recoveryAttempts     int64
	successfulRecoveries int64
}

// NewMQErrorHandler creates a new MQ error handler
func NewMQErrorHandler(
	logger Logger,
	metricsCollector MetricsCollector,
	maxRetries int,
	baseRetryDelay time.Duration,
) *MQErrorHandler {
	return &MQErrorHandler{
		logger:               logger,
		metricsCollector:     metricsCollector,
		maxRetries:           maxRetries,
		baseRetryDelay:       baseRetryDelay,
		maxRetryDelay:        30 * time.Second,
		timeoutDuration:      5 * time.Second,
		permanentErrorCodes:  make(map[string]bool),
		transientErrorCodes:  make(map[string]bool),
		consecutiveErrors:    0,
		degradedMode:         false,
		recoveryAttempts:     0,
		successfulRecoveries: 0,
	}
}

// ClassifyError classifies an error into a specific type using errors.Is/As
// instead of fragile string matching.
func (eh *MQErrorHandler) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// Check for project sentinel errors via errors.Is
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrDeadlineExceeded) {
		return ErrorTypeTimeout
	}
	if errors.Is(err, ErrConnectionRefused) || errors.Is(err, ErrConnectionReset) {
		return ErrorTypeConnection
	}
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrNotFound) || errors.Is(err, ErrBadRequest) ||
		errors.Is(err, ErrInvalidState) || errors.Is(err, ErrAuthFailed) {
		return ErrorTypePermanent
	}
	if errors.Is(err, ErrDataCorruption) || errors.Is(err, ErrCriticalFailure) {
		return ErrorTypeCritical
	}

	// Check for standard library errors via errors.Is
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTypeTimeout
	}

	// Check for network errors via errors.As (covers "connection refused", "connection reset", etc.)
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrorTypeTimeout
		}
		return ErrorTypeConnection
	}

	// Check for syscall errors (ECONNREFUSED, ECONNRESET, EPIPE)
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		switch sysErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.EPIPE:
			return ErrorTypeConnection
		}
	}

	// Check for SystemError via errors.As (our custom type)
	var sysErr2 *SystemError
	if errors.As(err, &sysErr2) {
		switch sysErr2.Type {
		case ErrorTypeTransient:
			return ErrorTypeTransient
		case ErrorTypePermanent:
			return ErrorTypePermanent
		case ErrorTypeCritical:
			return ErrorTypeCritical
		}
	}

	// Check registered error codes (string-based fallback for external errors)
	errStr := err.Error()
	if eh.permanentErrorCodes[errStr] {
		return ErrorTypePermanent
	}
	if eh.transientErrorCodes[errStr] {
		return ErrorTypeTransient
	}

	// Default to transient for unknown errors
	return ErrorTypeTransient
}

// HandleError handles an error and returns whether to retry
func (eh *MQErrorHandler) HandleError(ctx context.Context, err error, operationName string) (bool, error) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	if err == nil {
		// Reset consecutive errors on success
		eh.consecutiveErrors = 0
		if eh.degradedMode {
			eh.degradedMode = false
			eh.logger.Info("recovered from degraded mode", "operation", operationName)
			eh.metricsCollector.RecordCounter("mq_recovery_from_degraded", int64(1), nil)
		}
		return false, nil
	}

	errorType := eh.ClassifyError(err)
	eh.consecutiveErrors++
	eh.lastErrorTime = time.Now()

	// Record error metrics
	eh.metricsCollector.RecordCounter("mq_errors", int64(1), map[string]string{
		"operation": operationName,
		"type":      string(errorType),
	})

	eh.logger.Error(
		"MQ operation error",
		"operation", operationName,
		"error_type", string(errorType),
		"error", err,
		"consecutive_errors", eh.consecutiveErrors,
	)

	// Determine if we should retry based on error type
	switch errorType {
	case ErrorTypeConnection:
		// Connection errors are retryable
		eh.logger.Warn("connection error, will retry", "operation", operationName)
		return true, nil

	case ErrorTypeTimeout:
		// Timeout errors are retryable
		eh.logger.Warn("timeout error, will retry", "operation", operationName)
		return true, nil

	case ErrorTypePermanent:
		// Permanent errors should not be retried
		eh.logger.Error("permanent error, will not retry", "operation", operationName)
		eh.degradedMode = true
		eh.metricsCollector.RecordCounter("mq_permanent_errors", int64(1), map[string]string{"operation": operationName})
		return false, err

	case ErrorTypeTransient:
		// Transient errors are retryable
		eh.logger.Warn("transient error, will retry", "operation", operationName)
		return true, nil

	default:
		// Unknown errors are retryable by default
		eh.logger.Warn("unknown error type, will retry", "operation", operationName)
		return true, nil
	}
}

// RetryWithBackoff retries an operation with exponential backoff
func (eh *MQErrorHandler) RetryWithBackoff(ctx context.Context, operation func() error, operationName string) error {
	var lastErr error

	for attempt := 1; attempt <= eh.maxRetries; attempt++ {
		// Try the operation
		err := operation()
		if err == nil {
			// Success
			eh.mu.Lock()
			if attempt > 1 {
				eh.successfulRecoveries++
				eh.mu.Unlock()
				eh.logger.Info("operation succeeded after retry", "operation", operationName, "attempt", attempt)
				eh.metricsCollector.RecordCounter("mq_successful_recovery", int64(1), map[string]string{"operation": operationName})
			} else {
				eh.mu.Unlock()
			}
			return nil
		}

		lastErr = err
		shouldRetry, _ := eh.HandleError(ctx, err, operationName)

		if !shouldRetry {
			// Don't retry permanent errors
			return err
		}

		// Don't wait after the last attempt
		if attempt < eh.maxRetries {
			// Calculate exponential backoff delay
			delay := eh.CalculateBackoffDelay(attempt)
			eh.logger.Info("retrying operation", "operation", operationName, "attempt", attempt+1, "delay_ms", delay.Milliseconds())

			// Record retry attempt
			eh.mu.Lock()
			eh.recoveryAttempts++
			eh.mu.Unlock()
			eh.metricsCollector.RecordCounter("mq_retry_attempts", int64(1), map[string]string{"operation": operationName})

			// Wait for the delay or until context is cancelled
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// All retries exhausted
	eh.mu.Lock()
	eh.degradedMode = true
	eh.mu.Unlock()
	eh.logger.Error("operation failed after all retries", "operation", operationName, "max_retries", eh.maxRetries, "last_error", lastErr)
	eh.metricsCollector.RecordCounter("mq_retries_exhausted", int64(1), map[string]string{"operation": operationName})

	return fmt.Errorf("operation %s failed after %d retries: %w", operationName, eh.maxRetries, lastErr)
}

// CalculateBackoffDelay calculates exponential backoff delay with jitter
// Formula: min(baseDelay * (2 ^ (attempt - 1)), maxRetryDelay) + jitter
func (eh *MQErrorHandler) CalculateBackoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return eh.baseRetryDelay
	}

	// Calculate 2^(attempt-1) using bit shift
	shift := attempt - 1
	// Bound the shift to avoid int->uint overflow warnings (gosec G115) and to keep
	// the multiplier within a sane range even if callers pass very large attempt values.
	maxShift := bits.UintSize - 2
	if shift < 0 {
		shift = 0
	} else if shift > maxShift {
		shift = maxShift
	}
	delayMultiplier := 1 << shift
	delay := eh.baseRetryDelay * time.Duration(delayMultiplier)

	// Cap at max retry delay
	if delay > eh.maxRetryDelay {
		delay = eh.maxRetryDelay
	}

	// Add jitter (±10% of delay)
	jitter := time.Duration(int64(delay) / 10)
	if jitter > 0 {
		// Simple jitter: subtract half jitter to add half jitter
		delay = delay - jitter/2
	}

	return delay
}

// RegisterPermanentError registers an error code as permanent
func (eh *MQErrorHandler) RegisterPermanentError(errorCode string) {
	eh.permanentErrorCodes[errorCode] = true
	eh.logger.Info("registered permanent error code", "error_code", errorCode)
}

// RegisterTransientError registers an error code as transient
func (eh *MQErrorHandler) RegisterTransientError(errorCode string) {
	eh.transientErrorCodes[errorCode] = true
	eh.logger.Info("registered transient error code", "error_code", errorCode)
}

// IsInDegradedMode returns whether the handler is in degraded mode
func (eh *MQErrorHandler) IsInDegradedMode() bool {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	return eh.degradedMode
}

// SetDegradedMode sets the degraded mode flag
func (eh *MQErrorHandler) SetDegradedMode(degraded bool) {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	eh.degradedMode = degraded
	if degraded {
		eh.logger.Warn("entering degraded mode")
		eh.metricsCollector.RecordCounter("mq_degraded_mode_entered", int64(1), nil)
	} else {
		eh.logger.Info("exiting degraded mode")
		eh.metricsCollector.RecordCounter("mq_degraded_mode_exited", int64(1), nil)
	}
}

// GetConsecutiveErrorCount returns the number of consecutive errors
func (eh *MQErrorHandler) GetConsecutiveErrorCount() int64 {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	return eh.consecutiveErrors
}

// ResetConsecutiveErrorCount resets the consecutive error count
func (eh *MQErrorHandler) ResetConsecutiveErrorCount() {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	eh.consecutiveErrors = 0
}

// GetRecoveryStats returns recovery statistics
func (eh *MQErrorHandler) GetRecoveryStats() map[string]interface{} {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	return map[string]interface{}{
		"recovery_attempts":     eh.recoveryAttempts,
		"successful_recoveries": eh.successfulRecoveries,
		"consecutive_errors":    eh.consecutiveErrors,
		"last_error_time":       eh.lastErrorTime,
		"degraded_mode":         eh.degradedMode,
		"max_retries":           eh.maxRetries,
		"base_retry_delay_ms":   eh.baseRetryDelay.Milliseconds(),
		"max_retry_delay_ms":    eh.maxRetryDelay.Milliseconds(),
		"timeout_duration_ms":   eh.timeoutDuration.Milliseconds(),
	}
}

// SetTimeoutDuration sets the timeout duration for operations
func (eh *MQErrorHandler) SetTimeoutDuration(duration time.Duration) {
	eh.timeoutDuration = duration
	eh.logger.Info("timeout duration set", "duration_ms", duration.Milliseconds())
}

// GetTimeoutDuration returns the timeout duration
func (eh *MQErrorHandler) GetTimeoutDuration() time.Duration {
	return eh.timeoutDuration
}

// SetMaxRetries sets the maximum number of retries
func (eh *MQErrorHandler) SetMaxRetries(maxRetries int) {
	eh.maxRetries = maxRetries
	eh.logger.Info("max retries set", "max_retries", maxRetries)
}

// SetMaxRetryDelay sets the maximum retry delay
func (eh *MQErrorHandler) SetMaxRetryDelay(delay time.Duration) {
	eh.maxRetryDelay = delay
	eh.logger.Info("max retry delay set", "delay_ms", delay.Milliseconds())
}
