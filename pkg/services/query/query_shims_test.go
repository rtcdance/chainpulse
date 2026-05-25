package query

import (
	"testing"
	"time"
)

func TestNewErrorClassifier_Shim(t *testing.T) {
	t.Parallel()
	c := NewErrorClassifier()
	if c == nil {
		t.Fatal("expected non-nil ErrorClassifier")
	}
}

func TestNewIndexManager_Shim(t *testing.T) {
	t.Parallel()
	m := NewIndexManager()
	if m == nil {
		t.Fatal("expected non-nil IndexManager")
	}
}

func TestNewQueryOptimizer_Shim(t *testing.T) {
	t.Parallel()
	o := NewQueryOptimizer(100)
	if o == nil {
		t.Fatal("expected non-nil QueryOptimizer")
	}
}

func TestNewQueryStatisticsCollector_Shim(t *testing.T) {
	t.Parallel()
	s := NewQueryStatisticsCollector(time.Minute)
	if s == nil {
		t.Fatal("expected non-nil QueryStatisticsCollector")
	}
}

func TestNewCircuitBreaker_Shim(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if cb == nil {
		t.Fatal("expected non-nil CircuitBreaker")
	}
}

func TestDefaultCircuitBreakerConfig_Shim(t *testing.T) {
	t.Parallel()
	cfg := DefaultCircuitBreakerConfig()
	if cfg == nil {
		t.Fatal("expected non-nil CircuitBreakerConfig")
	}
}

func TestNewCircuitBreakerPool_Shim(t *testing.T) {
	t.Parallel()
	pool := NewCircuitBreakerPool()
	if pool == nil {
		t.Fatal("expected non-nil CircuitBreakerPool")
	}
}
