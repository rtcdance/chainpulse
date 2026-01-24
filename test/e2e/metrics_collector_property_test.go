package e2e

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// Property 23: Metrics Accuracy
// Validates that metrics are accurately recorded and aggregated
func TestPropertyMetricsAccuracy(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_metrics_accuracy_%d", iteration))

		// Generate random metrics
		numMetrics := 10 + (iteration % 90)
		expectedSum := 0.0
		expectedMin := math.MaxFloat64
		expectedMax := -math.MaxFloat64

		for i := 0; i < numMetrics; i++ {
			value := float64(i*10 + 1)
			mc.RecordMetric(MetricTypeLatency, value, nil)
			expectedSum += value

			if value < expectedMin {
				expectedMin = value
			}
			if value > expectedMax {
				expectedMax = value
			}
		}

		agg := mc.Aggregate(MetricTypeLatency)

		// Verify accuracy
		if agg.Count != int64(numMetrics) {
			t.Errorf("Iteration %d: Expected count %d, got %d", iteration, numMetrics, agg.Count)
		}

		if agg.Sum != expectedSum {
			t.Errorf("Iteration %d: Expected sum %.2f, got %.2f", iteration, expectedSum, agg.Sum)
		}

		if agg.Min != expectedMin {
			t.Errorf("Iteration %d: Expected min %.2f, got %.2f", iteration, expectedMin, agg.Min)
		}

		if agg.Max != expectedMax {
			t.Errorf("Iteration %d: Expected max %.2f, got %.2f", iteration, expectedMax, agg.Max)
		}

		expectedAvg := expectedSum / float64(numMetrics)
		if math.Abs(agg.Avg-expectedAvg) > 0.01 {
			t.Errorf("Iteration %d: Expected avg %.2f, got %.2f", iteration, expectedAvg, agg.Avg)
		}
	}
}

// Property 24: Percentile Ordering
// Validates that percentiles are correctly ordered (P50 <= P95 <= P99)
func TestPropertyPercentileOrdering(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_percentile_ordering_%d", iteration))

		// Generate metrics
		for i := 1; i <= 100; i++ {
			mc.RecordMetric(MetricTypeLatency, float64(i), nil)
		}

		agg := mc.Aggregate(MetricTypeLatency)

		// Verify percentile ordering
		if agg.P50 > agg.P95 {
			t.Errorf("Iteration %d: P50 (%.2f) > P95 (%.2f)", iteration, agg.P50, agg.P95)
		}

		if agg.P95 > agg.P99 {
			t.Errorf("Iteration %d: P95 (%.2f) > P99 (%.2f)", iteration, agg.P95, agg.P99)
		}

		if agg.Min > agg.P50 {
			t.Errorf("Iteration %d: Min (%.2f) > P50 (%.2f)", iteration, agg.Min, agg.P50)
		}

		if agg.P99 > agg.Max {
			t.Errorf("Iteration %d: P99 (%.2f) > Max (%.2f)", iteration, agg.P99, agg.Max)
		}
	}
}

// Property 25: Error Rate Consistency
// Validates that error rate calculations are consistent
func TestPropertyMetricsErrorRateConsistency(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_error_rate_consistency_%d", iteration))

		successCount := 80 + (iteration % 20)
		errorCount := 10 + (iteration % 10)

		for i := 0; i < successCount; i++ {
			mc.RecordSuccess()
		}

		for i := 0; i < errorCount; i++ {
			mc.RecordError()
		}

		errorRate := mc.GetErrorRate()
		successRate := mc.GetSuccessRate()

		expectedErrorRate := float64(errorCount) / float64(successCount+errorCount) * 100
		expectedSuccessRate := 100 - expectedErrorRate

		if math.Abs(errorRate-expectedErrorRate) > 0.01 {
			t.Errorf("Iteration %d: Expected error rate %.2f%%, got %.2f%%", iteration, expectedErrorRate, errorRate)
		}

		if math.Abs(successRate-expectedSuccessRate) > 0.01 {
			t.Errorf("Iteration %d: Expected success rate %.2f%%, got %.2f%%", iteration, expectedSuccessRate, successRate)
		}

		if math.Abs(errorRate+successRate-100) > 0.01 {
			t.Errorf("Iteration %d: Error rate + success rate should equal 100, got %.2f", iteration, errorRate+successRate)
		}
	}
}

// Property 26: Metric Type Isolation
// Validates that different metric types don't interfere with each other
func TestPropertyMetricTypeIsolation(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_metric_type_isolation_%d", iteration))

		// Record different metric types
		latencyCount := 10 + (iteration % 20)
		throughputCount := 5 + (iteration % 15)
		errorRateCount := 3 + (iteration % 7)

		for i := 0; i < latencyCount; i++ {
			mc.RecordLatency(time.Duration(i*10)*time.Microsecond, nil)
		}

		for i := 0; i < throughputCount; i++ {
			mc.RecordThroughput(float64(i*100), nil)
		}

		for i := 0; i < errorRateCount; i++ {
			mc.RecordErrorRate(float64(i*5), nil)
		}

		latencyMetrics := mc.GetMetrics(MetricTypeLatency)
		throughputMetrics := mc.GetMetrics(MetricTypeThroughput)
		errorRateMetrics := mc.GetMetrics(MetricTypeErrorRate)

		// Verify isolation
		if len(latencyMetrics) != latencyCount {
			t.Errorf("Iteration %d: Expected %d latency metrics, got %d", iteration, latencyCount, len(latencyMetrics))
		}

		if len(throughputMetrics) != throughputCount {
			t.Errorf("Iteration %d: Expected %d throughput metrics, got %d", iteration, throughputCount, len(throughputMetrics))
		}

		if len(errorRateMetrics) != errorRateCount {
			t.Errorf("Iteration %d: Expected %d error rate metrics, got %d", iteration, errorRateCount, len(errorRateMetrics))
		}

		// Verify no cross-contamination
		for _, m := range latencyMetrics {
			if m.MetricType != MetricTypeLatency {
				t.Errorf("Iteration %d: Found non-latency metric in latency metrics", iteration)
			}
		}

		for _, m := range throughputMetrics {
			if m.MetricType != MetricTypeThroughput {
				t.Errorf("Iteration %d: Found non-throughput metric in throughput metrics", iteration)
			}
		}

		for _, m := range errorRateMetrics {
			if m.MetricType != MetricTypeErrorRate {
				t.Errorf("Iteration %d: Found non-error-rate metric in error rate metrics", iteration)
			}
		}
	}
}

// Property 27: Aggregation Idempotence
// Validates that calling Aggregate multiple times produces consistent results
func TestPropertyAggregationIdempotence(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_aggregation_idempotence_%d", iteration))

		// Record metrics
		for i := 1; i <= 50; i++ {
			mc.RecordMetric(MetricTypeLatency, float64(i), nil)
		}

		// Aggregate multiple times
		agg1 := mc.Aggregate(MetricTypeLatency)
		agg2 := mc.Aggregate(MetricTypeLatency)
		agg3 := mc.Aggregate(MetricTypeLatency)

		// Verify consistency
		if agg1.Min != agg2.Min || agg2.Min != agg3.Min {
			t.Errorf("Iteration %d: Min values differ: %.2f, %.2f, %.2f", iteration, agg1.Min, agg2.Min, agg3.Min)
		}

		if agg1.Max != agg2.Max || agg2.Max != agg3.Max {
			t.Errorf("Iteration %d: Max values differ: %.2f, %.2f, %.2f", iteration, agg1.Max, agg2.Max, agg3.Max)
		}

		if agg1.Avg != agg2.Avg || agg2.Avg != agg3.Avg {
			t.Errorf("Iteration %d: Avg values differ: %.2f, %.2f, %.2f", iteration, agg1.Avg, agg2.Avg, agg3.Avg)
		}

		if agg1.Count != agg2.Count || agg2.Count != agg3.Count {
			t.Errorf("Iteration %d: Count values differ: %d, %d, %d", iteration, agg1.Count, agg2.Count, agg3.Count)
		}
	}
}

// Property 28: Duration Monotonicity
// Validates that duration increases monotonically
func TestPropertyDurationMonotonicity(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_duration_monotonicity_%d", iteration))

		duration1 := mc.GetDuration()
		time.Sleep(1 * time.Millisecond)
		duration2 := mc.GetDuration()
		time.Sleep(1 * time.Millisecond)
		duration3 := mc.GetDuration()

		// Verify monotonicity
		if duration1 > duration2 {
			t.Errorf("Iteration %d: Duration decreased: %v > %v", iteration, duration1, duration2)
		}

		if duration2 > duration3 {
			t.Errorf("Iteration %d: Duration decreased: %v > %v", iteration, duration2, duration3)
		}
	}
}

// Property 29: Report Generation Completeness
// Validates that generated reports contain all expected information
func TestPropertyReportGenerationCompleteness(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		trg := NewTestReportGenerator()

		numTests := 2 + (iteration % 5)

		for i := 0; i < numTests; i++ {
			mc := NewMetricsCollector(fmt.Sprintf("test_%d_%d", iteration, i))
			mc.SetTestStatus("PASSED")
			mc.RecordSuccess()
			mc.RecordLatency(time.Duration(i*10)*time.Microsecond, nil)
			mc.RecordThroughput(float64(i*100), nil)
			trg.RegisterCollector(fmt.Sprintf("test_%d_%d", iteration, i), mc)
		}

		report := trg.GenerateReport()

		// Verify report contains expected sections
		if !contains(report, "E2E Test Report") {
			t.Errorf("Iteration %d: Report missing header", iteration)
		}

		if !contains(report, "Summary") {
			t.Errorf("Iteration %d: Report missing summary section", iteration)
		}

		if !contains(report, "Total Tests") {
			t.Errorf("Iteration %d: Report missing total tests", iteration)
		}

		if !contains(report, "Passed") {
			t.Errorf("Iteration %d: Report missing passed count", iteration)
		}

		if !contains(report, "Failed") {
			t.Errorf("Iteration %d: Report missing failed count", iteration)
		}
	}
}

// Property 30: Concurrent Metric Recording Safety
// Validates that concurrent metric recording is thread-safe
func TestPropertyConcurrentMetricRecordingSafety(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		mc := NewMetricsCollector(fmt.Sprintf("test_concurrent_safety_%d", iteration))

		done := make(chan bool)
		numGoroutines := 10 + (iteration % 20)
		metricsPerGoroutine := 10 + (iteration % 20)

		// Record metrics concurrently
		for g := 0; g < numGoroutines; g++ {
			go func(id int) {
				for i := 0; i < metricsPerGoroutine; i++ {
					mc.RecordMetric(MetricTypeLatency, float64(id*metricsPerGoroutine+i), nil)
					mc.RecordSuccess()
				}
				done <- true
			}(g)
		}

		// Wait for all goroutines
		for g := 0; g < numGoroutines; g++ {
			<-done
		}

		// Verify all metrics were recorded
		metrics := mc.GetMetrics(MetricTypeLatency)
		expectedCount := numGoroutines * metricsPerGoroutine

		if len(metrics) != expectedCount {
			t.Errorf("Iteration %d: Expected %d metrics, got %d", iteration, expectedCount, len(metrics))
		}

		if mc.successCount != int64(expectedCount) {
			t.Errorf("Iteration %d: Expected %d successes, got %d", iteration, expectedCount, mc.successCount)
		}
	}
}
