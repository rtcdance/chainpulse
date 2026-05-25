package query

import (
	"time"

	"github.com/rtcdance/chainpulse/pkg/services/query/qindex"
)

// Backward-compatible shims for types moved to query/qindex.

type (
	IndexManager             = qindex.Manager
	IndexInfo                = qindex.IndexInfo
	IndexStatistics          = qindex.Statistics
	PendingIndex             = qindex.PendingIndex
	QueryOptimizer           = qindex.Optimizer
	QueryStatistics          = qindex.QueryStats
	IndexRecommendation      = qindex.IndexRecommendation
	OptimizedQuery           = qindex.OptimizedQuery
	QueryStatisticsCollector = qindex.StatsCollector
	QueryMetrics             = qindex.QueryMetrics
	AggregatedMetrics        = qindex.AggregatedMetrics
)

func NewIndexManager() *IndexManager {
	return qindex.NewManager()
}

func NewQueryOptimizer(maxCacheSize int) *QueryOptimizer {
	return qindex.NewOptimizer(maxCacheSize)
}

func NewQueryStatisticsCollector(aggregationWindow time.Duration) *QueryStatisticsCollector {
	return qindex.NewStatsCollector(aggregationWindow)
}
