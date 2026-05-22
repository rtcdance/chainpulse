package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rtcdance/chainpulse/pkg/ports"
)

// Default worker pool size for event bus
const defaultEventBusWorkers = 16
const defaultPublishTimeout = 5 * time.Second

// eventBusJob wraps a handler call for the worker pool
type eventBusJob struct {
	handler        EventHandler
	event          any
	topic          string
	subscriberName string
	ctx            context.Context
}

// DefaultEventBus is the default implementation of EventBus
type DefaultEventBus struct {
	subscribers     map[string]map[uint64]EventHandler
	subIndex        map[uint64]string
	subscriberNames map[uint64]string
	nextSubID       atomic.Uint64
	mu              sync.RWMutex
	logger          Logger

	jobCh     chan eventBusJob
	workersWg sync.WaitGroup

	publishTimeout atomic.Int64

	stopped atomic.Bool
	done    chan struct{}

	// droppedJobs counts how many jobs were dropped due to channel saturation.
	// This happens when all workers are busy and the context deadline is exceeded.
	droppedJobs atomic.Uint64
}

// EventHandler is a function that handles events
type EventHandler = ports.EventHandler

// SubscribeTyped subscribes to topic with a type-safe handler function.
// It wraps the underlying EventBus.Subscribe, performing the type assertion
// from interface{} to T centrally so callers don't need to.
// If the assertion fails (e.g. wrong concrete type published to the topic),
// the handler is silently skipped — matching the existing !ok pattern.
func SubscribeTyped[T any](bus EventBus, ctx context.Context, topic string, handler func(T)) (uint64, error) { //nolint:revive // ctx cannot be first param; bus is the receiver-like primary argument
	return bus.Subscribe(ctx, topic, func(_ context.Context, raw any) error {
		typed, ok := raw.(T)
		if !ok {
			return nil
		}
		handler(typed)
		return nil
	})
}

// SubscribeTypedNamed is like SubscribeTyped but records a human-readable
// subscriber name for debugging. The name appears in panic recovery logs.
func SubscribeTypedNamed[T any](bus EventBus, ctx context.Context, topic, name string, handler func(T)) (uint64, error) { //nolint:revive // ctx cannot be first param; bus is the receiver-like primary argument
	return bus.SubscribeNamed(ctx, topic, name, func(_ context.Context, raw any) error {
		typed, ok := raw.(T)
		if !ok {
			return nil
		}
		handler(typed)
		return nil
	})
}

// NewEventBus creates a new event bus with a fixed-size worker pool.
// Workers are started immediately and consume jobs from a bounded channel,
// providing natural backpressure without unbounded goroutine growth.
func NewEventBus(logger Logger) *DefaultEventBus {
	jobCh := make(chan eventBusJob, defaultEventBusWorkers)
	eb := &DefaultEventBus{
		subscribers:     make(map[string]map[uint64]EventHandler),
		subIndex:        make(map[uint64]string),
		subscriberNames: make(map[uint64]string),
		logger:          logger,
		jobCh:           jobCh,
		done:            make(chan struct{}),
	}
	eb.publishTimeout.Store(int64(defaultPublishTimeout))

	// Start fixed worker goroutines
	for i := 0; i < defaultEventBusWorkers; i++ {
		eb.workersWg.Add(1)
		go eb.workerLoop()
	}

	return eb
}

// workerLoop is the main loop for each worker goroutine.
// It consumes jobs from the job channel until the bus is stopped.
// When the done channel closes, each worker drains any remaining
// jobs from jobCh before exiting, ensuring no in-flight events are lost.
func (eb *DefaultEventBus) workerLoop() {
	defer eb.workersWg.Done()
	for {
		select {
		case <-eb.done:
			for {
				select {
				case job, ok := <-eb.jobCh:
					if !ok {
						return
					}
					eb.executeJob(job)
				default:
					return
				}
			}
		case job, ok := <-eb.jobCh:
			if !ok {
				return
			}
			eb.executeJob(job)
		}
	}
}

// executeJob runs a single job with panic recovery.
func (eb *DefaultEventBus) executeJob(j eventBusJob) {
	defer func() {
		if r := recover(); r != nil {
			if eb.logger != nil {
				eb.logger.Error("handler panic", "topic", j.topic, "subscriber", j.subscriberName, "panic", r)
			}
		}
	}()
	if err := j.handler(j.ctx, j.event); err != nil {
		if eb.logger != nil {
			eb.logger.Warn("handler returned error", "topic", j.topic, "subscriber", j.subscriberName, "error", err)
		}
	}
}

// Publish publishes an event to a topic
func (eb *DefaultEventBus) Publish(ctx context.Context, topic string, event any) error {
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
	type subEntry struct {
		handler EventHandler
		name    string
	}
	snapshot := make([]subEntry, 0, len(eb.subscribers[topic]))
	for subID, h := range eb.subscribers[topic] {
		snapshot = append(snapshot, subEntry{handler: h, name: eb.subscriberNames[subID]})
	}
	eb.mu.RUnlock()

	if len(snapshot) == 0 {
		if eb.logger != nil {
			eb.logger.Debug("no subscribers for topic", "topic", topic)
		}
		return nil
	}

	// Publish event to all subscribers via fixed worker pool for backpressure.
	// Each handler invocation becomes a job sent to the bounded job channel.
	// When the channel is full (all workers busy), Publish waits up to 5 seconds
	// per job before dropping it — preventing deadlock when ctx has no timeout.
	for _, entry := range snapshot {
		select {
		case <-ctx.Done():
			if eb.logger != nil {
				eb.logger.Debug("context canceled, skipping remaining handlers", "topic", topic)
			}
			return nil
		case eb.jobCh <- eventBusJob{
			handler:        entry.handler,
			event:          event,
			topic:          topic,
			subscriberName: entry.name,
			ctx:            ctx,
		}:
			// Job enqueued successfully
		default:
			// Channel is full — use a bounded wait to avoid deadlock
			// when context has no deadline.
			waitTimer := time.NewTimer(time.Duration(eb.publishTimeout.Load()))
			select {
			case <-ctx.Done():
				waitTimer.Stop()
				if eb.logger != nil {
					eb.logger.Warn("context canceled while waiting for worker, dropping job", "topic", topic)
				}
				return nil
			case eb.jobCh <- eventBusJob{
				handler:        entry.handler,
				event:          event,
				topic:          topic,
				subscriberName: entry.name,
				ctx:            ctx,
			}:
				waitTimer.Stop()
			case <-waitTimer.C:
				eb.droppedJobs.Add(1)
				if eb.logger != nil {
					eb.logger.Warn("event bus worker pool saturated, dropping job",
						"topic", topic, "subscriber", entry.name,
						"total_dropped", eb.droppedJobs.Load(),
					)
				}
			}
		}
	}

	if eb.logger != nil {
		eb.logger.Debug("event published", "topic", topic, "subscribers", len(snapshot))
	}

	return nil
}

// SetPublishTimeout sets the maximum time Publish waits for a busy worker
// before dropping the job. Default is 5 seconds.
func (eb *DefaultEventBus) SetPublishTimeout(timeout time.Duration) {
	eb.publishTimeout.Store(int64(timeout))
}

// Subscribe subscribes to a topic and returns a subscription ID for later unsubscription
func (eb *DefaultEventBus) Subscribe(ctx context.Context, topic string, handler EventHandler) (uint64, error) {
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

// SubscribeNamed subscribes to a topic with a human-readable name for debugging.
// The name appears in panic recovery logs and debug output, making it easy to
// identify which component crashed or is handling an event. If name is empty,
// falls back to Subscribe (no name recorded).
func (eb *DefaultEventBus) SubscribeNamed(ctx context.Context, topic, name string, handler EventHandler) (uint64, error) {
	subID, err := eb.Subscribe(ctx, topic, handler)
	if err != nil {
		return 0, err
	}

	if name != "" {
		eb.mu.Lock()
		eb.subscriberNames[subID] = name
		eb.mu.Unlock()
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
	delete(eb.subscriberNames, subscriptionID)

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
	eb.workersWg.Wait()
}

// Stop prevents new publications and waits for in-flight handlers to finish.
// After Stop is called, Publish returns an error. It is safe to call Stop
// multiple times — subsequent calls are no-ops.
func (eb *DefaultEventBus) Stop() {
	if !eb.stopped.CompareAndSwap(false, true) {
		return // already stopped
	}
	close(eb.done)
	eb.workersWg.Wait()

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
	eb.subscriberNames = make(map[uint64]string)

	if eb.logger != nil {
		eb.logger.Info("event bus cleared")
	}
}

// GetDroppedJobs returns the number of jobs dropped due to worker pool saturation.
func (eb *DefaultEventBus) GetDroppedJobs() uint64 {
	return eb.droppedJobs.Load()
}

// Drain waits for all remaining jobs in the job channel to be processed.
// Call this before Stop() during graceful shutdown to ensure all published
// events are handled. Returns the number of jobs drained.
// If the bus is already stopped, drains immediately and returns.
func (eb *DefaultEventBus) Drain(timeout time.Duration) int {
	drained := 0
	deadline := time.Now().Add(timeout)

	for {
		select {
		case job, ok := <-eb.jobCh:
			if !ok {
				return drained
			}
			eb.executeJob(job)
			drained++
		default:
			if time.Now().After(deadline) {
				return drained
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// PublishSync publishes an event synchronously
func (eb *DefaultEventBus) PublishSync(ctx context.Context, topic string, event any) error {
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
				h(ctx, event)
			}(handler)
		}
	}

	if eb.logger != nil {
		eb.logger.Debug("event published synchronously", "topic", topic, "subscribers", len(snapshot))
	}

	return nil
}
