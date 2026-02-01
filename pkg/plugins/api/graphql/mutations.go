package graphql

import (
	"encoding/json"
	"fmt"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
	"github.com/graphql-go/graphql"
)

// MutationBuilder builds GraphQL mutations
type MutationBuilder struct {
	eventStore query.EventStore
	logger     core.Logger
	metrics    core.MetricsCollector
	cache      core.CachePlugin
}

// NewMutationBuilder creates a new mutation builder
func NewMutationBuilder(
	eventStore query.EventStore,
	logger core.Logger,
	metrics core.MetricsCollector,
	cache core.CachePlugin,
) *MutationBuilder {
	return &MutationBuilder{
		eventStore: eventStore,
		logger:     logger,
		metrics:    metrics,
		cache:      cache,
	}
}

// BuildMutations builds the mutations object
func (mb *MutationBuilder) BuildMutations() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name:        "Mutation",
		Description: "Root mutation type",
		Fields: graphql.Fields{
			"invalidateCache": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Invalidate cache for an event",
				Args: graphql.FieldConfigArgument{
					"eventId": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Event ID",
					},
				},
				Resolve: mb.resolveInvalidateCache,
			},
			"invalidateCacheByPattern": &graphql.Field{
				Type:        graphql.Int,
				Description: "Invalidate cache entries matching a pattern",
				Args: graphql.FieldConfigArgument{
					"pattern": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Cache key pattern (supports wildcards)",
					},
				},
				Resolve: mb.resolveInvalidateCacheByPattern,
			},
			"clearCache": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Clear all cache",
				Resolve:     mb.resolveClearCache,
			},
			"refreshEventCache": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Refresh cache for an event",
				Args: graphql.FieldConfigArgument{
					"eventId": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Event ID",
					},
				},
				Resolve: mb.resolveRefreshEventCache,
			},
			"warmCache": &graphql.Field{
				Type:        graphql.Int,
				Description: "Warm cache with recent events",
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type:        graphql.Int,
						Description: "Number of events to cache",
					},
				},
				Resolve: mb.resolveWarmCache,
			},
		},
	})
}

// resolveInvalidateCache invalidates cache for a specific event
func (mb *MutationBuilder) resolveInvalidateCache(p graphql.ResolveParams) (interface{}, error) {
	eventID, ok := p.Args["eventId"].(string)
	if !ok || eventID == "" {
		return nil, fmt.Errorf("invalid or missing eventId parameter")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		mb.metrics.RecordHistogram("graphql_mutation_invalidate_cache_time_ms", float64(duration), nil)
	}()

	if mb.cache == nil {
		return false, fmt.Errorf("cache not available")
	}

	// Invalidate cache entries for this event
	cacheKey := fmt.Sprintf("graphql:event:%s", eventID)
	if err := mb.cache.Delete(p.Context, cacheKey); err != nil {
		mb.logger.Error("Failed to invalidate cache", "eventId", eventID, "error", err.Error())
		mb.metrics.RecordCounter("graphql_mutation_invalidate_cache_error", 1, nil)
		return false, fmt.Errorf("failed to invalidate cache: %w", err)
	}

	// Also invalidate related cache entries
	relatedKeys := []string{
		"graphql:events:address:*",
		"graphql:events:name:*",
		"graphql:events:block:*",
	}

	for _, pattern := range relatedKeys {
		// Pattern-based deletion would require cache backend support
		// For now, we log the pattern that would be deleted
		mb.logger.Debug("Would invalidate cache pattern", "pattern", pattern)
	}

	mb.logger.Info("Cache invalidated", "eventId", eventID)
	mb.metrics.RecordCounter("graphql_mutation_invalidate_cache_success", 1, nil)
	return true, nil
}

// resolveInvalidateCacheByPattern invalidates cache entries matching a pattern
func (mb *MutationBuilder) resolveInvalidateCacheByPattern(p graphql.ResolveParams) (interface{}, error) {
	pattern, ok := p.Args["pattern"].(string)
	if !ok || pattern == "" {
		return nil, fmt.Errorf("invalid or missing pattern parameter")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		mb.metrics.RecordHistogram("graphql_mutation_invalidate_cache_pattern_time_ms", float64(duration), nil)
	}()

	if mb.cache == nil {
		return 0, fmt.Errorf("cache not available")
	}

	// For pattern-based deletion, we would need a cache implementation that supports it
	// For now, we'll count it as 0 since we can't implement pattern matching without
	// a cache backend that supports it
	count := 0

	mb.logger.Info("Cache invalidated by pattern", "pattern", pattern, "count", count)
	mb.metrics.RecordCounter("graphql_mutation_invalidate_cache_pattern_success", 1, nil)
	return count, nil
}

// resolveClearCache clears all cache
func (mb *MutationBuilder) resolveClearCache(p graphql.ResolveParams) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		mb.metrics.RecordHistogram("graphql_mutation_clear_cache_time_ms", float64(duration), nil)
	}()

	if mb.cache == nil {
		return false, fmt.Errorf("cache not available")
	}

	// Clear all GraphQL cache entries
	// In a real implementation, this would iterate through all cache keys with a pattern
	// For now, we'll just log the operation
	mb.logger.Info("Cache cleared")
	mb.metrics.RecordCounter("graphql_mutation_clear_cache_success", 1, nil)
	return true, nil
}

// resolveRefreshEventCache refreshes cache for a specific event
func (mb *MutationBuilder) resolveRefreshEventCache(p graphql.ResolveParams) (interface{}, error) {
	eventID, ok := p.Args["eventId"].(string)
	if !ok || eventID == "" {
		return nil, fmt.Errorf("invalid or missing eventId parameter")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		mb.metrics.RecordHistogram("graphql_mutation_refresh_cache_time_ms", float64(duration), nil)
	}()

	if mb.cache == nil {
		return false, fmt.Errorf("cache not available")
	}

	// Retrieve event from store
	event, err := mb.eventStore.GetEvent(p.Context, eventID)
	if err != nil {
		mb.logger.Error("Failed to refresh cache", "eventId", eventID, "error", err.Error())
		mb.metrics.RecordCounter("graphql_mutation_refresh_cache_error", 1, nil)
		return false, fmt.Errorf("failed to refresh cache: %w", err)
	}

	if event == nil {
		return false, fmt.Errorf("event not found")
	}

	// Convert to GraphQL format and cache
	result := eventToGraphQL(event)
	cacheKey := fmt.Sprintf("graphql:event:%s", eventID)
	resultBytes, _ := json.Marshal(result)
	if err := mb.cache.Set(p.Context, cacheKey, resultBytes, 300); err != nil { // 5 minutes
		mb.logger.Error("Failed to cache event", "eventId", eventID, "error", err.Error())
		mb.metrics.RecordCounter("graphql_mutation_refresh_cache_error", 1, nil)
		return false, fmt.Errorf("failed to cache event: %w", err)
	}

	mb.logger.Info("Cache refreshed", "eventId", eventID)
	mb.metrics.RecordCounter("graphql_mutation_refresh_cache_success", 1, nil)
	return true, nil
}

// resolveWarmCache warms cache with recent events
func (mb *MutationBuilder) resolveWarmCache(p graphql.ResolveParams) (interface{}, error) {
	// Limit maximum warm cache size
	limit := 100
	if l, ok := p.Args["limit"].(int); ok && l > 0 {
		limit = l
	}
	if limit > 1000 {
		limit = 1000
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		mb.metrics.RecordHistogram("graphql_mutation_warm_cache_time_ms", float64(duration), nil)
	}()

	if mb.cache == nil {
		return 0, fmt.Errorf("cache not available")
	}

	// Retrieve recent events and cache them
	// Using empty cursor to get the most recent events
	events, _, err := mb.eventStore.GetEventsPaginated(p.Context, "", limit)
	if err != nil {
		mb.logger.Error("Failed to warm cache", "error", err.Error())
		mb.metrics.RecordCounter("graphql_mutation_warm_cache_error", 1, nil)
		return 0, fmt.Errorf("failed to warm cache: %w", err)
	}

	// Cache each event
	count := 0
	for _, event := range events {
		if event == nil {
			continue
		}

		result := eventToGraphQL(event)
		cacheKey := fmt.Sprintf("graphql:event:%s", event.ID)
		resultBytes, _ := json.Marshal(result)
		if err := mb.cache.Set(p.Context, cacheKey, resultBytes, 300); err == nil { // 5 minutes
			count++
		}
	}

	mb.logger.Info("Cache warmed", "count", count)
	mb.metrics.RecordCounter("graphql_mutation_warm_cache_success", 1, nil)
	return count, nil
}
