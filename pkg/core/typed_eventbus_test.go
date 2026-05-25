package core

import (
	"context"
	"sync"
	"testing"
)

func TestTypedEventBus_New(t *testing.T) {
	inner := NewChannelEventBus()
	bus := NewTypedEventBus[string](inner)
	if bus == nil {
		t.Fatal("NewTypedEventBus returned nil")
	}
	if bus.Inner() != inner {
		t.Error("Inner() should return the wrapped bus")
	}
}

func TestTypedEventBus_Publish(t *testing.T) {
	inner := NewChannelEventBus()
	bus := NewTypedEventBus[string](inner)
	err := bus.Publish(context.Background(), "test-topic", "hello")
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
}

func TestTypedEventBus_SubscribeAndPublish(t *testing.T) {
	inner := NewChannelEventBus()
	bus := NewTypedEventBus[string](inner)

	ctx := context.Background()
	var received string
	var mu sync.Mutex
	done := make(chan struct{})

	id, err := bus.Subscribe(ctx, "test-topic", func(ctx context.Context, event string) error {
		mu.Lock()
		received = event
		mu.Unlock()
		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription ID")
	}

	err = bus.Publish(ctx, "test-topic", "hello-world")
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	<-done
	mu.Lock()
	if received != "hello-world" {
		t.Errorf("expected 'hello-world', got %q", received)
	}
	mu.Unlock()
}

func TestTypedEventBus_SubscribeNamedAndPublish(t *testing.T) {
	inner := NewChannelEventBus()
	bus := NewTypedEventBus[int](inner)

	ctx := context.Background()
	var received int
	var mu sync.Mutex
	done := make(chan struct{})

	id, err := bus.SubscribeNamed(ctx, "test-topic", "my-sub", func(ctx context.Context, event int) error {
		mu.Lock()
		received = event
		mu.Unlock()
		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeNamed error: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero subscription ID")
	}

	err = bus.Publish(ctx, "test-topic", 42)
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	<-done
	mu.Lock()
	if received != 42 {
		t.Errorf("expected 42, got %d", received)
	}
	mu.Unlock()
}

func TestTypedEventBus_SubscribeWrongType(t *testing.T) {
	bus := NewTypedEventBus[string](NewChannelEventBus())

	ctx := context.Background()
	var received string
	done := make(chan struct{})

	// Subscribe for string events
	bus.Subscribe(ctx, "topic", func(ctx context.Context, event string) error {
		received = event
		close(done)
		return nil
	})

	// Publish an int through the raw bus - the typed wrapper should silently skip it
	bus.Inner().Publish(ctx, "topic", 123)
	bus.Inner().Publish(ctx, "topic", "correct-type")

	<-done
	if received != "correct-type" {
		t.Errorf("expected 'correct-type', got %q", received)
	}
}

func TestTypedEventBus_Unsubscribe(t *testing.T) {
	inner := NewChannelEventBus()
	bus := NewTypedEventBus[int](inner)

	ctx := context.Background()
	id, _ := bus.Subscribe(ctx, "topic", func(ctx context.Context, event int) error {
		return nil
	})

	err := bus.Unsubscribe(id)
	if err != nil {
		t.Fatalf("Unsubscribe error: %v", err)
	}
}

func TestTypedEventBus_Inner(t *testing.T) {
	inner := NewChannelEventBus()
	bus := NewTypedEventBus[float64](inner)
	if bus.Inner() != inner {
		t.Error("Inner() should return the original bus")
	}
}
