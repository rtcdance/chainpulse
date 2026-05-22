// Package qindex provides index management, query optimization, and statistics collection.
package qindex

import (
	"sync"
	"time"
)

type Optimizer struct {
	mu                   sync.RWMutex
	queryStats           map[string]*QueryStats
	indexRecommendations map[string][]IndexRecommendation
	cache                map[string]*OptimizedQuery
	maxCacheSize         int
}

type QueryStats struct {
	QueryHash       string
	Query           string
	ExecutionCount  int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	ErrorCount      int64
	LastExecuted    time.Time
	EstimatedCost   float64
	ActualCost      float64
	IndexesUsed     []string
	FullTableScans  int64
}

type IndexRecommendation struct {
	TableName     string
	Columns       []string
	Type          string
	Priority      int
	EstimatedGain float64
	RecommendedAt time.Time
	ImplementedAt *time.Time
}

type OptimizedQuery struct {
	OriginalQuery   string
	OptimizedQuery  string
	Optimizations   []string
	EstimatedGain   float64
	CreatedAt       time.Time
	ExecutionCount  int64
	AverageDuration time.Duration
}

func NewOptimizer(maxCacheSize int) *Optimizer {
	if maxCacheSize <= 0 {
		maxCacheSize = 1000
	}
	return &Optimizer{
		queryStats:           make(map[string]*QueryStats),
		indexRecommendations: make(map[string][]IndexRecommendation),
		cache:                make(map[string]*OptimizedQuery),
		maxCacheSize:         maxCacheSize,
	}
}
