package api

import (
	"context"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// MockLogger implements core.Logger for testing
type MockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) WithCorrelationID(id string) core.Logger {
	return m
}

// MockMetricsCollector implements core.MetricsCollector for testing
type MockMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

// NewMockMetrics is an alias for NewMockMetricsCollector for backward compatibility
func NewMockMetrics() *MockMetricsCollector {
	return NewMockMetricsCollector()
}

func (m *MockMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *MockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *MockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], value)
}

func (m *MockMetricsCollector) RecordTimer(name string, duration time.Duration, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], float64(duration.Milliseconds()))
}

func (m *MockMetricsCollector) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

func (m *MockMetricsCollector) GetGauge(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

func (m *MockMetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	metrics := make(map[string]interface{})
	metrics["counters"] = m.counters
	metrics["gauges"] = m.gauges
	metrics["histograms"] = m.histograms
	return metrics
}

func (m *MockMetricsCollector) GetCounterValue(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

func (m *MockMetricsCollector) GetHistogramValues(name string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.histograms[name]
}

// MockCacheService implements cache service interface for testing
type MockCacheService struct {
	mu    sync.RWMutex
	cache map[string]interface{}
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		cache: make(map[string]interface{}),
	}
}

func (m *MockCacheService) Get(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache[key], nil
}

func (m *MockCacheService) Set(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *MockCacheService) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

func (m *MockCacheService) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]interface{})
	return nil
}

// MockCachePlugin implements core.CachePlugin for testing
type MockCachePlugin struct {
	mu    sync.RWMutex
	cache map[string][]byte
}

func NewMockCachePlugin() *MockCachePlugin {
	return &MockCachePlugin{
		cache: make(map[string][]byte),
	}
}

func (m *MockCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache[key], nil
}

func (m *MockCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *MockCachePlugin) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

func (m *MockCachePlugin) GetStats() core.CacheStats {
	return core.CacheStats{
		HitCount:      0,
		MissCount:     0,
		EvictionCount: 0,
		HitRate:       0.0,
	}
}

func (m *MockCachePlugin) Name() string {
	return "MockCachePlugin"
}

func (m *MockCachePlugin) Version() string {
	return "1.0.0"
}

func (m *MockCachePlugin) Initialize(config core.Config) error {
	return nil
}

func (m *MockCachePlugin) Start() error {
	return nil
}

func (m *MockCachePlugin) Stop() error {
	return nil
}

func (m *MockCachePlugin) Health() error {
	return nil
}

func (m *MockCachePlugin) Close() error {
	return nil
}
