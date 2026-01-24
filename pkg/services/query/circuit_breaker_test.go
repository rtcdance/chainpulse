package query

import (
	"errors"
	"testing"
	"time"
)

// TestCircuitBreakerInitialState tests initial state is closed
func TestCircuitBreakerInitialState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be closed, got %s", cb.GetState().String())
	}
}

// TestCircuitBreakerSuccessfulCall tests successful call
func TestCircuitBreakerSuccessfulCall(t *testing.T) {
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
	pool := NewCircuitBreakerPool()

	cb1 := pool.GetOrCreate("test", DefaultCircuitBreakerConfig())
	cb2 := pool.GetOrCreate("test", DefaultCircuitBreakerConfig())

	if cb1 != cb2 {
		t.Error("Expected same circuit breaker instance")
	}
}

// TestCircuitBreakerPoolGet tests pool get
func TestCircuitBreakerPoolGet(t *testing.T) {
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
