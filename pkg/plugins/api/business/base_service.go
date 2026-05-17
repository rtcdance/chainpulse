// Package business provides business logic services for the API layer.
package business

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Entity represents a generic entity in the system
type Entity interface {
	GetID() string
}

// logCacheErr logs a cache operation error. Uses the standard logger since
// AbstractService does not carry a structured logger.
func logCacheErr(op, key string, err error) {
	if err != nil {
		log.Printf("%s: cache %s error for key %s: %v", "AbstractService", op, key, err)
	}
}

// BaseService defines a generic service interface for CRUD operations
type BaseService interface {
	// Create creates a new entity
	Create(ctx context.Context, entity Entity) (Entity, error)

	// Read retrieves an entity by ID
	Read(ctx context.Context, id string) (Entity, error)

	// Update updates an existing entity
	Update(ctx context.Context, entity Entity) (Entity, error)

	// Delete deletes an entity by ID
	Delete(ctx context.Context, id string) error

	// List retrieves entities with pagination
	List(ctx context.Context, limit, offset int) ([]Entity, error)

	// Query retrieves entities matching a filter
	Query(ctx context.Context, filter map[string]any, limit, offset int) ([]Entity, error)

	// GetMetrics returns service metrics
	GetMetrics() map[string]any
}

// ServiceMetrics tracks service operation metrics
type ServiceMetrics struct {
	totalOperations int64
	cacheHits       int64
	cacheMisses     int64
	errors          int64
	totalDuration   time.Duration
	mu              sync.RWMutex
}

// AbstractService provides a base implementation of BaseService
type AbstractService struct {
	name     string
	backend  ServiceBackend
	cache    ServiceCache
	mu       sync.RWMutex
	metrics  *ServiceMetrics
	cacheTTL time.Duration
}

// ServiceBackend defines the backend data source interface
type ServiceBackend interface {
	Create(ctx context.Context, entity Entity) (Entity, error)
	Read(ctx context.Context, id string) (Entity, error)
	Update(ctx context.Context, entity Entity) (Entity, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Entity, error)
	Query(ctx context.Context, filter map[string]any, limit, offset int) ([]Entity, error)
}

// ServiceCache defines the caching interface
type ServiceCache interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewAbstractService creates a new abstract service
func NewAbstractService(name string, backend ServiceBackend, cache ServiceCache) *AbstractService {
	return &AbstractService{
		name:     name,
		backend:  backend,
		cache:    cache,
		metrics:  &ServiceMetrics{},
		cacheTTL: 5 * time.Minute,
	}
}

// Create creates a new entity
func (s *AbstractService) Create(ctx context.Context, entity Entity) (Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	defer func() {
		s.recordMetric(time.Since(start))
	}()

	// Create in backend
	created, err := s.backend.Create(ctx, entity)
	if err != nil {
		s.recordError()
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	// Invalidate list cache
	s.invalidateListCache()

	return created, nil
}

// Read retrieves an entity by ID
func (s *AbstractService) Read(ctx context.Context, id string) (Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	defer func() {
		s.recordMetric(time.Since(start))
	}()

	// Try cache first
	if s.cache != nil {
		cacheKey := s.getCacheKey("read", id)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			if entity, ok := cached.(Entity); ok {
				s.recordCacheHit()
				return entity, nil
			}
		}
		s.recordCacheMiss()
	}

	// Read from backend
	entity, err := s.backend.Read(ctx, id)
	if err != nil {
		s.recordError()
		return nil, fmt.Errorf("failed to read entity: %w", err)
	}

	// Cache result
	if s.cache != nil && entity != nil {
		cacheKey := s.getCacheKey("read", id)
		logCacheErr("set", cacheKey, s.cache.Set(ctx, cacheKey, entity, s.cacheTTL))
	}

	return entity, nil
}

// Update updates an existing entity
func (s *AbstractService) Update(ctx context.Context, entity Entity) (Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	defer func() {
		s.recordMetric(time.Since(start))
	}()

	// Update in backend
	updated, err := s.backend.Update(ctx, entity)
	if err != nil {
		s.recordError()
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}

	// Invalidate cache for this entity
	if s.cache != nil {
		cacheKey := s.getCacheKey("read", entity.GetID())
		logCacheErr("delete", cacheKey, s.cache.Delete(ctx, cacheKey))
	}

	// Invalidate list cache
	s.invalidateListCache()

	return updated, nil
}

// Delete deletes an entity by ID
func (s *AbstractService) Delete(ctx context.Context, id string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	defer func() {
		s.recordMetric(time.Since(start))
	}()

	// Delete from backend
	err := s.backend.Delete(ctx, id)
	if err != nil {
		s.recordError()
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	// Invalidate cache for this entity
	if s.cache != nil {
		cacheKey := s.getCacheKey("read", id)
		logCacheErr("delete", cacheKey, s.cache.Delete(ctx, cacheKey))
	}

	// Invalidate list cache
	s.invalidateListCache()

	return nil
}

// List retrieves entities with pagination
func (s *AbstractService) List(ctx context.Context, limit, offset int) ([]Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	defer func() {
		s.recordMetric(time.Since(start))
	}()

	// Validate pagination
	limit = s.validateLimit(limit)
	offset = s.validateOffset(offset)

	// Try cache first
	if s.cache != nil {
		cacheKey := s.getCacheKey("list", fmt.Sprintf("limit:%d:offset:%d", limit, offset))
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			if entities, ok := cached.([]Entity); ok {
				s.recordCacheHit()
				return entities, nil
			}
		}
		s.recordCacheMiss()
	}

	// List from backend
	entities, err := s.backend.List(ctx, limit, offset)
	if err != nil {
		s.recordError()
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	// Cache result
	if s.cache != nil && entities != nil {
		cacheKey := s.getCacheKey("list", fmt.Sprintf("limit:%d:offset:%d", limit, offset))
		logCacheErr("set", cacheKey, s.cache.Set(ctx, cacheKey, entities, s.cacheTTL))
	}

	return entities, nil
}

// Query retrieves entities matching a filter
func (s *AbstractService) Query(ctx context.Context, filter map[string]any, limit, offset int) ([]Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Now()
	defer func() {
		s.recordMetric(time.Since(start))
	}()

	// Validate pagination
	limit = s.validateLimit(limit)
	offset = s.validateOffset(offset)

	// Try cache first
	if s.cache != nil {
		cacheKey := s.getCacheKey("query", fmt.Sprintf("filter:%v:limit:%d:offset:%d", filter, limit, offset))
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			s.recordCacheHit()
			if typed, ok := cached.([]Entity); ok {
				return typed, nil
			}
		}
		s.recordCacheMiss()
	}

	// Query from backend
	entities, err := s.backend.Query(ctx, filter, limit, offset)
	if err != nil {
		s.recordError()
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}

	// Cache result
	if s.cache != nil && entities != nil {
		cacheKey := s.getCacheKey("query", fmt.Sprintf("filter:%v:limit:%d:offset:%d", filter, limit, offset))
		logCacheErr("set", cacheKey, s.cache.Set(ctx, cacheKey, entities, s.cacheTTL))
	}

	return entities, nil
}

// GetMetrics returns service metrics
func (s *AbstractService) GetMetrics() map[string]any {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	avgDuration := time.Duration(0)
	if s.metrics.totalOperations > 0 {
		avgDuration = s.metrics.totalDuration / time.Duration(s.metrics.totalOperations)
	}

	cacheHitRate := 0.0
	totalCacheOps := s.metrics.cacheHits + s.metrics.cacheMisses
	if totalCacheOps > 0 {
		cacheHitRate = float64(s.metrics.cacheHits) / float64(totalCacheOps) * 100.0
	}

	return map[string]any{
		"service_name":     s.name,
		"total_operations": s.metrics.totalOperations,
		"cache_hits":       s.metrics.cacheHits,
		"cache_misses":     s.metrics.cacheMisses,
		"cache_hit_rate":   cacheHitRate,
		"errors":           s.metrics.errors,
		"avg_duration_ms":  avgDuration.Milliseconds(),
		"total_duration":   s.metrics.totalDuration.String(),
	}
}

// SetCacheTTL sets the cache TTL
func (s *AbstractService) SetCacheTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
}

// Helper methods

func (s *AbstractService) getCacheKey(operation, params string) string {
	return fmt.Sprintf("%s:%s:%s", s.name, operation, params)
}

func (s *AbstractService) invalidateListCache() {
	// In a real implementation, this would invalidate all list cache entries
	// For now, we rely on TTL expiration
}

func (s *AbstractService) validateLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func (s *AbstractService) validateOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func (s *AbstractService) recordMetric(duration time.Duration) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.totalOperations++
	s.metrics.totalDuration += duration
}

func (s *AbstractService) recordCacheHit() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.cacheHits++
}

func (s *AbstractService) recordCacheMiss() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.cacheMisses++
}

func (s *AbstractService) recordError() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.errors++
}
