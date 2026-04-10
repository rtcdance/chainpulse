package e2e

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorType represents different types of errors
type ErrorType int

const (
	ErrorTypeTransient ErrorType = iota
	ErrorTypePermanent
	ErrorTypeCritical
	ErrorTypeTimeout
	ErrorTypeRateLimit
)

// ErrorInjectionConfig configures error injection behavior
type ErrorInjectionConfig struct {
	ErrorType       ErrorType
	ErrorRate       float64       // 0.0 to 1.0
	RecoveryTime    time.Duration // Time to recover from error
	MaxRetries      int
	RetryBackoff    time.Duration
	InjectionWindow time.Duration // How long to inject errors
}

// ErrorInjector injects errors for testing
type ErrorInjector struct {
	mu                sync.RWMutex
	config            ErrorInjectionConfig
	injectionStart    time.Time
	errorCount        int64
	recoveryCount     int64
	isRecovering      bool
	recoveryStartTime time.Time
}

// NewErrorInjector creates a new error injector
func NewErrorInjector(config ErrorInjectionConfig) *ErrorInjector {
	return &ErrorInjector{
		config:         config,
		injectionStart: time.Now(),
	}
}

// ShouldInjectError determines if an error should be injected
func (ei *ErrorInjector) ShouldInjectError() bool {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	// Check if injection window has passed
	if time.Since(ei.injectionStart) > ei.config.InjectionWindow {
		return false
	}

	// Check if currently recovering
	if ei.isRecovering {
		if time.Since(ei.recoveryStartTime) > ei.config.RecoveryTime {
			ei.isRecovering = false
			atomic.AddInt64(&ei.recoveryCount, 1)
		} else {
			return false
		}
	}

	// Probabilistic error injection
	return randomFloat() < ei.config.ErrorRate
}

// InjectError injects an error based on configuration
func (ei *ErrorInjector) InjectError() error {
	if !ei.ShouldInjectError() {
		return nil
	}

	ei.mu.Lock()
	defer ei.mu.Unlock()

	atomic.AddInt64(&ei.errorCount, 1)

	switch ei.config.ErrorType {
	case ErrorTypeTransient:
		ei.isRecovering = true
		ei.recoveryStartTime = time.Now()
		return errors.New("transient error: temporary service unavailable")

	case ErrorTypePermanent:
		return errors.New("permanent error: operation failed permanently")

	case ErrorTypeCritical:
		ei.isRecovering = true
		ei.recoveryStartTime = time.Now()
		return errors.New("critical error: system requires restart")

	case ErrorTypeTimeout:
		return errors.New("timeout error: operation exceeded time limit")

	case ErrorTypeRateLimit:
		ei.isRecovering = true
		ei.recoveryStartTime = time.Now()
		return errors.New("rate limit error: too many requests")

	default:
		return errors.New("unknown error type")
	}
}

// GetErrorCount returns the number of errors injected
func (ei *ErrorInjector) GetErrorCount() int64 {
	return atomic.LoadInt64(&ei.errorCount)
}

// GetRecoveryCount returns the number of recoveries
func (ei *ErrorInjector) GetRecoveryCount() int64 {
	return atomic.LoadInt64(&ei.recoveryCount)
}

// Reset resets the error injector
func (ei *ErrorInjector) Reset() {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	atomic.StoreInt64(&ei.errorCount, 0)
	atomic.StoreInt64(&ei.recoveryCount, 0)
	ei.isRecovering = false
	ei.injectionStart = time.Now()
}

// ErrorRecoveryHandler handles error recovery
type ErrorRecoveryHandler struct {
	mu                 sync.RWMutex
	retryCount         map[string]int
	lastErrorTime      map[string]time.Time
	recoveryStrategies map[string]RecoveryStrategy
}

// RecoveryStrategy defines how to recover from an error
type RecoveryStrategy interface {
	Recover(err error) error
	CanRecover(err error) bool
}

// ExponentialBackoffStrategy implements exponential backoff recovery
type ExponentialBackoffStrategy struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
}

// NewExponentialBackoffStrategy creates a new exponential backoff strategy
func NewExponentialBackoffStrategy(initialDelay, maxDelay time.Duration, multiplier float64) *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		initialDelay: initialDelay,
		maxDelay:     maxDelay,
		multiplier:   multiplier,
	}
}

// Recover implements recovery with exponential backoff
func (ebs *ExponentialBackoffStrategy) Recover(err error) error {
	// Backoff is handled by caller
	return nil
}

// CanRecover checks if this strategy can recover from the error
func (ebs *ExponentialBackoffStrategy) CanRecover(err error) bool {
	return err != nil
}

// GetBackoffDuration calculates backoff duration for retry count
func (ebs *ExponentialBackoffStrategy) GetBackoffDuration(retryCount int) time.Duration {
	delay := time.Duration(float64(ebs.initialDelay) * (ebs.multiplier * float64(retryCount)))
	if delay > ebs.maxDelay {
		delay = ebs.maxDelay
	}
	return delay
}

// CircuitBreakerStrategy implements circuit breaker recovery
type CircuitBreakerStrategy struct {
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	state            string // "closed", "open", "half-open"
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
}

// NewCircuitBreakerStrategy creates a new circuit breaker strategy
func NewCircuitBreakerStrategy(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreakerStrategy {
	return &CircuitBreakerStrategy{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            "closed",
	}
}

// Recover implements circuit breaker recovery
func (cbs *CircuitBreakerStrategy) Recover(err error) error {
	if err != nil {
		cbs.failureCount++
		cbs.lastFailureTime = time.Now()

		if cbs.failureCount >= cbs.failureThreshold {
			cbs.state = "open"
			return fmt.Errorf("circuit breaker open: %v", err)
		}
	} else {
		if cbs.state == "half-open" {
			cbs.successCount++
			if cbs.successCount >= cbs.successThreshold {
				cbs.state = "closed"
				cbs.failureCount = 0
				cbs.successCount = 0
			}
		}
	}

	// Check if we should transition to half-open
	if cbs.state == "open" && time.Since(cbs.lastFailureTime) > cbs.timeout {
		cbs.state = "half-open"
		cbs.successCount = 0
	}

	return nil
}

// CanRecover checks if this strategy can recover
func (cbs *CircuitBreakerStrategy) CanRecover(err error) bool {
	return cbs.state != "open"
}

// GetState returns the current circuit breaker state
func (cbs *CircuitBreakerStrategy) GetState() string {
	return cbs.state
}

// NewErrorRecoveryHandler creates a new error recovery handler
func NewErrorRecoveryHandler() *ErrorRecoveryHandler {
	return &ErrorRecoveryHandler{
		retryCount:         make(map[string]int),
		lastErrorTime:      make(map[string]time.Time),
		recoveryStrategies: make(map[string]RecoveryStrategy),
	}
}

// RegisterStrategy registers a recovery strategy for an operation
func (erh *ErrorRecoveryHandler) RegisterStrategy(operationName string, strategy RecoveryStrategy) {
	erh.mu.Lock()
	defer erh.mu.Unlock()

	erh.recoveryStrategies[operationName] = strategy
}

// HandleError handles an error with recovery
func (erh *ErrorRecoveryHandler) HandleError(operationName string, err error) error {
	erh.mu.Lock()
	defer erh.mu.Unlock()

	if err == nil {
		erh.retryCount[operationName] = 0
		return nil
	}

	erh.retryCount[operationName]++
	erh.lastErrorTime[operationName] = time.Now()

	strategy, exists := erh.recoveryStrategies[operationName]
	if !exists {
		return err
	}

	if !strategy.CanRecover(err) {
		return err
	}

	return strategy.Recover(err)
}

// GetRetryCount returns the retry count for an operation
func (erh *ErrorRecoveryHandler) GetRetryCount(operationName string) int {
	erh.mu.RLock()
	defer erh.mu.RUnlock()

	return erh.retryCount[operationName]
}

// ResetRetryCount resets the retry count for an operation
func (erh *ErrorRecoveryHandler) ResetRetryCount(operationName string) {
	erh.mu.Lock()
	defer erh.mu.Unlock()

	erh.retryCount[operationName] = 0
}

// randomFloat returns a random float between 0 and 1
func randomFloat() float64 {
	// Simple pseudo-random for testing
	return float64(time.Now().UnixNano()%1000) / 1000.0
}
