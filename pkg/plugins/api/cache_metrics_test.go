package api

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCacheMetrics tests cache metrics initialization
func TestNewCacheMetrics(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	cm := NewCacheMetrics(logger, metrics)

	require.NotNil(t, cm)
	assert.Equal(t, int64(0), cm.hitCount)
	assert.Equal(t, int64(0), cm.missCount)
	assert.Equal(t, int64(0), cm.evictionCount)
	assert.Equal(t, int64(0), cm.invalidationCount)
	assert.Equal(t, 0, len(cm.operationDurations))
}

// TestRecordHit tests recording cache hits
func TestRecordHit(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordHit("key1", 10*time.Millisecond)

	assert.Equal(t, int64(1), cm.hitCount)
	assert.Equal(t, int64(0), cm.missCount)
	assert.Equal(t, 1, len(cm.operationDurations))
	assert.Equal(t, int64(1), metrics.GetCounterValue("cache_hit"))
}

// TestRecordMiss tests recording cache misses
func TestRecordMiss(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordMiss("key1", 5*time.Millisecond)

	assert.Equal(t, int64(0), cm.hitCount)
	assert.Equal(t, int64(1), cm.missCount)
	assert.Equal(t, 1, len(cm.operationDurations))
	assert.Equal(t, int64(1), metrics.GetCounterValue("cache_miss"))
}

// TestRecordEviction tests recording cache evictions
func TestRecordEviction(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordEviction("key1", "lru")

	assert.Equal(t, int64(1), cm.evictionCount)
	assert.Equal(t, int64(1), metrics.GetCounterValue("cache_eviction"))
}

// TestRecordInvalidation tests recording cache invalidations
func TestRecordInvalidation(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordInvalidation("key1", "ttl_expired")

	assert.Equal(t, int64(1), cm.invalidationCount)
	assert.Equal(t, int64(1), metrics.GetCounterValue("cache_invalidation"))
}

// TestGetHitRate tests hit rate calculation
func TestGetHitRate(t *testing.T) {
	tests := []struct {
		name     string
		hits     int
		misses   int
		expected float64
	}{
		{"no operations", 0, 0, 0.0},
		{"all hits", 10, 0, 1.0},
		{"all misses", 0, 10, 0.0},
		{"50/50", 5, 5, 0.5},
		{"75/25", 3, 1, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &MockLogger{}
			metrics := NewMockMetricsCollector()
			cm := NewCacheMetrics(logger, metrics)

			for i := 0; i < tt.hits; i++ {
				cm.RecordHit("key", 1*time.Millisecond)
			}
			for i := 0; i < tt.misses; i++ {
				cm.RecordMiss("key", 1*time.Millisecond)
			}

			assert.Equal(t, tt.expected, cm.GetHitRate())
		})
	}
}

// TestGetStats tests statistics retrieval
func TestGetStats(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordHit("key1", 10*time.Millisecond)
	cm.RecordHit("key2", 20*time.Millisecond)
	cm.RecordMiss("key3", 5*time.Millisecond)
	cm.RecordEviction("key4", "lru")
	cm.RecordInvalidation("key5", "ttl")

	stats := cm.GetStats()

	assert.Equal(t, int64(2), stats["hit_count"])
	assert.Equal(t, int64(1), stats["miss_count"])
	assert.Equal(t, int64(1), stats["eviction_count"])
	assert.Equal(t, int64(1), stats["invalidation_count"])
	assert.Equal(t, int64(3), stats["total_operations"])
	assert.Equal(t, 2.0/3.0, stats["hit_rate"])
	assert.NotNil(t, stats["avg_operation_duration_ms"])
}

// TestGetStatsEmpty tests statistics with no operations
func TestGetStatsEmpty(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	stats := cm.GetStats()

	assert.Equal(t, int64(0), stats["hit_count"])
	assert.Equal(t, int64(0), stats["miss_count"])
	assert.Equal(t, int64(0), stats["eviction_count"])
	assert.Equal(t, int64(0), stats["invalidation_count"])
	assert.Equal(t, int64(0), stats["total_operations"])
	assert.Equal(t, 0.0, stats["hit_rate"])
	assert.Equal(t, 0.0, stats["avg_operation_duration_ms"])
}

// TestReset tests metrics reset
func TestReset(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordHit("key1", 10*time.Millisecond)
	cm.RecordMiss("key2", 5*time.Millisecond)
	cm.RecordEviction("key3", "lru")
	cm.RecordInvalidation("key4", "ttl")

	cm.Reset()

	assert.Equal(t, int64(0), cm.hitCount)
	assert.Equal(t, int64(0), cm.missCount)
	assert.Equal(t, int64(0), cm.evictionCount)
	assert.Equal(t, int64(0), cm.invalidationCount)
	assert.Equal(t, 0, len(cm.operationDurations))
}

// TestConcurrentRecordHit tests concurrent hit recording
func TestConcurrentRecordHit(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				cm.RecordHit("key", 1*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	expectedHits := int64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expectedHits, cm.hitCount)
	assert.Equal(t, expectedHits, int64(len(cm.operationDurations)))
}

// TestConcurrentRecordMiss tests concurrent miss recording
func TestConcurrentRecordMiss(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				cm.RecordMiss("key", 1*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	expectedMisses := int64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expectedMisses, cm.missCount)
}

// TestConcurrentRecordEviction tests concurrent eviction recording
func TestConcurrentRecordEviction(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				cm.RecordEviction("key", "lru")
			}
		}(i)
	}

	wg.Wait()

	expectedEvictions := int64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expectedEvictions, cm.evictionCount)
}

// TestConcurrentRecordInvalidation tests concurrent invalidation recording
func TestConcurrentRecordInvalidation(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				cm.RecordInvalidation("key", "ttl")
			}
		}(i)
	}

	wg.Wait()

	expectedInvalidations := int64(numGoroutines * operationsPerGoroutine)
	assert.Equal(t, expectedInvalidations, cm.invalidationCount)
}

// TestConcurrentMixedOperations tests concurrent mixed operations
func TestConcurrentMixedOperations(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				cm.RecordHit("key", 1*time.Millisecond)
				cm.RecordMiss("key", 1*time.Millisecond)
				cm.RecordEviction("key", "lru")
				cm.RecordInvalidation("key", "ttl")
			}
		}(i)
	}

	wg.Wait()

	expectedOps := int64(numGoroutines * 25)
	assert.Equal(t, expectedOps, cm.hitCount)
	assert.Equal(t, expectedOps, cm.missCount)
	assert.Equal(t, expectedOps, cm.evictionCount)
	assert.Equal(t, expectedOps, cm.invalidationCount)
}

// TestConcurrentGetStats tests concurrent stats retrieval
func TestConcurrentGetStats(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	// Pre-populate metrics
	for i := 0; i < 100; i++ {
		cm.RecordHit("key", 1*time.Millisecond)
		cm.RecordMiss("key", 1*time.Millisecond)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	statsResults := make([]map[string]interface{}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			statsResults[id] = cm.GetStats()
		}(i)
	}

	wg.Wait()

	// Verify all stats are consistent
	for i := 0; i < numGoroutines; i++ {
		assert.Equal(t, int64(100), statsResults[i]["hit_count"])
		assert.Equal(t, int64(100), statsResults[i]["miss_count"])
		assert.Equal(t, 0.5, statsResults[i]["hit_rate"])
	}
}

// TestMultipleEvictionReasons tests evictions with different reasons
func TestMultipleEvictionReasons(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	reasons := []string{"lru", "lfu", "fifo", "ttl"}
	for i := 0; i < 10; i++ {
		for _, reason := range reasons {
			cm.RecordEviction("key", reason)
		}
	}

	assert.Equal(t, int64(40), cm.evictionCount)
	assert.Equal(t, int64(40), metrics.GetCounterValue("cache_eviction"))
}

// TestMultipleInvalidationReasons tests invalidations with different reasons
func TestMultipleInvalidationReasons(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	reasons := []string{"ttl_expired", "manual", "dependency_change"}
	for i := 0; i < 10; i++ {
		for _, reason := range reasons {
			cm.RecordInvalidation("key", reason)
		}
	}

	assert.Equal(t, int64(30), cm.invalidationCount)
	assert.Equal(t, int64(30), metrics.GetCounterValue("cache_invalidation"))
}

// TestOperationDurationTracking tests operation duration tracking
func TestOperationDurationTracking(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	durations := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
	}

	for _, d := range durations {
		cm.RecordHit("key", d)
	}

	assert.Equal(t, len(durations), len(cm.operationDurations))

	stats := cm.GetStats()
	expectedAvg := 3.0 // (1+2+3+4+5)/5 = 3
	assert.Equal(t, expectedAvg, stats["avg_operation_duration_ms"])
}

// TestHistogramRecording tests histogram recording in metrics
func TestHistogramRecording(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordHit("key1", 10*time.Millisecond)
	cm.RecordHit("key2", 20*time.Millisecond)
	cm.RecordMiss("key3", 5*time.Millisecond)

	hitHistogram := metrics.GetHistogramValues("cache_hit_duration_ms")
	missHistogram := metrics.GetHistogramValues("cache_miss_duration_ms")

	assert.Equal(t, 2, len(hitHistogram))
	assert.Equal(t, 1, len(missHistogram))
	assert.Contains(t, hitHistogram, float64(10))
	assert.Contains(t, hitHistogram, float64(20))
	assert.Contains(t, missHistogram, float64(5))
}

// TestResetClearsOperationDurations tests that reset clears operation durations
func TestResetClearsOperationDurations(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	for i := 0; i < 100; i++ {
		cm.RecordHit("key", 1*time.Millisecond)
	}

	assert.Equal(t, 100, len(cm.operationDurations))

	cm.Reset()

	assert.Equal(t, 0, len(cm.operationDurations))
	stats := cm.GetStats()
	assert.Equal(t, 0.0, stats["avg_operation_duration_ms"])
}

// TestLargeScaleMetrics tests metrics with large number of operations
func TestLargeScaleMetrics(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	numOperations := 10000

	for i := 0; i < numOperations; i++ {
		if i%2 == 0 {
			cm.RecordHit("key", 1*time.Millisecond)
		} else {
			cm.RecordMiss("key", 1*time.Millisecond)
		}
	}

	assert.Equal(t, int64(numOperations/2), cm.hitCount)
	assert.Equal(t, int64(numOperations/2), cm.missCount)
	assert.Equal(t, 0.5, cm.GetHitRate())
	assert.Equal(t, numOperations, len(cm.operationDurations))
}

// TestStatsConsistency tests that stats remain consistent across calls
func TestStatsConsistency(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	cm := NewCacheMetrics(logger, metrics)

	cm.RecordHit("key1", 10*time.Millisecond)
	cm.RecordMiss("key2", 5*time.Millisecond)

	stats1 := cm.GetStats()
	stats2 := cm.GetStats()

	assert.Equal(t, stats1, stats2)
}
