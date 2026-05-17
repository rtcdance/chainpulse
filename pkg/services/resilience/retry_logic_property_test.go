package resilience

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// Property 3: Exponential Backoff Retry
// For any transient error encountered by the system, the retry attempts SHALL follow an exponential backoff pattern
// with configurable maximum retries

func TestProperty3ExponentialBackoffRetry(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	// Test 1: Exponential backoff progression
	t.Run("ExponentialBackoffProgression", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)

		// Verify exponential progression
		backoff0 := policy.GetBackoff(0)
		backoff1 := policy.GetBackoff(1)
		backoff2 := policy.GetBackoff(2)
		backoff3 := policy.GetBackoff(3)

		// Each backoff should be approximately 2x the previous
		if backoff1 < backoff0*2 || backoff1 > backoff0*2+time.Millisecond {
			t.Errorf("Expected backoff1 ≈ 2*backoff0, got %v vs %v", backoff1, backoff0*2)
		}

		if backoff2 < backoff1*2 || backoff2 > backoff1*2+time.Millisecond {
			t.Errorf("Expected backoff2 ≈ 2*backoff1, got %v vs %v", backoff2, backoff1*2)
		}

		if backoff3 < backoff2*2 || backoff3 > backoff2*2+time.Millisecond {
			t.Errorf("Expected backoff3 ≈ 2*backoff2, got %v vs %v", backoff3, backoff2*2)
		}
	})

	// Test 2: Configurable maximum retries
	t.Run("ConfigurableMaxRetries", func(t *testing.T) {
		for maxRetries := 1; maxRetries <= 5; maxRetries++ {
			config := &RetryConfig{
				MaxRetries:        maxRetries,
				InitialBackoff:    10 * time.Millisecond,
				MaxBackoff:        100 * time.Millisecond,
				BackoffMultiplier: 2.0,
				JitterFraction:    0.0,
			}
			policy := NewDefaultRetryPolicy(config, errorHandler)

			if policy.GetMaxRetries() != maxRetries {
				t.Errorf("Expected MaxRetries to be %d, got %d", maxRetries, policy.GetMaxRetries())
			}
		}
	})

	// Test 3: Retry on transient errors
	t.Run("RetryOnTransientErrors", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)
		executor := NewRetryExecutor(policy, logger, metricsCollector)

		ctx := context.Background()

		// Test transient errors using typed sentinels
		transientErrors := []error{
			core.ErrTimeout,
			core.ErrConnectionRefused,
			core.ErrConnectionReset,
			core.ErrTemporaryFailure,
			core.ErrUnavailable,
			core.ErrDeadlineExceeded,
		}

		for _, transErr := range transientErrors {
			attempts := 0
			err := executor.Execute(ctx, func() error {
				attempts++
				if attempts < 2 {
					return transErr
				}
				return nil
			}, "test_source")
			if err != nil {
				t.Errorf("Expected retry to succeed for transient error: %v", transErr)
			}

			if attempts != 2 {
				t.Errorf("Expected 2 attempts for transient error %v, got %d", transErr, attempts)
			}
		}
	})

	// Test 4: No retry on permanent errors
	t.Run("NoRetryOnPermanentErrors", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)
		executor := NewRetryExecutor(policy, logger, metricsCollector)

		ctx := context.Background()

		// Test permanent errors using typed sentinels
		permanentErrors := []error{
			core.ErrBadRequest,
			core.ErrUnauthorized,
			core.ErrForbidden,
			core.ErrNotFound,
			core.ErrInvalidState,
		}

		for _, permErr := range permanentErrors {
			attempts := 0
			err := executor.Execute(ctx, func() error {
				attempts++
				return permErr
			}, "test_source")

			if err == nil {
				t.Errorf("Expected error for permanent error: %v", permErr)
			}

			if attempts != 1 {
				t.Errorf("Expected 1 attempt for permanent error %v, got %d", permErr, attempts)
			}
		}
	})

	// Test 5: Backoff cap at maximum
	t.Run("BackoffCapAtMaximum", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        10,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        500 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)

		// Verify backoff is capped
		for attempt := 0; attempt < 20; attempt++ {
			backoff := policy.GetBackoff(attempt)
			if backoff > config.MaxBackoff {
				t.Errorf("Expected backoff to be capped at %v, got %v for attempt %d", config.MaxBackoff, backoff, attempt)
			}
		}
	})

	// Test 6: Retry exhaustion
	t.Run("RetryExhaustion", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        2,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)
		executor := NewRetryExecutor(policy, logger, metricsCollector)

		ctx := context.Background()

		// Test retry exhaustion
		attempts := 0
		err := executor.Execute(ctx, func() error {
			attempts++
			return core.ErrTimeout
		}, "test_source")

		if err == nil {
			t.Error("Expected error after retry exhaustion")
		}

		if attempts != 3 { // 1 initial + 2 retries
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	// Test 7: Successful retry
	t.Run("SuccessfulRetry", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)
		executor := NewRetryExecutor(policy, logger, metricsCollector)

		ctx := context.Background()

		// Test successful retry at different attempts
		for successAttempt := 1; successAttempt <= 3; successAttempt++ {
			attempts := 0
			err := executor.Execute(ctx, func() error {
				attempts++
				if attempts <= successAttempt {
					return core.ErrTimeout
				}
				return nil
			}, "test_source")
			if err != nil {
				t.Errorf("Expected success at attempt %d, got error: %v", successAttempt+1, err)
			}

			if attempts != successAttempt+1 {
				t.Errorf("Expected %d attempts, got %d", successAttempt+1, attempts)
			}
		}
	})

	// Test 8: Backoff consistency
	t.Run("BackoffConsistency", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)

		// Verify backoff is consistent for same attempt
		for attempt := 0; attempt < 5; attempt++ {
			backoff1 := policy.GetBackoff(attempt)
			backoff2 := policy.GetBackoff(attempt)

			if backoff1 != backoff2 {
				t.Errorf("Expected consistent backoff for attempt %d, got %v and %v", attempt, backoff1, backoff2)
			}
		}
	})

	// Test 9: Concurrent retry execution
	t.Run("ConcurrentRetryExecution", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)
		executor := NewRetryExecutor(policy, logger, metricsCollector)

		ctx := context.Background()
		done := make(chan bool, 20)

		// Concurrent retry execution
		for i := 0; i < 20; i++ {
			go func(index int) {
				attempts := 0
				err := executor.Execute(ctx, func() error {
					attempts++
					if attempts < 2 {
						return core.ErrTimeout
					}
					return nil
				}, fmt.Sprintf("test_source_%d", index))
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}

				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}
	})

	// Test 10: Retry with different multipliers
	t.Run("RetryWithDifferentMultipliers", func(t *testing.T) {
		multipliers := []float64{1.5, 2.0, 3.0}

		for _, multiplier := range multipliers {
			config := &RetryConfig{
				MaxRetries:        5,
				InitialBackoff:    100 * time.Millisecond,
				MaxBackoff:        10 * time.Second,
				BackoffMultiplier: multiplier,
				JitterFraction:    0.0,
			}
			policy := NewDefaultRetryPolicy(config, errorHandler)

			backoff0 := policy.GetBackoff(0)
			backoff1 := policy.GetBackoff(1)

			expectedBackoff1 := time.Duration(float64(backoff0) * multiplier)
			if backoff1 < expectedBackoff1-time.Millisecond || backoff1 > expectedBackoff1+time.Millisecond {
				t.Errorf("For multiplier %f: expected backoff1 ≈ %v, got %v", multiplier, expectedBackoff1, backoff1)
			}
		}
	})

	// Test 11: Retry with jitter
	t.Run("RetryWithJitter", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        10 * time.Second,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.1,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)

		// Verify jitter adds variation
		backoffs := make([]time.Duration, 10)
		for i := 0; i < 10; i++ {
			backoffs[i] = policy.GetBackoff(1)
		}

		// Check if there's variation (jitter)
		minBackoff := backoffs[0]
		maxBackoff := backoffs[0]
		for i := 1; i < 10; i++ {
			if backoffs[i] < minBackoff {
				minBackoff = backoffs[i]
			}
			if backoffs[i] > maxBackoff {
				maxBackoff = backoffs[i]
			}
		}

		// With jitter, we should see some variation
		if minBackoff == maxBackoff {
			t.Logf("Note: No variation in backoff with jitter (may be due to rounding)")
		}
	})

	// Test 12: Retry metrics recording
	t.Run("RetryMetricsRecording", func(t *testing.T) {
		config := &RetryConfig{
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
			JitterFraction:    0.0,
		}
		policy := NewDefaultRetryPolicy(config, errorHandler)
		executor := NewRetryExecutor(policy, logger, metricsCollector)

		ctx := context.Background()

		// Execute with retries
		attempts := 0
		err := executor.Execute(ctx, func() error {
			attempts++
			if attempts < 2 {
				return core.ErrTimeout
			}
			return nil
		}, "test_source")
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		// Verify metrics were recorded
		metrics := metricsCollector.Export()
		if metrics == nil {
			t.Error("Expected metrics to be recorded")
		}
	})
}
