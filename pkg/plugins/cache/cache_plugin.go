package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// CachePlugin defines the interface for cache implementations
type CachePlugin interface {
	// Initialize initializes the cache plugin
	Initialize(config *core.Config) error

	// Start starts the cache plugin
	Start() error

	// Stop stops the cache plugin
	Stop() error

	// Health returns the health status of the plugin
	Health() *core.HealthStatus

	// Get retrieves a value from cache
	Get(key string) (*core.CacheEntry, error)

	// Set stores a value in cache with TTL
	Set(entry *core.CacheEntry) error

	// Delete removes a value from cache
	Delete(key string) error

	// Clear clears all cache entries
	Clear() error

	// GetStats returns cache statistics
	GetStats() *CacheStats
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	HitCount      int64
	MissCount     int64
	EvictionCount int64
	TotalSize     int64
	EntryCount    int64
}

// BaseCachePlugin provides base implementation for cache plugins
type BaseCachePlugin struct {
	mu               sync.RWMutex
	initialized      bool
	running          bool
	config           *core.Config
	logger           core.Logger
	metricsCollector core.MetricsCollector
	lastHealthCheck  *core.HealthStatus
	hitCount         int64
	missCount        int64
	evictionCount    int64
}

// NewBaseCachePlugin creates a new base cache plugin
func NewBaseCachePlugin(logger core.Logger, metricsCollector core.MetricsCollector) *BaseCachePlugin {
	return &BaseCachePlugin{
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the cache plugin
func (p *BaseCachePlugin) Initialize(config *core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("cache plugin already initialized")
	}

	if config == nil {
		return fmt.Errorf("config is required")
	}

	p.config = config
	p.initialized = true

	if p.logger != nil {
		p.logger.Info("Cache plugin initialized", "component", "cache")
	}

	return nil
}

// Start starts the cache plugin
func (p *BaseCachePlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("cache plugin not initialized")
	}

	if p.running {
		return fmt.Errorf("cache plugin already running")
	}

	p.running = true
	p.lastHealthCheck = &core.HealthStatus{
		Status:    "healthy",
		Message:   "Cache plugin started",
		Details:   make(map[string]any),
		Timestamp: time.Now(),
	}

	if p.logger != nil {
		p.logger.Info("Cache plugin started", "component", "cache")
	}

	return nil
}

// Stop stops the cache plugin
func (p *BaseCachePlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	p.running = false
	p.lastHealthCheck = &core.HealthStatus{
		Status:    "healthy",
		Message:   "Cache plugin stopped",
		Details:   make(map[string]any),
		Timestamp: time.Now(),
	}

	if p.logger != nil {
		p.logger.Info("Cache plugin stopped", "component", "cache")
	}

	return nil
}

// Health returns the health status of the plugin
func (p *BaseCachePlugin) Health() *core.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return &core.HealthStatus{
			Status:    "unhealthy",
			Message:   "Cache plugin not initialized",
			Details:   make(map[string]any),
			Timestamp: time.Now(),
		}
	}

	if !p.running {
		return &core.HealthStatus{
			Status:    "unhealthy",
			Message:   "Cache plugin not running",
			Details:   make(map[string]any),
			Timestamp: time.Now(),
		}
	}

	return &core.HealthStatus{
		Status:    "healthy",
		Message:   "Cache plugin healthy",
		Details:   make(map[string]any),
		Timestamp: time.Now(),
	}
}

// RecordHit records a cache hit
func (p *BaseCachePlugin) RecordHit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.hitCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_hit", 1, map[string]string{})
	}
}

// RecordMiss records a cache miss
func (p *BaseCachePlugin) RecordMiss() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.missCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_miss", 1, map[string]string{})
	}
}

// RecordEviction records a cache eviction
func (p *BaseCachePlugin) RecordEviction() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.evictionCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_eviction", 1, map[string]string{})
	}
}

// GetHitCount returns the hit count
func (p *BaseCachePlugin) GetHitCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.hitCount
}

// GetMissCount returns the miss count
func (p *BaseCachePlugin) GetMissCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.missCount
}

// GetEvictionCount returns the eviction count
func (p *BaseCachePlugin) GetEvictionCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.evictionCount
}

// DefaultInMemoryCachePlugin provides in-memory cache implementation
type DefaultInMemoryCachePlugin struct {
	*BaseCachePlugin
	data map[string]*core.CacheEntry
}

// NewDefaultInMemoryCachePlugin creates a new in-memory cache plugin
func NewDefaultInMemoryCachePlugin(logger core.Logger, metricsCollector core.MetricsCollector) *DefaultInMemoryCachePlugin {
	return &DefaultInMemoryCachePlugin{
		BaseCachePlugin: NewBaseCachePlugin(logger, metricsCollector),
		data:            make(map[string]*core.CacheEntry),
	}
}

// Get retrieves a value from cache
func (p *DefaultInMemoryCachePlugin) Get(key string) (*core.CacheEntry, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		p.mu.RUnlock()
		p.RecordMiss()
		return nil, nil
	}

	// Check if entry has expired
	if entry.ExpiresAt.Before(time.Now()) {
		p.mu.RUnlock()
		p.RecordEviction()
		p.mu.Lock()
		delete(p.data, key)
		p.mu.Unlock()
		return nil, nil
	}

	p.mu.RUnlock()
	p.RecordHit()
	return entry, nil
}

// Set stores a value in cache with TTL
func (p *DefaultInMemoryCachePlugin) Set(entry *core.CacheEntry) error {
	if entry == nil {
		return fmt.Errorf("entry is required")
	}

	if entry.Key == "" {
		return fmt.Errorf("entry key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	// Set expiration time if TTL is specified
	if entry.TTL > 0 {
		entry.ExpiresAt = time.Now().Add(time.Duration(entry.TTL) * time.Second)
	} else {
		entry.ExpiresAt = time.Now().Add(24 * time.Hour) // default 24 hours
	}

	p.data[entry.Key] = entry

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_set", 1, map[string]string{})
	}

	return nil
}

// Delete removes a value from cache
func (p *DefaultInMemoryCachePlugin) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	delete(p.data, key)

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_delete", 1, map[string]string{})
	}

	return nil
}

// Clear clears all cache entries
func (p *DefaultInMemoryCachePlugin) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	p.data = make(map[string]*core.CacheEntry)

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_clear", 1, map[string]string{})
	}

	return nil
}

// GetStats returns cache statistics
func (p *DefaultInMemoryCachePlugin) GetStats() *CacheStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalSize := int64(0)
	for _, entry := range p.data {
		if entry.Value != nil {
			// Rough estimate of size
			totalSize += 100
		}
	}

	return &CacheStats{
		HitCount:      p.hitCount,
		MissCount:     p.missCount,
		EvictionCount: p.evictionCount,
		TotalSize:     totalSize,
		EntryCount:    int64(len(p.data)),
	}
}
