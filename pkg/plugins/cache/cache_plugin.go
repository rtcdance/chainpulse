package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

const defaultCacheTTL = 24 * time.Hour

type BaseCachePlugin struct {
	name             string
	version          string
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

func NewBaseCachePlugin(name, version string, logger core.Logger, metricsCollector core.MetricsCollector) *BaseCachePlugin {
	return &BaseCachePlugin{
		name:             name,
		version:          version,
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

func (p *BaseCachePlugin) Name() string {
	return p.name
}

func (p *BaseCachePlugin) Version() string {
	return p.version
}

func (p *BaseCachePlugin) Initialize(ctx context.Context, config core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("cache plugin already initialized")
	}

	p.config = &config
	p.initialized = true

	if p.logger != nil {
		p.logger.Info("Cache plugin initialized", "component", "cache")
	}

	return nil
}

func (p *BaseCachePlugin) Start(ctx context.Context) error {
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

func (p *BaseCachePlugin) Stop(ctx context.Context) error {
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

// Health satisfies core.Plugin
func (p *BaseCachePlugin) Health(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return fmt.Errorf("cache plugin not initialized")
	}

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	return nil
}

// HealthCheck satisfies core.CachePlugin
func (p *BaseCachePlugin) HealthCheck(ctx context.Context) error {
	return p.Health(ctx)
}

func (p *BaseCachePlugin) RecordHit() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.hitCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_hit", 1, map[string]string{})
	}
}

func (p *BaseCachePlugin) RecordMiss() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.missCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_miss", 1, map[string]string{})
	}
}

func (p *BaseCachePlugin) RecordEviction() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.evictionCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_eviction", 1, map[string]string{})
	}
}

func (p *BaseCachePlugin) recordHitCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.hitCount
}

func (p *BaseCachePlugin) recordMissCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.missCount
}

func (p *BaseCachePlugin) recordEvictionCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.evictionCount
}

type DefaultInMemoryCachePlugin struct {
	*BaseCachePlugin
	data map[string][]byte
	ttls map[string]time.Time
}

func NewDefaultInMemoryCachePlugin(name, version string, logger core.Logger, metricsCollector core.MetricsCollector) *DefaultInMemoryCachePlugin {
	return &DefaultInMemoryCachePlugin{
		BaseCachePlugin: NewBaseCachePlugin(name, version, logger, metricsCollector),
		data:            make(map[string][]byte),
		ttls:            make(map[string]time.Time),
	}
}

// Get satisfies core.CachePlugin.Get
func (p *DefaultInMemoryCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("cache plugin not running")
	}

	value, exists := p.data[key]
	if !exists {
		p.mu.RUnlock()
		p.RecordMiss()
		return nil, nil
	}

	expiresAt, hasTTL := p.ttls[key]
	if hasTTL && expiresAt.Before(time.Now()) {
		p.mu.RUnlock()
		p.RecordEviction()
		p.mu.Lock()
		delete(p.data, key)
		delete(p.ttls, key)
		p.mu.Unlock()
		return nil, nil
	}

	p.mu.RUnlock()
	p.RecordHit()
	return value, nil
}

// Set satisfies core.CachePlugin.Set
func (p *DefaultInMemoryCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	if ttl > 0 {
		p.ttls[key] = time.Now().Add(time.Duration(ttl) * time.Second)
	} else {
		p.ttls[key] = time.Now().Add(defaultCacheTTL)
	}

	p.data[key] = value

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_set", 1, map[string]string{})
	}

	return nil
}

// Delete satisfies core.CachePlugin.Delete
func (p *DefaultInMemoryCachePlugin) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	delete(p.data, key)
	delete(p.ttls, key)

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_delete", 1, map[string]string{})
	}

	return nil
}

// Clear removes all cache entries
func (p *DefaultInMemoryCachePlugin) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("cache plugin not running")
	}

	p.data = make(map[string][]byte)
	p.ttls = make(map[string]time.Time)

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("cache_clear", 1, map[string]string{})
	}

	return nil
}

// GetStats satisfies core.CachePlugin.GetStats
func (p *DefaultInMemoryCachePlugin) GetStats() core.CacheStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalSize := int64(0)
	for _, value := range p.data {
		totalSize += int64(len(value))
	}

	return core.CacheStats{
		HitCount:      p.hitCount,
		MissCount:     p.missCount,
		EvictionCount: p.evictionCount,
		TotalSize:     totalSize,
		EntryCount:    int64(len(p.data)),
	}
}
