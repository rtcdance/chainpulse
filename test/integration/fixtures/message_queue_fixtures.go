package fixtures

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// MessageQueueFixture provides message queue setup and teardown for integration tests
type MessageQueueFixture struct {
	mq       core.MQPlugin
	t        *testing.T
	messages map[string][][]byte
	mu       sync.RWMutex
}

// NewMessageQueueFixture creates a new message queue fixture
func NewMessageQueueFixture(t *testing.T, mq core.MQPlugin) *MessageQueueFixture {
	return &MessageQueueFixture{
		mq:       mq,
		t:        t,
		messages: make(map[string][][]byte),
	}
}

// Setup initializes the message queue for testing
func (f *MessageQueueFixture) Setup(ctx context.Context) error {
	if f.mq == nil {
		return fmt.Errorf("message queue plugin is nil")
	}

	// Verify MQ health
	if err := f.mq.Health(); err != nil {
		return fmt.Errorf("message queue health check failed: %w", err)
	}

	// Clear any existing messages
	f.mu.Lock()
	f.messages = make(map[string][][]byte)
	f.mu.Unlock()

	return nil
}

// Cleanup cleans up the message queue after testing
func (f *MessageQueueFixture) Cleanup(ctx context.Context) error {
	if f.mq == nil {
		return nil
	}

	// Clear messages
	f.mu.Lock()
	f.messages = make(map[string][][]byte)
	f.mu.Unlock()

	return nil
}

// Close closes the message queue connection
func (f *MessageQueueFixture) Close() error {
	if f.mq != nil {
		return f.mq.Stop()
	}
	return nil
}

// Publish publishes a message to a topic
func (f *MessageQueueFixture) Publish(ctx context.Context, topic string, message []byte) error {
	if err := f.mq.Publish(ctx, topic, message); err != nil {
		return fmt.Errorf("failed to publish message to topic %s: %w", topic, err)
	}

	// Track message for testing
	f.mu.Lock()
	f.messages[topic] = append(f.messages[topic], message)
	f.mu.Unlock()

	return nil
}

// Subscribe subscribes to a topic with a handler
func (f *MessageQueueFixture) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	if err := f.mq.Subscribe(ctx, topic, handler); err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}
	return nil
}

// GetQueueDepth returns the depth of a queue
func (f *MessageQueueFixture) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	depth, err := f.mq.GetQueueDepth(ctx, topic)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue depth for topic %s: %w", topic, err)
	}
	return depth, nil
}

// GetPublishedMessages returns all messages published to a topic during the test
func (f *MessageQueueFixture) GetPublishedMessages(topic string) [][]byte {
	f.mu.RLock()
	defer f.mu.RUnlock()

	messages := f.messages[topic]
	// Return a copy to prevent external modification
	result := make([][]byte, len(messages))
	copy(result, messages)
	return result
}

// GetPublishedMessageCount returns the count of messages published to a topic
func (f *MessageQueueFixture) GetPublishedMessageCount(topic string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.messages[topic])
}

// ClearPublishedMessages clears the tracked messages for a topic
func (f *MessageQueueFixture) ClearPublishedMessages(topic string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages[topic] = [][]byte{}
}

// WaitForMessage waits for a message to be published to a topic
func (f *MessageQueueFixture) WaitForMessage(ctx context.Context, topic string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for message on topic %s", topic)
		case <-ticker.C:
			messages := f.GetPublishedMessages(topic)
			if len(messages) > 0 {
				return messages[len(messages)-1], nil
			}
		}
	}
}

// WaitForMessageCount waits for a specific number of messages to be published to a topic
func (f *MessageQueueFixture) WaitForMessageCount(ctx context.Context, topic string, expectedCount int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			currentCount := f.GetPublishedMessageCount(topic)
			return fmt.Errorf("timeout waiting for %d messages on topic %s, got %d", expectedCount, topic, currentCount)
		case <-ticker.C:
			if f.GetPublishedMessageCount(topic) >= expectedCount {
				return nil
			}
		}
	}
}

// PublishMultiple publishes multiple messages to a topic
func (f *MessageQueueFixture) PublishMultiple(ctx context.Context, topic string, messages [][]byte) error {
	for _, message := range messages {
		if err := f.Publish(ctx, topic, message); err != nil {
			return err
		}
	}
	return nil
}

// WithTimeout returns a context with timeout
func (f *MessageQueueFixture) WithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
