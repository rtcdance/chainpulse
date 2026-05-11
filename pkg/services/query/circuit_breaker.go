package query

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	// StateClosed represents normal operation
	StateClosed CircuitBreakerState = iota
	// StateOpen represents failure state, rejecting requests
	StateOpen
	// StateHalfOpen represents testing state, allowing limited requests
	StateHalfOpen
)

// String returns the string representation of CircuitBreakerState
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig defines the circuit breaker configuration
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	// SuccessThreshold is the number of successes before closing the circuit from half-open
	SuccessThreshold int
	// Timeout is the duration to wait before transitioning from open to half-open
	Timeout time.Duration
	// HalfOpenProbeLimit is the maximum number of concurrent requests allowed in half-open state.
	// Default is 1 (only one probe request at a time).
	HalfOpenProbeLimit int32
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:   5,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config          *CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time
	mu              sync.RWMutex
	stateChangeHook func(oldState, newState CircuitBreakerState)
	halfOpenProbes  int32 // atomic counter for concurrent probes in half-open state
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state
}

// Call executes the function if the circuit breaker allows it.
// Deprecated: Use CallWithContext(ctx, fn) instead to propagate cancellation.
func (cb *CircuitBreaker) Call(fn func() error) error {
	return cb.CallWithContext(context.Background(), fn)
}

// CallWithContext executes the function if the circuit breaker allows it,
// respecting context cancellation. If the context is cancelled before the
// function is executed, it returns the context error immediately without
// recording a failure against the circuit breaker.
func (cb *CircuitBreaker) CallWithContext(ctx context.Context, fn func() error) error {
	// Check context before acquiring the lock
	if err := ctx.Err(); err != nil {
		return err
	}

	cb.mu.Lock()

	// Re-check context after acquiring the lock
	if err := ctx.Err(); err != nil {
		cb.mu.Unlock()
		return err
	}

	// Check if we need to transition from open to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.transitionToHalfOpen()
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// In half-open state, limit concurrent probe requests
	if cb.state == StateHalfOpen {
		probeLimit := cb.config.HalfOpenProbeLimit
		if probeLimit <= 0 {
			probeLimit = 1
		}
		if atomic.LoadInt32(&cb.halfOpenProbes) >= probeLimit {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is half-open, probe limit reached")
		}
		atomic.AddInt32(&cb.halfOpenProbes, 1)
		cb.mu.Unlock()

		// Execute fn() outside the lock, ensuring probe counter is always decremented
		var err error
		func() {
			defer atomic.AddInt32(&cb.halfOpenProbes, -1)
			err = fn()
		}()

		// Record the result under the lock
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			cb.recordFailure()
		} else {
			cb.recordSuccess()
		}

		return err
	}

	// Closed state: execute normally outside the lock
	cb.mu.Unlock()

	err := fn()

	// Record the result under the lock
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// recordFailure records a failure and updates the state
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		// Transition back to open on failure in half-open state
		cb.transitionToOpen()
	} else if cb.state == StateClosed && cb.failureCount >= cb.config.FailureThreshold {
		// Transition to open when failure threshold is reached
		cb.transitionToOpen()
	}
}

// recordSuccess records a success and updates the state
func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			// Transition to closed when success threshold is reached
			cb.transitionToClosed()
		}
	}
}

// transitionToClosed transitions the circuit breaker to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()
	atomic.StoreInt32(&cb.halfOpenProbes, 0)

	if cb.stateChangeHook != nil {
		cb.stateChangeHook(oldState, StateClosed)
	}
}

// transitionToOpen transitions the circuit breaker to open state
func (cb *CircuitBreaker) transitionToOpen() {
	oldState := cb.state
	cb.state = StateOpen
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if cb.stateChangeHook != nil {
		cb.stateChangeHook(oldState, StateOpen)
	}
}

// transitionToHalfOpen transitions the circuit breaker to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	oldState := cb.state
	cb.state = StateHalfOpen
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if cb.stateChangeHook != nil {
		cb.stateChangeHook(oldState, StateHalfOpen)
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if cb.stateChangeHook != nil {
		cb.stateChangeHook(oldState, StateClosed)
	}
}

// GetStats returns the current statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:                cb.state,
		FailureCount:         cb.failureCount,
		SuccessCount:         cb.successCount,
		LastFailureTime:      cb.lastFailureTime,
		LastStateChange:      cb.lastStateChange,
		TimeSinceLastFailure: time.Since(cb.lastFailureTime),
		TimeSinceStateChange: time.Since(cb.lastStateChange),
	}
}

// SetStateChangeHook sets a hook to be called when the state changes
func (cb *CircuitBreaker) SetStateChangeHook(hook func(oldState, newState CircuitBreakerState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.stateChangeHook = hook
}

// CircuitBreakerStats represents the statistics of the circuit breaker
type CircuitBreakerStats struct {
	// State is the current state
	State CircuitBreakerState
	// FailureCount is the current failure count
	FailureCount int
	// SuccessCount is the current success count
	SuccessCount int
	// LastFailureTime is the time of the last failure
	LastFailureTime time.Time
	// LastStateChange is the time of the last state change
	LastStateChange time.Time
	// TimeSinceLastFailure is the duration since the last failure
	TimeSinceLastFailure time.Duration
	// TimeSinceStateChange is the duration since the last state change
	TimeSinceStateChange time.Duration
}

// CircuitBreakerPool manages multiple circuit breakers
type CircuitBreakerPool struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewCircuitBreakerPool creates a new circuit breaker pool
func NewCircuitBreakerPool() *CircuitBreakerPool {
	return &CircuitBreakerPool{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets or creates a circuit breaker for the given name
func (cbp *CircuitBreakerPool) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	cbp.mu.Lock()
	defer cbp.mu.Unlock()

	if cb, exists := cbp.breakers[name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(config)
	cbp.breakers[name] = cb
	return cb
}

// Get gets a circuit breaker by name
func (cbp *CircuitBreakerPool) Get(name string) *CircuitBreaker {
	cbp.mu.RLock()
	defer cbp.mu.RUnlock()

	return cbp.breakers[name]
}

// ResetAll resets all circuit breakers
func (cbp *CircuitBreakerPool) ResetAll() {
	cbp.mu.RLock()
	defer cbp.mu.RUnlock()

	for _, cb := range cbp.breakers {
		cb.Reset()
	}
}

// GetStats returns statistics for all circuit breakers
func (cbp *CircuitBreakerPool) GetStats() map[string]CircuitBreakerStats {
	cbp.mu.RLock()
	defer cbp.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for name, cb := range cbp.breakers {
		stats[name] = cb.GetStats()
	}
	return stats
}
