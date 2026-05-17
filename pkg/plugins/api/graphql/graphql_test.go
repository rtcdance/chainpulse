package graphql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/graphql-go/graphql"
	"github.com/rtcdance/chainpulse/pkg/core"
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

func (m *MockCache) Health() error {
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

func (m *MockCache) Initialize(config core.Config) error {
	return nil
}

func (m *MockCache) Start() error {
	return nil
}

func (m *MockCache) Stop() error {
	return nil
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
