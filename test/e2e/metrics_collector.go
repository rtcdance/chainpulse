package e2e

import (
	"fmt"
	"sync"
	"time"
)

// MetricType represents the type of metric being collected
type MetricType string

const (
	MetricTypeLatency    MetricType = "latency"
	MetricTypeThroughput MetricType = "throughput"
	MetricTypeErrorRate  MetricType = "error_rate"
	MetricTypeMemory     MetricType = "memory"
	MetricTypeCPU        MetricType = "cpu"
)

// MetricPoint represents a single metric measurement
type MetricPoint struct {
	Timestamp time.Time
	Value     float64
	MetricType MetricType
	Labels    map[string]string
}

// MetricAggregation represents aggregated metrics
type MetricAggregation struct {
	Min       float64
	Max       float64
	Avg       float64
	P50       float64
	P95       float64
	P99       float64
	Count     int64
	Sum       float64
	Timestamp time.Time
}

// MetricsCollector collects and aggregates metrics from E2E tests
type MetricsCollector struct {
	mu              sync.RWMutex
	metrics         map[MetricType][]MetricPoint
	aggregations    map[MetricType]*MetricAggregation
	startTime       time.Time
	endTime         time.Time
	testName        string
	testStatus      string
	errorCount      int64
	successCount    int64
	totalOperations int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(testName string) *MetricsCollector {
	return &MetricsCollector{
		metrics:      make(map[MetricType][]MetricPoint),
		aggregations: make(map[MetricType]*MetricAggregation),
		testName:     testName,
		startTime:    time.Now(),
	}
}

// RecordMetric records a single metric point
func (mc *MetricsCollector) RecordMetric(metricType MetricType, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	point := MetricPoint{
		Timestamp:  time.Now(),
		Value:      value,
		MetricType: metricType,
		Labels:     labels,
	}

	mc.metrics[metricType] = append(mc.metrics[metricType], point)
	mc.totalOperations++
}

// RecordLatency records a latency measurement
func (mc *MetricsCollector) RecordLatency(duration time.Duration, labels map[string]string) {
	mc.RecordMetric(MetricTypeLatency, float64(duration.Microseconds()), labels)
}

// RecordThroughput records a throughput measurement (operations per second)
func (mc *MetricsCollector) RecordThroughput(opsPerSec float64, labels map[string]string) {
	mc.RecordMetric(MetricTypeThroughput, opsPerSec, labels)
}

// RecordErrorRate records an error rate measurement
func (mc *MetricsCollector) RecordErrorRate(errorRate float64, labels map[string]string) {
	mc.RecordMetric(MetricTypeErrorRate, errorRate, labels)
}

// RecordSuccess increments the success counter
func (mc *MetricsCollector) RecordSuccess() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.successCount++
}

// RecordError increments the error counter
func (mc *MetricsCollector) RecordError() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.errorCount++
}

// GetMetrics returns all collected metrics for a given type
func (mc *MetricsCollector) GetMetrics(metricType MetricType) []MetricPoint {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := mc.metrics[metricType]
	result := make([]MetricPoint, len(metrics))
	copy(result, metrics)
	return result
}

// Aggregate calculates aggregated metrics for a given type
func (mc *MetricsCollector) Aggregate(metricType MetricType) *MetricAggregation {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := mc.metrics[metricType]
	if len(metrics) == 0 {
		return &MetricAggregation{
			Timestamp: time.Now(),
		}
	}

	// Calculate basic statistics
	var sum float64
	var min, max = metrics[0].Value, metrics[0].Value
	values := make([]float64, len(metrics))

	for i, m := range metrics {
		values[i] = m.Value
		sum += m.Value
		if m.Value < min {
			min = m.Value
		}
		if m.Value > max {
			max = m.Value
		}
	}

	avg := sum / float64(len(metrics))

	// Calculate percentiles
	p50 := calculatePercentile(values, 50)
	p95 := calculatePercentile(values, 95)
	p99 := calculatePercentile(values, 99)

	agg := &MetricAggregation{
		Min:       min,
		Max:       max,
		Avg:       avg,
		P50:       p50,
		P95:       p95,
		P99:       p99,
		Count:     int64(len(metrics)),
		Sum:       sum,
		Timestamp: time.Now(),
	}

	mc.aggregations[metricType] = agg
	return agg
}

// GetAggregation returns cached aggregation for a metric type
func (mc *MetricsCollector) GetAggregation(metricType MetricType) *MetricAggregation {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if agg, exists := mc.aggregations[metricType]; exists {
		return agg
	}
	return nil
}

// SetTestStatus sets the test status
func (mc *MetricsCollector) SetTestStatus(status string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.testStatus = status
}

// Finalize finalizes the metrics collection
func (mc *MetricsCollector) Finalize() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.endTime = time.Now()
}

// GetDuration returns the total test duration
func (mc *MetricsCollector) GetDuration() time.Duration {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.endTime.IsZero() {
		return time.Since(mc.startTime)
	}
	return mc.endTime.Sub(mc.startTime)
}

// GetErrorRate returns the error rate as a percentage
func (mc *MetricsCollector) GetErrorRate() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.successCount+mc.errorCount == 0 {
		return 0
	}
	return float64(mc.errorCount) / float64(mc.successCount+mc.errorCount) * 100
}

// GetSuccessRate returns the success rate as a percentage
func (mc *MetricsCollector) GetSuccessRate() float64 {
	return 100 - mc.GetErrorRate()
}

// GetStats returns overall statistics
func (mc *MetricsCollector) GetStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"test_name":         mc.testName,
		"test_status":       mc.testStatus,
		"duration":          mc.GetDuration().String(),
		"total_operations":  mc.totalOperations,
		"success_count":     mc.successCount,
		"error_count":       mc.errorCount,
		"error_rate":        mc.GetErrorRate(),
		"success_rate":      mc.GetSuccessRate(),
		"start_time":        mc.startTime,
		"end_time":          mc.endTime,
	}
}

// TestReportGenerator generates test reports from collected metrics
type TestReportGenerator struct {
	collectors map[string]*MetricsCollector
	mu         sync.RWMutex
}

// NewTestReportGenerator creates a new test report generator
func NewTestReportGenerator() *TestReportGenerator {
	return &TestReportGenerator{
		collectors: make(map[string]*MetricsCollector),
	}
}

// RegisterCollector registers a metrics collector
func (trg *TestReportGenerator) RegisterCollector(testName string, collector *MetricsCollector) {
	trg.mu.Lock()
	defer trg.mu.Unlock()
	trg.collectors[testName] = collector
}

// GenerateReport generates a comprehensive test report
func (trg *TestReportGenerator) GenerateReport() string {
	trg.mu.RLock()
	defer trg.mu.RUnlock()

	report := "=== E2E Test Report ===\n\n"

	totalTests := len(trg.collectors)
	passedTests := 0
	failedTests := 0
	totalDuration := time.Duration(0)
	totalOperations := int64(0)
	totalErrors := int64(0)

	for testName, collector := range trg.collectors {
		collector.Aggregate(MetricTypeLatency)
		collector.Aggregate(MetricTypeThroughput)
		collector.Aggregate(MetricTypeErrorRate)

		report += fmt.Sprintf("Test: %s\n", testName)
		report += fmt.Sprintf("  Status: %s\n", collector.testStatus)
		report += fmt.Sprintf("  Duration: %v\n", collector.GetDuration())
		report += fmt.Sprintf("  Operations: %d\n", collector.totalOperations)
		report += fmt.Sprintf("  Success: %d\n", collector.successCount)
		report += fmt.Sprintf("  Errors: %d\n", collector.errorCount)
		report += fmt.Sprintf("  Error Rate: %.2f%%\n", collector.GetErrorRate())

		// Latency metrics
		if latencyAgg := collector.GetAggregation(MetricTypeLatency); latencyAgg != nil && latencyAgg.Count > 0 {
			report += "  Latency (µs):\n"
			report += fmt.Sprintf("    Min: %.2f\n", latencyAgg.Min)
			report += fmt.Sprintf("    Max: %.2f\n", latencyAgg.Max)
			report += fmt.Sprintf("    Avg: %.2f\n", latencyAgg.Avg)
			report += fmt.Sprintf("    P50: %.2f\n", latencyAgg.P50)
			report += fmt.Sprintf("    P95: %.2f\n", latencyAgg.P95)
			report += fmt.Sprintf("    P99: %.2f\n", latencyAgg.P99)
		}

		// Throughput metrics
		if throughputAgg := collector.GetAggregation(MetricTypeThroughput); throughputAgg != nil && throughputAgg.Count > 0 {
			report += "  Throughput (ops/sec):\n"
			report += fmt.Sprintf("    Min: %.2f\n", throughputAgg.Min)
			report += fmt.Sprintf("    Max: %.2f\n", throughputAgg.Max)
			report += fmt.Sprintf("    Avg: %.2f\n", throughputAgg.Avg)
		}

		report += "\n"

		if collector.testStatus == "PASSED" {
			passedTests++
		} else {
			failedTests++
		}

		totalDuration += collector.GetDuration()
		totalOperations += collector.totalOperations
		totalErrors += collector.errorCount
	}

	// Summary
	report += "=== Summary ===\n"
	report += fmt.Sprintf("Total Tests: %d\n", totalTests)
	report += fmt.Sprintf("Passed: %d\n", passedTests)
	report += fmt.Sprintf("Failed: %d\n", failedTests)
	report += fmt.Sprintf("Total Duration: %v\n", totalDuration)
	report += fmt.Sprintf("Total Operations: %d\n", totalOperations)
	report += fmt.Sprintf("Total Errors: %d\n", totalErrors)

	if totalOperations > 0 {
		report += fmt.Sprintf("Overall Error Rate: %.2f%%\n", float64(totalErrors)/float64(totalOperations)*100)
	}

	return report
}

// calculatePercentile calculates the percentile value from a sorted slice
func calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	quickSort(sorted, 0, len(sorted)-1)

	// Calculate index
	index := int(float64(len(sorted)-1) * percentile / 100)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// quickSort sorts a slice of floats in ascending order
func quickSort(arr []float64, low, high int) {
	if low < high {
		pi := partition(arr, low, high)
		quickSort(arr, low, pi-1)
		quickSort(arr, pi+1, high)
	}
}

// partition partitions the array for quicksort
func partition(arr []float64, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}
