package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// QueryOptimizer analyzes and optimizes database queries
type QueryOptimizer struct {
	mu                sync.RWMutex
	queryStats        map[string]*QueryStatistics
	indexRecommendations map[string][]IndexRecommendation
	cache             map[string]*OptimizedQuery
	maxCacheSize      int
}

// QueryStatistics tracks query performance metrics
type QueryStatistics struct {
	QueryHash        string
	Query            string
	ExecutionCount   int64
	TotalDuration    time.Duration
	AverageDuration  time.Duration
	MinDuration      time.Duration
	MaxDuration      time.Duration
	ErrorCount       int64
	LastExecuted     time.Time
	EstimatedCost    float64
	ActualCost       float64
	IndexesUsed      []string
	FullTableScans   int64
}

// IndexRecommendation suggests an index for optimization
type IndexRecommendation struct {
	TableName       string
	Columns         []string
	Type            string // "BTREE", "HASH", "FULLTEXT"
	Priority        int    // 1-10, higher is more important
	EstimatedGain   float64
	RecommendedAt   time.Time
	ImplementedAt   *time.Time
}

// OptimizedQuery represents an optimized query plan
type OptimizedQuery struct {
	Original         string
	Optimized        string
	QueryHash        string
	EstimatedCost    float64
	RewriteRules     []string
	IndexesUsed      []string
	OptimizedAt      time.Time
	CachedUntil      time.Time
}

// QueryPlan represents the execution plan for a query
type QueryPlan struct {
	Query            string
	Operations       []Operation
	EstimatedCost    float64
	EstimatedRows    int64
	IndexesAvailable []string
	IndexesUsed      []string
	FullTableScan    bool
	Recommendations  []string
}

// Operation represents a single operation in a query plan
type Operation struct {
	Type            string // "Scan", "Filter", "Join", "Sort", "Aggregate"
	Table           string
	Columns         []string
	Condition       string
	EstimatedRows   int64
	EstimatedCost   float64
	IndexUsed       string
	FullTableScan   bool
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(maxCacheSize int) *QueryOptimizer {
	return &QueryOptimizer{
		queryStats:           make(map[string]*QueryStatistics),
		indexRecommendations: make(map[string][]IndexRecommendation),
		cache:                make(map[string]*OptimizedQuery),
		maxCacheSize:         maxCacheSize,
	}
}

// OptimizeQuery analyzes and optimizes a query
func (qo *QueryOptimizer) OptimizeQuery(ctx context.Context, query string) (*OptimizedQuery, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	queryHash := hashQuery(query)

	// Check cache first
	qo.mu.RLock()
	if cached, exists := qo.cache[queryHash]; exists {
		if time.Now().Before(cached.CachedUntil) {
			qo.mu.RUnlock()
			return cached, nil
		}
	}
	qo.mu.RUnlock()

	// Analyze query
	plan, err := qo.AnalyzeQueryPlan(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	// Apply optimization rules
	optimized := qo.applyOptimizationRules(query, plan)

	// Cache the result
	qo.mu.Lock()
	if len(qo.cache) >= qo.maxCacheSize {
		qo.evictOldestCacheEntry()
	}
	qo.cache[queryHash] = optimized
	qo.mu.Unlock()

	return optimized, nil
}

// AnalyzeQueryPlan generates a query execution plan
func (qo *QueryOptimizer) AnalyzeQueryPlan(ctx context.Context, query string) (*QueryPlan, error) {
	plan := &QueryPlan{
		Query:            query,
		Operations:       []Operation{},
		EstimatedCost:    0,
		EstimatedRows:    0,
		IndexesAvailable: []string{},
		IndexesUsed:      []string{},
		FullTableScan:    false,
		Recommendations:  []string{},
	}

	// Parse query to extract operations
	operations := qo.parseQueryOperations(query)
	plan.Operations = operations

	// Estimate cost and rows
	for _, op := range operations {
		plan.EstimatedCost += op.EstimatedCost
		plan.EstimatedRows += op.EstimatedRows

		if op.FullTableScan {
			plan.FullTableScan = true
		}

		if op.IndexUsed != "" {
			plan.IndexesUsed = append(plan.IndexesUsed, op.IndexUsed)
		}
	}

	// Generate recommendations
	plan.Recommendations = qo.generateRecommendations(plan)

	return plan, nil
}

// parseQueryOperations extracts operations from a query
func (qo *QueryOptimizer) parseQueryOperations(query string) []Operation {
	operations := []Operation{}

	// Simple parsing logic - in production, use a proper SQL parser
	upperQuery := strings.ToUpper(query)

	// Detect SELECT operation
	if strings.Contains(upperQuery, "SELECT") {
		op := Operation{
			Type:          "Scan",
			EstimatedRows: 1000, // Default estimate
			EstimatedCost: 100,
			FullTableScan: !strings.Contains(upperQuery, "WHERE"),
		}

		// Extract table name
		if idx := strings.Index(upperQuery, "FROM"); idx != -1 {
			rest := upperQuery[idx+4:]
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				op.Table = strings.TrimSpace(parts[0])
			}
		}

		operations = append(operations, op)
	}

	// Detect WHERE clause (Filter operation)
	if strings.Contains(upperQuery, "WHERE") {
		op := Operation{
			Type:          "Filter",
			EstimatedRows: 100,
			EstimatedCost: 50,
			FullTableScan: false,
		}
		operations = append(operations, op)
	}

	// Detect JOIN operations
	if strings.Contains(upperQuery, "JOIN") {
		op := Operation{
			Type:          "Join",
			EstimatedRows: 500,
			EstimatedCost: 200,
		}
		operations = append(operations, op)
	}

	// Detect ORDER BY (Sort operation)
	if strings.Contains(upperQuery, "ORDER BY") {
		op := Operation{
			Type:          "Sort",
			EstimatedRows: 100,
			EstimatedCost: 150,
		}
		operations = append(operations, op)
	}

	// Detect GROUP BY (Aggregate operation)
	if strings.Contains(upperQuery, "GROUP BY") {
		op := Operation{
			Type:          "Aggregate",
			EstimatedRows: 50,
			EstimatedCost: 100,
		}
		operations = append(operations, op)
	}

	return operations
}

// applyOptimizationRules applies optimization rules to a query
func (qo *QueryOptimizer) applyOptimizationRules(query string, plan *QueryPlan) *OptimizedQuery {
	optimized := &OptimizedQuery{
		Original:      query,
		Optimized:     query,
		QueryHash:     hashQuery(query),
		EstimatedCost: plan.EstimatedCost,
		IndexesUsed:   plan.IndexesUsed,
		OptimizedAt:   time.Now(),
		CachedUntil:   time.Now().Add(1 * time.Hour),
		RewriteRules:  []string{},
	}

	// Rule 1: Push filters down
	if strings.Contains(strings.ToUpper(query), "WHERE") {
		optimized.RewriteRules = append(optimized.RewriteRules, "filter_pushdown")
	}

	// Rule 2: Use indexes for WHERE clauses
	if plan.FullTableScan && len(plan.Operations) > 0 {
		optimized.RewriteRules = append(optimized.RewriteRules, "index_recommendation")
	}

	// Rule 3: Reorder joins
	if strings.Contains(strings.ToUpper(query), "JOIN") {
		optimized.RewriteRules = append(optimized.RewriteRules, "join_reordering")
	}

	// Rule 4: Eliminate redundant operations
	if strings.Count(strings.ToUpper(query), "SELECT") > 1 {
		optimized.RewriteRules = append(optimized.RewriteRules, "redundancy_elimination")
	}

	return optimized
}

// generateRecommendations generates optimization recommendations
func (qo *QueryOptimizer) generateRecommendations(plan *QueryPlan) []string {
	recommendations := []string{}

	// Recommend indexes for full table scans
	if plan.FullTableScan {
		recommendations = append(recommendations, "Add index on WHERE clause columns")
	}

	// Recommend index for ORDER BY
	if len(plan.Operations) > 0 {
		for _, op := range plan.Operations {
			if op.Type == "Sort" && op.IndexUsed == "" {
				recommendations = append(recommendations, "Add index on ORDER BY columns")
				break
			}
		}
	}

	// Recommend covering index
	if len(plan.IndexesUsed) > 0 {
		recommendations = append(recommendations, "Consider covering index for better performance")
	}

	// Recommend query rewrite
	if plan.EstimatedCost > 1000 {
		recommendations = append(recommendations, "Consider rewriting query for better performance")
	}

	return recommendations
}

// RecordQueryExecution records query execution statistics
func (qo *QueryOptimizer) RecordQueryExecution(query string, duration time.Duration, err error, indexesUsed []string) {
	queryHash := hashQuery(query)

	qo.mu.Lock()
	defer qo.mu.Unlock()

	stats, exists := qo.queryStats[queryHash]
	if !exists {
		stats = &QueryStatistics{
			QueryHash: queryHash,
			Query:     query,
		}
		qo.queryStats[queryHash] = stats
	}

	stats.ExecutionCount++
	stats.TotalDuration += duration
	stats.AverageDuration = stats.TotalDuration / time.Duration(stats.ExecutionCount)
	stats.LastExecuted = time.Now()
	stats.IndexesUsed = indexesUsed

	if duration < stats.MinDuration || stats.MinDuration == 0 {
		stats.MinDuration = duration
	}
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}

	if err != nil {
		stats.ErrorCount++
	}
}

// GetQueryStatistics returns statistics for a query
func (qo *QueryOptimizer) GetQueryStatistics(query string) *QueryStatistics {
	queryHash := hashQuery(query)

	qo.mu.RLock()
	defer qo.mu.RUnlock()

	return qo.queryStats[queryHash]
}

// GetAllQueryStatistics returns all query statistics
func (qo *QueryOptimizer) GetAllQueryStatistics() []*QueryStatistics {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	stats := make([]*QueryStatistics, 0, len(qo.queryStats))
	for _, s := range qo.queryStats {
		stats = append(stats, s)
	}

	// Sort by execution count (descending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].ExecutionCount > stats[j].ExecutionCount
	})

	return stats
}

// RecommendIndexes returns index recommendations
func (qo *QueryOptimizer) RecommendIndexes(tableName string) []IndexRecommendation {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	return qo.indexRecommendations[tableName]
}

// AddIndexRecommendation adds an index recommendation
func (qo *QueryOptimizer) AddIndexRecommendation(rec IndexRecommendation) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	qo.indexRecommendations[rec.TableName] = append(qo.indexRecommendations[rec.TableName], rec)
}

// MarkIndexImplemented marks an index as implemented
func (qo *QueryOptimizer) MarkIndexImplemented(tableName string, columns []string) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	recs := qo.indexRecommendations[tableName]
	now := time.Now()
	for i := range recs {
		if columnsEqual(recs[i].Columns, columns) {
			recs[i].ImplementedAt = &now
		}
	}
}

// ClearCache clears the optimization cache
func (qo *QueryOptimizer) ClearCache() {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	qo.cache = make(map[string]*OptimizedQuery)
}

// evictOldestCacheEntry removes the oldest entry from cache
func (qo *QueryOptimizer) evictOldestCacheEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, val := range qo.cache {
		if oldestTime.IsZero() || val.OptimizedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = val.OptimizedAt
		}
	}

	if oldestKey != "" {
		delete(qo.cache, oldestKey)
	}
}

// hashQuery generates a hash for a query
func hashQuery(query string) string {
	// Simple hash - in production, use a proper hash function
	return fmt.Sprintf("%d", len(query))
}

// columnsEqual checks if two column lists are equal
func columnsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
