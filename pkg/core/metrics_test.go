package core

import (
	"strings"
	"testing"
)

// TestNewDefaultMetricsCollector tests metrics collector creation
func TestNewDefaultMetricsCollector(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	if collector == nil {
		t.Fatal("expected collector, got nil")
	}
	if collector.GetCounterCount() != 0 {
		t.Error("expected empty counter map")
	}
	if collector.GetGaugeCount() != 0 {
		t.Error("expected empty gauge map")
	}
	if collector.GetHistogramCount() != 0 {
		t.Error("expected empty histogram map")
	}
}

// TestRecordCounter tests counter recording
func TestRecordCounter(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 1, tags)
	collector.RecordCounter("requests", 2, tags)

	value := collector.GetCounter("requests", tags)
	if value != 3 {
		t.Errorf("expected counter value 3, got %d", value)
	}
}

// TestRecordGauge tests gauge recording
func TestRecordGauge(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordGauge("memory_usage", 512.5, tags)
	value := collector.GetGauge("memory_usage", tags)

	if value != 512.5 {
		t.Errorf("expected gauge value 512.5, got %f", value)
	}
}

// TestRecordHistogram tests histogram recording
func TestRecordHistogram(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordHistogram("latency", 10.5, tags)
	collector.RecordHistogram("latency", 20.3, tags)
	collector.RecordHistogram("latency", 15.7, tags)

	stats := collector.GetHistogramStats("latency", tags)
	if stats.Count != 3 {
		t.Errorf("expected count 3, got %d", stats.Count)
	}
	if stats.Sum != 46.5 {
		t.Errorf("expected sum 46.5, got %f", stats.Sum)
	}
}

// TestGetMetrics tests getting all metrics
func TestGetMetrics(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 5, tags)
	collector.RecordGauge("memory", 512.0, tags)
	collector.RecordHistogram("latency", 10.0, tags)

	metrics := collector.GetMetrics()

	if metrics["counters"] == nil {
		t.Error("expected counters in metrics")
	}
	if metrics["gauges"] == nil {
		t.Error("expected gauges in metrics")
	}
	if metrics["histograms"] == nil {
		t.Error("expected histograms in metrics")
	}
}

func TestExportPrometheusIncludesCounterGaugeAndHistogram(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api", "chain-id": "ethereum"}

	collector.RecordCounter("requests.total", 5, tags)
	collector.RecordGauge("memory-usage", 512.5, tags)
	collector.RecordHistogram("latency_ms", 10, tags)
	collector.RecordHistogram("latency_ms", 20, tags)

	output := collector.ExportPrometheus()

	for _, expected := range []string{
		`# TYPE chainpulse_requests_total counter`,
		`chainpulse_requests_total{chain_id="ethereum",service="api"} 5`,
		`# TYPE chainpulse_memory_usage gauge`,
		`chainpulse_memory_usage{chain_id="ethereum",service="api"} 512.5`,
		`# TYPE chainpulse_latency_ms histogram`,
		`chainpulse_latency_ms_bucket{chain_id="ethereum",le="10",service="api"} 1`,
		`chainpulse_latency_ms_bucket{chain_id="ethereum",le="25",service="api"} 2`,
		`chainpulse_latency_ms_count{chain_id="ethereum",service="api"} 2`,
		`# TYPE go_goroutines gauge`,
		`# TYPE go_memstats_alloc_bytes gauge`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestFormatPrometheusMetricsFallback(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"counters": map[string]any{
			"test_counter:service=api": map[string]any{
				"value": int64(3),
				"tags":  map[string]any{"service": "api"},
			},
		},
		"gauges": map[string]any{
			"test_gauge": map[string]any{
				"value": float64(7),
			},
		},
	}

	output := FormatPrometheusMetrics(payload)
	if !strings.Contains(output, "# TYPE chainpulse_test_counter counter") {
		t.Fatalf("expected fallback counter output, got:\n%s", output)
	}
	if !strings.Contains(output, `chainpulse_test_counter{chain_id="global",service="api"} 3`) {
		t.Fatalf("expected fallback counter labels to be normalized, got:\n%s", output)
	}
	if !strings.Contains(output, `chainpulse_test_gauge{chain_id="global"} 7`) {
		t.Fatalf("expected fallback gauge output, got:\n%s", output)
	}
}

func TestExportPrometheusDefaultsChainIDForApplicationMetrics(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	collector.RecordCounter("gateway_request_success", 1, map[string]string{"service": "api-gateway"})

	output := collector.ExportPrometheus()

	if !strings.Contains(output, `chainpulse_gateway_request_success{chain_id="global",service="api_gateway"} 1`) &&
		!strings.Contains(output, `chainpulse_gateway_request_success{chain_id="global",service="api-gateway"} 1`) {
		t.Fatalf("expected exporter to inject global chain_id label, got:\n%s", output)
	}
}

func TestExportPrometheusNormalizesChainLabelAliases(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	collector.RecordGauge("indexing_runtime_started", 1, map[string]string{"chain": "polygon"})

	output := collector.ExportPrometheus()

	if !strings.Contains(output, `chainpulse_indexing_runtime_started{chain_id="polygon"} 1`) {
		t.Fatalf("expected exporter to normalize chain label to chain_id, got:\n%s", output)
	}
}

// TestReset tests resetting all metrics
func TestReset(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 5, tags)
	collector.RecordGauge("memory", 512.0, tags)

	if collector.GetCounterCount() != 1 {
		t.Error("expected 1 counter before reset")
	}

	collector.Reset()

	if collector.GetCounterCount() != 0 {
		t.Error("expected 0 counters after reset")
	}
	if collector.GetGaugeCount() != 0 {
		t.Error("expected 0 gauges after reset")
	}
}

// TestResetCounter tests resetting a specific counter
func TestResetCounter(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 5, tags)
	collector.ResetCounter("requests", tags)

	value := collector.GetCounter("requests", tags)
	if value != 0 {
		t.Errorf("expected counter value 0 after reset, got %d", value)
	}
}

// TestResetGauge tests resetting a specific gauge
func TestResetGauge(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordGauge("memory", 512.0, tags)
	collector.ResetGauge("memory", tags)

	value := collector.GetGauge("memory", tags)
	if value != 0 {
		t.Errorf("expected gauge value 0 after reset, got %f", value)
	}
}

// TestResetHistogram tests resetting a specific histogram
func TestResetHistogram(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordHistogram("latency", 10.0, tags)
	collector.ResetHistogram("latency", tags)

	stats := collector.GetHistogramStats("latency", tags)
	if stats.Count != 0 {
		t.Errorf("expected count 0 after reset, got %d", stats.Count)
	}
}

// TestMultipleTags tests metrics with different tags
func TestMultipleTags(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}

	collector.RecordCounter("requests", 5, tags1)
	collector.RecordCounter("requests", 3, tags2)

	value1 := collector.GetCounter("requests", tags1)
	value2 := collector.GetCounter("requests", tags2)

	if value1 != 5 {
		t.Errorf("expected counter value 5 for tags1, got %d", value1)
	}
	if value2 != 3 {
		t.Errorf("expected counter value 3 for tags2, got %d", value2)
	}
}

// TestNoTags tests metrics without tags
func TestNoTags(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()

	collector.RecordCounter("requests", 5, nil)
	value := collector.GetCounter("requests", nil)

	if value != 5 {
		t.Errorf("expected counter value 5, got %d", value)
	}
}

// TestHistogramStats tests histogram statistics calculation
func TestHistogramStats(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for _, v := range values {
		collector.RecordHistogram("latency", v, tags)
	}

	stats := collector.GetHistogramStats("latency", tags)

	if stats.Count != 5 {
		t.Errorf("expected count 5, got %d", stats.Count)
	}
	if stats.Sum != 150.0 {
		t.Errorf("expected sum 150.0, got %f", stats.Sum)
	}
	if stats.Min != 10.0 {
		t.Errorf("expected min 10.0, got %f", stats.Min)
	}
	if stats.Max != 50.0 {
		t.Errorf("expected max 50.0, got %f", stats.Max)
	}
	if stats.Mean != 30.0 {
		t.Errorf("expected mean 30.0, got %f", stats.Mean)
	}
}

// TestHistogramPercentiles tests histogram percentile calculation
func TestHistogramPercentiles(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	// Record 100 values from 1 to 100
	for i := 1; i <= 100; i++ {
		collector.RecordHistogram("latency", float64(i), tags)
	}

	stats := collector.GetHistogramStats("latency", tags)

	// Percentiles should be approximately correct
	if stats.Percentile50 < 40 || stats.Percentile50 > 60 {
		t.Errorf("expected p50 around 50, got %f", stats.Percentile50)
	}
	if stats.Percentile95 < 85 || stats.Percentile95 > 100 {
		t.Errorf("expected p95 around 95, got %f", stats.Percentile95)
	}
	if stats.Percentile99 < 95 || stats.Percentile99 > 100 {
		t.Errorf("expected p99 around 99, got %f", stats.Percentile99)
	}
}

// TestConcurrentRecording tests concurrent metric recording
func TestConcurrentRecording(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			collector.RecordCounter("requests", 1, tags)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	value := collector.GetCounter("requests", tags)
	if value != 10 {
		t.Errorf("expected counter value 10, got %d", value)
	}
}

func TestExportAlias(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}
	collector.RecordCounter("requests", 5, tags)

	exported := collector.Export()
	if exported["counters"] == nil {
		t.Error("expected counters in export")
	}
}

func TestSetDBPoolStats(t *testing.T) {
	t.Parallel()
	stats := DBPoolStats{
		MaxOpenConnections: 25,
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
		WaitCount:          100,
		WaitDuration:       0,
	}
	SetDBPoolStats(stats)

	val := dbPoolStats.Load()
	if val == nil {
		t.Fatal("expected db pool stats to be set")
	}
	loaded, ok := val.(DBPoolStats)
	if !ok {
		t.Fatal("expected DBPoolStats type")
	}
	if loaded.MaxOpenConnections != 25 {
		t.Errorf("MaxOpenConnections = %d, want 25", loaded.MaxOpenConnections)
	}
}

func TestIncrementUnknownEventSignatures(t *testing.T) {
	ResetUnknownEventSignatureCount()
	if GetUnknownEventSignatureCount() != 0 {
		t.Fatal("expected 0 after reset")
	}
	IncrementUnknownEventSignatures()
	IncrementUnknownEventSignatures()
	if GetUnknownEventSignatureCount() != 2 {
		t.Errorf("expected 2, got %d", GetUnknownEventSignatureCount())
	}
	ResetUnknownEventSignatureCount()
	if GetUnknownEventSignatureCount() != 0 {
		t.Errorf("expected 0 after reset, got %d", GetUnknownEventSignatureCount())
	}
}

func TestExportPrometheusIncludesDBPoolStats(t *testing.T) {
	t.Parallel()
	stats := DBPoolStats{
		MaxOpenConnections: 25,
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
	}
	SetDBPoolStats(stats)

	collector := NewDefaultMetricsCollector()
	output := collector.ExportPrometheus()

	for _, expected := range []string{
		`# TYPE chainpulse_db_pool_max_open_connections gauge`,
		`chainpulse_db_pool_max_open_connections 25`,
		`chainpulse_db_pool_open_connections 10`,
		`chainpulse_db_pool_in_use 5`,
		`chainpulse_db_pool_idle 5`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

// TestCounterAccumulation tests counter accumulation
func TestCounterAccumulation(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	for i := 0; i < 5; i++ {
		collector.RecordCounter("requests", 1, tags)
	}

	value := collector.GetCounter("requests", tags)
	if value != 5 {
		t.Errorf("expected counter value 5, got %d", value)
	}
}

// TestGaugeOverwrite tests gauge overwrite behavior
func TestGaugeOverwrite(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordGauge("memory", 100.0, tags)
	collector.RecordGauge("memory", 200.0, tags)
	collector.RecordGauge("memory", 150.0, tags)

	value := collector.GetGauge("memory", tags)
	if value != 150.0 {
		t.Errorf("expected gauge value 150.0, got %f", value)
	}
}

// TestEmptyHistogramStats tests histogram stats with no values
func TestEmptyHistogramStats(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	stats := collector.GetHistogramStats("latency", tags)
	if stats.Count != 0 {
		t.Errorf("expected count 0, got %d", stats.Count)
	}
}

// TestMetricsWithComplexTags tests metrics with multiple tag values
func TestMetricsWithComplexTags(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{
		"service": "api",
		"method":  "GET",
		"status":  "200",
	}

	collector.RecordCounter("requests", 5, tags)
	value := collector.GetCounter("requests", tags)

	if value != 5 {
		t.Errorf("expected counter value 5, got %d", value)
	}
}

// TestGetCounterCount tests counter count
func TestGetCounterCount(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}

	collector.RecordCounter("requests", 5, tags1)
	collector.RecordCounter("requests", 3, tags2)

	count := collector.GetCounterCount()
	if count != 2 {
		t.Errorf("expected counter count 2, got %d", count)
	}
}

// TestGetGaugeCount tests gauge count
func TestGetGaugeCount(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}

	collector.RecordGauge("memory", 100.0, tags1)
	collector.RecordGauge("memory", 200.0, tags2)

	count := collector.GetGaugeCount()
	if count != 2 {
		t.Errorf("expected gauge count 2, got %d", count)
	}
}

// TestGetHistogramCount tests histogram count
func TestGetHistogramCount(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags1 := map[string]string{"service": "api"}
	tags2 := map[string]string{"service": "worker"}

	collector.RecordHistogram("latency", 10.0, tags1)
	collector.RecordHistogram("latency", 20.0, tags2)

	count := collector.GetHistogramCount()
	if count != 2 {
		t.Errorf("expected histogram count 2, got %d", count)
	}
}

// TestHistogramSingleValue tests histogram with single value
func TestHistogramSingleValue(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordHistogram("latency", 42.0, tags)
	stats := collector.GetHistogramStats("latency", tags)

	if stats.Count != 1 {
		t.Errorf("expected count 1, got %d", stats.Count)
	}
	if stats.Sum != 42.0 {
		t.Errorf("expected sum 42.0, got %f", stats.Sum)
	}
	if stats.Min != 42.0 {
		t.Errorf("expected min 42.0, got %f", stats.Min)
	}
	if stats.Max != 42.0 {
		t.Errorf("expected max 42.0, got %f", stats.Max)
	}
	if stats.Mean != 42.0 {
		t.Errorf("expected mean 42.0, got %f", stats.Mean)
	}
}

// TestNegativeValues tests metrics with negative values
func TestNegativeValues(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("delta", -5, tags)
	collector.RecordGauge("temperature", -10.5, tags)
	collector.RecordHistogram("change", -3.2, tags)

	counterValue := collector.GetCounter("delta", tags)
	gaugeValue := collector.GetGauge("temperature", tags)
	histStats := collector.GetHistogramStats("change", tags)

	if counterValue != -5 {
		t.Errorf("expected counter value -5, got %d", counterValue)
	}
	if gaugeValue != -10.5 {
		t.Errorf("expected gauge value -10.5, got %f", gaugeValue)
	}
	if histStats.Min != -3.2 {
		t.Errorf("expected histogram min -3.2, got %f", histStats.Min)
	}
}

// TestZeroValues tests metrics with zero values
func TestZeroValues(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	tags := map[string]string{"service": "api"}

	collector.RecordCounter("requests", 0, tags)
	collector.RecordGauge("memory", 0.0, tags)
	collector.RecordHistogram("latency", 0.0, tags)

	counterValue := collector.GetCounter("requests", tags)
	gaugeValue := collector.GetGauge("memory", tags)
	histStats := collector.GetHistogramStats("latency", tags)

	if counterValue != 0 {
		t.Errorf("expected counter value 0, got %d", counterValue)
	}
	if gaugeValue != 0.0 {
		t.Errorf("expected gauge value 0.0, got %f", gaugeValue)
	}
	if histStats.Count != 1 {
		t.Errorf("expected histogram count 1, got %d", histStats.Count)
	}
}

func TestExportMetricsPrometheus_NilInput(t *testing.T) {
	t.Parallel()
	output := ExportMetricsPrometheus(nil)
	if output != "" {
		t.Errorf("expected empty string for nil input, got %s", output)
	}
}

func TestExportMetricsPrometheus_WithExporter(t *testing.T) {
	t.Parallel()
	collector := NewDefaultMetricsCollector()
	collector.RecordCounter("test", 1, nil)
	output := ExportMetricsPrometheus(collector)
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(output, "chainpulse_test") {
		t.Errorf("expected prometheus format output, got: %s", output)
	}
}

func TestExportMetricsPrometheus_WithoutExporter(t *testing.T) {
	t.Parallel()
	collector := &plainMetricsCollector{
		metrics: map[string]any{
			"counters": map[string]any{
				"test_counter:": map[string]any{
					"value": int64(5),
					"tags":  map[string]any{},
				},
			},
			"gauges":     map[string]any{},
			"histograms": map[string]any{},
		},
	}
	output := ExportMetricsPrometheus(collector)
	if !strings.Contains(output, "# TYPE chainpulse_test_counter counter") {
		t.Errorf("expected fallback prometheus format, got: %s", output)
	}
}

type plainMetricsCollector struct {
	metrics map[string]any
}

func (p *plainMetricsCollector) RecordCounter(name string, value int64, tags map[string]string)     {}
func (p *plainMetricsCollector) RecordGauge(name string, value float64, tags map[string]string)     {}
func (p *plainMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {}
func (p *plainMetricsCollector) GetMetrics() map[string]any {
	return p.metrics
}
