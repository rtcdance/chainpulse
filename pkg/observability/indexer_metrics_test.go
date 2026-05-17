package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIndexerMetrics(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	require.NotNil(t, metrics)
	assert.Equal(t, uint64(0), metrics.CurrentBlockNumber)
	assert.Equal(t, uint64(0), metrics.LatestBlockNumber)
	assert.Equal(t, int64(0), metrics.EventsIndexed)
	assert.Equal(t, int64(0), metrics.EventsProcessed)
	assert.Equal(t, int64(0), metrics.EventsFailed)
	assert.NotNil(t, metrics.IndexingLatencies)
	assert.NotNil(t, metrics.QueryLatencies)
	assert.Equal(t, 0, metrics.IndexingLatencies.Len())
	assert.NotNil(t, metrics.ErrorCount)
}

func TestRecordIndexingProgress(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordIndexingProgress(1000, 1100)

	assert.Equal(t, uint64(1000), metrics.CurrentBlockNumber)
	assert.Equal(t, uint64(1100), metrics.LatestBlockNumber)
	assert.Equal(t, uint64(100), metrics.IndexingLag)
}

func TestRecordIndexingProgressNoLag(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordIndexingProgress(1000, 1000)

	assert.Equal(t, uint64(1000), metrics.CurrentBlockNumber)
	assert.Equal(t, uint64(1000), metrics.LatestBlockNumber)
	assert.Equal(t, uint64(0), metrics.IndexingLag)
}

func TestRecordEventIndexed(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	latency := 50 * time.Millisecond
	metrics.RecordEventIndexed(latency)

	assert.Equal(t, int64(1), metrics.EventsIndexed)
	assert.Equal(t, 1, metrics.IndexingLatencies.Len())
	assert.Equal(t, latency, metrics.IndexingLatencies.All()[0])
}

func TestRecordEventIndexedMultiple(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	for i := 0; i < 5; i++ {
		metrics.RecordEventIndexed(time.Duration(i*10) * time.Millisecond)
	}

	assert.Equal(t, int64(5), metrics.EventsIndexed)
	assert.Equal(t, 5, metrics.IndexingLatencies.Len())
}

func TestRecordEventProcessed(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordEventProcessed()
	metrics.RecordEventProcessed()

	assert.Equal(t, int64(2), metrics.EventsProcessed)
}

func TestRecordEventFailed(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordEventFailed("decode_error")
	metrics.RecordEventFailed("validation_error")
	metrics.RecordEventFailed("decode_error")

	assert.Equal(t, int64(3), metrics.EventsFailed)
	assert.Equal(t, int64(2), metrics.ErrorCount["decode_error"])
	assert.Equal(t, int64(1), metrics.ErrorCount["validation_error"])
}

func TestRecordQueryLatency(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	latency := 100 * time.Millisecond
	metrics.RecordQueryLatency(latency)

	assert.Equal(t, 1, metrics.QueryLatencies.Len())
	assert.Equal(t, latency, metrics.QueryLatencies.All()[0])
}

func TestRecordReorg(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordReorg(50)

	assert.Equal(t, int64(1), metrics.ReorgsDetected)
	assert.Equal(t, int64(50), metrics.BlocksRolledBack)
	assert.False(t, metrics.LastReorgTime.IsZero())
}

func TestRecordCacheHit(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordCacheHit()
	metrics.RecordCacheHit()

	assert.Equal(t, int64(2), metrics.CacheHits)
}

func TestRecordCacheMiss(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordCacheMiss()
	metrics.RecordCacheMiss()
	metrics.RecordCacheMiss()

	assert.Equal(t, int64(3), metrics.CacheMisses)
}

func TestGetAverageIndexingLatency(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordEventIndexed(10 * time.Millisecond)
	metrics.RecordEventIndexed(20 * time.Millisecond)
	metrics.RecordEventIndexed(30 * time.Millisecond)

	avg := metrics.GetAverageIndexingLatency()
	assert.Equal(t, 20*time.Millisecond, avg)
}

func TestGetAverageIndexingLatencyEmpty(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	avg := metrics.GetAverageIndexingLatency()
	assert.Equal(t, time.Duration(0), avg)
}

func TestGetMaxIndexingLatency(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordEventIndexed(10 * time.Millisecond)
	metrics.RecordEventIndexed(50 * time.Millisecond)
	metrics.RecordEventIndexed(30 * time.Millisecond)

	max := metrics.GetMaxIndexingLatency()
	assert.Equal(t, 50*time.Millisecond, max)
}

func TestGetMaxIndexingLatencyEmpty(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	max := metrics.GetMaxIndexingLatency()
	assert.Equal(t, time.Duration(0), max)
}

func TestGetAverageQueryLatency(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordQueryLatency(50 * time.Millisecond)
	metrics.RecordQueryLatency(100 * time.Millisecond)
	metrics.RecordQueryLatency(150 * time.Millisecond)

	avg := metrics.GetAverageQueryLatency()
	assert.Equal(t, 100*time.Millisecond, avg)
}

func TestGetMaxQueryLatency(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordQueryLatency(50 * time.Millisecond)
	metrics.RecordQueryLatency(200 * time.Millisecond)
	metrics.RecordQueryLatency(100 * time.Millisecond)

	max := metrics.GetMaxQueryLatency()
	assert.Equal(t, 200*time.Millisecond, max)
}

func TestGetCacheHitRate(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()
	metrics.RecordCacheMiss()

	rate := metrics.GetCacheHitRate()
	assert.Equal(t, 50.0, rate)
}

func TestGetCacheHitRateEmpty(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	rate := metrics.GetCacheHitRate()
	assert.Equal(t, 0.0, rate)
}

func TestGetCacheHitRateAllHits(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()

	rate := metrics.GetCacheHitRate()
	assert.Equal(t, 100.0, rate)
}

func TestGetIndexingRate(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	// Record some events
	for i := 0; i < 10; i++ {
		metrics.RecordEventIndexed(time.Millisecond)
	}

	rate := metrics.GetIndexingRate()
	assert.Greater(t, rate, 0.0)
}

func TestGetErrorRate(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	for i := 0; i < 8; i++ {
		metrics.RecordEventProcessed()
	}
	for i := 0; i < 2; i++ {
		metrics.RecordEventFailed("error")
	}

	rate := metrics.GetErrorRate()
	assert.Equal(t, 20.0, rate)
}

func TestGetErrorRateEmpty(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	rate := metrics.GetErrorRate()
	assert.Equal(t, 0.0, rate)
}

func TestGetMetricsSummary(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordIndexingProgress(1000, 1100)
	metrics.RecordEventIndexed(50 * time.Millisecond)
	metrics.RecordEventProcessed()
	metrics.RecordEventFailed("error")
	metrics.RecordCacheHit()
	metrics.RecordReorg(50)

	summary := metrics.GetMetricsSummary()

	require.NotNil(t, summary)
	assert.Equal(t, uint64(1000), summary["current_block"])
	assert.Equal(t, uint64(1100), summary["latest_block"])
	assert.Equal(t, uint64(100), summary["indexing_lag"])
	assert.Equal(t, int64(1), summary["events_indexed"])
	assert.Equal(t, int64(1), summary["events_processed"])
	assert.Equal(t, int64(1), summary["events_failed"])
	assert.Equal(t, int64(1), summary["cache_hits"])
	assert.Equal(t, int64(1), summary["reorgs_detected"])
	assert.Equal(t, int64(50), summary["blocks_rolled_back"])
}

func TestReset(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	metrics.RecordIndexingProgress(1000, 1100)
	metrics.RecordEventIndexed(50 * time.Millisecond)
	metrics.RecordEventProcessed()
	metrics.RecordEventFailed("error")
	metrics.RecordCacheHit()
	metrics.RecordReorg(50)

	metrics.Reset()

	assert.Equal(t, uint64(0), metrics.CurrentBlockNumber)
	assert.Equal(t, uint64(0), metrics.LatestBlockNumber)
	assert.Equal(t, int64(0), metrics.EventsIndexed)
	assert.Equal(t, int64(0), metrics.EventsProcessed)
	assert.Equal(t, int64(0), metrics.EventsFailed)
	assert.Equal(t, int64(0), metrics.CacheHits)
	assert.Equal(t, int64(0), metrics.ReorgsDetected)
	assert.Equal(t, 0, metrics.IndexingLatencies.Len())
	assert.Equal(t, 0, metrics.QueryLatencies.Len())
	assert.Equal(t, 0, len(metrics.ErrorCount))
}

// Property-based tests

func TestPropertyMetricsThreadSafety(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	// Simulate concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				metrics.RecordEventIndexed(time.Millisecond)
				metrics.RecordEventProcessed()
				metrics.RecordCacheHit()
				metrics.GetMetricsSummary()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	assert.Equal(t, int64(1000), metrics.EventsIndexed)
	assert.Equal(t, int64(1000), metrics.EventsProcessed)
	assert.Equal(t, int64(1000), metrics.CacheHits)
}

func TestPropertyMetricsConsistency(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	// Record events
	for i := 0; i < 100; i++ {
		metrics.RecordEventIndexed(time.Duration(i) * time.Millisecond)
	}

	// Verify consistency
	assert.Equal(t, int64(100), metrics.EventsIndexed)
	assert.Equal(t, 100, metrics.IndexingLatencies.Len())

	// Average should be within expected range
	avg := metrics.GetAverageIndexingLatency()
	assert.Greater(t, avg, time.Duration(0))
	assert.Less(t, avg, 100*time.Millisecond)
}

func TestPropertyMetricsAccuracy(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	// Record specific values
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()

	// Verify accuracy
	rate := metrics.GetCacheHitRate()
	assert.Equal(t, 75.0, rate)

	total := metrics.CacheHits + metrics.CacheMisses
	assert.Equal(t, int64(4), total)
}

func TestPropertyMetricsLatencyBounds(t *testing.T) {
	t.Parallel()
	metrics := NewIndexerMetrics()

	// Record latencies
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	for _, latency := range latencies {
		metrics.RecordEventIndexed(latency)
	}

	// Verify bounds
	avg := metrics.GetAverageIndexingLatency()
	max := metrics.GetMaxIndexingLatency()

	assert.Equal(t, 30*time.Millisecond, avg)
	assert.Equal(t, 50*time.Millisecond, max)
	assert.LessOrEqual(t, avg, max)
}
