package query

import (
	"sort"
	"sync"
	"time"
)

// QueryStatisticsCollector collects and aggregates query statistics
type QueryStatisticsCollector struct {
	mu                sync.RWMutex
	stats             map[string]*QueryMetrics
	aggregatedStats   *AggregatedMetrics
	lastAggregation   time.Time
	aggregationWindow time.Duration
}

// QueryMetrics represents metrics for a specific query
type QueryMetrics struct {
	QueryHash         string
	Query             string
	ExecutionCount    int64
	SuccessCount      int64
	ErrorCount        int64
	TotalDuration     time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	AverageDuration   time.Duration
	P50Duration       time.Duration
	P95Duration       time.Duration
	P99Duration       time.Duration
	RowsReturned      int64
	RowsScanned       int64
	CacheHits         int64
	CacheMisses       int64
	IndexesUsed       []string
	LastExecuted      time.Time
	FirstExecuted     time.Time
}

// AggregatedMetrics represents aggregated statistics across all queries
type AggregatedMetrics struct {
	TotalQueries      int64
	TotalExecutions   int64
	TotalDuration     time.Duration
	AverageDuration   time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	P50Duration       time.Duration
	P95Duration       time.Duration
	P99Duration       time.Duration
	SuccessRate       float64
	ErrorRate         float64
	CacheHitRate      float64
	AverageRowsReturned int64
	AverageRowsScanned  int64
	TopQueries        []*QueryMetrics
	SlowQueries       []*QueryMetrics
	ErrorQueries      []*QueryMetrics
	LastUpdated       time.Time
}

// ExecutionRecord represents a single query execution
type ExecutionRecord struct {
	QueryHash    string
	Duration     time.Duration
	RowsReturned int64
	RowsScanned  int64
	CacheHit     bool
	Error        error
	IndexesUsed  []string
	Timestamp    time.Time
}

// NewQueryStatisticsCollector creates a new query statistics collector
func NewQueryStatisticsCollector(aggregationWindow time.Duration) *QueryStatisticsCollector {
	return &QueryStatisticsCollector{
		stats:             make(map[string]*QueryMetrics),
		aggregationWindow: aggregationWindow,
		lastAggregation:   time.Now(),
	}
}

// RecordExecution records a query execution
func (qsc *QueryStatisticsCollector) RecordExecution(record ExecutionRecord) {
	qsc.mu.Lock()
	defer qsc.mu.Unlock()

	metrics, exists := qsc.stats[record.QueryHash]
	if !exists {
		metrics = &QueryMetrics{
			QueryHash:    record.QueryHash,
			FirstExecuted: record.Timestamp,
		}
		qsc.stats[record.QueryHash] = metrics
	}

	metrics.ExecutionCount++
	metrics.TotalDuration += record.Duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.ExecutionCount)
	metrics.LastExecuted = record.Timestamp
	metrics.RowsReturned += record.RowsReturned
	metrics.RowsScanned += record.RowsScanned
	metrics.IndexesUsed = record.IndexesUsed

	if record.Duration < metrics.MinDuration || metrics.MinDuration == 0 {
		metrics.MinDuration = record.Duration
	}
	if record.Duration > metrics.MaxDuration {
		metrics.MaxDuration = record.Duration
	}

	if record.CacheHit {
		metrics.CacheHits++
	} else {
		metrics.CacheMisses++
	}

	if record.Error != nil {
		metrics.ErrorCount++
	} else {
		metrics.SuccessCount++
	}
}

// GetQueryMetrics retrieves metrics for a specific query
func (qsc *QueryStatisticsCollector) GetQueryMetrics(queryHash string) *QueryMetrics {
	qsc.mu.RLock()
	defer qsc.mu.RUnlock()

	return qsc.stats[queryHash]
}

// GetAllQueryMetrics retrieves metrics for all queries
func (qsc *QueryStatisticsCollector) GetAllQueryMetrics() []*QueryMetrics {
	qsc.mu.RLock()
	defer qsc.mu.RUnlock()

	metrics := make([]*QueryMetrics, 0, len(qsc.stats))
	for _, m := range qsc.stats {
		metrics = append(metrics, m)
	}

	return metrics
}

// AggregateMetrics aggregates statistics across all queries
func (qsc *QueryStatisticsCollector) AggregateMetrics() *AggregatedMetrics {
	qsc.mu.Lock()
	defer qsc.mu.Unlock()

	if time.Since(qsc.lastAggregation) < qsc.aggregationWindow {
		if qsc.aggregatedStats != nil {
			return qsc.aggregatedStats
		}
	}

	agg := &AggregatedMetrics{
		TotalQueries:    int64(len(qsc.stats)),
		TopQueries:      []*QueryMetrics{},
		SlowQueries:     []*QueryMetrics{},
		ErrorQueries:    []*QueryMetrics{},
		LastUpdated:     time.Now(),
	}

	if agg.TotalQueries == 0 {
		qsc.aggregatedStats = agg
		qsc.lastAggregation = time.Now()
		return agg
	}

	// Collect all metrics
	allMetrics := make([]*QueryMetrics, 0, len(qsc.stats))
	successCount := int64(0)
	errorCount := int64(0)
	cacheHits := int64(0)
	cacheMisses := int64(0)
	rowsReturned := int64(0)
	rowsScanned := int64(0)

	for _, m := range qsc.stats {
		allMetrics = append(allMetrics, m)
		agg.TotalExecutions += m.ExecutionCount
		agg.TotalDuration += m.TotalDuration
		successCount += m.SuccessCount
		errorCount += m.ErrorCount
		cacheHits += m.CacheHits
		cacheMisses += m.CacheMisses
		rowsReturned += m.RowsReturned
		rowsScanned += m.RowsScanned
	}

	// Calculate averages
	if agg.TotalExecutions > 0 {
		agg.AverageDuration = agg.TotalDuration / time.Duration(agg.TotalExecutions)
		agg.SuccessRate = float64(successCount) / float64(agg.TotalExecutions) * 100
		agg.ErrorRate = float64(errorCount) / float64(agg.TotalExecutions) * 100
		agg.AverageRowsReturned = rowsReturned / agg.TotalExecutions
		agg.AverageRowsScanned = rowsScanned / agg.TotalExecutions
	}

	if cacheHits+cacheMisses > 0 {
		agg.CacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	// Find min/max durations
	for _, m := range allMetrics {
		if m.MinDuration > 0 && (agg.MinDuration == 0 || m.MinDuration < agg.MinDuration) {
			agg.MinDuration = m.MinDuration
		}
		if m.MaxDuration > agg.MaxDuration {
			agg.MaxDuration = m.MaxDuration
		}
	}

	// Sort for top and slow queries
	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].ExecutionCount > allMetrics[j].ExecutionCount
	})

	// Top 10 queries by execution count
	topCount := 10
	if len(allMetrics) < topCount {
		topCount = len(allMetrics)
	}
	agg.TopQueries = allMetrics[:topCount]

	// Slow queries (by average duration)
	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].AverageDuration > allMetrics[j].AverageDuration
	})

	slowCount := 10
	if len(allMetrics) < slowCount {
		slowCount = len(allMetrics)
	}
	agg.SlowQueries = allMetrics[:slowCount]

	// Error queries
	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].ErrorCount > allMetrics[j].ErrorCount
	})

	errorQueryCount := 10
	if len(allMetrics) < errorQueryCount {
		errorQueryCount = len(allMetrics)
	}
	for i := 0; i < errorQueryCount; i++ {
		if allMetrics[i].ErrorCount > 0 {
			agg.ErrorQueries = append(agg.ErrorQueries, allMetrics[i])
		}
	}

	qsc.aggregatedStats = agg
	qsc.lastAggregation = time.Now()

	return agg
}

// GetAggregatedMetrics returns the last aggregated metrics
func (qsc *QueryStatisticsCollector) GetAggregatedMetrics() *AggregatedMetrics {
	qsc.mu.RLock()
	defer qsc.mu.RUnlock()

	return qsc.aggregatedStats
}

// ResetStatistics resets all statistics
func (qsc *QueryStatisticsCollector) ResetStatistics() {
	qsc.mu.Lock()
	defer qsc.mu.Unlock()

	qsc.stats = make(map[string]*QueryMetrics)
	qsc.aggregatedStats = nil
	qsc.lastAggregation = time.Now()
}

// GetQueryCount returns the number of unique queries tracked
func (qsc *QueryStatisticsCollector) GetQueryCount() int {
	qsc.mu.RLock()
	defer qsc.mu.RUnlock()

	return len(qsc.stats)
}

// CalculatePercentiles calculates percentile durations for a query
func (qsc *QueryStatisticsCollector) CalculatePercentiles(queryHash string) map[string]time.Duration {
	qsc.mu.RLock()
	metrics, exists := qsc.stats[queryHash]
	qsc.mu.RUnlock()

	if !exists {
		return nil
	}

	// In production, this would track actual execution times
	// For now, estimate based on min/max/average
	percentiles := make(map[string]time.Duration)
	percentiles["p50"] = metrics.AverageDuration
	percentiles["p95"] = metrics.AverageDuration + (metrics.MaxDuration-metrics.MinDuration)/2
	percentiles["p99"] = metrics.MaxDuration

	return percentiles
}

// GetCacheHitRate returns the cache hit rate for a query
func (qsc *QueryStatisticsCollector) GetCacheHitRate(queryHash string) float64 {
	qsc.mu.RLock()
	defer qsc.mu.RUnlock()

	metrics, exists := qsc.stats[queryHash]
	if !exists {
		return 0
	}

	total := metrics.CacheHits + metrics.CacheMisses
	if total == 0 {
		return 0
	}

	return float64(metrics.CacheHits) / float64(total) * 100
}

// GetErrorRate returns the error rate for a query
func (qsc *QueryStatisticsCollector) GetErrorRate(queryHash string) float64 {
	qsc.mu.RLock()
	defer qsc.mu.RUnlock()

	metrics, exists := qsc.stats[queryHash]
	if !exists {
		return 0
	}

	if metrics.ExecutionCount == 0 {
		return 0
	}

	return float64(metrics.ErrorCount) / float64(metrics.ExecutionCount) * 100
}
