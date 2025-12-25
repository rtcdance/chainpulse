package api

import (
	"sync"
	"time"
)

// PluginMetrics holds metrics for a specific plugin
type PluginMetrics struct {
	Name               string
	TotalRequests      int64
	TotalErrors        int64
	TotalSuccess       int64
	TotalResponseTime  time.Duration
	RequestCount       int64
	AvgResponseTime    time.Duration
	LastError          string
	LastErrorTime      time.Time
	LastRequestTime    time.Time
}

// MetricsCollector collects metrics for API plugins
type MetricsCollector struct {
	mu            sync.Mutex
	totalRequests int64
	totalErrors   int64
	totalSuccess  int64
	totalResponseTime time.Duration
	requestCount  int64
	avgResponseTime time.Duration
	pluginMetrics map[string]*PluginMetrics
}

// GlobalMetricsCollector is a global instance for collecting metrics
var GlobalMetricsCollector = &MetricsCollector{
	pluginMetrics: make(map[string]*PluginMetrics),
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		pluginMetrics: make(map[string]*PluginMetrics),
	}
}

// RecordRequest records a request for the given plugin
func (mc *MetricsCollector) RecordRequest(pluginName string, duration time.Duration, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update global metrics
	mc.totalRequests++
	mc.totalResponseTime += duration
	mc.requestCount++
	mc.avgResponseTime = mc.totalResponseTime / time.Duration(mc.requestCount)

	if err != nil {
		mc.totalErrors++
	} else {
		mc.totalSuccess++
	}

	// Update plugin-specific metrics
	pluginMetric, exists := mc.pluginMetrics[pluginName]
	if !exists {
		pluginMetric = &PluginMetrics{
			Name: pluginName,
		}
		mc.pluginMetrics[pluginName] = pluginMetric
	}

	pluginMetric.TotalRequests++
	pluginMetric.TotalResponseTime += duration
	pluginMetric.RequestCount++
	pluginMetric.AvgResponseTime = pluginMetric.TotalResponseTime / time.Duration(pluginMetric.RequestCount)
	pluginMetric.LastRequestTime = time.Now()

	if err != nil {
		pluginMetric.TotalErrors++
		pluginMetric.LastErrorTime = time.Now()
		pluginMetric.LastError = err.Error()
	} else {
		pluginMetric.TotalSuccess++
	}
}

// GetGlobalMetrics returns global metrics
func (mc *MetricsCollector) GetGlobalMetrics() (int64, int64, int64, time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	return mc.totalRequests, mc.totalErrors, mc.totalSuccess, mc.avgResponseTime
}

// GetPluginMetrics returns metrics for a specific plugin
func (mc *MetricsCollector) GetPluginMetrics(pluginName string) (*PluginMetrics, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.pluginMetrics[pluginName]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	result := *metrics
	return &result, true
}

// GetAllMetrics returns metrics for all plugins
func (mc *MetricsCollector) GetAllMetrics() map[string]*PluginMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	result := make(map[string]*PluginMetrics)
	for name, metrics := range mc.pluginMetrics {
		// Return a copy to avoid race conditions
		copied := *metrics
		result[name] = &copied
	}

	return result
}