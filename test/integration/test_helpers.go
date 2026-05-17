package integration

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Log level constants
const (
	LogLevelInfo  = "info"
	LogLevelDebug = "debug"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Level         string         `json:"level"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Fields        map[string]any `json:"fields,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

// BlockchainEvent is a test helper struct for blockchain events
type BlockchainEvent struct {
	ID        string
	ChainID   string
	BlockNum  uint64
	TxHash    string
	EventName string
}

// MockLogger is a mock implementation of core.Logger for testing
type MockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *MockLogger) Info(msg string, _ ...any) {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
}

func (m *MockLogger) Debug(msg string, _ ...any) {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
}

func (m *MockLogger) Warn(msg string, _ ...any) {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
}

func (m *MockLogger) Error(msg string, _ ...any) {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
}

func (m *MockLogger) Fatal(msg string, _ ...any) {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
}

func (m *MockLogger) WithCorrelationID(_ string) core.Logger {
	return m
}

// MockDataPuller is a mock implementation of core.DataPullerPlugin for testing
type MockDataPuller struct {
	chainID string
	events  []BlockchainEvent
	calls   int64
	mu      sync.Mutex
}

func (m *MockDataPuller) Name() string {
	return "mock-puller-" + m.chainID
}

func (m *MockDataPuller) Version() string {
	return "1.0.0"
}

func (m *MockDataPuller) Health() error {
	return nil
}

func (m *MockDataPuller) Initialize(_ core.Config) error {
	return nil
}

func (m *MockDataPuller) Start() error {
	return nil
}

func (m *MockDataPuller) Stop() error {
	return nil
}

func (m *MockDataPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	result := make([]core.BlockchainEvent, 0)
	for _, event := range m.events {
		if event.BlockNum >= fromBlock && event.BlockNum <= toBlock {
			result = append(result, core.BlockchainEvent{
				ID:              event.ID,
				ChainID:         event.ChainID,
				BlockNumber:     event.BlockNum,
				TransactionHash: common.HexToHash(event.TxHash),
				EventName:       event.EventName,
			})
		}
	}
	return result, nil
}

func (m *MockDataPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	if len(m.events) == 0 {
		return 0, nil
	}
	maxBlock := m.events[0].BlockNum
	for _, event := range m.events {
		if event.BlockNum > maxBlock {
			maxBlock = event.BlockNum
		}
	}
	return maxBlock, nil
}

func (m *MockDataPuller) GetStats() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	return map[string]any{
		"chain_id":    m.chainID,
		"event_count": len(m.events),
		"calls":       m.calls,
	}
}

func (m *MockDataPuller) ChainID() string {
	return m.chainID
}

func (m *MockDataPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	return nil
}

// SlowMockDataPuller is a mock data puller that simulates slow operations
type SlowMockDataPuller struct {
	chainID string
	delay   time.Duration
}

func (m *SlowMockDataPuller) Name() string {
	return "slow-puller-" + m.chainID
}

func (m *SlowMockDataPuller) Version() string {
	return "1.0.0"
}

func (m *SlowMockDataPuller) Health() error {
	return nil
}

func (m *SlowMockDataPuller) Initialize(_ core.Config) error {
	return nil
}

func (m *SlowMockDataPuller) Start() error {
	return nil
}

func (m *SlowMockDataPuller) Stop() error {
	return nil
}

func (m *SlowMockDataPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	time.Sleep(m.delay)
	return []core.BlockchainEvent{}, nil
}

func (m *SlowMockDataPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	time.Sleep(m.delay)
	return 0, nil
}

func (m *SlowMockDataPuller) GetStats() map[string]any {
	return map[string]any{
		"chain_id": m.chainID,
	}
}

func (m *SlowMockDataPuller) ChainID() string {
	return m.chainID
}

func (m *SlowMockDataPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	return nil
}

// MockDatabasePlugin is a mock implementation of core.DatabasePlugin for testing
type MockDatabasePlugin struct {
	events map[string]*core.BlockchainEvent
}

func NewMockDatabasePlugin() *MockDatabasePlugin {
	return &MockDatabasePlugin{
		events: make(map[string]*core.BlockchainEvent),
	}
}

func (m *MockDatabasePlugin) Name() string {
	return "mock-database"
}

func (m *MockDatabasePlugin) Version() string {
	return "1.0.0"
}

func (m *MockDatabasePlugin) Initialize(config core.Config) error {
	return nil
}

func (m *MockDatabasePlugin) Start() error {
	return nil
}

func (m *MockDatabasePlugin) Stop() error {
	return nil
}

func (m *MockDatabasePlugin) Health() error {
	return nil
}

func (m *MockDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	result := make([]any, 0)
	for _, event := range m.events {
		result = append(result, event)
	}
	return result, nil
}

func (m *MockDatabasePlugin) StoreEvent(ctx context.Context, event any) error {
	if blockchainEvent, ok := event.(*core.BlockchainEvent); ok {
		m.events[blockchainEvent.ID] = blockchainEvent
	}
	return nil
}

func (m *MockDatabasePlugin) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	return m.events[id], nil
}

func (m *MockDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error {
	delete(m.events, eventID)
	return nil
}

func (m *MockDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error {
	for _, event := range events {
		if blockchainEvent, ok := event.(*core.BlockchainEvent); ok {
			m.events[blockchainEvent.ID] = blockchainEvent
		}
	}
	return nil
}

func (m *MockDatabasePlugin) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	result := make([]*core.BlockchainEvent, 0)
	for _, event := range m.events {
		result = append(result, event)
	}
	return result, nil
}

func (m *MockDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	return []*core.Block{}, nil
}

func (m *MockDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	result := make([]*core.BlockchainEvent, 0)
	for _, event := range m.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			result = append(result, event)
		}
	}
	return result, nil
}

func (m *MockDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	return nil, nil
}

func (m *MockDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (m *MockDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (m *MockDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (m *MockDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return nil, nil
}

func (m *MockDatabasePlugin) Close() error {
	return nil
}

// MockCachePlugin is a mock implementation of core.CachePlugin for testing
type MockCachePlugin struct {
	data map[string]string
}

func NewMockCachePlugin() *MockCachePlugin {
	return &MockCachePlugin{
		data: make(map[string]string),
	}
}

func (m *MockCachePlugin) Name() string {
	return "mock-cache"
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

func (m *MockCachePlugin) HealthCheck(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *MockCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return []byte(val), nil
	}
	return nil, fmt.Errorf("key not found: %s", key)
}

func (m *MockCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	m.data[key] = string(value)
	return nil
}

func (m *MockCachePlugin) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCachePlugin) Clear() error {
	m.data = make(map[string]string)
	return nil
}

func (m *MockCachePlugin) Close() error {
	return nil
}

func (m *MockCachePlugin) GetStats() core.CacheStats {
	return core.CacheStats{
		HitCount:      0,
		MissCount:     0,
		EvictionCount: 0,
		HitRate:       0,
	}
}

// MockPlugin is a mock implementation of core.Plugin for testing
type MockPlugin struct {
	name    string
	version string
	healthy bool
}

func (m *MockPlugin) Name() string {
	return m.name
}

func (m *MockPlugin) Version() string {
	return m.version
}

func (m *MockPlugin) Initialize(config core.Config) error {
	return nil
}

func (m *MockPlugin) Start() error {
	return nil
}

func (m *MockPlugin) Stop() error {
	return nil
}

func (m *MockPlugin) Health() error {
	if !m.healthy {
		return fmt.Errorf("plugin unhealthy")
	}
	return nil
}

// MockMQPlugin is a mock implementation of core.MQPlugin for testing
type MockMQPlugin struct {
	name    string
	version string
	//nolint:unused
	logger core.Logger
	//nolint:unused
	metrics core.MetricsCollector
}

func (m *MockMQPlugin) Name() string {
	if m.name == "" {
		return "mock-mq"
	}
	return m.name
}

func (m *MockMQPlugin) Version() string {
	if m.version == "" {
		return "1.0.0"
	}
	return m.version
}

func (m *MockMQPlugin) Initialize(config core.Config) error {
	return nil
}

func (m *MockMQPlugin) Start() error {
	return nil
}

func (m *MockMQPlugin) Stop() error {
	return nil
}

func (m *MockMQPlugin) Health() error {
	return nil
}

func (m *MockMQPlugin) Publish(ctx context.Context, topic string, message []byte) error {
	return nil
}

func (m *MockMQPlugin) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	return nil
}

func (m *MockMQPlugin) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	return 0, nil
}

//nolint:unused
func createTestBlockchainEvent(chainID string, blockNum uint64, eventID string) *core.BlockchainEvent {
	return &core.BlockchainEvent{
		ID:              eventID,
		ChainID:         chainID,
		BlockNumber:     blockNum,
		TransactionHash: common.HexToHash("0x" + eventID),
		EventName:       "TestEvent",
	}
}

// NewDefaultPluginRegistry creates a default plugin registry for testing
func NewDefaultPluginRegistry() core.PluginRegistry {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	return core.NewPluginRegistry(logger)
}

// NewDefaultLogger creates a default logger for testing
func NewDefaultLogger(logLevel string) core.Logger {
	// Convert string log level to core.LogLevel
	level := core.ParseLogLevel(logLevel)
	return core.NewDefaultLogger(level)
}

// NewDefaultLoggerWithOutput creates a default logger with custom output for testing
func NewDefaultLoggerWithOutput(logLevel string, output io.Writer) core.Logger {
	// Convert string log level to core.LogLevel
	level := core.ParseLogLevel(logLevel)
	return core.NewDefaultLoggerWithOutput(level, output)
}

// NewDefaultConfigManager creates a default config manager for testing
func NewDefaultConfigManager() core.ConfigManager {
	return &DefaultConfigManager{
		config: core.Config{
			DeploymentMode: "monolithic",
			LogLevel:       LogLevelInfo,
		},
		values: make(map[string]any),
	}
}

// NewDefaultEventBus creates a default event bus for testing
func NewDefaultEventBus() core.EventBus {
	return &DefaultEventBus{
		subscribers: make(map[string][]func(any)),
	}
}

// NewDefaultMetricsCollector creates a default metrics collector for testing
func NewDefaultMetricsCollector() core.MetricsCollector {
	return &DefaultMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

// NewDefaultHealthChecker creates a default health checker for testing
func NewDefaultHealthChecker(registry core.PluginRegistry, configManager core.ConfigManager, eventBus core.EventBus, metricsCollector core.MetricsCollector, logger core.Logger) core.HealthChecker {
	return &DefaultHealthChecker{
		registry:         registry,
		configManager:    configManager,
		eventBus:         eventBus,
		metricsCollector: metricsCollector,
		logger:           logger,
	}
}

// DefaultConfigManager is a default implementation of ConfigManager for testing
type DefaultConfigManager struct {
	config core.Config
	values map[string]any
}

func (m *DefaultConfigManager) Load() (core.Config, error) {
	return m.config, nil
}

func (m *DefaultConfigManager) Validate(config core.Config) error {
	if config.DeploymentMode == "" {
		return fmt.Errorf("deployment mode is required")
	}
	return nil
}

func (m *DefaultConfigManager) Get(key string) (any, error) {
	// Check config struct first
	switch key {
	case "deployment_mode":
		return m.config.DeploymentMode, nil
	case "log_level":
		return m.config.LogLevel, nil
	}

	// Then check values map
	if val, ok := m.values[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key not found: %s", key)
}

func (m *DefaultConfigManager) Set(key string, value any) error {
	m.values[key] = value
	return nil
}

// DefaultEventBus is a default implementation of EventBus for testing
type DefaultEventBus struct {
	subscribers map[string][]func(any)
	nextID      uint64
}

func (b *DefaultEventBus) Subscribe(ctx context.Context, topic string, handler func(any)) (uint64, error) {
	b.subscribers[topic] = append(b.subscribers[topic], handler)
	b.nextID++
	return b.nextID, nil
}

func (b *DefaultEventBus) SubscribeNamed(ctx context.Context, topic, name string, handler func(any)) (uint64, error) {
	return b.Subscribe(ctx, topic, handler)
}

func (b *DefaultEventBus) Publish(ctx context.Context, topic string, event any) error {
	if handlers, ok := b.subscribers[topic]; ok {
		for _, handler := range handlers {
			handler(event)
		}
	}
	return nil
}

func (b *DefaultEventBus) Unsubscribe(subscriptionID uint64) error {
	// Simplified test implementation: remove all handlers for the topic
	// In production code, use the proper subscription ID-based approach
	return nil
}

// DefaultMetricsCollector is a default implementation of MetricsCollector for testing
type DefaultMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
}

func (m *DefaultMetricsCollector) RecordCounter(name string, value int64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *DefaultMetricsCollector) RecordGauge(name string, value float64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *DefaultMetricsCollector) RecordHistogram(name string, value float64, _ map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], value)
}

func (m *DefaultMetricsCollector) GetMetrics() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]any{
		"counters":   m.counters,
		"gauges":     m.gauges,
		"histograms": m.histograms,
	}
}

func (m *DefaultMetricsCollector) GetCounter(name string, tags map[string]string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

func (m *DefaultMetricsCollector) GetGauge(name string, tags map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

func (m *DefaultMetricsCollector) GetHistogramStats(name string, tags map[string]string) map[string]any {
	m.mu.RLock()
	values := m.histograms[name]
	m.mu.RUnlock()

	if len(values) == 0 {
		return map[string]any{"count": 0}
	}

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

	return map[string]any{
		"count": len(values),
		"sum":   sum,
		"min":   min,
		"max":   max,
		"avg":   sum / float64(len(values)),
	}
}

// DefaultHealthChecker is a default implementation of HealthChecker for testing
type DefaultHealthChecker struct {
	registry         core.PluginRegistry
	configManager    core.ConfigManager
	eventBus         core.EventBus
	metricsCollector core.MetricsCollector
	logger           core.Logger
}

func (h *DefaultHealthChecker) Check(ctx context.Context) (core.HealthStatus, error) {
	return core.HealthStatus{
		Status: "healthy",
		Details: map[string]any{
			"timestamp": time.Now(),
		},
	}, nil
}
