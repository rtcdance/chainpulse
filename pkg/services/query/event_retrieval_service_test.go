package query

import (
	"context"
	"fmt"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"go.uber.org/mock/gomock"
)

func TestNewEventRetrievalService(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	if s == nil {
		t.Fatal("expected non-nil service")
	}
	if s.initialized {
		t.Error("expected service to not be initialized")
	}
}

func TestEventRetrievalService_GetEventReader(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	reader := s.GetEventReader()
	if reader != nil {
		t.Error("expected nil reader when eventStore is nil")
	}
}

func TestEventRetrievalService_Initialize_NilEventStore(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	err := s.Initialize(context.Background())
	if err == nil {
		t.Error("expected error when event store is nil")
	}
}

func TestEventRetrievalService_Health_NotInitialized(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	health := s.Health(context.Background())
	if health == nil {
		t.Fatal("expected non-nil health status")
	}
	if health.Status != "unhealthy" {
		t.Errorf("expected unhealthy status, got %s", health.Status)
	}
}

func TestEventRetrievalService_Close_NotInitialized(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	err := s.Close(context.Background())
	if err != nil {
		t.Errorf("expected no error closing uninitialized service, got %v", err)
	}
}

func TestEventRetrievalService_Initialize_AlreadyInitialized(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	s.initialized = true
	err := s.Initialize(context.Background())
	if err != nil {
		t.Errorf("expected no error when already initialized, got %v", err)
	}
}

func TestEventRetrievalService_Close_Initialized(t *testing.T) {
	t.Parallel()

	s := NewEventRetrievalService(nil, nil, nil, nil)
	s.initialized = true
	err := s.Close(context.Background())
	if err != nil {
		t.Errorf("expected no error closing initialized service, got %v", err)
	}
	if s.initialized {
		t.Error("expected service to be marked as not initialized after close")
	}
}

func newInitService(t *testing.T) (*EventRetrievalService, *MockEventStore, *MockEventMetadataStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	es := NewMockEventStore(ctrl)
	ms := NewMockEventMetadataStore(ctrl)
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	s := NewEventRetrievalService(es, ms, logger, metrics)
	s.initialized = true
	return s, es, ms
}

func TestGetEventWithMetadata_NotInitialized(t *testing.T) {
	t.Parallel()
	s := NewEventRetrievalService(nil, nil, nil, nil)
	_, err := s.GetEventWithMetadata(context.Background(), "evt-1")
	if err == nil || err.Error() != "event retrieval service not initialized" {
		t.Errorf("expected 'not initialized' error, got %v", err)
	}
}

func TestGetEventWithMetadata_EmptyID(t *testing.T) {
	t.Parallel()
	s, _, _ := newInitService(t)
	_, err := s.GetEventWithMetadata(context.Background(), "")
	if err == nil || err.Error() != "event ID is required" {
		t.Errorf("expected 'event ID is required' error, got %v", err)
	}
}

func TestGetEventWithMetadata_StoreError(t *testing.T) {
	t.Parallel()
	s, es, _ := newInitService(t)
	es.EXPECT().GetEvent(gomock.Any(), "evt-1").Return(nil, fmt.Errorf("db error"))
	_, err := s.GetEventWithMetadata(context.Background(), "evt-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEventWithMetadata_NotFound(t *testing.T) {
	t.Parallel()
	s, es, _ := newInitService(t)
	es.EXPECT().GetEvent(gomock.Any(), "evt-1").Return(nil, nil)
	result, err := s.GetEventWithMetadata(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when event not found")
	}
}

func TestGetEventWithMetadata_MetadataError(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	event := &core.BlockchainEvent{ID: "evt-1"}
	es.EXPECT().GetEvent(gomock.Any(), "evt-1").Return(event, nil)
	ms.EXPECT().GetMetadata(gomock.Any(), "evt-1").Return(nil, fmt.Errorf("meta error"))
	_, err := s.GetEventWithMetadata(context.Background(), "evt-1")
	if err == nil {
		t.Fatal("expected metadata error")
	}
}

func TestGetEventWithMetadata_Success(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	event := &core.BlockchainEvent{ID: "evt-1"}
	metadata := &EventMetadata{EventID: "evt-1"}
	es.EXPECT().GetEvent(gomock.Any(), "evt-1").Return(event, nil)
	ms.EXPECT().GetMetadata(gomock.Any(), "evt-1").Return(metadata, nil)
	result, err := s.GetEventWithMetadata(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Event != event || result.Metadata != metadata {
		t.Error("result mismatch")
	}
}

func TestGetEventsByChainWithMetadata_NotInitialized(t *testing.T) {
	t.Parallel()
	s := NewEventRetrievalService(nil, nil, nil, nil)
	_, err := s.GetEventsByChainWithMetadata(context.Background(), 1, 10, 0)
	if err == nil || err.Error() != "event retrieval service not initialized" {
		t.Errorf("expected 'not initialized' error, got %v", err)
	}
}

func TestGetEventsByChainWithMetadata_StoreError(t *testing.T) {
	t.Parallel()
	s, es, _ := newInitService(t)
	es.EXPECT().GetEventsByChain(gomock.Any(), 1, 10, 0).Return(nil, fmt.Errorf("db error"))
	_, err := s.GetEventsByChainWithMetadata(context.Background(), 1, 10, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEventsByChainWithMetadata_EmptyResult(t *testing.T) {
	t.Parallel()
	s, es, _ := newInitService(t)
	es.EXPECT().GetEventsByChain(gomock.Any(), 1, 10, 0).Return([]*core.BlockchainEvent{}, nil)
	result, err := s.GetEventsByChainWithMetadata(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestGetEventsByChainWithMetadata_AttachError(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	events := []*core.BlockchainEvent{{ID: "evt-1"}}
	es.EXPECT().GetEventsByChain(gomock.Any(), 1, 10, 0).Return(events, nil)
	ms.EXPECT().GetMetadataBatch(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("batch error"))
	_, err := s.GetEventsByChainWithMetadata(context.Background(), 1, 10, 0)
	if err == nil {
		t.Fatal("expected metadata batch error")
	}
}

func TestGetEventsByChainWithMetadata_Success(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	events := []*core.BlockchainEvent{{ID: "evt-1"}}
	metaMap := map[string]*EventMetadata{"evt-1": {EventID: "evt-1"}}
	es.EXPECT().GetEventsByChain(gomock.Any(), 1, 10, 0).Return(events, nil)
	ms.EXPECT().GetMetadataBatch(gomock.Any(), gomock.Any()).Return(metaMap, nil)
	result, err := s.GetEventsByChainWithMetadata(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestGetEventsByContractWithMetadata_NotInitialized(t *testing.T) {
	t.Parallel()
	s := NewEventRetrievalService(nil, nil, nil, nil)
	_, err := s.GetEventsByContractWithMetadata(context.Background(), "0x123", 10, 0)
	if err == nil || err.Error() != "event retrieval service not initialized" {
		t.Errorf("expected 'not initialized' error, got %v", err)
	}
}

func TestGetEventsByContractWithMetadata_EmptyAddress(t *testing.T) {
	t.Parallel()
	s, _, _ := newInitService(t)
	_, err := s.GetEventsByContractWithMetadata(context.Background(), "", 10, 0)
	if err == nil || err.Error() != "contract address is required" {
		t.Errorf("expected 'contract address is required' error, got %v", err)
	}
}

func TestGetEventsByContractWithMetadata_Success(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	events := []*core.BlockchainEvent{{ID: "evt-1"}}
	metaMap := map[string]*EventMetadata{"evt-1": {EventID: "evt-1"}}
	es.EXPECT().GetEventsByContract(gomock.Any(), "0x123", 10, 0).Return(events, nil)
	ms.EXPECT().GetMetadataBatch(gomock.Any(), gomock.Any()).Return(metaMap, nil)
	result, err := s.GetEventsByContractWithMetadata(context.Background(), "0x123", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestGetEventsByEventNameWithMetadata_NotInitialized(t *testing.T) {
	t.Parallel()
	s := NewEventRetrievalService(nil, nil, nil, nil)
	_, err := s.GetEventsByEventNameWithMetadata(context.Background(), "Transfer", 10, 0)
	if err == nil || err.Error() != "event retrieval service not initialized" {
		t.Errorf("expected 'not initialized' error, got %v", err)
	}
}

func TestGetEventsByEventNameWithMetadata_EmptyName(t *testing.T) {
	t.Parallel()
	s, _, _ := newInitService(t)
	_, err := s.GetEventsByEventNameWithMetadata(context.Background(), "", 10, 0)
	if err == nil || err.Error() != "event name is required" {
		t.Errorf("expected 'event name is required' error, got %v", err)
	}
}

func TestGetEventsByEventNameWithMetadata_Success(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	events := []*core.BlockchainEvent{{ID: "evt-1"}}
	metaMap := map[string]*EventMetadata{"evt-1": {EventID: "evt-1"}}
	es.EXPECT().GetEventsByEventName(gomock.Any(), "Transfer", 10, 0).Return(events, nil)
	ms.EXPECT().GetMetadataBatch(gomock.Any(), gomock.Any()).Return(metaMap, nil)
	result, err := s.GetEventsByEventNameWithMetadata(context.Background(), "Transfer", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestGetEventsByCorrelationID_NotInitialized(t *testing.T) {
	t.Parallel()
	s := NewEventRetrievalService(nil, nil, nil, nil)
	_, err := s.GetEventsByCorrelationID(context.Background(), "corr-1", 10, 0)
	if err == nil || err.Error() != "event retrieval service not initialized" {
		t.Errorf("expected 'not initialized' error, got %v", err)
	}
}

func TestGetEventsByCorrelationID_EmptyID(t *testing.T) {
	t.Parallel()
	s, _, _ := newInitService(t)
	_, err := s.GetEventsByCorrelationID(context.Background(), "", 10, 0)
	if err == nil || err.Error() != "correlation ID is required" {
		t.Errorf("expected 'correlation ID is required' error, got %v", err)
	}
}

func TestGetEventsByCorrelationID_Success(t *testing.T) {
	t.Parallel()
	s, es, ms := newInitService(t)
	events := []*core.BlockchainEvent{{ID: "evt-1"}}
	metaMap := map[string]*EventMetadata{"evt-1": {EventID: "evt-1"}}
	es.EXPECT().GetEventsByCorrelationID(gomock.Any(), "corr-1", 10, 0).Return(events, nil)
	ms.EXPECT().GetMetadataBatch(gomock.Any(), gomock.Any()).Return(metaMap, nil)
	result, err := s.GetEventsByCorrelationID(context.Background(), "corr-1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestGetEventsByCorrelationID_StoreError(t *testing.T) {
	t.Parallel()
	s, es, _ := newInitService(t)
	es.EXPECT().GetEventsByCorrelationID(gomock.Any(), "corr-1", 10, 0).Return(nil, fmt.Errorf("db error"))
	_, err := s.GetEventsByCorrelationID(context.Background(), "corr-1", 10, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}