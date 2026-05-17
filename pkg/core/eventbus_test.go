package core

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func skipEventBusStressTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping event bus concurrency/publish stress test in short mode")
	}
}

// EventBusTestLogger for testing
type EventBusTestLogger struct {
	messages []string
	mu       sync.Mutex
}

func (ml *EventBusTestLogger) Debug(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Info(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Warn(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Error(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) Fatal(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *EventBusTestLogger) WithCorrelationID(id string) Logger {
	return ml
}

// TestNewEventBus tests event bus creation
func TestNewEventBus(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.subscribers)
	assert.Equal(t, 0, len(eb.subscribers))
	assert.Equal(t, logger, eb.logger)
}

// TestNewEventBusWithNilLogger tests event bus creation with nil logger
func TestNewEventBusWithNilLogger(t *testing.T) {
	t.Parallel()
	eb := NewEventBus(nil)

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.subscribers)
	assert.Nil(t, eb.logger)
}

// TestSubscribe tests basic subscription returns a subscription ID
func TestSubscribe(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}
	subID, err := eb.Subscribe(ctx, "test-topic", handler)

	assert.NoError(t, err)
	assert.Equal(t, uint64(1), subID)
	assert.Equal(t, 1, eb.GetSubscriberCount("test-topic"))
}

// TestSubscribeReturnsIncrementingIDs tests that subscription IDs are unique and incrementing
func TestSubscribeReturnsIncrementingIDs(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}
	subID1, _ := eb.Subscribe(ctx, "test-topic", handler)
	subID2, _ := eb.Subscribe(ctx, "test-topic", handler)
	subID3, _ := eb.Subscribe(ctx, "other-topic", handler)

	assert.Equal(t, uint64(1), subID1)
	assert.Equal(t, uint64(2), subID2)
	assert.Equal(t, uint64(3), subID3)
}

// TestMultipleSubscribers tests multiple subscribers to same topic
func TestMultipleSubscribers(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler1 := func(event any) {}
	handler2 := func(event any) {}
	handler3 := func(event any) {}

	_, _ = eb.Subscribe(ctx, "test-topic", handler1)
	_, _ = eb.Subscribe(ctx, "test-topic", handler2)
	_, _ = eb.Subscribe(ctx, "test-topic", handler3)

	assert.Equal(t, 3, eb.GetSubscriberCount("test-topic"))
}

// TestSubscribeEmptyTopic tests subscription with empty topic
func TestSubscribeEmptyTopic(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}
	_, err := eb.Subscribe(ctx, "", handler)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestSubscribeNilHandler tests subscription with nil handler
func TestSubscribeNilHandler(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	_, err := eb.Subscribe(ctx, "test-topic", nil)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestPublish tests basic event publishing
func TestPublish(t *testing.T) {
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var eventReceived atomic.Bool
	handler := func(event any) {
		eventReceived.Store(true)
	}

	_, _ = eb.Subscribe(ctx, "test-topic", handler)
	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
	assert.True(t, eventReceived.Load())
}

// TestPublishEmptyTopic tests publishing to empty topic
func TestPublishEmptyTopic(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
}

// TestPublishMultipleSubscribers tests publishing to multiple subscribers
func TestPublishMultipleSubscribers(t *testing.T) {
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter int32
	handler1 := func(event any) {
		atomic.AddInt32(&counter, 1)
	}
	handler2 := func(event any) {
		atomic.AddInt32(&counter, 1)
	}
	handler3 := func(event any) {
		atomic.AddInt32(&counter, 1)
	}

	_, _ = eb.Subscribe(ctx, "test-topic", handler1)
	_, _ = eb.Subscribe(ctx, "test-topic", handler2)
	_, _ = eb.Subscribe(ctx, "test-topic", handler3)

	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(3), atomic.LoadInt32(&counter))
}

// TestPublishContextCanceled tests publishing with canceled context
func TestPublishContextCanceled(t *testing.T) {
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	eventReceived := false
	handler := func(event any) {
		eventReceived = true
	}

	ctx := context.Background()
	_, _ = eb.Subscribe(ctx, "test-topic", handler)

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
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	panicHandler := func(event any) {
		panic("test panic")
	}

	normalHandler := func(event any) {
		// This should still be called
	}

	_, _ = eb.Subscribe(ctx, "test-topic", panicHandler)
	_, _ = eb.Subscribe(ctx, "test-topic", normalHandler)

	// Should not panic
	err := eb.Publish(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
}

// TestUnsubscribe tests unsubscribing by subscription ID
func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}
	subID, _ := eb.Subscribe(ctx, "test-topic", handler)
	assert.Equal(t, 1, eb.GetSubscriberCount("test-topic"))

	err := eb.Unsubscribe(subID)

	assert.NoError(t, err)
	assert.Equal(t, 0, eb.GetSubscriberCount("test-topic"))
}

// TestUnsubscribeRemovesCorrectHandler tests that Unsubscribe removes only the targeted handler
func TestUnsubscribeRemovesCorrectHandler(t *testing.T) {
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var callLog []string
	var mu sync.Mutex

	handler1 := func(event any) {
		mu.Lock()
		callLog = append(callLog, "handler1")
		mu.Unlock()
	}
	handler2 := func(event any) {
		mu.Lock()
		callLog = append(callLog, "handler2")
		mu.Unlock()
	}
	handler3 := func(event any) {
		mu.Lock()
		callLog = append(callLog, "handler3")
		mu.Unlock()
	}

	_, _ = eb.Subscribe(ctx, "test-topic", handler1)
	subID2, _ := eb.Subscribe(ctx, "test-topic", handler2)
	_, _ = eb.Subscribe(ctx, "test-topic", handler3)

	// Unsubscribe the middle handler (handler2)
	err := eb.Unsubscribe(subID2)
	assert.NoError(t, err)
	assert.Equal(t, 2, eb.GetSubscriberCount("test-topic"))

	// Publish and verify only handler1 and handler3 are called
	_ = eb.Publish(ctx, "test-topic", "test-event")
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 2, len(callLog))
	assert.Contains(t, callLog, "handler1")
	assert.Contains(t, callLog, "handler3")
	assert.NotContains(t, callLog, "handler2")
	mu.Unlock()
}

// TestUnsubscribeNonexistentID tests unsubscribing with a non-existent subscription ID
func TestUnsubscribeNonexistentID(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	err := eb.Unsubscribe(999)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeNotFound, sysErr.Code)
}

// TestUnsubscribeTwice tests that double unsubscribe returns an error
func TestUnsubscribeTwice(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}
	subID, _ := eb.Subscribe(ctx, "test-topic", handler)

	err := eb.Unsubscribe(subID)
	assert.NoError(t, err)

	// Second unsubscribe should fail
	err = eb.Unsubscribe(subID)
	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeNotFound, sysErr.Code)
}

// TestGetSubscriberCount tests getting subscriber count
func TestGetSubscriberCount(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	assert.Equal(t, 0, eb.GetSubscriberCount("test-topic"))

	handler1 := func(event any) {}
	handler2 := func(event any) {}

	_, _ = eb.Subscribe(ctx, "test-topic", handler1)
	assert.Equal(t, 1, eb.GetSubscriberCount("test-topic"))

	_, _ = eb.Subscribe(ctx, "test-topic", handler2)
	assert.Equal(t, 2, eb.GetSubscriberCount("test-topic"))
}

// TestGetSubscriberCountNonexistentTopic tests getting count for non-existent topic
func TestGetSubscriberCountNonexistentTopic(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	assert.Equal(t, 0, eb.GetSubscriberCount("nonexistent-topic"))
}

// TestGetTopics tests getting all topics
func TestGetTopics(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}

	_, _ = eb.Subscribe(ctx, "topic1", handler)
	_, _ = eb.Subscribe(ctx, "topic2", handler)
	_, _ = eb.Subscribe(ctx, "topic3", handler)

	topics := eb.GetTopics()

	assert.Equal(t, 3, len(topics))
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
	assert.Contains(t, topics, "topic3")
}

// TestGetTopicsEmpty tests getting topics when none exist
func TestGetTopicsEmpty(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	topics := eb.GetTopics()

	assert.Equal(t, 0, len(topics))
}

// TestClearEventBus tests clearing all subscribers
func TestClearEventBus(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}

	_, _ = eb.Subscribe(ctx, "topic1", handler)
	_, _ = eb.Subscribe(ctx, "topic2", handler)

	assert.Equal(t, 2, len(eb.GetTopics()))

	eb.Clear()

	assert.Equal(t, 0, len(eb.GetTopics()))
	assert.Equal(t, 0, eb.GetSubscriberCount("topic1"))
	assert.Equal(t, 0, eb.GetSubscriberCount("topic2"))
}

// TestPublishSync tests synchronous publishing
func TestPublishSync(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	eventReceived := false
	handler := func(event any) {
		eventReceived = true
	}

	_, _ = eb.Subscribe(ctx, "test-topic", handler)
	err := eb.PublishSync(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	// Synchronous, so should be received immediately
	assert.True(t, eventReceived)
}

// TestPublishSyncEmptyTopic tests synchronous publishing with empty topic
func TestPublishSyncEmptyTopic(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)

	eventReceived := false
	handler := func(event any) {
		eventReceived = true
	}

	ctx := context.Background()
	_, _ = eb.Subscribe(ctx, "test-topic", handler)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := eb.PublishSync(canceledCtx, "test-topic", "test-event")

	assert.Error(t, err)
	assert.False(t, eventReceived)
}

// TestPublishSyncMultipleSubscribers tests synchronous publishing to multiple subscribers
func TestPublishSyncMultipleSubscribers(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter int32
	handler1 := func(event any) {
		atomic.AddInt32(&counter, 1)
	}
	handler2 := func(event any) {
		atomic.AddInt32(&counter, 1)
	}
	handler3 := func(event any) {
		atomic.AddInt32(&counter, 1)
	}

	_, _ = eb.Subscribe(ctx, "test-topic", handler1)
	_, _ = eb.Subscribe(ctx, "test-topic", handler2)
	_, _ = eb.Subscribe(ctx, "test-topic", handler3)

	err := eb.PublishSync(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&counter))
}

// TestPublishSyncHandlerPanic tests synchronous publishing with handler that panics
func TestPublishSyncHandlerPanic(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	panicHandler := func(event any) {
		panic("test panic")
	}

	_, _ = eb.Subscribe(ctx, "test-topic", panicHandler)

	// Should not panic
	err := eb.PublishSync(ctx, "test-topic", "test-event")

	assert.NoError(t, err)
}

func TestPublishSyncPanicDoesNotSkipSubsequentHandlers(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var handler2Called bool

	// Handler 1 panics, handler 2 should still execute
	panicHandler := func(event any) {
		panic("handler1 panic")
	}
	normalHandler := func(event any) {
		handler2Called = true
	}

	_, _ = eb.Subscribe(ctx, "test-topic", panicHandler)
	_, _ = eb.Subscribe(ctx, "test-topic", normalHandler)

	err := eb.PublishSync(ctx, "test-topic", "test-event")
	assert.NoError(t, err)

	if !handler2Called {
		t.Error("handler 2 was not called after handler 1 panicked — panic recovery must be per-handler")
	}
}

// TestConcurrentSubscribePublish tests concurrent subscribe and publish operations
func TestConcurrentSubscribePublish(t *testing.T) {
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

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
			handler := func(event any) {
				atomic.AddInt32(&counter, 1)
			}
			_, _ = eb.Subscribe(ctx, "test-topic", handler)
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
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	tests := []struct {
		name  string
		event any
	}{
		{"String", "test-string"},
		{"Integer", 42},
		{"Float", 3.14},
		{"Boolean", true},
		{"Struct", struct{ Name string }{Name: "test"}},
		{"Map", map[string]any{"key": "value"}},
		{"Slice", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedEvent atomic.Pointer[any]
			handler := func(event any) {
				receivedEvent.Store(&event)
			}

			_, _ = eb.Subscribe(ctx, "test-topic", handler)
			err := eb.Publish(ctx, "test-topic", tt.event)

			assert.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
			stored := receivedEvent.Load()
			if stored != nil {
				assert.Equal(t, tt.event, *stored)
			} else {
				t.Fatal("expected event to be received but got nil")
			}

			eb.Clear()
		})
	}
}

// TestMultipleTopics tests publishing to multiple topics
func TestMultipleTopics(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	var counter1, counter2, counter3 int32

	handler1 := func(event any) {
		atomic.AddInt32(&counter1, 1)
	}
	handler2 := func(event any) {
		atomic.AddInt32(&counter2, 1)
	}
	handler3 := func(event any) {
		atomic.AddInt32(&counter3, 1)
	}

	_, _ = eb.Subscribe(ctx, "topic1", handler1)
	_, _ = eb.Subscribe(ctx, "topic2", handler2)
	_, _ = eb.Subscribe(ctx, "topic3", handler3)

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
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	handler := func(event any) {}

	// Add subscribers
	for i := 0; i < 100; i++ {
		_, _ = eb.Subscribe(ctx, "test-topic", handler)
	}

	count := eb.GetSubscriberCount("test-topic")
	assert.Equal(t, 100, count)

	// Verify count is consistent across multiple calls
	for i := 0; i < 10; i++ {
		assert.Equal(t, 100, eb.GetSubscriberCount("test-topic"))
	}
}

// TestEventBusStop tests that Stop prevents new publications and waits for in-flight handlers
func TestEventBusStop(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	ctx := context.Background()

	// Publish an event with a slow handler to verify Wait behavior
	var handlerCalled atomic.Bool
	handler := func(event any) {
		time.Sleep(50 * time.Millisecond)
		handlerCalled.Store(true)
	}
	_, _ = eb.Subscribe(ctx, "test-topic", handler)

	_ = eb.Publish(ctx, "test-topic", "test-event")
	eb.Stop()

	// Handler should have completed
	assert.True(t, handlerCalled.Load())

	// Publish after Stop should return error
	err := eb.Publish(ctx, "test-topic", "another-event")
	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)

	// Stop is idempotent
	eb.Stop()
}

// TestPublishConcurrentSubscribeNoRace verifies that concurrent Publish and
// Subscribe calls on the same topic do not produce a data race.
// This is a regression test for the map-reference-leak bug where Publish
// held a reference to the subscribers map after releasing RLock.
func TestPublishConcurrentSubscribeNoRace(t *testing.T) {
	t.Parallel()
	skipEventBusStressTestsInShortMode(t)

	eb := NewEventBus(nil)
	defer eb.Stop()

	ctx := context.Background()
	topic := "race-topic"

	// Pre-subscribe one handler so Publish has work to do
	var received atomic.Int32
	_, _ = eb.Subscribe(ctx, topic, func(_ any) {
		received.Add(1)
	})

	var wg sync.WaitGroup
	const iterations = 200

	// Concurrent publishers
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = eb.Publish(ctx, topic, "event")
			}
		}()
	}

	// Concurrent subscribers adding/removing
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				subID, _ := eb.Subscribe(ctx, topic, func(_ any) {})
				if subID > 0 {
					_ = eb.Unsubscribe(subID)
				}
			}
		}(i)
	}

	wg.Wait()

	// Give in-flight goroutines time to complete
	time.Sleep(200 * time.Millisecond)
	eb.Wait()

	// If we reach here without a race detector failure, the test passes.
	assert.True(t, received.Load() > 0, "at least one event should have been received")
}

func TestSubscribeTyped(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	eb := NewEventBus(logger)
	defer eb.Stop()

	ctx := context.Background()

	// Test 1: Type match — handler receives the correct concrete type
	var received atomic.Pointer[BlockchainEvent]
	subID, err := SubscribeTyped[BlockchainEvent](eb, ctx, "blockchain-events", func(evt BlockchainEvent) {
		received.Store(&evt)
	})
	assert.NoError(t, err)
	assert.Greater(t, subID, uint64(0))

	event := BlockchainEvent{BlockNumber: 42, ChainID: "1"}
	err = eb.Publish(ctx, "blockchain-events", event)
	assert.NoError(t, err)

	eb.Wait()
	stored := received.Load()
	assert.NotNil(t, stored, "typed handler should have received the event")
	if stored != nil {
		assert.Equal(t, uint64(42), stored.BlockNumber)
		assert.Equal(t, "1", stored.ChainID)
	}

	// Test 2: Type mismatch — handler is silently skipped
	var stringReceived atomic.Bool
	_, err = SubscribeTyped[string](eb, ctx, "blockchain-events", func(s string) {
		stringReceived.Store(true)
	})
	assert.NoError(t, err)

	err = eb.Publish(ctx, "blockchain-events", BlockchainEvent{BlockNumber: 99})
	assert.NoError(t, err)

	eb.Wait()
	assert.False(t, stringReceived.Load(), "string handler should NOT be called for BlockchainEvent")

	// Test 3: Pointer type
	var ptrReceived atomic.Pointer[ReorgRollbackEvent]
	_, err = SubscribeTyped[*ReorgRollbackEvent](eb, ctx, "reorg-rollback", func(evt *ReorgRollbackEvent) {
		ptrReceived.Store(evt)
	})
	assert.NoError(t, err)

	reorgEvt := &ReorgRollbackEvent{ChainID: "1", FromBlock: 100, ToBlock: 110}
	err = eb.Publish(ctx, "reorg-rollback", reorgEvt)
	assert.NoError(t, err)

	eb.Wait()
	ptrStored := ptrReceived.Load()
	assert.NotNil(t, ptrStored, "typed handler should receive pointer event")
	if ptrStored != nil {
		assert.Equal(t, "1", ptrStored.ChainID)
		assert.Equal(t, uint64(100), ptrStored.FromBlock)
	}
}
