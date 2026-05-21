package shared

import (
	"sync"
	"time"
)

// Monitoring tracks metrics for API operations
type Monitoring struct {
	protocols map[string]*ProtocolMetrics
	mu        sync.RWMutex
}

// ProtocolMetrics tracks metrics for a specific protocol
type ProtocolMetrics struct {
	name               string
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	totalDuration      time.Duration
	minDuration        time.Duration
	maxDuration        time.Duration
	lastRequestTime    time.Time
	mu                 sync.RWMutex
}

// NewMonitoring creates a new monitoring instance
func NewMonitoring() *Monitoring {
	return &Monitoring{
		protocols: make(map[string]*ProtocolMetrics),
	}
}

// NewProtocolMetrics creates a new protocol metrics instance
func NewProtocolMetrics(name string) *ProtocolMetrics {
	return &ProtocolMetrics{
		name:        name,
		minDuration: time.Duration(1<<63 - 1), // Max int64
	}
}

// RecordRequest records a request
func (m *Monitoring) RecordRequest(protocol string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, ok := m.protocols[protocol]
	if !ok {
		metrics = NewProtocolMetrics(protocol)
		m.protocols[protocol] = metrics
	}

	metrics.recordRequest(duration, success)
}

// recordRequest records a request in protocol metrics
func (pm *ProtocolMetrics) recordRequest(duration time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.totalRequests++
	pm.totalDuration += duration
	pm.lastRequestTime = time.Now()

	if success {
		pm.successfulRequests++
	} else {
		pm.failedRequests++
	}

	if duration < pm.minDuration {
		pm.minDuration = duration
	}

	if duration > pm.maxDuration {
		pm.maxDuration = duration
	}
}

// GetMetrics returns metrics for a protocol
func (m *Monitoring) GetMetrics(protocol string) map[string]any {
	m.mu.RLock()
	metrics, ok := m.protocols[protocol]
	m.mu.RUnlock()

	if !ok {
		return map[string]any{
			"protocol": protocol,
			"error":    "protocol not found",
		}
	}

	protocolMetrics := metrics.getMetrics()
	totalRequests, _ := protocolMetrics["total_requests"].(int64)
	successfulRequests, _ := protocolMetrics["successful_requests"].(int64)
	failedRequests, _ := protocolMetrics["failed_requests"].(int64)
	errorRate, _ := protocolMetrics["error_rate"].(float64)
	avgDurationMS, _ := protocolMetrics["avg_duration_ms"].(int64)

	coveragePosture := classifyMonitoringCoveragePosture(totalRequests, successfulRequests, failedRequests)
	runtimePosture := classifyMonitoringRuntimePosture(totalRequests, errorRate, avgDurationMS)

	protocolMetrics["coverage_posture"] = coveragePosture
	protocolMetrics["runtime_posture"] = runtimePosture
	protocolMetrics["reliability_hint"] = buildMonitoringReliabilityHint(coveragePosture, runtimePosture)
	return protocolMetrics
}

// GetProtocolRuntimeMetrics returns protocol-scoped compact runtime metrics for
// request health and delivery latency on top of raw monitoring counters.
func (m *Monitoring) GetProtocolRuntimeMetrics(protocol string) map[string]any {
	metrics := m.GetMetrics(protocol)
	if _, ok := metrics["error"]; ok {
		return metrics
	}

	totalRequests, _ := metrics["total_requests"].(int64)
	successfulRequests, _ := metrics["successful_requests"].(int64)
	failedRequests, _ := metrics["failed_requests"].(int64)
	errorRate, _ := metrics["error_rate"].(float64)
	avgDurationMS, _ := metrics["avg_duration_ms"].(int64)

	return map[string]any{
		"protocol":            protocol,
		"total_requests":      totalRequests,
		"successful_requests": successfulRequests,
		"failed_requests":     failedRequests,
		"success_rate":        metrics["success_rate"],
		"error_rate":          errorRate,
		"avg_duration_ms":     avgDurationMS,
		"last_request_time":   metrics["last_request_time"],
		"coverage_posture":    metrics["coverage_posture"],
		"runtime_posture":     metrics["runtime_posture"],
		"reliability_hint":    metrics["reliability_hint"],
	}
}

// GetRuntimeMetrics returns an aggregate compact runtime surface across all
// monitored protocols.
func (m *Monitoring) GetRuntimeMetrics() map[string]any {
	totalRequests := m.GetTotalRequests()
	successfulRequests := m.GetTotalSuccessfulRequests()
	failedRequests := m.GetTotalFailedRequests()
	avgDurationMS := m.GetAverageResponseTime().Milliseconds()

	errorRate := 0.0
	successRate := 0.0
	if totalRequests > 0 {
		errorRate = float64(failedRequests) / float64(totalRequests) * 100.0
		successRate = float64(successfulRequests) / float64(totalRequests) * 100.0
	}

	coveragePosture := classifyMonitoringCoveragePosture(totalRequests, successfulRequests, failedRequests)
	runtimePosture := classifyMonitoringRuntimePosture(totalRequests, errorRate, avgDurationMS)

	return map[string]any{
		"protocol_count":      m.GetProtocolCount(),
		"total_requests":      totalRequests,
		"successful_requests": successfulRequests,
		"failed_requests":     failedRequests,
		"success_rate":        successRate,
		"error_rate":          errorRate,
		"avg_duration_ms":     avgDurationMS,
		"coverage_posture":    coveragePosture,
		"runtime_posture":     runtimePosture,
		"reliability_hint":    buildMonitoringReliabilityHint(coveragePosture, runtimePosture),
	}
}

// getMetrics returns metrics from protocol metrics
func (pm *ProtocolMetrics) getMetrics() map[string]any {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	avgDuration := time.Duration(0)
	if pm.totalRequests > 0 {
		avgDuration = pm.totalDuration / time.Duration(pm.totalRequests)
	}

	successRate := 0.0
	if pm.totalRequests > 0 {
		successRate = float64(pm.successfulRequests) / float64(pm.totalRequests) * 100.0
	}

	errorRate := 0.0
	if pm.totalRequests > 0 {
		errorRate = float64(pm.failedRequests) / float64(pm.totalRequests) * 100.0
	}

	return map[string]any{
		"protocol":            pm.name,
		"total_requests":      pm.totalRequests,
		"successful_requests": pm.successfulRequests,
		"failed_requests":     pm.failedRequests,
		"success_rate":        successRate,
		"error_rate":          errorRate,
		"avg_duration_ms":     avgDuration.Milliseconds(),
		"min_duration_ms":     pm.minDuration.Milliseconds(),
		"max_duration_ms":     pm.maxDuration.Milliseconds(),
		"total_duration":      pm.totalDuration.String(),
		"last_request_time":   pm.lastRequestTime,
	}
}

// GetAllMetrics returns metrics for all protocols
func (m *Monitoring) GetAllMetrics() map[string]map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]map[string]any)
	for protocol, metrics := range m.protocols {
		result[protocol] = metrics.getMetrics()
	}

	return result
}

// ResetMetrics resets metrics for a protocol
func (m *Monitoring) ResetMetrics(protocol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metrics, ok := m.protocols[protocol]; ok {
		metrics.reset()
	}
}

// reset resets protocol metrics
func (pm *ProtocolMetrics) reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.totalRequests = 0
	pm.successfulRequests = 0
	pm.failedRequests = 0
	pm.totalDuration = 0
	pm.minDuration = time.Duration(1<<63 - 1)
	pm.maxDuration = 0
	pm.lastRequestTime = time.Time{}
}

// GetProtocolCount returns the number of protocols being monitored
func (m *Monitoring) GetProtocolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.protocols)
}

// GetTotalRequests returns total requests across all protocols
func (m *Monitoring) GetTotalRequests() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := int64(0)
	for _, metrics := range m.protocols {
		metrics.mu.RLock()
		total += metrics.totalRequests
		metrics.mu.RUnlock()
	}

	return total
}

// GetTotalSuccessfulRequests returns total successful requests across all protocols
func (m *Monitoring) GetTotalSuccessfulRequests() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := int64(0)
	for _, metrics := range m.protocols {
		metrics.mu.RLock()
		total += metrics.successfulRequests
		metrics.mu.RUnlock()
	}

	return total
}

// GetTotalFailedRequests returns total failed requests across all protocols
func (m *Monitoring) GetTotalFailedRequests() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := int64(0)
	for _, metrics := range m.protocols {
		metrics.mu.RLock()
		total += metrics.failedRequests
		metrics.mu.RUnlock()
	}

	return total
}

// GetAverageResponseTime returns average response time across all protocols
func (m *Monitoring) GetAverageResponseTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalDuration := time.Duration(0)
	totalRequests := int64(0)

	for _, metrics := range m.protocols {
		metrics.mu.RLock()
		totalDuration += metrics.totalDuration
		totalRequests += metrics.totalRequests
		metrics.mu.RUnlock()
	}

	if totalRequests == 0 {
		return 0
	}

	return totalDuration / time.Duration(totalRequests)
}

func classifyMonitoringCoveragePosture(totalRequests int64, successfulRequests int64, failedRequests int64) string {
	if totalRequests == 0 {
		return "monitoring-unobserved"
	}
	if failedRequests == 0 {
		return "monitoring-success-only"
	}
	if successfulRequests == 0 {
		return "monitoring-fail-only"
	}
	return "monitoring-mixed"
}

func classifyMonitoringRuntimePosture(totalRequests int64, errorRate float64, avgDurationMS int64) string {
	if totalRequests == 0 {
		return "monitoring-unobserved"
	}
	if errorRate >= 50 {
		return "monitoring-degraded"
	}
	if avgDurationMS > 1000 {
		return "monitoring-slow"
	}
	return "monitoring-healthy"
}

func buildMonitoringReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "monitoring-degraded":
		return "protocol monitoring shows high failure rate; investigate request errors before treating the path as healthy"
	case runtimePosture == "monitoring-slow":
		return "protocol monitoring shows elevated latency; verify downstream responsiveness and request load"
	case coveragePosture == "monitoring-fail-only":
		return "protocol monitoring has only observed failures; verify availability before relying on this path"
	case coveragePosture == "monitoring-mixed":
		return "protocol monitoring shows mixed outcomes; continue observing success rate and failure drift"
	case coveragePosture == "monitoring-success-only":
		return "protocol monitoring is observing successful traffic with healthy runtime posture"
	default:
		return "protocol monitoring has not observed traffic yet"
	}
}
