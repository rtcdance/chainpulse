package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

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
	subscriptions map[string]map[uint64]chan any
	nextID        atomic.Uint64
	droppedCount  atomic.Uint64
	logger        Logger
	metrics       MetricsCollector
}

func NewChannelEventBus() *ChannelEventBus {
	return &ChannelEventBus{
		subscriptions: make(map[string]map[uint64]chan any),
	}
}

func (b *ChannelEventBus) Publish(ctx context.Context, topic string, event any) error {
	b.mu.RLock()
	subs := b.subscriptions[topic]
	channels := make([]chan any, 0, len(subs))
	for _, ch := range subs {
		channels = append(channels, ch)
	}
	b.mu.RUnlock()

	var dropped int
	for _, ch := range channels {
		select {
		case ch <- event:
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

func (b *ChannelEventBus) Subscribe(ctx context.Context, topic string, handler func(any)) (uint64, error) {
	return b.SubscribeNamed(ctx, topic, "", handler)
}

func (b *ChannelEventBus) SubscribeNamed(ctx context.Context, topic, name string, handler func(any)) (uint64, error) {
	id := b.nextID.Add(1)
	ch := make(chan any, 64)

	b.mu.Lock()
	if b.subscriptions[topic] == nil {
		b.subscriptions[topic] = make(map[uint64]chan any)
	}
	b.subscriptions[topic][id] = ch
	b.mu.Unlock()

	go func() {
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				handler(event)
			case <-ctx.Done():
				_ = b.Unsubscribe(id)
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
		if ch, ok := subs[subscriptionID]; ok {
			close(ch)
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
