package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCacheManager manages distributed caching using Redis
type RedisCacheManager struct {
	client           *redis.Client
	localCache       map[string]*CacheEntry
	localCacheMutex  sync.RWMutex
	stats            *CacheStatistics
	statsMutex       sync.RWMutex
	config           *DistributedCacheConfig
	fallbackMode     bool
	fallbackMutex    sync.RWMutex
	healthCheckTick  *time.Ticker
	done             chan struct{}
}

// CacheEntry represents a cached value with metadata
type CacheEntry struct {
	Key        string
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AccessedAt time.Time
	TTL        int // Time to live in seconds
}

// CacheStatistics tracks cache performance metrics
type CacheStatistics struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	Sets       int64
	Deletes    int64
	Errors     int64
	LastReset  time.Time
	StartTime  time.Time
}

// NewRedisCacheManager creates a new Redis cache manager
func NewRedisCacheManager(config *DistributedCacheConfig) *RedisCacheManager {
	return &RedisCacheManager{
		localCache: make(map[string]*CacheEntry),
		stats: &CacheStatistics{
			StartTime: time.Now(),
			LastReset: time.Now(),
		},
		config:       config,
		done:         make(chan struct{}),
		fallbackMode: false,
	}
}

// Initialize initializes the Redis connection
func (rcm *RedisCacheManager) Initialize(ctx context.Context) error {
	// Create Redis client with connection pooling
	rcm.client = redis.NewClient(&redis.Options{
		Addr:         rcm.config.RedisAddr,
		Password:     rcm.config.RedisPassword,
		DB:           rcm.config.RedisDB,
		PoolSize:     rcm.config.PoolSize,
		MinIdleConns: rcm.config.MinIdleConns,
		MaxRetries:   rcm.config.MaxRetries,
		DialTimeout:  rcm.config.DialTimeout,
		ReadTimeout:  rcm.config.ReadTimeout,
		WriteTimeout: rcm.config.WriteTimeout,
	})

	// Test connection
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rcm.client.Ping(testCtx).Err(); err != nil {
		rcm.setFallbackMode(true)
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	rcm.setFallbackMode(false)

	// Start health check goroutine
	rcm.healthCheckTick = time.NewTicker(rcm.config.HealthCheckInterval)
	go rcm.healthCheckLoop()

	return nil
}

// Get retrieves a value from cache
func (rcm *RedisCacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	rcm.statsMutex.Lock()
	defer rcm.statsMutex.Unlock()

	// Try Redis first if not in fallback mode
	if !rcm.isFallbackMode() {
		val, err := rcm.getFromRedis(ctx, key)
		if err == nil {
			rcm.stats.Hits++
			return val, nil
		}
		if err != redis.Nil {
			rcm.stats.Errors++
		}
	}

	// Try local cache
	val, err := rcm.getFromLocalCache(key)
	if err == nil {
		rcm.stats.Hits++
		return val, nil
	}

	rcm.stats.Misses++
	return nil, fmt.Errorf("key not found: %s", key)
}

// Set stores a value in cache with TTL
func (rcm *RedisCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	rcm.statsMutex.Lock()
	rcm.stats.Sets++
	rcm.statsMutex.Unlock()

	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		rcm.statsMutex.Lock()
		rcm.stats.Errors++
		rcm.statsMutex.Unlock()
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Set in Redis if not in fallback mode and client is initialized
	if !rcm.isFallbackMode() && rcm.client != nil {
		if err := rcm.client.Set(ctx, key, data, ttl).Err(); err != nil {
			rcm.statsMutex.Lock()
			rcm.stats.Errors++
			rcm.statsMutex.Unlock()
		}
	}

	// Always set in local cache
	rcm.setInLocalCache(key, value, ttl)

	return nil
}

// Delete removes a value from cache
func (rcm *RedisCacheManager) Delete(ctx context.Context, key string) error {
	rcm.statsMutex.Lock()
	rcm.stats.Deletes++
	rcm.statsMutex.Unlock()

	// Delete from Redis if not in fallback mode and client is initialized
	if !rcm.isFallbackMode() && rcm.client != nil {
		if err := rcm.client.Del(ctx, key).Err(); err != nil {
			rcm.statsMutex.Lock()
			rcm.stats.Errors++
			rcm.statsMutex.Unlock()
		}
	}

	// Delete from local cache
	rcm.deleteFromLocalCache(key)

	return nil
}

// Exists checks if a key exists in cache
func (rcm *RedisCacheManager) Exists(ctx context.Context, key string) (bool, error) {
	// Check Redis first if not in fallback mode and client is initialized
	if !rcm.isFallbackMode() && rcm.client != nil {
		exists, err := rcm.client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return true, nil
		}
	}

	// Check local cache
	rcm.localCacheMutex.RLock()
	entry, ok := rcm.localCache[key]
	rcm.localCacheMutex.RUnlock()

	if ok && (entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now())) {
		return true, nil
	}

	return false, nil
}

// Invalidate removes a key from cache
func (rcm *RedisCacheManager) Invalidate(ctx context.Context, key string) error {
	return rcm.Delete(ctx, key)
}

// InvalidatePattern removes all keys matching a pattern
func (rcm *RedisCacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	// Invalidate in Redis if not in fallback mode and client is initialized
	if !rcm.isFallbackMode() && rcm.client != nil {
		iter := rcm.client.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			if err := rcm.client.Del(ctx, iter.Val()).Err(); err != nil {
				rcm.statsMutex.Lock()
				rcm.stats.Errors++
				rcm.statsMutex.Unlock()
			}
		}
		if err := iter.Err(); err != nil {
			rcm.statsMutex.Lock()
			rcm.stats.Errors++
			rcm.statsMutex.Unlock()
		}
	}

	// Invalidate in local cache
	rcm.invalidateLocalCachePattern(pattern)

	return nil
}

// Flush clears all cache entries
func (rcm *RedisCacheManager) Flush(ctx context.Context) error {
	// Flush Redis if not in fallback mode and client is initialized
	if !rcm.isFallbackMode() && rcm.client != nil {
		if err := rcm.client.FlushDB(ctx).Err(); err != nil {
			rcm.statsMutex.Lock()
			rcm.stats.Errors++
			rcm.statsMutex.Unlock()
		}
	}

	// Flush local cache
	rcm.localCacheMutex.Lock()
	rcm.localCache = make(map[string]*CacheEntry)
	rcm.localCacheMutex.Unlock()

	return nil
}

// GetStatistics returns cache statistics
func (rcm *RedisCacheManager) GetStatistics() *CacheStatistics {
	rcm.statsMutex.RLock()
	defer rcm.statsMutex.RUnlock()

	stats := *rcm.stats
	return &stats
}

// ResetStatistics resets cache statistics
func (rcm *RedisCacheManager) ResetStatistics() {
	rcm.statsMutex.Lock()
	defer rcm.statsMutex.Unlock()

	rcm.stats = &CacheStatistics{
		StartTime: time.Now(),
		LastReset: time.Now(),
	}
}

// GetHitRate returns the cache hit rate
func (rcm *RedisCacheManager) GetHitRate() float64 {
	rcm.statsMutex.RLock()
	defer rcm.statsMutex.RUnlock()

	total := rcm.stats.Hits + rcm.stats.Misses
	if total == 0 {
		return 0
	}

	return float64(rcm.stats.Hits) / float64(total)
}

// Close closes the Redis connection
func (rcm *RedisCacheManager) Close(_ context.Context) error {
	close(rcm.done)

	if rcm.healthCheckTick != nil {
		rcm.healthCheckTick.Stop()
	}

	if rcm.client != nil {
		return rcm.client.Close()
	}

	return nil
}

// Private helper methods

func (rcm *RedisCacheManager) getFromRedis(ctx context.Context, key string) (interface{}, error) {
	if rcm.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	val, err := rcm.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (rcm *RedisCacheManager) getFromLocalCache(key string) (interface{}, error) {
	rcm.localCacheMutex.RLock()
	defer rcm.localCacheMutex.RUnlock()

	entry, ok := rcm.localCache[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	// Check if expired
	if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("key expired")
	}

	// Update accessed time
	entry.AccessedAt = time.Now()

	return entry.Value, nil
}

func (rcm *RedisCacheManager) setInLocalCache(key string, value interface{}, ttl time.Duration) {
	rcm.localCacheMutex.Lock()
	defer rcm.localCacheMutex.Unlock()

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	rcm.localCache[key] = &CacheEntry{
		Key:        key,
		Value:      value,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		TTL:        int(ttl.Seconds()),
	}

	// Evict expired entries if cache is too large
	if len(rcm.localCache) > rcm.config.MaxLocalCacheSize {
		rcm.evictExpiredEntries()
	}
}

func (rcm *RedisCacheManager) deleteFromLocalCache(key string) {
	rcm.localCacheMutex.Lock()
	defer rcm.localCacheMutex.Unlock()

	delete(rcm.localCache, key)
}

func (rcm *RedisCacheManager) invalidateLocalCachePattern(pattern string) {
	rcm.localCacheMutex.Lock()
	defer rcm.localCacheMutex.Unlock()

	// Simple pattern matching (prefix matching)
	for key := range rcm.localCache {
		if matchPattern(key, pattern) {
			delete(rcm.localCache, key)
		}
	}
}

func (rcm *RedisCacheManager) evictExpiredEntries() {
	now := time.Now()
	evicted := 0

	for key, entry := range rcm.localCache {
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			delete(rcm.localCache, key)
			evicted++
		}
	}

	if evicted > 0 {
		rcm.statsMutex.Lock()
		rcm.stats.Evictions += int64(evicted)
		rcm.statsMutex.Unlock()
	}
}

func (rcm *RedisCacheManager) isFallbackMode() bool {
	rcm.fallbackMutex.RLock()
	defer rcm.fallbackMutex.RUnlock()
	return rcm.fallbackMode
}

func (rcm *RedisCacheManager) setFallbackMode(fallback bool) {
	rcm.fallbackMutex.Lock()
	defer rcm.fallbackMutex.Unlock()
	rcm.fallbackMode = fallback
}

func (rcm *RedisCacheManager) healthCheckLoop() {
	for {
		select {
		case <-rcm.done:
			return
		case <-rcm.healthCheckTick.C:
			if rcm.client != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := rcm.client.Ping(ctx).Err(); err != nil {
					rcm.setFallbackMode(true)
				} else {
					rcm.setFallbackMode(false)
				}
				cancel()
			}
		}
	}
}

// matchPattern performs simple pattern matching (prefix matching)
func matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}
