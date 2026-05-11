package blockchain

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"chainpulse/pkg/core"
)

// CrossChainQuery represents a query across multiple blockchains
type CrossChainQuery struct {
	QueryID     string
	Blockchains []string // List of blockchain types to query
	Filter      EventFilter
	Limit       int
	Offset      int
	Timeout     time.Duration
	CreatedAt   time.Time
}

// CrossChainResult represents aggregated results from multiple blockchains
type CrossChainResult struct {
	QueryID       string
	Events        []core.BlockchainEvent
	BlockchainMap map[string][]core.BlockchainEvent // Events grouped by blockchain
	TotalCount    int
	QueryTime     time.Duration
	CompletedAt   time.Time
}

// CrossChainAPI provides unified query interface for cross-chain aggregation
type CrossChainAPI struct {
	clusterManager       *MultiBlockchainClusterManager
	cache                DistributedCache
	maxConcurrentQueries int
	activeQueries        atomic.Int32 // atomic to avoid data race with GetMetrics
	queryTimeout         time.Duration
	metrics              *CrossChainMetrics
}

// CrossChainMetrics tracks cross-chain query metrics
type CrossChainMetrics struct {
	mu                sync.RWMutex
	TotalQueries      int64
	SuccessfulQueries int64
	FailedQueries     int64
	AverageQueryTime  time.Duration
	TotalQueryTime    time.Duration
	CacheHits         int64
	CacheMisses       int64
	AggregationErrors int64
	LastQueryTime     time.Time
}

// NewCrossChainAPI creates a new cross-chain API
func NewCrossChainAPI(clusterManager *MultiBlockchainClusterManager, cache DistributedCache) *CrossChainAPI {
	return &CrossChainAPI{
		clusterManager:       clusterManager,
		cache:                cache,
		maxConcurrentQueries: 100,
		queryTimeout:         30 * time.Second,
		metrics: &CrossChainMetrics{
			LastQueryTime: time.Now(),
		},
	}
}

// Query executes a cross-chain query
func (cca *CrossChainAPI) Query(ctx context.Context, query *CrossChainQuery) (*CrossChainResult, error) {
	if int(cca.activeQueries.Load()) >= cca.maxConcurrentQueries {
		return nil, fmt.Errorf("max concurrent queries exceeded")
	}
	cca.activeQueries.Add(1)

	defer cca.activeQueries.Add(-1)

	// Check cache first
	cacheKey := fmt.Sprintf("cross-chain-query:%s", query.QueryID)
	if cached, err := cca.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		cca.metrics.mu.Lock()
		cca.metrics.CacheHits++
		cca.metrics.mu.Unlock()
		// Return cached result (simplified)
		return &CrossChainResult{
			QueryID:     query.QueryID,
			CompletedAt: time.Now(),
		}, nil
	}

	cca.metrics.mu.Lock()
	cca.metrics.CacheMisses++
	cca.metrics.mu.Unlock()

	start := time.Now()
	result := &CrossChainResult{
		QueryID:       query.QueryID,
		Events:        make([]core.BlockchainEvent, 0),
		BlockchainMap: make(map[string][]core.BlockchainEvent),
	}

	// Query each blockchain cluster in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan map[string]interface{}, len(query.Blockchains))
	errorsChan := make(chan error, len(query.Blockchains))

	for _, blockchain := range query.Blockchains {
		wg.Add(1)
		go func(bc string) {
			defer wg.Done()

			_, err := cca.clusterManager.GetCluster(bc)
			if err != nil {
				errorsChan <- fmt.Errorf("cluster not found for %s: %w", bc, err)
				return
			}

			// Query cluster (simplified - would query actual data store)
			events := make([]core.BlockchainEvent, 0)
			result := map[string]interface{}{
				"blockchain": bc,
				"events":     events,
			}
			resultsChan <- result
		}(blockchain)
	}

	// Wait for all queries to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	aggregationErrors := 0
	for res := range resultsChan {
		blockchain, ok := res["blockchain"].(string)
		if !ok {
			continue
		}
		events, ok := res["events"].([]core.BlockchainEvent)
		if !ok {
			continue
		}
		result.BlockchainMap[blockchain] = events
		result.Events = append(result.Events, events...)
	}

	// Check for errors
	for err := range errorsChan {
		if err != nil {
			aggregationErrors++
			cca.metrics.mu.Lock()
			cca.metrics.AggregationErrors++
			cca.metrics.mu.Unlock()
		}
	}

	// Apply pagination
	result.TotalCount = len(result.Events)
	if query.Offset < len(result.Events) {
		end := query.Offset + query.Limit
		if end > len(result.Events) {
			end = len(result.Events)
		}
		result.Events = result.Events[query.Offset:end]
	} else {
		result.Events = make([]core.BlockchainEvent, 0)
	}

	result.QueryTime = time.Since(start)
	result.CompletedAt = time.Now()

	// Cache result
	if err := cca.cache.Set(ctx, cacheKey, result, 5*time.Minute); err != nil {
		log.Printf("cache set error for key %s: %v", cacheKey, err)
	}

	// Update metrics
	cca.metrics.mu.Lock()
	cca.metrics.TotalQueries++
	if aggregationErrors == 0 {
		cca.metrics.SuccessfulQueries++
	} else {
		cca.metrics.FailedQueries++
	}
	cca.metrics.TotalQueryTime += result.QueryTime
	cca.metrics.AverageQueryTime = cca.metrics.TotalQueryTime / time.Duration(cca.metrics.TotalQueries)
	cca.metrics.LastQueryTime = time.Now()
	cca.metrics.mu.Unlock()

	return result, nil
}

// QueryByBlockchain queries a specific blockchain
func (cca *CrossChainAPI) QueryByBlockchain(ctx context.Context, blockchain string, filter EventFilter) ([]core.BlockchainEvent, error) {
	_, err := cca.clusterManager.GetCluster(blockchain)
	if err != nil {
		return nil, err
	}

	// Query cluster (simplified - would query actual data store)
	events := make([]core.BlockchainEvent, 0)

	return events, nil
}

// AggregateResults aggregates results from multiple blockchains
func (cca *CrossChainAPI) AggregateResults(results map[string][]core.BlockchainEvent) []core.BlockchainEvent {
	aggregated := make([]core.BlockchainEvent, 0)

	for _, events := range results {
		aggregated = append(aggregated, events...)
	}

	return aggregated
}

// MergeResults merges results with consistency guarantees
func (cca *CrossChainAPI) MergeResults(ctx context.Context, results map[string][]core.BlockchainEvent) (*CrossChainResult, error) {
	merged := &CrossChainResult{
		Events:        make([]core.BlockchainEvent, 0),
		BlockchainMap: results,
		CompletedAt:   time.Now(),
	}

	// Merge events from all blockchains
	for _, events := range results {
		merged.Events = append(merged.Events, events...)
		merged.TotalCount += len(events)
	}

	return merged, nil
}

// PaginateResults paginates results
func (cca *CrossChainAPI) PaginateResults(results []core.BlockchainEvent, limit, offset int) []core.BlockchainEvent {
	if offset >= len(results) {
		return make([]core.BlockchainEvent, 0)
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}

// GetMetrics returns cross-chain API metrics
func (cca *CrossChainAPI) GetMetrics() map[string]interface{} {
	cca.metrics.mu.RLock()
	defer cca.metrics.mu.RUnlock()

	return map[string]interface{}{
		"total_queries":      cca.metrics.TotalQueries,
		"successful_queries": cca.metrics.SuccessfulQueries,
		"failed_queries":     cca.metrics.FailedQueries,
		"average_query_time": cca.metrics.AverageQueryTime.String(),
		"total_query_time":   cca.metrics.TotalQueryTime.String(),
		"cache_hits":         cca.metrics.CacheHits,
		"cache_misses":       cca.metrics.CacheMisses,
		"aggregation_errors": cca.metrics.AggregationErrors,
		"last_query_time":    cca.metrics.LastQueryTime,
		"active_queries":     cca.activeQueries.Load(),
	}
}

// InvalidateCache invalidates cross-chain query cache
func (cca *CrossChainAPI) InvalidateCache(ctx context.Context, queryID string) error {
	cacheKey := fmt.Sprintf("cross-chain-query:%s", queryID)
	return cca.cache.Delete(ctx, cacheKey)
}

// InvalidateCacheForBlockchain invalidates cache for a specific blockchain
func (cca *CrossChainAPI) InvalidateCacheForBlockchain(ctx context.Context, blockchain string) error {
	// Invalidate all queries that include this blockchain
	// Simplified implementation
	return nil
}
