package query

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestDefaultRetryPolicy tests the default retry policy
func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts=3, got %d", policy.MaxAttempts)
	}

	if policy.InitialBackoff != 100*time.Millisecond {
		t.Errorf("Expected InitialBackoff=100ms, got %v", policy.InitialBackoff)
	}

	if policy.MaxBackoff != 10*time.Second {
		t.Errorf("Expected MaxBackoff=10s, got %v", policy.MaxBackoff)
	}

	if policy.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier=2.0, got %f", policy.BackoffMultiplier)
	}

	if policy.JitterFraction != 0.1 {
		t.Errorf("Expected JitterFraction=0.1, got %f", policy.JitterFraction)
	}
}

// TestCalculateBackoff tests backoff calculation
func TestCalculateBackoff(t *testing.T) {
	policy := DefaultRetryPolicy()

	testCases := []struct {
		attempt int
		minTime time.Duration
		maxTime time.Duration
	}{
		{1, 90 * time.Millisecond, 110 * time.Millisecond},
		{2, 180 * time.Millisecond, 220 * time.Millisecond},
		{3, 360 * time.Millisecond, 440 * time.Millisecond},
	}

	for _, tc := range testCases {
		backoff := policy.CalculateBackoff(tc.attempt)

		// Allow some tolerance for jitter
		if backoff < tc.minTime || backoff > tc.maxTime {
			t.Errorf("Attempt %d: backoff %v not in range [%v, %v]", tc.attempt, backoff, tc.minTime, tc.maxTime)
		}
	}
}

// TestCalculateBackoffZeroAttempt tests backoff for zero attempt
func TestCalculateBackoffZeroAttempt(t *testing.T) {
	policy := DefaultRetryPolicy()

	backoff := policy.CalculateBackoff(0)
	if backoff != 0 {
		t.Errorf("Expected backoff=0 for attempt 0, got %v", backoff)
	}
}

// TestCalculateBackoffNegativeAttempt tests backoff for negative attempt
func TestCalculateBackoffNegativeAttempt(t *testing.T) {
	policy := DefaultRetryPolicy()

	backoff := policy.CalculateBackoff(-1)
	if backoff != 0 {
		t.Errorf("Expected backoff=0 for negative attempt, got %v", backoff)
	}
}

// TestCalculateBackoffMaxBackoff tests that backoff is capped at max
func TestCalculateBackoffMaxBackoff(t *testing.T) {
	policy := DefaultRetryPolicy()

	// Large attempt number should be capped at MaxBackoff
	backoff := policy.CalculateBackoff(100)
	if backoff > policy.MaxBackoff*2 {
		t.Errorf("Expected backoff <= %v, got %v", policy.MaxBackoff*2, backoff)
	}
}

// TestShouldRetryTransientError tests retry for transient errors
func TestShouldRetryTransientError(t *testing.T) {
	policy := DefaultRetryPolicy()

	err := errors.New("connection refused")
	if !policy.ShouldRetry(err, 1) {
		t.Error("Should retry transient error")
	}
}

// TestShouldRetryPermanentError tests no retry for permanent errors
func TestShouldRetryPermanentError(t *testing.T) {
	policy := DefaultRetryPolicy()

	err := errors.New("unique constraint violation")
	if policy.ShouldRetry(err, 1) {
		t.Error("Should not retry permanent error")
	}
}

// TestShouldRetryMaxAttempts tests no retry when max attempts reached
func TestShouldRetryMaxAttempts(t *testing.T) {
	policy := DefaultRetryPolicy()

	err := errors.New("connection refused")
	if policy.ShouldRetry(err, policy.MaxAttempts) {
		t.Error("Should not retry when max attempts reached")
	}
}

// TestShouldRetryNilError tests no retry for nil error
func TestShouldRetryNilError(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.ShouldRetry(nil, 1) {
		t.Error("Should not retry nil error")
	}
}

// TestRetryHandlerExecuteSuccess tests successful execution
func TestRetryHandlerExecuteSuccess(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return nil
	}

	ctx := context.Background()
	err := handler.Execute(ctx, fn)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

// TestRetryHandlerExecuteFailure tests failure after max attempts
func TestRetryHandlerExecuteFailure(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("connection refused")
	}

	ctx := context.Background()
	err := handler.Execute(ctx, fn)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryHandlerExecuteEventualSuccess tests eventual success after retries
func TestRetryHandlerExecuteEventualSuccess(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("connection refused")
		}
		return nil
	}

	ctx := context.Background()
	err := handler.Execute(ctx, fn)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryHandlerExecutePermanentError tests no retry for permanent error
func TestRetryHandlerExecutePermanentError(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("unique constraint violation")
	}

	ctx := context.Background()
	err := handler.Execute(ctx, fn)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

// TestRetryHandlerExecuteContextCancellation tests context cancellation
func TestRetryHandlerExecuteContextCancellation(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("connection refused")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := handler.Execute(ctx, fn)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if attempts != 0 {
		t.Errorf("Expected 0 attempts, got %d", attempts)
	}
}

// TestRetryHandlerExecuteWithTimeout tests execution with timeout
func TestRetryHandlerExecuteWithTimeout(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return errors.New("connection refused")
	}

	ctx := context.Background()
	err := handler.ExecuteWithTimeout(ctx, 50*time.Millisecond, fn)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should have at least 1 attempt
	if attempts < 1 {
		t.Errorf("Expected at least 1 attempt, got %d", attempts)
	}
}

// TestRetryHandlerExecuteWithStats tests execution with statistics
func TestRetryHandlerExecuteWithStats(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("connection refused")
		}
		return nil
	}

	ctx := context.Background()
	stats, err := handler.ExecuteWithStats(ctx, fn)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if stats.TotalAttempts != 3 {
		t.Errorf("Expected TotalAttempts=3, got %d", stats.TotalAttempts)
	}

	if stats.SuccessfulAttempts != 1 {
		t.Errorf("Expected SuccessfulAttempts=1, got %d", stats.SuccessfulAttempts)
	}

	if stats.FailedAttempts != 2 {
		t.Errorf("Expected FailedAttempts=2, got %d", stats.FailedAttempts)
	}

	if stats.TotalBackoffTime == 0 {
		t.Error("Expected TotalBackoffTime > 0")
	}
}

// TestRetryHandlerExecuteWithStatsFailure tests statistics on failure
func TestRetryHandlerExecuteWithStatsFailure(t *testing.T) {
	handler := NewRetryHandler(DefaultRetryPolicy())

	fn := func(ctx context.Context) error {
		return errors.New("connection refused")
	}

	ctx := context.Background()
	stats, err := handler.ExecuteWithStats(ctx, fn)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if stats.TotalAttempts != 3 {
		t.Errorf("Expected TotalAttempts=3, got %d", stats.TotalAttempts)
	}

	if stats.SuccessfulAttempts != 0 {
		t.Errorf("Expected SuccessfulAttempts=0, got %d", stats.SuccessfulAttempts)
	}

	if stats.FailedAttempts != 3 {
		t.Errorf("Expected FailedAttempts=3, got %d", stats.FailedAttempts)
	}

	if stats.LastError == nil {
		t.Error("Expected LastError to be set")
	}
}

// TestRetryHandlerNilPolicy tests retry handler with nil policy
func TestRetryHandlerNilPolicy(t *testing.T) {
	handler := NewRetryHandler(nil)

	if handler.policy == nil {
		t.Error("Expected policy to be set to default")
	}

	if handler.policy.MaxAttempts != 3 {
		t.Errorf("Expected default MaxAttempts=3, got %d", handler.policy.MaxAttempts)
	}
}

// TestRetryPolicyCustom tests custom retry policy
func TestRetryPolicyCustom(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:       5,
		InitialBackoff:    50 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 1.5,
		JitterFraction:    0.2,
	}

	handler := NewRetryHandler(policy)

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 5 {
			return errors.New("connection refused")
		}
		return nil
	}

	ctx := context.Background()
	err := handler.Execute(ctx, fn)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 5 {
		t.Errorf("Expected 5 attempts, got %d", attempts)
	}
}
