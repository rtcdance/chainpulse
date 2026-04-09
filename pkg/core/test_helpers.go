package core

import (
	"context"
	"io"
	"runtime"
	"sync"
	"time"
)

// NewTestLogger creates a logger for testing
func NewTestLogger() Logger {
	return NewDefaultLoggerWithOutput(LogLevelDebug, io.Discard)
}

// GoroutineTracker tracks goroutines to detect leaks
type GoroutineTracker struct {
	initialCount int
	finalCount   int
}

// NewGoroutineTracker creates a new goroutine tracker
func NewGoroutineTracker() *GoroutineTracker {
	return &GoroutineTracker{
		initialCount: runtime.NumGoroutine(),
	}
}

// Check verifies that no goroutines were leaked
func (gt *GoroutineTracker) Check() int {
	gt.finalCount = runtime.NumGoroutine()
	return gt.finalCount - gt.initialCount
}

// GetLeak returns the number of leaked goroutines
func (gt *GoroutineTracker) GetLeak() int {
	return gt.finalCount - gt.initialCount
}

// NewTestMetricsCollector creates a metrics collector for testing
func NewTestMetricsCollector() MetricsCollector {
	return NewDefaultMetricsCollector()
}

// TestLogger is a logger implementation for testing
type TestLogger struct {
	mu       sync.RWMutex
	messages []string
	level    LogLevel
}

// NewTestLoggerWithCapture creates a logger that captures messages
func NewTestLoggerWithCapture() *TestLogger {
	return &TestLogger{
		messages: make([]string, 0),
		level:    LogLevelDebug,
	}
}

// Debug logs a debug message
func (l *TestLogger) Debug(msg string, _ ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

// Info logs an info message
func (l *TestLogger) Info(msg string, _ ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

// Warn logs a warning message
func (l *TestLogger) Warn(msg string, _ ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

// Error logs an error message
func (l *TestLogger) Error(msg string, fields ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

// Fatal logs a fatal message
func (l *TestLogger) Fatal(msg string, fields ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

// WithCorrelationID returns a new logger with correlation ID
func (l *TestLogger) WithCorrelationID(_ string) Logger {
	return l
}

// WithField adds a field to the logger
func (l *TestLogger) WithField(_ string, _ interface{}) Logger {
	return l
}

// GetMessages returns all captured messages
func (l *TestLogger) GetMessages() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	messages := make([]string, len(l.messages))
	copy(messages, l.messages)
	return messages
}

// LogDebug logs a debug message with fields
func (l *TestLogger) LogDebug(msg string, fields ...interface{}) {
	l.Debug(msg, fields...)
}

// LogInfo logs an info message with fields
func (l *TestLogger) LogInfo(msg string, fields ...interface{}) {
	l.Info(msg, fields...)
}

// LogWarn logs a warning message with fields
func (l *TestLogger) LogWarn(msg string, fields ...interface{}) {
	l.Warn(msg, fields...)
}

// LogError logs an error message with fields
func (l *TestLogger) LogError(msg string, fields ...interface{}) {
	l.Error(msg, fields...)
}

// LogFatal logs a fatal message with fields
func (l *TestLogger) LogFatal(msg string, fields ...interface{}) {
	l.Fatal(msg, fields...)
}

// TestMetricsCollector is a metrics collector for testing
type TestMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
}

// NewTestMetricsCollectorWithCapture creates a metrics collector that captures metrics
func NewTestMetricsCollectorWithCapture() *TestMetricsCollector {
	return &TestMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

// RecordCounter records a counter metric
func (m *TestMetricsCollector) RecordCounter(name string, value int64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] = m.counters[name] + value
}

// RecordGauge records a gauge metric
func (m *TestMetricsCollector) RecordGauge(name string, value float64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

// RecordHistogram records a histogram metric
func (m *TestMetricsCollector) RecordHistogram(name string, value float64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], value)
}

// GetMetrics returns all collected metrics
func (m *TestMetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})
	result["counters"] = m.counters
	result["gauges"] = m.gauges
	result["histograms"] = m.histograms
	return result
}

// GetCounter returns a counter value
func (m *TestMetricsCollector) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

// GetGauge returns a gauge value
func (m *TestMetricsCollector) GetGauge(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

// GetHistogram returns histogram values
func (m *TestMetricsCollector) GetHistogram(name string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]float64, len(m.histograms[name]))
	copy(values, m.histograms[name])
	return values
}

// Reset resets all metrics
func (m *TestMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters = make(map[string]int64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
}

// GetHistogramStats returns histogram statistics
func (m *TestMetricsCollector) GetHistogramStats(name string) HistogramStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values := m.histograms[name]
	if len(values) == 0 {
		return HistogramStats{}
	}

	sum := 0.0
	minValue := values[0]
	maxValue := values[0]

	for _, v := range values {
		sum += v
		if v < minValue {
			minValue = v
		}
		if v > maxValue {
			maxValue = v
		}
	}

	mean := sum / float64(len(values))

	return HistogramStats{
		Count: int64(len(values)),
		Sum:   sum,
		Min:   minValue,
		Max:   maxValue,
		Mean:  mean,
	}
}

// GetAllMetrics returns all metrics as a map
func (m *TestMetricsCollector) GetAllMetrics() map[string]interface{} {
	return m.GetMetrics()
}

// RecordLatency records a latency metric
func (m *TestMetricsCollector) RecordLatency(name string, duration time.Duration, tags map[string]string) {
	m.RecordHistogram(name, float64(duration.Milliseconds()), tags)
}

// RecordError records an error metric
func (m *TestMetricsCollector) RecordError(name string, tags map[string]string) {
	m.RecordCounter(name, 1, tags)
}

// RecordSuccess records a success metric
func (m *TestMetricsCollector) RecordSuccess(name string, tags map[string]string) {
	m.RecordCounter(name, 1, tags)
}

// MockCache is a mock implementation of a cache for testing
type MockCache struct {
	mu    sync.RWMutex
	data  map[string][]byte
	ttl   map[string]time.Time
	calls map[string]int
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		data:  make(map[string][]byte),
		ttl:   make(map[string]time.Time),
		calls: make(map[string]int),
	}
}

// Get retrieves a value from the cache
func (mc *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	mc.calls["Get"]++

	if expiry, exists := mc.ttl[key]; exists && time.Now().After(expiry) {
		return nil, nil
	}

	value, exists := mc.data[key]
	if !exists {
		return nil, nil
	}

	return value, nil
}

// Set stores a value in the cache
func (mc *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.calls["Set"]++

	mc.data[key] = value
	if ttl > 0 {
		mc.ttl[key] = time.Now().Add(ttl)
	}

	return nil
}

// Delete removes a value from the cache
func (mc *MockCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.calls["Delete"]++

	delete(mc.data, key)
	delete(mc.ttl, key)

	return nil
}

// Clear clears all values from the cache
func (mc *MockCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.calls["Clear"]++

	mc.data = make(map[string][]byte)
	mc.ttl = make(map[string]time.Time)

	return nil
}

// GetCallCount returns the number of times a method was called
func (mc *MockCache) GetCallCount(method string) int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.calls[method]
}

// MockEventBus is a mock implementation of an event bus for testing
type MockEventBus struct {
	mu        sync.RWMutex
	listeners map[string][]func(interface{})
	events    []interface{}
}

// NewMockEventBus creates a new mock event bus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		listeners: make(map[string][]func(interface{})),
		events:    make([]interface{}, 0),
	}
}

// Subscribe subscribes to an event
func (meb *MockEventBus) Subscribe(eventType string, handler func(interface{})) {
	meb.mu.Lock()
	defer meb.mu.Unlock()

	meb.listeners[eventType] = append(meb.listeners[eventType], handler)
}

// Publish publishes an event
func (meb *MockEventBus) Publish(eventType string, event interface{}) {
	meb.mu.Lock()
	handlers := meb.listeners[eventType]
	meb.events = append(meb.events, event)
	meb.mu.Unlock()

	for _, handler := range handlers {
		handler(event)
	}
}

// GetPublishedEvents returns all published events
func (meb *MockEventBus) GetPublishedEvents() []interface{} {
	meb.mu.RLock()
	defer meb.mu.RUnlock()

	events := make([]interface{}, len(meb.events))
	copy(events, meb.events)
	return events
}

// GetPublishedEventCount returns the count of published events
func (meb *MockEventBus) GetPublishedEventCount() int {
	meb.mu.RLock()
	defer meb.mu.RUnlock()
	return len(meb.events)
}

// MockRegistry is a mock implementation of a registry for testing
type MockRegistry struct {
	mu        sync.RWMutex
	plugins   map[string]interface{}
	factories map[string]func() interface{}
}

// NewMockRegistry creates a new mock registry
func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		plugins:   make(map[string]interface{}),
		factories: make(map[string]func() interface{}),
	}
}

// Register registers a plugin
func (mr *MockRegistry) Register(name string, plugin interface{}) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.plugins[name] = plugin
	return nil
}

// Get retrieves a registered plugin
func (mr *MockRegistry) Get(name string) (interface{}, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	plugin, exists := mr.plugins[name]
	if !exists {
		return nil, nil
	}

	return plugin, nil
}

// RegisterFactory registers a plugin factory
func (mr *MockRegistry) RegisterFactory(name string, factory func() interface{}) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.factories[name] = factory
	return nil
}

// GetFactory retrieves a registered factory
func (mr *MockRegistry) GetFactory(name string) (func() interface{}, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	factory, exists := mr.factories[name]
	if !exists {
		return nil, nil
	}

	return factory, nil
}

// TestContextBuilder builds test contexts with specific configurations
type TestContextBuilder struct {
	timeout    time.Duration
	values     map[interface{}]interface{}
	cancelFunc context.CancelFunc
	ctx        context.Context
}

// NewTestContextBuilder creates a new test context builder
func NewTestContextBuilder() *TestContextBuilder {
	return &TestContextBuilder{
		timeout: 30 * time.Second,
		values:  make(map[interface{}]interface{}),
	}
}

// WithTimeout sets the timeout for the context
func (tcb *TestContextBuilder) WithTimeout(timeout time.Duration) *TestContextBuilder {
	tcb.timeout = timeout
	return tcb
}

// WithValue adds a value to the context
func (tcb *TestContextBuilder) WithValue(key, value interface{}) *TestContextBuilder {
	tcb.values[key] = value
	return tcb
}

// Build builds the context
func (tcb *TestContextBuilder) Build() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), tcb.timeout)

	for key, value := range tcb.values {
		ctx = context.WithValue(ctx, key, value)
	}

	tcb.ctx = ctx
	tcb.cancelFunc = cancel

	return ctx, cancel
}

// TestDataBuilder builds test data for common types
type TestDataBuilder struct {
	chainID    string
	blockNum   uint64
	eventCount int
}

// NewTestDataBuilder creates a new test data builder
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		chainID:    "ethereum",
		blockNum:   1,
		eventCount: 10,
	}
}

// WithChainID sets the chain ID
func (tdb *TestDataBuilder) WithChainID(chainID string) *TestDataBuilder {
	tdb.chainID = chainID
	return tdb
}

// WithBlockNumber sets the block number
func (tdb *TestDataBuilder) WithBlockNumber(blockNum uint64) *TestDataBuilder {
	tdb.blockNum = blockNum
	return tdb
}

// WithEventCount sets the event count
func (tdb *TestDataBuilder) WithEventCount(count int) *TestDataBuilder {
	tdb.eventCount = count
	return tdb
}

// BuildEvent builds a single test event
func (tdb *TestDataBuilder) BuildEvent(id string) *BlockchainEvent {
	return &BlockchainEvent{
		ID:              id,
		ChainID:         tdb.chainID,
		BlockNumber:     tdb.blockNum,
		TransactionHash: [32]byte{},
		EventName:       "TestEvent",
	}
}

// BuildEvents builds multiple test events
func (tdb *TestDataBuilder) BuildEvents() []*BlockchainEvent {
	events := make([]*BlockchainEvent, tdb.eventCount)
	for i := 0; i < tdb.eventCount; i++ {
		events[i] = tdb.BuildEvent(string(rune(i)))
	}
	return events
}

// NilPointerDetector detects nil pointer dereferences in tests
type NilPointerDetector struct {
	mu       sync.RWMutex
	detected []string
}

// NewNilPointerDetector creates a new nil pointer detector
func NewNilPointerDetector() *NilPointerDetector {
	return &NilPointerDetector{
		detected: make([]string, 0),
	}
}

// Check checks if a value is nil and records it
func (npd *NilPointerDetector) Check(name string, value interface{}) bool {
	if value == nil {
		npd.mu.Lock()
		defer npd.mu.Unlock()
		npd.detected = append(npd.detected, name)
		return true
	}
	return false
}

// GetDetected returns all detected nil pointers
func (npd *NilPointerDetector) GetDetected() []string {
	npd.mu.RLock()
	defer npd.mu.RUnlock()

	detected := make([]string, len(npd.detected))
	copy(detected, npd.detected)
	return detected
}

// HasDetected returns true if any nil pointers were detected
func (npd *NilPointerDetector) HasDetected() bool {
	npd.mu.RLock()
	defer npd.mu.RUnlock()
	return len(npd.detected) > 0
}

// Reset clears all detected nil pointers
func (npd *NilPointerDetector) Reset() {
	npd.mu.Lock()
	defer npd.mu.Unlock()
	npd.detected = make([]string, 0)
}
