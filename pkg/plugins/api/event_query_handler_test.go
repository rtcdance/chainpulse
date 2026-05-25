package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

type mockDomainQueryService struct {
	query       func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error)
	queryByHash func(ctx context.Context, hash string) (*core.BlockchainEvent, error)
}

func (m *mockDomainQueryService) Query(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
	if m.query != nil {
		return m.query(ctx, req)
	}
	return nil, nil
}

func (m *mockDomainQueryService) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	if m.queryByHash != nil {
		return m.queryByHash(ctx, hash)
	}

	return nil, nil
}

func (m *mockDomainQueryService) InvalidateCache(ctx context.Context, key string) error {
	return nil
}

func (m *mockDomainQueryService) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Message: "ok"}
}

type mockEventStore struct {
	getEvent         func(ctx context.Context, eventID string) (*core.BlockchainEvent, error)
	getEventsByChain func(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error)
}

func (m *mockEventStore) Initialize(ctx context.Context) error { return nil }
func (m *mockEventStore) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	return nil
}

func (m *mockEventStore) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	return nil
}

func (m *mockEventStore) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	if m.getEvent != nil {
		return m.getEvent(ctx, eventID)
	}

	return nil, nil
}

func (m *mockEventStore) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if m.getEventsByChain != nil {
		return m.getEventsByChain(ctx, chainID, limit, offset)
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	return nil, false, nil
}
func (m *mockEventStore) CountEvents(ctx context.Context) (int64, error)         { return 0, nil }
func (m *mockEventStore) DeleteExpiredEvents(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockEventStore) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Message: "ok"}
}
func (m *mockEventStore) Close(ctx context.Context) error { return nil }
func (m *mockEventStore) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

type mockMetadataStore struct {
	getMetadata func(ctx context.Context, eventID string) (*query.EventMetadata, error)
}

func (m *mockMetadataStore) Initialize(ctx context.Context) error { return nil }
func (m *mockMetadataStore) InsertMetadata(ctx context.Context, metadata *query.EventMetadata) error {
	return nil
}

func (m *mockMetadataStore) InsertMetadataBatch(ctx context.Context, metadataList []*query.EventMetadata) error {
	return nil
}

func (m *mockMetadataStore) GetMetadata(ctx context.Context, eventID string) (*query.EventMetadata, error) {
	if m.getMetadata != nil {
		return m.getMetadata(ctx, eventID)
	}
	return nil, nil
}

func (m *mockMetadataStore) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*query.EventMetadata, error) {
	return nil, nil
}

func (m *mockMetadataStore) GetMetadataBatch(ctx context.Context, eventIDs []string) (map[string]*query.EventMetadata, error) {
	result := make(map[string]*query.EventMetadata, len(eventIDs))
	for _, id := range eventIDs {
		if meta, err := m.GetMetadata(ctx, id); err != nil {
			return nil, err
		} else if meta != nil {
			result[id] = meta
		}
	}
	return result, nil
}

func (m *mockMetadataStore) UpdateMetadata(ctx context.Context, metadata *query.EventMetadata) error {
	return nil
}

func (m *mockMetadataStore) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Message: "ok"}
}
func (m *mockMetadataStore) Close(ctx context.Context) error { return nil }

func TestEventQueryHandlerDomainFirstSuccess(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		queryByHash: func(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
			return &core.BlockchainEvent{
				ID:             "domain-hit",
				BlockNumber:    123,
				BlockTimestamp: time.Now().Unix(),
				EventName:      "Transfer",
				DecodedData:    map[string]any{"ok": true},
				CreatedAt:      time.Now(),
				ProcessedAt:    time.Now(),
				IndexedAt:      time.Now(),
			}, nil
		},
	})

	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/0x1111111111111111111111111111111111111111111111111111111111111111", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventByID(rr, req, "0x1111111111111111111111111111111111111111111111111111111111111111")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "domain-query" {
		t.Fatalf("expected meta source domain-query, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "domain-service" {
		t.Fatalf("expected querySourcePosture domain-service, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-first" {
		t.Fatalf("expected queryPath domain-first, got %v", got)
	}
	if _, exists := meta["fallbackUsed"]; exists {
		t.Fatalf("expected fallbackUsed to be omitted for false, got %v", meta["fallbackUsed"])
	}
	if got := meta["metadataCompleteness"]; got != "none" {
		t.Fatalf("expected metadataCompleteness none, got %v", got)
	}
	if got := meta["metadataCoveragePosture"]; got != "coverage-missing" {
		t.Fatalf("expected metadataCoveragePosture coverage-missing, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "domain-direct" {
		t.Fatalf("expected consistencyPosture domain-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served directly from domain query path without fallback" {
		t.Fatalf("expected queryReliabilityHint for domain direct, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-first:domain-query:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-first:domain-query:coverage-missing, got %v", got)
	}
	if _, exists := meta["metadataAttachedCount"]; exists {
		t.Fatalf("expected metadataAttachedCount to be omitted for zero, got %v", meta["metadataAttachedCount"])
	}
	if got := meta["metadataMissingCount"]; got != float64(1) {
		t.Fatalf("expected metadataMissingCount 1, got %v", got)
	}
}

func TestEventQueryHandlerDomainFirstFallbackToRetrieval(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{
		getEvent: func(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
			return &core.BlockchainEvent{
				ID:             "retrieval-hit",
				BlockNumber:    456,
				BlockTimestamp: time.Now().Unix(),
				EventName:      "Approval",
				DecodedData:    map[string]any{"fallback": true},
				CreatedAt:      time.Now(),
				ProcessedAt:    time.Now(),
				IndexedAt:      time.Now(),
			}, nil
		},
	}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		queryByHash: func(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
			return nil, errors.New("domain backend unavailable")
		},
	})

	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/0x2222222222222222222222222222222222222222222222222222222222222222", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventByID(rr, req, "0x2222222222222222222222222222222222222222222222222222222222222222")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object in response: %v", payload)
	}
	if got, ok := data["eventId"].(string); !ok || got != "retrieval-hit" {
		t.Fatalf("expected fallback retrieval eventId retrieval-hit, got %v", data["eventId"])
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "event-retrieval" {
		t.Fatalf("expected meta source event-retrieval, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "retrieval-fallback" {
		t.Fatalf("expected querySourcePosture retrieval-fallback, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-first" {
		t.Fatalf("expected queryPath domain-first, got %v", got)
	}
	if got := meta["fallbackUsed"]; got != true {
		t.Fatalf("expected fallbackUsed true, got %v", got)
	}
	if got := meta["metadataCompleteness"]; got != "none" {
		t.Fatalf("expected metadataCompleteness none, got %v", got)
	}
	if got := meta["metadataCoveragePosture"]; got != "coverage-missing" {
		t.Fatalf("expected metadataCoveragePosture coverage-missing, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "fallback-served" {
		t.Fatalf("expected consistencyPosture fallback-served, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served through fallback path; verify query-service availability if this persists" {
		t.Fatalf("expected queryReliabilityHint for fallback, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-first:event-retrieval:fallback:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-first:event-retrieval:fallback:coverage-missing, got %v", got)
	}
	if _, exists := meta["metadataAttachedCount"]; exists {
		t.Fatalf("expected metadataAttachedCount to be omitted for zero, got %v", meta["metadataAttachedCount"])
	}
	if got := meta["metadataMissingCount"]; got != float64(1) {
		t.Fatalf("expected metadataMissingCount 1, got %v", got)
	}
}

func TestEventQueryHandlerConvertEventResponseParsesNumericChainID(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	response := handler.convertEventToResponse(&query.EventWithMetadata{
		Event: &core.BlockchainEvent{
			ID:              "evt-1",
			ChainID:         "31337",
			BlockNumber:     12,
			TransactionHash: common.HexToHash("0x1"),
		},
	})
	if response == nil {
		t.Fatal("expected response")
	}
	if response.ChainID != "31337" {
		t.Fatalf("expected parsed chain id 31337, got %s", response.ChainID)
	}
}

func TestEventQueryHandlerConvertEventResponseResolvesNamedChainID(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	response := handler.convertEventToResponse(&query.EventWithMetadata{
		Event: &core.BlockchainEvent{
			ID:              "evt-2",
			ChainID:         "ethereum",
			BlockNumber:     13,
			TransactionHash: common.HexToHash("0x2"),
		},
	})
	if response == nil {
		t.Fatal("expected response")
	}
	if response.ChainID != "1" {
		t.Fatalf("expected resolved chain id 1, got %s", response.ChainID)
	}
}

func TestEventQueryHandlerGetByChainIncludesQueryMeta(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{
		getEventsByChain: func(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
			return []*core.BlockchainEvent{
				{
					ID:             "event-1",
					BlockNumber:    100,
					BlockTimestamp: time.Now().Unix(),
					EventName:      "Transfer",
					DecodedData:    map[string]any{"value": "1"},
					CreatedAt:      time.Now(),
					ProcessedAt:    time.Now(),
					IndexedAt:      time.Now(),
				},
				{
					ID:             "event-2",
					BlockNumber:    101,
					BlockTimestamp: time.Now().Unix(),
					EventName:      "Approval",
					DecodedData:    map[string]any{"value": "2"},
					CreatedAt:      time.Now(),
					ProcessedAt:    time.Now(),
					IndexedAt:      time.Now(),
				},
			}, nil
		},
	}, &mockMetadataStore{
		getMetadata: func(ctx context.Context, eventID string) (*query.EventMetadata, error) {
			if eventID == "event-1" {
				return &query.EventMetadata{EventID: eventID, ProcessedAt: time.Now()}, nil
			}
			return nil, nil
		},
	}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/chain/1?limit=2&offset=0", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventsByChain(rr, req, "1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "event-retrieval" {
		t.Fatalf("expected meta source event-retrieval, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "retrieval-service" {
		t.Fatalf("expected querySourcePosture retrieval-service, got %v", got)
	}
	if got := meta["queryPath"]; got != "retrieval-chain" {
		t.Fatalf("expected queryPath retrieval-list, got %v", got)
	}
	if got := meta["metadataCompleteness"]; got != "partial" {
		t.Fatalf("expected metadataCompleteness partial, got %v", got)
	}
	if got := meta["metadataCoveragePosture"]; got != "coverage-partial" {
		t.Fatalf("expected metadataCoveragePosture coverage-partial, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "retrieval-partial" {
		t.Fatalf("expected consistencyPosture retrieval-partial, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served with partial metadata coverage; verify metadata completeness before relying on full event context" {
		t.Fatalf("expected queryReliabilityHint for retrieval partial, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "retrieval-chain:event-retrieval:coverage-partial" {
		t.Fatalf("expected queryExecutionSummary retrieval-list:event-retrieval:coverage-partial, got %v", got)
	}
	if got := meta["metadataAttachedCount"]; got != float64(1) {
		t.Fatalf("expected metadataAttachedCount 1, got %v", got)
	}
	if got := meta["metadataMissingCount"]; got != float64(1) {
		t.Fatalf("expected metadataMissingCount 1, got %v", got)
	}
	if _, exists := meta["fallbackUsed"]; exists {
		t.Fatalf("expected fallbackUsed to be omitted for false, got %v", meta["fallbackUsed"])
	}
	if got := meta["resultCount"]; got != float64(2) {
		t.Fatalf("expected resultCount 2, got %v", got)
	}
}

func TestEventQueryHandlerGetByChainDomainQueryMeta(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if req.Collection != "events" {
				t.Fatalf("expected collection events, got %q", req.Collection)
			}
			if req.QueryType != "mongodb" {
				t.Fatalf("expected query type mongodb, got %q", req.QueryType)
			}
			if got := req.Filter["chainId"]; got == nil {
				t.Fatal("expected chainId filter")
			} else if inMap, ok := got.(map[string]any); ok {
				values, _ := inMap["$in"].([]any)
				found := false
				for _, v := range values {
					if v == 1 || v == "1" {
						found = true
					}
				}
				if !found {
					t.Fatalf(`expected $in filter to include 1 or "1", got %v`, values)
				}
			} else if got != 1 {
				t.Fatalf("expected chainId filter 1 or $in filter, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "domain-chain-1",
						BlockNumber:    222,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]any{"path": "domain-chain"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "mongodb",
			}, nil
		},
	})
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/chain/1?limit=1&offset=0", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventsByChain(rr, req, "1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "mongodb" {
		t.Fatalf("expected meta source mongodb, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "mongodb-live" {
		t.Fatalf("expected querySourcePosture mongodb-live, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-chain" {
		t.Fatalf("expected queryPath domain-chain, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served directly from query-service live store path" {
		t.Fatalf("expected queryReliabilityHint for mongodb-live query-service-direct, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-chain:mongodb:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-chain:mongodb:coverage-missing, got %v", got)
	}
}

func TestEventQueryHandlerGetByChainStringDomainQueryMeta(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if got := req.Filter["chainId"]; got != "ethereum" {
				t.Fatalf("expected chainId filter ethereum, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "domain-chain-eth-1",
						ChainID:        "ethereum",
						BlockNumber:    333,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]any{"path": "domain-chain-string"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "monolithic-indexing",
			}, nil
		},
	})
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/chain/ethereum?limit=1&offset=0", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventsByChain(rr, req, "ethereum")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "monolithic-indexing" {
		t.Fatalf("expected meta source monolithic-indexing, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-chain" {
		t.Fatalf("expected queryPath domain-chain, got %v", got)
	}
}

func TestEventQueryHandlerGetAllEventsDomainListMeta(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "domain-list-1",
						BlockNumber:    111,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]any{"path": "domain-list"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "cache",
			}, nil
		},
	})

	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events?limit=1&offset=0", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetAllEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "cache" {
		t.Fatalf("expected meta source cache, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "cache-hit" {
		t.Fatalf("expected querySourcePosture cache-hit, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-all" {
		t.Fatalf("expected queryPath domain-all, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served from query-service cache; verify freshness expectations before treating as latest" {
		t.Fatalf("expected queryReliabilityHint for cache-hit query-service-direct, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-all:cache:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-all:cache:coverage-missing, got %v", got)
	}
}

func TestEventQueryHandlerGetByNameDomainQueryMeta(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if req.Collection != "events" {
				t.Fatalf("expected collection events, got %q", req.Collection)
			}
			if req.QueryType != "mongodb" {
				t.Fatalf("expected query type mongodb, got %q", req.QueryType)
			}
			if got := req.Filter["eventName"]; got != "Transfer" {
				t.Fatalf("expected eventName filter Transfer, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "domain-name-1",
						BlockNumber:    333,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]any{"path": "domain-name"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "mongodb",
			}, nil
		},
	})
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/name/Transfer?limit=1&offset=0", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventsByName(rr, req, "Transfer")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "mongodb" {
		t.Fatalf("expected meta source mongodb, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "mongodb-live" {
		t.Fatalf("expected querySourcePosture mongodb-live, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-name" {
		t.Fatalf("expected queryPath domain-name, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served directly from query-service live store path" {
		t.Fatalf("expected queryReliabilityHint for mongodb-live query-service-direct, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-name:mongodb:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-name:mongodb:coverage-missing, got %v", got)
	}
}

func TestEventQueryHandlerGetByContractDomainQueryMeta(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)

	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval service: %v", err)
	}

	handler := NewEventQueryHandler(retrieval, logger, metrics)
	handler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if req.Collection != "events" {
				t.Fatalf("expected collection events, got %q", req.Collection)
			}
			if req.QueryType != "mongodb" {
				t.Fatalf("expected query type mongodb, got %q", req.QueryType)
			}
			if got := req.Filter["contractAddress"]; got != "0xabc0000000000000000000000000000000000000" {
				t.Fatalf("expected contractAddress filter, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "domain-contract-1",
						BlockNumber:    444,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]any{"path": "domain-contract"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "mongodb",
			}, nil
		},
	})
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/contract/0xabc0000000000000000000000000000000000000?limit=1&offset=0", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetEventsByContract(rr, req, "0xabc0000000000000000000000000000000000000")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta object in response: %v", payload)
	}
	if got := meta["source"]; got != "mongodb" {
		t.Fatalf("expected meta source mongodb, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "mongodb-live" {
		t.Fatalf("expected querySourcePosture mongodb-live, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-contract" {
		t.Fatalf("expected queryPath domain-contract, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served directly from query-service live store path" {
		t.Fatalf("expected queryReliabilityHint for mongodb-live query-service-direct, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-contract:mongodb:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-contract:mongodb:coverage-missing, got %v", got)
	}
}

func TestEventQueryHandlerParseInt64Param(t *testing.T) {
	t.Parallel()

	handler := &EventQueryHandler{}

	tests := []struct {
		name         string
		queryString  string
		paramName    string
		defaultValue int64
		want         int64
	}{
		{
			name:         "valid positive int64",
			queryString:  "value=12345",
			paramName:    "value",
			defaultValue: 0,
			want:         12345,
		},
		{
			name:         "valid negative int64",
			queryString:  "value=-9999",
			paramName:    "value",
			defaultValue: 0,
			want:         -9999,
		},
		{
			name:         "valid zero",
			queryString:  "value=0",
			paramName:    "value",
			defaultValue: 100,
			want:         0,
		},
		{
			name:         "empty value returns default",
			queryString:  "value=",
			paramName:    "value",
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "missing param returns default",
			queryString:  "other=1",
			paramName:    "value",
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "non-numeric returns default",
			queryString:  "value=abc",
			paramName:    "value",
			defaultValue: 10,
			want:         10,
		},
		{
			name:         "float string returns default",
			queryString:  "value=3.14",
			paramName:    "value",
			defaultValue: 99,
			want:         99,
		},
		{
			name:         "max int64",
			queryString:  "value=9223372036854775807",
			paramName:    "value",
			defaultValue: 0,
			want:         9223372036854775807,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			got := handler.parseInt64Param(req, tt.paramName, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseInt64Param() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEventQueryHandlerParseUint64Param(t *testing.T) {
	t.Parallel()

	handler := &EventQueryHandler{}

	tests := []struct {
		name         string
		queryString  string
		paramName    string
		defaultValue uint64
		want         uint64
	}{
		{
			name:         "valid positive uint64",
			queryString:  "block=12345",
			paramName:    "block",
			defaultValue: 0,
			want:         12345,
		},
		{
			name:         "valid zero",
			queryString:  "block=0",
			paramName:    "block",
			defaultValue: 100,
			want:         0,
		},
		{
			name:         "empty value returns default",
			queryString:  "block=",
			paramName:    "block",
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "missing param returns default",
			queryString:  "other=1",
			paramName:    "block",
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "negative returns default",
			queryString:  "block=-1",
			paramName:    "block",
			defaultValue: 10,
			want:         10,
		},
		{
			name:         "non-numeric returns default",
			queryString:  "block=xyz",
			paramName:    "block",
			defaultValue: 5,
			want:         5,
		},
		{
			name:         "float string returns default",
			queryString:  "block=2.5",
			paramName:    "block",
			defaultValue: 99,
			want:         99,
		},
		{
			name:         "max uint64",
			queryString:  "block=18446744073709551615",
			paramName:    "block",
			defaultValue: 0,
			want:         18446744073709551615,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			got := handler.parseUint64Param(req, tt.paramName, tt.defaultValue)
			if got != tt.want {
				t.Errorf("parseUint64Param() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidateFilterParams(t *testing.T) {
	t.Parallel()

	handler := &EventQueryHandler{}

	tests := []struct {
		name string
		fp   *filterParams
		want string
	}{
		{
			name: "valid defaults",
			fp:   &filterParams{Limit: 20, Offset: 0},
			want: "",
		},
		{
			name: "valid with filters",
			fp: &filterParams{
				FromBlock: 100,
				ToBlock:   200,
				Limit:     50,
				Offset:    0,
				Status:    "confirmed",
			},
			want: "",
		},
		{
			name: "from_block greater than to_block",
			fp: &filterParams{
				FromBlock: 200,
				ToBlock:   100,
				Limit:     20,
			},
			want: "from_block must be less than or equal to to_block",
		},
		{
			name: "from_time greater than to_time",
			fp: &filterParams{
				FromTime: 2000,
				ToTime:   1000,
				Limit:    20,
			},
			want: "from_time must be less than or equal to to_time",
		},
		{
			name: "limit zero",
			fp: &filterParams{
				Limit: 0,
			},
			want: "limit must be between 1 and 1000",
		},
		{
			name: "limit negative",
			fp: &filterParams{
				Limit: -5,
			},
			want: "limit must be between 1 and 1000",
		},
		{
			name: "limit exceeds max",
			fp: &filterParams{
				Limit: 1001,
			},
			want: "limit must be between 1 and 1000",
		},
		{
			name: "limit at max boundary valid",
			fp: &filterParams{
				Limit: 1000,
			},
			want: "",
		},
		{
			name: "limit at min boundary valid",
			fp: &filterParams{
				Limit: 1,
			},
			want: "",
		},
		{
			name: "negative offset",
			fp: &filterParams{
				Limit:  20,
				Offset: -1,
			},
			want: "offset must be greater than or equal to 0",
		},
		{
			name: "invalid status",
			fp: &filterParams{
				Limit:  20,
				Status: "unknown",
			},
			want: "status must be one of: pending, confirmed, failed, reorged",
		},
		{
			name: "valid status pending",
			fp: &filterParams{
				Limit:  20,
				Status: "pending",
			},
			want: "",
		},
		{
			name: "valid status failed",
			fp: &filterParams{
				Limit:  20,
				Status: "failed",
			},
			want: "",
		},
		{
			name: "valid status reorged",
			fp: &filterParams{
				Limit:  20,
				Status: "reorged",
			},
			want: "",
		},
		{
			name: "empty status valid",
			fp: &filterParams{
				Limit:  20,
				Status: "",
			},
			want: "",
		},
		{
			name: "from_block equals to_block valid",
			fp: &filterParams{
				FromBlock: 100,
				ToBlock:   100,
				Limit:     20,
			},
			want: "",
		},
		{
			name: "from_time equals to_time valid",
			fp: &filterParams{
				FromTime: 1000,
				ToTime:   1000,
				Limit:    20,
			},
			want: "",
		},
		{
			name: "to_block only valid",
			fp: &filterParams{
				ToBlock: 200,
				Limit:   20,
			},
			want: "",
		},
		{
			name: "from_block only valid",
			fp: &filterParams{
				FromBlock: 100,
				Limit:     20,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := handler.validateFilterParams(tt.fp)
			if got != tt.want {
				t.Errorf("validateFilterParams() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyEventQueryMetadataCoveragePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		resultCount   int
		attachedCount int
		want          string
	}{
		{
			name:          "zero result count",
			resultCount:   0,
			attachedCount: 0,
			want:          "coverage-empty",
		},
		{
			name:          "negative result count",
			resultCount:   -1,
			attachedCount: 0,
			want:          "coverage-empty",
		},
		{
			name:          "no metadata attached",
			resultCount:   10,
			attachedCount: 0,
			want:          "coverage-missing",
		},
		{
			name:          "negative attached count",
			resultCount:   10,
			attachedCount: -1,
			want:          "coverage-missing",
		},
		{
			name:          "all attached",
			resultCount:   5,
			attachedCount: 5,
			want:          "coverage-complete",
		},
		{
			name:          "more attached than results",
			resultCount:   5,
			attachedCount: 10,
			want:          "coverage-complete",
		},
		{
			name:          "partial coverage",
			resultCount:   10,
			attachedCount: 3,
			want:          "coverage-partial",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyEventQueryMetadataCoveragePosture(tt.resultCount, tt.attachedCount)
			if got != tt.want {
				t.Errorf("classifyEventQueryMetadataCoveragePosture(%d, %d) = %q, want %q", tt.resultCount, tt.attachedCount, got, tt.want)
			}
		})
	}
}

func TestClassifyEventQueryConsistencyPosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		source          string
		queryPath       string
		fallbackUsed    bool
		coveragePosture string
		want            string
	}{
		{
			name:            "domain direct",
			source:          "domain-query",
			queryPath:       "domain-first",
			fallbackUsed:    false,
			coveragePosture: "coverage-missing",
			want:            "domain-direct",
		},
		{
			name:            "domain-first with fallback",
			source:          "domain-query",
			queryPath:       "domain-first",
			fallbackUsed:    true,
			coveragePosture: "",
			want:            "fallback-served",
		},
		{
			name:            "query-service-direct",
			source:          "mongodb",
			queryPath:       "domain-chain",
			fallbackUsed:    false,
			coveragePosture: "coverage-missing",
			want:            "query-service-direct",
		},
		{
			name:            "query-service-direct with domain-all",
			source:          "cache",
			queryPath:       "domain-all",
			fallbackUsed:    false,
			coveragePosture: "coverage-missing",
			want:            "query-service-direct",
		},
		{
			name:            "fallback served",
			source:          "event-retrieval",
			queryPath:       "retrieval-chain",
			fallbackUsed:    true,
			coveragePosture: "",
			want:            "fallback-served",
		},
		{
			name:            "retrieval complete",
			source:          "event-retrieval",
			queryPath:       "retrieval-list",
			fallbackUsed:    false,
			coveragePosture: "coverage-complete",
			want:            "retrieval-complete",
		},
		{
			name:            "retrieval partial",
			source:          "event-retrieval",
			queryPath:       "retrieval-chain",
			fallbackUsed:    false,
			coveragePosture: "coverage-partial",
			want:            "retrieval-partial",
		},
		{
			name:            "retrieval metadata missing",
			source:          "event-retrieval",
			queryPath:       "retrieval-list",
			fallbackUsed:    false,
			coveragePosture: "coverage-missing",
			want:            "retrieval-metadata-missing",
		},
		{
			name:            "empty result",
			source:          "event-retrieval",
			queryPath:       "retrieval-list",
			fallbackUsed:    false,
			coveragePosture: "coverage-empty",
			want:            "empty-result",
		},
		{
			name:            "unknown consistency",
			source:          "unknown-source",
			queryPath:       "unknown-path",
			fallbackUsed:    false,
			coveragePosture: "something-else",
			want:            "consistency-unknown",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyEventQueryConsistencyPosture(tt.source, tt.queryPath, tt.fallbackUsed, tt.coveragePosture)
			if got != tt.want {
				t.Errorf("classifyEventQueryConsistencyPosture() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyEventQuerySourcePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		source       string
		fallbackUsed bool
		cacheHit     bool
		want         string
	}{
		{
			name:         "cache hit via boolean",
			source:       "mongodb",
			fallbackUsed: false,
			cacheHit:     true,
			want:         "cache-hit",
		},
		{
			name:         "cache hit via source string",
			source:       "cache",
			fallbackUsed: false,
			cacheHit:     false,
			want:         "cache-hit",
		},
		{
			name:         "retrieval fallback",
			source:       "event-retrieval",
			fallbackUsed: true,
			cacheHit:     false,
			want:         "retrieval-fallback",
		},
		{
			name:         "domain service",
			source:       "domain-query",
			fallbackUsed: false,
			cacheHit:     false,
			want:         "domain-service",
		},
		{
			name:         "retrieval service",
			source:       "event-retrieval",
			fallbackUsed: false,
			cacheHit:     false,
			want:         "retrieval-service",
		},
		{
			name:         "mongodb live",
			source:       "mongodb",
			fallbackUsed: false,
			cacheHit:     false,
			want:         "mongodb-live",
		},
		{
			name:         "postgres fallback",
			source:       "postgresql",
			fallbackUsed: false,
			cacheHit:     false,
			want:         "postgres-fallback",
		},
		{
			name:         "unknown source",
			source:       "something-random",
			fallbackUsed: false,
			cacheHit:     false,
			want:         "source-unknown",
		},
		{
			name:         "cache hit overrides fallback",
			source:       "event-retrieval",
			fallbackUsed: true,
			cacheHit:     true,
			want:         "cache-hit",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyEventQuerySourcePosture(tt.source, tt.fallbackUsed, tt.cacheHit)
			if got != tt.want {
				t.Errorf("classifyEventQuerySourcePosture() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyDomainListQuerySourcePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result *domainquery.Result
		want   string
	}{
		{
			name:   "nil result",
			result: nil,
			want:   "domain-service",
		},
		{
			name: "mongodb source no cache",
			result: &domainquery.Result{
				Source:   "mongodb",
				CacheHit: false,
			},
			want: "mongodb-live",
		},
		{
			name: "cache hit result",
			result: &domainquery.Result{
				Source:   "mongodb",
				CacheHit: true,
			},
			want: "cache-hit",
		},
		{
			name: "postgresql source",
			result: &domainquery.Result{
				Source:   "postgresql",
				CacheHit: false,
			},
			want: "postgres-fallback",
		},
		{
			name: "unknown source",
			result: &domainquery.Result{
				Source:   "some-custom-source",
				CacheHit: false,
			},
			want: "source-unknown",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyDomainListQuerySourcePosture(tt.result)
			if got != tt.want {
				t.Errorf("classifyDomainListQuerySourcePosture() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildEventQueryReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		sourcePosture      string
		consistencyPosture string
		want               string
	}{
		{
			name:               "cache-hit query-service-direct",
			sourcePosture:      "cache-hit",
			consistencyPosture: "query-service-direct",
			want:               "served from query-service cache; verify freshness expectations before treating as latest",
		},
		{
			name:               "mongodb-live query-service-direct",
			sourcePosture:      "mongodb-live",
			consistencyPosture: "query-service-direct",
			want:               "served directly from query-service live store path",
		},
		{
			name:               "domain-service domain-direct",
			sourcePosture:      "domain-service",
			consistencyPosture: "domain-direct",
			want:               "served directly from domain query path without fallback",
		},
		{
			name:               "retrieval-fallback with any consistency",
			sourcePosture:      "retrieval-fallback",
			consistencyPosture: "anything",
			want:               "served through fallback path; verify query-service availability if this persists",
		},
		{
			name:               "fallback-served consistency",
			sourcePosture:      "retrieval-service",
			consistencyPosture: "fallback-served",
			want:               "served through fallback path; verify query-service availability if this persists",
		},
		{
			name:               "retrieval-partial",
			sourcePosture:      "retrieval-service",
			consistencyPosture: "retrieval-partial",
			want:               "served with partial metadata coverage; verify metadata completeness before relying on full event context",
		},
		{
			name:               "retrieval-metadata-missing",
			sourcePosture:      "retrieval-service",
			consistencyPosture: "retrieval-metadata-missing",
			want:               "served without attached metadata; verify metadata pipeline before relying on enriched fields",
		},
		{
			name:               "empty-result",
			sourcePosture:      "retrieval-service",
			consistencyPosture: "empty-result",
			want:               "query returned no results; verify filters and upstream indexing freshness if unexpected",
		},
		{
			name:               "unknown combination",
			sourcePosture:      "source-unknown",
			consistencyPosture: "consistency-unknown",
			want:               "verify query source and metadata coverage before relying on this response",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildEventQueryReliabilityHint(tt.sourcePosture, tt.consistencyPosture)
			if got != tt.want {
				t.Errorf("buildEventQueryReliabilityHint(%q, %q) = %q, want %q", tt.sourcePosture, tt.consistencyPosture, got, tt.want)
			}
		})
	}
}

func TestEventQueryHandler_Health(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	t.Run("not initialized", func(t *testing.T) {
		t.Parallel()
		h := NewEventQueryHandler(nil, logger, metrics)
		status := h.Health(context.Background())
		if status.Status != "unhealthy" {
			t.Errorf("expected unhealthy, got %q", status.Status)
		}
	})

	t.Run("initialized nil retrieval service", func(t *testing.T) {
		t.Parallel()
		h := NewEventQueryHandler(nil, logger, metrics)
		h.initialized = true
		status := h.Health(context.Background())
		if status.Status != "unhealthy" {
			t.Errorf("expected unhealthy for nil service, got %q", status.Status)
		}
	})
}

func TestEventQueryHandler_Close(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	t.Run("not initialized", func(t *testing.T) {
		t.Parallel()
		h := NewEventQueryHandler(nil, logger, metrics)
		if err := h.Close(context.Background()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("initialized", func(t *testing.T) {
		t.Parallel()
		h := NewEventQueryHandler(nil, logger, metrics)
		h.initialized = true
		if err := h.Close(context.Background()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if h.initialized {
			t.Error("expected handler to be marked as not initialized")
		}
	})
}

func TestEventQueryHandler_respondError(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	h := NewEventQueryHandler(nil, logger, metrics)

	w := httptest.NewRecorder()
	h.respondError(w, http.StatusBadRequest, "INVALID_PARAM", "correlationId is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "INVALID_PARAM" {
		t.Errorf("expected error INVALID_PARAM, got %v", body["error"])
	}
}

func TestEventQueryHandler_HandleGetCorrelatedEvents_EmptyID(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	h := NewEventQueryHandler(nil, logger, metrics)

	req := httptest.NewRequest(http.MethodGet, "/events/correlated/", nil)
	w := httptest.NewRecorder()

	h.HandleGetCorrelatedEvents(w, req, "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestEventQueryHandler_HandleGetCorrelatedEvents_NoService(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	h := NewEventQueryHandler(nil, logger, metrics)

	req := httptest.NewRequest(http.MethodGet, "/events/correlated/abc123", nil)
	w := httptest.NewRecorder()

	h.HandleGetCorrelatedEvents(w, req, "abc123")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestBuildDomainListQueryMeta_nilResult(t *testing.T) {
	t.Parallel()
	meta := buildDomainListQueryMeta(nil)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
}

func TestBuildDomainListQueryMeta_withResult(t *testing.T) {
	t.Parallel()
	result := &domainquery.Result{
		Events: []core.BlockchainEvent{
			{ID: "evt-1", ChainID: "1"},
		},
		Total:  1,
		Source: "mongo",
	}
	meta := buildDomainListQueryMeta(result)
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta.ResultCount != 1 {
		t.Errorf("expected ResultCount=1, got %d", meta.ResultCount)
	}
}

func TestBuildDomainQueryListMeta_nil(t *testing.T) {
	t.Parallel()
	meta := buildDomainQueryListMeta(nil, "test-path")
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
}
