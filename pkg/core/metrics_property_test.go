package core

import (
	"testing"
)

// Property 22: Metrics Emission
// The system SHALL emit metrics for all operations with proper aggregation and tagging.

// TestProperty22_CounterAccumulation tests that counters properly accumulate
func TestProperty22_CounterAccumulation(t *testing.T) {
	// Property: Counter values must accumulate correctly
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	testCases := []struct {
		increments []int64
		expected   int64
	}{
		{[]int64{1, 2, 3}, 6},
		{[]int64{10, 20, 30}, 60},
		{[]int64{-5, 10, -3}, 2},
		{[]int64{0, 0, 0}, 0},
	}

	for _, tc := range testCases {
		collector.Reset()
		for _, inc := range tc.increments {
			collector.RecordCounter("requests", inc, tags)
		}

		value := collector.GetCounter("requests", tags)
		if value != tc.expected {
			t.Errorf("expected counter %d, got %d", tc.expected, value)
		}
	}
}

// TestProperty22_GaugeReplacement tests that gauges replace previous values
func TestProperty22_GaugeReplacement(t *testing.T) {
	// Property: Gauge values must replace previous values, not accumulate
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	testCases := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{100.0, 200.0, 150.0}, 150.0},
		{[]float64{0.0, 50.0}, 50.0},
		{[]float64{-10.0, 10.0}, 10.0},
	}

	for _, tc := range testCases {
		collector.Reset()
		for _, val := range tc.values {
			collector.RecordGauge("memory", val, tags)
		}

		value := collector.GetGauge("memory", tags)
		if value != tc.expected {
			t.Errorf("expected gauge %f, got %f", tc.expected, value)
		}
	}
}

// TestProperty22_HistogramDataPreservation tests that histogram values are preserved
func TestProperty22_HistogramDataPreservation(t *testing.T) {
	// Property: All histogram values must be preserved for statistical analysis
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for _, v := range values {
		collector.RecordHistogram("latency", v, tags)
	}

	stats := collector.GetHistogramStats("latency", tags)

	if stats.Count != int64(len(values)) {
		t.Errorf("expected count %d, got %d", len(values), stats.Count)
	}

	expectedSum := 0.0
	for _, v := range values {
		expectedSum += v
	}
	if stats.Sum != expectedSum {
		t.Errorf("expected sum %f, got %f", expectedSum, stats.Sum)
	}
}

// TestProperty22_TagIsolation tests that metrics with different tags are isolated
func TestProperty22_TagIsolation(t *testing.T) {
	// Property: Metrics with different tags must not interfere with each other
	collector := NewDefaultMetricsCollector()
	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}
	tags3 := map[string]string{"service": "cache"}

	collector.RecordCounter("requests", 10, tags1)
	collector.RecordCounter("requests", 20, tags2)
	collector.RecordCounter("requests", 30, tags3)

	value1 := collector.GetCounter("requests", tags1)
	value2 := collector.GetCounter("requests", tags2)
	value3 := collector.GetCounter("requests", tags3)

	if value1 != 10 {
		t.Errorf("expected value1 10, got %d", value1)
	}
	if value2 != 20 {
		t.Errorf("expected value2 20, got %d", value2)
	}
	if value3 != 30 {
		t.Errorf("expected value3 30, got %d", value3)
	}
}

// TestProperty22_MetricsConsistency tests that metrics remain consistent
func TestProperty22_MetricsConsistency(t *testing.T) {
	// Property: Metrics must remain consistent across multiple reads
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 42, tags)

	value1 := collector.GetCounter("requests", tags)
	value2 := collector.GetCounter("requests", tags)
	value3 := collector.GetCounter("requests", tags)

	if value1 != value2 || value2 != value3 {
		t.Errorf("metrics inconsistent: %d, %d, %d", value1, value2, value3)
	}
}

// TestProperty22_ConcurrentMetricsRecording tests concurrent metric recording
func TestProperty22_ConcurrentMetricsRecording(t *testing.T) {
	// Property: Concurrent metric recording must not lose data
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	done := make(chan bool)
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			collector.RecordCounter("requests", 1, tags)
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	value := collector.GetCounter("requests", tags)
	if value != int64(numGoroutines) {
		t.Errorf("expected counter %d, got %d", numGoroutines, value)
	}
}

// TestProperty22_HistogramStatisticsAccuracy tests histogram statistics accuracy
func TestProperty22_HistogramStatisticsAccuracy(t *testing.T) {
	// Property: Histogram statistics must be calculated correctly
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	for _, v := range values {
		collector.RecordHistogram("latency", v, tags)
	}

	stats := collector.GetHistogramStats("latency", tags)

	// Verify basic statistics
	if stats.Count != 10 {
		t.Errorf("expected count 10, got %d", stats.Count)
	}
	if stats.Sum != 55.0 {
		t.Errorf("expected sum 55.0, got %f", stats.Sum)
	}
	if stats.Min != 1.0 {
		t.Errorf("expected min 1.0, got %f", stats.Min)
	}
	if stats.Max != 10.0 {
		t.Errorf("expected max 10.0, got %f", stats.Max)
	}
	if stats.Mean != 5.5 {
		t.Errorf("expected mean 5.5, got %f", stats.Mean)
	}
}

// TestProperty22_MetricsReset tests that reset clears all metrics
func TestProperty22_MetricsReset(t *testing.T) {
	// Property: Reset must clear all metrics completely
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 10, tags)
	collector.RecordGauge("memory", 512.0, tags)
	collector.RecordHistogram("latency", 50.0, tags)

	collector.Reset()

	if collector.GetCounterCount() != 0 {
		t.Error("expected 0 counters after reset")
	}
	if collector.GetGaugeCount() != 0 {
		t.Error("expected 0 gauges after reset")
	}
	if collector.GetHistogramCount() != 0 {
		t.Error("expected 0 histograms after reset")
	}
}

// TestProperty22_SelectiveReset tests that selective reset works correctly
func TestProperty22_SelectiveReset(t *testing.T) {
	// Property: Selective reset must only clear specified metrics
	collector := NewDefaultMetricsCollector()
	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}

	collector.RecordCounter("requests", 10, tags1)
	collector.RecordCounter("requests", 20, tags2)

	collector.ResetCounter("requests", tags1)

	value1 := collector.GetCounter("requests", tags1)
	value2 := collector.GetCounter("requests", tags2)

	if value1 != 0 {
		t.Errorf("expected value1 0 after reset, got %d", value1)
	}
	if value2 != 20 {
		t.Errorf("expected value2 20 after selective reset, got %d", value2)
	}
}

// TestProperty22_MultipleMetricTypes tests multiple metric types together
func TestProperty22_MultipleMetricTypes(t *testing.T) {
	// Property: Different metric types must not interfere with each other
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 100, tags)
	collector.RecordGauge("memory", 512.0, tags)
	collector.RecordHistogram("latency", 50.0, tags)

	counterValue := collector.GetCounter("requests", tags)
	gaugeValue := collector.GetGauge("memory", tags)
	histStats := collector.GetHistogramStats("latency", tags)

	if counterValue != 100 {
		t.Errorf("expected counter 100, got %d", counterValue)
	}
	if gaugeValue != 512.0 {
		t.Errorf("expected gauge 512.0, got %f", gaugeValue)
	}
	if histStats.Count != 1 {
		t.Errorf("expected histogram count 1, got %d", histStats.Count)
	}
}

// TestProperty22_LargeMetricValues tests handling of large metric values
func TestProperty22_LargeMetricValues(t *testing.T) {
	// Property: Large metric values must be handled correctly
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	largeCounter := int64(9223372036854775800) // Near max int64
	largeGauge := 1e15
	largeHistogram := 1e10

	collector.RecordCounter("requests", largeCounter, tags)
	collector.RecordGauge("memory", largeGauge, tags)
	collector.RecordHistogram("latency", largeHistogram, tags)

	counterValue := collector.GetCounter("requests", tags)
	gaugeValue := collector.GetGauge("memory", tags)
	histStats := collector.GetHistogramStats("latency", tags)

	if counterValue != largeCounter {
		t.Errorf("expected counter %d, got %d", largeCounter, counterValue)
	}
	if gaugeValue != largeGauge {
		t.Errorf("expected gauge %f, got %f", largeGauge, gaugeValue)
	}
	if histStats.Max != largeHistogram {
		t.Errorf("expected histogram max %f, got %f", largeHistogram, histStats.Max)
	}
}

// TestProperty22_EmptyMetrics tests behavior with empty metrics
func TestProperty22_EmptyMetrics(t *testing.T) {
	// Property: Empty metrics should return zero values
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	counterValue := collector.GetCounter("nonexistent", tags)
	gaugeValue := collector.GetGauge("nonexistent", tags)
	histStats := collector.GetHistogramStats("nonexistent", tags)

	if counterValue != 0 {
		t.Errorf("expected counter 0, got %d", counterValue)
	}
	if gaugeValue != 0.0 {
		t.Errorf("expected gauge 0.0, got %f", gaugeValue)
	}
	if histStats.Count != 0 {
		t.Errorf("expected histogram count 0, got %d", histStats.Count)
	}
}

// TestProperty22_MetricsExport tests that metrics can be exported
func TestProperty22_MetricsExport(t *testing.T) {
	// Property: All metrics must be exportable via GetMetrics
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 10, tags)
	collector.RecordGauge("memory", 512.0, tags)
	collector.RecordHistogram("latency", 50.0, tags)

	metrics := collector.GetMetrics()

	if metrics["counters"] == nil {
		t.Error("expected counters in export")
	}
	if metrics["gauges"] == nil {
		t.Error("expected gauges in export")
	}
	if metrics["histograms"] == nil {
		t.Error("expected histograms in export")
	}
}

// TestProperty22_HistogramPercentileMonotonicity tests percentile monotonicity
func TestProperty22_HistogramPercentileMonotonicity(t *testing.T) {
	// Property: Percentiles must be monotonically increasing
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	for i := 1; i <= 100; i++ {
		collector.RecordHistogram("latency", float64(i), tags)
	}

	stats := collector.GetHistogramStats("latency", tags)

	if stats.Percentile50 > stats.Percentile95 {
		t.Errorf("p50 (%f) should be <= p95 (%f)", stats.Percentile50, stats.Percentile95)
	}
	if stats.Percentile95 > stats.Percentile99 {
		t.Errorf("p95 (%f) should be <= p99 (%f)", stats.Percentile95, stats.Percentile99)
	}
	if stats.Percentile99 > stats.Max {
		t.Errorf("p99 (%f) should be <= max (%f)", stats.Percentile99, stats.Max)
	}
}

// TestProperty22_MetricsCount tests metric counting
func TestProperty22_MetricsCount(t *testing.T) {
	// Property: Metric counts must accurately reflect recorded metrics
	collector := NewDefaultMetricsCollector()

	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}
	tags3 := map[string]string{"service": "cache"}

	collector.RecordCounter("requests", 1, tags1)
	collector.RecordCounter("requests", 1, tags2)
	collector.RecordGauge("memory", 100.0, tags1)
	collector.RecordGauge("memory", 200.0, tags3)
	collector.RecordHistogram("latency", 50.0, tags1)

	if collector.GetCounterCount() != 2 {
		t.Errorf("expected 2 counters, got %d", collector.GetCounterCount())
	}
	if collector.GetGaugeCount() != 2 {
		t.Errorf("expected 2 gauges, got %d", collector.GetGaugeCount())
	}
	if collector.GetHistogramCount() != 1 {
		t.Errorf("expected 1 histogram, got %d", collector.GetHistogramCount())
	}
}

// TestProperty22_NegativeCounterAccumulation tests negative counter accumulation
func TestProperty22_NegativeCounterAccumulation(t *testing.T) {
	// Property: Counters must handle negative increments correctly
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("delta", 100, tags)
	collector.RecordCounter("delta", -30, tags)
	collector.RecordCounter("delta", -20, tags)

	value := collector.GetCounter("delta", tags)
	if value != 50 {
		t.Errorf("expected counter 50, got %d", value)
	}
}

// TestProperty22_HistogramMinMaxTracking tests min/max tracking
func TestProperty22_HistogramMinMaxTracking(t *testing.T) {
	// Property: Histogram must correctly track min and max values
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	values := []float64{50.0, 10.0, 100.0, 25.0, 75.0}
	for _, v := range values {
		collector.RecordHistogram("latency", v, tags)
	}

	stats := collector.GetHistogramStats("latency", tags)

	if stats.Min != 10.0 {
		t.Errorf("expected min 10.0, got %f", stats.Min)
	}
	if stats.Max != 100.0 {
		t.Errorf("expected max 100.0, got %f", stats.Max)
	}
}
