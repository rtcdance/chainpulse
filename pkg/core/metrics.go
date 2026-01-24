package core

import (
	"sort"
	"sync"
	"time"
)

// DefaultMetricsCollector collects and aggregates metrics
type DefaultMetricsCollector struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64
	histograms map[string][]float64
	tags     map[string]map[string]string
	timestamps map[string]time.Time
}

// MetricEntry represents a single metric entry
type MetricEntry struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Value     interface{}            `json:"value"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// HistogramStats represents histogram statistics
type HistogramStats struct {
	Count      int64   `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Mean       float64 `json:"mean"`
	Percentile50 float64 `json:"percentile_50"`
	Percentile95 float64 `json:"percentile_95"`
	Percentile99 float64 `json:"percentile_99"`
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
