package resilience

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retry attempts
	InitialBackoff  time.Duration // Initial backoff duration
	MaxBackoff      time.Duration // Maximum backoff duration
	BackoffMultiplier float64      // Multiplier for exponential backoff
	JitterFraction  float64       // Fraction of backoff to add as jitter (0-1)
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
	}
}

// RetryPolicy defines retry behavior
type RetryPolicy interface {
	ShouldRetry(attempt int, err error) bool
	GetBackoff(attempt int) time.Duration
	GetMaxRetries() int
}

// DefaultRetryPolicy implements RetryPolicy
type DefaultRetryPolicy struct {
	config        *RetryConfig
	errorHandler  *ErrorHandler
	shouldRetryFn func(err error) bool
}

// NewDefaultRetryPolicy creates a new default retry policy
func NewDefaultRetryPolicy(config *RetryConfig, errorHandler *ErrorHandler) *DefaultRetryPolicy {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &DefaultRetryPolicy{
		config:       config,
		errorHandler: errorHandler,
		shouldRetryFn: func(err error) bool {
			if errorHandler == nil {
				return true
			}
			return errorHandler.IsTransient(err)
		},
	}
}

// ShouldRetry determines if an operation should be retried
func (p *DefaultRetryPolicy) ShouldRetry(attempt int, err error) bool {
	if attempt >= p.config.MaxRetries {
		return false
	}

	if err == nil {
		return false
	}

	return p.shouldRetryFn(err)
}

// GetBackoff calculates the backoff duration for an attempt
func (p *DefaultRetryPolicy) GetBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate exponential backoff
	backoff := time.Duration(float64(p.config.InitialBackoff) * math.Pow(p.config.BackoffMultiplier, float64(attempt)))

	// Cap at max backoff
	if backoff > p.config.MaxBackoff {
		backoff = p.config.MaxBackoff
	}

	// Add jitter (random value between 0 and jitter fraction of backoff)
	if p.config.JitterFraction > 0 {
		jitterAmount := time.Duration(float64(backoff) * p.config.JitterFraction)
		// Use a simple pseudo-random jitter based on attempt number
		// This ensures deterministic behavior for testing
		jitterValue := time.Duration((attempt * 7) % 100) * jitterAmount / 100
		backoff = backoff + jitterValue
	}

	return backoff
}

// GetMaxRetries returns the maximum number of retries
func (p *DefaultRetryPolicy) GetMaxRetries() int {
	return p.config.MaxRetries
}

// RetryExecutor executes operations with retry logic
type RetryExecutor struct {
	policy           RetryPolicy
	logger           core.Logger
	metricsCollector core.MetricsCollector
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(policy RetryPolicy, logger core.Logger, metricsCollector core.MetricsCollector) *RetryExecutor {
	if policy == nil {
		policy = NewDefaultRetryPolicy(DefaultRetryConfig(), nil)
	}

	return &RetryExecutor{
		policy:           policy,
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Execute executes an operation with retry logic
func (e *RetryExecutor) Execute(ctx context.Context, operation func() error, source string) error {
	var lastErr error

	for attempt := 0; attempt <= e.policy.GetMaxRetries(); attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute operation
		err := operation()

		// Record attempt
		e.metricsCollector.RecordCounter("retry_attempt", 1, map[string]string{
			"source":  source,
			"attempt": fmt.Sprintf("%d", attempt),
		})

		if err == nil {
			// Success
			if attempt > 0 {
				e.metricsCollector.RecordCounter("retry_success", 1, map[string]string{
					"source": source,
				})
				e.logger.Info(fmt.Sprintf("Operation succeeded after %d retries (source: %s)", attempt, source))
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !e.policy.ShouldRetry(attempt, err) {
			e.metricsCollector.RecordCounter("retry_exhausted", 1, map[string]string{
				"source": source,
			})
			e.logger.Warn(fmt.Sprintf("Operation failed with non-retryable error after %d attempts (source: %s): %v", attempt+1, source, err))
			return err
		}

		// Check if we've exhausted retries
		if attempt >= e.policy.GetMaxRetries() {
			e.metricsCollector.RecordCounter("retry_exhausted", 1, map[string]string{
				"source": source,
			})
			e.logger.Warn(fmt.Sprintf("Operation failed after %d retries (source: %s): %v", attempt+1, source, err))
			return err
		}

		// Calculate backoff
		backoff := e.policy.GetBackoff(attempt)

		e.logger.Debug(fmt.Sprintf("Retrying operation after %v (attempt %d, source: %s)", backoff, attempt+1, source))

		// Wait before retry
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

// ExecuteWithFallback executes an operation with retry logic and fallback
func (e *RetryExecutor) ExecuteWithFallback(ctx context.Context, operation func() error, fallback func() error, source string) error {
	err := e.Execute(ctx, operation, source)
	if err != nil && fallback != nil {
		e.logger.Info(fmt.Sprintf("Executing fallback operation (source: %s)", source))
		return fallback()
	}
	return err
}

// RetryableOperation wraps an operation with retry configuration
type RetryableOperation struct {
	operation func() error
	config    *RetryConfig
	source    string
}

// NewRetryableOperation creates a new retryable operation
func NewRetryableOperation(operation func() error, config *RetryConfig, source string) *RetryableOperation {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &RetryableOperation{
		operation: operation,
		config:    config,
		source:    source,
	}
}

// Execute executes the retryable operation
func (r *RetryableOperation) Execute(ctx context.Context, executor *RetryExecutor) error {
	return executor.Execute(ctx, r.operation, r.source)
}

// RetryStats tracks retry statistics
type RetryStats struct {
	TotalAttempts    int64
	SuccessfulRetries int64
	FailedRetries    int64
	ExhaustedRetries int64
	mu               sync.RWMutex
}

// NewRetryStats creates new retry statistics
func NewRetryStats() *RetryStats {
	return &RetryStats{}
}

// RecordAttempt records a retry attempt
func (s *RetryStats) RecordAttempt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalAttempts++
}

// RecordSuccess records a successful retry
func (s *RetryStats) RecordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SuccessfulRetries++
}

// RecordFailure records a failed retry
func (s *RetryStats) RecordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FailedRetries++
}

// RecordExhausted records an exhausted retry
func (s *RetryStats) RecordExhausted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExhaustedRetries++
}

// GetStats returns retry statistics
func (s *RetryStats) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	successRate := 0.0
	if s.TotalAttempts > 0 {
		successRate = float64(s.SuccessfulRetries) / float64(s.TotalAttempts) * 100
	}

	return map[string]interface{}{
		"total_attempts":      s.TotalAttempts,
		"successful_retries":  s.SuccessfulRetries,
		"failed_retries":      s.FailedRetries,
		"exhausted_retries":   s.ExhaustedRetries,
		"success_rate":        successRate,
	}
}

// Reset resets retry statistics
func (s *RetryStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalAttempts = 0
	s.SuccessfulRetries = 0
	s.FailedRetries = 0
	s.ExhaustedRetries = 0
}
