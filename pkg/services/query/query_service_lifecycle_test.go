package query

import (
	"context"
	"testing"

	"chainpulse/pkg/core"
)

type lifecycleCacheService struct {
	initialized bool
	running     bool
}

func (m *lifecycleCacheService) Initialize(ctx context.Context) error {
	_ = ctx
	m.initialized = true
	return nil
}

func (m *lifecycleCacheService) Start(ctx context.Context) error {
	_ = ctx
	m.running = true
	return nil
}

func (m *lifecycleCacheService) Stop(ctx context.Context) error {
	_ = ctx
	m.running = false
	return nil
}

func (m *lifecycleCacheService) Get(ctx context.Context, key string) ([]core.BlockchainEvent, error) {
	_ = ctx
	_ = key
	return nil, nil
}

func (m *lifecycleCacheService) GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error) {
	_ = ctx
	_ = key
	return nil, nil
}

func (m *lifecycleCacheService) Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl interface{}) error {
	_ = ctx
	_ = key
	_ = value
	_ = ttl
	return nil
}

func (m *lifecycleCacheService) SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl interface{}) error {
	_ = ctx
	_ = key
	_ = value
	_ = ttl
	return nil
}

func (m *lifecycleCacheService) Delete(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}

func (m *lifecycleCacheService) Health(ctx context.Context) *core.HealthStatus {
	_ = ctx
	if !m.running {
		return &core.HealthStatus{Status: "unhealthy"}
	}
	return &core.HealthStatus{Status: "healthy"}
}

func TestQueryServiceStartStopManagesCacheLifecycle(t *testing.T) {
	ctx := context.Background()
	cache := &lifecycleCacheService{}
	logger := &MockLogger{}
	metricsCollector := NewMockMetricsCollector()
	service := NewQueryService(
		nil,
		&MockMongoDBAdapter{healthy: true},
		&MockPostgreSQLAdapter{healthy: true},
		cache,
		logger,
		metricsCollector,
	)

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("initialize query service: %v", err)
	}
	if !cache.initialized {
		t.Fatal("expected cache service to be initialized")
	}

	if err := service.Start(ctx); err != nil {
		t.Fatalf("start query service: %v", err)
	}
	if !cache.running {
		t.Fatal("expected cache service to be started with query service")
	}

	if err := service.Stop(ctx); err != nil {
		t.Fatalf("stop query service: %v", err)
	}
	if cache.running {
		t.Fatal("expected cache service to be stopped with query service")
	}
}
