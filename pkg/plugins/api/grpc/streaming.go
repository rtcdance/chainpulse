package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// StreamingService provides gRPC streaming capabilities
type StreamingService struct {
	backend StreamingBackend
	mu      sync.RWMutex
	metrics *StreamingMetrics
}

// StreamingBackend defines the backend for streaming operations
type StreamingBackend interface {
	// Server streaming
	GetEventStream(ctx context.Context) (<-chan *core.BlockchainEvent, error)

	// Client streaming
	ProcessEventBatch(ctx context.Context, events <-chan *core.BlockchainEvent) (int64, error)
}

// StreamingMetrics tracks streaming operation metrics
type StreamingMetrics struct {
	totalStreams      int64
	activeStreams     int64
	itemsStreamed     int64
	errors            int64
	totalDuration     time.Duration
	mu                sync.RWMutex
}

// NewStreamingService creates a new streaming service
func NewStreamingService(backend StreamingBackend) *StreamingService {
	return &StreamingService{
		backend: backend,
		metrics: &StreamingMetrics{},
	}
}

// ServerStreamEvents implements server streaming for events
func (s *StreamingService) ServerStreamEvents(ctx context.Context, handler func(*core.BlockchainEvent) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	s.recordStreamStart()
	defer func() {
		s.recordStreamEnd(time.Since(start))
	}()

	// Get event stream from backend
	eventChan, err := s.backend.GetEventStream(ctx)
	if err != nil {
		s.recordError()
		return fmt.Errorf("failed to get event stream: %w", err)
	}

	// Process events
	for {
		select {
		case <-ctx.Done():
			s.recordError()
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			if event == nil {
				continue
			}
			// Check context before processing handler
			select {
			case <-ctx.Done():
				s.recordError()
				return ctx.Err()
			default:
			}
			// Execute handler with context timeout check
			done := make(chan error, 1)
			go func() {
				done <- handler(event)
			}()
			select {
			case <-ctx.Done():
				s.recordError()
				return ctx.Err()
			case err := <-done:
				if err != nil {
					s.recordError()
					return fmt.Errorf("failed to handle event: %w", err)
				}
				s.recordItem()
				// Check context again after handler completes
				select {
				case <-ctx.Done():
					s.recordError()
					return ctx.Err()
				default:
				}
			}
		}
	}
}

// ClientStreamEvents implements client streaming for events
func (s *StreamingService) ClientStreamEvents(ctx context.Context, events <-chan *core.BlockchainEvent) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	s.recordStreamStart()
	defer func() {
		s.recordStreamEnd(time.Since(start))
	}()

	// Process batch through backend
	result, err := s.backend.ProcessEventBatch(ctx, events)
	if err != nil {
		s.recordError()
		return 0, fmt.Errorf("failed to process event batch: %w", err)
	}

	s.recordItem()
	return result, nil
}

// GetMetrics returns streaming metrics
func (s *StreamingService) GetMetrics() map[string]interface{} {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	avgDuration := time.Duration(0)
	if s.metrics.totalStreams > 0 {
		avgDuration = s.metrics.totalDuration / time.Duration(s.metrics.totalStreams)
	}

	return map[string]interface{}{
		"total_streams":    s.metrics.totalStreams,
		"active_streams":   s.metrics.activeStreams,
		"items_streamed":   s.metrics.itemsStreamed,
		"errors":           s.metrics.errors,
		"avg_duration_ms":  avgDuration.Milliseconds(),
		"total_duration":   s.metrics.totalDuration.String(),
	}
}

// Helper methods

func (s *StreamingService) recordStreamStart() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.totalStreams++
	s.metrics.activeStreams++
}

func (s *StreamingService) recordStreamEnd(duration time.Duration) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.activeStreams--
	s.metrics.totalDuration += duration
}

func (s *StreamingService) recordItem() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.itemsStreamed++
}

func (s *StreamingService) recordError() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.errors++
}
