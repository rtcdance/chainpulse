package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// MockStreamingBackend implements StreamingBackend for testing
type MockStreamingBackend struct {
	events []*core.BlockchainEvent
}

func NewMockStreamingBackend() *MockStreamingBackend {
	return &MockStreamingBackend{
		events: make([]*core.BlockchainEvent, 0),
	}
}

func (m *MockStreamingBackend) GetEventStream(ctx context.Context) (<-chan *core.BlockchainEvent, error) {
	ch := make(chan *core.BlockchainEvent)
	go func() {
		defer close(ch)
		for _, event := range m.events {
			select {
			case <-ctx.Done():
				return
			case ch <- event:
			}
		}
	}()
	return ch, nil
}

func (m *MockStreamingBackend) ProcessEventBatch(ctx context.Context, events <-chan *core.BlockchainEvent) (int64, error) {
	count := int64(0)
	for range events {
		count++
	}
	return count, nil
}

func TestServerStreamEvents(t *testing.T) {
	backend := NewMockStreamingBackend()
	backend.events = []*core.BlockchainEvent{
		{ID: "1", EventName: "Event1"},
		{ID: "2", EventName: "Event2"},
		{ID: "3", EventName: "Event3"},
	}

	service := NewStreamingService(backend)
	ctx := context.Background()

	count := 0
	err := service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to stream events: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 events, got %d", count)
	}

	metrics := service.GetMetrics()
	if metrics["items_streamed"].(int64) != 3 {
		t.Errorf("Expected 3 items streamed, got %v", metrics["items_streamed"])
	}
}

func TestClientStreamEvents(t *testing.T) {
	backend := NewMockStreamingBackend()
	service := NewStreamingService(backend)
	ctx := context.Background()

	// Create event channel
	eventChan := make(chan *core.BlockchainEvent)
	go func() {
		defer close(eventChan)
		for i := 1; i <= 5; i++ {
			eventChan <- &core.BlockchainEvent{ID: fmt.Sprintf("%d", i)}
		}
	}()

	result, err := service.ClientStreamEvents(ctx, eventChan)
	if err != nil {
		t.Fatalf("Failed to stream events: %v", err)
	}

	if result != 5 {
		t.Errorf("Expected 5 events processed, got %d", result)
	}
}

func TestStreamingContextCancellation(t *testing.T) {
	backend := NewMockStreamingBackend()
	backend.events = []*core.BlockchainEvent{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}

	service := NewStreamingService(backend)
	ctx, cancel := context.WithCancel(context.Background())

	count := 0
	err := service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		count++
		if count == 1 {
			cancel()
		}
		return nil
	})

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if count != 1 {
		t.Errorf("Expected 1 event before cancellation, got %d", count)
	}
}

func TestStreamingMetrics(t *testing.T) {
	backend := NewMockStreamingBackend()
	backend.events = []*core.BlockchainEvent{
		{ID: "1"},
		{ID: "2"},
	}

	service := NewStreamingService(backend)
	ctx := context.Background()

	if err := service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		return nil
	}); err != nil {
		t.Fatalf("failed to stream events: %v", err)
	}

	metrics := service.GetMetrics()

	totalStreams := metrics["total_streams"].(int64)
	if totalStreams != 1 {
		t.Errorf("Expected 1 total stream, got %d", totalStreams)
	}

	itemsStreamed := metrics["items_streamed"].(int64)
	if itemsStreamed != 2 {
		t.Errorf("Expected 2 items streamed, got %d", itemsStreamed)
	}

	activeStreams := metrics["active_streams"].(int64)
	if activeStreams != 0 {
		t.Errorf("Expected 0 active streams, got %d", activeStreams)
	}
}

func TestStreamingErrorHandling(t *testing.T) {
	backend := NewMockStreamingBackend()
	backend.events = []*core.BlockchainEvent{
		{ID: "1"},
		{ID: "2"},
	}

	service := NewStreamingService(backend)
	ctx := context.Background()

	count := 0
	err := service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		count++
		if count == 1 {
			return fmt.Errorf("handler error")
		}
		return nil
	})

	if err == nil {
		t.Error("Expected handler error")
	}

	metrics := service.GetMetrics()
	errors := metrics["errors"].(int64)
	if errors < 1 {
		t.Errorf("Expected at least 1 error, got %d", errors)
	}
}

func TestStreamingTimeout(t *testing.T) {
	backend := NewMockStreamingBackend()
	backend.events = []*core.BlockchainEvent{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}

	service := NewStreamingService(backend)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	count := 0
	err := service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		count++
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestMultipleStreamsMetrics(t *testing.T) {
	backend := NewMockStreamingBackend()
	backend.events = []*core.BlockchainEvent{{ID: "1"}}

	service := NewStreamingService(backend)
	ctx := context.Background()

	_ = service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		return nil
	})

	_ = service.ServerStreamEvents(ctx, func(event *core.BlockchainEvent) error {
		return nil
	})

	metrics := service.GetMetrics()

	totalStreams := metrics["total_streams"].(int64)
	if totalStreams != 2 {
		t.Errorf("Expected 2 total streams, got %d", totalStreams)
	}

	itemsStreamed := metrics["items_streamed"].(int64)
	if itemsStreamed != 2 {
		t.Errorf("Expected 2 items streamed, got %d", itemsStreamed)
	}
}
