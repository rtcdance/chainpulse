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

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

type mockDomainQueryService struct {
	query       func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error)
	queryByHash func(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error)
}

func (m *mockDomainQueryService) Query(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
	if m.query != nil {
		return m.query(ctx, req)
	}
	return nil, nil
}

func (m *mockDomainQueryService) QueryByHash(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
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
	getEvent         func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error)
	getEventsByChain func(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error)
}

func (m *mockEventStore) Initialize(ctx context.Context) error { return nil }
func (m *mockEventStore) InsertEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	return nil
}

func (m *mockEventStore) InsertEventBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error {
	return nil
}

func (m *mockEventStore) GetEvent(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
	if m.getEvent != nil {
		return m.getEvent(ctx, eventID)
	}

	return nil, nil
}

func (m *mockEventStore) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	if m.getEventsByChain != nil {
		return m.getEventsByChain(ctx, chainID, limit, offset)
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStore) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*blockchain.BlockchainEvent, bool, error) {
	return nil, false, nil
}
func (m *mockEventStore) CountEvents(ctx context.Context) (int64, error)         { return 0, nil }
func (m *mockEventStore) DeleteExpiredEvents(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockEventStore) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Message: "ok"}
}
func (m *mockEventStore) Close(ctx context.Context) error { return nil }
func (m *mockEventStore) GetEventStats(ctx context.Context) (map[string]int64, map[string]int64, int64, error) {
	return nil, nil, 0, nil
}
func (m *mockEventStore) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
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
		queryByHash: func(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
			return &blockchain.BlockchainEvent{
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
		getEvent: func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
			return &blockchain.BlockchainEvent{
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
		queryByHash: func(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
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
		Event: &blockchain.BlockchainEvent{
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
		Event: &blockchain.BlockchainEvent{
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
		getEventsByChain: func(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
			return []*blockchain.BlockchainEvent{
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
				Events: []blockchain.BlockchainEvent{
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
				Events: []blockchain.BlockchainEvent{
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
				Events: []blockchain.BlockchainEvent{
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
				Events: []blockchain.BlockchainEvent{
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
				Events: []blockchain.BlockchainEvent{
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
