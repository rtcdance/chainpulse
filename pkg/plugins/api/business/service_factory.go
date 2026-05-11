package business

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ServiceHandler defines the interface for service handlers
type ServiceHandler interface {
	// Handle processes a request and returns a response
	Handle(ctx context.Context, req interface{}) (interface{}, error)

	// GetMetrics returns service metrics
	GetMetrics() map[string]interface{}
}

// ServiceFactory creates and manages service handlers
type ServiceFactory struct {
	services map[string]ServiceHandler
	mu       sync.RWMutex
}

// NewServiceFactory creates a new service factory
func NewServiceFactory() *ServiceFactory {
	return &ServiceFactory{
		services: make(map[string]ServiceHandler),
	}
}

// Register registers a service handler
func (f *ServiceFactory) Register(name string, handler ServiceHandler) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.services[name]; exists {
		return fmt.Errorf("service already registered: %s", name)
	}

	f.services[name] = handler
	return nil
}

// Get retrieves a service handler by name
func (f *ServiceFactory) Get(name string) (ServiceHandler, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	handler, exists := f.services[name]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	return handler, nil
}

// List returns all registered service names
func (f *ServiceFactory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.services))
	for name := range f.services {
		names = append(names, name)
	}

	return names
}

// GetMetrics returns metrics for all services
func (f *ServiceFactory) GetMetrics() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	metrics := make(map[string]interface{})
	for name, handler := range f.services {
		metrics[name] = handler.GetMetrics()
	}

	return metrics
}

// GenericServiceHandler provides a generic service handler implementation
type GenericServiceHandler struct {
	name     string
	backend  ServiceBackend
	cache    ServiceCache
	mu       sync.RWMutex
	metrics  *ServiceMetrics
	cacheTTL time.Duration
}

// NewGenericServiceHandler creates a new generic service handler
func NewGenericServiceHandler(name string, backend ServiceBackend, cache ServiceCache) *GenericServiceHandler {
	return &GenericServiceHandler{
		name:     name,
		backend:  backend,
		cache:    cache,
		metrics:  &ServiceMetrics{},
		cacheTTL: 5 * time.Minute,
	}
}

// Handle processes a request
func (h *GenericServiceHandler) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	start := time.Now()
	defer func() {
		h.recordMetric(time.Since(start))
	}()

	// Type assert request to determine operation
	switch r := req.(type) {
	case *CreateRequest:
		return h.handleCreate(ctx, r)
	case *ReadRequest:
		return h.handleRead(ctx, r)
	case *UpdateRequest:
		return h.handleUpdate(ctx, r)
	case *DeleteRequest:
		return h.handleDelete(ctx, r)
	case *ListRequest:
		return h.handleList(ctx, r)
	case *QueryRequest:
		return h.handleQuery(ctx, r)
	default:
		h.recordError()
		return nil, fmt.Errorf("unsupported request type: %T", req)
	}
}

// GetMetrics returns service metrics
func (h *GenericServiceHandler) GetMetrics() map[string]interface{} {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	avgDuration := time.Duration(0)
	if h.metrics.totalOperations > 0 {
		avgDuration = h.metrics.totalDuration / time.Duration(h.metrics.totalOperations)
	}

	cacheHitRate := 0.0
	totalCacheOps := h.metrics.cacheHits + h.metrics.cacheMisses
	if totalCacheOps > 0 {
		cacheHitRate = float64(h.metrics.cacheHits) / float64(totalCacheOps) * 100.0
	}

	return map[string]interface{}{
		"service_name":     h.name,
		"total_operations": h.metrics.totalOperations,
		"cache_hits":       h.metrics.cacheHits,
		"cache_misses":     h.metrics.cacheMisses,
		"cache_hit_rate":   cacheHitRate,
		"errors":           h.metrics.errors,
		"avg_duration_ms":  avgDuration.Milliseconds(),
		"total_duration":   h.metrics.totalDuration.String(),
	}
}

// SetCacheTTL sets the cache TTL
func (h *GenericServiceHandler) SetCacheTTL(ttl time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cacheTTL = ttl
}

// Request types

// CreateRequest represents a create operation
type CreateRequest struct {
	Entity Entity
}

// ReadRequest represents a read operation
type ReadRequest struct {
	ID string
}

// UpdateRequest represents an update operation
type UpdateRequest struct {
	Entity Entity
}

// DeleteRequest represents a delete operation
type DeleteRequest struct {
	ID string
}

// ListRequest represents a list operation
type ListRequest struct {
	Limit  int
	Offset int
}

// QueryRequest represents a query operation
type QueryRequest struct {
	Filter map[string]interface{}
	Limit  int
	Offset int
}

// Handler methods

func (h *GenericServiceHandler) handleCreate(ctx context.Context, req *CreateRequest) (interface{}, error) {
	created, err := h.backend.Create(ctx, req.Entity)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	// Invalidate list cache
	h.invalidateListCache()

	return created, nil
}

func (h *GenericServiceHandler) handleRead(ctx context.Context, req *ReadRequest) (interface{}, error) {
	// Try cache first
	if h.cache != nil {
		cacheKey := h.getCacheKey("read", req.ID)
		if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			h.recordCacheHit()
			return cached, nil
		}
		h.recordCacheMiss()
	}

	// Read from backend
	entity, err := h.backend.Read(ctx, req.ID)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to read entity: %w", err)
	}

	// Cache result
	if h.cache != nil && entity != nil {
		cacheKey := h.getCacheKey("read", req.ID)
		logCacheErr("set", cacheKey, h.cache.Set(ctx, cacheKey, entity, h.cacheTTL))
	}

	return entity, nil
}

func (h *GenericServiceHandler) handleUpdate(ctx context.Context, req *UpdateRequest) (interface{}, error) {
	updated, err := h.backend.Update(ctx, req.Entity)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}

	// Invalidate cache for this entity
	if h.cache != nil {
		cacheKey := h.getCacheKey("read", req.Entity.GetID())
		logCacheErr("delete", cacheKey, h.cache.Delete(ctx, cacheKey))
	}

	// Invalidate list cache
	h.invalidateListCache()

	return updated, nil
}

func (h *GenericServiceHandler) handleDelete(ctx context.Context, req *DeleteRequest) (interface{}, error) {
	err := h.backend.Delete(ctx, req.ID)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to delete entity: %w", err)
	}

	// Invalidate cache for this entity
	if h.cache != nil {
		cacheKey := h.getCacheKey("read", req.ID)
		logCacheErr("delete", cacheKey, h.cache.Delete(ctx, cacheKey))
	}

	// Invalidate list cache
	h.invalidateListCache()

	return nil, nil
}

func (h *GenericServiceHandler) handleList(ctx context.Context, req *ListRequest) (interface{}, error) {
	// Validate pagination
	limit := h.validateLimit(req.Limit)
	offset := h.validateOffset(req.Offset)

	// Try cache first
	if h.cache != nil {
		cacheKey := h.getCacheKey("list", fmt.Sprintf("limit:%d:offset:%d", limit, offset))
		if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			h.recordCacheHit()
			return cached, nil
		}
		h.recordCacheMiss()
	}

	// List from backend
	entities, err := h.backend.List(ctx, limit, offset)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Cache result
	if h.cache != nil && entities != nil {
		cacheKey := h.getCacheKey("list", fmt.Sprintf("limit:%d:offset:%d", limit, offset))
		logCacheErr("set", cacheKey, h.cache.Set(ctx, cacheKey, entities, h.cacheTTL))
	}

	return entities, nil
}

func (h *GenericServiceHandler) handleQuery(ctx context.Context, req *QueryRequest) (interface{}, error) {
	// Validate pagination
	limit := h.validateLimit(req.Limit)
	offset := h.validateOffset(req.Offset)

	// Try cache first
	if h.cache != nil {
		cacheKey := h.getCacheKey("query", fmt.Sprintf("filter:%v:limit:%d:offset:%d", req.Filter, limit, offset))
		if cached, err := h.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			h.recordCacheHit()
			return cached, nil
		}
		h.recordCacheMiss()
	}

	// Query from backend
	entities, err := h.backend.Query(ctx, req.Filter, limit, offset)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}

	// Cache result
	if h.cache != nil && entities != nil {
		cacheKey := h.getCacheKey("query", fmt.Sprintf("filter:%v:limit:%d:offset:%d", req.Filter, limit, offset))
		logCacheErr("set", cacheKey, h.cache.Set(ctx, cacheKey, entities, h.cacheTTL))
	}

	return entities, nil
}

// Helper methods

func (h *GenericServiceHandler) getCacheKey(operation, params string) string {
	return fmt.Sprintf("%s:%s:%s", h.name, operation, params)
}

func (h *GenericServiceHandler) invalidateListCache() {
	// In a real implementation, this would invalidate all list cache entries
	// For now, we rely on TTL expiration
}

func (h *GenericServiceHandler) validateLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (h *GenericServiceHandler) validateOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func (h *GenericServiceHandler) recordMetric(duration time.Duration) {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()
	h.metrics.totalOperations++
	h.metrics.totalDuration += duration
}

func (h *GenericServiceHandler) recordCacheHit() {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()
	h.metrics.cacheHits++
}

func (h *GenericServiceHandler) recordCacheMiss() {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()
	h.metrics.cacheMisses++
}

func (h *GenericServiceHandler) recordError() {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()
	h.metrics.errors++
}
