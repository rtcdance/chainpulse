package ports

import "context"

// EventHandler processes an event. Returning an error allows the bus to
// track failures, implement retry logic, or route to a dead-letter queue.
// Implementations MUST NOT block indefinitely — the bus worker pool has
// bounded concurrency and a blocked handler stalls other subscribers.
type EventHandler func(ctx context.Context, payload any) error

// EventBus provides pub-sub communication
type EventBus interface {
	Publish(ctx context.Context, topic string, event any) error
	Subscribe(ctx context.Context, topic string, handler EventHandler) (uint64, error)
	SubscribeNamed(ctx context.Context, topic, name string, handler EventHandler) (uint64, error)
	Unsubscribe(subscriptionID uint64) error
}
