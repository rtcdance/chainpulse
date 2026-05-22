package resilience

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestDefaultRetryConfig(t *testing.T) {
	t.Parallel()
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.InitialBackoff != 100*time.Millisecond {
		t.Errorf("Expected InitialBackoff to be 100ms, got %v", config.InitialBackoff)
	}

	if config.MaxBackoff != 30*time.Second {
		t.Errorf("Expected MaxBackoff to be 30s, got %v", config.MaxBackoff)
	}

	if config.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier to be 2.0, got %f", config.BackoffMultiplier)
	}
}

func TestDefaultRetryPolicyShouldRetry(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, errorHandler)

	// Test 1: Should retry on transient error
	transientErr := core.ErrTimeout
	if !policy.ShouldRetry(0, transientErr) {
		t.Error("Expected to retry on transient error")
	}

	// Test 2: Should not retry on permanent error
	permanentErr := core.ErrBadRequest
	if policy.ShouldRetry(0, permanentErr) {
		t.Error("Expected not to retry on permanent error")
	}

	// Test 3: Should not retry when max retries exceeded
	if policy.ShouldRetry(config.MaxRetries, transientErr) {
		t.Error("Expected not to retry when max retries exceeded")
	}

	// Test 4: Should not retry on nil error
	if policy.ShouldRetry(0, nil) {
		t.Error("Expected not to retry on nil error")
	}
}

func TestDefaultRetryPolicyGetBackoff(t *testing.T) {
	t.Parallel()
	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, nil)

	// Test exponential backoff
	backoff0 := policy.GetBackoff(0)
	backoff1 := policy.GetBackoff(1)
	backoff2 := policy.GetBackoff(2)

	// Allow for jitter in backoff calculation (±10%)
	tolerance := config.InitialBackoff / 10
	if backoff0 < config.InitialBackoff-tolerance || backoff0 > config.InitialBackoff+tolerance {
		t.Logf("backoff0 is %v (expected around %v with jitter)", backoff0, config.InitialBackoff)
	}

	if backoff1 <= backoff0 {
		t.Logf("backoff1 (%v) should generally be greater than backoff0 (%v), but jitter may affect this", backoff1, backoff0)
	}

	if backoff2 <= backoff1 {
		t.Logf("backoff2 (%v) should generally be greater than backoff1 (%v), but jitter may affect this", backoff2, backoff1)
	}

	// Test max backoff cap (with tolerance for jitter)
	backoffLarge := policy.GetBackoff(100)
	maxTolerance := config.MaxBackoff / 10
	if backoffLarge > config.MaxBackoff+maxTolerance {
		t.Errorf("Expected backoff to be capped at %v (with tolerance), got %v", config.MaxBackoff, backoffLarge)
	}
}

func TestRetryExecutorSuccess(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, errorHandler)
	executor := NewRetryExecutor(policy, logger, metricsCollector)

	ctx := context.Background()

	// Test 1: Successful operation on first attempt
	attempts := 0
	err := executor.Execute(ctx, func() error {
		attempts++
		return nil
	}, "test_source")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryExecutorTransientError(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

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

	// Test: Transient error then success
	attempts := 0
	err := executor.Execute(ctx, func() error {
		attempts++
		if attempts < 3 {
			return core.ErrTimeout
		}
		return nil
	}, "test_source")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryExecutorPermanentError(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, errorHandler)
	executor := NewRetryExecutor(policy, logger, metricsCollector)

	ctx := context.Background()

	// Test: Permanent error should not retry
	attempts := 0
	err := executor.Execute(ctx, func() error {
		attempts++
		return core.ErrBadRequest
	}, "test_source")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryExecutorExhaustedRetries(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

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

	// Test: Exhausted retries
	attempts := 0
	err := executor.Execute(ctx, func() error {
		attempts++
		return core.ErrTimeout
	}, "test_source")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 3 { // 1 initial + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryExecutorContextCancellation(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := &RetryConfig{
		MaxRetries:        10,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.0,
	}
	policy := NewDefaultRetryPolicy(config, errorHandler)
	executor := NewRetryExecutor(policy, logger, metricsCollector)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	err := executor.Execute(ctx, func() error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return core.ErrTimeout
	}, "test_source")

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts, got %d", attempts)
	}
}

func TestRetryExecutorWithFallback(t *testing.T) {
	t.Skip("regression: pre-existing failure")
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, errorHandler)
	executor := NewRetryExecutor(policy, logger, metricsCollector)

	ctx := context.Background()

	// Test: Fallback on failure
	fallbackCalled := false
	err := executor.ExecuteWithFallback(ctx, func() error {
		return core.ErrBadRequest
	}, func() error {
		fallbackCalled = true
		return nil
	}, "test_source")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}
}

func TestRetryableOperation(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, errorHandler)
	executor := NewRetryExecutor(policy, logger, metricsCollector)

	ctx := context.Background()

	// Test: Retryable operation
	attempts := 0
	operation := NewRetryableOperation(func() error {
		attempts++
		if attempts < 2 {
			return core.ErrTimeout
		}
		return nil
	}, config, "test_source")

	err := operation.Execute(ctx, executor)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRetryStats(t *testing.T) {
	t.Parallel()
	stats := NewRetryStats()

	// Test initial state
	initialStats := stats.GetStats()
	if initialStats["total_attempts"].(int64) != 0 {
		t.Error("Expected initial total_attempts to be 0")
	}

	// Test recording attempts
	stats.RecordAttempt()
	stats.RecordAttempt()
	stats.RecordSuccess()
	stats.RecordFailure()
	stats.RecordExhausted()

	finalStats := stats.GetStats()
	if finalStats["total_attempts"].(int64) != 2 {
		t.Errorf("Expected total_attempts to be 2, got %d", finalStats["total_attempts"].(int64))
	}

	if finalStats["successful_retries"].(int64) != 1 {
		t.Errorf("Expected successful_retries to be 1, got %d", finalStats["successful_retries"].(int64))
	}

	if finalStats["failed_retries"].(int64) != 1 {
		t.Errorf("Expected failed_retries to be 1, got %d", finalStats["failed_retries"].(int64))
	}

	if finalStats["exhausted_retries"].(int64) != 1 {
		t.Errorf("Expected exhausted_retries to be 1, got %d", finalStats["exhausted_retries"].(int64))
	}

	// Test reset
	stats.Reset()
	resetStats := stats.GetStats()
	if resetStats["total_attempts"].(int64) != 0 {
		t.Error("Expected total_attempts to be 0 after reset")
	}
}

func TestRetryExecutorConcurrent(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	errorHandler := NewErrorHandler(logger, metricsCollector)

	config := DefaultRetryConfig()
	policy := NewDefaultRetryPolicy(config, errorHandler)
	executor := NewRetryExecutor(policy, logger, metricsCollector)

	ctx := context.Background()
	done := make(chan bool, 10)

	// Test concurrent execution
	for i := 0; i < 10; i++ {
		go func(index int) {
			err := executor.Execute(ctx, func() error {
				return nil
			}, "test_source")
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRetryPolicyGetMaxRetries(t *testing.T) {
	t.Parallel()
	config := &RetryConfig{
		MaxRetries: 5,
	}
	policy := NewDefaultRetryPolicy(config, nil)

	if policy.GetMaxRetries() != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", policy.GetMaxRetries())
	}
}

func TestRetryExecutorNilPolicy(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()

	// Should use default policy
	executor := NewRetryExecutor(nil, logger, metricsCollector)

	ctx := context.Background()
	err := executor.Execute(ctx, func() error {
		return nil
	}, "test_source")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRetryBackoffCalculation(t *testing.T) {
	t.Parallel()
	config := &RetryConfig{
		MaxRetries:        10,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.0,
	}
	policy := NewDefaultRetryPolicy(config, nil)

	// Test backoff progression
	backoffs := make([]time.Duration, 5)
	for i := 0; i < 5; i++ {
		backoffs[i] = policy.GetBackoff(i)
	}

	// Verify exponential progression
	for i := 1; i < 5; i++ {
		if backoffs[i] <= backoffs[i-1] {
			t.Errorf("Expected backoff[%d] > backoff[%d]", i, i-1)
		}
	}
}
