package query

import (
	"context"
	"testing"
	"time"
)

func TestErrorMetricsCollectorInitialization(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)

	err := collector.Initialize(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Initialize again should not error
	err = collector.Initialize(ctx)
	if err != nil {
		t.Fatalf("Expected no error on second initialize, got %v", err)
	}
}

func TestRecordError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)

	errorMetrics := collector.GetErrorMetrics(ctx)
	if errorMetrics.TotalErrors != 1 {
		t.Fatalf("Expected 1 total error, got %d", errorMetrics.TotalErrors)
	}

	if errorMetrics.TransientErrors != 1 {
		t.Fatalf("Expected 1 transient error, got %d", errorMetrics.TransientErrors)
	}

	if errorMetrics.ErrorsBySource["mongodb"] != 1 {
		t.Fatalf("Expected 1 mongodb error, got %d", errorMetrics.ErrorsBySource["mongodb"])
	}
}

func TestRecordErrorTypes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)
	collector.RecordError(ctx, "permanent", "postgresql", 50*time.Millisecond)
	collector.RecordError(ctx, "critical", "cache", 200*time.Millisecond)

	errorMetrics := collector.GetErrorMetrics(ctx)
	if errorMetrics.TotalErrors != 3 {
		t.Fatalf("Expected 3 total errors, got %d", errorMetrics.TotalErrors)
	}

	if errorMetrics.TransientErrors != 1 {
		t.Fatalf("Expected 1 transient error, got %d", errorMetrics.TransientErrors)
	}

	if errorMetrics.PermanentErrors != 1 {
		t.Fatalf("Expected 1 permanent error, got %d", errorMetrics.PermanentErrors)
	}

	if errorMetrics.CriticalErrors != 1 {
		t.Fatalf("Expected 1 critical error, got %d", errorMetrics.CriticalErrors)
	}
}

func TestRecordErrorSources(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)
	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)
	collector.RecordError(ctx, "transient", "postgresql", 100*time.Millisecond)

	errorMetrics := collector.GetErrorMetrics(ctx)
	if errorMetrics.ErrorsBySource["mongodb"] != 2 {
		t.Fatalf("Expected 2 mongodb errors, got %d", errorMetrics.ErrorsBySource["mongodb"])
	}

	if errorMetrics.ErrorsBySource["postgresql"] != 1 {
		t.Fatalf("Expected 1 postgresql error, got %d", errorMetrics.ErrorsBySource["postgresql"])
	}
}

func TestRecordRetryAttempt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordRetryAttempt(ctx, 1, true)
	collector.RecordRetryAttempt(ctx, 2, false)
	collector.RecordRetryAttempt(ctx, 1, true)

	retryMetrics := collector.GetRetryMetrics(ctx)
	if retryMetrics.TotalAttempts != 3 {
		t.Fatalf("Expected 3 total attempts, got %d", retryMetrics.TotalAttempts)
	}

	if retryMetrics.SuccessfulRetries != 2 {
		t.Fatalf("Expected 2 successful retries, got %d", retryMetrics.SuccessfulRetries)
	}

	if retryMetrics.FailedRetries != 1 {
		t.Fatalf("Expected 1 failed retry, got %d", retryMetrics.FailedRetries)
	}
}

func TestRetrySuccessRate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordRetryAttempt(ctx, 1, true)
	collector.RecordRetryAttempt(ctx, 2, true)
	collector.RecordRetryAttempt(ctx, 1, false)

	retryMetrics := collector.GetRetryMetrics(ctx)
	expectedRate := 2.0 / 3.0
	if retryMetrics.RetrySuccessRate < expectedRate-0.01 || retryMetrics.RetrySuccessRate > expectedRate+0.01 {
		t.Fatalf("Expected success rate ~%.2f, got %.2f", expectedRate, retryMetrics.RetrySuccessRate)
	}
}

func TestRecordCircuitBreakerStateChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordCircuitBreakerStateChange(ctx, "mongodb", "Closed", "Open")
	collector.RecordCircuitBreakerStateChange(ctx, "mongodb", "Open", "HalfOpen")
	collector.RecordCircuitBreakerStateChange(ctx, "postgresql", "Closed", "Open")

	cbMetrics := collector.GetCircuitBreakerMetrics(ctx)
	if cbMetrics.TotalStateChanges != 3 {
		t.Fatalf("Expected 3 state changes, got %d", cbMetrics.TotalStateChanges)
	}

	if cbMetrics.OpenCount != 2 {
		t.Fatalf("Expected 2 open states, got %d", cbMetrics.OpenCount)
	}

	if cbMetrics.HalfOpenCount != 1 {
		t.Fatalf("Expected 1 half-open state, got %d", cbMetrics.HalfOpenCount)
	}

	if cbMetrics.CurrentState["mongodb"] != "HalfOpen" {
		t.Fatalf("Expected mongodb state HalfOpen, got %s", cbMetrics.CurrentState["mongodb"])
	}
}

func TestRecordConsistencyCheck(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordConsistencyCheck(ctx, true, 100*time.Millisecond)
	collector.RecordConsistencyCheck(ctx, true, 150*time.Millisecond)
	collector.RecordConsistencyCheck(ctx, false, 200*time.Millisecond)

	consistencyMetrics := collector.GetConsistencyMetrics(ctx)
	if consistencyMetrics.TotalChecks != 3 {
		t.Fatalf("Expected 3 total checks, got %d", consistencyMetrics.TotalChecks)
	}

	if consistencyMetrics.PassedChecks != 2 {
		t.Fatalf("Expected 2 passed checks, got %d", consistencyMetrics.PassedChecks)
	}

	if consistencyMetrics.FailedChecks != 1 {
		t.Fatalf("Expected 1 failed check, got %d", consistencyMetrics.FailedChecks)
	}
}

func TestConsistencyPassRate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordConsistencyCheck(ctx, true, 100*time.Millisecond)
	collector.RecordConsistencyCheck(ctx, true, 100*time.Millisecond)
	collector.RecordConsistencyCheck(ctx, false, 100*time.Millisecond)
	collector.RecordConsistencyCheck(ctx, true, 100*time.Millisecond)

	consistencyMetrics := collector.GetConsistencyMetrics(ctx)
	expectedRate := 3.0 / 4.0
	if consistencyMetrics.PassRate < expectedRate-0.01 || consistencyMetrics.PassRate > expectedRate+0.01 {
		t.Fatalf("Expected pass rate ~%.2f, got %.2f", expectedRate, consistencyMetrics.PassRate)
	}
}

func TestRecordDegradationEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordDegradationEvent(ctx, "Normal", "")
	collector.RecordDegradationEvent(ctx, "MongoDBUnavailable", "Connection timeout")
	collector.RecordDegradationEvent(ctx, "BothUnavailable", "Both stores down")

	degradationMetrics := collector.GetDegradationMetrics(ctx)
	if degradationMetrics.TotalEvents != 3 {
		t.Fatalf("Expected 3 total events, got %d", degradationMetrics.TotalEvents)
	}

	if degradationMetrics.NormalMode != 1 {
		t.Fatalf("Expected 1 normal mode event, got %d", degradationMetrics.NormalMode)
	}

	if degradationMetrics.MongoDBUnavailable != 1 {
		t.Fatalf("Expected 1 mongodb unavailable event, got %d", degradationMetrics.MongoDBUnavailable)
	}

	if degradationMetrics.BothUnavailable != 1 {
		t.Fatalf("Expected 1 both unavailable event, got %d", degradationMetrics.BothUnavailable)
	}
}

func TestGetErrorMetricsReturnsACopy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)

	errorMetrics1 := collector.GetErrorMetrics(ctx)
	errorMetrics1.ErrorsBySource["mongodb"] = 999

	errorMetrics2 := collector.GetErrorMetrics(ctx)
	if errorMetrics2.ErrorsBySource["mongodb"] != 1 {
		t.Fatal("Expected copy to be independent")
	}
}

func TestHealthStatusInitialized(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	health := collector.Health(ctx)
	if health.Status != "healthy" {
		t.Fatalf("Expected healthy, got %v", health.Status)
	}
}

func TestHealthStatusNotInitialized(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)

	health := collector.Health(ctx)
	if health.Status != "unhealthy" {
		t.Fatalf("Expected unhealthy, got %v", health.Status)
	}
}

func TestClose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	err := collector.Close(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	health := collector.Health(ctx)
	if health.Status != "unhealthy" {
		t.Fatalf("Expected unhealthy after close, got %v", health.Status)
	}
}

func TestAverageDuration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)
	collector.RecordError(ctx, "transient", "mongodb", 200*time.Millisecond)
	collector.RecordError(ctx, "transient", "mongodb", 300*time.Millisecond)

	errorMetrics := collector.GetErrorMetrics(ctx)
	expectedAvg := 200 * time.Millisecond
	if errorMetrics.AverageDuration != expectedAvg {
		t.Fatalf("Expected average duration %v, got %v", expectedAvg, errorMetrics.AverageDuration)
	}
}

func TestAverageAttempts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordRetryAttempt(ctx, 1, true)
	collector.RecordRetryAttempt(ctx, 2, true)
	collector.RecordRetryAttempt(ctx, 3, true)

	retryMetrics := collector.GetRetryMetrics(ctx)
	expectedAvg := 2.0
	if retryMetrics.AverageAttempts != expectedAvg {
		t.Fatalf("Expected average attempts %.1f, got %.1f", expectedAvg, retryMetrics.AverageAttempts)
	}
}

func TestMultipleConcurrentRecords(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	// Record multiple errors concurrently
	for i := 0; i < 10; i++ {
		go func() {
			collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)
		}()
	}

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)

	errorMetrics := collector.GetErrorMetrics(ctx)
	if errorMetrics.TotalErrors < 10 {
		t.Fatalf("Expected at least 10 errors, got %d", errorMetrics.TotalErrors)
	}
}

func TestLastEventTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	before := time.Now()
	collector.RecordError(ctx, "transient", "mongodb", 100*time.Millisecond)
	after := time.Now()

	errorMetrics := collector.GetErrorMetrics(ctx)
	if errorMetrics.LastErrorTime.Before(before) || errorMetrics.LastErrorTime.After(after.Add(1*time.Second)) {
		t.Fatal("Expected LastErrorTime to be recent")
	}
}

func TestErrorMetricsUnknownType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	collector.RecordError(ctx, "unknown_type", "mongodb", 100*time.Millisecond)

	errorMetrics := collector.GetErrorMetrics(ctx)
	if errorMetrics.UnknownErrors != 1 {
		t.Fatalf("Expected 1 unknown error, got %d", errorMetrics.UnknownErrors)
	}
}

func TestDegradationAllModes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	collector := NewErrorMetricsCollector(logger, metrics)
	_ = collector.Initialize(ctx)

	modes := []string{"Normal", "MongoDBUnavailable", "PostgreSQLUnavailable", "BothUnavailable", "CacheUnavailable", "ReadOnly"}
	for _, mode := range modes {
		collector.RecordDegradationEvent(ctx, mode, "test")
	}

	degradationMetrics := collector.GetDegradationMetrics(ctx)
	if degradationMetrics.TotalEvents != 6 {
		t.Fatalf("Expected 6 total events, got %d", degradationMetrics.TotalEvents)
	}

	if degradationMetrics.NormalMode != 1 {
		t.Fatalf("Expected 1 normal mode, got %d", degradationMetrics.NormalMode)
	}

	if degradationMetrics.ReadOnlyMode != 1 {
		t.Fatalf("Expected 1 read-only mode, got %d", degradationMetrics.ReadOnlyMode)
	}
}
