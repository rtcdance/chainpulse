package mq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestMemoryMQ_StartStop(t *testing.T) {
	mq := NewMemoryMQ()

	if err := mq.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := mq.Health(); err != nil {
		t.Fatalf("Health check after Start should be nil, got: %v", err)
	}
	if err := mq.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestMemoryMQ_HealthNotStarted(t *testing.T) {
	mq := NewMemoryMQ()

	err := mq.Health()
	if err == nil {
		t.Fatal("Health check on not-started MQ should return error")
	}
}

func TestMemoryMQ_PublishSubscribe(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []byte
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1)

	err := mq.Subscribe(ctx, "test-topic", func(msg []byte) {
		mu.Lock()
		received = msg
		mu.Unlock()
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	err = mq.Publish(ctx, "test-topic", []byte("hello"))
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	mu.Lock()
	if string(received) != "hello" {
		t.Fatalf("expected 'hello', got '%s'", string(received))
	}
	mu.Unlock()
}

func TestMemoryMQ_PublishNoSubscribers(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	err := mq.Publish(context.Background(), "empty-topic", []byte("nobody"))
	if err != nil {
		t.Fatalf("Publish to empty topic should not error, got: %v", err)
	}
}

func TestMemoryMQ_MultipleSubscribers(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	received := make(map[int]string)
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		id := i
		err := mq.Subscribe(ctx, "multi-topic", func(msg []byte) {
			mu.Lock()
			received[id] = string(msg)
			mu.Unlock()
			wg.Done()
		})
		if err != nil {
			t.Fatalf("Subscribe %d failed: %v", i, err)
		}
	}

	err := mq.Publish(ctx, "multi-topic", []byte("broadcast"))
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if len(received) != 3 {
		t.Fatalf("expected 3 subscribers to receive, got %d", len(received))
	}
	for id, msg := range received {
		if msg != "broadcast" {
			t.Fatalf("subscriber %d: expected 'broadcast', got '%s'", id, msg)
		}
	}
}

func TestMemoryMQ_MultipleTopics(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	var topic1Msg, topic2Msg string
	var mu sync.Mutex

	mq.Subscribe(ctx, "topic-1", func(msg []byte) {
		mu.Lock()
		topic1Msg = string(msg)
		mu.Unlock()
		wg.Done()
	})

	mq.Subscribe(ctx, "topic-2", func(msg []byte) {
		mu.Lock()
		topic2Msg = string(msg)
		mu.Unlock()
		wg.Done()
	})

	mq.Publish(ctx, "topic-1", []byte("msg-1"))
	mq.Publish(ctx, "topic-2", []byte("msg-2"))

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if topic1Msg != "msg-1" {
		t.Fatalf("topic-1: expected 'msg-1', got '%s'", topic1Msg)
	}
	if topic2Msg != "msg-2" {
		t.Fatalf("topic-2: expected 'msg-2', got '%s'", topic2Msg)
	}
}

func TestMemoryMQ_ContextCancellation(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		mq.Subscribe(ctx, "ctx-topic", func(msg []byte) {
			// handler won't be called because we cancel before publish
			t.Error("handler should not be called after cancellation")
		})
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()
	wg.Wait()

	// After cancellation, subscriber goroutine should exit cleanly
	err := mq.Publish(context.Background(), "ctx-topic", []byte("too-late"))
	if err != nil {
		t.Fatalf("Publish after cancel should not error, got: %v", err)
	}
}

func TestMemoryMQ_GetQueueDepth(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	depth, err := mq.GetQueueDepth(context.Background(), "no-such-topic")
	if err != nil {
		t.Fatalf("GetQueueDepth on empty topic should not error: %v", err)
	}
	if depth != 0 {
		t.Fatalf("expected depth 0 for empty topic, got %d", depth)
	}
}

func TestMemoryMQ_GetQueueDepthWithMessages(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()
	defer mq.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscriber that doesn't consume (blocking the channel)
	blockCh := make(chan struct{})
	mq.Subscribe(ctx, "slow-topic", func(msg []byte) {
		<-blockCh // block forever
	})

	for i := 0; i < 3; i++ {
		mq.Publish(ctx, "slow-topic", []byte("msg"))
	}

	depth, err := mq.GetQueueDepth(context.Background(), "slow-topic")
	if err != nil {
		t.Fatalf("GetQueueDepth failed: %v", err)
	}
	// Channel buffer is 100, so 3 messages should be queued
	if depth < 1 {
		t.Fatalf("expected depth >= 1 for topic with buffered messages, got %d", depth)
	}

	close(blockCh)
}

func TestMemoryMQ_NameAndVersion(t *testing.T) {
	mq := NewMemoryMQ()

	if mq.Name() != "memory-mq" {
		t.Fatalf("expected name 'memory-mq', got '%s'", mq.Name())
	}
	if mq.Version() != "1.0.0" {
		t.Fatalf("expected version '1.0.0', got '%s'", mq.Version())
	}
}

func TestMemoryMQ_Initialize(t *testing.T) {
	mq := NewMemoryMQ()

	err := mq.Initialize(core.Config{})
	if err != nil {
		t.Fatalf("Initialize should always succeed, got: %v", err)
	}
}

func TestMemoryMQ_StopClosesSubscribers(t *testing.T) {
	mq := NewMemoryMQ()
	_ = mq.Start()

	ctx := context.Background()
	received := make(chan []byte, 1)

	mq.Subscribe(ctx, "close-topic", func(msg []byte) {
		received <- msg
	})

	mq.Publish(ctx, "close-topic", []byte("before-stop"))
	select {
	case msg := <-received:
		if string(msg) != "before-stop" {
			t.Fatalf("expected 'before-stop', got '%s'", string(msg))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}

	mq.Stop()

	err := mq.Publish(context.Background(), "close-topic", []byte("after-stop"))
	if err != nil {
		t.Fatalf("Publish after Stop should not error (no receivers), got: %v", err)
	}
}
