package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContext returns a context with timeout for tests
func TestContext(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// TestContextWithDeadline returns a context with deadline for tests
func TestContextWithDeadline(t *testing.T, deadline time.Time) context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	t.Cleanup(cancel)
	return ctx
}

// MustNoError asserts that error is nil
func MustNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	require.NoError(t, err, msgAndArgs...)
}

// MustError asserts that error is not nil
func MustError(t *testing.T, err error, msgAndArgs ...interface{}) {
	require.Error(t, err, msgAndArgs...)
}

// MustEqual asserts that two values are equal
func MustEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	require.Equal(t, expected, actual, msgAndArgs...)
}

// AssertInDelta asserts that two float64 values are within delta
func AssertInDelta(t *testing.T, expected, actual, delta float64, msgAndArgs ...interface{}) {
	assert.InDelta(t, expected, actual, delta, msgAndArgs...)
}

// Eventually retries until condition is met or timeout
func Eventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	t.Logf("condition not met within %v", timeout)
	return false
}

// Retry executes fn until it succeeds or max attempts reached
func Retry(t *testing.T, fn func() error, maxAttempts int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < maxAttempts-1 {
			time.Sleep(interval)
		}
	}
	return lastErr
}

// ParallelTest marks test as parallel and runs fn
func ParallelTest(t *testing.T, fn func(*testing.T)) {
	t.Parallel()
	fn(t)
}

// Subtest creates a subtest with common setup/teardown
func Subtest(t *testing.T, name string, fn func(*testing.T)) {
	t.Run(name, func(t *testing.T) {
		// 可以在这里添加通用的 setup/teardown
		fn(t)
	})
}
