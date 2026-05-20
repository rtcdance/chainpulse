package query

import (
	"time"

	"github.com/rtcdance/chainpulse/pkg/services/query/qindex"
)

// Backward-compatible shims for types moved to query/qindex.

type IndexManager = qindex.Manager
type IndexInfo = qindex.IndexInfo
type IndexStatistics = qindex.Statistics
type PendingIndex = qindex.PendingIndex
type QueryOptimizer = qindex.Optimizer
type QueryStatistics = qindex.QueryStats
type IndexRecommendation = qindex.IndexRecommendation
type OptimizedQuery = qindex.OptimizedQuery
type QueryStatisticsCollector = qindex.StatsCollector
type QueryMetrics = qindex.QueryMetrics
type AggregatedMetrics = qindex.AggregatedMetrics

func NewIndexManager() *IndexManager {
	return qindex.NewManager()
}

func NewQueryOptimizer(maxCacheSize int) *QueryOptimizer {
	return qindex.NewOptimizer(maxCacheSize)
}

func NewQueryStatisticsCollector(aggregationWindow time.Duration) *QueryStatisticsCollector {
	return qindex.NewStatsCollector(aggregationWindow)
}