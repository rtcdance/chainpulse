package observability

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
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
	IndexingLatencies *latencyRing
	QueryLatencies    *latencyRing

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

	// RPC latency percentiles
	RPCLatencyP50 time.Duration
	RPCLatencyP95 time.Duration
	RPCLatencyP99 time.Duration

	// Queue depth tracking
	EventQueueDepth int64
	ProcessingDepth int64

	// Block delay: difference between block timestamp and processing time
	BlockDelayLatencies *latencyRing

	// Timestamps
	StartTime      time.Time
	LastUpdateTime time.Time
}

// NewIndexerMetrics creates a new IndexerMetrics instance
func NewIndexerMetrics() *IndexerMetrics {
	return &IndexerMetrics{
		IndexingLatencies: newLatencyRing(1000),
		QueryLatencies:    newLatencyRing(1000),
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
	im.IndexingLatencies.Push(latency)

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

	im.QueryLatencies.Push(latency)

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

// RecordRPCLatency records an RPC call latency and updates percentiles
func (im *IndexerMetrics) RecordRPCLatency(latency time.Duration) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.IndexingLatencies.Push(latency)

	// Update percentiles from the ring buffer
	im.RPCLatencyP50 = im.IndexingLatencies.Percentile(0.50)
	im.RPCLatencyP95 = im.IndexingLatencies.Percentile(0.95)
	im.RPCLatencyP99 = im.IndexingLatencies.Percentile(0.99)

	im.LastUpdateTime = time.Now()
}

// RecordBlockDelay records the delay between block creation and processing
func (im *IndexerMetrics) RecordBlockDelay(delay time.Duration) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.BlockDelayLatencies == nil {
		im.BlockDelayLatencies = newLatencyRing(1000)
	}
	im.BlockDelayLatencies.Push(delay)
	im.LastUpdateTime = time.Now()
}

// RecordEventQueueDepth records the current event queue depth
func (im *IndexerMetrics) RecordEventQueueDepth(depth int64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.EventQueueDepth = depth
	im.LastUpdateTime = time.Now()
}

// RecordProcessingDepth records the current processing depth
func (im *IndexerMetrics) RecordProcessingDepth(depth int64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.ProcessingDepth = depth
	im.LastUpdateTime = time.Now()
}

// GetAverageIndexingLatency returns the average indexing latency
func (im *IndexerMetrics) GetAverageIndexingLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.IndexingLatencies.Avg()
}

// GetMaxIndexingLatency returns the maximum indexing latency
func (im *IndexerMetrics) GetMaxIndexingLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.IndexingLatencies.Max()
}

// GetAverageQueryLatency returns the average query latency
func (im *IndexerMetrics) GetAverageQueryLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.QueryLatencies.Avg()
}

// GetMaxQueryLatency returns the maximum query latency
func (im *IndexerMetrics) GetMaxQueryLatency() time.Duration {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.QueryLatencies.Max()
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
func (im *IndexerMetrics) GetMetricsSummary() map[string]any {
	im.mu.RLock()
	defer im.mu.RUnlock()

	errorSummary := make(map[string]int64)
	for errorType, count := range im.ErrorCount {
		errorSummary[errorType] = count
	}

	// Calculate averages and rates directly without calling other methods
	// to avoid potential lock contention
	avgIndexingLatency := im.IndexingLatencies.Avg()
	maxIndexingLatency := im.IndexingLatencies.Max()
	avgQueryLatency := im.QueryLatencies.Avg()
	maxQueryLatency := im.QueryLatencies.Max()

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

	return map[string]any{
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

// SyncToMetricsCollector pushes all business metrics to a MetricsCollector
// so they appear on the /metrics endpoint alongside system metrics.
func (im *IndexerMetrics) SyncToMetricsCollector(mc core.MetricsCollector) {
	if mc == nil {
		return
	}

	im.mu.RLock()
	defer im.mu.RUnlock()

	mc.RecordGauge("chainpulse_indexer_current_block", float64(im.CurrentBlockNumber), nil)
	mc.RecordGauge("chainpulse_indexer_latest_block", float64(im.LatestBlockNumber), nil)
	mc.RecordGauge("chainpulse_indexer_lag", float64(im.IndexingLag), nil)
	mc.RecordCounter("chainpulse_indexer_events_indexed_total", im.EventsIndexed, nil)
	mc.RecordCounter("chainpulse_indexer_events_processed_total", im.EventsProcessed, nil)
	mc.RecordCounter("chainpulse_indexer_events_failed_total", im.EventsFailed, nil)
	mc.RecordGauge("chainpulse_indexer_cache_hits", float64(im.CacheHits), nil)
	mc.RecordGauge("chainpulse_indexer_cache_misses", float64(im.CacheMisses), nil)
	mc.RecordGauge("chainpulse_indexer_dlq_depth", float64(im.DLQDepth), nil)
	mc.RecordGauge("chainpulse_indexer_consistency_mismatches", float64(im.ConsistencyMismatches), nil)
	mc.RecordCounter("chainpulse_indexer_reorgs_detected_total", im.ReorgsDetected, nil)
	mc.RecordCounter("chainpulse_indexer_blocks_rolled_back_total", im.BlocksRolledBack, nil)
	mc.RecordGauge("chainpulse_indexer_reorg_recovery_time_ms", float64(im.ReorgRecoveryTimeMs), nil)

	// Error breakdown by type
	for errorType, count := range im.ErrorCount {
		mc.RecordCounter("chainpulse_indexer_errors_total", count, map[string]string{
			"error_type": errorType,
		})
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
	im.IndexingLatencies.Reset()
	im.QueryLatencies.Reset()
	im.ReorgsDetected = 0
	im.BlocksRolledBack = 0
	im.CacheHits = 0
	im.CacheMisses = 0
	im.ErrorCount = make(map[string]int64)
	im.StartTime = time.Now()
	im.LastUpdateTime = time.Now()
}
