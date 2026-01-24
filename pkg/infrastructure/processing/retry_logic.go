package processing

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiplier float64
}

// CircuitBreakerState represents circuit breaker state
type CircuitBreakerState string

const (
	StateClosed     CircuitBreakerState = "closed"
	StateOpen       CircuitBreakerState = "open"
	StateHalfOpen   CircuitBreakerState = "half-open"
)

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitBreakerState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	lastSuccessTime   time.Time
	maxFailures       int
	resetTimeout      time.Duration
	halfOpenMaxTests  int
	metrics           *CircuitBreakerMetrics
}

// CircuitBreakerMetrics tracks circuit breaker metrics
type CircuitBreakerMetrics struct {
	mu              sync.RWMutex
	StateChanges    int64
	FailureCount    int64
	SuccessCount    int64
	RejectedCount   int64
	LastStateChange time.Time
}

// DeadLetterQueue stores failed events
type DeadLetterQueue struct {
	mu       sync.RWMutex
	events   []*DeadLetterEvent
	maxSize  int
	metrics  *DLQMetrics
}

// DeadLetterEvent represents an event in the DLQ
type DeadLetterEvent struct {
	Event         *Event
	FailureReason string
	FailureCount  int
	FirstFailure  time.Time
	LastFailure   time.Time
	Status        string // "pending", "archived", "resolved"
}

// DLQMetrics tracks DLQ metrics
type DLQMetrics struct {
	mu           sync.RWMutex
	EventsQueued int64
	EventsArchived int64
	EventsResolved int64
}

// RetryManager manages retry logic
type RetryManager struct {
	mu                sync.RWMutex
	policy            RetryPolicy
	circuitBreaker    *CircuitBreaker
	deadLetterQueue   *DeadLetterQueue
	retryCount        int64
	successCount      int64
	failureCount      int64
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(maxRetries int) RetryPolicy {
	return RetryPolicy{
		MaxRetries:        maxRetries,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		maxFailures:      maxFailures,
		resetTimeout:     resetTimeout,
		halfOpenMaxTests: 3,
		metrics: &CircuitBreakerMetrics{
			LastStateChange: time.Now(),
		},
	}
}

// Call executes an operation with circuit breaker protection
func (cb *CircuitBreaker) Call(operation func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we should transition from open to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.transitionToHalfOpen()
		} else {
			cb.metrics.mu.Lock()
			cb.metrics.RejectedCount++
			cb.metrics.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute operation
	err := operation()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		cb.metrics.mu.Lock()
		cb.metrics.FailureCount++
		cb.metrics.mu.Unlock()

		// Transition to open if max failures reached
		if cb.failureCount >= cb.maxFailures {
			cb.transitionToOpen()
		}

		return err
	}

	// Success
	cb.successCount++
	cb.lastSuccessTime = time.Now()

	cb.metrics.mu.Lock()
	cb.metrics.SuccessCount++
	cb.metrics.mu.Unlock()

	// Transition to closed if in half-open state
	if cb.state == StateHalfOpen && cb.successCount >= cb.halfOpenMaxTests {
		cb.transitionToClosed()
	}

	return nil
}

// transitionToOpen transitions circuit breaker to open state
func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.failureCount = 0
	cb.successCount = 0

	cb.metrics.mu.Lock()
	cb.metrics.StateChanges++
	cb.metrics.LastStateChange = time.Now()
	cb.metrics.mu.Unlock()
}

// transitionToHalfOpen transitions circuit breaker to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.state = StateHalfOpen
	cb.failureCount = 0
	cb.successCount = 0

	cb.metrics.mu.Lock()
	cb.metrics.StateChanges++
	cb.metrics.LastStateChange = time.Now()
	cb.metrics.mu.Unlock()
}

// transitionToClosed transitions circuit breaker to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0

	cb.metrics.mu.Lock()
	cb.metrics.StateChanges++
	cb.metrics.LastStateChange = time.Now()
	cb.metrics.mu.Unlock()
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	cb.metrics.mu.RLock()
	defer cb.metrics.mu.RUnlock()

	return map[string]interface{}{
		"state":              cb.state,
		"failure_count":      cb.failureCount,
		"success_count":      cb.successCount,
		"state_changes":      cb.metrics.StateChanges,
		"total_failures":     cb.metrics.FailureCount,
		"total_successes":    cb.metrics.SuccessCount,
		"rejected_count":     cb.metrics.RejectedCount,
		"last_state_change":  cb.metrics.LastStateChange,
	}
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(maxSize int) *DeadLetterQueue {
	return &DeadLetterQueue{
		events:  make([]*DeadLetterEvent, 0, maxSize),
		maxSize: maxSize,
		metrics: &DLQMetrics{},
	}
}

// Enqueue adds an event to the DLQ
func (dlq *DeadLetterQueue) Enqueue(event *Event, reason string) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Check if event already in queue
	for _, dlqEvent := range dlq.events {
		if dlqEvent.Event.ID == event.ID {
			dlqEvent.FailureCount++
			dlqEvent.LastFailure = time.Now()
			return nil
		}
	}

	// Add new event
	if len(dlq.events) >= dlq.maxSize {
		// Remove oldest event
		dlq.events = dlq.events[1:]
	}

	dlqEvent := &DeadLetterEvent{
		Event:         event,
		FailureReason: reason,
		FailureCount:  1,
		FirstFailure:  time.Now(),
		LastFailure:   time.Now(),
		Status:        "pending",
	}

	dlq.events = append(dlq.events, dlqEvent)

	dlq.metrics.mu.Lock()
	dlq.metrics.EventsQueued++
	dlq.metrics.mu.Unlock()

	return nil
}

// GetEvents returns all events in the DLQ
func (dlq *DeadLetterQueue) GetEvents() []*DeadLetterEvent {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	result := make([]*DeadLetterEvent, len(dlq.events))
	copy(result, dlq.events)
	return result
}

// GetMetrics returns DLQ metrics
func (dlq *DeadLetterQueue) GetMetrics() map[string]interface{} {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	dlq.metrics.mu.RLock()
	defer dlq.metrics.mu.RUnlock()

	return map[string]interface{}{
		"queued_count":   dlq.metrics.EventsQueued,
		"archived_count": dlq.metrics.EventsArchived,
		"resolved_count": dlq.metrics.EventsResolved,
		"current_size":   len(dlq.events),
		"max_size":       dlq.maxSize,
	}
}

// NewRetryManager creates a new retry manager
func NewRetryManager(policy RetryPolicy) *RetryManager {
	return &RetryManager{
		policy:          policy,
		circuitBreaker:  NewCircuitBreaker(5, 30*time.Second),
		deadLetterQueue: NewDeadLetterQueue(10000),
	}
}

// ExecuteWithRetry executes an operation with retry logic
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	backoff := rm.policy.InitialBackoff

	for attempt := 0; attempt <= rm.policy.MaxRetries; attempt++ {
		// Check circuit breaker
		err := rm.circuitBreaker.Call(operation)
		if err == nil {
			rm.mu.Lock()
			rm.successCount++
			rm.mu.Unlock()
			return nil
		}

		lastErr = err
		rm.mu.Lock()
		rm.retryCount++
		rm.mu.Unlock()

		if attempt < rm.policy.MaxRetries {
			// Calculate backoff with jitter
			jitter := time.Duration(float64(backoff) * 0.1)
			waitTime := backoff + jitter

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
				// Continue to next attempt
			}

			// Increase backoff for next attempt
			backoff = time.Duration(float64(backoff) * rm.policy.BackoffMultiplier)
			if backoff > rm.policy.MaxBackoff {
				backoff = rm.policy.MaxBackoff
			}
		}
	}

	rm.mu.Lock()
	rm.failureCount++
	rm.mu.Unlock()

	return fmt.Errorf("operation failed after %d retries: %w", rm.policy.MaxRetries, lastErr)
}

// GetMetrics returns retry manager metrics
func (rm *RetryManager) GetMetrics() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return map[string]interface{}{
		"retry_count":        rm.retryCount,
		"success_count":      rm.successCount,
		"failure_count":      rm.failureCount,
		"circuit_breaker":    rm.circuitBreaker.GetMetrics(),
		"dead_letter_queue":  rm.deadLetterQueue.GetMetrics(),
		"max_retries":        rm.policy.MaxRetries,
		"initial_backoff":    rm.policy.InitialBackoff.String(),
		"max_backoff":        rm.policy.MaxBackoff.String(),
		"backoff_multiplier": rm.policy.BackoffMultiplier,
	}
}

// CalculateBackoff calculates exponential backoff with jitter
func CalculateBackoff(attempt int, initialBackoff time.Duration, maxBackoff time.Duration, multiplier float64) time.Duration {
	backoff := time.Duration(float64(initialBackoff) * math.Pow(multiplier, float64(attempt)))
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}
