package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

type testMetricsCollector struct{}

func (m *testMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {}
func (m *testMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {}

func (m *testMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {}

func (m *testMetricsCollector) GetMetrics() map[string]any { return nil }

func TestNewChannelEventBus(t *testing.T) {
	bus := NewChannelEventBus()
	if bus == nil {
		t.Fatal("NewChannelEventBus returned nil")
	}
	if bus.SubscriberCount("test") != 0 {
		t.Error("expected 0 subscribers for empty topic")
	}
}

func TestChannelEventBus_SubscribeAndPublish(t *testing.T) {
	bus := NewChannelEventBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var received any
	var mu sync.Mutex
	done := make(chan struct{})

	handler := func(ctx context.Context, payload any) error {
		mu.Lock()
		received = payload
		mu.Unlock()
		close(done)
		return nil
	}

	id, err := bus.Subscribe(ctx, "test-topic", handler)
	if err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription id")
	}

	if bus.SubscriberCount("test-topic") != 1 {
		t.Error("expected 1 subscriber after subscribe")
	}

	_ = bus.Publish(ctx, "test-topic", "hello-world")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	mu.Lock()
	if received != "hello-world" {
		t.Errorf("received = %v, want 'hello-world'", received)
	}
	mu.Unlock()
}

func TestChannelEventBus_SubscribeNamed(t *testing.T) {
	bus := NewChannelEventBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var received any
	done := make(chan struct{})

	handler := func(ctx context.Context, payload any) error {
		received = payload
		close(done)
		return nil
	}

	id, err := bus.SubscribeNamed(ctx, "topic", "my-sub", handler)
	if err != nil {
		t.Fatalf("SubscribeNamed error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription id")
	}

	_ = bus.Publish(ctx, "topic", "named-event")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for named event")
	}

	if received != "named-event" {
		t.Errorf("received = %v", received)
	}
}

func TestChannelEventBus_Unsubscribe(t *testing.T) {
	bus := NewChannelEventBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handler := func(ctx context.Context, payload any) error { return nil }
	id, _ := bus.Subscribe(ctx, "topic", handler)

	if bus.SubscriberCount("topic") != 1 {
		t.Error("expected 1 subscriber")
	}

	if err := bus.Unsubscribe(id); err != nil {
		t.Errorf("Unsubscribe error: %v", err)
	}

	if bus.SubscriberCount("topic") != 0 {
		t.Error("expected 0 subscribers after unsubscribe")
	}

	if err := bus.Unsubscribe(id); err == nil {
		t.Error("expected error for already unsubscribed id")
	}
}

func TestChannelEventBus_SetLogger(t *testing.T) {
	bus := NewChannelEventBus()
	bus.SetLogger(&testLogger{})
}

func TestChannelEventBus_SetMetrics(t *testing.T) {
	bus := NewChannelEventBus()
	bus.SetMetrics(&testMetricsCollector{})
}

func TestChannelEventBus_DroppedEvents(t *testing.T) {
	bus := NewChannelEventBus()
	if bus.DroppedEvents() != 0 {
		t.Error("expected 0 dropped events for new bus")
	}
}

func TestChannelEventBus_PublishNoSubscriber(t *testing.T) {
	bus := NewChannelEventBus()
	ctx := context.Background()
	if err := bus.Publish(ctx, "no-subscribers", "event"); err != nil {
		t.Errorf("Publish without subscribers should not error: %v", err)
	}
}

func TestChannelEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewChannelEventBus()
	ctx := context.Background()

	var wg sync.WaitGroup
	received := make([]string, 0)
	var mu sync.Mutex

	for i := range 3 {
		wg.Add(1)
		idx := i
		handler := func(ctx context.Context, payload any) error {
			defer wg.Done()
			mu.Lock()
			received = append(received, payload.(string))
			mu.Unlock()
			_ = idx
			return nil
		}
		bus.Subscribe(ctx, "multi", handler)
	}

	if bus.SubscriberCount("multi") != 3 {
		t.Errorf("expected 3 subscribers, got %d", bus.SubscriberCount("multi"))
	}

	_ = bus.Publish(ctx, "multi", "broadcast")

	wg.Wait()

	mu.Lock()
	if len(received) != 3 {
		t.Errorf("expected 3 handlers to receive event, got %d", len(received))
	}
	mu.Unlock()
}

func TestSubscribeTyped(t *testing.T) {
	bus := NewChannelEventBus()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var received string
	done := make(chan struct{})

	id, err := SubscribeTyped[string](bus, ctx, "typed-topic", func(val string) {
		received = val
		close(done)
	})
	if err != nil {
		t.Fatalf("SubscribeTyped error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription id")
	}

	_ = bus.Publish(ctx, "typed-topic", "hello")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	if received != "hello" {
		t.Errorf("received = %v", received)
	}
}

func TestSubscribeTypedNamed(t *testing.T) {
	bus := NewChannelEventBus()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var received int
	done := make(chan struct{})

	id, err := SubscribeTypedNamed[int](bus, ctx, "typed-named-topic", "my-sub", func(val int) {
		received = val
		close(done)
	})
	if err != nil {
		t.Fatalf("SubscribeTypedNamed error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription id")
	}

	_ = bus.Publish(ctx, "typed-named-topic", 42)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	if received != 42 {
		t.Errorf("received = %v", received)
	}
}

func TestDefaultEventBus_SetPublishTimeout(t *testing.T) {
	eb := NewEventBus(nil)
	eb.SetPublishTimeout(10 * time.Second)

	if eb.publishTimeout.Load() != int64(10*time.Second) {
		t.Error("publishTimeout not updated")
	}
}

func TestDefaultEventBus_SubscribeNamed(t *testing.T) {
	eb := NewEventBus(nil)
	ctx := context.Background()

	handler := func(ctx context.Context, payload any) error { return nil }
	id, err := eb.SubscribeNamed(ctx, "topic", "my-name", handler)
	if err != nil {
		t.Fatalf("SubscribeNamed error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription id")
	}

	eb.mu.RLock()
	name := eb.subscriberNames[id]
	eb.mu.RUnlock()
	if name != "my-name" {
		t.Errorf("subscriberName = %s", name)
	}
}

func TestDefaultEventBus_SubscribeNamedEmptyName(t *testing.T) {
	eb := NewEventBus(nil)
	ctx := context.Background()

	handler := func(ctx context.Context, payload any) error { return nil }
	id, err := eb.SubscribeNamed(ctx, "topic", "", handler)
	if err != nil {
		t.Fatalf("SubscribeNamed error: %v", err)
	}

	eb.mu.RLock()
	_, hasName := eb.subscriberNames[id]
	eb.mu.RUnlock()
	if hasName {
		t.Error("empty name should not be recorded")
	}
}

func TestDefaultEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus(nil)
	ctx := context.Background()

	handler := func(ctx context.Context, payload any) error { return nil }
	id, _ := eb.Subscribe(ctx, "topic", handler)

	if err := eb.Unsubscribe(id); err != nil {
		t.Errorf("Unsubscribe error: %v", err)
	}

	if err := eb.Unsubscribe(id); err == nil {
		t.Error("expected error for already unsubscribed id")
	}

	if err := eb.Unsubscribe(99999); err == nil {
		t.Error("expected error for nonexistent id")
	}
}
