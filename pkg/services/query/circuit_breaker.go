package query

import (
	"fmt"
	"sync"
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
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config           *CircuitBreakerConfig
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	lastStateChange  time.Time
	mu               sync.RWMutex
	stateChangeHook  func(oldState, newState CircuitBreakerState)
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

// Call executes the function if the circuit breaker allows it
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we need to transition from open to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.transitionToHalfOpen()
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute the function
	err := fn()

	// Handle the result
	if err != nil {
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
		State:              cb.state,
		FailureCount:       cb.failureCount,
		SuccessCount:       cb.successCount,
		LastFailureTime:    cb.lastFailureTime,
		LastStateChange:    cb.lastStateChange,
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
