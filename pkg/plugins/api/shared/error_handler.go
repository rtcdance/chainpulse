package shared

import (
	"fmt"
	"sync"
	"time"
)

// ErrorHandler handles errors with circuit breaker and retry logic
type ErrorHandler struct {
	circuitBreaker *CircuitBreaker
	retryPolicy    *RetryPolicy
	mu             sync.RWMutex
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	state              CircuitState
	failureCount       int
	successCount       int
	lastFailureTime    time.Time
	failureThreshold   int
	successThreshold   int
	timeout            time.Duration
	mu                 sync.RWMutex
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	maxRetries      int
	initialBackoff  time.Duration
	maxBackoff      time.Duration
	backoffMultiplier float64
}

// ErrorClassification represents error severity
type ErrorClassification int

const (
	ErrorTransient ErrorClassification = iota
	ErrorPermanent
	ErrorUnknown
)

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		circuitBreaker: NewCircuitBreaker(5, 2, 30*time.Second),
		retryPolicy:    NewRetryPolicy(3, 100*time.Millisecond, 5*time.Second, 2.0),
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(maxRetries int, initialBackoff, maxBackoff time.Duration, multiplier float64) *RetryPolicy {
	return &RetryPolicy{
		maxRetries:        maxRetries,
		initialBackoff:    initialBackoff,
		maxBackoff:        maxBackoff,
		backoffMultiplier: multiplier,
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Stay closed
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}

	case StateOpen:
		// Ignore success in open state
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		}

	case StateHalfOpen:
		cb.state = StateOpen
		cb.failureCount = 0
		cb.successCount = 0

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = StateHalfOpen
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// CanExecute checks if operation can be executed
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateHalfOpen:
		return true

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			return true
		}
		return false

	default:
		return false
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetBackoffDuration calculates backoff duration for retry attempt
func (rp *RetryPolicy) GetBackoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	backoff := time.Duration(float64(rp.initialBackoff) * (rp.backoffMultiplier * float64(attempt-1)))

	if backoff > rp.maxBackoff {
		backoff = rp.maxBackoff
	}

	return backoff
}

// ClassifyError classifies an error
func (eh *ErrorHandler) ClassifyError(err error) ErrorClassification {
	if err == nil {
		return ErrorUnknown
	}

	errMsg := err.Error()

	// Transient errors
	transientPatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"unavailable",
		"service unavailable",
	}

	for _, pattern := range transientPatterns {
		if len(errMsg) > 0 && len(pattern) > 0 {
			// Simple substring check
			for i := 0; i <= len(errMsg)-len(pattern); i++ {
				if errMsg[i:i+len(pattern)] == pattern {
					return ErrorTransient
				}
			}
		}
	}

	// Permanent errors
	permanentPatterns := []string{
		"not found",
		"invalid",
		"unauthorized",
		"forbidden",
		"bad request",
	}

	for _, pattern := range permanentPatterns {
		if len(errMsg) > 0 && len(pattern) > 0 {
			for i := 0; i <= len(errMsg)-len(pattern); i++ {
				if errMsg[i:i+len(pattern)] == pattern {
					return ErrorPermanent
				}
			}
		}
	}

	return ErrorUnknown
}

// ShouldRetry determines if operation should be retried
func (eh *ErrorHandler) ShouldRetry(err error, attempt int) bool {
	if attempt >= eh.retryPolicy.maxRetries {
		return false
	}

	classification := eh.ClassifyError(err)
	return classification == ErrorTransient || classification == ErrorUnknown
}

// Handle handles an error with retry and circuit breaker logic
func (eh *ErrorHandler) Handle(err error, operation func() error) error {
	if !eh.circuitBreaker.CanExecute() {
		return fmt.Errorf("circuit breaker open: %w", err)
	}

	for attempt := 0; attempt < eh.retryPolicy.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := eh.retryPolicy.GetBackoffDuration(attempt)
			time.Sleep(backoff)
		}

		result := operation()
		if result == nil {
			eh.circuitBreaker.RecordSuccess()
			return nil
		}

		if !eh.ShouldRetry(result, attempt) {
			eh.circuitBreaker.RecordFailure()
			return result
		}

		if attempt == eh.retryPolicy.maxRetries-1 {
			eh.circuitBreaker.RecordFailure()
			return result
		}
	}

	eh.circuitBreaker.RecordFailure()
	return err
}

// GetMetrics returns error handler metrics
func (eh *ErrorHandler) GetMetrics() map[string]interface{} {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	cb := eh.circuitBreaker
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stateStr := "unknown"
	switch cb.state {
	case StateClosed:
		stateStr = "closed"
	case StateOpen:
		stateStr = "open"
	case StateHalfOpen:
		stateStr = "half-open"
	}

	return map[string]interface{}{
		"circuit_breaker_state": stateStr,
		"failure_count":         cb.failureCount,
		"success_count":         cb.successCount,
		"last_failure_time":     cb.lastFailureTime,
		"max_retries":           eh.retryPolicy.maxRetries,
	}
}
