package core

import "context"

// TypedEventBus wraps a standard EventBus and provides type-safe
// publish/subscribe methods using Go generics.
//
// This eliminates the need for type assertions in event handlers,
// catching type mismatches at compile time rather than runtime.
//
// Usage:
//
//	bus := core.NewChannelEventBus()
//	typed := core.NewTypedEventBus[blockchain.BlockchainEvent](bus)
//
//	typed.Subscribe(ctx, "Transfer", func(evt blockchain.BlockchainEvent) {
//	    fmt.Println(evt.BlockNumber) // full IDE autocomplete
//	})
//
//	typed.Publish(ctx, "Transfer", evt)
type TypedEventBus[T any] struct {
	bus EventBus
}

func NewTypedEventBus[T any](bus EventBus) *TypedEventBus[T] {
	return &TypedEventBus[T]{bus: bus}
}

func (b *TypedEventBus[T]) Publish(ctx context.Context, topic string, event T) error {
	return b.bus.Publish(ctx, topic, event)
}

func (b *TypedEventBus[T]) Subscribe(ctx context.Context, topic string, handler func(context.Context, T) error) (uint64, error) {
	return b.bus.SubscribeNamed(ctx, topic, "", func(ctx context.Context, event any) error {
		if typed, ok := event.(T); ok {
			return handler(ctx, typed)
		}
		return nil
	})
}

func (b *TypedEventBus[T]) SubscribeNamed(ctx context.Context, topic, name string, handler func(context.Context, T) error) (uint64, error) {
	return b.bus.SubscribeNamed(ctx, topic, name, func(ctx context.Context, event any) error {
		if typed, ok := event.(T); ok {
			return handler(ctx, typed)
		}
		return nil
	})
}

func (b *TypedEventBus[T]) Unsubscribe(subscriptionID uint64) error {
	return b.bus.Unsubscribe(subscriptionID)
}

func (b *TypedEventBus[T]) Inner() EventBus {
	return b.bus
}
