package mq

import (
	"context"
)

// MessageQueue interface defines the methods for message queue operations
type MessageQueue interface {
	Publish(topic string, message interface{}) error
	Consume(ctx context.Context, topic string, handler MessageHandler) error
	Close() error
}

// MessageHandler defines the function signature for handling messages
type MessageHandler func(message []byte) error
