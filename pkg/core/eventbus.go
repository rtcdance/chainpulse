package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Default worker pool size for event bus
const defaultEventBusWorkers = 16

// eventBusJob wraps a handler call for the worker pool
type eventBusJob struct {
	handler EventHandler
	event   interface{}
	topic   string
	ctx     context.Context
}

// DefaultEventBus is the default implementation of EventBus
type DefaultEventBus struct {
	subscribers map[string]map[uint64]EventHandler // topic -> subID -> handler
	subIndex    map[uint64]string                  // subID -> topic (reverse lookup for Unsubscribe)
	nextSubID   atomic.Uint64
	mu          sync.RWMutex
	logger      Logger

	// Worker pool for backpressure
	workerPool chan struct{}
	wg         sync.WaitGroup

	// Graceful shutdown
	stopped atomic.Bool
	done    chan struct{}
}

// EventHandler is a function that handles events
type EventHandler func(interface{})

// SubscribeTyped subscribes to topic with a type-safe handler function.
// It wraps the underlying EventBus.Subscribe, performing the type assertion
// from interface{} to T centrally so callers don't need to.
// If the assertion fails (e.g. wrong concrete type published to the topic),
// the handler is silently skipped — matching the existing !ok pattern.
func SubscribeTyped[T any](bus EventBus, ctx context.Context, topic string, handler func(T)) (uint64, error) { //nolint:revive // ctx cannot be first param; bus is the receiver-like primary argument
	return bus.Subscribe(ctx, topic, func(raw interface{}) {
		typed, ok := raw.(T)
		if !ok {
			return
		}
		handler(typed)
	})
}

// NewEventBus creates a new event bus with a bounded worker pool
func NewEventBus(logger Logger) *DefaultEventBus {
	pool := make(chan struct{}, defaultEventBusWorkers)
	for i := 0; i < defaultEventBusWorkers; i++ {
		pool <- struct{}{}
	}
	return &DefaultEventBus{
		subscribers: make(map[string]map[uint64]EventHandler),
		subIndex:    make(map[uint64]string),
		logger:      logger,
		workerPool:  pool,
		done:        make(chan struct{}),
	}
}

// Publish publishes an event to a topic
func (eb *DefaultEventBus) Publish(ctx context.Context, topic string, event interface{}) error {
	if eb.stopped.Load() {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"event bus is stopped",
			nil,
		)
	}

	if topic == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"topic cannot be empty",
			nil,
		)
	}

	if event == nil {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"event cannot be nil",
			nil,
		)
	}

	// Copy handlers under RLock to avoid data race: holding a reference to the
	// map and iterating it after releasing the lock races with concurrent
	// Subscribe/Unsubscribe writes. A snapshot slice is safe to iterate unlocked.
	eb.mu.RLock()
	snapshot := make([]EventHandler, 0, len(eb.subscribers[topic]))
	for _, h := range eb.subscribers[topic] {
		snapshot = append(snapshot, h)
	}
	eb.mu.RUnlock()

	if len(snapshot) == 0 {
		if eb.logger != nil {
			eb.logger.Debug("no subscribers for topic", "topic", topic)
		}
		return nil
	}

	// Publish event to all subscribers via worker pool for backpressure
	for _, handler := range snapshot {
		// Check context before dispatching — avoid spawning goroutines that will immediately exit
		select {
		case <-ctx.Done():
			if eb.logger != nil {
				eb.logger.Debug("context canceled, skipping remaining handlers", "topic", topic)
			}
			return nil
		default:
		}

		eb.wg.Add(1)
		job := eventBusJob{
			handler: handler,
			event:   event,
			topic:   topic,
			ctx:     ctx,
		}
		go func(j eventBusJob) {
			defer eb.wg.Done()

			// Acquire worker slot (blocks if pool is full)
			select {
			case <-j.ctx.Done():
				if eb.logger != nil {
					eb.logger.Debug("context canceled while waiting for worker slot", "topic", j.topic)
				}
				return
			case <-eb.workerPool:
				// Got a worker slot
			}

			// Release worker slot when done
			defer func() { eb.workerPool <- struct{}{} }()

			// Execute handler with panic recovery
			defer func() {
				if r := recover(); r != nil {
					if eb.logger != nil {
						eb.logger.Error("handler panic", "topic", j.topic, "panic", r)
					}
				}
			}()

			j.handler(j.event)
		}(job)
	}

	if eb.logger != nil {
		eb.logger.Debug("event published", "topic", topic, "subscribers", len(snapshot))
	}

	return nil
}

// Subscribe subscribes to a topic and returns a subscription ID for later unsubscription
func (eb *DefaultEventBus) Subscribe(ctx context.Context, topic string, handler func(interface{})) (uint64, error) {
	if topic == "" {
		return 0, NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"topic cannot be empty",
			nil,
		)
	}

	if handler == nil {
		return 0, NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"handler cannot be nil",
			nil,
		)
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	subID := eb.nextSubID.Add(1)

	if eb.subscribers[topic] == nil {
		eb.subscribers[topic] = make(map[uint64]EventHandler)
	}
	eb.subscribers[topic][subID] = handler
	eb.subIndex[subID] = topic

	if eb.logger != nil {
		eb.logger.Debug("subscriber added", "topic", topic, "subscription_id", subID, "total_subscribers", len(eb.subscribers[topic]))
	}

	return subID, nil
}

// Unsubscribe removes a subscription by its ID
func (eb *DefaultEventBus) Unsubscribe(subscriptionID uint64) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	topic, exists := eb.subIndex[subscriptionID]
	if !exists {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeNotFound,
			fmt.Sprintf("subscription %d not found", subscriptionID),
			nil,
		)
	}

	delete(eb.subscribers[topic], subscriptionID)
	delete(eb.subIndex, subscriptionID)

	// Clean up empty topic map
	if len(eb.subscribers[topic]) == 0 {
		delete(eb.subscribers, topic)
	}

	if eb.logger != nil {
		eb.logger.Debug("subscriber removed", "topic", topic, "subscription_id", subscriptionID, "total_subscribers", len(eb.subscribers[topic]))
	}

	return nil
}

// Wait blocks until all in-flight event handlers have completed.
// Call this during graceful shutdown to ensure no events are lost.
func (eb *DefaultEventBus) Wait() {
	eb.wg.Wait()
}

// Stop prevents new publications and waits for in-flight handlers to finish.
// After Stop is called, Publish returns an error. It is safe to call Stop
// multiple times — subsequent calls are no-ops.
func (eb *DefaultEventBus) Stop() {
	if !eb.stopped.CompareAndSwap(false, true) {
		return // already stopped
	}
	close(eb.done)
	eb.wg.Wait()

	if eb.logger != nil {
		eb.logger.Info("event bus stopped")
	}
}

// GetSubscriberCount returns the number of subscribers for a topic
func (eb *DefaultEventBus) GetSubscriberCount(topic string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers, exists := eb.subscribers[topic]
	if !exists {
		return 0
	}

	return len(handlers)
}

// GetTopics returns all topics with subscribers
func (eb *DefaultEventBus) GetTopics() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	topics := make([]string, 0, len(eb.subscribers))
	for topic := range eb.subscribers {
		topics = append(topics, topic)
	}

	return topics
}

// Clear removes all subscribers
func (eb *DefaultEventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers = make(map[string]map[uint64]EventHandler)
	eb.subIndex = make(map[uint64]string)

	if eb.logger != nil {
		eb.logger.Info("event bus cleared")
	}
}

// PublishSync publishes an event synchronously
func (eb *DefaultEventBus) PublishSync(ctx context.Context, topic string, event interface{}) error {
	if topic == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"topic cannot be empty",
			nil,
		)
	}

	if event == nil {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"event cannot be nil",
			nil,
		)
	}

	eb.mu.RLock()
	snapshot := make([]EventHandler, 0, len(eb.subscribers[topic]))
	for _, h := range eb.subscribers[topic] {
		snapshot = append(snapshot, h)
	}
	eb.mu.RUnlock()

	if len(snapshot) == 0 {
		if eb.logger != nil {
			eb.logger.Debug("no subscribers for topic", "topic", topic)
		}
		return nil
	}

	// Execute handlers synchronously
	for _, handler := range snapshot {
		// Check context before executing handler
		select {
		case <-ctx.Done():
			if eb.logger != nil {
				eb.logger.Debug("context canceled before handler execution", "topic", topic)
			}
			return ctx.Err()
		default:
			// Execute handler with per-invocation panic recovery.
			// Must use a separate function — defer in a loop stacks until
			// function exit and a panic from handler N would skip handlers N+1..K.
			func(h EventHandler) {
				defer func() {
					if r := recover(); r != nil {
						if eb.logger != nil {
							eb.logger.Error("handler panic", "topic", topic, "panic", r)
						}
					}
				}()
				h(event)
			}(handler)
		}
	}

	if eb.logger != nil {
		eb.logger.Debug("event published synchronously", "topic", topic, "subscribers", len(snapshot))
	}

	return nil
}
