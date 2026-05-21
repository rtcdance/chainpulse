// Package cache provides caching functionality for the plugin system.
package cache

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
)

// CacheKeyGenerator generates cache keys for different types of queries.
//
// Renaming would break many external uses.
type CacheKeyGenerator struct {
	prefix string
}

// NewCacheKeyGenerator creates a new cache key generator
func NewCacheKeyGenerator(prefix string) *CacheKeyGenerator {
	return &CacheKeyGenerator{
		prefix: prefix,
	}
}

// GenerateEventKey generates a cache key for an event
func (ckg *CacheKeyGenerator) GenerateEventKey(eventID string) string {
	return fmt.Sprintf("%s:event:%s", ckg.prefix, eventID)
}

// GenerateEventsByAddressKey generates a cache key for events by address
func (ckg *CacheKeyGenerator) GenerateEventsByAddressKey(address string, offset, limit int) string {
	return fmt.Sprintf("%s:events:address:%s:offset:%d:limit:%d", ckg.prefix, address, offset, limit)
}

// GenerateEventsByBlockKey generates a cache key for events by block
func (ckg *CacheKeyGenerator) GenerateEventsByBlockKey(blockNumber int64, offset, limit int) string {
	return fmt.Sprintf("%s:events:block:%d:offset:%d:limit:%d", ckg.prefix, blockNumber, offset, limit)
}

// GenerateEventsByTopicKey generates a cache key for events by topic
func (ckg *CacheKeyGenerator) GenerateEventsByTopicKey(topic string, offset, limit int) string {
	return fmt.Sprintf("%s:events:topic:%s:offset:%d:limit:%d", ckg.prefix, topic, offset, limit)
}

// GenerateEventCountKey generates a cache key for event count
func (ckg *CacheKeyGenerator) GenerateEventCountKey(filters map[string]any) string {
	hash := ckg.hashFilters(filters)
	return fmt.Sprintf("%s:event:count:%s", ckg.prefix, hash)
}

// GenerateAggregationKey generates a cache key for aggregation results
func (ckg *CacheKeyGenerator) GenerateAggregationKey(aggregationType string, timeWindow string, filters map[string]any) string {
	hash := ckg.hashFilters(filters)
	return fmt.Sprintf("%s:aggregation:%s:%s:%s", ckg.prefix, aggregationType, timeWindow, hash)
}

// GenerateQueryKey generates a cache key for a query result
func (ckg *CacheKeyGenerator) GenerateQueryKey(queryType string, params map[string]any) string {
	hash := ckg.hashFilters(params)
	return fmt.Sprintf("%s:query:%s:%s", ckg.prefix, queryType, hash)
}

// GenerateGraphQLKey generates a cache key for a GraphQL query
func (ckg *CacheKeyGenerator) GenerateGraphQLKey(query string, variables map[string]any) string {
	queryHash := ckg.hashString(query)
	varsHash := ckg.hashFilters(variables)
	return fmt.Sprintf("%s:graphql:%s:%s", ckg.prefix, queryHash, varsHash)
}

// GenerateSubscriptionKey generates a cache key for a subscription
func (ckg *CacheKeyGenerator) GenerateSubscriptionKey(subscriptionID string) string {
	return fmt.Sprintf("%s:subscription:%s", ckg.prefix, subscriptionID)
}

// GenerateMetadataKey generates a cache key for metadata
func (ckg *CacheKeyGenerator) GenerateMetadataKey(metadataType string, id string) string {
	return fmt.Sprintf("%s:metadata:%s:%s", ckg.prefix, metadataType, id)
}

// GenerateIndexKey generates a cache key for index information
func (ckg *CacheKeyGenerator) GenerateIndexKey(indexName string) string {
	return fmt.Sprintf("%s:index:%s", ckg.prefix, indexName)
}

// GenerateStatsKey generates a cache key for statistics
func (ckg *CacheKeyGenerator) GenerateStatsKey(statsType string) string {
	return fmt.Sprintf("%s:stats:%s", ckg.prefix, statsType)
}

// GenerateHealthKey generates a cache key for health check
func (ckg *CacheKeyGenerator) GenerateHealthKey() string {
	return fmt.Sprintf("%s:health", ckg.prefix)
}

// GenerateConfigKey generates a cache key for configuration
func (ckg *CacheKeyGenerator) GenerateConfigKey(configType string) string {
	return fmt.Sprintf("%s:config:%s", ckg.prefix, configType)
}

// GenerateSessionKey generates a cache key for session data
func (ckg *CacheKeyGenerator) GenerateSessionKey(sessionID string) string {
	return fmt.Sprintf("%s:session:%s", ckg.prefix, sessionID)
}

// GenerateUserKey generates a cache key for user data
func (ckg *CacheKeyGenerator) GenerateUserKey(userID string) string {
	return fmt.Sprintf("%s:user:%s", ckg.prefix, userID)
}

// GeneratePermissionKey generates a cache key for permissions
func (ckg *CacheKeyGenerator) GeneratePermissionKey(userID string, resource string) string {
	return fmt.Sprintf("%s:permission:%s:%s", ckg.prefix, userID, resource)
}

// GenerateRateLimitKey generates a cache key for rate limiting
func (ckg *CacheKeyGenerator) GenerateRateLimitKey(clientID string) string {
	return fmt.Sprintf("%s:ratelimit:%s", ckg.prefix, clientID)
}

// GenerateEventsByTimeRangeKey generates a cache key for events by time range
func (ckg *CacheKeyGenerator) GenerateEventsByTimeRangeKey(startTime, endTime int64, offset, limit int) string {
	return fmt.Sprintf("%s:events:time:%d:%d:offset:%d:limit:%d", ckg.prefix, startTime, endTime, offset, limit)
}

// GenerateEventsByTypeKey generates a cache key for events by type
func (ckg *CacheKeyGenerator) GenerateEventsByTypeKey(eventType string, offset, limit int) string {
	return fmt.Sprintf("%s:events:type:%s:offset:%d:limit:%d", ckg.prefix, eventType, offset, limit)
}

// GenerateEventsByStatusKey generates a cache key for events by status
func (ckg *CacheKeyGenerator) GenerateEventsByStatusKey(status string, offset, limit int) string {
	return fmt.Sprintf("%s:events:status:%s:offset:%d:limit:%d", ckg.prefix, status, offset, limit)
}

// GenerateEventSearchKey generates a cache key for event search results
func (ckg *CacheKeyGenerator) GenerateEventSearchKey(searchQuery string, offset, limit int) string {
	hash := ckg.hashString(searchQuery)
	return fmt.Sprintf("%s:events:search:%s:offset:%d:limit:%d", ckg.prefix, hash, offset, limit)
}

// GenerateRelatedEventsKey generates a cache key for related events
func (ckg *CacheKeyGenerator) GenerateRelatedEventsKey(eventID string) string {
	return fmt.Sprintf("%s:events:related:%s", ckg.prefix, eventID)
}

// GenerateEventChainKey generates a cache key for event chain
func (ckg *CacheKeyGenerator) GenerateEventChainKey(eventID string) string {
	return fmt.Sprintf("%s:events:chain:%s", ckg.prefix, eventID)
}

// GenerateEventHistoryKey generates a cache key for event history
func (ckg *CacheKeyGenerator) GenerateEventHistoryKey(eventID string) string {
	return fmt.Sprintf("%s:events:history:%s", ckg.prefix, eventID)
}

// GenerateEventDependenciesKey generates a cache key for event dependencies
func (ckg *CacheKeyGenerator) GenerateEventDependenciesKey(eventID string) string {
	return fmt.Sprintf("%s:events:dependencies:%s", ckg.prefix, eventID)
}

// Private helper methods

func (ckg *CacheKeyGenerator) hashFilters(filters map[string]any) string {
	if len(filters) == 0 {
		return "empty"
	}

	// Sort keys for consistent hashing
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build string representation
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s:%v:", k, filters[k]))
	}

	return ckg.hashString(sb.String())
}

func (ckg *CacheKeyGenerator) hashString(s string) string {
	hash := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", hash)[:16]
}
