package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/graphql-go/graphql"
	"github.com/rtcdance/chainpulse/pkg/core"
	apicore "github.com/rtcdance/chainpulse/pkg/plugins/api/core"
)

// MockEventStore implements core.EventStore for testing
type MockEventStore struct {
	events             map[string]*core.BlockchainEvent
	getEventsPaginated func(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error)
	getEventsByName    func(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error)
	getEventsByAddress func(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error)
	getEventsByBlock   func(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error)
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events: make(map[string]*core.BlockchainEvent),
	}
}

func (m *MockEventStore) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	return m.events[eventID], nil
}

func (m *MockEventStore) StoreEvent(ctx context.Context, event *core.BlockchainEvent) error {
	m.events[event.ID] = event
	return nil
}

func (m *MockEventStore) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockEventStore) Close(ctx context.Context) error {
	return nil
}

func (m *MockEventStore) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockEventStore) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	if m.getEventsByBlock != nil {
		return m.getEventsByBlock(ctx, blockNumber)
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	if m.getEventsByAddress != nil {
		return m.getEventsByAddress(ctx, address, limit)
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	if m.getEventsByName != nil {
		return m.getEventsByName(ctx, eventName, limit)
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStore) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	if m.getEventsPaginated != nil {
		return m.getEventsPaginated(ctx, cursor, limit)
	}
	return []*core.BlockchainEvent{}, false, nil
}

func (m *MockEventStore) CountEvents(ctx context.Context) (int64, error) {
	return int64(len(m.events)), nil
}

func (m *MockEventStore) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{
		Status: "healthy",
	}
}

func (m *MockEventStore) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	m.events[event.ID] = event
	return nil
}

func (m *MockEventStore) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	for _, event := range events {
		m.events[event.ID] = event
	}
	return nil
}

func (m *MockEventStore) GetEventStats(ctx context.Context) (map[string]int64, map[string]int64, int64, error) {
	return make(map[string]int64), make(map[string]int64), 0, nil
}

// MockLogger implements core.Logger for testing
type MockLogger struct {
	messages []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		messages: make([]string, 0),
	}
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Fatal(msg string, args ...any) {
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) WithCorrelationID(correlationID string) core.Logger {
	return m
}

// MockMetrics implements core.MetricsCollector for testing
type MockMetrics struct {
	metrics map[string]float64
}

func NewMockMetrics() *MockMetrics {
	return &MockMetrics{
		metrics: make(map[string]float64),
	}
}

func (m *MockMetrics) RecordMetric(name string, value float64) {
	m.metrics[name] += value
}

func (m *MockMetrics) RecordCounter(name string, value int64, tags map[string]string) {
	m.metrics[name] += float64(value)
}

func (m *MockMetrics) RecordGauge(name string, value float64, tags map[string]string) {
	m.metrics[name] = value
}

func (m *MockMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	m.metrics[name] += value
}

func (m *MockMetrics) GetMetrics() map[string]any {
	result := make(map[string]any)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

// MockCache implements core.Cache for testing
type MockCache struct {
	data map[string]any
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]any),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, nil
	}
	if b, ok := val.([]byte); ok {
		return b, nil
	}
	return nil, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl int) error {
	m.data[key] = value
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCache) GetStats() core.CacheStats {
	return core.CacheStats{
		HitCount:      0,
		MissCount:     0,
		EvictionCount: 0,
		HitRate:       0,
	}
}

func (m *MockCache) Health(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *MockCache) Initialize(ctx context.Context, config core.Config) error {
	_ = ctx
	_ = config
	return nil
}

func (m *MockCache) Start(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *MockCache) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *MockCache) HealthCheck(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *MockCache) Name() string {
	return "mock-cache"
}

func (m *MockCache) Version() string {
	return "1.0.0"
}

// Test Schema Builder
func TestSchemaBuilder_BuildSchema(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	authMiddleware := NewAuthMiddleware(logger, metrics)

	builder := NewSchemaBuilder(eventStore, logger, metrics, nil, authMiddleware)
	schema, err := builder.BuildSchema()
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	if schema.QueryType() == nil {
		t.Fatal("Query type is nil")
	}

	if schema.MutationType() == nil {
		t.Fatal("Mutation type is nil")
	}
}

func TestSchemaBuilderResolveEventIncludesCacheQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	payload, err := json.Marshal(map[string]any{
		"id":                 "evt-schema-cache",
		"querySourcePosture": "graphql-event-store",
	})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	cache.data["graphql:event:evt-schema-cache"] = payload

	builder := NewSchemaBuilder(NewMockEventStore(), logger, metrics, cache, NewAuthMiddleware(logger, metrics))

	result, err := builder.resolveEvent(mockResolveParams(map[string]any{
		"id": "evt-schema-cache",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	item, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %#v", result)
	}
	if got := item["querySourcePosture"]; got != "graphql-cache-hit" {
		t.Fatalf("expected querySourcePosture graphql-cache-hit, got %v", got)
	}
}

func TestSchemaBuilderResolveEventsByNameIncludesLiveQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByName = func(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-name-schema",
				EventName:   eventName,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, nil
	}

	builder := NewSchemaBuilder(eventStore, logger, metrics, nil, NewAuthMiddleware(logger, metrics))

	result, err := builder.resolveEventsByName(mockResolveParams(map[string]any{
		"eventName": "Transfer",
		"limit":     1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result item, got %#v", result)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result item, got %#v", items[0])
	}
	if got := first["querySourcePosture"]; got != "graphql-event-store" {
		t.Fatalf("expected querySourcePosture graphql-event-store, got %v", got)
	}
}

func TestSubscriptionManager_GetAllSubscriptions(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	manager := NewSubscriptionManager(logger, metrics, nil)

	if _, err := manager.Subscribe("topic1"); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if _, err := manager.Subscribe("topic1"); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if _, err := manager.Subscribe("topic2"); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	all := manager.GetAllSubscriptions()
	if all["topic1"] != 2 {
		t.Errorf("expected 2 subscribers for topic1, got %d", all["topic1"])
	}
	if all["topic2"] != 1 {
		t.Errorf("expected 1 subscriber for topic2, got %d", all["topic2"])
	}
}

func TestSubscriptionManager_GetAllSubscriptions_Empty(t *testing.T) {
	t.Parallel()
	manager := NewSubscriptionManager(NewMockLogger(), NewMockMetrics(), nil)

	all := manager.GetAllSubscriptions()
	if len(all) != 0 {
		t.Errorf("expected 0 topics, got %d", len(all))
	}
}

func TestSubscriptionHandler_OnEventUpdated(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("event:updated")

	handler := NewSubscriptionHandler(manager, logger, metrics)

	err := handler.OnEventUpdated(&core.BlockchainEvent{
		ID:     "evt-updated",
		Status: core.EventStatusConfirmed,
	})
	if err != nil {
		t.Fatalf("OnEventUpdated failed: %v", err)
	}

	select {
	case msg := <-subscription.Channel:
		payload, ok := msg.(EventSubscriptionPayload)
		if !ok {
			t.Fatalf("expected EventSubscriptionPayload, got %T", msg)
		}
		if payload.Type != "updated" {
			t.Errorf("expected type 'updated', got '%s'", payload.Type)
		}
		if payload.EventID != "evt-updated" {
			t.Errorf("expected eventID 'evt-updated', got '%s'", payload.EventID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

func TestSubscriptionHandler_OnEventDeleted(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("event:deleted")

	handler := NewSubscriptionHandler(manager, logger, metrics)

	err := handler.OnEventDeleted("evt-deleted")
	if err != nil {
		t.Fatalf("OnEventDeleted failed: %v", err)
	}

	select {
	case msg := <-subscription.Channel:
		payload, ok := msg.(EventSubscriptionPayload)
		if !ok {
			t.Fatalf("expected EventSubscriptionPayload, got %T", msg)
		}
		if payload.Type != "deleted" {
			t.Errorf("expected type 'deleted', got '%s'", payload.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

func TestSubscriptionHandler_OnEventConfirmed(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("event:confirmed")

	handler := NewSubscriptionHandler(manager, logger, metrics)

	err := handler.OnEventConfirmed(&core.BlockchainEvent{
		ID:     "evt-confirmed",
		Status: core.EventStatusConfirmed,
	})
	if err != nil {
		t.Fatalf("OnEventConfirmed failed: %v", err)
	}

	select {
	case msg := <-subscription.Channel:
		payload, ok := msg.(EventSubscriptionPayload)
		if !ok {
			t.Fatalf("expected EventSubscriptionPayload, got %T", msg)
		}
		if payload.Type != "confirmed" {
			t.Errorf("expected type 'confirmed', got '%s'", payload.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

func TestSubscriptionHandler_OnEventFailed(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("event:failed")

	handler := NewSubscriptionHandler(manager, logger, metrics)

	err := handler.OnEventFailed(&core.BlockchainEvent{
		ID:     "evt-failed",
		Status: core.EventStatusFailed,
	})
	if err != nil {
		t.Fatalf("OnEventFailed failed: %v", err)
	}

	select {
	case msg := <-subscription.Channel:
		payload, ok := msg.(EventSubscriptionPayload)
		if !ok {
			t.Fatalf("expected EventSubscriptionPayload, got %T", msg)
		}
		if payload.Type != "failed" {
			t.Errorf("expected type 'failed', got '%s'", payload.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

func TestSubscriptionHandler_OnCacheInvalidated(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("cache:invalidated")

	handler := NewSubscriptionHandler(manager, logger, metrics)

	err := handler.OnCacheInvalidated("event:test-id")
	if err != nil {
		t.Fatalf("OnCacheInvalidated failed: %v", err)
	}

	select {
	case msg := <-subscription.Channel:
		payload, ok := msg.(map[string]any)
		if !ok {
			t.Fatalf("expected map payload, got %T", msg)
		}
		if payload["type"] != "invalidated" {
			t.Errorf("expected type 'invalidated', got '%v'", payload["type"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

func TestLoggingMiddleware_LogMutation(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	middleware := NewLoggingMiddleware(logger)

	middleware.LogMutation("user-1", "mutation { invalidateCache(eventId: \"1\") }", 50*time.Millisecond)

	if len(logger.messages) == 0 {
		t.Fatal("Expected log message for mutation")
	}
}

func TestLoggingMiddleware_LogError(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	middleware := NewLoggingMiddleware(logger)

	middleware.LogError("user-1", "query", fmt.Errorf("test error"))

	if len(logger.messages) == 0 {
		t.Fatal("Expected log message for error")
	}
}

func TestSchemaBuilderResolveEventsIncludesCacheQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	payload, err := json.Marshal(map[string]any{
		"edges": []any{
			map[string]any{
				"cursor": "cursor_0",
				"node": map[string]any{
					"id":                 "evt-schema-root-cache",
					"querySourcePosture": "graphql-event-store",
				},
			},
		},
		"pageInfo": map[string]any{
			"hasNextPage": false,
			"totalCount":  1,
		},
	})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	cache.data["graphql:events:root:after::first:1"] = payload

	builder := NewSchemaBuilder(NewMockEventStore(), logger, metrics, cache, NewAuthMiddleware(logger, metrics))

	result, err := builder.resolveEvents(mockResolveParams(map[string]any{
		"first": 1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	connection, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected connection result, got %#v", result)
	}
	edges, ok := connection["edges"].([]any)
	if !ok || len(edges) != 1 {
		t.Fatalf("expected one edge, got %#v", connection["edges"])
	}
	edge, ok := edges[0].(map[string]any)
	if !ok {
		t.Fatalf("expected edge map, got %#v", edges[0])
	}
	node, ok := edge["node"].(map[string]any)
	if !ok {
		t.Fatalf("expected node map, got %#v", edge["node"])
	}
	if got := node["querySourcePosture"]; got != "graphql-cache-hit" {
		t.Fatalf("expected querySourcePosture graphql-cache-hit, got %v", got)
	}
}

// Test Event Resolver
func TestEventResolver_ResolveEvent(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	cache := NewMockCache()

	ctx := &ResolverContext{
		EventStore: eventStore,
		Logger:     logger,
		Metrics:    metrics,
		Cache:      cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	}

	resolver := NewEventResolver(ctx)

	// Test with missing event
	result, err := resolver.ResolveEvent(mockResolveParams(map[string]any{
		"id": "nonexistent",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != nil {
		t.Fatal("Expected nil result for nonexistent event")
	}
}

func TestEventResolverResolveEventIncludesLiveQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.events["evt-live"] = &core.BlockchainEvent{
		ID:          "evt-live",
		Status:      core.EventStatusConfirmed,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		IndexedAt:   time.Now(),
	}

	resolver := NewEventResolver(&ResolverContext{
		EventStore: eventStore,
		Logger:     logger,
		Metrics:    metrics,
		Cache:      NewMockCache(),
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEvent(mockResolveParams(map[string]any{
		"id": "evt-live",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	item, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %#v", result)
	}
	if got := item["querySourcePosture"]; got != "graphql-event-store" {
		t.Fatalf("expected querySourcePosture graphql-event-store, got %v", got)
	}
}

func TestEventResolverResolveEventIncludesCacheQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	payload, err := json.Marshal(map[string]any{
		"id":                 "evt-cache",
		"querySourcePosture": "graphql-event-store",
	})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	cache.data["graphql:event:evt-cache"] = payload

	resolver := NewEventResolver(&ResolverContext{
		EventStore: NewMockEventStore(),
		Logger:     logger,
		Metrics:    metrics,
		Cache:      cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEvent(mockResolveParams(map[string]any{
		"id": "evt-cache",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	item, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %#v", result)
	}
	if got := item["querySourcePosture"]; got != "graphql-cache-hit" {
		t.Fatalf("expected querySourcePosture graphql-cache-hit, got %v", got)
	}
}

func TestEventResolverResolveEventsIncludesLiveQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsPaginated = func(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-root-live",
				BlockNumber: 99,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, false, nil
	}

	resolver := NewEventResolver(&ResolverContext{
		EventStore: eventStore,
		Logger:     logger,
		Metrics:    metrics,
		Cache:      NewMockCache(),
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEvents(mockResolveParams(map[string]any{
		"first": 1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	connection, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected connection result, got %#v", result)
	}
	edges, ok := connection["edges"].([]any)
	if !ok || len(edges) != 1 {
		t.Fatalf("expected one edge, got %#v", connection["edges"])
	}
	edge, ok := edges[0].(map[string]any)
	if !ok {
		t.Fatalf("expected edge map, got %#v", edges[0])
	}
	node, ok := edge["node"].(map[string]any)
	if !ok {
		t.Fatalf("expected node map, got %#v", edge["node"])
	}
	if got := node["querySourcePosture"]; got != "graphql-event-store" {
		t.Fatalf("expected querySourcePosture graphql-event-store, got %v", got)
	}
}

func TestEventResolverResolveEventsIncludesCacheQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	payload, err := json.Marshal(map[string]any{
		"edges": []any{
			map[string]any{
				"cursor": "cursor_0",
				"node": map[string]any{
					"id":                 "evt-root-cache",
					"querySourcePosture": "graphql-event-store",
				},
			},
		},
		"pageInfo": map[string]any{
			"hasNextPage": false,
			"totalCount":  1,
		},
	})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	cache.data["graphql:events:root:after::first:1"] = payload

	resolver := NewEventResolver(&ResolverContext{
		EventStore: NewMockEventStore(),
		Logger:     logger,
		Metrics:    metrics,
		Cache:      cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEvents(mockResolveParams(map[string]any{
		"first": 1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	connection, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected connection result, got %#v", result)
	}
	edges, ok := connection["edges"].([]any)
	if !ok || len(edges) != 1 {
		t.Fatalf("expected one edge, got %#v", connection["edges"])
	}
	edge, ok := edges[0].(map[string]any)
	if !ok {
		t.Fatalf("expected edge map, got %#v", edges[0])
	}
	node, ok := edge["node"].(map[string]any)
	if !ok {
		t.Fatalf("expected node map, got %#v", edge["node"])
	}
	if got := node["querySourcePosture"]; got != "graphql-cache-hit" {
		t.Fatalf("expected querySourcePosture graphql-cache-hit, got %v", got)
	}
}

// Test Cache Resolver
func TestCacheResolver_ResolveInvalidateCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()

	// Set a cache entry
	data, _ := json.Marshal(map[string]any{"id": "test-id"})
	if err := cache.Set(context.Background(), "graphql:event:test-id", data, 300); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	ctx := &ResolverContext{
		Logger:  logger,
		Metrics: metrics,
		Cache:   cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"manage:cache"},
		},
	}

	resolver := NewCacheResolver(ctx)

	result, err := resolver.ResolveInvalidateCache(mockResolveParams(map[string]any{
		"eventId": "test-id",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != true {
		t.Fatal("Expected true result")
	}

	// Verify cache entry was deleted
	if _, ok := cache.data["graphql:event:test-id"]; ok {
		t.Fatal("Cache entry should be deleted")
	}
}

// Test Subscription Manager
func TestSubscriptionManager_Subscribe(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, err := manager.Subscribe("test:topic")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if subscription == nil {
		t.Fatal("Subscription is nil")
	}

	if subscription.Topic != "test:topic" {
		t.Fatalf("Expected topic 'test:topic', got '%s'", subscription.Topic)
	}

	count := manager.GetSubscriberCount("test:topic")
	if count != 1 {
		t.Fatalf("Expected 1 subscriber, got %d", count)
	}
}

// Test Subscription Manager Publish
func TestSubscriptionManager_Publish(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("test:topic")

	// Publish message
	err := manager.Publish("test:topic", map[string]any{"message": "test"})
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Receive message
	select {
	case msg := <-subscription.Channel:
		if msg == nil {
			t.Fatal("Received nil message")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// Test Subscription Manager Unsubscribe
func TestSubscriptionManager_Unsubscribe(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	manager := NewSubscriptionManager(logger, metrics, nil)

	subscription, _ := manager.Subscribe("test:topic")

	count := manager.GetSubscriberCount("test:topic")
	if count != 1 {
		t.Fatalf("Expected 1 subscriber, got %d", count)
	}

	err := manager.Unsubscribe(subscription)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	count = manager.GetSubscriberCount("test:topic")
	if count != 0 {
		t.Fatalf("Expected 0 subscribers, got %d", count)
	}
}

// Test Auth Middleware
func TestAuthMiddleware_Authenticate(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	middleware := NewAuthMiddleware(logger, metrics)
	// Set requireAuth=false so Authenticate works without a configured TokenValidator
	// (the old behavior was to allow any token; the new behavior rejects when no validator is set)
	middleware.SetRequireAuth(false)

	authCtx, err := middleware.Authenticate(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	if authCtx == nil {
		t.Fatal("Auth context is nil")
	}

	if !authCtx.CanReadEvent("test-id") {
		t.Fatal("User should be able to read events")
	}
}

// Test Complexity Middleware
func TestComplexityMiddleware_AnalyzeQuery(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	middleware := NewComplexityMiddleware(100, logger, metrics)

	complexity, err := middleware.AnalyzeQuery("{ event(id: \"1\") { id name } }")
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	if complexity <= 0 {
		t.Fatal("Complexity should be greater than 0")
	}
}

// Test Complexity Middleware Exceeded
func TestComplexityMiddleware_ExceededLimit(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	middleware := NewComplexityMiddleware(5, logger, metrics)

	_, err := middleware.AnalyzeQuery("{ event(id: \"1\") { id name block { number hash } } }")
	if err == nil {
		t.Fatal("Expected error for high complexity query")
	}
}

// Test Rate Limit Middleware
func TestRateLimitMiddleware_CheckLimit(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	middleware := NewRateLimitMiddleware(2, logger, metrics)

	// First request should pass
	err := middleware.CheckLimit("user-1")
	if err != nil {
		t.Fatalf("First request should pass: %v", err)
	}

	// Second request should pass
	err = middleware.CheckLimit("user-1")
	if err != nil {
		t.Fatalf("Second request should pass: %v", err)
	}

	// Third request should fail
	err = middleware.CheckLimit("user-1")
	if err == nil {
		t.Fatal("Third request should fail")
	}
}

// Test Caching Middleware
func TestCachingMiddleware_GetCached(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()

	middleware := NewCachingMiddleware(cache, 5*time.Minute, logger, metrics)

	query := "{ event(id: \"1\") { id } }"
	result := map[string]any{"id": "1"}

	// Set cache
	err := middleware.SetCached(query, result)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get cache
	cached, err := middleware.GetCached(query)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if cached == nil {
		t.Fatal("Cached result is nil")
	}
}

// Test Validation Middleware
func TestValidationMiddleware_ValidateQuery(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	middleware := NewValidationMiddleware(logger, metrics)

	// Valid query
	err := middleware.ValidateQuery("{ event(id: \"1\") { id } }")
	if err != nil {
		t.Fatalf("Valid query should pass: %v", err)
	}

	// Empty query
	err = middleware.ValidateQuery("")
	if err == nil {
		t.Fatal("Empty query should fail")
	}

	// Query too large (> 10000 characters)
	largeQuery := ""
	for i := 0; i < 10001; i++ {
		largeQuery += "a"
	}
	err = middleware.ValidateQuery(largeQuery)
	if err == nil {
		t.Fatal("Large query should fail")
	}
}

// Test Logging Middleware
func TestLoggingMiddleware_LogQuery(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	middleware := NewLoggingMiddleware(logger)

	middleware.LogQuery("user-1", "{ event(id: \"1\") { id } }", 100*time.Millisecond)

	if len(logger.messages) == 0 {
		t.Fatal("Expected log message")
	}
}

// Test Mutation Builder
func TestMutationBuilder_BuildMutations(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()

	builder := NewMutationBuilder(eventStore, logger, metrics, cache)
	mutations := builder.BuildMutations()

	if mutations == nil {
		t.Fatal("Mutations object is nil")
	}

	if mutations.Name() != "Mutation" {
		t.Fatalf("Expected mutation name 'Mutation', got '%s'", mutations.Name())
	}
}

// Test Subscription Handler
func TestSubscriptionHandler_OnEventCreated(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	manager := NewSubscriptionManager(logger, metrics, nil)

	handler := NewSubscriptionHandler(manager, logger, metrics)

	// Subscribe to event created
	subscription, _ := manager.Subscribe("event:created")

	// Create event
	event := &core.BlockchainEvent{
		ID:     "test-id",
		Status: core.EventStatusConfirmed,
	}

	err := handler.OnEventCreated(event)
	if err != nil {
		t.Fatalf("Failed to publish event created: %v", err)
	}

	// Receive notification
	select {
	case msg := <-subscription.Channel:
		if msg == nil {
			t.Fatal("Received nil message")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

// Helper function to create mock resolve params
func mockResolveParams(args map[string]any) graphql.ResolveParams {
	return graphql.ResolveParams{
		Args:    args,
		Context: context.Background(),
	}
}

// Test Query Complexity Analyzer
func TestQueryComplexityAnalyzer_AnalyzeComplexity(t *testing.T) {
	t.Parallel()
	analyzer := NewQueryComplexityAnalyzer(100)

	complexity, err := analyzer.AnalyzeComplexity("{ event(id: \"1\") { id } }")
	if err != nil {
		t.Fatalf("Failed to analyze complexity: %v", err)
	}

	if complexity <= 0 {
		t.Fatal("Complexity should be greater than 0")
	}
}

// Test Query Complexity Analyzer Exceeded
func TestQueryComplexityAnalyzer_ExceededLimit(t *testing.T) {
	t.Parallel()
	analyzer := NewQueryComplexityAnalyzer(5)

	// Create a query that will have complexity > 5
	// Complexity = len(query) / 10, so we need len > 50
	complexQuery := "{ event(id: \"1\") { id name block { number hash timestamp } transaction { hash from to value } } }"
	_, err := analyzer.AnalyzeComplexity(complexQuery)
	if err == nil {
		t.Fatal("Expected error for high complexity query")
	}
}

// Test Auth Context Expiration
func TestAuthContext_IsExpired(t *testing.T) {
	t.Parallel()
	// Expired context
	expiredCtx := &AuthContext{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	if !expiredCtx.IsExpired() {
		t.Fatal("Context should be expired")
	}

	// Valid context
	validCtx := &AuthContext{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if validCtx.IsExpired() {
		t.Fatal("Context should not be expired")
	}
}

// Test Subscription Stats
func TestSubscriptionManager_GetStats(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	manager := NewSubscriptionManager(logger, metrics, nil)

	// Create subscriptions
	if _, err := manager.Subscribe("topic1"); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if _, err := manager.Subscribe("topic1"); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	if _, err := manager.Subscribe("topic2"); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	stats := manager.GetStats()

	if stats.TotalSubscriptions != 3 {
		t.Fatalf("Expected 3 total subscriptions, got %d", stats.TotalSubscriptions)
	}

	if stats.ActiveTopics != 2 {
		t.Fatalf("Expected 2 active topics, got %d", stats.ActiveTopics)
	}
}

// Test Event To GraphQL Conversion
func TestEventToGraphQL(t *testing.T) {
	t.Parallel()
	event := &core.BlockchainEvent{
		ID:     "test-id",
		Status: core.EventStatusConfirmed,
	}

	result := eventToGraphQL(event)

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result["id"] != "test-id" {
		t.Fatalf("Expected id 'test-id', got '%v'", result["id"])
	}

	if result["status"] != "confirmed" {
		t.Fatalf("Expected status 'confirmed', got '%v'", result["status"])
	}
}

func TestEventResolverResolveEventsByNameIncludesLiveQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByName = func(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-live",
				EventName:   eventName,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, nil
	}

	resolver := NewEventResolver(&ResolverContext{
		EventStore: eventStore,
		Logger:     logger,
		Metrics:    metrics,
		Cache:      NewMockCache(),
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEventsByName(mockResolveParams(map[string]any{
		"eventName": "Transfer",
		"limit":     1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result item, got %#v", result)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result item, got %#v", items[0])
	}
	if got := first["querySourcePosture"]; got != "graphql-event-store" {
		t.Fatalf("expected querySourcePosture graphql-event-store, got %v", got)
	}
}

func TestEventResolverResolveEventsByNameIncludesCacheQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	payload, err := json.Marshal([]map[string]any{
		{
			"id":                 "evt-cache",
			"eventName":          "Transfer",
			"querySourcePosture": "graphql-event-store",
		},
	})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	cache.data["graphql:events:name:Transfer:limit:1"] = payload

	resolver := NewEventResolver(&ResolverContext{
		EventStore: NewMockEventStore(),
		Logger:     logger,
		Metrics:    metrics,
		Cache:      cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEventsByName(mockResolveParams(map[string]any{
		"eventName": "Transfer",
		"limit":     1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result item, got %#v", result)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result item, got %#v", items[0])
	}
	if got := first["querySourcePosture"]; got != "graphql-cache-hit" {
		t.Fatalf("expected querySourcePosture graphql-cache-hit, got %v", got)
	}
}

func TestEventResolverResolveEventsByAddressIncludesLiveQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByAddress = func(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:              "evt-address-live",
				ContractAddress: common.HexToAddress("0xabc"),
				Status:          core.EventStatusConfirmed,
				CreatedAt:       time.Now(),
				ProcessedAt:     time.Now(),
				IndexedAt:       time.Now(),
			},
		}, nil
	}

	resolver := NewEventResolver(&ResolverContext{
		EventStore: eventStore,
		Logger:     logger,
		Metrics:    metrics,
		Cache:      NewMockCache(),
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEventsByAddress(mockResolveParams(map[string]any{
		"address": "0xabc",
		"limit":   1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result item, got %#v", result)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result item, got %#v", items[0])
	}
	if got := first["querySourcePosture"]; got != "graphql-event-store" {
		t.Fatalf("expected querySourcePosture graphql-event-store, got %v", got)
	}
}

func TestEventResolverResolveEventsByAddressIncludesCacheQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	payload, err := json.Marshal([]map[string]any{
		{
			"id":                 "evt-address-cache",
			"contractAddress":    "0xabc",
			"querySourcePosture": "graphql-event-store",
		},
	})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	cache.data["graphql:events:address:0xabc:limit:1"] = payload

	resolver := NewEventResolver(&ResolverContext{
		EventStore: NewMockEventStore(),
		Logger:     logger,
		Metrics:    metrics,
		Cache:      cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEventsByAddress(mockResolveParams(map[string]any{
		"address": "0xabc",
		"limit":   1,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result item, got %#v", result)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result item, got %#v", items[0])
	}
	if got := first["querySourcePosture"]; got != "graphql-cache-hit" {
		t.Fatalf("expected querySourcePosture graphql-cache-hit, got %v", got)
	}
}

func TestRequireMutationAuth_NoAuthMiddleware(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	store := &MockEventStore{}

	sb := NewSchemaBuilder(store, logger, metrics, nil, nil)

	// No auth middleware configured — development mode, should allow
	params := graphql.ResolveParams{Context: context.Background()}
	if err := sb.requireMutationAuth(params); err != nil {
		t.Errorf("expected nil error in dev mode (no auth middleware), got: %v", err)
	}
}

func TestRequireMutationAuth_MissingToken(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	store := &MockEventStore{}
	authMiddleware := NewAuthMiddleware(logger, metrics)

	sb := NewSchemaBuilder(store, logger, metrics, nil, authMiddleware)

	// No token in context
	params := graphql.ResolveParams{Context: context.Background()}
	err := sb.requireMutationAuth(params)
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if err.Error() != "authentication required for mutations" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRequireMutationAuth_InvalidToken(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	store := &MockEventStore{}
	authMiddleware := NewAuthMiddleware(logger, metrics)

	sb := NewSchemaBuilder(store, logger, metrics, nil, authMiddleware)

	// Token present but too short (Authenticate returns error for < 3 chars)
	ctx := context.WithValue(context.Background(), authTokenContextKey, "x")
	params := graphql.ResolveParams{Context: ctx}
	err := sb.requireMutationAuth(params)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if err.Error() != "authentication failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRequireMutationAuth_InsufficientScope(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	store := &MockEventStore{}
	authMiddleware := NewAuthMiddleware(logger, metrics)
	authMiddleware.SetRequireAuth(false)

	sb := NewSchemaBuilder(store, logger, metrics, nil, authMiddleware)

	// Use a valid-format token but without write scope
	ctx := context.WithValue(context.Background(), authTokenContextKey, "valid-test-token-12345")
	params := graphql.ResolveParams{Context: ctx}

	// The Authenticate method returns an AuthContext without "write:cache" or "admin"
	err := sb.requireMutationAuth(params)
	if err == nil {
		t.Fatal("expected error for insufficient scope")
	}
	if err.Error() != "insufficient permissions for mutation" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEventResolverResolveEventsByBlockIncludesLiveQuerySourcePosture(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByBlock = func(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-block-live",
				BlockNumber: 42,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, nil
	}

	resolver := NewEventResolver(&ResolverContext{
		EventStore: eventStore,
		Logger:     logger,
		Metrics:    metrics,
		Cache:      NewMockCache(),
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	})

	result, err := resolver.ResolveEventsByBlock(mockResolveParams(map[string]any{
		"blockNumber": 42,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result item, got %#v", result)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result item, got %#v", items[0])
	}
	if got := first["querySourcePosture"]; got != "graphql-event-store" {
		t.Fatalf("expected querySourcePosture graphql-event-store, got %v", got)
	}
}

// ========================
// GraphQLPlugin tests
// ========================

func TestGraphQLPlugin_GetName(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test-graphql", 8080, apiLayer)

	if got := p.GetName(); got != "test-graphql" {
		t.Errorf("expected 'test-graphql', got '%s'", got)
	}
}

func TestGraphQLPlugin_GetProtocolName(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test-protocol", 8080, apiLayer)

	if got := p.GetProtocolName(); got != "test-protocol" {
		t.Errorf("expected 'test-protocol', got '%s'", got)
	}
}

func TestGraphQLPlugin_SetSchemaBuilder(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	sb := &SchemaBuilder{}
	p.SetSchemaBuilder(sb)

	if p.schemaBuilder != sb {
		t.Fatal("schemaBuilder was not set")
	}
}

func TestGraphQLPlugin_SetSubscriptionManager(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	sm := NewSubscriptionManager(NewMockLogger(), NewMockMetrics(), nil)
	p.SetSubscriptionManager(sm)

	if p.subscriptionManager != sm {
		t.Fatal("subscriptionManager was not set")
	}
}

func TestGraphQLPlugin_WithAllowedOrigins(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	origins := []string{"http://localhost:3000", "https://example.com"}
	result := p.WithAllowedOrigins(origins)

	if result != p {
		t.Fatal("WithAllowedOrigins should return self")
	}
	if len(p.allowedOrigins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(p.allowedOrigins))
	}
}

func TestGraphQLPlugin_CheckOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		expected       bool
	}{
		{"empty allowed origins allows all", nil, "http://localhost:3000", true},
		{"matching origin", []string{"http://localhost:3000"}, "http://localhost:3000", true},
		{"case insensitive match", []string{"https://EXAMPLE.com"}, "https://example.com", true},
		{"non-matching origin", []string{"https://example.com"}, "http://evil.com", false},
		{"empty origin header allowed", []string{"https://example.com"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			apiLayer := apicore.NewAPILayer()
			p := NewGraphQLPlugin("test", 8080, apiLayer)
			if tt.allowedOrigins != nil {
				p.WithAllowedOrigins(tt.allowedOrigins)
			}

			req, _ := http.NewRequest("GET", "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			if got := p.checkOrigin(req); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestGraphQLPlugin_RegisterResolver(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	resolver := func(p graphql.ResolveParams) (any, error) {
		return "resolved", nil
	}
	p.RegisterResolver("testResolver", resolver)

	if _, ok := p.resolvers["testResolver"]; !ok {
		t.Fatal("resolver was not registered")
	}
}

func TestGraphQLPlugin_RegisterRoute(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	handler := apicore.HandlerFunc(func(req apicore.Request) (apicore.Response, error) {
		return apicore.NewBaseResponse(nil), nil
	})
	err := p.RegisterRoute("/test", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLPlugin_RegisterRoute_WhileRunning(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)
	p.running = true

	handler := apicore.HandlerFunc(func(req apicore.Request) (apicore.Response, error) {
		return apicore.NewBaseResponse(nil), nil
	})
	err := p.RegisterRoute("/test", handler)
	if err == nil {
		t.Fatal("expected error when registering route while running")
	}
}

func TestGraphQLPlugin_Use(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	mw := func(next apicore.Handler) apicore.Handler { return next }
	err := p.Use(mw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.middleware) != 1 {
		t.Fatalf("expected 1 middleware, got %d", len(p.middleware))
	}
}

func TestGraphQLPlugin_Use_WhileRunning(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)
	p.running = true

	mw := func(next apicore.Handler) apicore.Handler { return next }
	err := p.Use(mw)
	if err == nil {
		t.Fatal("expected error when adding middleware while running")
	}
}

func TestGraphQLPlugin_GetSchema(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"ping": &graphql.Field{Type: graphql.String},
			},
		}),
	})
	if err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}
	p.schema = schema

	if got := p.GetSchema(); got.QueryType() == nil {
		t.Fatal("GetSchema returned schema without query type")
	}
}

func TestGraphQLPlugin_BuildSchema_Fallback(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	if err := p.buildSchema(); err != nil {
		t.Fatalf("buildSchema failed: %v", err)
	}

	schema := p.GetSchema()
	if schema.QueryType() == nil {
		t.Fatal("expected Query type in schema")
	}
}

func TestGraphQLPlugin_BuildSchema_WithSchemaBuilder(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	authMiddleware := NewAuthMiddleware(logger, metrics)
	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, authMiddleware)
	p.SetSchemaBuilder(sb)

	if err := p.buildSchema(); err != nil {
		t.Fatalf("buildSchema with SchemaBuilder failed: %v", err)
	}
}

func TestGraphQLPlugin_HandlePlayground(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/graphql/playground", nil)
	p.handlePlayground(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("expected Content-Type text/html, got '%s'", contentType)
	}
	if !strings.Contains(w.Body.String(), "GraphQL Playground") {
		t.Fatal("response does not contain playground HTML")
	}
}

func TestGraphQLPlugin_IsRunning(t *testing.T) {
	t.Parallel()
	apiLayer := apicore.NewAPILayer()
	p := NewGraphQLPlugin("test", 8080, apiLayer)

	if p.IsRunning() {
		t.Fatal("expected not running initially")
	}

	p.running = true
	if !p.IsRunning() {
		t.Fatal("expected running after setting flag")
	}
}

// ========================
// eventToGraphQL tests
// ========================

func TestEventToGraphQL_WithDecodedData(t *testing.T) {
	t.Parallel()
	event := &core.BlockchainEvent{
		ID:          "evt-1",
		Status:      core.EventStatusConfirmed,
		DecodedData: map[string]any{"from": "0xabc", "to": "0xdef", "value": "100"},
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		IndexedAt:   time.Now(),
	}

	result := eventToGraphQL(event)
	if result["decodedData"] == "" {
		t.Fatal("expected non-empty decodedData")
	}
	decodedStr, ok := result["decodedData"].(string)
	if !ok || decodedStr == "" {
		t.Fatal("decodedData should be a non-empty string")
	}
}

func TestEventToGraphQL_NilDecodedData(t *testing.T) {
	t.Parallel()
	event := &core.BlockchainEvent{
		ID:          "evt-2",
		Status:      core.EventStatusPending,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		IndexedAt:   time.Now(),
	}

	result := eventToGraphQL(event)
	if result["decodedData"] != "" {
		t.Errorf("expected empty decodedData for nil DecodedData, got '%v'", result["decodedData"])
	}
}

func TestEventToGraphQL_AllFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	event := &core.BlockchainEvent{
		ID:               "evt-all",
		EventHash:        "0xhash",
		BlockNumber:      12345,
		BlockHash:        common.HexToHash("0xblockhash"),
		BlockTimestamp:   1000000,
		TransactionHash:  common.HexToHash("0xtxhash"),
		TransactionIndex: 5,
		LogIndex:         3,
		ContractAddress:  common.HexToAddress("0xcontract"),
		EventName:        "Transfer",
		ChainID:          "1",
		Network:          "mainnet",
		Status:           core.EventStatusConfirmed,
		Removed:          false,
		GasUsed:          21000,
		GasPrice:         big.NewInt(1000000000),
		DecodedData:      map[string]any{"amount": "1000"},
		CreatedAt:        now,
		ProcessedAt:      now,
		IndexedAt:        now,
	}

	result := eventToGraphQL(event)

	if result["id"] != "evt-all" {
		t.Errorf("expected id 'evt-all', got '%v'", result["id"])
	}
	if result["eventHash"] != "0xhash" {
		t.Errorf("expected eventHash '0xhash', got '%v'", result["eventHash"])
	}
	if result["blockNumber"] != uint64(12345) {
		t.Errorf("expected blockNumber 12345, got '%v'", result["blockNumber"])
	}
	if result["eventName"] != "Transfer" {
		t.Errorf("expected eventName 'Transfer', got '%v'", result["eventName"])
	}
	if result["chainId"] != "1" {
		t.Errorf("expected chainId '1', got '%v'", result["chainId"])
	}
	if result["network"] != "mainnet" {
		t.Errorf("expected network 'mainnet', got '%v'", result["network"])
	}
	if result["status"] != "confirmed" {
		t.Errorf("expected status 'confirmed', got '%v'", result["status"])
	}
	if result["transactionIndex"] != uint64(5) {
		t.Errorf("expected transactionIndex 5, got '%v'", result["transactionIndex"])
	}
	if result["logIndex"] != uint64(3) {
		t.Errorf("expected logIndex 3, got '%v'", result["logIndex"])
	}
}

// ========================
// CacheResolver ResolveClearCache tests
// ========================

func TestCacheResolver_ResolveClearCache_Success(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()

	ctx := &ResolverContext{
		Logger:  logger,
		Metrics: metrics,
		Cache:   cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"manage:cache"},
		},
	}

	resolver := NewCacheResolver(ctx)

	result, err := resolver.ResolveClearCache(mockResolveParams(nil))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestCacheResolver_ResolveClearCache_NoCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()

	ctx := &ResolverContext{
		Logger:  logger,
		Metrics: metrics,
		Cache:   nil,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"manage:cache"},
		},
	}

	resolver := NewCacheResolver(ctx)

	_, err := resolver.ResolveClearCache(mockResolveParams(nil))
	if err == nil {
		t.Fatal("expected error when cache is nil")
	}
}

func TestCacheResolver_ResolveClearCache_Unauthorized(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()

	ctx := &ResolverContext{
		Logger:  logger,
		Metrics: metrics,
		Cache:   cache,
		AuthContext: &AuthContext{
			UserID: "test-user",
			Scopes: []string{"read:events"},
		},
	}

	resolver := NewCacheResolver(ctx)

	_, err := resolver.ResolveClearCache(mockResolveParams(nil))
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ========================
// MutationBuilder resolver tests
// ========================

func TestMutationBuilder_ResolveInvalidateCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	_ = cache.Set(context.Background(), "graphql:event:test-id", []byte("cached"), 300)

	builder := NewMutationBuilder(NewMockEventStore(), logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{"eventId": "test-id"},
		Context: context.Background(),
	}

	result, err := builder.BuildMutations().Fields()["invalidateCache"].Resolve(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestMutationBuilder_ResolveInvalidateCache_MissingArg(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), NewMockCache())

	params := graphql.ResolveParams{
		Args:    map[string]any{},
		Context: context.Background(),
	}

	_, err := builder.BuildMutations().Fields()["invalidateCache"].Resolve(params)
	if err == nil {
		t.Fatal("expected error for missing eventId")
	}
}

func TestMutationBuilder_ResolveInvalidateCache_NoCache(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil)

	params := graphql.ResolveParams{
		Args:    map[string]any{"eventId": "test-id"},
		Context: context.Background(),
	}

	result, _ := builder.BuildMutations().Fields()["invalidateCache"].Resolve(params)
	if result != false {
		t.Errorf("expected false when cache is nil, got %v", result)
	}
}

func TestMutationBuilder_ResolveInvalidateCacheByPattern(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()

	builder := NewMutationBuilder(NewMockEventStore(), logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{"pattern": "graphql:*"},
		Context: context.Background(),
	}

	result, err := builder.BuildMutations().Fields()["invalidateCacheByPattern"].Resolve(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.(int) != 0 {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestMutationBuilder_ResolveInvalidateCacheByPattern_MissingArg(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), NewMockCache())

	params := graphql.ResolveParams{
		Args:    map[string]any{},
		Context: context.Background(),
	}

	_, err := builder.BuildMutations().Fields()["invalidateCacheByPattern"].Resolve(params)
	if err == nil {
		t.Fatal("expected error for missing pattern")
	}
}

func TestMutationBuilder_ResolveInvalidateCacheByPattern_NoCache(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil)

	params := graphql.ResolveParams{
		Args:    map[string]any{"pattern": "graphql:*"},
		Context: context.Background(),
	}

	result, _ := builder.BuildMutations().Fields()["invalidateCacheByPattern"].Resolve(params)
	if result.(int) != 0 {
		t.Errorf("expected 0 when cache is nil, got %v", result)
	}
}

func TestMutationBuilder_ResolveClearCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()

	builder := NewMutationBuilder(NewMockEventStore(), logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{},
		Context: context.Background(),
	}

	result, err := builder.BuildMutations().Fields()["clearCache"].Resolve(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestMutationBuilder_ResolveClearCache_NoCache(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil)

	params := graphql.ResolveParams{
		Args:    map[string]any{},
		Context: context.Background(),
	}

	result, _ := builder.BuildMutations().Fields()["clearCache"].Resolve(params)
	if result != false {
		t.Errorf("expected false when cache is nil, got %v", result)
	}
}

func TestMutationBuilder_ResolveRefreshEventCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()
	eventStore.events["evt-refresh"] = &core.BlockchainEvent{
		ID:          "evt-refresh",
		Status:      core.EventStatusConfirmed,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		IndexedAt:   time.Now(),
	}

	builder := NewMutationBuilder(eventStore, logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{"eventId": "evt-refresh"},
		Context: context.Background(),
	}

	result, err := builder.BuildMutations().Fields()["refreshEventCache"].Resolve(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestMutationBuilder_ResolveRefreshEventCache_MissingArg(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), NewMockCache())

	params := graphql.ResolveParams{
		Args:    map[string]any{},
		Context: context.Background(),
	}

	_, err := builder.BuildMutations().Fields()["refreshEventCache"].Resolve(params)
	if err == nil {
		t.Fatal("expected error for missing eventId")
	}
}

func TestMutationBuilder_ResolveRefreshEventCache_NoCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.events["evt-refresh"] = &core.BlockchainEvent{
		ID:          "evt-refresh",
		Status:      core.EventStatusConfirmed,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		IndexedAt:   time.Now(),
	}

	builder := NewMutationBuilder(eventStore, logger, metrics, nil)

	params := graphql.ResolveParams{
		Args:    map[string]any{"eventId": "evt-refresh"},
		Context: context.Background(),
	}

	result, _ := builder.BuildMutations().Fields()["refreshEventCache"].Resolve(params)
	if result != false {
		t.Errorf("expected false when cache is nil, got %v", result)
	}
}

func TestMutationBuilder_ResolveRefreshEventCache_EventNotFound(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()

	builder := NewMutationBuilder(eventStore, logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{"eventId": "nonexistent"},
		Context: context.Background(),
	}

	_, err := builder.BuildMutations().Fields()["refreshEventCache"].Resolve(params)
	if err == nil {
		t.Fatal("expected error for non-existent event")
	}
}

func TestMutationBuilder_ResolveWarmCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()
	eventStore.getEventsPaginated = func(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-warm-1",
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
			{
				ID:          "evt-warm-2",
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, false, nil
	}

	builder := NewMutationBuilder(eventStore, logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{"limit": 10},
		Context: context.Background(),
	}

	result, err := builder.BuildMutations().Fields()["warmCache"].Resolve(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.(int) != 2 {
		t.Errorf("expected 2 cached events, got %v", result)
	}
}

func TestMutationBuilder_ResolveWarmCache_NoCache(t *testing.T) {
	t.Parallel()
	builder := NewMutationBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil)

	params := graphql.ResolveParams{
		Args:    map[string]any{},
		Context: context.Background(),
	}

	result, _ := builder.BuildMutations().Fields()["warmCache"].Resolve(params)
	if result.(int) != 0 {
		t.Errorf("expected 0 when cache is nil, got %v", result)
	}
}

func TestMutationBuilder_ResolveWarmCache_LimitCapped(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()
	eventStore.getEventsPaginated = func(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
		if limit > maxWarmCacheLimit {
			t.Errorf("limit should be capped at %d, got %d", maxWarmCacheLimit, limit)
		}
		return []*core.BlockchainEvent{}, false, nil
	}

	builder := NewMutationBuilder(eventStore, logger, metrics, cache)

	params := graphql.ResolveParams{
		Args:    map[string]any{"limit": 9999},
		Context: context.Background(),
	}

	_, err := builder.BuildMutations().Fields()["warmCache"].Resolve(params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// ========================
// SchemaBuilder resolve/subscribe tests
// ========================

func TestSchemaBuilder_SetSubscriptionManager(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)
	sm := NewSubscriptionManager(NewMockLogger(), NewMockMetrics(), nil)
	sb.SetSubscriptionManager(sm)

	if sb.subscriptionManager != sm {
		t.Fatal("subscriptionManager was not set")
	}
}

func TestSchemaBuilder_SubscribeEventCreated_NoManager(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.subscribeEventCreated(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when subscription manager is nil")
	}
}

func TestSchemaBuilder_SubscribeEventConfirmed_NoManager(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.subscribeEventConfirmed(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when subscription manager is nil")
	}
}

func TestSchemaBuilder_SubscribeEventFailed_NoManager(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.subscribeEventFailed(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when subscription manager is nil")
	}
}

func TestSchemaBuilder_ResolveEventsByBlock(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByBlock = func(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-block-1",
				BlockNumber: uint64(blockNumber),
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, nil)

	result, err := sb.resolveEventsByBlock(mockResolveParams(map[string]any{
		"blockNumber": 42,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result, got %#v", result)
	}
}

func TestSchemaBuilder_ResolveEventsByBlock_BadArg(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.resolveEventsByBlock(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing blockNumber")
	}
}

func TestSchemaBuilder_ResolveEventsByAddress(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByAddress = func(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:              "evt-addr-1",
				ContractAddress: common.HexToAddress(address),
				Status:          core.EventStatusConfirmed,
				CreatedAt:       time.Now(),
				ProcessedAt:     time.Now(),
				IndexedAt:       time.Now(),
			},
		}, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, nil)

	result, err := sb.resolveEventsByAddress(mockResolveParams(map[string]any{
		"address": "0x1234567890123456789012345678901234567890",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result, got %#v", result)
	}
}

func TestSchemaBuilder_ResolveEventsByAddress_BadArg(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.resolveEventsByAddress(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing address")
	}
}

func TestSchemaBuilder_ResolveEventsByAddress_WithCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()
	eventStore.getEventsByAddress = func(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:              "evt-addr-cache",
				ContractAddress: common.HexToAddress(address),
				Status:          core.EventStatusConfirmed,
				CreatedAt:       time.Now(),
				ProcessedAt:     time.Now(),
				IndexedAt:       time.Now(),
			},
		}, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, cache, nil)

	result, err := sb.resolveEventsByAddress(mockResolveParams(map[string]any{
		"address": "0x1234567890123456789012345678901234567890",
		"limit":   10,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result, got %#v", result)
	}
}

func TestSchemaBuilder_ResolveInvalidateCache_NoAuth(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	authMiddleware := NewAuthMiddleware(logger, metrics)
	sb := NewSchemaBuilder(NewMockEventStore(), logger, metrics, NewMockCache(), authMiddleware)

	_, err := sb.resolveInvalidateCache(mockResolveParams(map[string]any{
		"eventId": "evt-1",
	}))
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestSchemaBuilder_ResolveInvalidateCache(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	result, err := sb.resolveInvalidateCache(mockResolveParams(map[string]any{
		"eventId": "evt-1",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestSchemaBuilder_ResolveClearCache_NoAuth(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	authMiddleware := NewAuthMiddleware(logger, metrics)
	sb := NewSchemaBuilder(NewMockEventStore(), logger, metrics, NewMockCache(), authMiddleware)

	_, err := sb.resolveClearCache(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestSchemaBuilder_ResolveClearCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	sb := NewSchemaBuilder(NewMockEventStore(), logger, metrics, cache, nil)

	result, err := sb.resolveClearCache(mockResolveParams(map[string]any{}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestSchemaBuilder_ResolveEvents_Filter(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsPaginated = func(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-filter-1",
				EventName:   "Transfer",
				BlockNumber: 100,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
			{
				ID:          "evt-filter-2",
				EventName:   "Approval",
				BlockNumber: 200,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
			{
				ID:          "evt-filter-3",
				EventName:   "Transfer",
				BlockNumber: 300,
				Status:      core.EventStatusFailed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, false, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, nil)

	result, err := sb.resolveEvents(mockResolveParams(map[string]any{
		"first": 10,
		"filter": map[string]any{
			"eventName": "Transfer",
		},
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	connection, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected connection, got %#v", result)
	}
	edges, ok := connection["edges"].([]any)
	if !ok {
		t.Fatalf("expected edges, got %#v", connection["edges"])
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 events matching Transfer, got %d", len(edges))
	}
}

func TestSchemaBuilder_ResolveEvents_FilterWithSort(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsPaginated = func(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-sort-1",
				BlockNumber: 200,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
			{
				ID:          "evt-sort-2",
				BlockNumber: 100,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, false, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, nil)

	result, err := sb.resolveEvents(mockResolveParams(map[string]any{
		"first": 10,
		"sort": map[string]any{
			"field": "blockNumber",
			"order": "asc",
		},
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	connection, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected connection, got %#v", result)
	}
	edges, ok := connection["edges"].([]any)
	if !ok || len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	firstEdge, _ := edges[0].(map[string]any)
	firstNode, _ := firstEdge["node"].(map[string]any)
	if firstNode["blockNumber"] != uint64(100) {
		t.Errorf("expected blockNumber 100 first (asc), got %v", firstNode["blockNumber"])
	}
}

func TestSchemaBuilder_ResolveEvent_NoCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.events["evt-nocache"] = &core.BlockchainEvent{
		ID:          "evt-nocache",
		Status:      core.EventStatusConfirmed,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		IndexedAt:   time.Now(),
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, nil)

	result, err := sb.resolveEvent(mockResolveParams(map[string]any{
		"id": "evt-nocache",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	item, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %#v", result)
	}
	if item["id"] != "evt-nocache" {
		t.Errorf("expected id 'evt-nocache', got '%v'", item["id"])
	}
}

func TestSchemaBuilder_ResolveEvent_BadArg(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.resolveEvent(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestSchemaBuilder_ResolveEventsByName_NoCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	eventStore := NewMockEventStore()
	eventStore.getEventsByName = func(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-name-nocache",
				EventName:   eventName,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, nil, nil)

	result, err := sb.resolveEventsByName(mockResolveParams(map[string]any{
		"eventName": "Transfer",
		"limit":     10,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result, got %#v", result)
	}
}

func TestSchemaBuilder_ResolveEventsByName_WithCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	eventStore := NewMockEventStore()
	eventStore.getEventsByName = func(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
		return []*core.BlockchainEvent{
			{
				ID:          "evt-name-cache-set",
				EventName:   eventName,
				Status:      core.EventStatusConfirmed,
				CreatedAt:   time.Now(),
				ProcessedAt: time.Now(),
				IndexedAt:   time.Now(),
			},
		}, nil
	}

	sb := NewSchemaBuilder(eventStore, logger, metrics, cache, nil)

	result, err := sb.resolveEventsByName(mockResolveParams(map[string]any{
		"eventName": "Transfer",
		"limit":     10,
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	items, ok := result.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one result, got %#v", result)
	}
}

func TestSchemaBuilder_ResolveInvalidateCache_WithCache(t *testing.T) {
	t.Parallel()
	logger := NewMockLogger()
	metrics := NewMockMetrics()
	cache := NewMockCache()
	_ = cache.Set(context.Background(), "graphql:event:evt-cache-inv", []byte("data"), 300)
	sb := NewSchemaBuilder(NewMockEventStore(), logger, metrics, cache, nil)

	result, err := sb.resolveInvalidateCache(mockResolveParams(map[string]any{
		"eventId": "evt-cache-inv",
	}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestSchemaBuilder_ResolveInvalidateCache_BadArg(t *testing.T) {
	t.Parallel()
	sb := NewSchemaBuilder(NewMockEventStore(), NewMockLogger(), NewMockMetrics(), nil, nil)

	_, err := sb.resolveInvalidateCache(mockResolveParams(map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing eventId")
	}
}
