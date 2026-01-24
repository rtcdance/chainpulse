package query

import (
	"context"
	"fmt"
	"time"

	"chainpulse/pkg/core"
)

// EventRetrievalService provides unified event retrieval from MongoDB and PostgreSQL
type EventRetrievalService struct {
	eventStore    EventStore
	metadataStore EventMetadataStore
	logger        core.Logger
	metrics       core.MetricsCollector
	initialized   bool
}

// NewEventRetrievalService creates a new event retrieval service
func NewEventRetrievalService(
	eventStore EventStore,
	metadataStore EventMetadataStore,
	logger core.Logger,
	metrics core.MetricsCollector,
) *EventRetrievalService {
	return &EventRetrievalService{
		eventStore:    eventStore,
		metadataStore: metadataStore,
		logger:        logger,
		metrics:       metrics,
		initialized:   false,
	}
}

// Initialize initializes the event retrieval service
func (s *EventRetrievalService) Initialize(ctx context.Context) error {
	if s.initialized {
		return nil
	}

	if s.eventStore == nil {
		return fmt.Errorf("event store is required")
	}

	if s.metadataStore == nil {
		return fmt.Errorf("metadata store is required")
	}

	s.initialized = true
	s.logger.Info("Event retrieval service initialized")
	return nil
}

// EventWithMetadata represents an event with its metadata
type EventWithMetadata struct {
	Event    *core.BlockchainEvent
	Metadata *EventMetadata
}

// GetEventWithMetadata retrieves an event and its metadata by event ID
func (s *EventRetrievalService) GetEventWithMetadata(ctx context.Context, eventID string) (*EventWithMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event retrieval service not initialized")
	}

	if eventID == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordGauge("event_retrieval_get_with_metadata_time_ms", float64(duration), nil)
	}()

	// Get event from MongoDB
	event, err := s.eventStore.GetEvent(ctx, eventID)
	if err != nil {
		s.logger.Error("Failed to get event", "eventId", eventID, "error", err.Error())
		s.metrics.RecordCounter("event_retrieval_get_with_metadata_error", 1, nil)
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if event == nil {
		return nil, nil
	}

	// Get metadata from PostgreSQL
	metadata, err := s.metadataStore.GetMetadata(ctx, eventID)
	if err != nil {
		s.logger.Error("Failed to get metadata", "eventId", eventID, "error", err.Error())
		s.metrics.RecordCounter("event_retrieval_get_with_metadata_error", 1, nil)
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	s.metrics.RecordCounter("event_retrieval_get_with_metadata_success", 1, nil)
	return &EventWithMetadata{
		Event:    event,
		Metadata: metadata,
	}, nil
}

// GetEventsByChainWithMetadata retrieves events for a chain with their metadata
func (s *EventRetrievalService) GetEventsByChainWithMetadata(
	ctx context.Context,
	chainID int,
	limit int,
	offset int,
) ([]*EventWithMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event retrieval service not initialized")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordGauge("event_retrieval_get_by_chain_with_metadata_time_ms", float64(duration), nil)
	}()

	// Get events from MongoDB
	events, err := s.eventStore.GetEventsByChain(ctx, chainID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get events by chain", "chainId", chainID, "error", err.Error())
		s.metrics.RecordCounter("event_retrieval_get_by_chain_with_metadata_error", 1, nil)
		return nil, fmt.Errorf("failed to get events by chain: %w", err)
	}

	if len(events) == 0 {
		return []*EventWithMetadata{}, nil
	}

	// Get metadata for all events
	result := make([]*EventWithMetadata, 0, len(events))
	for _, event := range events {
		metadata, err := s.metadataStore.GetMetadata(ctx, event.ID)
		if err != nil {
			s.logger.Warn("Failed to get metadata for event", "eventId", event.ID, "error", err.Error())
			// Continue with nil metadata if retrieval fails
			result = append(result, &EventWithMetadata{
				Event:    event,
				Metadata: nil,
			})
			continue
		}

		result = append(result, &EventWithMetadata{
			Event:    event,
			Metadata: metadata,
		})
	}

	s.metrics.RecordCounter("event_retrieval_get_by_chain_with_metadata_success", int64(len(result)), nil)
	return result, nil
}

// GetEventsByContractWithMetadata retrieves events for a contract with their metadata
func (s *EventRetrievalService) GetEventsByContractWithMetadata(
	ctx context.Context,
	contractAddress string,
	limit int,
	offset int,
) ([]*EventWithMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event retrieval service not initialized")
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordGauge("event_retrieval_get_by_contract_with_metadata_time_ms", float64(duration), nil)
	}()

	// Get events from MongoDB
	events, err := s.eventStore.GetEventsByContract(ctx, contractAddress, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get events by contract", "contractAddress", contractAddress, "error", err.Error())
		s.metrics.RecordCounter("event_retrieval_get_by_contract_with_metadata_error", 1, nil)
		return nil, fmt.Errorf("failed to get events by contract: %w", err)
	}

	if len(events) == 0 {
		return []*EventWithMetadata{}, nil
	}

	// Get metadata for all events
	result := make([]*EventWithMetadata, 0, len(events))
	for _, event := range events {
		metadata, err := s.metadataStore.GetMetadata(ctx, event.ID)
		if err != nil {
			s.logger.Warn("Failed to get metadata for event", "eventId", event.ID, "error", err.Error())
			// Continue with nil metadata if retrieval fails
			result = append(result, &EventWithMetadata{
				Event:    event,
				Metadata: nil,
			})
			continue
		}

		result = append(result, &EventWithMetadata{
			Event:    event,
			Metadata: metadata,
		})
	}

	s.metrics.RecordCounter("event_retrieval_get_by_contract_with_metadata_success", int64(len(result)), nil)
	return result, nil
}

// GetEventsByEventNameWithMetadata retrieves events by event name with their metadata
func (s *EventRetrievalService) GetEventsByEventNameWithMetadata(
	ctx context.Context,
	eventName string,
	limit int,
	offset int,
) ([]*EventWithMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event retrieval service not initialized")
	}

	if eventName == "" {
		return nil, fmt.Errorf("event name is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordGauge("event_retrieval_get_by_name_with_metadata_time_ms", float64(duration), nil)
	}()

	// Get events from MongoDB
	events, err := s.eventStore.GetEventsByEventName(ctx, eventName, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get events by name", "eventName", eventName, "error", err.Error())
		s.metrics.RecordCounter("event_retrieval_get_by_name_with_metadata_error", 1, nil)
		return nil, fmt.Errorf("failed to get events by name: %w", err)
	}

	if len(events) == 0 {
		return []*EventWithMetadata{}, nil
	}

	// Get metadata for all events
	result := make([]*EventWithMetadata, 0, len(events))
	for _, event := range events {
		metadata, err := s.metadataStore.GetMetadata(ctx, event.ID)
		if err != nil {
			s.logger.Warn("Failed to get metadata for event", "eventId", event.ID, "error", err.Error())
			// Continue with nil metadata if retrieval fails
			result = append(result, &EventWithMetadata{
				Event:    event,
				Metadata: nil,
			})
			continue
		}

		result = append(result, &EventWithMetadata{
			Event:    event,
			Metadata: metadata,
		})
	}

	s.metrics.RecordCounter("event_retrieval_get_by_name_with_metadata_success", int64(len(result)), nil)
	return result, nil
}

// Health returns the health status of the event retrieval service
func (s *EventRetrievalService) Health(ctx context.Context) *core.HealthStatus {
	if !s.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "event retrieval service not initialized",
		}
	}

	// Check both stores
	eventStoreHealth := s.eventStore.Health(ctx)
	metadataStoreHealth := s.metadataStore.Health(ctx)

	if eventStoreHealth.Status != "healthy" || metadataStoreHealth.Status != "healthy" {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "one or more data stores are unhealthy",
		}
	}

	return &core.HealthStatus{
		Status: "healthy",
	}
}

// Close closes the event retrieval service
func (s *EventRetrievalService) Close(ctx context.Context) error {
	if !s.initialized {
		return nil
	}

	s.initialized = false
	return nil
}
