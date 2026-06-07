package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/graphql-go/graphql"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

func makeTestEvent(id string, blockNum uint64, eventName string) *blockchain.BlockchainEvent {
	return &blockchain.BlockchainEvent{
		ID:               id,
		EventHash:        "0xhash" + id,
		EventSignature:   common.HexToHash("0xabc"),
		BlockNumber:      blockNum,
		BlockHash:        common.HexToHash("0xblock"),
		BlockTimestamp:   1234567890,
		TransactionHash:  common.HexToHash("0xtx"),
		TransactionIndex: 5,
		GasUsed:          21000,
		LogIndex:         3,
		ContractAddress:  common.HexToAddress("0xcontract"),
		EventName:        eventName,
		EventData:        []byte("data"),
		DecodedData:      map[string]any{"key": "value"},
		ChainID:          "1",
		Status:           blockchain.EventStatusConfirmed,
	}
}

func TestEventToMap(t *testing.T) {
	t.Parallel()

	t.Run("nil event", func(t *testing.T) {
		t.Parallel()
		result := eventToMap(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("pointer BlockchainEvent", func(t *testing.T) {
		t.Parallel()
		evt := makeTestEvent("evt-1", 100, "Transfer")
		result := eventToMap(evt)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["id"] != "evt-1" {
			t.Errorf("expected id 'evt-1', got %v", result["id"])
		}
		if result["blockNumber"] != uint64(100) {
			t.Errorf("expected blockNumber 100, got %v", result["blockNumber"])
		}
		if result["eventName"] != "Transfer" {
			t.Errorf("expected eventName 'Transfer', got %v", result["eventName"])
		}
		if result["chainId"] != "1" {
			t.Errorf("expected chainId '1', got %v", result["chainId"])
		}
		if result["status"] != string(blockchain.EventStatusConfirmed) {
			t.Errorf("expected status 'confirmed', got %v", result["status"])
		}
		if result["blockTimestamp"] != int64(1234567890) {
			t.Errorf("expected blockTimestamp 1234567890, got %v", result["blockTimestamp"])
		}
	})

	t.Run("value BlockchainEvent", func(t *testing.T) {
		t.Parallel()
		evt := *makeTestEvent("evt-2", 200, "Swap")
		result := eventToMap(evt)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["id"] != "evt-2" {
			t.Errorf("expected id 'evt-2', got %v", result["id"])
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		t.Parallel()
		result := eventToMap("not an event")
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["id"] != "unknown" {
			t.Errorf("expected id 'unknown', got %v", result["id"])
		}
	})
}

func TestEventDataMap(t *testing.T) {
	t.Parallel()

	t.Run("nil data", func(t *testing.T) {
		t.Parallel()
		result := eventDataMap(nil)
		if result == nil {
			t.Fatal("expected non-nil result for nil input")
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("non-nil data", func(t *testing.T) {
		t.Parallel()
		data := map[string]any{"from": "0xAlice", "to": "0xBob", "value": "100"}
		result := eventDataMap(data)
		if result["from"] != "0xAlice" {
			t.Errorf("expected '0xAlice', got %v", result["from"])
		}
		if result["value"] != "100" {
			t.Errorf("expected '100', got %v", result["value"])
		}
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()
		result := eventDataMap(map[string]any{})
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})
}

func TestNewGraphQLHandler(t *testing.T) {
	t.Parallel()

	qs := &mockDomainQueryService{}
	es := &mockEventStore{
		getEvent: func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
			return makeTestEvent(eventID, 42, "Transfer"), nil
		},
	}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewGraphQLHandler(qs, es, logger, metrics)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.queryService == nil {
		t.Error("expected queryService to be set")
	}
	if handler.eventStore == nil {
		t.Error("expected eventStore to be set")
	}
	if handler.schema == nil {
		t.Error("expected schema to be built")
	}
}

func TestNewGraphQLHandler_NilEventStore(t *testing.T) {
	t.Parallel()

	qs := &mockDomainQueryService{}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewGraphQLHandler(qs, nil, logger, metrics)
	if handler == nil {
		t.Fatal("expected non-nil handler even with nil event store")
	}
}

func TestGraphQLHandler_Initialize(t *testing.T) {
	t.Parallel()

	qs := &mockDomainQueryService{}
	es := &mockEventStore{}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewGraphQLHandler(qs, es, logger, metrics)

	err := handler.Initialize(&core.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handler.initialized {
		t.Error("expected handler to be initialized")
	}
}

func TestGraphQLHandler_InitializeAlreadyInitialized(t *testing.T) {
	t.Parallel()

	qs := &mockDomainQueryService{}
	es := &mockEventStore{}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewGraphQLHandler(qs, es, logger, metrics)

	_ = handler.Initialize(&core.Config{})
	err := handler.Initialize(&core.Config{})
	if err == nil {
		t.Error("expected error when initializing already initialized handler")
	}
}

func TestGraphQLHandler_Stop(t *testing.T) {
	t.Parallel()

	qs := &mockDomainQueryService{}
	es := &mockEventStore{}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewGraphQLHandler(qs, es, logger, metrics)
	_ = handler.Initialize(&core.Config{})

	err := handler.Stop()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler.initialized {
		t.Error("expected handler to be stopped")
	}
}

func TestGraphQLHandler_HandleNotInitialized(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  false,
	}

	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestGraphQLHandler_HandleOptions(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
	}

	req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGraphQLHandler_HandleMethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
	}

	req := httptest.NewRequest(http.MethodPut, "/graphql", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestGraphQLHandler_HandleGet(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
	}

	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestGraphQLHandler_HandlePostNoQuery(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGraphQLHandler_HandlePostInvalidJSON(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGraphQLHandler_HandlePostFormEncoded(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
		schema:       nil,
	}

	body := "query={events{edges{node{id}}}}"
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 (no schema), got %d", w.Code)
	}
}

func TestGraphQLHandler_HandlePostSchemaNotInitialized(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
		schema:       nil,
	}

	body := `{"query": "{ events { edges { node { id } } } }"}`
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestGraphQLHandler_HandlePostQueryTooDeep(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
		initialized:  true,
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("failed to build schema: %v", err)
	}
	handler.schema = schema

	deepQuery := "{ events { " + strings.Repeat("{ ", maxQueryDepth+5) + strings.Repeat("}", maxQueryDepth+5) + " } }"
	body, _ := json.Marshal(map[string]string{"query": deepQuery})
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGraphQLHandler_writeError(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		logger: &MockLogger{},
	}

	w := httptest.NewRecorder()
	handler.writeError(w, http.StatusTeapot, "test error message")

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content type, got %q", contentType)
	}

	var resp GraphQLResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(resp.Errors))
	}
	if resp.Errors[0].Message != "test error message" {
		t.Errorf("expected 'test error message', got %q", resp.Errors[0].Message)
	}
}

func TestBuildSchema(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore: &mockEventStore{
			getEvent: func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
				return makeTestEvent(eventID, 42, "Transfer"), nil
			},
		},
		logger:  &MockLogger{},
		metrics: NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
}

func TestBuildSchema_QueryEvent(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore: &mockEventStore{
			getEvent: func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
				return makeTestEvent(eventID, 42, "Transfer"), nil
			},
		},
		logger:  &MockLogger{},
		metrics: NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	result := graphqlDo(schema, `{ event(id: "evt-1") { id eventName blockNumber } }`, nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
}

func TestBuildSchema_QueryEventMissingID(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   &mockEventStore{},
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	result := graphqlDo(schema, `{ event { id } }`, nil)
	if len(result.Errors) == 0 {
		t.Error("expected errors for missing id argument")
	}
}

func TestBuildSchema_QueryEventNoEventStore(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   nil,
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	result := graphqlDo(schema, `{ event(id: "evt-1") { id } }`, nil)
	if len(result.Errors) == 0 {
		t.Error("expected errors when event store is not configured")
	}
}

func TestBuildSchema_QueryEvents(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore: &mockEventStore{
			getEvent: func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
				return makeTestEvent(eventID, 42, "Transfer"), nil
			},
			getEventsByChain: func(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
				return []*blockchain.BlockchainEvent{
					makeTestEvent("evt-1", 100, "Transfer"),
					makeTestEvent("evt-2", 101, "Swap"),
				}, nil
			},
		},
		logger:  &MockLogger{},
		metrics: NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	result := graphqlDo(schema, `{ events(first: 10) { edges { node { id eventName } } total } }`, nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
}

func TestBuildSchema_QueryEventsNoEventStore(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore:   nil,
		logger:       &MockLogger{},
		metrics:      NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	result := graphqlDo(schema, `{ events { edges { node { id } } } }`, nil)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors for nil event store: %v", result.Errors)
	}
}

func TestBuildSchema_QueryBlockUnsupported(t *testing.T) {
	t.Parallel()

	handler := &GraphQLHandler{
		queryService: &mockDomainQueryService{},
		eventStore: &mockEventStore{
			getEvent: func(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
				return makeTestEvent(eventID, 42, "Transfer"), nil
			},
		},
		logger:  &MockLogger{},
		metrics: NewMockMetricsCollector(),
	}

	schema, err := handler.buildSchema()
	if err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	result := graphqlDo(schema, `{ block(number: 100) { number } }`, nil)
	if len(result.Errors) == 0 {
		t.Error("expected error for unsupported block query")
	}
}

func graphqlDo(schema *graphql.Schema, query string, variables map[string]any) *graphql.Result {
	params := graphql.Params{
		Schema:         *schema,
		RequestString:  query,
		VariableValues: variables,
		Context:        context.Background(),
	}
	return graphql.Do(params)
}
