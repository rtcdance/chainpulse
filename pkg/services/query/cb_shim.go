package query

import "github.com/rtcdance/chainpulse/pkg/services/query/circuitbreaker"

// Backward-compatible shims for types moved to query/circuitbreaker.

type CircuitBreaker = circuitbreaker.CircuitBreaker
type CircuitBreakerConfig = circuitbreaker.Config
type CircuitBreakerStats = circuitbreaker.Stats
type CircuitBreakerPool = circuitbreaker.Pool
type CircuitBreakerState = circuitbreaker.State

const (
	StateClosed   = circuitbreaker.StateClosed
	StateOpen     = circuitbreaker.StateOpen
	StateHalfOpen = circuitbreaker.StateHalfOpen
)

func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return circuitbreaker.New(config)
}

func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return circuitbreaker.DefaultConfig()
}

func NewCircuitBreakerPool() *CircuitBreakerPool {
	return circuitbreaker.NewPool()
}