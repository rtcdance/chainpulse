package query

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// mockEventStoreDegradation for testing
type mockEventStoreDegradation struct {
	healthy bool
}

func (m *mockEventStoreDegradation) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockEventStoreDegradation) InsertEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	return nil
}

func (m *mockEventStoreDegradation) InsertEventBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error {
	return nil
}

func (m *mockEventStoreDegradation) GetEvent(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*blockchain.BlockchainEvent, bool, error) {
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

func (m *mockEventStoreDegradation) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockEventStoreDegradation) GetEventStats(ctx context.Context) (map[string]int64, map[string]int64, int64, error) {
	return make(map[string]int64), make(map[string]int64), 0, nil
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

func (m *mockMetadataStoreDegradation) GetMetadataBatch(ctx context.Context, eventIDs []string) (map[string]*EventMetadata, error) {
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

func (m *MockCacheService) Get(ctx context.Context, key string) ([]blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *MockCacheService) GetSingle(ctx context.Context, key string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []blockchain.BlockchainEvent, ttl time.Duration) error {
	return nil
}

func (m *MockCacheService) SetSingle(ctx context.Context, key string, value *blockchain.BlockchainEvent, ttl time.Duration) error {
	return nil
}

func (m *MockCacheService) SetQueryResult(ctx context.Context, key string, events []blockchain.BlockchainEvent, total int64, ttl time.Duration) error {
	return nil
}

func (m *MockCacheService) GetQueryResult(ctx context.Context, key string) ([]blockchain.BlockchainEvent, int64, error) {
	return nil, 0, nil
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

func (m *MockLogger) Debug(msg string, fields ...any) {}
func (m *MockLogger) Info(msg string, fields ...any)  {}
func (m *MockLogger) Warn(msg string, fields ...any)  {}
func (m *MockLogger) Error(msg string, fields ...any) {}
func (m *MockLogger) Fatal(msg string, fields ...any) {}
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

func (m *MockMetricsCollector) GetMetrics() map[string]any {
	result := make(map[string]any)
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestDegradationModeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode     DegradationMode
		expected string
	}{
		{DegradationModeNormal, "Normal"},
		{DegradationModeMongoDBAnavailable, "MongoDBUnavailable"},
		{DegradationModePostgreSQLUnavailable, "PostgreSQLUnavailable"},
		{DegradationModeBothUnavailable, "BothUnavailable"},
		{DegradationModeCacheUnavailable, "CacheUnavailable"},
		{DegradationModeReadOnly, "ReadOnly"},
		{DegradationMode(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.expected {
			t.Errorf("DegradationMode(%d).String() = %q, want %q", tt.mode, got, tt.expected)
		}
	}
}

func TestCacheOnlyStrategy(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		s := NewCacheOnlyStrategy(nil)
		if s.Name() != "CacheOnly" {
			t.Errorf("Expected CacheOnly, got %s", s.Name())
		}
	})

	t.Run("CanRetrieveEvent nil", func(t *testing.T) {
		t.Parallel()
		s := NewCacheOnlyStrategy(nil)
		if s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected false with nil cache")
		}
	})

	t.Run("CanRetrieveEvent non-nil", func(t *testing.T) {
		t.Parallel()
		s := NewCacheOnlyStrategy(&MockCacheService{healthy: true})
		if !s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected true with non-nil cache")
		}
	})

	t.Run("CanRetrieveMetadata", func(t *testing.T) {
		t.Parallel()
		s := NewCacheOnlyStrategy(nil)
		if s.CanRetrieveMetadata(context.Background()) {
			t.Error("Expected false")
		}
	})

	t.Run("CanWrite", func(t *testing.T) {
		t.Parallel()
		s := NewCacheOnlyStrategy(nil)
		if s.CanWrite(context.Background()) {
			t.Error("Expected false")
		}
	})
}

func TestMongoDBOnlyStrategy(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		s := NewMongoDBOnlyStrategy(nil)
		if s.Name() != "MongoDBOnly" {
			t.Errorf("Expected MongoDBOnly, got %s", s.Name())
		}
	})

	t.Run("CanRetrieveEvent nil", func(t *testing.T) {
		t.Parallel()
		s := NewMongoDBOnlyStrategy(nil)
		if s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected false with nil store")
		}
	})

	t.Run("CanRetrieveEvent non-nil", func(t *testing.T) {
		t.Parallel()
		s := NewMongoDBOnlyStrategy(&mockEventStoreDegradation{healthy: true})
		if !s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected true with non-nil store")
		}
	})

	t.Run("CanRetrieveMetadata", func(t *testing.T) {
		t.Parallel()
		s := NewMongoDBOnlyStrategy(nil)
		if s.CanRetrieveMetadata(context.Background()) {
			t.Error("Expected false")
		}
	})

	t.Run("CanWrite nil", func(t *testing.T) {
		t.Parallel()
		s := NewMongoDBOnlyStrategy(nil)
		if s.CanWrite(context.Background()) {
			t.Error("Expected false with nil store")
		}
	})

	t.Run("CanWrite non-nil", func(t *testing.T) {
		t.Parallel()
		s := NewMongoDBOnlyStrategy(&mockEventStoreDegradation{healthy: true})
		if !s.CanWrite(context.Background()) {
			t.Error("Expected true with non-nil store")
		}
	})
}

func TestPostgreSQLOnlyStrategy(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(nil, nil)
		if s.Name() != "PostgreSQLOnly" {
			t.Errorf("Expected PostgreSQLOnly, got %s", s.Name())
		}
	})

	t.Run("CanRetrieveEvent nil cache", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(nil, nil)
		if s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected false with nil cache")
		}
	})

	t.Run("CanRetrieveEvent non-nil cache", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(nil, &MockCacheService{healthy: true})
		if !s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected true with non-nil cache")
		}
	})

	t.Run("CanRetrieveMetadata nil", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(nil, nil)
		if s.CanRetrieveMetadata(context.Background()) {
			t.Error("Expected false with nil metadata store")
		}
	})

	t.Run("CanRetrieveMetadata non-nil", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(&mockMetadataStoreDegradation{healthy: true}, nil)
		if !s.CanRetrieveMetadata(context.Background()) {
			t.Error("Expected true with non-nil metadata store")
		}
	})

	t.Run("CanWrite nil", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(nil, nil)
		if s.CanWrite(context.Background()) {
			t.Error("Expected false with nil metadata store")
		}
	})

	t.Run("CanWrite non-nil", func(t *testing.T) {
		t.Parallel()
		s := NewPostgreSQLOnlyStrategy(&mockMetadataStoreDegradation{healthy: true}, nil)
		if !s.CanWrite(context.Background()) {
			t.Error("Expected true with non-nil metadata store")
		}
	})
}

func TestHybridStrategy(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, nil, nil)
		if s.Name() != "Hybrid" {
			t.Errorf("Expected Hybrid, got %s", s.Name())
		}
	})

	t.Run("CanRetrieveEvent all nil", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, nil, nil)
		if s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected false with all nil")
		}
	})

	t.Run("CanRetrieveEvent with event store", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(&mockEventStoreDegradation{healthy: true}, nil, nil)
		if !s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected true with event store")
		}
	})

	t.Run("CanRetrieveEvent with cache", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, nil, &MockCacheService{healthy: true})
		if !s.CanRetrieveEvent(context.Background()) {
			t.Error("Expected true with cache service")
		}
	})

	t.Run("CanRetrieveMetadata nil", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, nil, nil)
		if s.CanRetrieveMetadata(context.Background()) {
			t.Error("Expected false with nil metadata store")
		}
	})

	t.Run("CanRetrieveMetadata non-nil", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, &mockMetadataStoreDegradation{healthy: true}, nil)
		if !s.CanRetrieveMetadata(context.Background()) {
			t.Error("Expected true with non-nil metadata store")
		}
	})

	t.Run("CanWrite all nil", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, nil, nil)
		if s.CanWrite(context.Background()) {
			t.Error("Expected false with all nil")
		}
	})

	t.Run("CanWrite with event store", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(&mockEventStoreDegradation{healthy: true}, nil, nil)
		if !s.CanWrite(context.Background()) {
			t.Error("Expected true with event store")
		}
	})

	t.Run("CanWrite with metadata store", func(t *testing.T) {
		t.Parallel()
		s := NewHybridStrategy(nil, &mockMetadataStoreDegradation{healthy: true}, nil)
		if !s.CanWrite(context.Background()) {
			t.Error("Expected true with metadata store")
		}
	})
}

func TestReadOnlyStrategy(t *testing.T) {
	t.Parallel()

	s := NewReadOnlyStrategy()

	if s.Name() != "ReadOnly" {
		t.Errorf("Expected ReadOnly, got %s", s.Name())
	}

	if s.CanRetrieveEvent(context.Background()) {
		t.Error("Expected false for CanRetrieveEvent")
	}

	if s.CanRetrieveMetadata(context.Background()) {
		t.Error("Expected false for CanRetrieveMetadata")
	}

	if s.CanWrite(context.Background()) {
		t.Error("Expected false for CanWrite")
	}
}

func TestSelectStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		mongoHealthy    bool
		postgresHealthy bool
		cacheHealthy    bool
		expectedName    string
	}{
		{"Normal", true, true, true, "Hybrid"},
		{"MongoDBUnavailable", false, true, true, "PostgreSQLOnly"},
		{"PostgreSQLUnavailable", true, false, true, "MongoDBOnly"},
		{"BothUnavailable", false, false, true, "CacheOnly"},
		{"CacheUnavailable", true, true, false, "Hybrid"},
		{"ReadOnly", false, false, false, "ReadOnly"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			eventStore := &mockEventStoreDegradation{healthy: tt.mongoHealthy}
			metadataStore := &mockMetadataStoreDegradation{healthy: tt.postgresHealthy}
			cacheService := &MockCacheService{healthy: tt.cacheHealthy}
			logger := &MockLogger{}
			metrics := NewMockMetricsCollector()

			handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
			_ = handler.Initialize(ctx)

			strategy := handler.SelectStrategy(ctx)
			if strategy.Name() != tt.expectedName {
				t.Errorf("Expected strategy %s, got %s", tt.expectedName, strategy.Name())
			}
		})
	}
}

func TestDegradationHandlerUninitializedMode(t *testing.T) {
	t.Parallel()

	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	// Don't initialize

	mode := handler.GetDegradationMode(context.Background())
	if mode != DegradationModeReadOnly {
		t.Errorf("Expected DegradationModeReadOnly for uninitialized handler, got %v", mode)
	}
}

func TestDegradationHandlerClose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	eventStore := &mockEventStoreDegradation{healthy: true}
	metadataStore := &mockMetadataStoreDegradation{healthy: true}
	cacheService := &MockCacheService{healthy: true}
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	handler := NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
	_ = handler.Initialize(ctx)

	if err := handler.Close(ctx); err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}

	mode := handler.GetDegradationMode(ctx)
	if mode != DegradationModeReadOnly {
		t.Errorf("Expected DegradationModeReadOnly after close, got %v", mode)
	}
}

func TestDegradationHandlerHealth(t *testing.T) {
	t.Parallel()

	t.Run("uninitialized", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		handler := NewDegradationHandler(nil, nil, nil, &MockLogger{}, NewMockMetricsCollector())
		health := handler.Health(ctx)
		if health.Status != "unhealthy" {
			t.Errorf("Expected unhealthy, got %s", health.Status)
		}
	})

	t.Run("healthy", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		eventStore := &mockEventStoreDegradation{healthy: true}
		metadataStore := &mockMetadataStoreDegradation{healthy: true}
		cacheService := &MockCacheService{healthy: true}
		handler := NewDegradationHandler(eventStore, metadataStore, cacheService, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		health := handler.Health(ctx)
		if health.Status != "healthy" {
			t.Errorf("Expected healthy, got %s", health.Status)
		}
	})

	t.Run("readonly", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		eventStore := &mockEventStoreDegradation{healthy: false}
		metadataStore := &mockMetadataStoreDegradation{healthy: false}
		cacheService := &MockCacheService{healthy: false}
		handler := NewDegradationHandler(eventStore, metadataStore, cacheService, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		health := handler.Health(ctx)
		if health.Status != "unhealthy" {
			t.Errorf("Expected unhealthy, got %s", health.Status)
		}
	})

	t.Run("degraded", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		eventStore := &mockEventStoreDegradation{healthy: false}
		metadataStore := &mockMetadataStoreDegradation{healthy: true}
		cacheService := &MockCacheService{healthy: true}
		handler := NewDegradationHandler(eventStore, metadataStore, cacheService, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		health := handler.Health(ctx)
		if health.Status != "degraded" {
			t.Errorf("Expected degraded, got %s", health.Status)
		}
	})
}

func TestCanUseMongoDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("nil event store", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, nil, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if handler.CanUseMongoDB(ctx) {
			t.Error("Expected false with nil event store")
		}
	})

	t.Run("unhealthy event store", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(&mockEventStoreDegradation{healthy: false}, nil, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if handler.CanUseMongoDB(ctx) {
			t.Error("Expected false with unhealthy event store")
		}
	})

	t.Run("healthy event store", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(&mockEventStoreDegradation{healthy: true}, nil, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if !handler.CanUseMongoDB(ctx) {
			t.Error("Expected true with healthy event store")
		}
	})
}

func TestCanUsePostgreSQL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("nil metadata store", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, nil, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if handler.CanUsePostgreSQL(ctx) {
			t.Error("Expected false with nil metadata store")
		}
	})

	t.Run("unhealthy metadata store", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, &mockMetadataStoreDegradation{healthy: false}, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if handler.CanUsePostgreSQL(ctx) {
			t.Error("Expected false with unhealthy metadata store")
		}
	})

	t.Run("healthy metadata store", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, &mockMetadataStoreDegradation{healthy: true}, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if !handler.CanUsePostgreSQL(ctx) {
			t.Error("Expected true with healthy metadata store")
		}
	})
}

func TestCanUseCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("nil cache service", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, nil, nil, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if handler.CanUseCache(ctx) {
			t.Error("Expected false with nil cache service")
		}
	})

	t.Run("unhealthy cache service", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, nil, &MockCacheService{healthy: false}, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if handler.CanUseCache(ctx) {
			t.Error("Expected false with unhealthy cache")
		}
	})

	t.Run("healthy cache service", func(t *testing.T) {
		t.Parallel()
		handler := NewDegradationHandler(nil, nil, &MockCacheService{healthy: true}, &MockLogger{}, NewMockMetricsCollector())
		_ = handler.Initialize(ctx)
		if !handler.CanUseCache(ctx) {
			t.Error("Expected true with healthy cache")
		}
	})
}
