package mq

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestNewMemoryMQ(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	if m == nil {
		t.Fatal("expected non-nil MemoryMQ")
	}
	if m.Name() != "memory-mq" {
		t.Errorf("Name() = %q, want memory-mq", m.Name())
	}
	if m.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want 1.0.0", m.Version())
	}
}

func TestMemoryMQ_Initialize(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	if err := m.Initialize(core.Config{}); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
}

func TestMemoryMQ_StartStop(t *testing.T) {
	m := NewMemoryMQ()
	if err := m.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := m.Health(); err != nil {
		t.Fatalf("Health() error after start: %v", err)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestMemoryMQ_HealthNotStarted(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	if err := m.Health(); err == nil {
		t.Error("expected error from Health() when not started")
	}
}

func TestMemoryMQ_StopNotStarted(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop() when not started: %v", err)
	}
}

func TestMemoryMQ_PublishToEmpty(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	ctx := context.Background()
	err := m.Publish(ctx, "test-topic", []byte("hello"))
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}
}

func TestMemoryMQ_SubscribeAndPublish(t *testing.T) {
	m := NewMemoryMQ()
	ctx, cancel := context.WithCancel(context.Background())

	received := make(chan []byte, 1)
	err := m.Subscribe(ctx, "test-topic", func(msg []byte) {
		received <- msg
	})
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_ = m.Publish(ctx, "test-topic", []byte("hello"))

	select {
	case msg := <-received:
		if string(msg) != "hello" {
			t.Errorf("received %q, want hello", string(msg))
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for message")
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = m.Stop()
}

func TestMemoryMQ_EvictOnce(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	m.evictOnce()
}

func TestMemoryMQ_EvictOnce_WithSubscribers(t *testing.T) {
	t.Parallel()
	m := NewMemoryMQ()
	ctx, cancel := context.WithCancel(context.Background())

	_ = m.Subscribe(ctx, "test-topic", func(msg []byte) {})
	time.Sleep(10 * time.Millisecond)
	_ = m.Publish(ctx, "test-topic", []byte("msg"))
	time.Sleep(10 * time.Millisecond)

	m.evictOnce()

	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = m.Stop()
}

func TestMemoryMQ_GetQueueDepth_WithMessages(t *testing.T) {
	m := NewMemoryMQ()
	ctx, cancel := context.WithCancel(context.Background())

	_ = m.Subscribe(ctx, "test-topic", func(msg []byte) {})
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 5; i++ {
		_ = m.Publish(ctx, "test-topic", []byte("msg"))
	}

	depth, err := m.GetQueueDepth(ctx, "test-topic")
	if err != nil {
		t.Fatalf("GetQueueDepth() error: %v", err)
	}
	_ = depth

	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = m.Stop()
}

func TestMemoryMQ_GetQueueDepth(t *testing.T) {
	m := NewMemoryMQ()
	ctx, cancel := context.WithCancel(context.Background())
	_ = m.Start()

	depth, err := m.GetQueueDepth(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetQueueDepth() error: %v", err)
	}
	if depth != 0 {
		t.Errorf("depth = %d, want 0", depth)
	}

	ch := make(chan []byte, 100)
	_ = m.Subscribe(ctx, "test-topic", func(msg []byte) {
		ch <- msg
	})
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 5; i++ {
		_ = m.Publish(ctx, "test-topic", []byte("msg"))
	}
	time.Sleep(10 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = m.Stop()
}

func TestMemoryMQ_ConcurrentPublish(t *testing.T) {
	m := NewMemoryMQ()
	ctx, cancel := context.WithCancel(context.Background())

	received := make(chan []byte, 100)
	_ = m.Subscribe(ctx, "test-topic", func(msg []byte) {
		received <- msg
	})
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		go func() {
			_ = m.Publish(ctx, "test-topic", []byte("msg"))
		}()
	}

	count := 0
	timeout := time.After(2 * time.Second)
	for count < 10 {
		select {
		case <-received:
			count++
		case <-timeout:
			t.Fatalf("timeout: received %d/10 messages", count)
		}
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = m.Stop()
}

func TestMemoryMQ_SubscribeContextCancel(t *testing.T) {
	m := NewMemoryMQ()
	ctx, cancel := context.WithCancel(context.Background())

	err := m.Subscribe(ctx, "test-topic", func(msg []byte) {})
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = m.Stop()
}
