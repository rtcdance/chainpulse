package shared

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchRequest represents a single request in a batch
type BatchRequest struct {
	ID       string
	Payload  interface{}
	Response chan interface{}
	Error    chan error
}

// BatchProcessor processes a batch of requests
type BatchProcessor interface {
	Process(ctx context.Context, requests []*BatchRequest) error
}

// RequestBatcher batches multiple requests for efficient processing
type RequestBatcher struct {
	name          string
	processor     BatchProcessor
	batchSize     int
	batchTimeout  time.Duration
	queue         chan *BatchRequest
	metrics       *BatcherMetrics
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// BatcherMetrics tracks batcher metrics
type BatcherMetrics struct {
	totalRequests  int64
	totalBatches   int64
	avgBatchSize   float64
	errors         int64
	totalDuration  time.Duration
	mu             sync.RWMutex
}

// NewRequestBatcher creates a new request batcher
func NewRequestBatcher(name string, processor BatchProcessor, batchSize int, batchTimeout time.Duration) *RequestBatcher {
	ctx, cancel := context.WithCancel(context.Background())
	batcher := &RequestBatcher{
		name:         name,
		processor:    processor,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		queue:        make(chan *BatchRequest, batchSize*2),
		metrics:      &BatcherMetrics{},
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start batch processing goroutine
	batcher.wg.Add(1)
	go batcher.processingLoop()

	return batcher
}

// Submit submits a request to the batcher
func (b *RequestBatcher) Submit(ctx context.Context, id string, payload interface{}) (interface{}, error) {
	// Check if batcher is closed first
	select {
	case <-b.ctx.Done():
		return nil, fmt.Errorf("batcher is closed")
	default:
	}

	req := &BatchRequest{
		ID:       id,
		Payload:  payload,
		Response: make(chan interface{}, 1),
		Error:    make(chan error, 1),
	}

	// Use defer to recover from panic if channel is closed
	defer func() {
		if r := recover(); r != nil {
			_ = r // Channel was closed, ignore the panic
		}
	}()

	select {
	case b.queue <- req:
		// Wait for response
		select {
		case resp := <-req.Response:
			return resp, nil
		case err := <-req.Error:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.ctx.Done():
		return nil, fmt.Errorf("batcher is closed")
	}
}

// Close closes the batcher
func (b *RequestBatcher) Close() error {
	b.cancel()
	close(b.queue)
	b.wg.Wait()
	return nil
}

// GetMetrics returns batcher metrics
func (b *RequestBatcher) GetMetrics() map[string]interface{} {
	b.metrics.mu.RLock()
	defer b.metrics.mu.RUnlock()

	avgDuration := time.Duration(0)
	if b.metrics.totalBatches > 0 {
		avgDuration = b.metrics.totalDuration / time.Duration(b.metrics.totalBatches)
	}

	return map[string]interface{}{
		"batcher_name":    b.name,
		"total_requests":  b.metrics.totalRequests,
		"total_batches":   b.metrics.totalBatches,
		"avg_batch_size":  b.metrics.avgBatchSize,
		"errors":          b.metrics.errors,
		"avg_duration_ms": avgDuration.Milliseconds(),
		"total_duration":  b.metrics.totalDuration.String(),
	}
}

// Helper methods

func (b *RequestBatcher) processingLoop() {
	defer b.wg.Done()

	batch := make([]*BatchRequest, 0, b.batchSize)
	timer := time.NewTimer(b.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case req, ok := <-b.queue:
			if !ok {
				// Queue is closed, process remaining batch
				if len(batch) > 0 {
					b.processBatch(batch)
				}
				return
			}

			batch = append(batch, req)

			// Process batch if it reaches the size limit
			if len(batch) >= b.batchSize {
				b.processBatch(batch)
				batch = make([]*BatchRequest, 0, b.batchSize)
				timer.Reset(b.batchTimeout)
			}

		case <-timer.C:
			// Process batch if timeout is reached
			if len(batch) > 0 {
				b.processBatch(batch)
				batch = make([]*BatchRequest, 0, b.batchSize)
			}
			timer.Reset(b.batchTimeout)

		case <-b.ctx.Done():
			// Batcher is closed, process remaining batch
			if len(batch) > 0 {
				b.processBatch(batch)
			}
			return
		}
	}
}

func (b *RequestBatcher) processBatch(batch []*BatchRequest) {
	start := time.Now()
	defer func() {
		b.recordBatch(len(batch), time.Since(start))
	}()

	// Process the batch
	err := b.processor.Process(b.ctx, batch)

	// Send responses (with panic recovery to handle closed channels)
	for _, req := range batch {
		func() {
			defer func() {
				if r := recover(); r != nil {
					_ = r // Channel was closed, ignore the panic
				}
			}()

			if err != nil {
				req.Error <- err
				b.recordError()
			} else {
				// In a real implementation, the processor would populate responses
				req.Response <- nil
			}
		}()
	}
}

func (b *RequestBatcher) recordBatch(size int, duration time.Duration) {
	b.metrics.mu.Lock()
	defer b.metrics.mu.Unlock()

	b.metrics.totalRequests += int64(size)
	b.metrics.totalBatches++
	b.metrics.totalDuration += duration

	// Update average batch size
	if b.metrics.totalBatches > 0 {
		b.metrics.avgBatchSize = float64(b.metrics.totalRequests) / float64(b.metrics.totalBatches)
	}
}

func (b *RequestBatcher) recordError() {
	b.metrics.mu.Lock()
	defer b.metrics.mu.Unlock()
	b.metrics.errors++
}
