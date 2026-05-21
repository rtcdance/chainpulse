package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Property 4: Event Publishing Consistency
// For any event published to a topic with subscribers,
// all subscribers should receive the event

// TestPropertyEventPublishingConsistency verifies event publishing consistency
func TestPropertyEventPublishingConsistency(t *testing.T) {
	tests := []struct {
		name              string
		subscriberCount   int
		eventCount        int
		expectedCallCount int
	}{
		{
			name:              "single subscriber single event",
			subscriberCount:   1,
			eventCount:        1,
			expectedCallCount: 1,
		},
		{
			name:              "multiple subscribers single event",
			subscriberCount:   5,
			eventCount:        1,
			expectedCallCount: 5,
		},
		{
			name:              "single subscriber multiple events",
			subscriberCount:   1,
			eventCount:        5,
			expectedCallCount: 5,
		},
		{
			name:              "multiple subscribers multiple events",
			subscriberCount:   3,
			eventCount:        3,
			expectedCallCount: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := NewEventBus(nil)
			count := 0
			mu := sync.Mutex{}

			// Subscribe handlers
			for i := 0; i < tt.subscriberCount; i++ {
		handler := func(_ context.Context, event any) error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		}
				if _, err := eb.Subscribe(context.Background(), "test-topic", handler); err != nil {
					t.Fatalf("failed to subscribe: %v", err)
				}
			}

			// Publish events
			for i := 0; i < tt.eventCount; i++ {
				if err := eb.Publish(context.Background(), "test-topic", fmt.Sprintf("event-%d", i)); err != nil {
					t.Fatalf("failed to publish: %v", err)
				}
			}

			// Give goroutines time to execute
			time.Sleep(200 * time.Millisecond)

			mu.Lock()
			defer mu.Unlock()
			if count != tt.expectedCallCount {
				t.Errorf("expected %d handler calls, got %d", tt.expectedCallCount, count)
			}
		})
	}
}

// TestPropertySubscriberCountConsistency verifies subscriber count consistency
func TestPropertySubscriberCountConsistency(t *testing.T) {
	eb := NewEventBus(nil)
	handler := func(_ context.Context, event any) error { return nil }

	// For any sequence of subscriptions, the subscriber count should match
	for i := 0; i < 10; i++ {
		if _, err := eb.Subscribe(context.Background(), "test-topic", handler); err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}

		if eb.GetSubscriberCount("test-topic") != i+1 {
			t.Errorf("expected %d subscribers, got %d", i+1, eb.GetSubscriberCount("test-topic"))
		}
	}
}

// TestPropertyEventDataPreservation verifies event data is preserved
func TestPropertyEventDataPreservation(t *testing.T) {
	eb := NewEventBus(nil)

	// For any event data, it should be preserved when published
	testCases := []struct {
		name  string
		event any
	}{
		{"string-event", "string-event"},
		{"int-event", 123},
		{"float-event", 45.67},
		{"bool-event", true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var receivedEvent any
		handler := func(_ context.Context, event any) error {
			receivedEvent = event
			return nil
		}

			eb.Clear()
			if _, err := eb.Subscribe(context.Background(), "test-topic", handler); err != nil {
				t.Fatalf("failed to subscribe: %v", err)
			}
			if err := eb.PublishSync(context.Background(), "test-topic", tt.event); err != nil {
				t.Fatalf("failed to publish: %v", err)
			}

			if receivedEvent != tt.event {
				t.Errorf("expected event %v, got %v", tt.event, receivedEvent)
			}
		})
	}
}

// TestPropertyNoSubscribersNoError verifies publishing with no subscribers doesn't error
func TestPropertyNoSubscribersNoError(t *testing.T) {
	eb := NewEventBus(nil)

	// For any topic with no subscribers, publishing should not error
	for i := 0; i < 5; i++ {
		if err := eb.Publish(context.Background(), fmt.Sprintf("topic-%d", i), "event"); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}
	}
}

// TestPropertyTopicIsolation verifies topics are isolated
func TestPropertyTopicIsolation(t *testing.T) {
	eb := NewEventBus(nil)
	topic1Count := 0
	topic2Count := 0
	mu := sync.Mutex{}

	handler1 := func(_ context.Context, event any) error {
		mu.Lock()
		topic1Count++
		mu.Unlock()
		return nil
	}

	handler2 := func(_ context.Context, event any) error {
		mu.Lock()
		topic2Count++
		mu.Unlock()
		return nil
	}

	// For any two topics, events published to one should not affect the other
	if _, err := eb.Subscribe(context.Background(), "topic-1", handler1); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if _, err := eb.Subscribe(context.Background(), "topic-2", handler2); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	if err := eb.Publish(context.Background(), "topic-1", "event-1"); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if topic1Count != 1 {
		t.Errorf("expected topic-1 handler to be called once, got %d", topic1Count)
	}

	if topic2Count != 0 {
		t.Errorf("expected topic-2 handler not to be called, got %d", topic2Count)
	}
	mu.Unlock()

	if err := eb.Publish(context.Background(), "topic-2", "event-2"); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if topic1Count != 1 {
		t.Errorf("expected topic-1 handler to still be called once, got %d", topic1Count)
	}

	if topic2Count != 1 {
		t.Errorf("expected topic-2 handler to be called once, got %d", topic2Count)
	}
}

// TestPropertySyncVsAsyncConsistency verifies sync and async publishing consistency
func TestPropertySyncVsAsyncConsistency(t *testing.T) {
	// For any event, sync and async publishing should both deliver the event
	testEvents := []any{
		"event-1",
		map[string]any{"id": "event-2"},
		[]int{1, 2, 3},
	}

	for _, testEvent := range testEvents {
		// Test async publishing
		eb1 := NewEventBus(nil)
		asyncReceived := false
		mu1 := sync.Mutex{}
		handler1 := func(_ context.Context, event any) error {
			mu1.Lock()
			asyncReceived = true
			mu1.Unlock()
			return nil
		}
		if _, err := eb1.Subscribe(context.Background(), "test-topic", handler1); err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}
		if err := eb1.Publish(context.Background(), "test-topic", testEvent); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		// Test sync publishing
		eb2 := NewEventBus(nil)
		syncReceived := false
		mu2 := sync.Mutex{}
		handler2 := func(_ context.Context, event any) error {
			mu2.Lock()
			syncReceived = true
			mu2.Unlock()
			return nil
		}
		if _, err := eb2.Subscribe(context.Background(), "test-topic", handler2); err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}
		if err := eb2.PublishSync(context.Background(), "test-topic", testEvent); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}

		mu1.Lock()
		asyncOK := asyncReceived
		mu1.Unlock()

		mu2.Lock()
		syncOK := syncReceived
		mu2.Unlock()

		if !asyncOK {
			t.Errorf("expected async handler to receive event")
		}

		if !syncOK {
			t.Errorf("expected sync handler to receive event")
		}
	}
}

// TestPropertyClearConsistency verifies clear removes all subscribers
func TestPropertyClearConsistency(t *testing.T) {
	eb := NewEventBus(nil)
	handler := func(_ context.Context, event any) error { return nil }

	// For any set of subscribers, clear should remove all of them
	for i := 0; i < 5; i++ {
		if _, err := eb.Subscribe(context.Background(), fmt.Sprintf("topic-%d", i), handler); err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}
	}

	if len(eb.GetTopics()) != 5 {
		t.Errorf("expected 5 topics before clear")
	}

	eb.Clear()

	if len(eb.GetTopics()) != 0 {
		t.Errorf("expected 0 topics after clear, got %d", len(eb.GetTopics()))
	}

	// Verify no events are delivered after clear
	count := 0
	mu := sync.Mutex{}
	handler2 := func(_ context.Context, event any) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	}

	if _, err := eb.Subscribe(context.Background(), "test-topic", handler2); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	eb.Clear()
	if err := eb.Publish(context.Background(), "test-topic", "event"); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 0 {
		t.Errorf("expected no handlers to be called after clear, got %d", count)
	}
}

// TestPropertyConcurrentOperationsConsistency verifies concurrent operations are consistent
func TestPropertyConcurrentOperationsConsistency(t *testing.T) {
	eb := NewEventBus(nil)
	handler := func(_ context.Context, event any) error { return nil }

	// For concurrent subscriptions and publications, the system should remain consistent
	done := make(chan bool, 20)

	// Concurrent subscriptions
	for i := 0; i < 10; i++ {
		go func(index int) {
			_, _ = eb.Subscribe(context.Background(), fmt.Sprintf("topic-%d", index%3), handler)
			done <- true
		}(i)
	}

	// Concurrent publications
	for i := 0; i < 10; i++ {
		go func(index int) {
			_ = eb.Publish(context.Background(), fmt.Sprintf("topic-%d", index%3), "event")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify system is still consistent
	topics := eb.GetTopics()
	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}

	for i := 0; i < 3; i++ {
		count := eb.GetSubscriberCount(fmt.Sprintf("topic-%d", i))
		if count == 0 {
			t.Errorf("expected subscribers for topic-%d", i)
		}
	}
}

// TestPropertyHandlerPanicRecovery verifies handler panics don't crash the bus
func TestPropertyHandlerPanicRecovery(t *testing.T) {
	eb := NewEventBus(nil)

	// For any handler that panics, the event bus should recover
	panicHandler := func(_ context.Context, event any) error {
		panic("test panic")
	}

	normalHandler := func(_ context.Context, event any) error { return nil }

	_, err := eb.Subscribe(context.Background(), "test-topic", panicHandler)
	if err != nil {
		t.Fatalf("failed to subscribe panic handler: %v", err)
	}
	_, err = eb.Subscribe(context.Background(), "test-topic", normalHandler)
	if err != nil {
		t.Fatalf("failed to subscribe normal handler: %v", err)
	}

	// This should not panic
	err = eb.PublishSync(context.Background(), "test-topic", "event")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
