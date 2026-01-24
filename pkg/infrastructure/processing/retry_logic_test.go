package processing

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewRetryPolicy tests retry policy creation
func TestNewRetryPolicy(t *testing.T) {
	policy := NewRetryPolicy(3)

	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 30*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.BackoffMultiplier)
}

// TestNewCircuitBreaker tests circuit breaker creation
func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 5, cb.maxFailures)
	assert.Equal(t, 30*time.Second, cb.resetTimeout)
}

// TestCircuitBreakerSuccessfulCall tests successful operation
func TestCircuitBreakerSuccessfulCall(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	err := cb.Call(func() error {
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

// TestCircuitBreakerFailedCall tests failed operation
func TestCircuitBreakerFailedCall(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	err := cb.Call(func() error {
		return fmt.Errorf("operation failed")
	})

	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

// TestCircuitBreakerTransitionToOpen tests transition to open state
func TestCircuitBreakerTransitionToOpen(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	// Trigger 3 failures to open circuit
	for i := 0; i < 3; i++ {
		_ = cb.Call(func() error {
			return fmt.Errorf("operation failed")
		})
	}

	assert.Equal(t, StateOpen, cb.GetState())
}

// TestCircuitBreakerRejectedWhenOpen tests rejection when open
func TestCircuitBreakerRejectedWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second)

	// Trigger failure to open circuit
	_ = cb.Call(func() error {
		return fmt.Errorf("operation failed")
	})

	// Next call should be rejected
	err := cb.Call(func() error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

// TestCircuitBreakerHalfOpenTransition tests half-open transition
func TestCircuitBreakerHalfOpenTransition(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	// Open the circuit
	_ = cb.Call(func() error {
		return fmt.Errorf("operation failed")
	})

	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Next call should transition to half-open
	_ = cb.Call(func() error {
		return nil
	})

	assert.Equal(t, StateHalfOpen, cb.GetState())
}

// TestCircuitBreakerMetrics tests metrics collection
func TestCircuitBreakerMetrics(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	// Record some operations
	_ = cb.Call(func() error { return nil })
	_ = cb.Call(func() error { return fmt.Errorf("failed") })

	metrics := cb.GetMetrics()

	assert.Equal(t, StateClosed, metrics["state"])
	assert.Greater(t, metrics["total_successes"].(int64), int64(0))
	assert.Greater(t, metrics["total_failures"].(int64), int64(0))
}

// TestNewDeadLetterQueue tests DLQ creation
func TestNewDeadLetterQueue(t *testing.T) {
	dlq := NewDeadLetterQueue(1000)

	assert.NotNil(t, dlq)
	assert.Equal(t, 0, len(dlq.GetEvents()))
}

// TestDeadLetterQueueEnqueue tests enqueueing events
func TestDeadLetterQueueEnqueue(t *testing.T) {
	dlq := NewDeadLetterQueue(1000)

	event := &Event{
		ID:              "event-1",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
	}

	err := dlq.Enqueue(event, "processing failed")

	assert.NoError(t, err)
	assert.Equal(t, 1, len(dlq.GetEvents()))
}

// TestDeadLetterQueueDuplicateEvent tests duplicate event handling
func TestDeadLetterQueueDuplicateEvent(t *testing.T) {
	dlq := NewDeadLetterQueue(1000)

	event := &Event{
		ID:              "event-1",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
	}

	_ = dlq.Enqueue(event, "first failure")
	_ = dlq.Enqueue(event, "second failure")

	events := dlq.GetEvents()
	assert.Equal(t, 1, len(events))
	assert.Equal(t, 2, events[0].FailureCount)
}

// TestDeadLetterQueueMaxSize tests max size enforcement
func TestDeadLetterQueueMaxSize(t *testing.T) {
	dlq := NewDeadLetterQueue(3)

	for i := 0; i < 5; i++ {
		event := &Event{
			ID:              fmt.Sprintf("event-%d", i),
			ChainID:         "ethereum",
			ContractAddress: "0x123",
			EventName:       "Transfer",
			TransactionHash: fmt.Sprintf("0x%d", i),
		}
		_ = dlq.Enqueue(event, "failed")
	}

	events := dlq.GetEvents()
	assert.Equal(t, 3, len(events))
}

// TestDeadLetterQueueMetrics tests DLQ metrics
func TestDeadLetterQueueMetrics(t *testing.T) {
	dlq := NewDeadLetterQueue(1000)

	event := &Event{
		ID:              "event-1",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
	}

	_ = dlq.Enqueue(event, "failed")

	metrics := dlq.GetMetrics()

	assert.Equal(t, int64(1), metrics["queued_count"])
	assert.Equal(t, 1, metrics["current_size"])
	assert.Equal(t, 1000, metrics["max_size"])
}

// TestNewRetryManager tests retry manager creation
func TestNewRetryManager(t *testing.T) {
	policy := NewRetryPolicy(3)
	rm := NewRetryManager(policy)

	assert.NotNil(t, rm)
	assert.NotNil(t, rm.circuitBreaker)
	assert.NotNil(t, rm.deadLetterQueue)
}

// TestRetryManagerExecuteSuccess tests successful execution
func TestRetryManagerExecuteSuccess(t *testing.T) {
	policy := NewRetryPolicy(3)
	rm := NewRetryManager(policy)
	ctx := context.Background()

	err := rm.ExecuteWithRetry(ctx, func() error {
		return nil
	})

	assert.NoError(t, err)
}

// TestRetryManagerExecuteWithRetry tests retry on failure
func TestRetryManagerExecuteWithRetry(t *testing.T) {
	policy := NewRetryPolicy(3)
	rm := NewRetryManager(policy)
	ctx := context.Background()

	var attempts int32
	err := rm.ExecuteWithRetry(ctx, func() error {
		atomic.AddInt32(&attempts, 1)
		if atomic.LoadInt32(&attempts) < 2 {
			return fmt.Errorf("temporary failure")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

// TestRetryManagerExhaustedRetries tests exhausted retries
func TestRetryManagerExhaustedRetries(t *testing.T) {
	policy := NewRetryPolicy(2)
	rm := NewRetryManager(policy)
	ctx := context.Background()

	var attempts int32
	err := rm.ExecuteWithRetry(ctx, func() error {
		atomic.AddInt32(&attempts, 1)
		return fmt.Errorf("permanent failure")
	})

	assert.Error(t, err)
	assert.Greater(t, atomic.LoadInt32(&attempts), int32(0))
}

// TestRetryManagerContextCancellation tests context cancellation
func TestRetryManagerContextCancellation(t *testing.T) {
	policy := NewRetryPolicy(10)
	rm := NewRetryManager(policy)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rm.ExecuteWithRetry(ctx, func() error {
		return fmt.Errorf("failure")
	})

	assert.Error(t, err)
}

// TestRetryManagerMetrics tests metrics collection
func TestRetryManagerMetrics(t *testing.T) {
	policy := NewRetryPolicy(3)
	rm := NewRetryManager(policy)
	ctx := context.Background()

	_ = rm.ExecuteWithRetry(ctx, func() error {
		return nil
	})

	metrics := rm.GetMetrics()

	assert.Greater(t, metrics["success_count"].(int64), int64(0))
	assert.NotNil(t, metrics["circuit_breaker"])
	assert.NotNil(t, metrics["dead_letter_queue"])
}

// TestCalculateBackoff tests backoff calculation
func TestCalculateBackoff(t *testing.T) {
	initial := 100 * time.Millisecond
	max := 10 * time.Second
	multiplier := 2.0

	backoff0 := CalculateBackoff(0, initial, max, multiplier)
	backoff1 := CalculateBackoff(1, initial, max, multiplier)
	backoff2 := CalculateBackoff(2, initial, max, multiplier)

	assert.Equal(t, initial, backoff0)
	assert.Equal(t, 200*time.Millisecond, backoff1)
	assert.Equal(t, 400*time.Millisecond, backoff2)
}

// TestCalculateBackoffCap tests backoff cap
func TestCalculateBackoffCap(t *testing.T) {
	initial := 100 * time.Millisecond
	max := 1 * time.Second
	multiplier := 2.0

	backoff := CalculateBackoff(10, initial, max, multiplier)

	assert.LessOrEqual(t, backoff, max)
}

// TestConcurrentCircuitBreakerCalls tests concurrent calls
func TestConcurrentCircuitBreakerCalls(t *testing.T) {
	cb := NewCircuitBreaker(100, 30*time.Second)

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Call(func() error {
				return nil
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&successCount))
}

// TestConcurrentDeadLetterQueueOperations tests concurrent DLQ operations
func TestConcurrentDeadLetterQueueOperations(t *testing.T) {
	dlq := NewDeadLetterQueue(10000)

	var wg sync.WaitGroup
	var enqueuedCount int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := &Event{
				ID:              fmt.Sprintf("event-%d", id),
				ChainID:         "ethereum",
				ContractAddress: "0x123",
				EventName:       "Transfer",
				TransactionHash: fmt.Sprintf("0x%d", id),
			}
			_ = dlq.Enqueue(event, "failed")
			atomic.AddInt32(&enqueuedCount, 1)
		}(i)
	}

	wg.Wait()

	assert.Greater(t, atomic.LoadInt32(&enqueuedCount), int32(0))
}

// TestRetryPolicyBackoffMultiplier tests backoff multiplier
func TestRetryPolicyBackoffMultiplier(t *testing.T) {
	policy := NewRetryPolicy(5)

	assert.Equal(t, 2.0, policy.BackoffMultiplier)
}

// TestCircuitBreakerStateTransitions tests state transitions
func TestCircuitBreakerStateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Start in closed state
	assert.Equal(t, StateClosed, cb.GetState())

	// Trigger failures to open
	_ = cb.Call(func() error { return fmt.Errorf("fail") })
	_ = cb.Call(func() error { return fmt.Errorf("fail") })

	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Transition to half-open
	_ = cb.Call(func() error { return nil })

	assert.Equal(t, StateHalfOpen, cb.GetState())
}

// TestDeadLetterQueueNilEvent tests nil event handling
func TestDeadLetterQueueNilEvent(t *testing.T) {
	dlq := NewDeadLetterQueue(1000)

	err := dlq.Enqueue(nil, "failed")

	assert.Error(t, err)
}

// TestRetryManagerBackoffProgression tests backoff progression
func TestRetryManagerBackoffProgression(t *testing.T) {
	policy := NewRetryPolicy(3)
	rm := NewRetryManager(policy)
	ctx := context.Background()

	var attempts int32
	startTime := time.Now()

	_ = rm.ExecuteWithRetry(ctx, func() error {
		atomic.AddInt32(&attempts, 1)
		if atomic.LoadInt32(&attempts) < 3 {
			return fmt.Errorf("failure")
		}
		return nil
	})

	elapsed := time.Since(startTime)

	// Should have some delay due to backoff
	assert.Greater(t, elapsed, 50*time.Millisecond)
}

// TestCircuitBreakerMetricsStateChanges tests state change tracking
func TestCircuitBreakerMetricsStateChanges(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	// Trigger state changes
	_ = cb.Call(func() error { return fmt.Errorf("fail") })

	metrics := cb.GetMetrics()

	assert.Greater(t, metrics["state_changes"].(int64), int64(0))
}

// TestRetryManagerMultipleOperations tests multiple operations
func TestRetryManagerMultipleOperations(t *testing.T) {
	policy := NewRetryPolicy(2)
	rm := NewRetryManager(policy)
	ctx := context.Background()

	// Execute multiple operations
	for i := 0; i < 5; i++ {
		_ = rm.ExecuteWithRetry(ctx, func() error {
			return nil
		})
	}

	metrics := rm.GetMetrics()

	assert.Greater(t, metrics["success_count"].(int64), int64(0))
}

// TestDeadLetterQueueEventStatus tests event status tracking
func TestDeadLetterQueueEventStatus(t *testing.T) {
	dlq := NewDeadLetterQueue(1000)

	event := &Event{
		ID:              "event-1",
		ChainID:         "ethereum",
		ContractAddress: "0x123",
		EventName:       "Transfer",
		TransactionHash: "0xabc",
	}

	_ = dlq.Enqueue(event, "failed")

	events := dlq.GetEvents()
	assert.Equal(t, "pending", events[0].Status)
}

// TestCircuitBreakerConcurrentStateAccess tests concurrent state access
func TestCircuitBreakerConcurrentStateAccess(t *testing.T) {
	cb := NewCircuitBreaker(100, 30*time.Second)

	var wg sync.WaitGroup
	var stateReads int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.GetState()
			atomic.AddInt32(&stateReads, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&stateReads))
}

// TestRetryPolicyDefaults tests default retry policy values
func TestRetryPolicyDefaults(t *testing.T) {
	policy := NewRetryPolicy(5)

	assert.Equal(t, 5, policy.MaxRetries)
	assert.Greater(t, policy.InitialBackoff, time.Duration(0))
	assert.Greater(t, policy.MaxBackoff, policy.InitialBackoff)
	assert.Greater(t, policy.BackoffMultiplier, 1.0)
}
