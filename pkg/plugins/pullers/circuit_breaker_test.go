package pullers

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestCircuitBreakerInitialState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 3, Cooldown: time.Second, HalfOpenLimit: 2})
	if cb.State() != CircuitBreakerClosed {
		t.Fatalf("expected closed, got %v", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCircuitBreakerOpensOnFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute, HalfOpenLimit: 2})

	cb.Failure()
	cb.Failure()
	if cb.State() != CircuitBreakerClosed {
		t.Fatalf("expected closed after 2 failures, got %v", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	cb.Failure()
	if cb.State() != CircuitBreakerOpen {
		t.Fatalf("expected open after 3 failures, got %v", cb.State())
	}
	if err := cb.Allow(); !errors.Is(err, core.ErrRPCUnreachable) {
		t.Fatalf("expected ErrRPCUnreachable, got %v", err)
	}
}

func TestCircuitBreakerDoesNotCloseOnSuccessInOpenState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute, HalfOpenLimit: 2})

	cb.Failure()
	cb.Failure()
	cb.Failure()
	if cb.State() != CircuitBreakerOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}

	cb.Success()
	if cb.State() != CircuitBreakerOpen {
		t.Fatalf("expected still open after success, got %v", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 3, Cooldown: time.Minute, HalfOpenLimit: 2})

	cb.Failure()
	cb.Failure()
	cb.Failure()
	if cb.State() != CircuitBreakerOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}

	cb.Reset()
	if cb.State() != CircuitBreakerClosed {
		t.Fatalf("expected closed after reset, got %v", cb.State())
	}
}

func TestCircuitBreakerCounts(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 5, Cooldown: time.Minute, HalfOpenLimit: 2})

	cb.Failure()
	cb.Failure()

	current, threshold := cb.Counts()
	if current != 2 {
		t.Fatalf("expected 2 failures, got %d", current)
	}
	if threshold != 5 {
		t.Fatalf("expected threshold 5, got %d", threshold)
	}
}

func TestCircuitBreakerConcurrentSafe(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 10, Cooldown: time.Minute, HalfOpenLimit: 2})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Failure()
			_ = cb.Allow()
			cb.Success()
			cb.State()
			cb.Counts()
			cb.Reset()
		}()
	}
	wg.Wait()
}

func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 3, Cooldown: time.Millisecond, HalfOpenLimit: 2})

	cb.Failure()
	cb.Failure()
	cb.Failure()
	if cb.State() != CircuitBreakerOpen {
		t.Fatalf("expected open, got %v", cb.State())
	}

	time.Sleep(10 * time.Millisecond)

	if err := cb.Allow(); err != nil {
		t.Fatalf("expected nil after cooldown, got %v", err)
	}
	if cb.State() != CircuitBreakerHalfOpen {
		t.Fatalf("expected half-open, got %v", cb.State())
	}

	if err := cb.Allow(); !errors.Is(err, core.ErrRPCUnreachable) {
		t.Fatalf("expected ErrRPCUnreachable for second caller, got %v", err)
	}

	cb.Success()
	if cb.State() != CircuitBreakerHalfOpen {
		t.Fatalf("expected half-open after 1/2 success, got %v", cb.State())
	}

	if err := cb.Allow(); err != nil {
		t.Fatalf("expected nil for next probe, got %v", err)
	}

	cb.Success()
	if cb.State() != CircuitBreakerClosed {
		t.Fatalf("expected closed after 2/2 successes, got %v", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected nil after recovery, got %v", err)
	}
}

func TestCircuitBreakerHalfOpenFailureTripsBack(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 3, Cooldown: time.Millisecond, HalfOpenLimit: 2})

	cb.Failure()
	cb.Failure()
	cb.Failure()
	time.Sleep(10 * time.Millisecond)

	_ = cb.Allow()
	if cb.State() != CircuitBreakerHalfOpen {
		t.Fatalf("expected half-open, got %v", cb.State())
	}

	cb.Failure()
	if cb.State() != CircuitBreakerOpen {
		t.Fatalf("expected open after half-open failure, got %v", cb.State())
	}
}
