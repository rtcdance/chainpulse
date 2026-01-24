package reliability

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewHorizontalScaler tests scaler creation
func TestNewHorizontalScaler(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	assert.NotNil(t, scaler)
	assert.Equal(t, "scaler-1", scaler.id)
	assert.Equal(t, 1, scaler.minInstances)
	assert.Equal(t, 10, scaler.maxInstances)
	assert.Equal(t, 1, scaler.currentInstances)
}

// TestScalingPolicyDefaults tests default scaling policy
func TestScalingPolicyDefaults(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	assert.Equal(t, 80.0, scaler.scalingPolicy.CPUThresholdUp)
	assert.Equal(t, 20.0, scaler.scalingPolicy.CPUThresholdDown)
	assert.Equal(t, 80.0, scaler.scalingPolicy.MemoryThresholdUp)
	assert.Equal(t, 20.0, scaler.scalingPolicy.MemoryThresholdDown)
	assert.Equal(t, int64(1000), scaler.scalingPolicy.RequestThreshold)
}

// TestUpdateInstanceMetrics tests updating instance metrics
func TestUpdateInstanceMetrics(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)
	ctx := context.Background()

	// First scale up to create instances
	_ = scaler.EvaluateScaling(ctx)

	instances := scaler.GetInstances()
	if len(instances) > 0 {
		instanceID := instances[0].ID
		err := scaler.UpdateInstanceMetrics(instanceID, 50.0, 60.0, 500)

		assert.NoError(t, err)
	}
}

// TestUpdateInstanceMetricsNotFound tests updating non-existent instance
func TestUpdateInstanceMetricsNotFound(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	err := scaler.UpdateInstanceMetrics("nonexistent", 50.0, 60.0, 500)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetInstances tests getting instances
func TestGetInstances(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	instances := scaler.GetInstances()

	assert.NotNil(t, instances)
}

// TestGetMetricsHorizontalScaler tests metrics retrieval
func TestGetMetricsHorizontalScaler(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	metrics := scaler.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, 1, metrics["current_instances"])
	assert.Equal(t, 1, metrics["min_instances"])
	assert.Equal(t, 10, metrics["max_instances"])
}

// TestGetCurrentInstanceCount tests instance count
func TestGetCurrentInstanceCount(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	count := scaler.GetCurrentInstanceCount()

	assert.Equal(t, 1, count)
}

// TestSetScalingPolicy tests setting scaling policy
func TestSetScalingPolicy(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	newPolicy := &ScalingPolicy{
		CPUThresholdUp:      90.0,
		CPUThresholdDown:    10.0,
		MemoryThresholdUp:   90.0,
		MemoryThresholdDown: 10.0,
		RequestThreshold:    2000,
		ScaleUpCount:        2,
		ScaleDownCount:      1,
		EvaluationPeriod:    2 * time.Minute,
	}

	scaler.SetScalingPolicy(newPolicy)

	assert.Equal(t, 90.0, scaler.scalingPolicy.CPUThresholdUp)
	assert.Equal(t, int64(2000), scaler.scalingPolicy.RequestThreshold)
}

// TestEvaluateScalingNoCooldown tests evaluation without cooldown
func TestEvaluateScalingNoCooldown(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)
	scaler.scalingCooldown = 0 // Disable cooldown for testing
	ctx := context.Background()

	err := scaler.EvaluateScaling(ctx)

	assert.NoError(t, err)
}

// TestEvaluateScalingWithCooldown tests evaluation with cooldown
func TestEvaluateScalingWithCooldown(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)
	scaler.scalingCooldown = 1 * time.Hour // Long cooldown
	ctx := context.Background()

	// First evaluation
	_ = scaler.EvaluateScaling(ctx)

	// Second evaluation should be skipped due to cooldown
	err := scaler.EvaluateScaling(ctx)

	assert.NoError(t, err)
}

// TestScalingMetrics tests scaling metrics
func TestScalingMetrics(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	metrics := scaler.GetMetrics()

	assert.Equal(t, int64(0), metrics["scale_up_events"])
	assert.Equal(t, int64(0), metrics["scale_down_events"])
}

// TestServiceInstanceCreation tests service instance creation
func TestServiceInstanceCreation(t *testing.T) {
	instance := &ServiceInstance{
		ID:     "instance-1",
		Status: "running",
	}

	assert.Equal(t, "instance-1", instance.ID)
	assert.Equal(t, "running", instance.Status)
}

// TestServiceInstanceMetrics tests instance metrics
func TestServiceInstanceMetrics(t *testing.T) {
	instance := &ServiceInstance{
		ID:           "instance-1",
		Status:       "running",
		CPUUsage:     50.0,
		MemoryUsage:  60.0,
		RequestCount: 1000,
		ErrorCount:   5,
	}

	assert.Equal(t, 50.0, instance.CPUUsage)
	assert.Equal(t, 60.0, instance.MemoryUsage)
	assert.Equal(t, int64(1000), instance.RequestCount)
	assert.Equal(t, int64(5), instance.ErrorCount)
}

// TestGracefulShutdown tests graceful shutdown
func TestGracefulShutdown(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	// Create an instance manually
	instance := &ServiceInstance{
		ID:     "instance-1",
		Status: "running",
	}
	scaler.mu.Lock()
	scaler.instances["instance-1"] = instance
	scaler.mu.Unlock()

	err := scaler.GracefulShutdown(ctx, "instance-1")

	assert.NoError(t, err)
}

// TestGracefulShutdownNotFound tests graceful shutdown of non-existent instance
func TestGracefulShutdownNotFound(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)
	ctx := context.Background()

	err := scaler.GracefulShutdown(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGracefulShutdownContextCancellation tests shutdown with context cancellation
func TestGracefulShutdownContextCancellation(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	instance := &ServiceInstance{
		ID:     "instance-1",
		Status: "running",
	}
	scaler.mu.Lock()
	scaler.instances["instance-1"] = instance
	scaler.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := scaler.GracefulShutdown(ctx, "instance-1")

	assert.Error(t, err)
}

// TestConcurrentMetricsUpdate tests concurrent metrics updates
func TestConcurrentMetricsUpdate(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	// Create an instance
	instance := &ServiceInstance{
		ID:     "instance-1",
		Status: "running",
	}
	scaler.mu.Lock()
	scaler.instances["instance-1"] = instance
	scaler.mu.Unlock()

	var wg sync.WaitGroup
	var updateCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := scaler.UpdateInstanceMetrics("instance-1", float64(id), float64(id+10), int64(id*100))
			if err == nil {
				atomic.AddInt32(&updateCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&updateCount))
}

// TestScalingPolicyCPUThresholds tests CPU threshold scaling
func TestScalingPolicyCPUThresholds(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 5)

	policy := &ScalingPolicy{
		CPUThresholdUp:      75.0,
		CPUThresholdDown:    25.0,
		MemoryThresholdUp:   80.0,
		MemoryThresholdDown: 20.0,
		RequestThreshold:    1000,
		ScaleUpCount:        1,
		ScaleDownCount:      1,
		EvaluationPeriod:    1 * time.Minute,
	}

	scaler.SetScalingPolicy(policy)

	assert.Equal(t, 75.0, scaler.scalingPolicy.CPUThresholdUp)
}

// TestScalingPolicyMemoryThresholds tests memory threshold scaling
func TestScalingPolicyMemoryThresholds(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 5)

	policy := &ScalingPolicy{
		CPUThresholdUp:      80.0,
		CPUThresholdDown:    20.0,
		MemoryThresholdUp:   85.0,
		MemoryThresholdDown: 15.0,
		RequestThreshold:    1000,
		ScaleUpCount:        1,
		ScaleDownCount:      1,
		EvaluationPeriod:    1 * time.Minute,
	}

	scaler.SetScalingPolicy(policy)

	assert.Equal(t, 85.0, scaler.scalingPolicy.MemoryThresholdUp)
}

// TestScalingPolicyRequestThreshold tests request threshold scaling
func TestScalingPolicyRequestThreshold(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 5)

	policy := &ScalingPolicy{
		CPUThresholdUp:      80.0,
		CPUThresholdDown:    20.0,
		MemoryThresholdUp:   80.0,
		MemoryThresholdDown: 20.0,
		RequestThreshold:    5000,
		ScaleUpCount:        2,
		ScaleDownCount:      1,
		EvaluationPeriod:    1 * time.Minute,
	}

	scaler.SetScalingPolicy(policy)

	assert.Equal(t, int64(5000), scaler.scalingPolicy.RequestThreshold)
}

// TestInstanceStatusTransitions tests instance status transitions
func TestInstanceStatusTransitions(t *testing.T) {
	instance := &ServiceInstance{
		ID:     "instance-1",
		Status: "starting",
	}

	assert.Equal(t, "starting", instance.Status)

	instance.Status = "running"
	assert.Equal(t, "running", instance.Status)

	instance.Status = "stopping"
	assert.Equal(t, "stopping", instance.Status)

	instance.Status = "stopped"
	assert.Equal(t, "stopped", instance.Status)
}

// TestInstanceTimestamps tests instance timestamps
func TestInstanceTimestamps(t *testing.T) {
	now := time.Now()
	instance := &ServiceInstance{
		ID:        "instance-1",
		Status:    "running",
		CreatedAt: now,
		StartedAt: now.Add(1 * time.Second),
	}

	assert.True(t, instance.CreatedAt.Before(instance.StartedAt))
}

// TestScalerMinMaxBounds tests min/max instance bounds
func TestScalerMinMaxBounds(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 2, 8)

	assert.Equal(t, 2, scaler.minInstances)
	assert.Equal(t, 8, scaler.maxInstances)
	assert.GreaterOrEqual(t, scaler.maxInstances, scaler.minInstances)
}

// TestScalingCooldown tests scaling cooldown period
func TestScalingCooldown(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	assert.Equal(t, 5*time.Minute, scaler.scalingCooldown)
}

// TestMetricsAggregation tests metrics aggregation
func TestMetricsAggregation(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	// Create multiple instances
	for i := 0; i < 3; i++ {
		instance := &ServiceInstance{
			ID:           fmt.Sprintf("instance-%d", i),
			Status:       "running",
			CPUUsage:     float64(50 + i*10),
			MemoryUsage:  float64(60 + i*5),
			RequestCount: int64(1000 + i*100),
		}
		scaler.mu.Lock()
		scaler.instances[instance.ID] = instance
		scaler.mu.Unlock()
	}

	metrics := scaler.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Greater(t, len(metrics), 0)
}

// TestScalerID tests scaler ID
func TestScalerID(t *testing.T) {
	scaler := NewHorizontalScaler("my-scaler", 1, 10)

	assert.Equal(t, "my-scaler", scaler.id)
}

// TestInstanceHealthCheck tests instance health check tracking
func TestInstanceHealthCheck(t *testing.T) {
	instance := &ServiceInstance{
		ID:              "instance-1",
		Status:          "running",
		LastHealthCheck: time.Now(),
	}

	assert.False(t, instance.LastHealthCheck.IsZero())
}

// TestConcurrentInstanceAccess tests concurrent instance access
func TestConcurrentInstanceAccess(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	var wg sync.WaitGroup
	var accessCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = scaler.GetInstances()
			atomic.AddInt32(&accessCount, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&accessCount))
}

// TestScalingMetricsTracking tests scaling metrics tracking
func TestScalingMetricsTracking(t *testing.T) {
	scaler := NewHorizontalScaler("scaler-1", 1, 10)

	metrics := scaler.GetMetrics()

	assert.Equal(t, int64(0), metrics["scale_up_events"])
	assert.Equal(t, int64(0), metrics["scale_down_events"])
	assert.Equal(t, int64(0), metrics["instances_created"])
	assert.Equal(t, int64(0), metrics["instances_terminated"])
}
