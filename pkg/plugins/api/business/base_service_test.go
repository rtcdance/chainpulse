package business

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockEntity implements Entity interface for testing
type MockEntity struct {
	ID   string
	Name string
	Data string
}

func (m *MockEntity) GetID() string {
	return m.ID
}

// MockBackend implements ServiceBackend interface for testing
type MockBackend struct {
	entities map[string]Entity
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		entities: make(map[string]Entity),
	}
}

func (m *MockBackend) Create(ctx context.Context, entity Entity) (Entity, error) {
	m.entities[entity.GetID()] = entity
	return entity, nil
}

func (m *MockBackend) Read(ctx context.Context, id string) (Entity, error) {
	entity, ok := m.entities[id]
	if !ok {
		return nil, fmt.Errorf("entity not found")
	}
	return entity, nil
}

func (m *MockBackend) Update(ctx context.Context, entity Entity) (Entity, error) {
	if _, ok := m.entities[entity.GetID()]; !ok {
		return nil, fmt.Errorf("entity not found")
	}
	m.entities[entity.GetID()] = entity
	return entity, nil
}

func (m *MockBackend) Delete(ctx context.Context, id string) error {
	if _, ok := m.entities[id]; !ok {
		return fmt.Errorf("entity not found")
	}
	delete(m.entities, id)
	return nil
}

func (m *MockBackend) List(ctx context.Context, limit, offset int) ([]Entity, error) {
	var result []Entity
	count := 0
	for _, entity := range m.entities {
		if count >= offset && count < offset+limit {
			result = append(result, entity)
		}
		count++
	}
	return result, nil
}

func (m *MockBackend) Query(ctx context.Context, filter map[string]any, limit, offset int) ([]Entity, error) {
	var result []Entity
	count := 0
	for _, entity := range m.entities {
		if count >= offset && count < offset+limit {
			result = append(result, entity)
		}
		count++
	}
	return result, nil
}

// MockCache implements ServiceCache interface for testing
type MockCache struct {
	data map[string]any
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]any),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) (any, error) {
	value, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return value, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func TestAbstractServiceCreate(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test", Data: "test data"}
	created, err := service.Create(context.Background(), entity)
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if created.GetID() != "1" {
		t.Errorf("Expected ID 1, got %s", created.GetID())
	}
}

func TestAbstractServiceRead(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test", Data: "test data"}
	if _, err := backend.Create(context.Background(), entity); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	read, err := service.Read(context.Background(), "1")
	if err != nil {
		t.Fatalf("Failed to read entity: %v", err)
	}

	if read.GetID() != "1" {
		t.Errorf("Expected ID 1, got %s", read.GetID())
	}
}

func TestAbstractServiceReadCaching(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test", Data: "test data"}
	if _, err := backend.Create(context.Background(), entity); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// First read - should hit backend
	read1, err := service.Read(context.Background(), "1")
	if err != nil {
		t.Fatalf("Failed to read entity: %v", err)
	}

	// Second read - should hit cache
	read2, err := service.Read(context.Background(), "1")
	if err != nil {
		t.Fatalf("Failed to read entity from cache: %v", err)
	}

	if read1.GetID() != read2.GetID() {
		t.Errorf("Cache read returned different entity")
	}

	metrics := service.GetMetrics()
	cacheHits := metrics["cache_hits"].(int64)
	if cacheHits < 1 {
		t.Errorf("Expected at least 1 cache hit, got %d", cacheHits)
	}
}

func TestAbstractServiceUpdate(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test", Data: "test data"}
	if _, err := backend.Create(context.Background(), entity); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	updated := &MockEntity{ID: "1", Name: "Updated", Data: "updated data"}
	result, err := service.Update(context.Background(), updated)
	if err != nil {
		t.Fatalf("Failed to update entity: %v", err)
	}

	if result.(*MockEntity).Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", result.(*MockEntity).Name)
	}
}

func TestAbstractServiceDelete(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test", Data: "test data"}
	if _, err := backend.Create(context.Background(), entity); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	err := service.Delete(context.Background(), "1")
	if err != nil {
		t.Fatalf("Failed to delete entity: %v", err)
	}

	_, err = backend.Read(context.Background(), "1")
	if err == nil {
		t.Error("Expected error when reading deleted entity")
	}
}

func TestAbstractServiceList(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	// Create multiple entities
	for i := 1; i <= 5; i++ {
		entity := &MockEntity{ID: fmt.Sprintf("%d", i), Name: fmt.Sprintf("Test%d", i)}
		if _, err := backend.Create(context.Background(), entity); err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}
	}

	entities, err := service.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("Failed to list entities: %v", err)
	}

	if len(entities) != 5 {
		t.Errorf("Expected 5 entities, got %d", len(entities))
	}
}

func TestAbstractServiceListPagination(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	// Create multiple entities
	for i := 1; i <= 10; i++ {
		entity := &MockEntity{ID: fmt.Sprintf("%d", i), Name: fmt.Sprintf("Test%d", i)}
		if _, err := backend.Create(context.Background(), entity); err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}
	}

	// Test limit validation
	entities, err := service.List(context.Background(), 200, 0)
	if err != nil {
		t.Fatalf("Failed to list entities: %v", err)
	}

	// Should be capped at 100
	if len(entities) > 100 {
		t.Errorf("Expected max 100 entities, got %d", len(entities))
	}

	// Test offset
	entities, err = service.List(context.Background(), 5, 5)
	if err != nil {
		t.Fatalf("Failed to list entities with offset: %v", err)
	}

	if len(entities) > 5 {
		t.Errorf("Expected max 5 entities, got %d", len(entities))
	}
}

func TestAbstractServiceQuery(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	// Create multiple entities
	for i := 1; i <= 5; i++ {
		entity := &MockEntity{ID: fmt.Sprintf("%d", i), Name: fmt.Sprintf("Test%d", i)}
		if _, err := backend.Create(context.Background(), entity); err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}
	}

	filter := map[string]any{"name": "Test"}
	entities, err := service.Query(context.Background(), filter, 10, 0)
	if err != nil {
		t.Fatalf("Failed to query entities: %v", err)
	}

	if len(entities) != 5 {
		t.Errorf("Expected 5 entities, got %d", len(entities))
	}
}

func TestAbstractServiceMetrics(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	entity := &MockEntity{ID: "1", Name: "Test"}
	if _, err := service.Create(context.Background(), entity); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}
	if _, err := service.Read(context.Background(), "1"); err != nil {
		t.Fatalf("Failed to read entity: %v", err)
	}
	if _, err := service.Update(context.Background(), entity); err != nil {
		t.Fatalf("Failed to update entity: %v", err)
	}

	metrics := service.GetMetrics()

	totalOps := metrics["total_operations"].(int64)
	if totalOps < 3 {
		t.Errorf("Expected at least 3 operations, got %d", totalOps)
	}

	serviceName := metrics["service_name"].(string)
	if serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", serviceName)
	}
}

func TestAbstractServiceSetCacheTTL(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	newTTL := 10 * time.Minute
	service.SetCacheTTL(newTTL)

	if service.cacheTTL != newTTL {
		t.Errorf("Expected TTL %v, got %v", newTTL, service.cacheTTL)
	}
}

func TestAbstractServiceWithoutCache(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	service := NewAbstractService("test-service", backend, nil)

	entity := &MockEntity{ID: "1", Name: "Test"}
	if _, err := backend.Create(context.Background(), entity); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	read, err := service.Read(context.Background(), "1")
	if err != nil {
		t.Fatalf("Failed to read entity without cache: %v", err)
	}

	if read.GetID() != "1" {
		t.Errorf("Expected ID 1, got %s", read.GetID())
	}

	metrics := service.GetMetrics()
	cacheHits := metrics["cache_hits"].(int64)
	if cacheHits != 0 {
		t.Errorf("Expected 0 cache hits without cache, got %d", cacheHits)
	}
}

func TestAbstractServiceErrorHandling(t *testing.T) {
	t.Parallel()
	backend := NewMockBackend()
	cache := NewMockCache()
	service := NewAbstractService("test-service", backend, cache)

	// Try to read non-existent entity
	_, err := service.Read(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error when reading non-existent entity")
	}

	metrics := service.GetMetrics()
	errors := metrics["errors"].(int64)
	if errors < 1 {
		t.Errorf("Expected at least 1 error, got %d", errors)
	}
}

func TestValidateLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"zero_returns_default", 0, 20},
		{"negative_returns_default", -1, 20},
		{"normal_value", 50, 50},
		{"max_value", 100, 100},
		{"exceeds_max", 101, 100},
		{"large_exceeds_max", 9999, 100},
		{"positive_one", 1, 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &AbstractService{}
			got := s.validateLimit(tt.limit)
			if got != tt.want {
				t.Errorf("validateLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}

func TestValidateOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		offset int
		want   int
	}{
		{"zero", 0, 0},
		{"positive", 10, 10},
		{"negative", -1, 0},
		{"large_negative", -100, 0},
		{"large_positive", 10000, 10000},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &AbstractService{}
			got := s.validateOffset(tt.offset)
			if got != tt.want {
				t.Errorf("validateOffset(%d) = %d, want %d", tt.offset, got, tt.want)
			}
		})
	}
}

func TestGetCacheKey(t *testing.T) {
	t.Parallel()

	s := &AbstractService{name: "test-service"}
	key := s.getCacheKey("read", "entity-123")
	if key != "test-service:read:entity-123" {
		t.Errorf("getCacheKey = %q, want %q", key, "test-service:read:entity-123")
	}

	s2 := &AbstractService{name: "another"}
	key2 := s2.getCacheKey("list", "")
	if key2 != "another:list:" {
		t.Errorf("getCacheKey = %q, want %q", key2, "another:list:")
	}
}
