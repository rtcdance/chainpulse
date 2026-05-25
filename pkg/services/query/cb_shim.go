package query

import "github.com/rtcdance/chainpulse/pkg/services/query/circuitbreaker"

// Backward-compatible shims for types moved to query/circuitbreaker.

type (
	CircuitBreaker       = circuitbreaker.CircuitBreaker
	CircuitBreakerConfig = circuitbreaker.Config
	CircuitBreakerStats  = circuitbreaker.Stats
	CircuitBreakerPool   = circuitbreaker.Pool
	CircuitBreakerState  = circuitbreaker.State
)

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
