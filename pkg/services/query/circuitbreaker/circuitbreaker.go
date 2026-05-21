// Package circuitbreaker implements the circuit breaker pattern for query service resilience.
package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
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

type Config struct {
	FailureThreshold   int
	SuccessThreshold   int
	Timeout            time.Duration
	HalfOpenProbeLimit int32
}

func DefaultConfig() *Config {
	return &Config{
		FailureThreshold:   5,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
}

type CircuitBreaker struct {
	config          *Config
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time
	mu              sync.RWMutex
	stateChangeHook func(oldState, newState State)
	halfOpenProbes  int32
}

func New(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}
	return &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	return cb.CallWithContext(context.Background(), fn)
}

func (cb *CircuitBreaker) CallWithContext(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cb.mu.Lock()
	if err := ctx.Err(); err != nil {
		cb.mu.Unlock()
		return err
	}
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.transitionToHalfOpen()
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}
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
		var err error
		func() {
			defer atomic.AddInt32(&cb.halfOpenProbes, -1)
			err = fn()
		}()
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
	cb.mu.Unlock()
	err := fn()
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

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	if cb.state == StateHalfOpen {
		cb.transitionToOpen()
	} else if cb.state == StateClosed && cb.failureCount >= cb.config.FailureThreshold {
		cb.transitionToOpen()
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0
	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.transitionToClosed()
		}
	}
}

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

func (cb *CircuitBreaker) transitionToOpen() {
	oldState := cb.state
	cb.state = StateOpen
	cb.successCount = 0
	cb.lastStateChange = time.Now()
	if cb.stateChangeHook != nil {
		cb.stateChangeHook(oldState, StateOpen)
	}
}

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

type Stats struct {
	State                State
	FailureCount         int
	SuccessCount         int
	LastFailureTime      time.Time
	LastStateChange      time.Time
	TimeSinceLastFailure time.Duration
	TimeSinceStateChange time.Duration
}

func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Stats{
		State:                cb.state,
		FailureCount:         cb.failureCount,
		SuccessCount:         cb.successCount,
		LastFailureTime:      cb.lastFailureTime,
		LastStateChange:      cb.lastStateChange,
		TimeSinceLastFailure: time.Since(cb.lastFailureTime),
		TimeSinceStateChange: time.Since(cb.lastStateChange),
	}
}

func (cb *CircuitBreaker) SetStateChangeHook(hook func(oldState, newState State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.stateChangeHook = hook
}

type Pool struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

func NewPool() *Pool {
	return &Pool{breakers: make(map[string]*CircuitBreaker)}
}

func (cbp *Pool) GetOrCreate(name string, config *Config) *CircuitBreaker {
	cbp.mu.Lock()
	defer cbp.mu.Unlock()
	if cb, exists := cbp.breakers[name]; exists {
		return cb
	}
	cb := New(config)
	cbp.breakers[name] = cb
	return cb
}

func (cbp *Pool) Get(name string) *CircuitBreaker {
	cbp.mu.RLock()
	defer cbp.mu.RUnlock()
	return cbp.breakers[name]
}

func (cbp *Pool) ResetAll() {
	cbp.mu.RLock()
	defer cbp.mu.RUnlock()
	for _, cb := range cbp.breakers {
		cb.Reset()
	}
}

func (cbp *Pool) Stats() map[string]Stats {
	cbp.mu.RLock()
	defer cbp.mu.RUnlock()
	stats := make(map[string]Stats)
	for name, cb := range cbp.breakers {
		stats[name] = cb.Stats()
	}
	return stats
}