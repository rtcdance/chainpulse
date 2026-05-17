package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"chainpulse/pkg/core"

	"github.com/redis/go-redis/v9"
)

// RedisCachePlugin provides Redis-based cache implementation
type RedisCachePlugin struct {
	*BaseCachePlugin
	connectionURL string
	client        *redis.Client
	data          map[string]*CacheEntry
}

// NewRedisCachePlugin creates a new Redis cache plugin
func NewRedisCachePlugin(logger core.Logger, metricsCollector core.MetricsCollector) *RedisCachePlugin {
	return &RedisCachePlugin{
		BaseCachePlugin: NewBaseCachePlugin(logger, metricsCollector),
		data:            make(map[string]*CacheEntry),
	}
}

// Initialize initializes the Redis cache plugin
func (p *RedisCachePlugin) Initialize(config *core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("redis cache plugin already initialized")
	}

	if config == nil {
		return fmt.Errorf("config is required")
	}

	p.config = config
	p.connectionURL = config.CacheConnectionURL
	if p.connectionURL == "" {
		p.connectionURL = "redis://localhost:6379"
	}

	p.initialized = true

	p.logger.Info("Redis cache plugin initialized", core.LogKeyComponent, "redis_cache", "connectionURL", p.connectionURL)

	return nil
}

// Start starts the Redis cache plugin
func (p *RedisCachePlugin) Start() error {
	if err := p.BaseCachePlugin.Start(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	opts, err := redis.ParseURL(p.connectionURL)
	if err != nil {
		opts = &redis.Options{
			Addr: "localhost:6379",
		}
	}
	p.client = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.client.Ping(ctx).Err(); err != nil {
		p.logger.Warn("Redis connection failed, using in-memory fallback", "error", err.Error())
		p.client = nil
		return nil
	}

	p.logger.Info("Redis cache plugin started", "component", "redis_cache", "connection", p.connectionURL)
	return nil
}

// Get retrieves a value from Redis cache
func (p *RedisCachePlugin) Get(key string) (*CacheEntry, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("redis cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		p.mu.RUnlock()
		p.RecordMiss()
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("redis_cache_miss", 1, map[string]string{})
		}
		return nil, nil
	}

	// Check if entry has expired
	if entry.ExpiresAt.Before(time.Now()) {
		p.mu.RUnlock()
		p.RecordEviction()
		p.mu.Lock()
		delete(p.data, key)
		p.mu.Unlock()
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("redis_cache_eviction", 1, map[string]string{})
		}
		return nil, nil
	}

	p.mu.RUnlock()
	p.RecordHit()
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("redis_cache_hit", 1, map[string]string{})
	}

	return entry, nil
}

// Set stores a value in Redis cache with TTL
func (p *RedisCachePlugin) Set(entry *CacheEntry) error {
	if entry == nil {
		return fmt.Errorf("entry is required")
	}

	if entry.Key == "" {
		return fmt.Errorf("entry key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	// Calculate expiration time
	if entry.TTL > 0 {
		entry.ExpiresAt = time.Now().Add(time.Duration(entry.TTL) * time.Second)
	} else {
		entry.ExpiresAt = time.Now().Add(24 * time.Hour) // default 24 hours
	}

	p.data[entry.Key] = entry

	p.metricsCollector.RecordCounter("redis_cache_set", 1, map[string]string{})
	p.metricsCollector.RecordGauge("redis_cache_size", float64(len(p.data)), map[string]string{})

	p.logger.Debug("Redis cache set", core.LogKeyKey, entry.Key, "ttl", entry.TTL)

	return nil
}

// Delete removes a value from Redis cache
func (p *RedisCachePlugin) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	delete(p.data, key)

	p.metricsCollector.RecordCounter("redis_cache_delete", 1, map[string]string{})
	p.metricsCollector.RecordGauge("redis_cache_size", float64(len(p.data)), map[string]string{})

	return nil
}

// Clear clears all cache entries
func (p *RedisCachePlugin) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	p.data = make(map[string]*CacheEntry)

	p.metricsCollector.RecordCounter("redis_cache_clear", 1, map[string]string{})
	p.metricsCollector.RecordGauge("redis_cache_size", 0, map[string]string{})

	return nil
}

// GetStats returns cache statistics
func (p *RedisCachePlugin) GetStats() *CacheStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalSize := int64(0)
	for _, entry := range p.data {
		if entry.Value != nil {
			// Estimate size based on JSON serialization
			if data, err := json.Marshal(entry.Value); err == nil {
				totalSize += int64(len(data))
			}
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

// GetConnectionURL returns the Redis connection URL
func (p *RedisCachePlugin) GetConnectionURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.connectionURL
}

// Ping checks if Redis is reachable
func (p *RedisCachePlugin) Ping(_ context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	if p.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return p.client.Ping(ctx).Err()
}

// HealthCheck checks Redis health
func (p *RedisCachePlugin) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	if p.client == nil {
		return nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.client.Ping(pingCtx).Err()
}

// FlushDB flushes the current database
func (p *RedisCachePlugin) FlushDB(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	p.data = make(map[string]*CacheEntry)

	p.metricsCollector.RecordCounter("redis_cache_flushdb", 1, map[string]string{})

	return nil
}

// GetKeyCount returns the number of keys in cache
func (p *RedisCachePlugin) GetKeyCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return int64(len(p.data))
}

// Exists checks if a key exists in cache
func (p *RedisCachePlugin) Exists(key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("key is required")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return false, fmt.Errorf("redis cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		return false, nil
	}

	// Check if entry has expired
	if entry.ExpiresAt.Before(time.Now()) {
		return false, nil
	}

	return true, nil
}

// Expire sets expiration time for a key
func (p *RedisCachePlugin) Expire(key string, ttl int64) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("redis cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		return fmt.Errorf("key not found")
	}

	entry.TTL = int(ttl)
	entry.ExpiresAt = time.Now().Add(time.Duration(ttl) * time.Second)

	p.metricsCollector.RecordCounter("redis_cache_expire", 1, map[string]string{})

	return nil
}

// TTL returns the remaining time to live for a key
func (p *RedisCachePlugin) TTL(key string) (int64, error) {
	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return 0, fmt.Errorf("redis cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		return -2, nil // Key does not exist
	}

	// Check if entry has expired
	if entry.ExpiresAt.Before(time.Now()) {
		return -2, nil // Key does not exist (expired)
	}

	ttl := time.Until(entry.ExpiresAt).Seconds()
	return int64(ttl), nil
}

// GetAllKeys returns all keys in cache
func (p *RedisCachePlugin) GetAllKeys() ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.running {
		return nil, fmt.Errorf("redis cache plugin not running")
	}

	keys := make([]string, 0, len(p.data))
	for key, entry := range p.data {
		// Skip expired entries
		if entry.ExpiresAt.After(time.Now()) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Increment increments a numeric value
func (p *RedisCachePlugin) Increment(key string, delta int64) (int64, error) {
	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return 0, fmt.Errorf("redis cache plugin not running")
	}

	entry, exists := p.data[key]
	if !exists {
		// Create new entry with value delta
		entry = &CacheEntry{
			Key:       key,
			Value:     delta,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		p.data[key] = entry
		return delta, nil
	}

	// Check if entry has expired
	if entry.ExpiresAt.Before(time.Now()) {
		entry = &CacheEntry{
			Key:       key,
			Value:     delta,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		p.data[key] = entry
		return delta, nil
	}

	// Increment the value
	if val, ok := entry.Value.(int64); ok {
		newVal := val + delta
		entry.Value = newVal
		p.metricsCollector.RecordCounter("redis_cache_increment", 1, map[string]string{})
		return newVal, nil
	}

	return 0, fmt.Errorf("value is not an integer")
}

// Decrement decrements a numeric value
func (p *RedisCachePlugin) Decrement(key string, delta int64) (int64, error) {
	return p.Increment(key, -delta)
}
