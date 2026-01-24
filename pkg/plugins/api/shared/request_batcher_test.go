package shared

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockBatchProcessor implements BatchProcessor for testing
type MockBatchProcessor struct {
	processedBatches int
	mu               sync.Mutex
}

func (m *MockBatchProcessor) Process(ctx context.Context, requests []*BatchRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedBatches++

	// Send responses
	for _, req := range requests {
		req.Response <- fmt.Sprintf("response-%s", req.ID)
	}

	return nil
}

func TestRequestBatcherSubmit(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 5, 100*time.Millisecond)
	defer func() {
		if err := batcher.Close(); err != nil {
			t.Logf("failed to close batcher: %v", err)
		}
	}()

	resp, err := batcher.Submit(context.Background(), "req-1", "payload-1")
	if err != nil {
		t.Fatalf("failed to submit request: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}
}

func TestRequestBatcherBatchSize(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 3, 1*time.Second)
	defer func() {
		if err := batcher.Close(); err != nil {
			t.Logf("failed to close batcher: %v", err)
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = batcher.Submit(context.Background(), fmt.Sprintf("req-%d", id), fmt.Sprintf("payload-%d", id))
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	metrics := batcher.GetMetrics()
	if metrics["total_batches"].(int64) < 1 {
		t.Errorf("expected at least 1 batch, got %v", metrics["total_batches"])
	}
}

func TestRequestBatcherTimeout(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 10, 100*time.Millisecond)
	defer func() {
		if err := batcher.Close(); err != nil {
			t.Logf("failed to close batcher: %v", err)
		}
	}()

	// Submit a single request
	go func() {
		_, _ = batcher.Submit(context.Background(), "req-1", "payload-1")
	}()

	// Wait for timeout to trigger batch processing
	time.Sleep(200 * time.Millisecond)

	metrics := batcher.GetMetrics()
	if metrics["total_batches"].(int64) < 1 {
		t.Errorf("expected batch to be processed on timeout, got %v batches", metrics["total_batches"])
	}
}

func TestRequestBatcherConcurrent(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 5, 100*time.Millisecond)
	defer func() {
		if err := batcher.Close(); err != nil {
			t.Logf("failed to close batcher: %v", err)
		}
	}()

	var wg sync.WaitGroup
	numRequests := 20

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = batcher.Submit(context.Background(), fmt.Sprintf("req-%d", id), fmt.Sprintf("payload-%d", id))
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	metrics := batcher.GetMetrics()
	if metrics["total_requests"].(int64) != int64(numRequests) {
		t.Errorf("expected %d requests, got %v", numRequests, metrics["total_requests"])
	}
}

func TestRequestBatcherMetrics(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 5, 100*time.Millisecond)
	defer func() {
		if err := batcher.Close(); err != nil {
			t.Logf("failed to close batcher: %v", err)
		}
	}()

	for i := 0; i < 5; i++ {
		_, _ = batcher.Submit(context.Background(), fmt.Sprintf("req-%d", i), fmt.Sprintf("payload-%d", i))
	}

	time.Sleep(200 * time.Millisecond)

	metrics := batcher.GetMetrics()
	if metrics["batcher_name"] != "test" {
		t.Errorf("expected batcher_name 'test', got %v", metrics["batcher_name"])
	}

	if metrics["total_requests"].(int64) != 5 {
		t.Errorf("expected 5 total requests, got %v", metrics["total_requests"])
	}

	if metrics["avg_batch_size"].(float64) <= 0 {
		t.Errorf("expected positive avg_batch_size, got %v", metrics["avg_batch_size"])
	}
}

func TestRequestBatcherClose(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 5, 100*time.Millisecond)

	if err := batcher.Close(); err != nil {
		t.Fatalf("failed to close batcher: %v", err)
	}

	// Try to submit after close
	_, err := batcher.Submit(context.Background(), "req-1", "payload-1")
	if err == nil {
		t.Fatal("expected error when submitting to closed batcher")
	}
}

func TestRequestBatcherContextCancellation(t *testing.T) {
	processor := &MockBatchProcessor{}
	batcher := NewRequestBatcher("test", processor, 5, 100*time.Millisecond)
	defer func() {
		if err := batcher.Close(); err != nil {
			t.Logf("failed to close batcher: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := batcher.Submit(ctx, "req-1", "payload-1")
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}
