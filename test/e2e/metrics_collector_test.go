package e2e

import (
	"testing"
	"time"
)

func TestMetricsCollectorRecordMetric(t *testing.T) {
	mc := NewMetricsCollector("test_metric_recording")

	labels := map[string]string{"operation": "query"}
	mc.RecordMetric(MetricTypeLatency, 100.0, labels)

	metrics := mc.GetMetrics(MetricTypeLatency)
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Value != 100.0 {
		t.Errorf("Expected value 100.0, got %f", metrics[0].Value)
	}

	if metrics[0].MetricType != MetricTypeLatency {
		t.Errorf("Expected MetricTypeLatency, got %v", metrics[0].MetricType)
	}
}

func TestMetricsCollectorRecordLatency(t *testing.T) {
	mc := NewMetricsCollector("test_latency_recording")

	duration := 50 * time.Millisecond
	labels := map[string]string{"operation": "insert"}
	mc.RecordLatency(duration, labels)

	metrics := mc.GetMetrics(MetricTypeLatency)
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	expectedMicros := float64(duration.Microseconds())
	if metrics[0].Value != expectedMicros {
		t.Errorf("Expected %f microseconds, got %f", expectedMicros, metrics[0].Value)
	}
}

func TestMetricsCollectorRecordThroughput(t *testing.T) {
	mc := NewMetricsCollector("test_throughput_recording")

	opsPerSec := 1000.0
	labels := map[string]string{"operation": "batch"}
	mc.RecordThroughput(opsPerSec, labels)

	metrics := mc.GetMetrics(MetricTypeThroughput)
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Value != opsPerSec {
		t.Errorf("Expected %f ops/sec, got %f", opsPerSec, metrics[0].Value)
	}
}

func TestMetricsCollectorRecordErrorRate(t *testing.T) {
	mc := NewMetricsCollector("test_error_rate_recording")

	errorRate := 5.0
	labels := map[string]string{"operation": "query"}
	mc.RecordErrorRate(errorRate, labels)

	metrics := mc.GetMetrics(MetricTypeErrorRate)
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Value != errorRate {
		t.Errorf("Expected %f%% error rate, got %f", errorRate, metrics[0].Value)
	}
}

func TestMetricsCollectorSuccessAndErrorCounting(t *testing.T) {
	mc := NewMetricsCollector("test_success_error_counting")

	for i := 0; i < 10; i++ {
		mc.RecordSuccess()
	}

	for i := 0; i < 2; i++ {
		mc.RecordError()
	}

	if mc.successCount != 10 {
		t.Errorf("Expected 10 successes, got %d", mc.successCount)
	}

	if mc.errorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", mc.errorCount)
	}

	errorRate := mc.GetErrorRate()
	expectedErrorRate := 2.0 / 12.0 * 100
	// Allow small floating point tolerance (0.1%)
	tolerance := 0.1
	if errorRate < expectedErrorRate-tolerance || errorRate > expectedErrorRate+tolerance {
		t.Errorf("Expected error rate %.2f%%, got %.2f%%", expectedErrorRate, errorRate)
	}
}

func TestMetricsCollectorAggregation(t *testing.T) {
	mc := NewMetricsCollector("test_aggregation")

	// Record multiple latency measurements
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	for _, v := range values {
		mc.RecordMetric(MetricTypeLatency, v, nil)
	}

	agg := mc.Aggregate(MetricTypeLatency)

	if agg.Count != 10 {
		t.Errorf("Expected count 10, got %d", agg.Count)
	}

	if agg.Min != 10 {
		t.Errorf("Expected min 10, got %f", agg.Min)
	}

	if agg.Max != 100 {
		t.Errorf("Expected max 100, got %f", agg.Max)
	}

	expectedAvg := 55.0
	if agg.Avg != expectedAvg {
		t.Errorf("Expected avg %.2f, got %.2f", expectedAvg, agg.Avg)
	}

	if agg.Sum != 550 {
		t.Errorf("Expected sum 550, got %f", agg.Sum)
	}
}

func TestMetricsCollectorPercentiles(t *testing.T) {
	mc := NewMetricsCollector("test_percentiles")

	// Record 100 values from 1 to 100
	for i := 1; i <= 100; i++ {
		mc.RecordMetric(MetricTypeLatency, float64(i), nil)
	}

	agg := mc.Aggregate(MetricTypeLatency)

	// P50 should be around 50
	if agg.P50 < 45 || agg.P50 > 55 {
		t.Errorf("Expected P50 around 50, got %f", agg.P50)
	}

	// P95 should be around 95
	if agg.P95 < 90 || agg.P95 > 100 {
		t.Errorf("Expected P95 around 95, got %f", agg.P95)
	}

	// P99 should be around 99
	if agg.P99 < 95 || agg.P99 > 100 {
		t.Errorf("Expected P99 around 99, got %f", agg.P99)
	}
}

func TestMetricsCollectorGetAggregation(t *testing.T) {
	mc := NewMetricsCollector("test_get_aggregation")

	mc.RecordMetric(MetricTypeLatency, 50, nil)
	mc.Aggregate(MetricTypeLatency)

	agg := mc.GetAggregation(MetricTypeLatency)
	if agg == nil {
		t.Fatalf("Expected aggregation, got nil")
	}

	if agg.Count != 1 {
		t.Errorf("Expected count 1, got %d", agg.Count)
	}
}

func TestMetricsCollectorDuration(t *testing.T) {
	mc := NewMetricsCollector("test_duration")

	time.Sleep(10 * time.Millisecond)
	mc.Finalize()

	duration := mc.GetDuration()
	if duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", duration)
	}
}

func TestMetricsCollectorTestStatus(t *testing.T) {
	mc := NewMetricsCollector("test_status")

	mc.SetTestStatus("PASSED")
	if mc.testStatus != "PASSED" {
		t.Errorf("Expected status PASSED, got %s", mc.testStatus)
	}

	mc.SetTestStatus("FAILED")
	if mc.testStatus != "FAILED" {
		t.Errorf("Expected status FAILED, got %s", mc.testStatus)
	}
}

func TestMetricsCollectorGetStats(t *testing.T) {
	mc := NewMetricsCollector("test_stats")

	mc.RecordSuccess()
	mc.RecordSuccess()
	mc.RecordError()
	mc.SetTestStatus("PASSED")

	stats := mc.GetStats()

	if stats["test_name"] != "test_stats" {
		t.Errorf("Expected test_name 'test_stats', got %v", stats["test_name"])
	}

	if stats["success_count"] != int64(2) {
		t.Errorf("Expected success_count 2, got %v", stats["success_count"])
	}

	if stats["error_count"] != int64(1) {
		t.Errorf("Expected error_count 1, got %v", stats["error_count"])
	}
}

func TestTestReportGeneratorRegisterCollector(t *testing.T) {
	trg := NewTestReportGenerator()

	mc1 := NewMetricsCollector("test1")
	mc1.SetTestStatus("PASSED")
	mc1.RecordSuccess()

	mc2 := NewMetricsCollector("test2")
	mc2.SetTestStatus("FAILED")
	mc2.RecordError()

	trg.RegisterCollector("test1", mc1)
	trg.RegisterCollector("test2", mc2)

	if len(trg.collectors) != 2 {
		t.Errorf("Expected 2 collectors, got %d", len(trg.collectors))
	}
}

func TestTestReportGeneratorGenerateReport(t *testing.T) {
	trg := NewTestReportGenerator()

	mc := NewMetricsCollector("test_report")
	mc.SetTestStatus("PASSED")
	mc.RecordSuccess()
	mc.RecordSuccess()
	mc.RecordError()

	// Record some metrics
	mc.RecordLatency(100*time.Microsecond, nil)
	mc.RecordLatency(200*time.Microsecond, nil)
	mc.RecordThroughput(1000, nil)

	trg.RegisterCollector("test_report", mc)

	report := trg.GenerateReport()

	if report == "" {
		t.Error("Expected non-empty report")
	}

	if !containsString(report, "test_report") {
		t.Error("Expected report to contain test name")
	}

	if !containsString(report, "PASSED") {
		t.Error("Expected report to contain test status")
	}

	if !containsString(report, "Summary") {
		t.Error("Expected report to contain summary section")
	}
}

func TestMetricsCollectorMultipleMetricTypes(t *testing.T) {
	mc := NewMetricsCollector("test_multiple_types")

	// Record different metric types
	mc.RecordLatency(100*time.Microsecond, nil)
	mc.RecordThroughput(1000, nil)
	mc.RecordErrorRate(5.0, nil)

	latencyMetrics := mc.GetMetrics(MetricTypeLatency)
	throughputMetrics := mc.GetMetrics(MetricTypeThroughput)
	errorRateMetrics := mc.GetMetrics(MetricTypeErrorRate)

	if len(latencyMetrics) != 1 {
		t.Errorf("Expected 1 latency metric, got %d", len(latencyMetrics))
	}

	if len(throughputMetrics) != 1 {
		t.Errorf("Expected 1 throughput metric, got %d", len(throughputMetrics))
	}

	if len(errorRateMetrics) != 1 {
		t.Errorf("Expected 1 error rate metric, got %d", len(errorRateMetrics))
	}
}

func TestMetricsCollectorConcurrentRecording(t *testing.T) {
	mc := NewMetricsCollector("test_concurrent")

	done := make(chan bool)

	// Record metrics concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				mc.RecordMetric(MetricTypeLatency, float64(id*10+j), nil)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := mc.GetMetrics(MetricTypeLatency)
	if len(metrics) != 100 {
		t.Errorf("Expected 100 metrics, got %d", len(metrics))
	}
}

func TestMetricsCollectorEmptyMetrics(t *testing.T) {
	mc := NewMetricsCollector("test_empty")

	agg := mc.Aggregate(MetricTypeLatency)

	if agg.Count != 0 {
		t.Errorf("Expected count 0, got %d", agg.Count)
	}

	if agg.Min != 0 {
		t.Errorf("Expected min 0, got %f", agg.Min)
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
