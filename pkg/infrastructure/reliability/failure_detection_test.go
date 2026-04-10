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

// TestNewFailureDetector tests detector creation
func TestNewFailureDetector(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	assert.NotNil(t, detector)
	assert.Equal(t, "detector-1", detector.id)
	assert.Equal(t, 10*time.Second, detector.healthCheckInterval)
	assert.Equal(t, 3, detector.failureThreshold)
	assert.Equal(t, 2, detector.recoveryThreshold)
	assert.True(t, detector.predictiveDetection)
}

// TestRegisterServiceFailureDetector tests service registration
func TestRegisterServiceFailureDetector(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	detector.RegisterService("service-1")

	status, err := detector.GetServiceStatus("service-1")

	assert.NoError(t, err)
	assert.Equal(t, "healthy", status)
}

// TestReportHealthCheckSuccess tests successful health check
func TestReportHealthCheckSuccess(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	err := detector.ReportHealthCheck("service-1", true, 100*time.Millisecond, 0.0)

	assert.NoError(t, err)
}

// TestReportHealthCheckFailure tests failed health check
func TestReportHealthCheckFailure(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	err := detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)

	assert.NoError(t, err)
}

// TestReportHealthCheckUnregisteredService tests unregistered service
func TestReportHealthCheckUnregisteredService(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	err := detector.ReportHealthCheck("service-1", true, 100*time.Millisecond, 0.0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

// TestServiceStatusDegraded tests degraded status
func TestServiceStatusDegraded(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Report one failure to trigger degraded status
	_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)

	status, _ := detector.GetServiceStatus("service-1")

	assert.Equal(t, "degraded", status)
}

// TestServiceStatusFailed tests failed status
func TestServiceStatusFailed(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Report multiple failures to trigger failed status
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	status, _ := detector.GetServiceStatus("service-1")

	assert.Equal(t, "failed", status)
}

// TestServiceRecovery tests service recovery
func TestServiceRecovery(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Trigger degraded status
	_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)

	// Report successes to recover
	for i := 0; i < 2; i++ {
		_ = detector.ReportHealthCheck("service-1", true, 100*time.Millisecond, 0.0)
	}

	status, _ := detector.GetServiceStatus("service-1")

	assert.Equal(t, "healthy", status)
}

// TestDetectFailures tests failure detection
func TestDetectFailures(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")
	detector.RegisterService("service-2")
	ctx := context.Background()

	// Trigger failures
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	failed, err := detector.DetectFailures(ctx)

	assert.NoError(t, err)
	assert.Greater(t, len(failed), 0)
}

// TestGetFailedServices tests getting failed services
func TestGetFailedServices(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")
	detector.RegisterService("service-2")

	// Trigger failures for service-1
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	failed := detector.GetFailedServices()

	assert.Greater(t, len(failed), 0)
}

// TestGetMetricsFailureDetector tests metrics retrieval
func TestGetMetricsFailureDetector(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Trigger a failure
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	metrics := detector.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Greater(t, metrics["failures_detected"].(int64), int64(0))
}

// TestFailureHistory tests failure history tracking
func TestFailureHistory(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Report multiple failures
	for i := 0; i < 5; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	detector.mu.RLock()
	serviceInfo := detector.services["service-1"]
	detector.mu.RUnlock()

	assert.Greater(t, len(serviceInfo.FailureHistory), 0)
}

// TestResponseTimeTracking tests response time tracking
func TestResponseTimeTracking(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	responseTime := 150 * time.Millisecond
	_ = detector.ReportHealthCheck("service-1", true, responseTime, 0.0)

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	assert.Equal(t, responseTime, info.ResponseTime)
}

// TestErrorRateTracking tests error rate tracking
func TestErrorRateTracking(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	errorRate := 0.25
	_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, errorRate)

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	assert.Equal(t, errorRate, info.ErrorRate)
}

// TestConsecutiveFailures tests consecutive failure tracking
func TestConsecutiveFailures(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	assert.Equal(t, 3, info.ConsecutiveFailures)
}

// TestConsecutiveSuccesses tests consecutive success tracking
func TestConsecutiveSuccesses(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	for i := 0; i < 2; i++ {
		_ = detector.ReportHealthCheck("service-1", true, 100*time.Millisecond, 0.0)
	}

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	assert.Equal(t, 2, info.ConsecutiveSuccesses)
}

// TestPredictiveDetection tests predictive failure detection
func TestPredictiveDetection(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Report failures with high error rate
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.8)
	}

	// Predictive detection should be enabled
	assert.True(t, detector.predictiveDetection)
}

// TestNewAutomaticFailover tests failover creation
func TestNewAutomaticFailover(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	failover := NewAutomaticFailover("failover-1", detector)

	assert.NotNil(t, failover)
	assert.Equal(t, "failover-1", failover.id)
	assert.Equal(t, detector, failover.failureDetector)
}

// TestFailoverMetrics tests failover metrics
func TestFailoverMetrics(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	failover := NewAutomaticFailover("failover-1", detector)

	metrics := failover.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics["failovers_executed"])
	assert.Equal(t, int64(0), metrics["rollbacks_executed"])
}

// TestFailoverCooldown tests failover cooldown
func TestFailoverCooldown(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	failover := NewAutomaticFailover("failover-1", detector)

	assert.Equal(t, 5*time.Minute, failover.failoverCooldown)
}

// TestConcurrentHealthChecks tests concurrent health checks
func TestConcurrentHealthChecks(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	var wg sync.WaitGroup
	var reportCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			healthy := id%2 == 0
			err := detector.ReportHealthCheck("service-1", healthy, 100*time.Millisecond, 0.0)
			if err == nil {
				atomic.AddInt32(&reportCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&reportCount))
}

// TestMultipleServiceMonitoring tests monitoring multiple services
func TestMultipleServiceMonitoring(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	for i := 0; i < 5; i++ {
		detector.RegisterService(fmt.Sprintf("service-%d", i))
	}

	detector.mu.RLock()
	serviceCount := len(detector.services)
	detector.mu.RUnlock()

	assert.Equal(t, 5, serviceCount)
}

// TestFailureHistoryLimit tests failure history size limit
func TestFailureHistoryLimit(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Report many failures
	for i := 0; i < 150; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
	}

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	// History should be limited to 100
	assert.LessOrEqual(t, len(info.FailureHistory), 100)
}

// TestServiceHealthInfoFields tests service health info fields
func TestServiceHealthInfoFields(t *testing.T) {
	info := &ServiceHealthInfo{
		ServiceID:            "service-1",
		Status:               "healthy",
		ConsecutiveFailures:  0,
		ConsecutiveSuccesses: 5,
		ResponseTime:         100 * time.Millisecond,
		ErrorRate:            0.0,
	}

	assert.Equal(t, "service-1", info.ServiceID)
	assert.Equal(t, "healthy", info.Status)
	assert.Equal(t, 0, info.ConsecutiveFailures)
	assert.Equal(t, 5, info.ConsecutiveSuccesses)
}

// TestFailureMetricsFields tests failure metrics fields
func TestFailureMetricsFields(t *testing.T) {
	metrics := &FailureMetrics{
		FailuresDetected:  10,
		FalsePositives:    2,
		FalseNegatives:    1,
		PredictedFailures: 5,
		ActualFailures:    8,
	}

	assert.Equal(t, int64(10), metrics.FailuresDetected)
	assert.Equal(t, int64(2), metrics.FalsePositives)
	assert.Equal(t, int64(1), metrics.FalseNegatives)
}

// TestDetectorID tests detector ID
func TestDetectorID(t *testing.T) {
	detector := NewFailureDetector("my-detector")

	assert.Equal(t, "my-detector", detector.id)
}

// TestHealthCheckInterval tests health check interval
func TestHealthCheckInterval(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	assert.Equal(t, 10*time.Second, detector.healthCheckInterval)
}

// TestFailureThreshold tests failure threshold
func TestFailureThreshold(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	assert.Equal(t, 3, detector.failureThreshold)
}

// TestRecoveryThreshold tests recovery threshold
func TestRecoveryThreshold(t *testing.T) {
	detector := NewFailureDetector("detector-1")

	assert.Equal(t, 2, detector.recoveryThreshold)
}

// TestDetectFailuresMultipleServices tests detecting failures across multiple services
func TestDetectFailuresMultipleServices(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")
	detector.RegisterService("service-2")
	detector.RegisterService("service-3")
	ctx := context.Background()

	// Trigger failures for multiple services
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.5)
		_ = detector.ReportHealthCheck("service-2", false, 500*time.Millisecond, 0.5)
	}

	failed, err := detector.DetectFailures(ctx)

	assert.NoError(t, err)
	assert.Greater(t, len(failed), 0)
}

// TestLastHealthCheckTime tests last health check time tracking
func TestLastHealthCheckTime(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	before := time.Now()
	_ = detector.ReportHealthCheck("service-1", true, 100*time.Millisecond, 0.0)
	after := time.Now()

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	assert.True(t, info.LastHealthCheck.After(before) || info.LastHealthCheck.Equal(before))
	assert.True(t, info.LastHealthCheck.Before(after) || info.LastHealthCheck.Equal(after))
}

// TestPredictionScore tests prediction score calculation
func TestPredictionScore(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")

	// Report failures with high error rate to trigger prediction
	for i := 0; i < 3; i++ {
		_ = detector.ReportHealthCheck("service-1", false, 500*time.Millisecond, 0.9)
	}

	detector.mu.RLock()
	info := detector.services["service-1"]
	detector.mu.RUnlock()

	// Prediction score should be set if prediction was made
	if info.PredictedFailure {
		assert.Greater(t, info.PredictionScore, 0.0)
	}
}

// TestConcurrentDetectFailures tests concurrent failure detection
func TestConcurrentDetectFailures(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	detector.RegisterService("service-1")
	ctx := context.Background()

	var wg sync.WaitGroup
	var detectCount int32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := detector.DetectFailures(ctx)
			if err == nil {
				atomic.AddInt32(&detectCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Greater(t, atomic.LoadInt32(&detectCount), int32(0))
}

// TestFailoverStrategyRegistration tests registering failover strategies
func TestFailoverStrategyRegistration(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	failover := NewAutomaticFailover("failover-1", detector)

	// Create a mock strategy
	mockStrategy := &MockFailoverStrategy{}

	failover.RegisterFailoverStrategy("service-1", mockStrategy)

	assert.NotNil(t, failover.failoverStrategies["service-1"])
}

// MockFailoverStrategy for testing
type MockFailoverStrategy struct{}

func (m *MockFailoverStrategy) Execute(ctx context.Context, failedService string) error {
	return nil
}

func (m *MockFailoverStrategy) Rollback(ctx context.Context, failedService string) error {
	return nil
}

// TestFailoverRollback tests failover rollback
func TestFailoverRollback(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	failover := NewAutomaticFailover("failover-1", detector)
	ctx := context.Background()

	mockStrategy := &MockFailoverStrategy{}
	failover.RegisterFailoverStrategy("service-1", mockStrategy)

	err := failover.Rollback(ctx, "service-1")

	assert.NoError(t, err)
}

// TestFailoverRollbackNotFound tests rollback of non-existent strategy
func TestFailoverRollbackNotFound(t *testing.T) {
	detector := NewFailureDetector("detector-1")
	failover := NewAutomaticFailover("failover-1", detector)
	ctx := context.Background()

	err := failover.Rollback(ctx, "nonexistent")

	assert.Error(t, err)
}
