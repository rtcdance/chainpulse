package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// subscriber holds a channel and a done signal for safe concurrent access.
type subscriber struct {
	ch   chan any
	done chan struct{}
}

// ChannelEventBus is a pure-Go EventBus implementation backed by channels.
// It implements the core.EventBus interface without any external dependencies.
//
// This is the ideal implementation for learning Go's concurrency model:
//   - CSP (Communicating Sequential Processes) via channels
//   - sync.RWMutex for subscriber management
//   - atomic for subscription ID generation
//   - context.Context for cancellation
//
// For production use with persistence and clustering, use Kafka or Redis Stream
// implementations instead.
type ChannelEventBus struct {
	mu            sync.RWMutex
	subscriptions map[string]map[uint64]*subscriber
	nextID        atomic.Uint64
	droppedCount  atomic.Uint64
	logger        Logger
	metrics       MetricsCollector
}

func NewChannelEventBus() *ChannelEventBus {
	return &ChannelEventBus{
		subscriptions: make(map[string]map[uint64]*subscriber),
	}
}

func (b *ChannelEventBus) Publish(ctx context.Context, topic string, event any) error {
	b.mu.RLock()
	subs := b.subscriptions[topic]
	// Snapshot subscriber pointers to avoid close-vs-send race.
	// Each sub.done is only closed once (in Unsubscribe), so after
	// acquiring the snapshot the goroutine can safely check sub.done
	// before sending to sub.ch.
	subList := make([]*subscriber, 0, len(subs))
	for _, sub := range subs {
		subList = append(subList, sub)
	}
	b.mu.RUnlock()

	var dropped int
	for _, sub := range subList {
		// Skip if subscriber has been unsubscribed
		select {
		case <-sub.done:
			continue
		default:
		}

		select {
		case sub.ch <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			dropped++
		}
	}

	if dropped > 0 {
		b.droppedCount.Add(uint64(dropped))
		if b.logger != nil {
			b.logger.Warn("ChannelEventBus: events dropped due to full subscriber buffer",
				"topic", topic,
				"dropped", dropped,
				"total_dropped", b.droppedCount.Load(),
			)
		}
		if b.metrics != nil {
			b.metrics.RecordCounter("eventbus_dropped_events", int64(dropped), map[string]string{
				"topic": topic,
			})
		}
	}

	return nil
}

func (b *ChannelEventBus) Subscribe(ctx context.Context, topic string, handler EventHandler) (uint64, error) {
	return b.SubscribeNamed(ctx, topic, "", handler)
}

func (b *ChannelEventBus) SubscribeNamed(ctx context.Context, topic, name string, handler EventHandler) (uint64, error) {
	id := b.nextID.Add(1)
	sub := &subscriber{
		ch:   make(chan any, 64),
		done: make(chan struct{}),
	}

	b.mu.Lock()
	if b.subscriptions[topic] == nil {
		b.subscriptions[topic] = make(map[uint64]*subscriber)
	}
	b.subscriptions[topic][id] = sub
	b.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if b.logger != nil {
					b.logger.Error("goroutine panic recovered", "panic", r)
				}
			}
		}()
		for {
			select {
			case event, ok := <-sub.ch:
				if !ok {
					return
				}
				if err := handler(ctx, event); err != nil && b.logger != nil {
					b.logger.Error("handler returned error", "topic", topic, "error", err)
				}
			case <-ctx.Done():
				_ = b.Unsubscribe(id)
				return
			case <-sub.done:
				return
			}
		}
	}()

	return id, nil
}

func (b *ChannelEventBus) Unsubscribe(subscriptionID uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for topic, subs := range b.subscriptions {
		if sub, ok := subs[subscriptionID]; ok {
			close(sub.done)
			delete(subs, subscriptionID)
			if len(subs) == 0 {
				delete(b.subscriptions, topic)
			}
			return nil
		}
	}
	return fmt.Errorf("subscription %d not found", subscriptionID)
}

func (b *ChannelEventBus) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscriptions[topic])
}

func (b *ChannelEventBus) SetLogger(logger Logger) {
	b.logger = logger
}

func (b *ChannelEventBus) SetMetrics(metrics MetricsCollector) {
	b.metrics = metrics
}

func (b *ChannelEventBus) DroppedEvents() uint64 {
	return b.droppedCount.Load()
}
