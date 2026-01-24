package query

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy defines the retry configuration
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// JitterFraction is the fraction of jitter to add (0.0 to 1.0)
	JitterFraction float64
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
	}
}

// CalculateBackoff calculates the backoff duration for the given attempt
func (rp *RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential backoff
	backoff := float64(rp.InitialBackoff) * math.Pow(rp.BackoffMultiplier, float64(attempt-1))

	// Cap at max backoff
	if backoff > float64(rp.MaxBackoff) {
		backoff = float64(rp.MaxBackoff)
	}

	// Add jitter
	jitterAmount := backoff * rp.JitterFraction
	jitter := (rand.Float64() - 0.5) * 2 * jitterAmount
	backoff = backoff + jitter

	// Ensure backoff is positive
	if backoff < 0 {
		backoff = 0
	}

	return time.Duration(backoff)
}

// ShouldRetry returns true if the error should be retried
func (rp *RetryPolicy) ShouldRetry(err error, attempt int) bool {
	if err == nil {
		return false
	}

	if attempt >= rp.MaxAttempts {
		return false
	}

	// Only retry transient errors
	classifier := NewErrorClassifier()
	return classifier.IsTransient(err)
}

// RetryHandler handles retry logic
type RetryHandler struct {
	policy     *RetryPolicy
	classifier *ErrorClassifier
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(policy *RetryPolicy) *RetryHandler {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	return &RetryHandler{
		policy:     policy,
		classifier: NewErrorClassifier(),
	}
}

// RetryFunc is a function that can be retried
type RetryFunc func(ctx context.Context) error

// Execute executes the function with retry logic
func (rh *RetryHandler) Execute(ctx context.Context, fn RetryFunc) error {
	var lastErr error

	for attempt := 1; attempt <= rh.policy.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !rh.policy.ShouldRetry(err, attempt) {
			return err
		}

		// Calculate backoff
		backoff := rh.policy.CalculateBackoff(attempt)

		// Wait before retrying
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

// ExecuteWithTimeout executes the function with retry logic and timeout
func (rh *RetryHandler) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn RetryFunc) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return rh.Execute(ctx, fn)
}

// RetryStats tracks retry statistics
type RetryStats struct {
	// TotalAttempts is the total number of attempts
	TotalAttempts int
	// SuccessfulAttempts is the number of successful attempts
	SuccessfulAttempts int
	// FailedAttempts is the number of failed attempts
	FailedAttempts int
	// TotalBackoffTime is the total backoff time
	TotalBackoffTime time.Duration
	// LastError is the last error encountered
	LastError error
}

// ExecuteWithStats executes the function with retry logic and statistics
func (rh *RetryHandler) ExecuteWithStats(ctx context.Context, fn RetryFunc) (*RetryStats, error) {
	stats := &RetryStats{}
	var totalBackoff time.Duration

	for attempt := 1; attempt <= rh.policy.MaxAttempts; attempt++ {
		stats.TotalAttempts++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			stats.SuccessfulAttempts++
			return stats, nil
		}

		stats.FailedAttempts++
		stats.LastError = err

		// Check if we should retry
		if !rh.policy.ShouldRetry(err, attempt) {
			return stats, err
		}

		// Calculate backoff
		backoff := rh.policy.CalculateBackoff(attempt)
		totalBackoff += backoff
		stats.TotalBackoffTime = totalBackoff

		// Wait before retrying
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return stats, ctx.Err()
		}
	}

	return stats, stats.LastError
}
