package core

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// EventBusTestLogger for testing
type EventBusTestLogger struct {
	messages []string
	mu       sync.Mutex
}

func (ml *EventBusTestLogger) Debug(msg string, fields ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Info(msg string, fields ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Warn(msg string, fields ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Error(msg string, fields ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Fatal(msg string, fields ...interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) WithCorrelationID(id string) Logger {
	return ml
}

// TestNewEventBus tests event bus creation
func TestNewEventBus(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.subscribers)
	assert.Equal(t, 0, len(eb.subscribers))
	assert.Equal(t, logger, eb.logger)
}

// TestNewEventBusWithNilLogger tests event bus creation with nil logger
func TestNewEventBusWithNilLogger(t *testing.T) {
	eb := NewEventBus(nil)

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.subscribers)
	assert.Nil(t, eb.logger)
}

// TestSubscribe tests basic subscription
func TestSubscribe(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event interface{}) {}
	err := eb.Subscribe(ctx, "test-topic", handler)

	assert.NoError(t, err)
	assert.Equal(t, 1, eb.GetSubscriberCount("test-topic"))
}

// TestMultipleSubscribers tests multiple subscribers to same topic
func TestMultipleSubscribers(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler1 := func(event interface{}) {}
	handler2 := func(event interface{}) {}
	handler3 := func(event interface{}) {}

	_ = eb.Subscribe(ctx, "test-topic", handler1)
	_ = eb.Subscribe(ctx, "test-topic", handler2)
	_ = eb.Subscribe(ctx, "test-topic", handler3)

	assert.Equal(t, 3, eb.GetSubscriberCount("test-topic"))
}

// TestSubscribeEmptyTopic tests subscription with empty topic
func TestSubscribeEmptyTopic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event interface{}) {}
	err := eb.Subscribe(ctx, "", handler)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestSubscribeNilHandler tests subscription with nil handler
func TestSubscribeNilHandler(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.Subscribe(ctx, "test-topic", nil)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestPublish tests basic event publishing
func TestPublish(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	eventReceived := false
	handler := func(event interface{}) {
		eventReceived = true
	}

	_ = eb.Subscribe(ctx, "test-topic", handler)
	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
	assert.True(t, eventReceived)
}

// TestPublishEmptyTopic tests publishing to empty topic
func TestPublishEmptyTopic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.Publish(ctx, "", "test-event")

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestPublishNilEvent tests publishing nil event
func TestPublishNilEvent(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.Publish(ctx, "test-topic", nil)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestPublishNoSubscribers tests publishing with no subscribers
func TestPublishNoSubscribers(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
}

// TestPublishMultipleSubscribers tests publishing to multiple subscribers
func TestPublishMultipleSubscribers(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter int32
	handler1 := func(event interface{}) {
		atomic.AddInt32(&counter, 1)
	}
	handler2 := func(event interface{}) {
		atomic.AddInt32(&counter, 1)
	}
	handler3 := func(event interface{}) {
		atomic.AddInt32(&counter, 1)
	}

	_ = eb.Subscribe(ctx, "test-topic", handler1)
	_ = eb.Subscribe(ctx, "test-topic", handler2)
	_ = eb.Subscribe(ctx, "test-topic", handler3)

	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(3), atomic.LoadInt32(&counter))
}

// TestPublishContextCanceled tests publishing with canceled context
func TestPublishContextCanceled(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	eventReceived := false
	handler := func(event interface{}) {
		eventReceived = true
	}

	ctx := context.Background()
	_ = eb.Subscribe(ctx, "test-topic", handler)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := eb.Publish(canceledCtx, "test-topic", "test-event")

	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	// Handler should not be called due to canceled context
	assert.False(t, eventReceived)
}

// TestPublishHandlerPanic tests publishing with handler that panics
func TestPublishHandlerPanic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	panicHandler := func(event interface{}) {
		panic("test panic")
	}

	normalHandler := func(event interface{}) {
		// This should still be called
	}

	_ = eb.Subscribe(ctx, "test-topic", panicHandler)
	_ = eb.Subscribe(ctx, "test-topic", normalHandler)

	// Should not panic
	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}

// TestUnsubscribe tests unsubscribing from topic
func TestUnsubscribe(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event interface{}) {}
	_ = eb.Subscribe(ctx, "test-topic", handler)
	assert.Equal(t, 1, eb.GetSubscriberCount("test-topic"))

	err := eb.Unsubscribe("test-topic", handler)

	assert.NoError(t, err)
	assert.Equal(t, 0, eb.GetSubscriberCount("test-topic"))
}

// TestUnsubscribeEmptyTopic tests unsubscribing with empty topic
func TestUnsubscribeEmptyTopic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	handler := func(event interface{}) {}
	err := eb.Unsubscribe("", handler)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestUnsubscribeNilHandler tests unsubscribing with nil handler
func TestUnsubscribeNilHandler(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	err := eb.Unsubscribe("test-topic", nil)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestUnsubscribeNonexistentTopic tests unsubscribing from non-existent topic
func TestUnsubscribeNonexistentTopic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	handler := func(event interface{}) {}
	err := eb.Unsubscribe("nonexistent-topic", handler)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeNotFound, sysErr.Code)
}

// TestGetSubscriberCount tests getting subscriber count
func TestGetSubscriberCount(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	assert.Equal(t, 0, eb.GetSubscriberCount("test-topic"))

	handler1 := func(event interface{}) {}
	handler2 := func(event interface{}) {}

	_ = eb.Subscribe(ctx, "test-topic", handler1)
	assert.Equal(t, 1, eb.GetSubscriberCount("test-topic"))

	_ = eb.Subscribe(ctx, "test-topic", handler2)
	assert.Equal(t, 2, eb.GetSubscriberCount("test-topic"))
}

// TestGetSubscriberCountNonexistentTopic tests getting count for non-existent topic
func TestGetSubscriberCountNonexistentTopic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	assert.Equal(t, 0, eb.GetSubscriberCount("nonexistent-topic"))
}

// TestGetTopics tests getting all topics
func TestGetTopics(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event interface{}) {}

	_ = eb.Subscribe(ctx, "topic1", handler)
	_ = eb.Subscribe(ctx, "topic2", handler)
	_ = eb.Subscribe(ctx, "topic3", handler)

	topics := eb.GetTopics()

	assert.Equal(t, 3, len(topics))
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
	assert.Contains(t, topics, "topic3")
}

// TestGetTopicsEmpty tests getting topics when none exist
func TestGetTopicsEmpty(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	topics := eb.GetTopics()

	assert.Equal(t, 0, len(topics))
}

// TestClearEventBus tests clearing all subscribers
func TestClearEventBus(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event interface{}) {}

	_ = eb.Subscribe(ctx, "topic1", handler)
	_ = eb.Subscribe(ctx, "topic2", handler)

	assert.Equal(t, 2, len(eb.GetTopics()))

	eb.Clear()

	assert.Equal(t, 0, len(eb.GetTopics()))
	assert.Equal(t, 0, eb.GetSubscriberCount("topic1"))
	assert.Equal(t, 0, eb.GetSubscriberCount("topic2"))
}

// TestPublishSync tests synchronous publishing
func TestPublishSync(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	eventReceived := false
	handler := func(event interface{}) {
		eventReceived = true
	}

	_ = eb.Subscribe(ctx, "test-topic", handler)
	err := eb.PublishSync(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	// Synchronous, so should be received immediately
	assert.True(t, eventReceived)
}

// TestPublishSyncEmptyTopic tests synchronous publishing with empty topic
func TestPublishSyncEmptyTopic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.PublishSync(ctx, "", "test-event")

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestPublishSyncNilEvent tests synchronous publishing with nil event
func TestPublishSyncNilEvent(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.PublishSync(ctx, "test-topic", nil)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestPublishSyncContextCanceled tests synchronous publishing with canceled context
func TestPublishSyncContextCanceled(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	eventReceived := false
	handler := func(event interface{}) {
		eventReceived = true
	}

	ctx := context.Background()
	_ = eb.Subscribe(ctx, "test-topic", handler)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := eb.PublishSync(canceledCtx, "test-topic", "test-event")

	assert.Error(t, err)
	assert.False(t, eventReceived)
}

// TestPublishSyncMultipleSubscribers tests synchronous publishing to multiple subscribers
func TestPublishSyncMultipleSubscribers(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter int32
	handler1 := func(event interface{}) {
		atomic.AddInt32(&counter, 1)
	}
	handler2 := func(event interface{}) {
		atomic.AddInt32(&counter, 1)
	}
	handler3 := func(event interface{}) {
		atomic.AddInt32(&counter, 1)
	}

	_ = eb.Subscribe(ctx, "test-topic", handler1)
	_ = eb.Subscribe(ctx, "test-topic", handler2)
	_ = eb.Subscribe(ctx, "test-topic", handler3)

	err := eb.PublishSync(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&counter))
}

// TestPublishSyncHandlerPanic tests synchronous publishing with handler that panics
func TestPublishSyncHandlerPanic(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	panicHandler := func(event interface{}) {
		panic("test panic")
	}

	_ = eb.Subscribe(ctx, "test-topic", panicHandler)

	// Should not panic
	err := eb.PublishSync(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
}

// TestConcurrentSubscribePublish tests concurrent subscribe and publish operations
func TestConcurrentSubscribePublish(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter int32
	var wg sync.WaitGroup

	// Subscribe from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler := func(event interface{}) {
				atomic.AddInt32(&counter, 1)
			}
			_ = eb.Subscribe(ctx, "test-topic", handler)
		}()
	}

	wg.Wait()

	// Publish from multiple goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = eb.Publish(ctx, "test-topic", "test-event")
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Should have 10 subscribers, each called 5 times = 50
	assert.Equal(t, int32(50), atomic.LoadInt32(&counter))
}

// TestEventDataTypes tests publishing different event data types
func TestEventDataTypes(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	tests := []struct {
		name  string
		event interface{}
	}{
		{"String", "test-string"},
		{"Integer", 42},
		{"Float", 3.14},
		{"Boolean", true},
		{"Struct", struct{ Name string }{Name: "test"}},
		{"Map", map[string]interface{}{"key": "value"}},
		{"Slice", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedEvent interface{}
			handler := func(event interface{}) {
				receivedEvent = event
			}

			_ = eb.Subscribe(ctx, "test-topic", handler)
			err := eb.Publish(ctx, "test-topic", tt.event)

			assert.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
			assert.Equal(t, tt.event, receivedEvent)

			eb.Clear()
		})
	}
}

// TestMultipleTopics tests publishing to multiple topics
func TestMultipleTopics(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter1, counter2, counter3 int32

	handler1 := func(event interface{}) {
		atomic.AddInt32(&counter1, 1)
	}
	handler2 := func(event interface{}) {
		atomic.AddInt32(&counter2, 1)
	}
	handler3 := func(event interface{}) {
		atomic.AddInt32(&counter3, 1)
	}

	_ = eb.Subscribe(ctx, "topic1", handler1)
	_ = eb.Subscribe(ctx, "topic2", handler2)
	_ = eb.Subscribe(ctx, "topic3", handler3)

	_ = eb.Publish(ctx, "topic1", "event1")
	_ = eb.Publish(ctx, "topic2", "event2")
	_ = eb.Publish(ctx, "topic3", "event3")

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&counter1))
	assert.Equal(t, int32(1), atomic.LoadInt32(&counter2))
	assert.Equal(t, int32(1), atomic.LoadInt32(&counter3))
}

// TestSubscriberConsistency tests that subscriber count remains consistent
func TestSubscriberConsistency(t *testing.T) {
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event interface{}) {}

	// Add subscribers
	for i := 0; i < 100; i++ {
		_ = eb.Subscribe(ctx, "test-topic", handler)
	}

	count := eb.GetSubscriberCount("test-topic")
	assert.Equal(t, 100, count)

	// Verify count is consistent across multiple calls
	for i := 0; i < 10; i++ {
		assert.Equal(t, 100, eb.GetSubscriberCount("test-topic"))
	}
}
