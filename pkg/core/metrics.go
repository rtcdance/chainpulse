package core

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultMetricsCollector collects and aggregates metrics
type DefaultMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
	tags       map[string]map[string]string
	timestamps map[string]time.Time
}

// MetricEntry represents a single metric entry
type MetricEntry struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Value     interface{}       `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// HistogramStats represents histogram statistics
type HistogramStats struct {
	Count        int64   `json:"count"`
	Sum          float64 `json:"sum"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Mean         float64 `json:"mean"`
	Percentile50 float64 `json:"percentile_50"`
	Percentile95 float64 `json:"percentile_95"`
	Percentile99 float64 `json:"percentile_99"`
}

// PrometheusMetricsExporter is an optional capability for collectors that can
// render Prometheus exposition text directly.
type PrometheusMetricsExporter interface {
	ExportPrometheus() string
}

// NewDefaultMetricsCollector creates a new metrics collector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
		tags:       make(map[string]map[string]string),
		timestamps: make(map[string]time.Time),
	}
}

// RecordCounter records a counter metric
func (m *DefaultMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.counters[key] = m.counters[key] + value
	m.tags[key] = tags
	m.timestamps[key] = time.Now().UTC()
}

// RecordGauge records a gauge metric
func (m *DefaultMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.gauges[key] = value
	m.tags[key] = tags
	m.timestamps[key] = time.Now().UTC()
}

// RecordHistogram records a histogram metric
func (m *DefaultMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	m.histograms[key] = append(m.histograms[key], value)
	m.tags[key] = tags
	m.timestamps[key] = time.Now().UTC()
}

// GetMetrics returns all collected metrics
func (m *DefaultMetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})

	// Add counters
	counters := make(map[string]interface{})
	for key, value := range m.counters {
		counters[key] = map[string]interface{}{
			"value":     value,
			"tags":      m.tags[key],
			"timestamp": m.timestamps[key],
		}
	}
	result["counters"] = counters

	// Add gauges
	gauges := make(map[string]interface{})
	for key, value := range m.gauges {
		gauges[key] = map[string]interface{}{
			"value":     value,
			"tags":      m.tags[key],
			"timestamp": m.timestamps[key],
		}
	}
	result["gauges"] = gauges

	// Add histograms
	histograms := make(map[string]interface{})
	for key, values := range m.histograms {
		histograms[key] = map[string]interface{}{
			"stats":     m.calculateHistogramStats(values),
			"tags":      m.tags[key],
			"timestamp": m.timestamps[key],
		}
	}
	result["histograms"] = histograms

	return result
}

// Export returns all collected metrics (alias for GetMetrics)
func (m *DefaultMetricsCollector) Export() map[string]interface{} {
	return m.GetMetrics()
}

// ExportPrometheus renders the current metric set using the Prometheus text
// exposition format.
func (m *DefaultMetricsCollector) ExportPrometheus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var builder strings.Builder

	counterKeys := make([]string, 0, len(m.counters))
	for key := range m.counters {
		counterKeys = append(counterKeys, key)
	}

	sort.Strings(counterKeys)

	for _, key := range counterKeys {
		name, tags := m.metricNameAndTags(key)
		writePrometheusHeader(&builder, name, "counter")
		writePrometheusSample(&builder, name, tags, strconv.FormatInt(m.counters[key], 10))
	}

	gaugeKeys := make([]string, 0, len(m.gauges))
	for key := range m.gauges {
		gaugeKeys = append(gaugeKeys, key)
	}

	sort.Strings(gaugeKeys)

	for _, key := range gaugeKeys {
		name, tags := m.metricNameAndTags(key)
		writePrometheusHeader(&builder, name, "gauge")
		writePrometheusSample(&builder, name, tags, formatPrometheusFloat(m.gauges[key]))
	}

	histogramKeys := make([]string, 0, len(m.histograms))
	for key := range m.histograms {
		histogramKeys = append(histogramKeys, key)
	}

	sort.Strings(histogramKeys)

	for _, key := range histogramKeys {
		name, tags := m.metricNameAndTags(key)
		writePrometheusHeader(&builder, name, "histogram")
		writePrometheusHistogram(&builder, name, tags, m.histograms[key])
	}

	writePrometheusRuntimeMetrics(&builder)

	return builder.String()
}

// ExportMetricsPrometheus renders Prometheus exposition text for a collector.
func ExportMetricsPrometheus(metrics MetricsCollector) string {
	if metrics == nil {
		return ""
	}

	if exporter, ok := metrics.(PrometheusMetricsExporter); ok {
		return exporter.ExportPrometheus()
	}

	return FormatPrometheusMetrics(metrics.GetMetrics())
}

// FormatPrometheusMetrics renders a generic metrics payload into Prometheus text.
func FormatPrometheusMetrics(payload map[string]interface{}) string {
	var builder strings.Builder

	writePrometheusPayloadSection(&builder, payload, "counters", "counter")
	writePrometheusPayloadSection(&builder, payload, "gauges", "gauge")

	return builder.String()
}

// GetCounter returns a specific counter value
func (m *DefaultMetricsCollector) GetCounter(name string, tags map[string]string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	return m.counters[key]
}

// GetGauge returns a specific gauge value
func (m *DefaultMetricsCollector) GetGauge(name string, tags map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	return m.gauges[key]
}

// GetHistogramStats returns statistics for a histogram
func (m *DefaultMetricsCollector) GetHistogramStats(name string, tags map[string]string) HistogramStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	values := m.histograms[key]
	return m.calculateHistogramStats(values)
}

// Reset clears all metrics
func (m *DefaultMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]int64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
	m.tags = make(map[string]map[string]string)
	m.timestamps = make(map[string]time.Time)
}

// ResetCounter resets a specific counter
func (m *DefaultMetricsCollector) ResetCounter(name string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	delete(m.counters, key)
	delete(m.tags, key)
	delete(m.timestamps, key)
}

// ResetGauge resets a specific gauge
func (m *DefaultMetricsCollector) ResetGauge(name string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	delete(m.gauges, key)
	delete(m.tags, key)
	delete(m.timestamps, key)
}

// ResetHistogram resets a specific histogram
func (m *DefaultMetricsCollector) ResetHistogram(name string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, tags)
	delete(m.histograms, key)
	delete(m.tags, key)
	delete(m.timestamps, key)
}

// GetCounterCount returns the number of counters
func (m *DefaultMetricsCollector) GetCounterCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.counters)
}

// GetGaugeCount returns the number of gauges
func (m *DefaultMetricsCollector) GetGaugeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.gauges)
}

// GetHistogramCount returns the number of histograms
func (m *DefaultMetricsCollector) GetHistogramCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.histograms)
}

// buildKey creates a key from name and tags
func (m *DefaultMetricsCollector) buildKey(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}

	// Sort tags by key to ensure consistent key generation
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	key := name
	for _, k := range keys {
		key += ":" + k + "=" + tags[k]
	}
	return key
}

func (m *DefaultMetricsCollector) metricNameAndTags(key string) (string, map[string]string) {
	tags := m.tags[key]
	if tags == nil {
		tags = map[string]string{}
	}

	name := normalizePrometheusMetricName(strings.SplitN(key, ":", 2)[0])

	return normalizeExportedPrometheusMetricName(name), normalizeExportedPrometheusLabels(tags, name)
}

func writePrometheusPayloadSection(builder *strings.Builder, payload map[string]interface{}, section, metricType string) {
	rawSection, ok := payload[section].(map[string]interface{})
	if !ok {
		return
	}

	keys := make([]string, 0, len(rawSection))
	for key := range rawSection {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		entry, ok := rawSection[key].(map[string]interface{})
		if !ok {
			continue
		}

		name := normalizePrometheusMetricName(strings.SplitN(key, ":", 2)[0])
		normalizedName := normalizeExportedPrometheusMetricName(name)
		normalizedTags := normalizeExportedPrometheusLabels(interfaceMapToStringMap(entry["tags"]), name)

		writePrometheusHeader(builder, normalizedName, metricType)
		writePrometheusSample(builder, normalizedName, normalizedTags, formatPrometheusInterface(entry["value"]))
	}
}

func writePrometheusHeader(builder *strings.Builder, name, metricType string) {
	builder.WriteString("# TYPE ")
	builder.WriteString(name)
	builder.WriteByte(' ')
	builder.WriteString(metricType)
	builder.WriteByte('\n')
}

func writePrometheusSample(builder *strings.Builder, name string, tags map[string]string, value string) {
	builder.WriteString(name)
	writePrometheusTags(builder, tags)
	builder.WriteByte(' ')
	builder.WriteString(value)
	builder.WriteByte('\n')
}

func writePrometheusHistogram(builder *strings.Builder, name string, tags map[string]string, values []float64) {
	buckets := prometheusHistogramBuckets(values)
	sum := 0.0

	for _, value := range values {
		sum += value
	}

	for _, upperBound := range buckets {
		cumulative := 0

		for _, value := range values {
			if value <= upperBound {
				cumulative++
			}
		}

		bucketTags := copyMetricTags(tags)
		bucketTags["le"] = formatPrometheusFloat(upperBound)
		writePrometheusSample(builder, name+"_bucket", bucketTags, strconv.Itoa(cumulative))
	}

	infTags := copyMetricTags(tags)
	infTags["le"] = "+Inf"
	writePrometheusSample(builder, name+"_bucket", infTags, strconv.Itoa(len(values)))
	writePrometheusSample(builder, name+"_sum", tags, formatPrometheusFloat(sum))
	writePrometheusSample(builder, name+"_count", tags, strconv.Itoa(len(values)))
}

func writePrometheusTags(builder *strings.Builder, tags map[string]string) {
	if len(tags) == 0 {
		return
	}

	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	builder.WriteByte('{')

	for idx, key := range keys {
		if idx > 0 {
			builder.WriteByte(',')
		}

		builder.WriteString(normalizePrometheusLabelName(key))
		builder.WriteString(`="`)
		builder.WriteString(escapePrometheusLabelValue(tags[key]))
		builder.WriteByte('"')
	}

	builder.WriteByte('}')
}

func normalizePrometheusMetricName(name string) string {
	replacer := strings.NewReplacer(".", "_", "-", "_", " ", "_", "/", "_")
	name = replacer.Replace(name)

	if name == "" {
		return "chainpulse_metric"
	}

	return name
}

func normalizeExportedPrometheusMetricName(name string) string {
	name = normalizePrometheusMetricName(name)

	if isPrometheusRuntimeMetric(name) || strings.HasPrefix(name, "chainpulse_") {
		return name
	}

	return "chainpulse_" + name
}

func normalizePrometheusLabelName(name string) string {
	return normalizePrometheusMetricName(name)
}

func normalizeExportedPrometheusLabels(tags map[string]string, metricName string) map[string]string {
	if isPrometheusRuntimeMetric(metricName) {
		return copyMetricTags(tags)
	}

	normalized := map[string]string{}

	for key, value := range tags {
		switch normalizePrometheusLabelName(key) {
		case "chain_id", "chain":
			normalized["chain_id"] = value
		default:
			normalized[normalizePrometheusLabelName(key)] = value
		}
	}

	if _, ok := normalized["chain_id"]; !ok {
		normalized["chain_id"] = "global"
	}

	return normalized
}

func isPrometheusRuntimeMetric(name string) bool {
	return strings.HasPrefix(name, "go_")
}

func escapePrometheusLabelValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)

	return replacer.Replace(value)
}

func formatPrometheusFloat(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "+Inf"
	case math.IsInf(value, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func formatPrometheusInterface(value interface{}) string {
	switch typed := value.(type) {
	case float64:
		return formatPrometheusFloat(typed)
	case float32:
		return formatPrometheusFloat(float64(typed))
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case nil:
		return "0"
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func interfaceMapToStringMap(raw interface{}) map[string]string {
	typed, ok := raw.(map[string]string)
	if ok {
		return typed
	}

	values, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}

	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = fmt.Sprintf("%v", value)
	}

	return out
}

func copyMetricTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return map[string]string{}
	}

	out := make(map[string]string, len(tags))

	for key, value := range tags {
		out[key] = value
	}

	return out
}

func prometheusHistogramBuckets(values []float64) []float64 {
	if len(values) == 0 {
		return []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000}
	}

	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}

	base := []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000}
	buckets := make([]float64, 0, len(base))

	for _, bucket := range base {
		buckets = append(buckets, bucket)

		if bucket >= maxValue {
			return buckets
		}
	}

	last := base[len(base)-1]
	for last < maxValue {
		last *= 2
		buckets = append(buckets, last)
	}

	return buckets
}

func writePrometheusRuntimeMetrics(builder *strings.Builder) {
	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	writePrometheusHeader(builder, "go_goroutines", "gauge")
	writePrometheusSample(builder, "go_goroutines", nil, strconv.Itoa(runtime.NumGoroutine()))
	writePrometheusHeader(builder, "go_memstats_alloc_bytes", "gauge")
	writePrometheusSample(builder, "go_memstats_alloc_bytes", nil, strconv.FormatUint(memStats.Alloc, 10))
	writePrometheusHeader(builder, "go_memstats_sys_bytes", "gauge")
	writePrometheusSample(builder, "go_memstats_sys_bytes", nil, strconv.FormatUint(memStats.Sys, 10))
}

// calculateHistogramStats calculates statistics for histogram values
func (m *DefaultMetricsCollector) calculateHistogramStats(values []float64) HistogramStats {
	if len(values) == 0 {
		return HistogramStats{}
	}

	stats := HistogramStats{
		Count: int64(len(values)),
	}

	// Calculate sum and find min/max
	sum := 0.0
	min := values[0]
	max := values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	stats.Sum = sum
	stats.Min = min
	stats.Max = max
	stats.Mean = sum / float64(len(values))

	// Calculate percentiles
	stats.Percentile50 = m.calculatePercentile(values, 50)
	stats.Percentile95 = m.calculatePercentile(values, 95)
	stats.Percentile99 = m.calculatePercentile(values, 99)

	return stats
}

// calculatePercentile calculates a percentile value
func (m *DefaultMetricsCollector) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple percentile calculation
	index := int(float64(len(values)) * percentile / 100)
	if index >= len(values) {
		index = len(values) - 1
	}

	// Sort values (simple bubble sort for small arrays)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[index]
}
