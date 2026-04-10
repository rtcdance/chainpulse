package query

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// DefaultCacheService provides cache operations with Redis backend
type DefaultCacheService struct {
	mu               sync.RWMutex
	cache            map[string]cacheEntry
	logger           core.Logger
	metricsCollector core.MetricsCollector
	initialized      bool
	running          bool
	done             chan struct{}
}

// cacheEntry represents a cached value with expiration
type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// NewCacheService creates a new cache service
func NewCacheService(
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) CacheService {
	return &DefaultCacheService{
		cache:            make(map[string]cacheEntry),
		logger:           logger,
		metricsCollector: metricsCollector,
		done:             make(chan struct{}),
	}
}

// Initialize initializes the cache service
func (cs *DefaultCacheService) Initialize(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.initialized {
		return fmt.Errorf("cache service already initialized")
	}

	cs.initialized = true
	cs.logger.Info("Cache service initialized", map[string]interface{}{
		"component": "cache-service",
	})

	return nil
}

// Start starts the cache service
func (cs *DefaultCacheService) Start(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.initialized {
		return fmt.Errorf("cache service not initialized")
	}

	if cs.running {
		return fmt.Errorf("cache service already running")
	}

	cs.running = true

	// Start cleanup goroutine
	go cs.cleanupExpiredEntries()

	cs.logger.Info("Cache service started", map[string]interface{}{
		"component": "cache-service",
	})

	return nil
}

// Stop stops the cache service
func (cs *DefaultCacheService) Stop(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.running {
		return fmt.Errorf("cache service not running")
	}

	cs.running = false

	// Signal the cleanup goroutine to stop
	select {
	case <-cs.done:
		// Already closed
	default:
		close(cs.done)
	}

	cs.logger.Info("Cache service stopped", map[string]interface{}{
		"component": "cache-service",
	})

	return nil
}

// Get retrieves a cached value
func (cs *DefaultCacheService) Get(ctx context.Context, key string) ([]core.BlockchainEvent, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if !cs.running {
		return nil, fmt.Errorf("cache service not running")
	}

	if key == "" {
		return nil, fmt.Errorf("cache key is required")
	}

	entry, exists := cs.cache[key]
	if !exists {
		cs.metricsCollector.RecordCounter("cache_miss", 1, map[string]string{})
		return nil, fmt.Errorf("cache miss")
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		cs.metricsCollector.RecordCounter("cache_expired", 1, map[string]string{})
		return nil, fmt.Errorf("cache entry expired")
	}

	// Unmarshal data
	var events []core.BlockchainEvent
	if err := json.Unmarshal(entry.data, &events); err != nil {
		cs.logger.Error("Failed to unmarshal cached data", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	cs.metricsCollector.RecordCounter("cache_hit", 1, map[string]string{})
	cs.logger.Debug("Cache hit", map[string]interface{}{
		"key":   key,
		"count": len(events),
	})

	return events, nil
}

// GetSingle retrieves a single cached value
func (cs *DefaultCacheService) GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if !cs.running {
		return nil, fmt.Errorf("cache service not running")
	}

	if key == "" {
		return nil, fmt.Errorf("cache key is required")
	}

	entry, exists := cs.cache[key]
	if !exists {
		cs.metricsCollector.RecordCounter("cache_miss", 1, map[string]string{})
		return nil, fmt.Errorf("cache miss")
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		cs.metricsCollector.RecordCounter("cache_expired", 1, map[string]string{})
		return nil, fmt.Errorf("cache entry expired")
	}

	// Unmarshal data
	var event core.BlockchainEvent
	if err := json.Unmarshal(entry.data, &event); err != nil {
		cs.logger.Error("Failed to unmarshal cached data", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	cs.metricsCollector.RecordCounter("cache_hit", 1, map[string]string{})
	cs.logger.Debug("Cache hit (single)", map[string]interface{}{
		"key": key,
	})

	return &event, nil
}

// Set sets a cached value
func (cs *DefaultCacheService) Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl interface{}) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.running {
		return fmt.Errorf("cache service not running")
	}

	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	if value == nil {
		return fmt.Errorf("cache value is required")
	}

	// Parse TTL
	duration := 1 * time.Hour // default
	if ttlDuration, ok := ttl.(time.Duration); ok {
		duration = ttlDuration
	}

	// Marshal data
	data, err := json.Marshal(value)
	if err != nil {
		cs.logger.Error("Failed to marshal cache data", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	// Store in cache
	cs.cache[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(duration),
	}

	cs.metricsCollector.RecordCounter("cache_set", 1, map[string]string{})
	cs.logger.Debug("Cache set", map[string]interface{}{
		"key":    key,
		"count":  len(value),
		"ttl_ms": duration.Milliseconds(),
	})

	return nil
}

// SetSingle sets a single cached value
func (cs *DefaultCacheService) SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl interface{}) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.running {
		return fmt.Errorf("cache service not running")
	}

	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	if value == nil {
		return fmt.Errorf("cache value is required")
	}

	// Parse TTL
	duration := 1 * time.Hour // default
	if ttlDuration, ok := ttl.(time.Duration); ok {
		duration = ttlDuration
	}

	// Marshal data
	data, err := json.Marshal(value)
	if err != nil {
		cs.logger.Error("Failed to marshal cache data", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	// Store in cache
	cs.cache[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(duration),
	}

	cs.metricsCollector.RecordCounter("cache_set", 1, map[string]string{})
	cs.logger.Debug("Cache set (single)", map[string]interface{}{
		"key":    key,
		"ttl_ms": duration.Milliseconds(),
	})

	return nil
}

// Delete deletes a cached value
func (cs *DefaultCacheService) Delete(ctx context.Context, key string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.running {
		return fmt.Errorf("cache service not running")
	}

	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	delete(cs.cache, key)

	cs.metricsCollector.RecordCounter("cache_delete", 1, map[string]string{})
	cs.logger.Debug("Cache deleted", map[string]interface{}{
		"key": key,
	})

	return nil
}

// Health returns the health status
func (cs *DefaultCacheService) Health(ctx context.Context) *core.HealthStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if !cs.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Cache service not initialized",
		}
	}

	if !cs.running {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Cache service not running",
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "Cache service healthy",
	}
}

// cleanupExpiredEntries periodically removes expired cache entries
func (cs *DefaultCacheService) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cs.done:
			// Stop signal received
			return
		case <-ticker.C:
			cs.mu.Lock()
			now := time.Now()
			count := 0
			for key, entry := range cs.cache {
				if now.After(entry.expiresAt) {
					delete(cs.cache, key)
					count++
				}
			}
			cs.mu.Unlock()

			if count > 0 {
				cs.logger.Debug("Cache cleanup", map[string]interface{}{
					"expired_entries": count,
				})
			}
		}
	}
}
