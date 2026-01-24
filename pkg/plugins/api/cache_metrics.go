package api

import (
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// CacheMetrics collects cache performance metrics
type CacheMetrics struct {
	hitCount      int64
	missCount     int64
	evictionCount int64
	invalidationCount int64
	operationDurations []time.Duration

	logger  core.Logger
	metrics core.MetricsCollector

	mu sync.RWMutex
}

// NewCacheMetrics creates a new cache metrics collector
func NewCacheMetrics(logger core.Logger, metrics core.MetricsCollector) *CacheMetrics {
	return &CacheMetrics{
		hitCount:           0,
		missCount:          0,
		evictionCount:      0,
		invalidationCount:  0,
		operationDurations: make([]time.Duration, 0),
		logger:             logger,
		metrics:            metrics,
	}
}

// RecordHit records a cache hit
func (cm *CacheMetrics) RecordHit(key string, duration time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.hitCount++
	cm.operationDurations = append(cm.operationDurations, duration)

	cm.metrics.RecordCounter("cache_hit", 1, map[string]string{
		"key": key,
	})
	cm.metrics.RecordHistogram("cache_hit_duration_ms", float64(duration.Milliseconds()), map[string]string{
		"key": key,
	})
}

// RecordMiss records a cache miss
func (cm *CacheMetrics) RecordMiss(key string, duration time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.missCount++
	cm.operationDurations = append(cm.operationDurations, duration)

	cm.metrics.RecordCounter("cache_miss", 1, map[string]string{
		"key": key,
	})
	cm.metrics.RecordHistogram("cache_miss_duration_ms", float64(duration.Milliseconds()), map[string]string{
		"key": key,
	})
}

// RecordEviction records a cache eviction
func (cm *CacheMetrics) RecordEviction(key string, reason string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.evictionCount++

	cm.metrics.RecordCounter("cache_eviction", 1, map[string]string{
		"key":    key,
		"reason": reason,
	})
}

// RecordInvalidation records a cache invalidation
func (cm *CacheMetrics) RecordInvalidation(key string, reason string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.invalidationCount++

	cm.metrics.RecordCounter("cache_invalidation", 1, map[string]string{
		"key":    key,
		"reason": reason,
	})
}

// GetHitRate returns the cache hit rate
func (cm *CacheMetrics) GetHitRate() float64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	total := cm.hitCount + cm.missCount
	if total == 0 {
		return 0.0
	}

	return float64(cm.hitCount) / float64(total)
}

// GetStats returns cache statistics
func (cm *CacheMetrics) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	total := cm.hitCount + cm.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(cm.hitCount) / float64(total)
	}

	avgDuration := 0.0
	if len(cm.operationDurations) > 0 {
		var totalDuration time.Duration
		for _, d := range cm.operationDurations {
			totalDuration += d
		}
		avgDuration = float64(totalDuration.Milliseconds()) / float64(len(cm.operationDurations))
	}

	return map[string]interface{}{
		"hit_count":           cm.hitCount,
		"miss_count":          cm.missCount,
		"eviction_count":      cm.evictionCount,
		"invalidation_count":  cm.invalidationCount,
		"hit_rate":            hitRate,
		"total_operations":    total,
		"avg_operation_duration_ms": avgDuration,
	}
}

// Reset resets all metrics
func (cm *CacheMetrics) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.hitCount = 0
	cm.missCount = 0
	cm.evictionCount = 0
	cm.invalidationCount = 0
	cm.operationDurations = make([]time.Duration, 0)
}
