package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockMessageQueue is a mock implementation of a message queue for testing
type MockMessageQueue struct {
	mu             sync.RWMutex
	messages       map[string][][]byte
	subscribers    map[string][]func([]byte)
	calls          map[string]int
	errors         map[string]error
	failNext       map[string]bool
	publishCount   map[string]int64
	subscribeCount map[string]int64
}

// NewMockMessageQueue creates a new mock message queue
func NewMockMessageQueue() *MockMessageQueue {
	return &MockMessageQueue{
		messages:       make(map[string][][]byte),
		subscribers:    make(map[string][]func([]byte)),
		calls:          make(map[string]int),
		errors:         make(map[string]error),
		failNext:       make(map[string]bool),
		publishCount:   make(map[string]int64),
		subscribeCount: make(map[string]int64),
	}
}

// Publish publishes a message to a topic
func (mmq *MockMessageQueue) Publish(ctx context.Context, topic string, message []byte) error {
	mmq.mu.Lock()
	defer mmq.mu.Unlock()

	mmq.calls["Publish"]++
	mmq.publishCount[topic]++

	if mmq.failNext["Publish"] {
		mmq.failNext["Publish"] = false
		return fmt.Errorf("publish failed")
	}

	if err, exists := mmq.errors["Publish"]; exists {
		return err
	}

	mmq.messages[topic] = append(mmq.messages[topic], message)

	// Notify subscribers
	if handlers, exists := mmq.subscribers[topic]; exists {
		for _, handler := range handlers {
			go handler(message)
		}
	}

	return nil
}

// Subscribe subscribes to a topic
func (mmq *MockMessageQueue) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	mmq.mu.Lock()
	defer mmq.mu.Unlock()

	mmq.calls["Subscribe"]++
	mmq.subscribeCount[topic]++

	if mmq.failNext["Subscribe"] {
		mmq.failNext["Subscribe"] = false
		return fmt.Errorf("subscribe failed")
	}

	if err, exists := mmq.errors["Subscribe"]; exists {
		return err
	}

	mmq.subscribers[topic] = append(mmq.subscribers[topic], handler)

	return nil
}

// Unsubscribe unsubscribes from a topic
func (mmq *MockMessageQueue) Unsubscribe(ctx context.Context, topic string) error {
	mmq.mu.Lock()
	defer mmq.mu.Unlock()

	mmq.calls["Unsubscribe"]++

	if mmq.failNext["Unsubscribe"] {
		mmq.failNext["Unsubscribe"] = false
		return fmt.Errorf("unsubscribe failed")
	}

	if err, exists := mmq.errors["Unsubscribe"]; exists {
		return err
	}

	delete(mmq.subscribers, topic)

	return nil
}

// GetQueueDepth returns the depth of a queue
func (mmq *MockMessageQueue) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	mmq.calls["GetQueueDepth"]++

	if mmq.failNext["GetQueueDepth"] {
		mmq.failNext["GetQueueDepth"] = false
		return 0, fmt.Errorf("get queue depth failed")
	}

	if err, exists := mmq.errors["GetQueueDepth"]; exists {
		return 0, err
	}

	return int64(len(mmq.messages[topic])), nil
}

// GetPublishedMessages returns all messages published to a topic
func (mmq *MockMessageQueue) GetPublishedMessages(topic string) [][]byte {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	messages := mmq.messages[topic]
	result := make([][]byte, len(messages))
	copy(result, messages)

	return result
}

// GetPublishedMessageCount returns the count of messages published to a topic
func (mmq *MockMessageQueue) GetPublishedMessageCount(topic string) int64 {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	return mmq.publishCount[topic]
}

// GetSubscriberCount returns the count of subscribers for a topic
func (mmq *MockMessageQueue) GetSubscriberCount(topic string) int {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	return len(mmq.subscribers[topic])
}

// GetCallCount returns the number of times a method was called
func (mmq *MockMessageQueue) GetCallCount(method string) int {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	return mmq.calls[method]
}

// SetError sets an error to be returned by a method
func (mmq *MockMessageQueue) SetError(method string, err error) {
	mmq.mu.Lock()
	defer mmq.mu.Unlock()

	mmq.errors[method] = err
}

// FailNext causes the next call to a method to fail
func (mmq *MockMessageQueue) FailNext(method string) {
	mmq.mu.Lock()
	defer mmq.mu.Unlock()

	mmq.failNext[method] = true
}

// Clear clears all messages and subscribers
func (mmq *MockMessageQueue) Clear() {
	mmq.mu.Lock()
	defer mmq.mu.Unlock()

	mmq.messages = make(map[string][][]byte)
	mmq.subscribers = make(map[string][]func([]byte))
	mmq.calls = make(map[string]int)
	mmq.publishCount = make(map[string]int64)
	mmq.subscribeCount = make(map[string]int64)
}

// WaitForMessage waits for a message to be published to a topic
func (mmq *MockMessageQueue) WaitForMessage(ctx context.Context, topic string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for message on topic %s", topic)
		case <-ticker.C:
			messages := mmq.GetPublishedMessages(topic)
			if len(messages) > 0 {
				return messages[len(messages)-1], nil
			}
		}
	}
}

// WaitForMessageCount waits for a specific number of messages to be published to a topic
func (mmq *MockMessageQueue) WaitForMessageCount(ctx context.Context, topic string, expectedCount int64, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			currentCount := mmq.GetPublishedMessageCount(topic)
			return fmt.Errorf("timeout waiting for %d messages on topic %s, got %d", expectedCount, topic, currentCount)
		case <-ticker.C:
			if mmq.GetPublishedMessageCount(topic) >= expectedCount {
				return nil
			}
		}
	}
}

// PublishMultiple publishes multiple messages to a topic
func (mmq *MockMessageQueue) PublishMultiple(ctx context.Context, topic string, messages [][]byte) error {
	for _, message := range messages {
		if err := mmq.Publish(ctx, topic, message); err != nil {
			return err
		}
	}
	return nil
}

// GetAllTopics returns all topics with messages
func (mmq *MockMessageQueue) GetAllTopics() []string {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	topics := make([]string, 0, len(mmq.messages))
	for topic := range mmq.messages {
		topics = append(topics, topic)
	}

	return topics
}

// GetTotalPublishedCount returns the total number of messages published across all topics
func (mmq *MockMessageQueue) GetTotalPublishedCount() int64 {
	mmq.mu.RLock()
	defer mmq.mu.RUnlock()

	total := int64(0)
	for _, count := range mmq.publishCount {
		total += count
	}

	return total
}
