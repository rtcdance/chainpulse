// Package qindex provides index management, query optimization, and statistics collection.
package qindex

import (
	"sync"
	"time"
)

type StatsCollector struct {
	mu                sync.RWMutex
	stats             map[string]*QueryMetrics
	aggregatedStats   *AggregatedMetrics
	lastAggregation   time.Time
	aggregationWindow time.Duration
}

type QueryMetrics struct {
	QueryHash       string
	Query           string
	ExecutionCount  int64
	SuccessCount    int64
	ErrorCount      int64
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	AverageDuration time.Duration
	P50Duration     time.Duration
	P95Duration     time.Duration
	P99Duration     time.Duration
	RowsReturned    int64
	RowsScanned     int64
	CacheHits       int64
	CacheMisses     int64
	IndexesUsed     []string
	LastExecuted    time.Time
	FirstExecuted   time.Time
}

type AggregatedMetrics struct {
	TotalQueries        int64
	TotalExecutions     int64
	TotalDuration       time.Duration
	AverageDuration     time.Duration
	MinDuration         time.Duration
	MaxDuration         time.Duration
	P50Duration         time.Duration
	P95Duration         time.Duration
	P99Duration         time.Duration
	TotalRowsReturned   int64
	TotalRowsScanned    int64
	CacheHitRate        float64
	CacheMissRate       float64
}

func NewStatsCollector(aggregationWindow time.Duration) *StatsCollector {
	if aggregationWindow <= 0 {
		aggregationWindow = 5 * time.Minute
	}
	return &StatsCollector{
		stats:             make(map[string]*QueryMetrics),
		aggregatedStats:   &AggregatedMetrics{},
		lastAggregation:   time.Now(),
		aggregationWindow: aggregationWindow,
	}
}