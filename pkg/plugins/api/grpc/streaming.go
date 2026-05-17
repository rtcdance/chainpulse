package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
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

// StreamingSourcePostureBackend optionally describes the source posture behind
// the streaming backend so gRPC metrics can expose a compact data-plane hint.
type StreamingSourcePostureBackend interface {
	ServerStreamSourcePosture() string
	ClientStreamSourcePosture() string
}

// StreamingMetrics tracks streaming operation metrics
type StreamingMetrics struct {
	totalStreams        int64
	activeStreams       int64
	itemsStreamed       int64
	errors              int64
	totalDuration       time.Duration
	serverSourcePosture string
	clientSourcePosture string
	mu                  sync.RWMutex
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
	s.recordServerSourcePosture()

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
	s.recordClientSourcePosture()
	result, err := s.backend.ProcessEventBatch(ctx, events)
	if err != nil {
		s.recordError()
		return 0, fmt.Errorf("failed to process event batch: %w", err)
	}

	s.recordItem()
	return result, nil
}

// GetMetrics returns streaming metrics
func (s *StreamingService) GetMetrics() map[string]any {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	avgDuration := time.Duration(0)
	if s.metrics.totalStreams > 0 {
		avgDuration = s.metrics.totalDuration / time.Duration(s.metrics.totalStreams)
	}

	return map[string]any{
		"total_streams":         s.metrics.totalStreams,
		"active_streams":        s.metrics.activeStreams,
		"items_streamed":        s.metrics.itemsStreamed,
		"errors":                s.metrics.errors,
		"avg_duration_ms":       avgDuration.Milliseconds(),
		"total_duration":        s.metrics.totalDuration.String(),
		"server_source_posture": s.metrics.serverSourcePosture,
		"client_source_posture": s.metrics.clientSourcePosture,
		"server_delivery_posture": classifyServerDeliveryPosture(
			s.metrics.serverSourcePosture,
			s.metrics.activeStreams,
			s.metrics.itemsStreamed,
			s.metrics.errors,
		),
		"client_delivery_posture": classifyClientDeliveryPosture(
			s.metrics.clientSourcePosture,
			s.metrics.itemsStreamed,
			s.metrics.errors,
		),
		"server_reliability_hint": buildServerReliabilityHint(
			s.metrics.serverSourcePosture,
			classifyServerDeliveryPosture(
				s.metrics.serverSourcePosture,
				s.metrics.activeStreams,
				s.metrics.itemsStreamed,
				s.metrics.errors,
			),
		),
		"client_reliability_hint": buildClientReliabilityHint(
			s.metrics.clientSourcePosture,
			classifyClientDeliveryPosture(
				s.metrics.clientSourcePosture,
				s.metrics.itemsStreamed,
				s.metrics.errors,
			),
		),
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

func (s *StreamingService) recordServerSourcePosture() {
	postureBackend, ok := s.backend.(StreamingSourcePostureBackend)
	if !ok {
		return
	}
	posture := postureBackend.ServerStreamSourcePosture()
	if posture == "" {
		return
	}
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.serverSourcePosture = posture
}

func (s *StreamingService) recordClientSourcePosture() {
	postureBackend, ok := s.backend.(StreamingSourcePostureBackend)
	if !ok {
		return
	}
	posture := postureBackend.ClientStreamSourcePosture()
	if posture == "" {
		return
	}
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.clientSourcePosture = posture
}

func classifyServerDeliveryPosture(source string, activeStreams int64, itemsStreamed int64, errors int64) string {
	if errors > 0 {
		return "stream-error"
	}
	if activeStreams > 0 {
		if source != "" {
			return "stream-active"
		}
		return "stream-open"
	}
	if itemsStreamed > 0 {
		if source != "" {
			return "stream-delivered"
		}
		return "stream-complete"
	}
	if source != "" {
		return "stream-idle"
	}
	return "stream-unobserved"
}

func classifyClientDeliveryPosture(source string, itemsStreamed int64, errors int64) string {
	if errors > 0 {
		return "client-batch-error"
	}
	if itemsStreamed > 0 {
		if source != "" {
			return "client-batch-delivered"
		}
		return "client-batch-complete"
	}
	if source != "" {
		return "client-batch-idle"
	}
	return "client-batch-unobserved"
}

func buildServerReliabilityHint(source string, delivery string) string {
	switch delivery {
	case "stream-error":
		return "server stream delivery is degraded; inspect backend stability and handler failures"
	case "stream-active":
		if source != "" {
			return "server stream is actively delivering events; continue observing stream health"
		}
		return "server stream is active; continue observing delivery stability"
	case "stream-delivered":
		if source != "" {
			return "server stream delivered successfully from the current source posture"
		}
		return "server stream delivered successfully"
	case "stream-idle":
		return "server stream source is configured but no events have been observed yet"
	case "stream-unobserved":
		return "server stream source has not been observed yet"
	default:
		return "inspect server stream delivery posture before relying on the stream"
	}
}

func buildClientReliabilityHint(source string, delivery string) string {
	switch delivery {
	case "client-batch-error":
		return "client stream batch delivery failed; inspect backend processing stability"
	case "client-batch-delivered":
		if source != "" {
			return "client stream batch completed successfully through the current source posture"
		}
		return "client stream batch completed successfully"
	case "client-batch-idle":
		return "client stream source is configured but no batch has completed yet"
	case "client-batch-unobserved":
		return "client stream source has not been observed yet"
	default:
		return "inspect client stream delivery posture before relying on batch processing"
	}
}
