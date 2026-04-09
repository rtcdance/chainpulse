package observability

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// IndexerMetrics tracks metrics for the Web3 indexer
type IndexerMetrics struct {
	mu sync.RWMutex

	// Block tracking
	CurrentBlockNumber uint64
	LatestBlockNumber  uint64
	IndexingLag        uint64

	// Event counters
	EventsIndexed   int64
	EventsProcessed int64
	EventsFailed    int64

	// Timing metrics
	IndexingLatencies []time.Duration
	QueryLatencies    []time.Duration

	// Reorg tracking
	ReorgsDetected   int64
	BlocksRolledBack int64
	LastReorgTime    time.Time

	// Cache metrics
	CacheHits   int64
	CacheMisses int64

	// DLQ metrics
	DLQDepth int64

	// Consistency metrics
	ConsistencyMismatches int64

	// Recovery metrics
	ReorgRecoveryTimeMs int64

	// Error tracking
	ErrorCount map[string]int64

	// Timestamps
	StartTime      time.Time
	LastUpdateTime time.Time
}

// NewIndexerMetrics creates a new IndexerMetrics instance
func NewIndexerMetrics() *IndexerMetrics {
	return &IndexerMetrics{
		IndexingLatencies: make([]time.Duration, 0),
		QueryLatencies:    make([]time.Duration, 0),
		ErrorCount:        make(map[string]int64),
		StartTime:         time.Now(),
		LastUpdateTime:    time.Now(),
	}
}

// RecordIndexingProgress records the current indexing progress
func (im *IndexerMetrics) RecordIndexingProgress(currentBlock, latestBlock uint64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.CurrentBlockNumber = currentBlock
	im.LatestBlockNumber = latestBlock

	if latestBlock > currentBlock {
		im.IndexingLag = latestBlock - currentBlock
	} else {
		im.IndexingLag = 0
	}

	im.LastUpdateTime = time.Now()
}

// RecordEventIndexed records that an event was indexed
func (im *IndexerMetrics) RecordEventIndexed(latency time.Duration) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.EventsIndexed++
	im.IndexingLatencies = append(im.IndexingLatencies, latency)

	// Keep only last 1000 latencies to avoid memory bloat
	if len(im.IndexingLatencies) > 1000 {
		im.IndexingLatencies = im.IndexingLatencies[1:]
	}

	im.LastUpdateTime = time.Now()
}

// RecordEventProcessed records that an event was processed
func (im *IndexerMetrics) RecordEventProcessed() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.EventsProcessed++
	im.LastUpdateTime = time.Now()
}

// RecordEventFailed records that an event processing failed
func (im *IndexerMetrics) RecordEventFailed(errorType string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.EventsFailed++
	im.ErrorCount[errorType]++
	im.LastUpdateTime = time.Now()
}

// RecordQueryLatency records the latency of a query
func (im *IndexerMetrics) RecordQueryLatency(latency time.Duration) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.QueryLatencies = append(im.QueryLatencies, latency)

	// Keep only last 1000 latencies to avoid memory bloat
	if len(im.QueryLatencies) > 1000 {
		im.QueryLatencies = im.QueryLatencies[1:]
	}

	im.LastUpdateTime = time.Now()
}

// RecordReorg records that a reorg was detected and handled
func (im *IndexerMetrics) RecordReorg(blocksRolledBack uint64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.ReorgsDetected++
	im.BlocksRolledBack += saturatingUint64ToInt64(blocksRolledBack)
	im.LastReorgTime = time.Now()
	im.LastUpdateTime = time.Now()
}

func saturatingUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

// RecordCacheHit records a cache hit
func (im *IndexerMetrics) RecordCacheHit() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.CacheHits++
	im.LastUpdateTime = time.Now()
}

// RecordCacheMiss records a cache miss
func (im *IndexerMetrics) RecordCacheMiss() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.CacheMisses++
	im.LastUpdateTime = time.Now()
}

// RecordDLQDepth records the current DLQ depth
func (im *IndexerMetrics) RecordDLQDepth(depth int64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.DLQDepth = depth
	im.LastUpdateTime = time.Now()
}

// RecordConsistencyMismatch records a consistency check mismatch
func (im *IndexerMetrics) RecordConsistencyMismatch() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.ConsistencyMismatches++
	im.LastUpdateTime = time.Now()
}

// RecordReorgRecoveryTime records the time taken to recover from a reorg
func (im *IndexerMetrics) RecordReorgRecoveryTime(recoveryTimeMs int64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.ReorgRecoveryTimeMs = recoveryTimeMs
	im.LastUpdateTime = time.Now()
}

// GetAverageIndexingLatency returns the average indexing latency
func (im *IndexerMetrics) GetAverageIndexingLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if len(im.IndexingLatencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range im.IndexingLatencies {
		total += latency
	}

	return total / time.Duration(len(im.IndexingLatencies))
}

// GetMaxIndexingLatency returns the maximum indexing latency
func (im *IndexerMetrics) GetMaxIndexingLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if len(im.IndexingLatencies) == 0 {
		return 0
	}

	var max time.Duration
	for _, latency := range im.IndexingLatencies {
		if latency > max {
			max = latency
		}
	}

	return max
}

// GetAverageQueryLatency returns the average query latency
func (im *IndexerMetrics) GetAverageQueryLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if len(im.QueryLatencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range im.QueryLatencies {
		total += latency
	}

	return total / time.Duration(len(im.QueryLatencies))
}

// GetMaxQueryLatency returns the maximum query latency
func (im *IndexerMetrics) GetMaxQueryLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if len(im.QueryLatencies) == 0 {
		return 0
	}

	var max time.Duration
	for _, latency := range im.QueryLatencies {
		if latency > max {
			max = latency
		}
	}

	return max
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (im *IndexerMetrics) GetCacheHitRate() float64 {
	im.mu.RLock()
	defer im.mu.RUnlock()

	total := im.CacheHits + im.CacheMisses
	if total == 0 {
		return 0
	}

	return float64(im.CacheHits) / float64(total) * 100
}

// GetIndexingRate returns the number of events indexed per second
func (im *IndexerMetrics) GetIndexingRate() float64 {
	im.mu.RLock()
	defer im.mu.RUnlock()

	elapsed := time.Since(im.StartTime).Seconds()
	if elapsed == 0 {
		return 0
	}

	return float64(im.EventsIndexed) / elapsed
}

// GetErrorRate returns the error rate as a percentage
func (im *IndexerMetrics) GetErrorRate() float64 {
	im.mu.RLock()
	defer im.mu.RUnlock()

	total := im.EventsProcessed + im.EventsFailed
	if total == 0 {
		return 0
	}

	return float64(im.EventsFailed) / float64(total) * 100
}

// GetMetricsSummary returns a summary of all metrics
func (im *IndexerMetrics) GetMetricsSummary() map[string]interface{} {
	im.mu.RLock()
	defer im.mu.RUnlock()

	errorSummary := make(map[string]int64)
	for errorType, count := range im.ErrorCount {
		errorSummary[errorType] = count
	}

	// Calculate averages and rates directly without calling other methods
	// to avoid potential lock contention
	avgIndexingLatency := time.Duration(0)
	if len(im.IndexingLatencies) > 0 {
		var total time.Duration
		for _, latency := range im.IndexingLatencies {
			total += latency
		}
		avgIndexingLatency = total / time.Duration(len(im.IndexingLatencies))
	}

	maxIndexingLatency := time.Duration(0)
	for _, latency := range im.IndexingLatencies {
		if latency > maxIndexingLatency {
			maxIndexingLatency = latency
		}
	}

	avgQueryLatency := time.Duration(0)
	if len(im.QueryLatencies) > 0 {
		var total time.Duration
		for _, latency := range im.QueryLatencies {
			total += latency
		}
		avgQueryLatency = total / time.Duration(len(im.QueryLatencies))
	}

	maxQueryLatency := time.Duration(0)
	for _, latency := range im.QueryLatencies {
		if latency > maxQueryLatency {
			maxQueryLatency = latency
		}
	}

	cacheHitRate := 0.0
	if im.CacheHits+im.CacheMisses > 0 {
		cacheHitRate = float64(im.CacheHits) / float64(im.CacheHits+im.CacheMisses) * 100
	}

	indexingRate := 0.0
	uptime := time.Since(im.StartTime).Seconds()
	if uptime > 0 {
		indexingRate = float64(im.EventsIndexed) / uptime
	}

	errorRate := 0.0
	if im.EventsIndexed > 0 {
		errorRate = float64(im.EventsFailed) / float64(im.EventsIndexed) * 100
	}

	return map[string]interface{}{
		"current_block":            im.CurrentBlockNumber,
		"latest_block":             im.LatestBlockNumber,
		"indexing_lag":             im.IndexingLag,
		"events_indexed":           im.EventsIndexed,
		"events_processed":         im.EventsProcessed,
		"events_failed":            im.EventsFailed,
		"average_indexing_latency": avgIndexingLatency.String(),
		"max_indexing_latency":     maxIndexingLatency.String(),
		"average_query_latency":    avgQueryLatency.String(),
		"max_query_latency":        maxQueryLatency.String(),
		"cache_hits":               im.CacheHits,
		"cache_misses":             im.CacheMisses,
		"cache_hit_rate":           fmt.Sprintf("%.2f%%", cacheHitRate),
		"indexing_rate":            fmt.Sprintf("%.2f events/sec", indexingRate),
		"error_rate":               fmt.Sprintf("%.2f%%", errorRate),
		"reorgs_detected":          im.ReorgsDetected,
		"blocks_rolled_back":       im.BlocksRolledBack,
		"last_reorg_time":          im.LastReorgTime.String(),
		"dlq_depth":                im.DLQDepth,
		"consistency_mismatches":   im.ConsistencyMismatches,
		"reorg_recovery_time_ms":   im.ReorgRecoveryTimeMs,
		"uptime":                   time.Since(im.StartTime).String(),
		"last_update_time":         im.LastUpdateTime.String(),
		"error_breakdown":          errorSummary,
	}
}

// Reset resets all metrics
func (im *IndexerMetrics) Reset() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.CurrentBlockNumber = 0
	im.LatestBlockNumber = 0
	im.IndexingLag = 0
	im.EventsIndexed = 0
	im.EventsProcessed = 0
	im.EventsFailed = 0
	im.IndexingLatencies = make([]time.Duration, 0)
	im.QueryLatencies = make([]time.Duration, 0)
	im.ReorgsDetected = 0
	im.BlocksRolledBack = 0
	im.CacheHits = 0
	im.CacheMisses = 0
	im.ErrorCount = make(map[string]int64)
	im.StartTime = time.Now()
	im.LastUpdateTime = time.Now()
}
