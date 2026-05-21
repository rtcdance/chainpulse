package pullers

import (
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type CircuitBreakerState int

const (
	CircuitBreakerClosed   CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "closed"
	case CircuitBreakerOpen:
		return "open"
	case CircuitBreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for RPC calls.
// It uses Allow() as a gate before making the call, then Success()/Failure()
// to report the outcome. A separate implementation for API routing exists in
// pkg/plugins/api/request_router.go (different API, same pattern).
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitBreakerState
	failureCount      int
	failureThreshold  int
	cooldown          time.Duration
	halfOpenLimit     int
	halfOpenSuccesses int
	probeInFlight     bool
	lastFailureTime   time.Time
}

var DefaultCircuitBreakerConfig = CircuitBreakerConfig{
	FailureThreshold: 5,
	Cooldown:         30 * time.Second,
	HalfOpenLimit:    3,
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
	HalfOpenLimit    int
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultCircuitBreakerConfig.FailureThreshold
	}
	if config.Cooldown <= 0 {
		config.Cooldown = DefaultCircuitBreakerConfig.Cooldown
	}
	if config.HalfOpenLimit <= 0 {
		config.HalfOpenLimit = DefaultCircuitBreakerConfig.HalfOpenLimit
	}
	return &CircuitBreaker{
		state:            CircuitBreakerClosed,
		failureThreshold: config.FailureThreshold,
		cooldown:         config.Cooldown,
		halfOpenLimit:    config.HalfOpenLimit,
	}
}

func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return nil
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailureTime) > cb.cooldown {
			cb.state = CircuitBreakerHalfOpen
			cb.halfOpenSuccesses = 0
			cb.probeInFlight = true
			return nil
		}
		return core.ErrRPCUnreachable
	case CircuitBreakerHalfOpen:
		if cb.probeInFlight {
			return core.ErrRPCUnreachable
		}
		cb.probeInFlight = true
		return nil
	default:
		return core.ErrRPCUnreachable
	}
}

func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerHalfOpen:
		cb.probeInFlight = false
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.halfOpenLimit {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
			cb.halfOpenSuccesses = 0
		}
	case CircuitBreakerClosed:
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitBreakerOpen
			cb.probeInFlight = false
		}
	case CircuitBreakerHalfOpen:
		cb.state = CircuitBreakerOpen
		cb.halfOpenSuccesses = 0
		cb.probeInFlight = false
	}
}

func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Counts() (current, threshold int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount, cb.failureThreshold
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitBreakerClosed
	cb.failureCount = 0
	cb.halfOpenSuccesses = 0
	cb.probeInFlight = false
}
