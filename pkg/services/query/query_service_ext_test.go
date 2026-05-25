package query

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type configurableMockAdapter struct {
	mu            sync.RWMutex
	healthy       bool
	initErr       error
	queryFn       func(ctx context.Context, req *QueryRequest) (*QueryResult, error)
	queryByHashFn func(ctx context.Context, hash string) (*core.BlockchainEvent, error)
}

func (m *configurableMockAdapter) Initialize(ctx context.Context) error {
	return m.initErr
}

func (m *configurableMockAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.queryFn != nil {
		return m.queryFn(ctx, req)
	}
	return &QueryResult{Events: []core.BlockchainEvent{{ID: "test"}}, Total: 1}, nil
}

func (m *configurableMockAdapter) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.queryByHashFn != nil {
		return m.queryByHashFn(ctx, hash)
	}
	return nil, nil
}

func (m *configurableMockAdapter) Health(ctx context.Context) *core.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := "healthy"
	if !m.healthy {
		status = "unhealthy"
	}
	return &core.HealthStatus{Status: status}
}

type queryServiceCacheMock struct {
	mu              sync.RWMutex
	initialized     bool
	running         bool
	healthStatus    string
	initErr         error
	startErr        error
	stopErr         error
	queryCacheData  []core.BlockchainEvent
	queryCacheTotal int64
	queryCacheErr   error
	singleData      *core.BlockchainEvent
	singleErr       error
	deleteErr       error
}

func (m *queryServiceCacheMock) Initialize(ctx context.Context) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.initialized = true
	return nil
}

func (m *queryServiceCacheMock) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.running = true
	return nil
}

func (m *queryServiceCacheMock) Stop(ctx context.Context) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.running = false
	return nil
}

func (m *queryServiceCacheMock) Get(ctx context.Context, key string) ([]core.BlockchainEvent, error) {
	return nil, nil
}

func (m *queryServiceCacheMock) GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.singleErr != nil {
		return nil, m.singleErr
	}
	return m.singleData, nil
}

func (m *queryServiceCacheMock) Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl time.Duration) error {
	return nil
}

func (m *queryServiceCacheMock) SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl time.Duration) error {
	return nil
}

func (m *queryServiceCacheMock) SetQueryResult(ctx context.Context, key string, events []core.BlockchainEvent, total int64, ttl time.Duration) error {
	return nil
}

func (m *queryServiceCacheMock) GetQueryResult(ctx context.Context, key string) ([]core.BlockchainEvent, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queryCacheData, m.queryCacheTotal, m.queryCacheErr
}

func (m *queryServiceCacheMock) Delete(ctx context.Context, key string) error {
	return m.deleteErr
}

func (m *queryServiceCacheMock) Health(ctx context.Context) *core.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &core.HealthStatus{Status: m.healthStatus}
}

func newQueryServiceForTest(t *testing.T, mongo *configurableMockAdapter, postgres *configurableMockAdapter, cache *queryServiceCacheMock) *DefaultQueryService {
	t.Helper()
	m := mongo
	if m == nil {
		m = &configurableMockAdapter{healthy: true}
	}
	p := postgres
	if p == nil {
		p = &configurableMockAdapter{healthy: true}
	}
	c := cache
	if c == nil {
		c = &queryServiceCacheMock{healthStatus: "healthy"}
	}
	svc := NewQueryService(m, p, c, &MockLogger{}, NewMockMetricsCollector())
	return svc
}

func initQueryService(t *testing.T, svc *DefaultQueryService) {
	t.Helper()
	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestClassifyCachePosture(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"healthy", "healthy", "cache-ready"},
		{"degraded", "degraded", "cache-degraded"},
		{"unhealthy", "unhealthy", "cache-unhealthy"},
		{"unknown", "unknown", "cache-unobserved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cache := &queryServiceCacheMock{healthStatus: tt.status}
			svc := newQueryServiceForTest(t, nil, nil, cache)
			got := svc.classifyCachePosture(context.Background())
			if got != tt.want {
				t.Errorf("classifyCachePosture = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyCachePosture_NilCache(t *testing.T) {
	t.Parallel()
	svc := &DefaultQueryService{}
	got := svc.classifyCachePosture(context.Background())
	if got != "cache-unavailable" {
		t.Errorf("classifyCachePosture with nil cache = %q, want %q", got, "cache-unavailable")
	}
}

func TestQueryService_Health_NotInitialized(t *testing.T) {
	t.Parallel()
	svc := NewQueryService(&configurableMockAdapter{}, &configurableMockAdapter{}, nil, &MockLogger{}, NewMockMetricsCollector())
	h := svc.Health(context.Background())
	if h.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", h.Status)
	}
}

func TestQueryService_Health_NotRunning(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	ctx := context.Background()
	_ = svc.Initialize(ctx)
	h := svc.Health(ctx)
	if h.Status != "unhealthy" {
		t.Errorf("expected unhealthy (not running), got %s", h.Status)
	}
}

func TestQueryService_Health_BothBackendsUnhealthy(t *testing.T) {
	t.Parallel()
	mongo := &configurableMockAdapter{healthy: false}
	postgres := &configurableMockAdapter{healthy: false}
	svc := newQueryServiceForTest(t, mongo, postgres, nil)
	initQueryService(t, svc)
	h := svc.Health(context.Background())
	if h.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", h.Status)
	}
}

func TestQueryService_Health_OneBackendDegraded(t *testing.T) {
	t.Parallel()
	mongo := &configurableMockAdapter{healthy: false}
	postgres := &configurableMockAdapter{healthy: true}
	svc := newQueryServiceForTest(t, mongo, postgres, nil)
	initQueryService(t, svc)
	h := svc.Health(context.Background())
	if h.Status != "degraded" {
		t.Errorf("expected degraded, got %s", h.Status)
	}
}

func TestQueryService_Health_CacheDegraded(t *testing.T) {
	t.Parallel()
	cache := &queryServiceCacheMock{healthStatus: "unhealthy"}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	initQueryService(t, svc)
	h := svc.Health(context.Background())
	if h.Status != "degraded" {
		t.Errorf("expected degraded, got %s", h.Status)
	}
}

func TestQueryService_Health_Healthy(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	h := svc.Health(context.Background())
	if h.Status != "healthy" {
		t.Errorf("expected healthy, got %s", h.Status)
	}
}

func TestQueryService_RuntimeSummary(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	rs := svc.RuntimeSummary(context.Background())
	if rs.Status != "healthy" {
		t.Errorf("expected healthy, got %s", rs.Status)
	}
	if rs.QueryPosture != "query-runtime-ready" {
		t.Errorf("expected query-runtime-ready, got %s", rs.QueryPosture)
	}
	if rs.CachePosture != "cache-ready" {
		t.Errorf("expected cache-ready, got %s", rs.CachePosture)
	}
	if rs.ReliabilityHint == "" {
		t.Error("expected non-empty reliability hint")
	}
}

func TestQueryService_RuntimeSummary_NotRunning(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	ctx := context.Background()
	_ = svc.Initialize(ctx)
	rs := svc.RuntimeSummary(ctx)
	if rs.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", rs.Status)
	}
}

func TestQueryService_Query_NotRunning(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	_, err := svc.Query(context.Background(), &QueryRequest{})
	if err == nil {
		t.Fatal("expected error for not running")
	}
}

func TestQueryService_Query_NilRequest(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	_, err := svc.Query(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestQueryService_Query_EmptyResult(t *testing.T) {
	t.Parallel()
	emptyResultFn := func(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
		return &QueryResult{Events: []core.BlockchainEvent{}, Total: 0}, nil
	}
	mongo := &configurableMockAdapter{healthy: true, queryFn: emptyResultFn}
	postgres := &configurableMockAdapter{healthy: true, queryFn: emptyResultFn}
	svc := newQueryServiceForTest(t, mongo, postgres, nil)
	initQueryService(t, svc)
	result, err := svc.Query(context.Background(), &QueryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(result.Events))
	}
}

func TestQueryService_Query_CacheHit(t *testing.T) {
	t.Parallel()
	event := core.BlockchainEvent{ID: "cached-event"}
	cache := &queryServiceCacheMock{
		healthStatus:   "healthy",
		queryCacheData: []core.BlockchainEvent{event},
		queryCacheTotal: 1,
	}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	initQueryService(t, svc)
	result, err := svc.Query(context.Background(), &QueryRequest{CacheKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.CacheHit {
		t.Error("expected cache hit")
	}
	if result.Source != "cache" {
		t.Errorf("expected source=cache, got %s", result.Source)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}
}

func TestQueryService_QueryByHash_NotRunning(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	_, err := svc.QueryByHash(context.Background(), "hash")
	if err == nil {
		t.Fatal("expected error for not running")
	}
}

func TestQueryService_QueryByHash_EmptyHash(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	_, err := svc.QueryByHash(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty hash")
	}
}

func TestQueryService_QueryByHash_NotFound(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	_, err := svc.QueryByHash(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent hash")
	}
}

func TestQueryService_QueryByHash_CacheHit(t *testing.T) {
	t.Parallel()
	event := &core.BlockchainEvent{ID: "cached-hash"}
	cache := &queryServiceCacheMock{
		healthStatus: "healthy",
		singleData:   event,
	}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	initQueryService(t, svc)
	result, err := svc.QueryByHash(context.Background(), "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "cached-hash" {
		t.Errorf("expected cached-hash, got %s", result.ID)
	}
}

func TestQueryService_QueryByHash_MongoHit(t *testing.T) {
	t.Parallel()
	event := &core.BlockchainEvent{ID: "mongo-hash"}
	mongo := &configurableMockAdapter{
		healthy: true,
		queryByHashFn: func(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
			return event, nil
		},
	}
	svc := newQueryServiceForTest(t, mongo, nil, nil)
	initQueryService(t, svc)
	result, err := svc.QueryByHash(context.Background(), "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "mongo-hash" {
		t.Errorf("expected mongo-hash, got %s", result.ID)
	}
}

func TestQueryService_InvalidateCache_NotRunning(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	err := svc.InvalidateCache(context.Background(), "key")
	if err == nil {
		t.Fatal("expected error for not running")
	}
}

func TestQueryService_InvalidateCache_EmptyKey(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	err := svc.InvalidateCache(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestQueryService_InvalidateCache_Success(t *testing.T) {
	t.Parallel()
	cache := &queryServiceCacheMock{healthStatus: "healthy"}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	initQueryService(t, svc)
	err := svc.InvalidateCache(context.Background(), "valid-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryService_InvalidateCache_DeleteError(t *testing.T) {
	t.Parallel()
	cache := &queryServiceCacheMock{
		healthStatus: "healthy",
		deleteErr:    errors.New("cache delete failed"),
	}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	initQueryService(t, svc)
	err := svc.InvalidateCache(context.Background(), "fail-key")
	if err == nil {
		t.Fatal("expected error from cache delete")
	}
}

func TestQueryService_InitializeTwice(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}
	if err := svc.Initialize(ctx); err == nil {
		t.Fatal("expected error on second Initialize")
	}
}

func TestQueryService_StartBeforeInit(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	err := svc.Start(context.Background())
	if err == nil {
		t.Fatal("expected error starting uninitialized service")
	}
}

func TestQueryService_StartTwice(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	initQueryService(t, svc)
	err := svc.Start(context.Background())
	if err == nil {
		t.Fatal("expected error on second Start")
	}
}

func TestQueryService_StopWhenNotRunning(t *testing.T) {
	t.Parallel()
	svc := newQueryServiceForTest(t, nil, nil, nil)
	ctx := context.Background()
	_ = svc.Initialize(ctx)
	err := svc.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop when not running should not error: %v", err)
	}
}

func TestQueryService_Initialize_MongoError(t *testing.T) {
	t.Parallel()
	mongo := &configurableMockAdapter{healthy: true, initErr: errors.New("mongo init failed")}
	svc := newQueryServiceForTest(t, mongo, nil, nil)
	err := svc.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error when mongo fails")
	}
}

func TestQueryService_Initialize_CacheError(t *testing.T) {
	t.Parallel()
	cache := &queryServiceCacheMock{healthStatus: "healthy", initErr: errors.New("cache init failed")}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	err := svc.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error when cache init fails")
	}
}

func TestQueryService_Start_CacheError(t *testing.T) {
	t.Parallel()
	cache := &queryServiceCacheMock{healthStatus: "healthy", startErr: errors.New("cache start failed")}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	ctx := context.Background()
	_ = svc.Initialize(ctx)
	err := svc.Start(ctx)
	if err == nil {
		t.Fatal("expected error when cache start fails")
	}
}

func TestQueryService_Stop_CacheError(t *testing.T) {
	t.Parallel()
	cache := &queryServiceCacheMock{healthStatus: "healthy", stopErr: errors.New("cache stop failed")}
	svc := newQueryServiceForTest(t, nil, nil, cache)
	initQueryService(t, svc)
	err := svc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected error when cache stop fails")
	}
}

func TestQueryService_Query_BothBackendsFail(t *testing.T) {
	t.Parallel()
	errFailed := errors.New("query failed")
	mongo := &configurableMockAdapter{
		healthy: true,
		queryFn: func(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
			return nil, errFailed
		},
	}
	postgres := &configurableMockAdapter{
		healthy: true,
		queryFn: func(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
			return nil, errFailed
		},
	}
	svc := newQueryServiceForTest(t, mongo, postgres, nil)
	initQueryService(t, svc)
	_, err := svc.Query(context.Background(), &QueryRequest{})
	if err == nil {
		t.Fatal("expected error when both backends fail")
	}
}

func TestQueryService_QueryByHash_BothBackendsFail(t *testing.T) {
	t.Parallel()
	errFailed := errors.New("hash query failed")
	mongo := &configurableMockAdapter{
		healthy: true,
		queryByHashFn: func(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
			return nil, errFailed
		},
	}
	postgres := &configurableMockAdapter{
		healthy: true,
		queryByHashFn: func(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
			return nil, errFailed
		},
	}
	svc := newQueryServiceForTest(t, mongo, postgres, nil)
	initQueryService(t, svc)
	_, err := svc.QueryByHash(context.Background(), "hash123")
	if err == nil {
		t.Fatal("expected error when both backends fail for hash query")
	}
}

func TestQueryService_RuntimeSummary_NilCache(t *testing.T) {
	t.Parallel()
	svc := &DefaultQueryService{}
	rs := svc.RuntimeSummary(context.Background())
	if rs.CachePosture != "cache-unavailable" {
		t.Errorf("expected cache-unavailable, got %s", rs.CachePosture)
	}
}

func TestQueryService_ClassifyCachePosture_NilHealth(t *testing.T) {
	t.Parallel()
	nilCache := &cacheWithNilHealth{}
	svc := newQueryServiceForTest(t, nil, nil, nil)
	svc.cacheService = nilCache
	got := svc.classifyCachePosture(context.Background())
	if got != "cache-unobserved" {
		t.Errorf("expected cache-unobserved, got %s", got)
	}
}

type cacheWithNilHealth struct{}

func (c *cacheWithNilHealth) Initialize(ctx context.Context) error         { return nil }
func (c *cacheWithNilHealth) Start(ctx context.Context) error              { return nil }
func (c *cacheWithNilHealth) Stop(ctx context.Context) error               { return nil }
func (c *cacheWithNilHealth) Get(ctx context.Context, key string) ([]core.BlockchainEvent, error) {
	return nil, nil
}
func (c *cacheWithNilHealth) GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error) {
	return nil, nil
}
func (c *cacheWithNilHealth) Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl time.Duration) error {
	return nil
}
func (c *cacheWithNilHealth) SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl time.Duration) error {
	return nil
}
func (c *cacheWithNilHealth) SetQueryResult(ctx context.Context, key string, events []core.BlockchainEvent, total int64, ttl time.Duration) error {
	return nil
}
func (c *cacheWithNilHealth) GetQueryResult(ctx context.Context, key string) ([]core.BlockchainEvent, int64, error) {
	return nil, 0, nil
}
func (c *cacheWithNilHealth) Delete(ctx context.Context, key string) error { return nil }
func (c *cacheWithNilHealth) Health(ctx context.Context) *core.HealthStatus {
	return nil
}
