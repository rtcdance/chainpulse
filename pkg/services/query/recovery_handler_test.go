package query

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type MockEventStoreForRecovery struct {
	healthy bool
	err     error
}

func (m *MockEventStoreForRecovery) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockEventStoreForRecovery) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockEventStoreForRecovery) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockEventStoreForRecovery) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return &core.BlockchainEvent{ID: eventID}, nil
}

func (m *MockEventStoreForRecovery) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	if !m.healthy {
		return nil, false, m.err
	}
	return []*core.BlockchainEvent{}, false, nil
}

func (m *MockEventStoreForRecovery) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*core.BlockchainEvent{}, nil
}

func (m *MockEventStoreForRecovery) CountEvents(ctx context.Context) (int64, error) {
	if !m.healthy {
		return 0, m.err
	}
	return 0, nil
}

func (m *MockEventStoreForRecovery) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	if !m.healthy {
		return 0, m.err
	}
	return 0, nil
}

func (m *MockEventStoreForRecovery) Health(ctx context.Context) *core.HealthStatus {
	if m.healthy {
		return &core.HealthStatus{Status: "healthy"}
	}
	return &core.HealthStatus{Status: "unhealthy"}
}

func (m *MockEventStoreForRecovery) Close(ctx context.Context) error {
	return nil
}

type MockMetadataStore struct {
	healthy bool
	err     error
}

func (m *MockMetadataStore) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockMetadataStore) InsertMetadata(ctx context.Context, metadata *EventMetadata) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockMetadataStore) InsertMetadataBatch(ctx context.Context, metadataList []*EventMetadata) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockMetadataStore) GetMetadata(ctx context.Context, eventID string) (*EventMetadata, error) {
	if !m.healthy {
		return nil, m.err
	}
	return &EventMetadata{EventID: eventID}, nil
}

func (m *MockMetadataStore) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*EventMetadata, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []*EventMetadata{}, nil
}

func (m *MockMetadataStore) GetMetadataBatch(ctx context.Context, eventIDs []string) (map[string]*EventMetadata, error) {
	if !m.healthy {
		return nil, m.err
	}
	return nil, nil
}

func (m *MockMetadataStore) UpdateMetadata(ctx context.Context, metadata *EventMetadata) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockMetadataStore) Health(ctx context.Context) *core.HealthStatus {
	if m.healthy {
		return &core.HealthStatus{Status: "healthy"}
	}
	return &core.HealthStatus{Status: "unhealthy"}
}

func (m *MockMetadataStore) Close(ctx context.Context) error {
	return nil
}

type MockCacheServiceForRecovery struct {
	healthy bool
	err     error
}

func (m *MockCacheServiceForRecovery) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockCacheServiceForRecovery) Start(ctx context.Context) error {
	return nil
}

func (m *MockCacheServiceForRecovery) Stop(ctx context.Context) error {
	return nil
}

func (m *MockCacheServiceForRecovery) Get(ctx context.Context, key string) ([]core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return []core.BlockchainEvent{}, nil
}

func (m *MockCacheServiceForRecovery) GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error) {
	if !m.healthy {
		return nil, m.err
	}
	return &core.BlockchainEvent{}, nil
}

func (m *MockCacheServiceForRecovery) Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl time.Duration) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockCacheServiceForRecovery) SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl time.Duration) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockCacheServiceForRecovery) SetQueryResult(ctx context.Context, key string, events []core.BlockchainEvent, total int64, ttl time.Duration) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockCacheServiceForRecovery) GetQueryResult(ctx context.Context, key string) ([]core.BlockchainEvent, int64, error) {
	if !m.healthy {
		return nil, 0, m.err
	}
	return []core.BlockchainEvent{}, 0, nil
}

func (m *MockCacheServiceForRecovery) Delete(ctx context.Context, key string) error {
	if !m.healthy {
		return m.err
	}
	return nil
}

func (m *MockCacheServiceForRecovery) Health(ctx context.Context) *core.HealthStatus {
	if m.healthy {
		return &core.HealthStatus{Status: "healthy"}
	}
	return &core.HealthStatus{Status: "unhealthy"}
}

func TestDefaultRecoveryHandlerConfig(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()

	if config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", config.MaxRetries)
	}
	if config.InitialBackoff != 100*time.Millisecond {
		t.Errorf("expected InitialBackoff 100ms, got %v", config.InitialBackoff)
	}
	if config.MaxBackoff != 10*time.Second {
		t.Errorf("expected MaxBackoff 10s, got %v", config.MaxBackoff)
	}
	if config.BackoffMultiplier != 2.0 {
		t.Errorf("expected BackoffMultiplier 2.0, got %f", config.BackoffMultiplier)
	}
}

func TestRecoveryHandlerInitialize(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := handler.Initialize(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRecoveryHandlerInitializeNilStores(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, nil, nil, nil, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := handler.Initialize(ctx)
	if err == nil {
		t.Error("Expected error for nil stores")
	}
}

func TestRecoveryWithHealthyStores(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = handler.Initialize(ctx)

	state := handler.GetRecoveryState()
	if state != RecoveryStateHealthy {
		t.Errorf("expected healthy state, got %v", state)
	}

	err := handler.RecoverState(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	recoveryMetrics := handler.GetRecoveryMetrics()
	if recoveryMetrics.SuccessfulRecoveries < 1 {
		t.Errorf("expected at least 1 successful recovery, got %d", recoveryMetrics.SuccessfulRecoveries)
	}
}

func TestRecoveryWithUnhealthyEventStore(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: false, err: fmt.Errorf("store error")}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = handler.Initialize(ctx)

	err := handler.RecoverState(ctx)
	if err == nil {
		t.Error("Expected error for unhealthy event store")
	}
}

func TestRecoveryWithUnhealthyMetadataStore(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: false, err: fmt.Errorf("store error")}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = handler.Initialize(ctx)

	err := handler.RecoverState(ctx)
	if err == nil {
		t.Error("Expected error for unhealthy metadata store")
	}
}

func TestRecoveryHandlerGetRecoveryState(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	state := handler.GetRecoveryState()
	if state != RecoveryStateHealthy {
		t.Errorf("expected healthy, got %v", state)
	}
}

func TestRecoveryHandlerGetLastRecoveryTime(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	lastTime := handler.GetLastRecoveryTime()
	if !lastTime.IsZero() {
		t.Errorf("expected zero time for new handler, got %v", lastTime)
	}
}

func TestRecoveryHandlerGetRecoveryMetrics(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	rm := handler.GetRecoveryMetrics()
	if rm.TotalRecoveryAttempts != 0 {
		t.Errorf("expected 0 attempts, got %d", rm.TotalRecoveryAttempts)
	}
}

func TestRecoveryHandlerCloseNotInitialized(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	err := handler.Close(context.Background())
	if err != nil {
		t.Errorf("expected no error on close when not initialized, got %v", err)
	}
}

func TestRecoveryHandlerCloseInitialized(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = handler.Initialize(ctx)

	err := handler.Close(context.Background())
	if err != nil {
		t.Errorf("expected no error on close, got %v", err)
	}
}

func TestRecoveryNotInitialized(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := handler.RecoverState(ctx)
	if err == nil {
		t.Error("expected error when not initialized")
	}

	err = handler.RecoverConnection(ctx, "mongodb")
	if err == nil {
		t.Error("expected error when not initialized")
	}

	err = handler.SyncData(ctx, "mongodb")
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestRecoveryHandlerRecoverConnection(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = handler.Initialize(ctx)

	err := handler.RecoverConnection(ctx, "mongodb")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRecoveryHandlerRecoverConnectionEmptyStore(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = handler.Initialize(ctx)

	err := handler.RecoverConnection(ctx, "")
	if err == nil {
		t.Error("expected error for empty store name")
	}
}

func TestRecoveryHandlerRecoverConnectionUnknown(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = handler.Initialize(ctx)

	err := handler.RecoverConnection(ctx, "unknown")
	if err == nil {
		t.Error("expected error for unknown store")
	}
}

func TestRecoveryHandlerSyncData(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = handler.Initialize(ctx)

	err := handler.SyncData(ctx, "mongodb")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRecoveryHandlerSyncDataUnhealthy(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: false, err: fmt.Errorf("store error")}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = handler.Initialize(ctx)

	err := handler.SyncData(ctx, "mongodb")
	if err == nil {
		t.Error("expected error for unhealthy store")
	}
}

func TestRecoveryHandlerStateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    RecoveryState
		expected string
	}{
		{RecoveryStateHealthy, "healthy"},
		{RecoveryStateRecovering, "recovering"},
		{RecoveryStateFailed, "failed"},
		{RecoveryState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}