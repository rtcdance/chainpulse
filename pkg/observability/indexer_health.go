package observability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the health status of the indexer
type HealthStatus string

const (
	// HealthStatusHealthy represents a healthy status.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded represents a degraded status.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy represents an unhealthy status.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status            HealthStatus
	Message           string
	CurrentBlock      uint64
	LatestBlock       uint64
	IndexingLag       uint64
	MaxAllowedLag     uint64
	DatabaseConnected bool
	CacheConnected    bool
	LastCheckTime     time.Time
	UpTime            time.Duration
	EventsIndexedRate float64
	ErrorRate         float64
	CacheHitRate      float64
	AverageLatency    time.Duration
	MaxLatency        time.Duration
	RecentErrors      map[string]int64
}

// IndexerHealth provides health checking for the indexer
type IndexerHealth struct {
	mu sync.RWMutex

	metrics               *IndexerMetrics
	maxAllowedLag         uint64
	maxErrorRate          float64
	minCacheHitRate       float64
	maxAverageLatency     time.Duration
	databaseCheckFunc     func(ctx context.Context) error
	cacheCheckFunc        func(ctx context.Context) error
	lastHealthCheckTime   time.Time
	lastHealthCheckResult *HealthCheckResult
}

// NewIndexerHealth creates a new IndexerHealth instance
func NewIndexerHealth(
	metrics *IndexerMetrics,
	maxAllowedLag uint64,
	maxErrorRate float64,
	minCacheHitRate float64,
	maxAverageLatency time.Duration,
) *IndexerHealth {
	return &IndexerHealth{
		metrics:             metrics,
		maxAllowedLag:       maxAllowedLag,
		maxErrorRate:        maxErrorRate,
		minCacheHitRate:     minCacheHitRate,
		maxAverageLatency:   maxAverageLatency,
		lastHealthCheckTime: time.Now(),
	}
}

// SetDatabaseCheckFunc sets the function to check database connectivity
func (ih *IndexerHealth) SetDatabaseCheckFunc(fn func(ctx context.Context) error) {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	ih.databaseCheckFunc = fn
}

// SetCacheCheckFunc sets the function to check cache connectivity
func (ih *IndexerHealth) SetCacheCheckFunc(fn func(ctx context.Context) error) {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	ih.cacheCheckFunc = fn
}

// CheckHealth performs a comprehensive health check
func (ih *IndexerHealth) CheckHealth(ctx context.Context) *HealthCheckResult {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	result := &HealthCheckResult{
		LastCheckTime: time.Now(),
		UpTime:        time.Since(ih.metrics.StartTime),
	}

	// Get current metrics
	ih.metrics.mu.RLock()
	result.CurrentBlock = ih.metrics.CurrentBlockNumber
	result.LatestBlock = ih.metrics.LatestBlockNumber
	result.IndexingLag = ih.metrics.IndexingLag
	result.EventsIndexedRate = ih.metrics.GetIndexingRate()
	result.ErrorRate = ih.metrics.GetErrorRate()
	result.CacheHitRate = ih.metrics.GetCacheHitRate()
	result.AverageLatency = ih.metrics.GetAverageIndexingLatency()
	result.MaxLatency = ih.metrics.GetMaxIndexingLatency()
	result.RecentErrors = make(map[string]int64)
	for errorType, count := range ih.metrics.ErrorCount {
		result.RecentErrors[errorType] = count
	}
	ih.metrics.mu.RUnlock()

	result.MaxAllowedLag = ih.maxAllowedLag

	// Check database connectivity
	if ih.databaseCheckFunc != nil {
		err := ih.databaseCheckFunc(ctx)
		result.DatabaseConnected = err == nil
	} else {
		result.DatabaseConnected = true
	}

	// Check cache connectivity
	if ih.cacheCheckFunc != nil {
		err := ih.cacheCheckFunc(ctx)
		result.CacheConnected = err == nil
	} else {
		result.CacheConnected = true
	}

	// Determine overall health status
	result.Status, result.Message = ih.determineHealthStatus(result)

	ih.lastHealthCheckTime = result.LastCheckTime
	ih.lastHealthCheckResult = result

	return result
}

// determineHealthStatus determines the overall health status based on metrics
func (ih *IndexerHealth) determineHealthStatus(result *HealthCheckResult) (HealthStatus, string) {
	issues := make([]string, 0)

	// Check database connectivity
	if !result.DatabaseConnected {
		issues = append(issues, "database disconnected")
	}

	// Check cache connectivity
	if !result.CacheConnected {
		issues = append(issues, "cache disconnected")
	}

	// Check indexing lag
	if result.IndexingLag > ih.maxAllowedLag {
		issues = append(issues, fmt.Sprintf("indexing lag (%d blocks) exceeds maximum (%d blocks)", result.IndexingLag, ih.maxAllowedLag))
	}

	// Check error rate
	if result.ErrorRate > ih.maxErrorRate {
		issues = append(issues, fmt.Sprintf("error rate (%.2f%%) exceeds maximum (%.2f%%)", result.ErrorRate, ih.maxErrorRate))
	}

	// Check cache hit rate
	if result.CacheHitRate < ih.minCacheHitRate && result.CacheHitRate > 0 {
		issues = append(issues, fmt.Sprintf("cache hit rate (%.2f%%) below minimum (%.2f%%)", result.CacheHitRate, ih.minCacheHitRate))
	}

	// Check average latency
	if result.AverageLatency > ih.maxAverageLatency {
		issues = append(issues, fmt.Sprintf("average latency (%v) exceeds maximum (%v)", result.AverageLatency, ih.maxAverageLatency))
	}

	// Determine status
	if len(issues) == 0 {
		return HealthStatusHealthy, "indexer is healthy"
	}

	// If critical issues (database/cache), mark as unhealthy
	if !result.DatabaseConnected || !result.CacheConnected {
		message := "indexer is unhealthy: " + fmt.Sprintf("%v", issues)
		return HealthStatusUnhealthy, message
	}

	// Otherwise, mark as degraded
	message := "indexer is degraded: " + fmt.Sprintf("%v", issues)
	return HealthStatusDegraded, message
}

// GetLastHealthCheck returns the last health check result
func (ih *IndexerHealth) GetLastHealthCheck() *HealthCheckResult {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	return ih.lastHealthCheckResult
}

// IsHealthy returns whether the indexer is healthy
func (ih *IndexerHealth) IsHealthy() bool {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	if ih.lastHealthCheckResult == nil {
		return false
	}

	return ih.lastHealthCheckResult.Status == HealthStatusHealthy
}

// IsDegraded returns whether the indexer is degraded
func (ih *IndexerHealth) IsDegraded() bool {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	if ih.lastHealthCheckResult == nil {
		return false
	}

	return ih.lastHealthCheckResult.Status == HealthStatusDegraded
}

// IsUnhealthy returns whether the indexer is unhealthy
func (ih *IndexerHealth) IsUnhealthy() bool {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	if ih.lastHealthCheckResult == nil {
		return false
	}

	return ih.lastHealthCheckResult.Status == HealthStatusUnhealthy
}

// GetHealthSummary returns a summary of the health status
func (ih *IndexerHealth) GetHealthSummary() map[string]any {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	if ih.lastHealthCheckResult == nil {
		return map[string]any{
			"status":  "unknown",
			"message": "no health check performed yet",
		}
	}

	result := ih.lastHealthCheckResult

	return map[string]any{
		"status":              string(result.Status),
		"message":             result.Message,
		"current_block":       result.CurrentBlock,
		"latest_block":        result.LatestBlock,
		"indexing_lag":        result.IndexingLag,
		"max_allowed_lag":     result.MaxAllowedLag,
		"database_connected":  result.DatabaseConnected,
		"cache_connected":     result.CacheConnected,
		"last_check_time":     result.LastCheckTime.String(),
		"uptime":              result.UpTime.String(),
		"events_indexed_rate": fmt.Sprintf("%.2f events/sec", result.EventsIndexedRate),
		"error_rate":          fmt.Sprintf("%.2f%%", result.ErrorRate),
		"cache_hit_rate":      fmt.Sprintf("%.2f%%", result.CacheHitRate),
		"average_latency":     result.AverageLatency.String(),
		"max_latency":         result.MaxLatency.String(),
		"recent_errors":       result.RecentErrors,
	}
}

// DetectLag detects if the indexing lag is too high
func (ih *IndexerHealth) DetectLag(_ context.Context) (bool, uint64) {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	ih.metrics.mu.RLock()
	lag := ih.metrics.IndexingLag
	ih.metrics.mu.RUnlock()

	return lag > ih.maxAllowedLag, lag
}

// GetLagPercentage returns the lag as a percentage of the maximum allowed lag
func (ih *IndexerHealth) GetLagPercentage() float64 {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	ih.metrics.mu.RLock()
	lag := ih.metrics.IndexingLag
	ih.metrics.mu.RUnlock()

	if ih.maxAllowedLag == 0 {
		return 0
	}

	return float64(lag) / float64(ih.maxAllowedLag) * 100
}
