package observability

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIndexerHealth(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	require.NotNil(t, health)
	assert.Equal(t, uint64(100), health.maxAllowedLag)
	assert.Equal(t, 5.0, health.maxErrorRate)
	assert.Equal(t, 80.0, health.minCacheHitRate)
	assert.Equal(t, 200*time.Millisecond, health.maxAverageLatency)
}

func TestCheckHealthHealthy(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Set up healthy metrics
	metrics.RecordIndexingProgress(1000, 1050)
	for i := 0; i < 95; i++ {
		metrics.RecordEventProcessed()
	}
	for i := 0; i < 5; i++ {
		metrics.RecordEventFailed("error")
	}
	for i := 0; i < 80; i++ {
		metrics.RecordCacheHit()
	}
	for i := 0; i < 20; i++ {
		metrics.RecordCacheMiss()
	}
	for i := 0; i < 10; i++ {
		metrics.RecordEventIndexed(50 * time.Millisecond)
	}

	result := health.CheckHealth(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Equal(t, uint64(1000), result.CurrentBlock)
	assert.Equal(t, uint64(1050), result.LatestBlock)
	assert.Equal(t, uint64(50), result.IndexingLag)
}

func TestCheckHealthDegradedHighLag(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Set up degraded metrics (high lag)
	metrics.RecordIndexingProgress(1000, 1200)

	result := health.CheckHealth(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthStatusDegraded, result.Status)
	assert.Equal(t, uint64(200), result.IndexingLag)
	assert.Contains(t, result.Message, "indexing lag")
}

func TestCheckHealthDegradedHighErrorRate(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Set up degraded metrics (high error rate)
	metrics.RecordIndexingProgress(1000, 1050)
	for i := 0; i < 50; i++ {
		metrics.RecordEventProcessed()
	}
	for i := 0; i < 50; i++ {
		metrics.RecordEventFailed("error")
	}

	result := health.CheckHealth(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthStatusDegraded, result.Status)
	assert.Contains(t, result.Message, "error rate")
}

func TestCheckHealthUnhealthyDatabaseDown(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Set database check to fail
	health.SetDatabaseCheckFunc(func(ctx context.Context) error {
		return fmt.Errorf("database connection failed")
	})

	result := health.CheckHealth(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.False(t, result.DatabaseConnected)
	assert.Contains(t, result.Message, "database disconnected")
}

func TestCheckHealthUnhealthyCacheDown(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Set cache check to fail
	health.SetCacheCheckFunc(func(ctx context.Context) error {
		return fmt.Errorf("cache connection failed")
	})

	result := health.CheckHealth(context.Background())

	require.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.False(t, result.CacheConnected)
	assert.Contains(t, result.Message, "cache disconnected")
}

func TestGetLastHealthCheck(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1050)
	result1 := health.CheckHealth(context.Background())

	result2 := health.GetLastHealthCheck()

	require.NotNil(t, result2)
	assert.Equal(t, result1.Status, result2.Status)
	assert.Equal(t, result1.CurrentBlock, result2.CurrentBlock)
}

func TestIsHealthy(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1050)
	health.CheckHealth(context.Background())

	assert.True(t, health.IsHealthy())
	assert.False(t, health.IsDegraded())
	assert.False(t, health.IsUnhealthy())
}

func TestIsDegraded(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1200)
	health.CheckHealth(context.Background())

	assert.False(t, health.IsHealthy())
	assert.True(t, health.IsDegraded())
	assert.False(t, health.IsUnhealthy())
}

func TestIsUnhealthy(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	health.SetDatabaseCheckFunc(func(ctx context.Context) error {
		return fmt.Errorf("database down")
	})

	health.CheckHealth(context.Background())

	assert.False(t, health.IsHealthy())
	assert.False(t, health.IsDegraded())
	assert.True(t, health.IsUnhealthy())
}

func TestGetHealthSummary(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1050)
	health.CheckHealth(context.Background())

	summary := health.GetHealthSummary()

	require.NotNil(t, summary)
	assert.Equal(t, "healthy", summary["status"])
	assert.Equal(t, uint64(1000), summary["current_block"])
	assert.Equal(t, uint64(1050), summary["latest_block"])
	assert.Equal(t, uint64(50), summary["indexing_lag"])
}

func TestGetHealthSummaryNoCheck(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	summary := health.GetHealthSummary()

	require.NotNil(t, summary)
	assert.Equal(t, "unknown", summary["status"])
}

func TestDetectLag(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1150)

	isLagging, lag := health.DetectLag(context.Background())

	assert.True(t, isLagging)
	assert.Equal(t, uint64(150), lag)
}

func TestDetectLagWithinThreshold(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1050)

	isLagging, lag := health.DetectLag(context.Background())

	assert.False(t, isLagging)
	assert.Equal(t, uint64(50), lag)
}

func TestGetLagPercentage(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1050)

	percentage := health.GetLagPercentage()

	assert.Equal(t, 50.0, percentage)
}

func TestGetLagPercentageAtMax(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1100)

	percentage := health.GetLagPercentage()

	assert.Equal(t, 100.0, percentage)
}

func TestGetLagPercentageExceedsMax(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	metrics.RecordIndexingProgress(1000, 1200)

	percentage := health.GetLagPercentage()

	assert.Equal(t, 200.0, percentage)
}

// Property-based tests

func TestPropertyHealthCheckConsistency(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Record metrics
	metrics.RecordIndexingProgress(1000, 1050)
	for i := 0; i < 100; i++ {
		metrics.RecordEventProcessed()
	}

	// Check health multiple times
	result1 := health.CheckHealth(context.Background())
	result2 := health.CheckHealth(context.Background())

	// Results should be consistent
	assert.Equal(t, result1.Status, result2.Status)
	assert.Equal(t, result1.CurrentBlock, result2.CurrentBlock)
	assert.Equal(t, result1.LatestBlock, result2.LatestBlock)
}

func TestPropertyHealthCheckAccuracy(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Set up specific metrics
	metrics.RecordIndexingProgress(1000, 1050)
	for i := 0; i < 95; i++ {
		metrics.RecordEventProcessed()
	}
	for i := 0; i < 5; i++ {
		metrics.RecordEventFailed("error")
	}

	result := health.CheckHealth(context.Background())

	// Verify accuracy
	assert.Equal(t, uint64(50), result.IndexingLag)
	assert.Equal(t, 5.0, result.ErrorRate)
	assert.Equal(t, HealthStatusHealthy, result.Status)
}

func TestPropertyHealthCheckBoundaries(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Test at boundary conditions
	testCases := []struct {
		lag    uint64
		status HealthStatus
	}{
		{50, HealthStatusHealthy},
		{100, HealthStatusHealthy},
		{101, HealthStatusDegraded},
		{200, HealthStatusDegraded},
	}

	for _, tc := range testCases {
		metrics.Reset()
		metrics.RecordIndexingProgress(1000, 1000+tc.lag)

		result := health.CheckHealth(context.Background())
		assert.Equal(t, tc.status, result.Status, fmt.Sprintf("lag=%d", tc.lag))
	}
}

func TestPropertyLagPercentageMonotonicity(t *testing.T) {
	metrics := NewIndexerMetrics()
	health := NewIndexerHealth(metrics, 100, 5.0, 80.0, 200*time.Millisecond)

	// Test that lag percentage increases monotonically
	previousPercentage := 0.0

	for lag := uint64(0); lag <= 200; lag += 20 {
		metrics.RecordIndexingProgress(1000, 1000+lag)
		percentage := health.GetLagPercentage()

		assert.GreaterOrEqual(t, percentage, previousPercentage)
		previousPercentage = percentage
	}
}
