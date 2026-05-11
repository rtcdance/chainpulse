package query

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// mockEventStoreDegradation for testing
type mockEventStoreDegradation struct {
	healthy bool
}

func (m *mockEventStoreDegradation) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockEventStoreDegradation) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	return nil
}

func (m *mockEventStoreDegradation) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	return nil
}

func (m *mockEventStoreDegradation) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	return nil, false, nil
}

func (m *mockEventStoreDegradation) CountEvents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockEventStoreDegradation) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockEventStoreDegradation) Health(ctx context.Context) *core.HealthStatus {
	if m.healthy {
		return &core.HealthStatus{Status: "healthy"}
	}
	return &core.HealthStatus{Status: "unhealthy"}
}

func (m *mockEventStoreDegradation) Close(ctx context.Context) error {
	return nil
}

// MockEventMetadataStore for testing
type mockMetadataStoreDegradation struct {
	healthy bool
}

func (m *mockMetadataStoreDegradation) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockMetadataStoreDegradation) InsertMetadata(ctx context.Context, metadata *EventMetadata) error {
	return nil
}

func (m *mockMetadataStoreDegradation) InsertMetadataBatch(ctx context.Context, metadataList []*EventMetadata) error {
	return nil
}

func (m *mockMetadataStoreDegradation) GetMetadata(ctx context.Context, eventID string) (*EventMetadata, error) {
	return nil, nil
}

func (m *mockMetadataStoreDegradation) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*EventMetadata, error) {
	return nil, nil
}

func (m *mockMetadataStoreDegradation) UpdateMetadata(ctx context.Context, metadata *EventMetadata) error {
	return nil
}

func (m *mockMetadataStoreDegradation) Health(ctx context.Context) *core.HealthStatus {
	if m.healthy {
		return &core.HealthStatus{Status: "healthy"}
	}
	return &core.HealthStatus{Status: "unhealthy"}
}

func (m *mockMetadataStoreDegradation) Close(ctx context.Context) error {
	return nil
}

// MockCacheService for testing
type MockCacheService struct {
	healthy bool
}

func (m *MockCacheService) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockCacheService) Start(ctx context.Context) error {
	return nil
}

func (m *MockCacheService) Stop(ctx context.Context) error {
	return nil
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]core.BlockchainEvent, error) {
	return nil, nil
}

func (m *MockCacheService) GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl interface{}) error {
	return nil
}

func (m *MockCacheService) SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl interface{}) error {
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCacheService) Health(ctx context.Context) *core.HealthStatus {
	if m.healthy {
		return &core.HealthStatus{Status: "healthy"}
	}
	return &core.HealthStatus{Status: "unhealthy"}
}

// MockLogger for testing
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {}
func (m *MockLogger) Info(msg string, fields ...interface{})  {}
func (m *MockLogger) Warn(msg string, fields ...interface{})  {}
func (m *MockLogger) Error(msg string, fields ...interface{}) {}
func (m *MockLogger) Fatal(msg string, fields ...interface{}) {}
func (m *MockLogger) WithCorrelationID(id string) core.Logger {
	return m
}

// MockMetricsCollector for testing
type MockMetricsCollector struct {
	metrics map[string]float64
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: make(map[string]float64),
	}
}

func (m *MockMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	m.metrics[name] = float64(value)
}

func (m *MockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.metrics[name] = value
}

func (m *MockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	m.metrics[name] = value
}

func (m *MockMetricsCollector) GetMetrics() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

func (m *MockMetricsCollector) GetMetric(name string) float64 {
	return m.metrics[name]
}

// Tests

func TestDegradationHandlerInitialization(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)

	err := handler.Initialize(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Initialize again should not error
	err = handler.Initialize(ctx)
	if err != nil {
		t.Fatalf("Expected no error on second initialize, got %v", err)
	}
}

func TestDegradationHandlerNilStores(t *testing.T) {
	ctx := context.Background()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(nil, nil, nil, logger, metrics)

	err := handler.Initialize(ctx)
	if err == nil {
		t.Fatal("Expected error for nil event store")
	}
}

func TestDegradationModeNormal(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	if err := handler.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize handler: %v", err)
	}

	mode := handler.GetDegradationMode(ctx)
	if mode != DegradationModeNormal {
		t.Fatalf("Expected DegradationModeNormal, got %v", mode)
	}
}

func TestDegradationModeMongoDBAnavailable(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: false}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	if err := handler.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize handler: %v", err)
	}

	mode := handler.GetDegradationMode(ctx)
	if mode != DegradationModeMongoDBAnavailable {
		t.Fatalf("Expected DegradationModeMongoDBAnavailable, got %v", mode)
	}
}

func TestDegradationModePostgreSQLUnavailable(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: false}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	if err := handler.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize handler: %v", err)
	}

	mode := handler.GetDegradationMode(ctx)
	if mode != DegradationModePostgreSQLUnavailable {
		t.Fatalf("Expected DegradationModePostgreSQLUnavailable, got %v", mode)
	}
}

func TestDegradationModeCacheUnavailable(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: false}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	_ = handler.Initialize(ctx)

	mode := handler.GetDegradationMode(ctx)
	if mode != DegradationModeCacheUnavailable {
		t.Fatalf("Expected DegradationModeCacheUnavailable, got %v", mode)
	}
}

func TestDegradationModeMultipleUnavailable(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: false}
	metadataStore := &mockMetadataStoreDegradation{healthy: false}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	_ = handler.Initialize(ctx)

	mode := handler.GetDegradationMode(ctx)
	if mode != DegradationModeBothUnavailable {
		t.Fatalf("Expected DegradationModeBothUnavailable, got %v", mode)
	}
}

func TestDegradationHandlerMetrics(t *testing.T) {
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	_ = handler.Initialize(ctx)

	// Record a degradation event to trigger metrics recording
	handler.RecordDegradation(ctx, DegradationModeNormal, "test")

	// Metrics should be recorded
	if metrics.GetMetric("degradation_mode_changes_total") == 0 {
		t.Error("Expected degradation mode metric to be recorded")
	}
}

func TestDegradationHandlerTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)

	// Initialize should complete even with a short timeout since it doesn't do blocking operations
	err := handler.Initialize(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
