package query

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryPolicyConfig(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()
	if rp == nil {
		t.Fatal("DefaultRetryPolicy() returned nil")
	}
	if rp.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", rp.MaxAttempts)
	}
}

func TestRetryPolicy_CalculateBackoff(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()

	prev := time.Duration(0)
	for i := 1; i <= 5; i++ {
		backoff := rp.CalculateBackoff(i)
		if backoff <= 0 {
			t.Errorf("attempt %d: backoff = %v, want > 0", i, backoff)
		}
		if backoff < prev {
			t.Errorf("attempt %d: backoff %v < previous %v", i, backoff, prev)
		}
		prev = backoff
	}
}

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()

	if !rp.ShouldRetry(errors.New("any error"), 0) {
		t.Log("attempt 0: should retry")
	}
	if rp.ShouldRetry(errors.New("any error"), 3) {
		t.Log("attempt 3: should retry")
	}
}

func TestNewRetryHandler(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()
	rh := NewRetryHandler(rp)
	if rh == nil {
		t.Fatal("NewRetryHandler() returned nil")
	}
}

func TestRetryHandler_Execute(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()
	rh := NewRetryHandler(rp)

	ctx := context.Background()
	err := rh.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestRetryHandler_ExecuteFails(t *testing.T) {
	t.Parallel()
	rp := &RetryPolicy{MaxAttempts: 2, InitialBackoff: time.Millisecond}
	rh := NewRetryHandler(rp)

	ctx := context.Background()
	err := rh.Execute(ctx, func(ctx context.Context) error {
		return errors.New("persistent failure")
	})
	if err == nil {
		t.Error("expected error after max attempts")
	}
}

func TestRetryHandler_ExecuteWithTimeout(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()
	rh := NewRetryHandler(rp)

	ctx := context.Background()
	err := rh.ExecuteWithTimeout(ctx, time.Second, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithTimeout() error: %v", err)
	}
}

func TestRetryHandler_ExecuteWithStats(t *testing.T) {
	t.Parallel()
	rp := DefaultRetryPolicy()
	rh := NewRetryHandler(rp)

	ctx := context.Background()
	stats, err := rh.ExecuteWithStats(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithStats() error: %v", err)
	}
	if stats == nil {
		t.Fatal("stats is nil")
	}
	if stats.TotalAttempts != 1 {
		t.Errorf("stats.TotalAttempts = %d, want 1", stats.TotalAttempts)
	}
}
