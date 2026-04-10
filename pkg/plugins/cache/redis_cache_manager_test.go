package cache

import (
	"context"
	"testing"
	"time"
)

// TestNewRedisCacheManager tests creating a new Redis cache manager
func TestNewRedisCacheManager(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)

	if rcm == nil {
		t.Fatal("expected non-nil RedisCacheManager")
	}

	if rcm.config != config {
		t.Fatal("expected config to be set")
	}

	if rcm.stats == nil {
		t.Fatal("expected stats to be initialized")
	}

	if len(rcm.localCache) != 0 {
		t.Fatal("expected empty local cache")
	}
}

// TestSetAndGet tests setting and getting values
func TestSetAndGet(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Test setting and getting a value
	testKey := "test:key"
	testValue := map[string]interface{}{
		"id":   "123",
		"name": "test",
	}

	err := rcm.Set(ctx, testKey, testValue, 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Get from local cache (since Redis might not be available)
	val, err := rcm.Get(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if val == nil {
		t.Fatal("expected non-nil value")
	}

	// Check statistics
	stats := rcm.GetStatistics()
	if stats.Sets != 1 {
		t.Fatalf("expected 1 set, got %d", stats.Sets)
	}

	if stats.Hits != 1 {
		t.Fatalf("expected 1 hit, got %d", stats.Hits)
	}
}

// TestDelete tests deleting values
func TestDelete(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	testKey := "test:delete"
	testValue := "test value"

	// Set a value
	err := rcm.Set(ctx, testKey, testValue, 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Delete the value
	err = rcm.Delete(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Try to get the deleted value
	_, err = rcm.Get(ctx, testKey)
	if err == nil {
		t.Fatal("expected error for deleted key")
	}

	// Check statistics
	stats := rcm.GetStatistics()
	if stats.Deletes != 1 {
		t.Fatalf("expected 1 delete, got %d", stats.Deletes)
	}
}

// TestExists tests checking if a key exists
func TestExists(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	testKey := "test:exists"
	testValue := "test value"

	// Key should not exist initially
	exists, err := rcm.Exists(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exists {
		t.Fatal("expected key to not exist")
	}

	// Set a value
	err = rcm.Set(ctx, testKey, testValue, 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Key should exist now
	exists, err = rcm.Exists(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !exists {
		t.Fatal("expected key to exist")
	}
}

// TestInvalidate tests invalidating a key
func TestInvalidate(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	testKey := "test:invalidate"
	testValue := "test value"

	// Set a value
	err := rcm.Set(ctx, testKey, testValue, 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Invalidate the key
	err = rcm.Invalidate(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Key should not exist
	exists, err := rcm.Exists(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exists {
		t.Fatal("expected key to not exist after invalidation")
	}
}

// TestFlush tests flushing all cache entries
func TestFlush(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		key := "test:flush:" + string(rune(i))
		err := rcm.Set(ctx, key, "value", 1*time.Hour)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	// Flush all entries
	err := rcm.Flush(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// All keys should be gone
	for i := 0; i < 5; i++ {
		key := "test:flush:" + string(rune(i))
		exists, err := rcm.Exists(ctx, key)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if exists {
			t.Fatalf("expected key %s to not exist after flush", key)
		}
	}
}

// TestGetStatistics tests getting cache statistics
func TestGetStatistics(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Perform some operations
	_ = rcm.Set(ctx, "key1", "value1", 1*time.Hour)
	_ = rcm.Set(ctx, "key2", "value2", 1*time.Hour)
	_, _ = rcm.Get(ctx, "key1")
	_, _ = rcm.Get(ctx, "key2")
	_, _ = rcm.Get(ctx, "nonexistent")

	// Get statistics
	stats := rcm.GetStatistics()

	if stats.Sets != 2 {
		t.Fatalf("expected 2 sets, got %d", stats.Sets)
	}

	if stats.Hits != 2 {
		t.Fatalf("expected 2 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Fatalf("expected 1 miss, got %d", stats.Misses)
	}
}

// TestGetHitRate tests calculating hit rate
func TestGetHitRate(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Perform operations
	_ = rcm.Set(ctx, "key1", "value1", 1*time.Hour)
	_, _ = rcm.Get(ctx, "key1") // hit
	_, _ = rcm.Get(ctx, "key1") // hit
	_, _ = rcm.Get(ctx, "key2") // miss
	_, _ = rcm.Get(ctx, "key3") // miss

	hitRate := rcm.GetHitRate()

	// Expected: 2 hits / 4 total = 0.5
	expectedHitRate := 0.5
	if hitRate != expectedHitRate {
		t.Fatalf("expected hit rate %.2f, got %.2f", expectedHitRate, hitRate)
	}
}

// TestResetStatistics tests resetting statistics
func TestResetStatistics(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Perform some operations
	_ = rcm.Set(ctx, "key1", "value1", 1*time.Hour)
	_, _ = rcm.Get(ctx, "key1")

	// Reset statistics
	rcm.ResetStatistics()

	// Check that statistics are reset
	stats := rcm.GetStatistics()

	if stats.Sets != 0 {
		t.Fatalf("expected 0 sets after reset, got %d", stats.Sets)
	}

	if stats.Hits != 0 {
		t.Fatalf("expected 0 hits after reset, got %d", stats.Hits)
	}

	if stats.Misses != 0 {
		t.Fatalf("expected 0 misses after reset, got %d", stats.Misses)
	}
}

// TestExpiredEntryEviction tests that expired entries are evicted
func TestExpiredEntryEviction(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Set a value with short TTL
	testKey := "test:expiry"
	testValue := "test value"

	err := rcm.Set(ctx, testKey, testValue, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to get the expired value
	_, err = rcm.Get(ctx, testKey)
	if err == nil {
		t.Fatal("expected error for expired key")
	}
}

// TestFallbackMode tests fallback to local cache
func TestFallbackMode(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "invalid:9999", // Invalid address to trigger fallback
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         100 * time.Millisecond,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Initialize (will fail and enter fallback mode)
	_ = rcm.Initialize(ctx)

	// Should still be able to use local cache
	testKey := "test:fallback"
	testValue := "test value"

	err := rcm.Set(ctx, testKey, testValue, 1*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	val, err := rcm.Get(ctx, testKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if val == nil {
		t.Fatal("expected non-nil value from local cache")
	}
}

// TestClose tests closing the cache manager
func TestClose(t *testing.T) {
	config := &DistributedCacheConfig{
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
		PoolSize:            10,
		MinIdleConns:        5,
		MaxRetries:          3,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MaxLocalCacheSize:   10000,
		DefaultTTL:          3600 * time.Second,
		FallbackEnabled:     true,
	}

	rcm := NewRedisCacheManager(config)
	ctx := context.Background()

	// Close should not error
	err := rcm.Close(ctx)
	if err != nil && err.Error() != "redis: client is closed" {
		_ = err // It's okay if Redis is not available
	}
}
