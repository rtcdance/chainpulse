package query

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestCircuitBreakerInitialState tests initial state is closed
func TestCircuitBreakerInitialState(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be closed, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerSuccessfulCall tests successful call
func TestCircuitBreakerSuccessfulCall(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain closed, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerFailedCall tests failed call
func TestCircuitBreakerFailedCall(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := cb.Call(func() error {
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain closed, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerTransitionToOpen tests transition to open state
func TestCircuitBreakerTransitionToOpen(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail 3 times to trigger open state
	for i := 0; i < 3; i++ {
		_ = cb.Call(func() error {
			return errors.New("test error")
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerRejectCallsWhenOpen tests that calls are rejected when open
func TestCircuitBreakerRejectCallsWhenOpen(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail once to trigger open state
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	// Next call should be rejected
	err := cb.Call(func() error {
		return nil
	})

	if err == nil {
		t.Error("Expected error when circuit is open, got nil")
	}
}

// TestCircuitBreakerTransitionToHalfOpen tests transition to half-open state
func TestCircuitBreakerTransitionToHalfOpen(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail once to trigger open state
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.GetState().String())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next call should transition to half-open
	_ = cb.Call(func() error {
		return nil
	})

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be half-open, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerTransitionToClosed tests transition from half-open to closed
func TestCircuitBreakerTransitionToClosed(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail once to trigger open state
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Succeed twice to transition to closed
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error {
			return nil
		})
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be closed, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerTransitionBackToOpen tests transition from half-open back to open
func TestCircuitBreakerTransitionBackToOpen(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail once to trigger open state
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Transition to half-open
	_ = cb.Call(func() error {
		return nil
	})

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be half-open, got %s", cb.GetState().String())
	}

	// Fail in half-open state to transition back to open
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerReset tests reset functionality
func TestCircuitBreakerReset(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail once to trigger open state
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be open, got %s", cb.GetState().String())
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be closed after reset, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerGetStats tests statistics
func TestCircuitBreakerGetStats(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail twice
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error {
			return errors.New("test error")
		})
	}

	stats := cb.GetStats()

	if stats.State != StateClosed {
		t.Errorf("Expected state to be closed, got %s", stats.State.String())
	}

	if stats.FailureCount != 2 {
		t.Errorf("Expected FailureCount=2, got %d", stats.FailureCount)
	}

	if stats.SuccessCount != 0 {
		t.Errorf("Expected SuccessCount=0, got %d", stats.SuccessCount)
	}

	if stats.TimeSinceLastFailure == 0 {
		t.Error("Expected TimeSinceLastFailure > 0")
	}
}

// TestCircuitBreakerStateChangeHook tests state change hook
func TestCircuitBreakerStateChangeHook(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	stateChanges := []struct {
		oldState CircuitBreakerState
		newState CircuitBreakerState
	}{}

	cb.SetStateChangeHook(func(oldState, newState CircuitBreakerState) {
		stateChanges = append(stateChanges, struct {
			oldState CircuitBreakerState
			newState CircuitBreakerState
		}{oldState, newState})
	})

	// Fail once to trigger open state
	_ = cb.Call(func() error {
		return errors.New("test error")
	})

	if len(stateChanges) != 1 {
		t.Errorf("Expected 1 state change, got %d", len(stateChanges))
	}

	if stateChanges[0].oldState != StateClosed || stateChanges[0].newState != StateOpen {
		t.Errorf("Expected state change from closed to open, got %s to %s", stateChanges[0].oldState.String(), stateChanges[0].newState.String())
	}
}

// TestCircuitBreakerNilConfig tests circuit breaker with nil config
func TestCircuitBreakerNilConfig(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(nil)

	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be closed, got %s", cb.GetState().String())
	}

	if cb.config.FailureThreshold != 5 {
		t.Errorf("Expected default FailureThreshold=5, got %d", cb.config.FailureThreshold)
	}
}

// TestCircuitBreakerPoolGetOrCreate tests pool get or create
func TestCircuitBreakerPoolGetOrCreate(t *testing.T) {
	t.Parallel()
	pool := NewCircuitBreakerPool()

	cb1 := pool.GetOrCreate("test", DefaultCircuitBreakerConfig())
	cb2 := pool.GetOrCreate("test", DefaultCircuitBreakerConfig())

	if cb1 != cb2 {
		t.Error("Expected same circuit breaker instance")
	}
}

// TestCircuitBreakerPoolGet tests pool get
func TestCircuitBreakerPoolGet(t *testing.T) {
	t.Parallel()
	pool := NewCircuitBreakerPool()

	cb1 := pool.GetOrCreate("test", DefaultCircuitBreakerConfig())
	cb2 := pool.Get("test")

	if cb1 != cb2 {
		t.Error("Expected same circuit breaker instance")
	}

	cb3 := pool.Get("nonexistent")
	if cb3 != nil {
		t.Error("Expected nil for nonexistent circuit breaker")
	}
}

// TestCircuitBreakerPoolResetAll tests pool reset all
func TestCircuitBreakerPoolResetAll(t *testing.T) {
	t.Parallel()
	pool := NewCircuitBreakerPool()

	cb1 := pool.GetOrCreate("test1", DefaultCircuitBreakerConfig())
	cb2 := pool.GetOrCreate("test2", DefaultCircuitBreakerConfig())

	// Fail both to trigger open state
	if err := cb1.Call(func() error {
		return errors.New("test error")
	}); err != nil {
		_ = err // Expected to fail
	}
	if err := cb2.Call(func() error {
		return errors.New("test error")
	}); err != nil {
		_ = err // Expected to fail
	}

	// Reset all
	pool.ResetAll()

	if cb1.GetState() != StateClosed {
		t.Errorf("Expected cb1 state to be closed, got %s", cb1.GetState().String())
	}

	if cb2.GetState() != StateClosed {
		t.Errorf("Expected cb2 state to be closed, got %s", cb2.GetState().String())
	}
}

// TestCircuitBreakerPoolGetStats tests pool get stats
func TestCircuitBreakerPoolGetStats(t *testing.T) {
	t.Parallel()
	pool := NewCircuitBreakerPool()

	cb1 := pool.GetOrCreate("test1", DefaultCircuitBreakerConfig())
	_ = pool.GetOrCreate("test2", DefaultCircuitBreakerConfig())

	// Fail cb1
	if err := cb1.Call(func() error {
		return errors.New("test error")
	}); err != nil {
		_ = err // Expected to fail
	}

	stats := pool.GetStats()

	if len(stats) != 2 {
		t.Errorf("Expected 2 circuit breakers in stats, got %d", len(stats))
	}

	if stats["test1"].FailureCount != 1 {
		t.Errorf("Expected test1 FailureCount=1, got %d", stats["test1"].FailureCount)
	}

	if stats["test2"].FailureCount != 0 {
		t.Errorf("Expected test2 FailureCount=0, got %d", stats["test2"].FailureCount)
	}
}

// TestCircuitBreakerStateString tests state string representation
func TestCircuitBreakerStateString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitBreakerState(999), "unknown"},
	}

	for _, tc := range testCases {
		result := tc.state.String()
		if result != tc.expected {
			t.Errorf("Expected %q, got %q", tc.expected, result)
		}
	}
}

// TestCircuitBreakerCallWithContextCancelled tests that a cancelled context
// is rejected without counting as a failure
func TestCircuitBreakerCallWithContextCancelled(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := cb.CallWithContext(ctx, func() error {
		t.Fatal("function should not be called with cancelled context")
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Should NOT count as a failure
	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("Expected 0 failures with cancelled context, got %d", stats.FailureCount)
	}
}

// TestCircuitBreakerCallWithContextSuccess tests normal operation with context
func TestCircuitBreakerCallWithContextSuccess(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := cb.CallWithContext(context.Background(), func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	stats := cb.GetStats()
	if stats.SuccessCount != 0 {
		t.Errorf("Expected SuccessCount=0 (reset on success in closed state), got %d", stats.SuccessCount)
	}
}

// TestCircuitBreakerCallWithContextFailure tests failure recording with context
func TestCircuitBreakerCallWithContextFailure(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	err := cb.CallWithContext(context.Background(), func() error {
		return errors.New("operation failed")
	})

	if err == nil {
		t.Fatal("Expected error from failed operation")
	}

	stats := cb.GetStats()
	if stats.FailureCount != 1 {
		t.Errorf("Expected FailureCount=1, got %d", stats.FailureCount)
	}
}

// TestCircuitBreakerCallWithContextCancellationDuringExecution tests that
// context cancellation during fn execution doesn't count as circuit failure
func TestCircuitBreakerCallWithContextCancellationDuringExecution(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	ctx, cancel := context.WithCancel(context.Background())

	err := cb.CallWithContext(ctx, func() error {
		cancel() // Cancel during execution
		return errors.New("operation failed")
	})

	// Since ctx was cancelled before fn returned, the circuit breaker should
	// return the context error, not count it as a circuit failure
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("Expected 0 failures (context cancellation shouldn't count), got %d", stats.FailureCount)
	}
}

// TestCircuitBreakerCallWithContextTimeout tests context deadline exceeded
func TestCircuitBreakerCallWithContextTimeout(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Let the deadline pass
	time.Sleep(10 * time.Millisecond)

	err := cb.CallWithContext(ctx, func() error {
		return nil
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Should NOT count as a failure
	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("Expected 0 failures with expired context, got %d", stats.FailureCount)
	}
}

// TestCircuitBreakerCallDelegatesToCallWithContext tests backward compatibility
func TestCircuitBreakerCallDelegatesToCallWithContext(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	err := cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error from Call(), got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenProbeLimit(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold:   1,
		SuccessThreshold:   2,
		Timeout:            50 * time.Millisecond,
		HalfOpenProbeLimit: 1, // Only 1 probe at a time
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Call(func() error { return fmt.Errorf("fail") })

	// Wait for half-open transition
	time.Sleep(100 * time.Millisecond)

	// First call should be allowed (takes the probe slot)
	started := make(chan struct{})
	blockDone := make(chan error, 1)

	go func() {
		close(started)
		err := cb.CallWithContext(context.Background(), func() error {
			time.Sleep(200 * time.Millisecond) // Block to hold the probe slot
			return nil
		})
		blockDone <- err
	}()

	// Wait for the goroutine to start
	<-started
	time.Sleep(20 * time.Millisecond) // Give it time to enter the probe

	// Second call should be rejected (probe limit reached)
	err := cb.Call(func() error { return nil })
	if err == nil {
		t.Error("expected error for second probe in half-open state")
	}
	if err.Error() != "circuit breaker is half-open, probe limit reached" {
		t.Errorf("unexpected error: %v", err)
	}

	// Wait for the first probe to complete
	<-blockDone

	// After the probe succeeds and circuit closes, calls should work
	time.Sleep(10 * time.Millisecond)
	if cb.GetState() != StateClosed {
		// The first probe succeeded, but SuccessThreshold=2, so might still be half-open
		// Make one more successful call
		err = cb.Call(func() error { return nil })
		if err != nil {
			t.Logf("post-probe call error (might still be half-open): %v", err)
		}
	}
}

func TestCircuitBreaker_HalfOpenProbeLimitMultiple(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold:   1,
		SuccessThreshold:   3,
		Timeout:            50 * time.Millisecond,
		HalfOpenProbeLimit: 3, // Allow up to 3 concurrent probes
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Call(func() error { return fmt.Errorf("fail") })

	// Wait for half-open transition
	time.Sleep(100 * time.Millisecond)

	// Start 3 concurrent probes
	var wg sync.WaitGroup
	results := make([]error, 4)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = cb.CallWithContext(context.Background(), func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			})
		}(i)
	}

	time.Sleep(20 * time.Millisecond) // Let goroutines start

	// 4th call should be rejected
	results[3] = cb.Call(func() error { return nil })

	wg.Wait()

	// First 3 should succeed
	for i := 0; i < 3; i++ {
		if results[i] != nil {
			t.Errorf("probe %d should have succeeded, got: %v", i, results[i])
		}
	}

	// 4th should have been rejected
	if results[3] == nil {
		t.Error("4th probe should have been rejected (limit=3)")
	}
}

func TestCircuitBreaker_HalfOpenProbeCounterResetOnClose(t *testing.T) {
	t.Parallel()
	config := &CircuitBreakerConfig{
		FailureThreshold:   1,
		SuccessThreshold:   1,
		Timeout:            50 * time.Millisecond,
		HalfOpenProbeLimit: 1,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.Call(func() error { return fmt.Errorf("fail") })

	// Wait for half-open
	time.Sleep(100 * time.Millisecond)

	// Successful probe should close the circuit
	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("probe call failed: %v", err)
	}

	// Circuit should now be closed
	if cb.GetState() != StateClosed {
		t.Errorf("state = %v, want Closed", cb.GetState())
	}

	// Probe counter should be reset
	if atomic.LoadInt32(&cb.halfOpenProbes) != 0 {
		t.Errorf("halfOpenProbes = %d, want 0 after closing", cb.halfOpenProbes)
	}
}
