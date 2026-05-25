package business

import (
	"context"
	"fmt"
	"testing"
)

type mockServiceHandler struct {
	name    string
	metrics map[string]any
}

func (m *mockServiceHandler) Handle(ctx context.Context, req any) (any, error) {
	return nil, nil
}

func (m *mockServiceHandler) GetMetrics() map[string]any {
	return m.metrics
}

func TestNewServiceFactory(t *testing.T) {
	f := NewServiceFactory()
	if f == nil {
		t.Fatal("NewServiceFactory returned nil")
	}
	if f.services == nil {
		t.Error("services map should not be nil")
	}
}

func TestServiceFactory_Register(t *testing.T) {
	f := NewServiceFactory()

	handler := &mockServiceHandler{name: "test"}
	err := f.Register("test-svc", handler)
	if err != nil {
		t.Errorf("Register error: %v", err)
	}

	err = f.Register("test-svc", handler)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestServiceFactory_Get(t *testing.T) {
	f := NewServiceFactory()

	handler := &mockServiceHandler{name: "svc1"}
	_ = f.Register("svc1", handler)

	h, err := f.Get("svc1")
	if err != nil {
		t.Errorf("Get error: %v", err)
	}
	if h != handler {
		t.Error("returned wrong handler")
	}

	_, err = f.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown service")
	}
}

func TestServiceFactory_List(t *testing.T) {
	f := NewServiceFactory()
	_ = f.Register("svc1", &mockServiceHandler{})
	_ = f.Register("svc2", &mockServiceHandler{})

	names := f.List()
	if len(names) != 2 {
		t.Errorf("List len = %d", len(names))
	}
}

func TestServiceFactory_GetMetrics(t *testing.T) {
	f := NewServiceFactory()
	_ = f.Register("svc1", &mockServiceHandler{
		metrics: map[string]any{"calls": 5},
	})

	metrics := f.GetMetrics()
	if metrics["svc1"] == nil {
		t.Error("expected metrics for svc1")
	}

	m := metrics["svc1"].(map[string]any)
	if m["calls"] != 5 {
		t.Errorf("calls = %v", m["calls"])
	}
}

func TestGenericServiceHandler_GetCacheKey(t *testing.T) {
	h := NewGenericServiceHandler("test-svc", nil, nil)
	key := h.getCacheKey("list", "offset=10")
	if key != "test-svc:list:offset=10" {
		t.Errorf("getCacheKey = %q", key)
	}
}

func TestGenericServiceHandler_Handle_UnsupportedType(t *testing.T) {
	h := NewGenericServiceHandler("test-svc", nil, nil)
	_, err := h.Handle(context.Background(), "unsupported")
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestGenericServiceHandler_Handle_Create(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	req := &CreateRequest{Entity: entity}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle create error: %v", err)
	}
	created, ok := result.(*MockEntity)
	if !ok {
		t.Fatalf("unexpected type: %T", result)
	}
	if created.Name != "Test" {
		t.Errorf("Name = %s", created.Name)
	}
}

func TestGenericServiceHandler_Handle_Read(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	req := &ReadRequest{ID: "1"}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle read error: %v", err)
	}
	read, ok := result.(*MockEntity)
	if !ok {
		t.Fatalf("unexpected type: %T", result)
	}
	if read.Name != "Test" {
		t.Errorf("Name = %s", read.Name)
	}
}

func TestGenericServiceHandler_Handle_Update(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	updated := &MockEntity{ID: "1", Name: "Updated"}
	req := &UpdateRequest{Entity: updated}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle update error: %v", err)
	}
	u, ok := result.(*MockEntity)
	if !ok {
		t.Fatalf("unexpected type: %T", result)
	}
	if u.Name != "Updated" {
		t.Errorf("Name = %s", u.Name)
	}
}

func TestGenericServiceHandler_Handle_Delete(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	req := &DeleteRequest{ID: "1"}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle delete error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result from delete")
	}
}

func TestGenericServiceHandler_Handle_List(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	for i := 1; i <= 3; i++ {
		entity := &MockEntity{ID: fmt.Sprintf("%d", i), Name: fmt.Sprintf("Test%d", i)}
		_, _ = backend.Create(context.Background(), entity)
	}

	req := &ListRequest{Limit: 10, Offset: 0}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle list error: %v", err)
	}
	entities, ok := result.([]Entity)
	if !ok {
		t.Fatalf("unexpected type: %T", result)
	}
	if len(entities) != 3 {
		t.Errorf("len = %d", len(entities))
	}
}

func TestGenericServiceHandler_Handle_Query(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	req := &QueryRequest{Filter: map[string]any{"name": "Test"}, Limit: 10, Offset: 0}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle query error: %v", err)
	}
	entities, ok := result.([]Entity)
	if !ok {
		t.Fatalf("unexpected type: %T", result)
	}
	if len(entities) != 1 {
		t.Errorf("len = %d", len(entities))
	}
}

func TestGenericServiceHandler_HandleRead_CacheHit(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	cacheKey := h.getCacheKey("read", "1")
	entity := &MockEntity{ID: "1", Name: "cached"}
	_ = cache.Set(context.Background(), cacheKey, entity, 0)

	req := &ReadRequest{ID: "1"}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle read cache error: %v", err)
	}
	e := result.(*MockEntity)
	if e.Name != "cached" {
		t.Errorf("Name = %s", e.Name)
	}
}

func TestGenericServiceHandler_HandleRead_CacheMiss(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	req := &ReadRequest{ID: "1"}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle read error: %v", err)
	}
	e := result.(*MockEntity)
	if e.Name != "Test" {
		t.Errorf("Name = %s", e.Name)
	}
}

func TestGenericServiceHandler_HandleRead_NoCache(t *testing.T) {
	backend := NewMockBackend()
	h := NewGenericServiceHandler("test-svc", backend, nil)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	req := &ReadRequest{ID: "1"}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle read error: %v", err)
	}
	_ = result.(*MockEntity)
}

func TestGenericServiceHandler_HandleUpdate_CacheInvalidation(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)

	updated := &MockEntity{ID: "1", Name: "Updated"}
	req := &UpdateRequest{Entity: updated}
	_, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle update error: %v", err)
	}

	cacheKey := h.getCacheKey("read", "1")
	_, cacheErr := cache.Get(context.Background(), cacheKey)
	if cacheErr == nil {
		t.Error("expected cache miss after update")
	}
}

func TestGenericServiceHandler_HandleList_CacheHit(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	cacheKey := h.getCacheKey("list", "limit:20:offset:0")
	entities := []Entity{&MockEntity{ID: "1", Name: "cached"}}
	_ = cache.Set(context.Background(), cacheKey, entities, 0)

	req := &ListRequest{Limit: 20, Offset: 0}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle list cache error: %v", err)
	}
	ents := result.([]Entity)
	if len(ents) != 1 {
		t.Errorf("len = %d", len(ents))
	}
}

func TestGenericServiceHandler_HandleQuery_CacheHit(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	filter := map[string]any{"status": "active"}
	cacheKey := h.getCacheKey("query", fmt.Sprintf("filter:%v:limit:20:offset:0", filter))
	entities := []Entity{&MockEntity{ID: "1", Name: "cached"}}
	_ = cache.Set(context.Background(), cacheKey, entities, 0)

	req := &QueryRequest{Filter: filter, Limit: 20, Offset: 0}
	result, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle query cache error: %v", err)
	}
	ents := result.([]Entity)
	if len(ents) != 1 {
		t.Errorf("len = %d", len(ents))
	}
}

func TestGenericServiceHandler_GetMetrics_Empty(t *testing.T) {
	h := NewGenericServiceHandler("test-svc", nil, nil)
	metrics := h.GetMetrics()
	if metrics["service_name"] != "test-svc" {
		t.Errorf("service_name = %v", metrics["service_name"])
	}
	if metrics["total_operations"] != int64(0) {
		t.Errorf("total_operations = %v", metrics["total_operations"])
	}
}

func TestGenericServiceHandler_GetMetrics_WithOps(t *testing.T) {
	backend := NewMockBackend()
	cache := NewMockCache()
	h := NewGenericServiceHandler("test-svc", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	_, _ = backend.Create(context.Background(), entity)
	_, _ = h.Handle(context.Background(), &ReadRequest{ID: "1"})
	_, _ = h.Handle(context.Background(), &ReadRequest{ID: "1"})

	metrics := h.GetMetrics()
	if metrics["total_operations"].(int64) < 2 {
		t.Errorf("total_operations = %v", metrics["total_operations"])
	}
}

func TestGenericServiceHandler_ValidateLimit(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 20},
		{-5, 20},
		{10, 10},
		{100, 100},
		{200, 100},
	}
	for _, tt := range tests {
		h := NewGenericServiceHandler("test", nil, nil)
		got := h.validateLimit(tt.input)
		if got != tt.want {
			t.Errorf("validateLimit(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestGenericServiceHandler_ValidateOffset(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{-1, 0},
		{10, 10},
		{100, 100},
	}
	for _, tt := range tests {
		h := NewGenericServiceHandler("test", nil, nil)
		got := h.validateOffset(tt.input)
		if got != tt.want {
			t.Errorf("validateOffset(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestGenericServiceHandler_RecordMethods(t *testing.T) {
	h := NewGenericServiceHandler("test-svc", nil, nil)

	h.recordMetric(100)
	h.recordMetric(200)
	h.recordCacheHit()
	h.recordCacheHit()
	h.recordCacheHit()
	h.recordCacheMiss()
	h.recordError()

	metrics := h.GetMetrics()
	if metrics["total_operations"] != int64(2) {
		t.Errorf("total_operations = %v, want 2", metrics["total_operations"])
	}
	if metrics["cache_hits"] != int64(3) {
		t.Errorf("cache_hits = %v, want 3", metrics["cache_hits"])
	}
	if metrics["cache_misses"] != int64(1) {
		t.Errorf("cache_misses = %v, want 1", metrics["cache_misses"])
	}
	if metrics["cache_hit_rate"] != 75.0 {
		t.Errorf("cache_hit_rate = %v, want 75.0", metrics["cache_hit_rate"])
	}
}

func TestGenericServiceHandler_SetCacheTTL(t *testing.T) {
	h := NewGenericServiceHandler("test", nil, nil)
	h.SetCacheTTL(10)
	// SetCacheTTL is tested by verifying it doesn't panic
}

func TestGenericServiceHandler_GetMetrics_NoOps(t *testing.T) {
	h := NewGenericServiceHandler("test-svc", nil, nil)
	metrics := h.GetMetrics()
	if metrics["total_operations"] != int64(0) {
		t.Errorf("total_operations = %v, want 0", metrics["total_operations"])
	}
	if metrics["cache_hit_rate"] != 0.0 {
		t.Errorf("cache_hit_rate = %v, want 0.0", metrics["cache_hit_rate"])
	}
}
