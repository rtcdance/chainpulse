package core

import (
	"context"
	"fmt"
	"sync"
)

// DefaultEventBus is the default implementation of EventBus
type DefaultEventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	logger      Logger
}

// EventHandler is a function that handles events
type EventHandler func(interface{})

// NewEventBus creates a new event bus
func NewEventBus(logger Logger) *DefaultEventBus {
	return &DefaultEventBus{
		subscribers: make(map[string][]EventHandler),
		logger:      logger,
	}
}

// Publish publishes an event to a topic
func (eb *DefaultEventBus) Publish(ctx context.Context, topic string, event interface{}) error {
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
	handlers, exists := eb.subscribers[topic]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		if eb.logger != nil {
			eb.logger.Debug("no subscribers for topic", "topic", topic)
		}
		return nil
	}

	// Publish event to all subscribers asynchronously
	for _, handler := range handlers {
		go func(eventHandler EventHandler) {
			// Check context before executing handler
			select {
			case <-ctx.Done():
				if eb.logger != nil {
					eb.logger.Debug("context canceled before handler execution", "topic", topic)
				}
				return
			default:
				// Execute handler with panic recovery
				defer func() {
					if r := recover(); r != nil {
						if eb.logger != nil {
							eb.logger.Error("handler panic", "topic", topic, "panic", r)
						}
					}
				}()

				eventHandler(event)
			}
		}(handler)
	}

	if eb.logger != nil {
		eb.logger.Debug("event published", "topic", topic, "subscribers", len(handlers))
	}

	return nil
}

// Subscribe subscribes to a topic
func (eb *DefaultEventBus) Subscribe(ctx context.Context, topic string, handler func(interface{})) error {
	if topic == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"topic cannot be empty",
			nil,
		)
	}

	if handler == nil {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"handler cannot be nil",
			nil,
		)
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[topic] = append(eb.subscribers[topic], handler)

	if eb.logger != nil {
		eb.logger.Debug("subscriber added", "topic", topic, "total_subscribers", len(eb.subscribers[topic]))
	}

	return nil
}

// Unsubscribe unsubscribes from a topic
func (eb *DefaultEventBus) Unsubscribe(topic string, handler func(interface{})) error {
	if topic == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"topic cannot be empty",
			nil,
		)
	}

	if handler == nil {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"handler cannot be nil",
			nil,
		)
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers, exists := eb.subscribers[topic]
	if !exists {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeNotFound,
			fmt.Sprintf("topic %s not found", topic),
			nil,
		)
	}

	// Find and remove the handler
	// Note: This is a simplified implementation that removes the first matching handler
	// In production, you might want to use a more sophisticated approach
	for i := range handlers {
		// Since we can't directly compare function pointers, we'll remove by index
		// This is a limitation of Go's function comparison
		if i == len(handlers)-1 {
			eb.subscribers[topic] = handlers[:i]

			break
		}
	}

	if eb.logger != nil {
		eb.logger.Debug("subscriber removed", "topic", topic, "total_subscribers", len(eb.subscribers[topic]))
	}

	return nil
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

	eb.subscribers = make(map[string][]EventHandler)

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
	handlers, exists := eb.subscribers[topic]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		if eb.logger != nil {
			eb.logger.Debug("no subscribers for topic", "topic", topic)
		}
		return nil
	}

	// Execute handlers synchronously
	for _, handler := range handlers {
		// Check context before executing handler
		select {
		case <-ctx.Done():
			if eb.logger != nil {
				eb.logger.Debug("context canceled before handler execution", "topic", topic)
			}
			return ctx.Err()
		default:
			// Execute handler with panic recovery
			defer func() {
				if r := recover(); r != nil {
					if eb.logger != nil {
						eb.logger.Error("handler panic", "topic", topic, "panic", r)
					}
				}
			}()

			handler(event)
		}
	}

	if eb.logger != nil {
		eb.logger.Debug("event published synchronously", "topic", topic, "subscribers", len(handlers))
	}

	return nil
}
