package cache

import (
	"strings"
	"testing"
)

// TestNewCacheKeyGenerator tests creating a new cache key generator
func TestNewCacheKeyGenerator(t *testing.T) {
	t.Parallel()
	prefix := "test"
	ckg := NewCacheKeyGenerator(prefix)

	if ckg == nil {
		t.Fatal("expected non-nil CacheKeyGenerator")
	}

	if ckg.prefix != prefix {
		t.Fatalf("expected prefix %s, got %s", prefix, ckg.prefix)
	}
}

// TestGenerateEventKey tests generating event cache keys
func TestGenerateEventKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	eventID := "event123"

	key := ckg.GenerateEventKey(eventID)

	if !strings.Contains(key, "test") {
		t.Fatal("expected key to contain prefix")
	}

	if !strings.Contains(key, "event") {
		t.Fatal("expected key to contain 'event'")
	}

	if !strings.Contains(key, eventID) {
		t.Fatal("expected key to contain event ID")
	}
}

// TestGenerateEventsByAddressKey tests generating events by address cache keys
func TestGenerateEventsByAddressKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	address := "0x123abc"
	offset := 0
	limit := 10

	key := ckg.GenerateEventsByAddressKey(address, offset, limit)

	if !strings.Contains(key, "address") {
		t.Fatal("expected key to contain 'address'")
	}

	if !strings.Contains(key, address) {
		t.Fatal("expected key to contain address")
	}

	if !strings.Contains(key, "offset") {
		t.Fatal("expected key to contain 'offset'")
	}

	if !strings.Contains(key, "limit") {
		t.Fatal("expected key to contain 'limit'")
	}
}

// TestGenerateEventsByBlockKey tests generating events by block cache keys
func TestGenerateEventsByBlockKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	blockNumber := int64(12345)
	offset := 0
	limit := 10

	key := ckg.GenerateEventsByBlockKey(blockNumber, offset, limit)

	if !strings.Contains(key, "block") {
		t.Fatal("expected key to contain 'block'")
	}

	if !strings.Contains(key, "12345") {
		t.Fatal("expected key to contain block number")
	}
}

// TestGenerateEventsByTopicKey tests generating events by topic cache keys
func TestGenerateEventsByTopicKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	topic := "Transfer"
	offset := 0
	limit := 10

	key := ckg.GenerateEventsByTopicKey(topic, offset, limit)

	if !strings.Contains(key, "topic") {
		t.Fatal("expected key to contain 'topic'")
	}

	if !strings.Contains(key, topic) {
		t.Fatal("expected key to contain topic")
	}
}

// TestGenerateEventCountKey tests generating event count cache keys
func TestGenerateEventCountKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	filters := map[string]any{
		"address": "0x123abc",
		"status":  "confirmed",
	}

	key := ckg.GenerateEventCountKey(filters)

	if !strings.Contains(key, "count") {
		t.Fatal("expected key to contain 'count'")
	}

	// Hash should be consistent
	key2 := ckg.GenerateEventCountKey(filters)
	if key != key2 {
		t.Fatal("expected consistent hash for same filters")
	}
}

// TestGenerateAggregationKey tests generating aggregation cache keys
func TestGenerateAggregationKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	aggregationType := "sum"
	timeWindow := "1h"
	filters := map[string]any{
		"address": "0x123abc",
	}

	key := ckg.GenerateAggregationKey(aggregationType, timeWindow, filters)

	if !strings.Contains(key, "aggregation") {
		t.Fatal("expected key to contain 'aggregation'")
	}

	if !strings.Contains(key, aggregationType) {
		t.Fatal("expected key to contain aggregation type")
	}

	if !strings.Contains(key, timeWindow) {
		t.Fatal("expected key to contain time window")
	}
}

// TestGenerateQueryKey tests generating query cache keys
func TestGenerateQueryKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	queryType := "events"
	params := map[string]any{
		"limit":  10,
		"offset": 0,
	}

	key := ckg.GenerateQueryKey(queryType, params)

	if !strings.Contains(key, "query") {
		t.Fatal("expected key to contain 'query'")
	}

	if !strings.Contains(key, queryType) {
		t.Fatal("expected key to contain query type")
	}
}

// TestGenerateGraphQLKey tests generating GraphQL cache keys
func TestGenerateGraphQLKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	query := "{ events { id name } }"
	variables := map[string]any{
		"limit": 10,
	}

	key := ckg.GenerateGraphQLKey(query, variables)

	if !strings.Contains(key, "graphql") {
		t.Fatal("expected key to contain 'graphql'")
	}
}

// TestGenerateSubscriptionKey tests generating subscription cache keys
func TestGenerateSubscriptionKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	subscriptionID := "sub123"

	key := ckg.GenerateSubscriptionKey(subscriptionID)

	if !strings.Contains(key, "subscription") {
		t.Fatal("expected key to contain 'subscription'")
	}

	if !strings.Contains(key, subscriptionID) {
		t.Fatal("expected key to contain subscription ID")
	}
}

// TestGenerateMetadataKey tests generating metadata cache keys
func TestGenerateMetadataKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	metadataType := "contract"
	id := "0x123abc"

	key := ckg.GenerateMetadataKey(metadataType, id)

	if !strings.Contains(key, "metadata") {
		t.Fatal("expected key to contain 'metadata'")
	}

	if !strings.Contains(key, metadataType) {
		t.Fatal("expected key to contain metadata type")
	}

	if !strings.Contains(key, id) {
		t.Fatal("expected key to contain ID")
	}
}

// TestGenerateIndexKey tests generating index cache keys
func TestGenerateIndexKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	indexName := "events_address_idx"

	key := ckg.GenerateIndexKey(indexName)

	if !strings.Contains(key, "index") {
		t.Fatal("expected key to contain 'index'")
	}

	if !strings.Contains(key, indexName) {
		t.Fatal("expected key to contain index name")
	}
}

// TestGenerateStatsKey tests generating stats cache keys
func TestGenerateStatsKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	statsType := "daily"

	key := ckg.GenerateStatsKey(statsType)

	if !strings.Contains(key, "stats") {
		t.Fatal("expected key to contain 'stats'")
	}

	if !strings.Contains(key, statsType) {
		t.Fatal("expected key to contain stats type")
	}
}

// TestGenerateHealthKey tests generating health cache keys
func TestGenerateHealthKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")

	key := ckg.GenerateHealthKey()

	if !strings.Contains(key, "health") {
		t.Fatal("expected key to contain 'health'")
	}
}

// TestGenerateConfigKey tests generating config cache keys
func TestGenerateConfigKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	configType := "api"

	key := ckg.GenerateConfigKey(configType)

	if !strings.Contains(key, "config") {
		t.Fatal("expected key to contain 'config'")
	}

	if !strings.Contains(key, configType) {
		t.Fatal("expected key to contain config type")
	}
}

// TestGenerateSessionKey tests generating session cache keys
func TestGenerateSessionKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	sessionID := "sess123"

	key := ckg.GenerateSessionKey(sessionID)

	if !strings.Contains(key, "session") {
		t.Fatal("expected key to contain 'session'")
	}

	if !strings.Contains(key, sessionID) {
		t.Fatal("expected key to contain session ID")
	}
}

// TestGenerateUserKey tests generating user cache keys
func TestGenerateUserKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	userID := "user123"

	key := ckg.GenerateUserKey(userID)

	if !strings.Contains(key, "user") {
		t.Fatal("expected key to contain 'user'")
	}

	if !strings.Contains(key, userID) {
		t.Fatal("expected key to contain user ID")
	}
}

// TestGeneratePermissionKey tests generating permission cache keys
func TestGeneratePermissionKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	userID := "user123"
	resource := "events"

	key := ckg.GeneratePermissionKey(userID, resource)

	if !strings.Contains(key, "permission") {
		t.Fatal("expected key to contain 'permission'")
	}

	if !strings.Contains(key, userID) {
		t.Fatal("expected key to contain user ID")
	}

	if !strings.Contains(key, resource) {
		t.Fatal("expected key to contain resource")
	}
}

// TestGenerateRateLimitKey tests generating rate limit cache keys
func TestGenerateRateLimitKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	clientID := "client123"

	key := ckg.GenerateRateLimitKey(clientID)

	if !strings.Contains(key, "ratelimit") {
		t.Fatal("expected key to contain 'ratelimit'")
	}

	if !strings.Contains(key, clientID) {
		t.Fatal("expected key to contain client ID")
	}
}

// TestGenerateEventsByTimeRangeKey tests generating events by time range cache keys
func TestGenerateEventsByTimeRangeKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	startTime := int64(1000000)
	endTime := int64(2000000)
	offset := 0
	limit := 10

	key := ckg.GenerateEventsByTimeRangeKey(startTime, endTime, offset, limit)

	if !strings.Contains(key, "time") {
		t.Fatal("expected key to contain 'time'")
	}

	if !strings.Contains(key, "1000000") {
		t.Fatal("expected key to contain start time")
	}

	if !strings.Contains(key, "2000000") {
		t.Fatal("expected key to contain end time")
	}
}

// TestGenerateEventsByTypeKey tests generating events by type cache keys
func TestGenerateEventsByTypeKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	eventType := "Transfer"
	offset := 0
	limit := 10

	key := ckg.GenerateEventsByTypeKey(eventType, offset, limit)

	if !strings.Contains(key, "type") {
		t.Fatal("expected key to contain 'type'")
	}

	if !strings.Contains(key, eventType) {
		t.Fatal("expected key to contain event type")
	}
}

// TestGenerateEventsByStatusKey tests generating events by status cache keys
func TestGenerateEventsByStatusKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	status := "confirmed"
	offset := 0
	limit := 10

	key := ckg.GenerateEventsByStatusKey(status, offset, limit)

	if !strings.Contains(key, "status") {
		t.Fatal("expected key to contain 'status'")
	}

	if !strings.Contains(key, status) {
		t.Fatal("expected key to contain status")
	}
}

// TestGenerateEventSearchKey tests generating event search cache keys
func TestGenerateEventSearchKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	searchQuery := "Transfer from 0x123"
	offset := 0
	limit := 10

	key := ckg.GenerateEventSearchKey(searchQuery, offset, limit)

	if !strings.Contains(key, "search") {
		t.Fatal("expected key to contain 'search'")
	}
}

// TestGenerateRelatedEventsKey tests generating related events cache keys
func TestGenerateRelatedEventsKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	eventID := "event123"

	key := ckg.GenerateRelatedEventsKey(eventID)

	if !strings.Contains(key, "related") {
		t.Fatal("expected key to contain 'related'")
	}

	if !strings.Contains(key, eventID) {
		t.Fatal("expected key to contain event ID")
	}
}

// TestGenerateEventChainKey tests generating event chain cache keys
func TestGenerateEventChainKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	eventID := "event123"

	key := ckg.GenerateEventChainKey(eventID)

	if !strings.Contains(key, "chain") {
		t.Fatal("expected key to contain 'chain'")
	}

	if !strings.Contains(key, eventID) {
		t.Fatal("expected key to contain event ID")
	}
}

// TestGenerateEventHistoryKey tests generating event history cache keys
func TestGenerateEventHistoryKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	eventID := "event123"

	key := ckg.GenerateEventHistoryKey(eventID)

	if !strings.Contains(key, "history") {
		t.Fatal("expected key to contain 'history'")
	}

	if !strings.Contains(key, eventID) {
		t.Fatal("expected key to contain event ID")
	}
}

// TestGenerateEventDependenciesKey tests generating event dependencies cache keys
func TestGenerateEventDependenciesKey(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")
	eventID := "event123"

	key := ckg.GenerateEventDependenciesKey(eventID)

	if !strings.Contains(key, "dependencies") {
		t.Fatal("expected key to contain 'dependencies'")
	}

	if !strings.Contains(key, eventID) {
		t.Fatal("expected key to contain event ID")
	}
}

// TestConsistentHashing tests that hashing is consistent
func TestConsistentHashing(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")

	filters := map[string]any{
		"address": "0x123abc",
		"status":  "confirmed",
	}

	key1 := ckg.GenerateEventCountKey(filters)
	key2 := ckg.GenerateEventCountKey(filters)

	if key1 != key2 {
		t.Fatal("expected consistent hashing for same filters")
	}
}

// TestEmptyFiltersHashing tests hashing with empty filters
func TestEmptyFiltersHashing(t *testing.T) {
	t.Parallel()
	ckg := NewCacheKeyGenerator("test")

	filters := make(map[string]any)

	key := ckg.GenerateEventCountKey(filters)

	if !strings.Contains(key, "empty") {
		t.Fatal("expected key to contain 'empty' for empty filters")
	}
}
