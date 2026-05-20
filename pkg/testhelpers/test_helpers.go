package testhelpers

import (
	"context"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// NewTestLogger creates a logger for testing
func NewTestLogger() core.Logger {
	return core.NewDefaultLoggerWithOutput(core.LogLevelDebug, io.Discard)
}

// GoroutineTracker tracks goroutines to detect leaks
type GoroutineTracker struct {
	initialCount int
	finalCount   int
}

func NewGoroutineTracker() *GoroutineTracker {
	return &GoroutineTracker{
		initialCount: runtime.NumGoroutine(),
	}
}

func (gt *GoroutineTracker) Check() int {
	gt.finalCount = runtime.NumGoroutine()
	return gt.finalCount - gt.initialCount
}

func (gt *GoroutineTracker) GetLeak() int {
	return gt.finalCount - gt.initialCount
}

func NewTestMetricsCollector() core.MetricsCollector {
	return core.NewDefaultMetricsCollector()
}

// TestLogger is a logger implementation for testing
type TestLogger struct {
	mu       sync.RWMutex
	messages []string
	level    core.LogLevel
}

func NewTestLoggerWithCapture() *TestLogger {
	return &TestLogger{
		messages: make([]string, 0),
		level:    core.LogLevelDebug,
	}
}

func (l *TestLogger) Debug(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

func (l *TestLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

func (l *TestLogger) Warn(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

func (l *TestLogger) Error(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

func (l *TestLogger) Fatal(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, msg)
}

func (l *TestLogger) WithCorrelationID(_ string) core.Logger { return l }

func (l *TestLogger) WithField(_ string, _ any) core.Logger { return l }

func (l *TestLogger) GetMessages() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	messages := make([]string, len(l.messages))
	copy(messages, l.messages)
	return messages
}

func (l *TestLogger) LogDebug(msg string, fields ...any)  { l.Debug(msg, fields...) }
func (l *TestLogger) LogInfo(msg string, fields ...any)   { l.Info(msg, fields...) }
func (l *TestLogger) LogWarn(msg string, fields ...any)   { l.Warn(msg, fields...) }
func (l *TestLogger) LogError(msg string, fields ...any)  { l.Error(msg, fields...) }
func (l *TestLogger) LogFatal(msg string, fields ...any)  { l.Fatal(msg, fields...) }

// TestMetricsCollector is a metrics collector for testing
type TestMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
}

func NewTestMetricsCollectorWithCapture() *TestMetricsCollector {
	return &TestMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

func (m *TestMetricsCollector) RecordCounter(name string, value int64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] = m.counters[name] + value
}

func (m *TestMetricsCollector) RecordGauge(name string, value float64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *TestMetricsCollector) RecordHistogram(name string, value float64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], value)
}

func (m *TestMetricsCollector) GetMetrics() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]any{
		"counters":   m.counters,
		"gauges":     m.gauges,
		"histograms": m.histograms,
	}
}

func (m *TestMetricsCollector) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

func (m *TestMetricsCollector) GetGauge(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

func (m *TestMetricsCollector) GetHistogram(name string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]float64, len(m.histograms[name]))
	copy(values, m.histograms[name])
	return values
}

func (m *TestMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters = make(map[string]int64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
}

func (m *TestMetricsCollector) GetHistogramStats(name string) core.HistogramStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := m.histograms[name]
	if len(values) == 0 {
		return core.HistogramStats{}
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
	return core.HistogramStats{
		Count: int64(len(values)),
		Sum:   sum,
		Min:   minValue,
		Max:   maxValue,
		Mean:  sum / float64(len(values)),
	}
}

func (m *TestMetricsCollector) GetAllMetrics() map[string]any { return m.GetMetrics() }

func (m *TestMetricsCollector) RecordLatency(name string, duration time.Duration, tags map[string]string) {
	m.RecordHistogram(name, float64(duration.Milliseconds()), tags)
}

func (m *TestMetricsCollector) RecordError(name string, tags map[string]string) {
	m.RecordCounter(name, 1, tags)
}

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

func NewMockCache() *MockCache {
	return &MockCache{
		data:  make(map[string][]byte),
		ttl:   make(map[string]time.Time),
		calls: make(map[string]int),
	}
}

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

func (mc *MockCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.calls["Delete"]++
	delete(mc.data, key)
	delete(mc.ttl, key)
	return nil
}

func (mc *MockCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.calls["Clear"]++
	mc.data = make(map[string][]byte)
	mc.ttl = make(map[string]time.Time)
	return nil
}

func (mc *MockCache) GetCallCount(method string) int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.calls[method]
}

// MockEventBus is a mock implementation of an event bus for testing
type MockEventBus struct {
	mu        sync.RWMutex
	listeners map[string][]func(any)
	events    []any
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		listeners: make(map[string][]func(any)),
		events:    make([]any, 0),
	}
}

func (meb *MockEventBus) Subscribe(eventType string, handler func(any)) {
	meb.mu.Lock()
	defer meb.mu.Unlock()
	meb.listeners[eventType] = append(meb.listeners[eventType], handler)
}

func (meb *MockEventBus) Publish(eventType string, event any) {
	meb.mu.Lock()
	handlers := meb.listeners[eventType]
	meb.events = append(meb.events, event)
	meb.mu.Unlock()
	for _, handler := range handlers {
		handler(event)
	}
}

func (meb *MockEventBus) GetPublishedEvents() []any {
	meb.mu.RLock()
	defer meb.mu.RUnlock()
	events := make([]any, len(meb.events))
	copy(events, meb.events)
	return events
}

func (meb *MockEventBus) GetPublishedEventCount() int {
	meb.mu.RLock()
	defer meb.mu.RUnlock()
	return len(meb.events)
}

// MockRegistry is a mock implementation of a registry for testing
type MockRegistry struct {
	mu        sync.RWMutex
	plugins   map[string]any
	factories map[string]func() any
}

func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		plugins:   make(map[string]any),
		factories: make(map[string]func() any),
	}
}

func (mr *MockRegistry) Register(name string, plugin any) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.plugins[name] = plugin
	return nil
}

func (mr *MockRegistry) Get(name string) (any, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	plugin, exists := mr.plugins[name]
	if !exists {
		return nil, nil
	}
	return plugin, nil
}

func (mr *MockRegistry) RegisterFactory(name string, factory func() any) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.factories[name] = factory
	return nil
}

func (mr *MockRegistry) GetFactory(name string) (func() any, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	factory, exists := mr.factories[name]
	if !exists {
		return nil, nil
	}
	return factory, nil
}

// TestContextBuilder builds test contexts
type TestContextBuilder struct {
	timeout    time.Duration
	values     map[any]any
	cancelFunc context.CancelFunc
	ctx        context.Context
}

func NewTestContextBuilder() *TestContextBuilder {
	return &TestContextBuilder{
		timeout: 30 * time.Second,
		values:  make(map[any]any),
	}
}

func (tcb *TestContextBuilder) WithTimeout(timeout time.Duration) *TestContextBuilder {
	tcb.timeout = timeout
	return tcb
}

func (tcb *TestContextBuilder) WithValue(key, value any) *TestContextBuilder {
	tcb.values[key] = value
	return tcb
}

func (tcb *TestContextBuilder) Build() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), tcb.timeout)
	for key, value := range tcb.values {
		ctx = context.WithValue(ctx, key, value)
	}
	tcb.ctx = ctx
	tcb.cancelFunc = cancel
	return ctx, cancel
}

// TestDataBuilder builds test data
type TestDataBuilder struct {
	chainID    string
	blockNum   uint64
	eventCount int
}

func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		chainID:    "ethereum",
		blockNum:   1,
		eventCount: 10,
	}
}

func (tdb *TestDataBuilder) WithChainID(chainID string) *TestDataBuilder {
	tdb.chainID = chainID
	return tdb
}

func (tdb *TestDataBuilder) WithBlockNumber(blockNum uint64) *TestDataBuilder {
	tdb.blockNum = blockNum
	return tdb
}

func (tdb *TestDataBuilder) WithEventCount(count int) *TestDataBuilder {
	tdb.eventCount = count
	return tdb
}

func (tdb *TestDataBuilder) BuildEvent(id string) *core.BlockchainEvent {
	return &core.BlockchainEvent{
		ID:              id,
		ChainID:         tdb.chainID,
		BlockNumber:     tdb.blockNum,
		TransactionHash: [32]byte{},
		EventName:       "TestEvent",
	}
}

func (tdb *TestDataBuilder) BuildEvents() []*core.BlockchainEvent {
	events := make([]*core.BlockchainEvent, tdb.eventCount)
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

func NewNilPointerDetector() *NilPointerDetector {
	return &NilPointerDetector{
		detected: make([]string, 0),
	}
}

func (npd *NilPointerDetector) Check(name string, value any) bool {
	if value == nil {
		npd.mu.Lock()
		defer npd.mu.Unlock()
		npd.detected = append(npd.detected, name)
		return true
	}
	return false
}

func (npd *NilPointerDetector) GetDetected() []string {
	npd.mu.RLock()
	defer npd.mu.RUnlock()
	detected := make([]string, len(npd.detected))
	copy(detected, npd.detected)
	return detected
}

func (npd *NilPointerDetector) HasDetected() bool {
	npd.mu.RLock()
	defer npd.mu.RUnlock()
	return len(npd.detected) > 0
}

func (npd *NilPointerDetector) Reset() {
	npd.mu.Lock()
	defer npd.mu.Unlock()
	npd.detected = make([]string, 0)
}