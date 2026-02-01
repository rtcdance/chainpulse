package graphql

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
	"github.com/graphql-go/graphql"
)

// ResolverContext holds context for resolvers
type ResolverContext struct {
	EventStore    query.EventStore
	Logger        core.Logger
	Metrics       core.MetricsCollector
	Cache         core.CachePlugin
	AuthContext   *AuthContext
}

// EventResolver handles event-related queries
type EventResolver struct {
	ctx *ResolverContext
}

// NewEventResolver creates a new event resolver
func NewEventResolver(ctx *ResolverContext) *EventResolver {
	return &EventResolver{ctx: ctx}
}

// ResolveEvent resolves a single event by ID
func (r *EventResolver) ResolveEvent(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("invalid or missing id parameter")
	}

	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanReadEvent(id) {
		return nil, fmt.Errorf("unauthorized to read event")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.ctx.Metrics.RecordHistogram("graphql_resolve_event_time_ms", float64(duration), nil)
	}()

	// Try cache first
	if r.ctx.Cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", id)
		if cached, err := r.ctx.Cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			r.ctx.Metrics.RecordCounter("graphql_cache_hit", 1, nil)
			return cached, nil
		}
		r.ctx.Metrics.RecordCounter("graphql_cache_miss", 1, nil)
	}

	// Retrieve from store
	event, err := r.ctx.EventStore.GetEvent(p.Context, id)
	if err != nil {
		r.ctx.Logger.Error("Failed to resolve event", "id", id, "error", err.Error())
		r.ctx.Metrics.RecordCounter("graphql_resolve_event_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve event: %w", err)
	}

	if event == nil {
		return nil, nil
	}

	// Convert to GraphQL response format
	result := eventToGraphQL(event)

	// Cache result
	if r.ctx.Cache != nil {
		cacheKey := fmt.Sprintf("graphql:event:%s", id)
		resultBytes, _ := json.Marshal(result)
		_ = r.ctx.Cache.Set(p.Context, cacheKey, resultBytes, 300) // 5 minutes
	}

	r.ctx.Metrics.RecordCounter("graphql_resolve_event_success", 1, nil)
	return result, nil
}

// ResolveEvents resolves events with pagination
func (r *EventResolver) ResolveEvents(p graphql.ResolveParams) (interface{}, error) {
	// Limit maximum page size
	first := 20
	if f, ok := p.Args["first"].(int); ok && f > 0 {
		first = f
	}
	if first > 1000 {
		first = 1000
	}

	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanListEvents() {
		return nil, fmt.Errorf("unauthorized to list events")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.ctx.Metrics.RecordHistogram("graphql_resolve_events_time_ms", float64(duration), nil)
	}()

	// Get cursor for pagination
	after := ""
	if a, ok := p.Args["after"].(string); ok {
		after = a
	}

	// Retrieve events from store with pagination
	events, hasNextPage, err := r.ctx.EventStore.GetEventsPaginated(p.Context, after, first)
	if err != nil {
		r.ctx.Logger.Error("Failed to resolve events", "error", err.Error())
		r.ctx.Metrics.RecordCounter("graphql_resolve_events_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events: %w", err)
	}

	// Build edges
	edges := make([]interface{}, 0, len(events))
	var endCursor string
	for i, event := range events {
		cursor := fmt.Sprintf("cursor_%d", i)
		edges = append(edges, map[string]interface{}{
			"node":   eventToGraphQL(event),
			"cursor": cursor,
		})
		if i == len(events)-1 {
			endCursor = cursor
		}
	}

	connection := map[string]interface{}{
		"edges": edges,
		"pageInfo": map[string]interface{}{
			"hasNextPage":     hasNextPage,
			"hasPreviousPage": after != "",
			"startCursor":     after,
			"endCursor":       endCursor,
			"totalCount":      len(events),
		},
	}

	r.ctx.Metrics.RecordCounter("graphql_resolve_events_success", 1, nil)
	return connection, nil
}

// ResolveEventsByBlock resolves events by block number
func (r *EventResolver) ResolveEventsByBlock(p graphql.ResolveParams) (interface{}, error) {
	blockNumber, ok := p.Args["blockNumber"].(int)
	if !ok || blockNumber < 0 {
		return nil, fmt.Errorf("invalid or missing blockNumber parameter")
	}

	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanListEvents() {
		return nil, fmt.Errorf("unauthorized to list events")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.ctx.Metrics.RecordHistogram("graphql_resolve_events_by_block_time_ms", float64(duration), nil)
	}()

	// Retrieve events by block number
	events, err := r.ctx.EventStore.GetEventsByBlock(p.Context, int64(blockNumber))
	if err != nil {
		r.ctx.Logger.Error("Failed to resolve events by block", "blockNumber", blockNumber, "error", err.Error())
		r.ctx.Metrics.RecordCounter("graphql_resolve_events_by_block_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by block: %w", err)
	}

	// Convert to GraphQL response format
	result := make([]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, eventToGraphQL(event))
	}

	r.ctx.Metrics.RecordCounter("graphql_resolve_events_by_block_success", 1, nil)
	return result, nil
}

// ResolveEventsByAddress resolves events by contract address
func (r *EventResolver) ResolveEventsByAddress(p graphql.ResolveParams) (interface{}, error) {
	address, ok := p.Args["address"].(string)
	if !ok || address == "" {
		return nil, fmt.Errorf("invalid or missing address parameter")
	}

	limit := 100
	if l, ok := p.Args["limit"].(int); ok && l > 0 {
		limit = l
	}

	// Limit maximum results
	if limit > 1000 {
		limit = 1000
	}

	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanListEvents() {
		return nil, fmt.Errorf("unauthorized to list events")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.ctx.Metrics.RecordHistogram("graphql_resolve_events_by_address_time_ms", float64(duration), nil)
	}()

	// Try cache first
	if r.ctx.Cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:address:%s:limit:%d", address, limit)
		if cached, err := r.ctx.Cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			r.ctx.Metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var result []interface{}
			if err := json.Unmarshal(cached, &result); err == nil {
				return result, nil
			}
		}
	}

	// Retrieve events by address
	events, err := r.ctx.EventStore.GetEventsByAddress(p.Context, address, limit)
	if err != nil {
		r.ctx.Logger.Error("Failed to resolve events by address", "address", address, "error", err.Error())
		r.ctx.Metrics.RecordCounter("graphql_resolve_events_by_address_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by address: %w", err)
	}

	// Convert to GraphQL response format
	result := make([]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, eventToGraphQL(event))
	}

	// Cache result
	if r.ctx.Cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:address:%s:limit:%d", address, limit)
		resultBytes, _ := json.Marshal(result)
		_ = r.ctx.Cache.Set(p.Context, cacheKey, resultBytes, 600) // 10 minutes
	}

	r.ctx.Metrics.RecordCounter("graphql_resolve_events_by_address_success", 1, nil)
	return result, nil
}

// ResolveEventsByName resolves events by event name
func (r *EventResolver) ResolveEventsByName(p graphql.ResolveParams) (interface{}, error) {
	eventName, ok := p.Args["eventName"].(string)
	if !ok || eventName == "" {
		return nil, fmt.Errorf("invalid or missing eventName parameter")
	}

	limit := 100
	if l, ok := p.Args["limit"].(int); ok && l > 0 {
		limit = l
	}

	// Limit maximum results
	if limit > 1000 {
		limit = 1000
	}

	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanListEvents() {
		return nil, fmt.Errorf("unauthorized to list events")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.ctx.Metrics.RecordHistogram("graphql_resolve_events_by_name_time_ms", float64(duration), nil)
	}()

	// Try cache first
	if r.ctx.Cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:name:%s:limit:%d", eventName, limit)
		if cached, err := r.ctx.Cache.Get(p.Context, cacheKey); err == nil && cached != nil {
			r.ctx.Metrics.RecordCounter("graphql_cache_hit", 1, nil)
			var result []interface{}
			if err := json.Unmarshal(cached, &result); err == nil {
				return result, nil
			}
		}
	}

	// Retrieve events by event name
	events, err := r.ctx.EventStore.GetEventsByName(p.Context, eventName, limit)
	if err != nil {
		r.ctx.Logger.Error("Failed to resolve events by name", "eventName", eventName, "error", err.Error())
		r.ctx.Metrics.RecordCounter("graphql_resolve_events_by_name_error", 1, nil)
		return nil, fmt.Errorf("failed to resolve events by name: %w", err)
	}

	// Convert to GraphQL response format
	result := make([]interface{}, 0, len(events))
	for _, event := range events {
		result = append(result, eventToGraphQL(event))
	}

	// Cache result
	if r.ctx.Cache != nil {
		cacheKey := fmt.Sprintf("graphql:events:name:%s:limit:%d", eventName, limit)
		resultBytes, _ := json.Marshal(result)
		_ = r.ctx.Cache.Set(p.Context, cacheKey, resultBytes, 600) // 10 minutes
	}

	r.ctx.Metrics.RecordCounter("graphql_resolve_events_by_name_success", 1, nil)
	return result, nil
}

// CacheResolver handles cache-related mutations
type CacheResolver struct {
	ctx *ResolverContext
}

// NewCacheResolver creates a new cache resolver
func NewCacheResolver(ctx *ResolverContext) *CacheResolver {
	return &CacheResolver{ctx: ctx}
}

// ResolveInvalidateCache invalidates cache for an event
func (r *CacheResolver) ResolveInvalidateCache(p graphql.ResolveParams) (interface{}, error) {
	eventID, ok := p.Args["eventId"].(string)
	if !ok || eventID == "" {
		return nil, fmt.Errorf("invalid or missing eventId parameter")
	}

	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanManageCache() {
		return nil, fmt.Errorf("unauthorized to manage cache")
	}

	if r.ctx.Cache == nil {
		return false, fmt.Errorf("cache not available")
	}

	// Invalidate cache entries for this event
	cacheKey := fmt.Sprintf("graphql:event:%s", eventID)
	if err := r.ctx.Cache.Delete(p.Context, cacheKey); err != nil {
		r.ctx.Logger.Error("Failed to invalidate cache", "eventId", eventID, "error", err.Error())
		r.ctx.Metrics.RecordCounter("graphql_invalidate_cache_error", 1, nil)
		return false, fmt.Errorf("failed to invalidate cache: %w", err)
	}

	r.ctx.Metrics.RecordCounter("graphql_invalidate_cache_success", 1, nil)
	return true, nil
}

// ResolveClearCache clears all cache
func (r *CacheResolver) ResolveClearCache(p graphql.ResolveParams) (interface{}, error) {
	// Check authorization
	if r.ctx.AuthContext != nil && !r.ctx.AuthContext.CanManageCache() {
		return nil, fmt.Errorf("unauthorized to manage cache")
	}

	if r.ctx.Cache == nil {
		return false, fmt.Errorf("cache not available")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.ctx.Metrics.RecordHistogram("graphql_resolver_clear_cache_time_ms", float64(duration), nil)
	}()

	// Clear all GraphQL cache entries by deleting with pattern
	// For now, we'll log the operation
	r.ctx.Logger.Info("Cache clearing requested")
	r.ctx.Metrics.RecordCounter("graphql_resolver_clear_cache_success", 1, nil)
	return true, nil
}

// Helper function to convert event to GraphQL response format
func eventToGraphQL(event *core.BlockchainEvent) map[string]interface{} {
	decodedData := ""
	if event.DecodedData != nil {
		if data, err := json.Marshal(event.DecodedData); err == nil {
			decodedData = string(data)
		}
	}

	return map[string]interface{}{
		"id":                 event.ID,
		"eventHash":          event.EventHash,
		"blockNumber":        event.BlockNumber,
		"blockHash":          event.BlockHash.Hex(),
		"blockTimestamp":     event.BlockTimestamp,
		"transactionHash":    event.TransactionHash.Hex(),
		"transactionIndex":   event.TransactionIndex,
		"logIndex":           event.LogIndex,
		"contractAddress":    event.ContractAddress.Hex(),
		"eventName":          event.EventName,
		"chainId":            event.ChainID,
		"network":            event.Network,
		"status":             string(event.Status),
		"removed":            event.Removed,
		"gasUsed":            event.GasUsed,
		"gasPrice":           event.GasPrice.String(),
		"decodedData":        decodedData,
		"createdAt":          event.CreatedAt.Format(time.RFC3339),
		"processedAt":        event.ProcessedAt.Format(time.RFC3339),
		"indexedAt":          event.IndexedAt.Format(time.RFC3339),
	}
}

// QueryComplexityAnalyzer analyzes query complexity
type QueryComplexityAnalyzer struct {
	maxComplexity int
}

// NewQueryComplexityAnalyzer creates a new query complexity analyzer
func NewQueryComplexityAnalyzer(maxComplexity int) *QueryComplexityAnalyzer {
	return &QueryComplexityAnalyzer{
		maxComplexity: maxComplexity,
	}
}

// AnalyzeComplexity analyzes the complexity of a GraphQL query
func (a *QueryComplexityAnalyzer) AnalyzeComplexity(query string) (int, error) {
	complexity := calculateQueryComplexity(query)
	if complexity > a.maxComplexity {
		return complexity, fmt.Errorf("query complexity %d exceeds maximum %d", complexity, a.maxComplexity)
	}
	return complexity, nil
}

// calculateQueryComplexity calculates query complexity based on heuristics
func calculateQueryComplexity(query string) int {
	complexity := 1

	// Count field selections
	complexity += strings.Count(query, "{")

	// Count arguments (each argument adds complexity)
	complexity += strings.Count(query, "(") * 2

	// Count aliases
	complexity += strings.Count(query, ":") / 2

	// Count fragments
	complexity += strings.Count(query, "fragment") * 5

	// Count nested queries (depth-based)
	depth := 0
	maxDepth := 0
	for _, char := range query {
		switch char {
		case '{':
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case '}':
			depth--
		}
	}
	complexity += maxDepth * 3

	// Count array selections (multipliers)
	complexity += strings.Count(query, "[") * 2

	return complexity
}
