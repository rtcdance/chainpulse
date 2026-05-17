package query

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// MockEventStoreForRecovery for testing
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

// MockMetadataStore for testing
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

// MockCacheServiceForRecovery for testing
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

// Test initialization
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

// Test recovery with healthy stores
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

	err := handler.RecoverState(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// Test recovery with unhealthy event store
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

// Test recovery with unhealthy metadata store
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

// Test recovery with unhealthy cache service
func TestRecoveryWithUnhealthyCacheService(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: false, err: fmt.Errorf("cache error")}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = handler.Initialize(ctx)

	// RecoverState should succeed even if cache is unhealthy (it only checks event/metadata stores)
	err := handler.RecoverState(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Test recovery timeout
func TestRecoveryTimeout(t *testing.T) {
	t.Parallel()
	config := DefaultRecoveryConfig()
	eventStore := &MockEventStoreForRecovery{healthy: true}
	metadataStore := &MockMetadataStore{healthy: true}
	cacheService := &MockCacheServiceForRecovery{healthy: true}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	errorClassifier := NewErrorClassifier()

	handler := NewRecoveryHandler(config, eventStore, metadataStore, cacheService, errorClassifier, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_ = handler.Initialize(ctx)

	// RecoverState should complete even with a short timeout since it doesn't do blocking operations
	err := handler.RecoverState(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Test recovery with nil stores
func TestRecoveryWithNilStores(t *testing.T) {
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

// Test recovery status
func TestRecoveryStatus(t *testing.T) {
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
	if state == RecoveryStateHealthy {
		// State is healthy, which is expected
	} else {
		t.Error("Expected state to be healthy")
	}
}
