package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Property 12: Cache Hit Return
// Validates: Requirements 3.2
// Data found in Redis cache should be returned immediately
func TestRedisCacheHitReturn(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: All set values are retrievable
	entries := make([]*CacheEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		if err := cache.Set(entries[i]); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Verify all are retrievable
	for i := 0; i < 10; i++ {
		retrieved, err := cache.Get(fmt.Sprintf("key_%d", i))
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved == nil {
			t.Fatalf("Expected to retrieve key_%d", i)
		}

		if retrieved.Value != fmt.Sprintf("value_%d", i) {
			t.Fatalf("Expected value_%d, got %v", i, retrieved.Value)
		}
	}

	_ = cache.Stop(ctx)
}

// Property: Cache hits are recorded in Redis
func TestRedisCacheHitRecording(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Each successful get increments hit count
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	for i := 0; i < 10; i++ {
		_, _ = cache.Get("test_key")
	}

	stats := cache.GetStats()
	if stats.HitCount != 10 {
		t.Fatalf("Expected 10 hits, got %d", stats.HitCount)
	}

	_ = cache.Stop(ctx)
}

// Property: Cache misses are recorded in Redis
func TestRedisCacheMissRecording(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Each failed get increments miss count
	for i := 0; i < 10; i++ {
		_, _ = cache.Get(fmt.Sprintf("nonexistent_key_%d", i))
	}

	stats := cache.GetStats()
	if stats.MissCount != 10 {
		t.Fatalf("Expected 10 misses, got %d", stats.MissCount)
	}

	_ = cache.Stop(ctx)
}

// Property 14: Cache Expiration
// Validates: Requirements 3.4
// Expired data in Redis should be evicted and not returned
func TestRedisCacheExpirationConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Expired entries are not returned
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   1, // 1 second
	}

	_ = cache.Set(entry)

	// Should exist immediately
	retrieved, _ := cache.Get("test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should not exist after expiration
	retrieved, _ = cache.Get("test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be expired")
	}

	_ = cache.Stop(ctx)
}

// Property: Eviction count increases for expired entries in Redis
func TestRedisCacheEvictionTracking(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Each expired entry increments eviction count
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   1, // 1 second
		}

		_ = cache.Set(entry)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Access expired entries to trigger eviction
	for i := 0; i < 5; i++ {
		_, _ = cache.Get(fmt.Sprintf("key_%d", i))
	}

	stats := cache.GetStats()
	if stats.EvictionCount != 5 {
		t.Fatalf("Expected 5 evictions, got %d", stats.EvictionCount)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache maintains data consistency across operations
func TestRedisCacheDataConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Set, Get, Update, Get sequence maintains consistency
	entry1 := &CacheEntry{
		Key:   "consistency_key",
		Value: "value1",
		TTL:   3600,
	}

	_ = cache.Set(entry1)

	retrieved1, _ := cache.Get("consistency_key")
	if retrieved1.Value != "value1" {
		t.Fatalf("Expected value1, got %v", retrieved1.Value)
	}

	// Update the value
	entry2 := &CacheEntry{
		Key:   "consistency_key",
		Value: "value2",
		TTL:   3600,
	}

	_ = cache.Set(entry2)

	retrieved2, _ := cache.Get("consistency_key")
	if retrieved2.Value != "value2" {
		t.Fatalf("Expected value2, got %v", retrieved2.Value)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache delete operation removes entries
func TestRedisCacheDeleteConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Deleted entries are not retrievable
	entry := &CacheEntry{
		Key:   "delete_key",
		Value: "delete_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	// Verify it exists
	retrieved, _ := cache.Get("delete_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist after set")
	}

	// Delete it
	_ = cache.Delete("delete_key")

	// Verify it's gone
	retrieved, _ = cache.Get("delete_key")
	if retrieved != nil {
		t.Fatal("Expected value to be deleted")
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache clear operation removes all entries
func TestRedisCacheClearConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Clear removes all entries
	for i := 0; i < 10; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Verify entries exist
	if cache.GetKeyCount() != 10 {
		t.Fatalf("Expected 10 keys, got %d", cache.GetKeyCount())
	}

	// Clear all
	_ = cache.Clear()

	// Verify all are gone
	if cache.GetKeyCount() != 0 {
		t.Fatalf("Expected 0 keys after clear, got %d", cache.GetKeyCount())
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache exists operation returns correct status
func TestRedisCacheExistsConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Exists returns true for existing keys
	entry := &CacheEntry{
		Key:   "exists_key",
		Value: "exists_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	exists, _ := cache.Exists("exists_key")
	if !exists {
		t.Fatal("Expected key to exist")
	}

	// Property: Exists returns false for non-existing keys
	exists, _ = cache.Exists("nonexistent_key")
	if exists {
		t.Fatal("Expected key to not exist")
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache TTL operation returns correct remaining time
func TestRedisCacheTTLConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: TTL returns remaining time for existing keys
	entry := &CacheEntry{
		Key:   "ttl_key",
		Value: "ttl_value",
		TTL:   10,
	}

	_ = cache.Set(entry)

	ttl, _ := cache.TTL("ttl_key")
	if ttl <= 0 || ttl > 10 {
		t.Fatalf("Expected TTL between 0 and 10, got %d", ttl)
	}

	// Property: TTL returns -2 for non-existing keys
	ttl, _ = cache.TTL("nonexistent_key")
	if ttl != -2 {
		t.Fatalf("Expected TTL -2 for non-existing key, got %d", ttl)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache expire operation updates TTL
func TestRedisCacheExpireConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Expire updates TTL for existing keys
	entry := &CacheEntry{
		Key:   "expire_key",
		Value: "expire_value",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	// Update TTL to 5 seconds
	_ = cache.Expire("expire_key", 5)

	ttl, _ := cache.TTL("expire_key")
	if ttl <= 0 || ttl > 5 {
		t.Fatalf("Expected TTL between 0 and 5, got %d", ttl)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache GetAllKeys returns all non-expired keys
func TestRedisCacheGetAllKeysConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: GetAllKeys returns all set keys
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	keys, _ := cache.GetAllKeys()
	if len(keys) != 5 {
		t.Fatalf("Expected 5 keys, got %d", len(keys))
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache increment operation increases numeric values
func TestRedisCacheIncrementConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Increment creates entry if not exists
	val, _ := cache.Increment("counter", 5)
	if val != 5 {
		t.Fatalf("Expected 5, got %d", val)
	}

	// Property: Increment adds to existing value
	val, _ = cache.Increment("counter", 3)
	if val != 8 {
		t.Fatalf("Expected 8, got %d", val)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache decrement operation decreases numeric values
func TestRedisCacheDecrementConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Decrement creates entry if not exists
	val, _ := cache.Decrement("counter", 5)
	if val != -5 {
		t.Fatalf("Expected -5, got %d", val)
	}

	// Property: Decrement subtracts from existing value
	val, _ = cache.Decrement("counter", 3)
	if val != -8 {
		t.Fatalf("Expected -8, got %d", val)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache statistics are accurate
func TestRedisCacheStatsAccuracy(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Stats reflect cache operations
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Generate hits
	for i := 0; i < 5; i++ {
		_, _ = cache.Get(fmt.Sprintf("key_%d", i))
	}

	// Generate misses
	for i := 0; i < 3; i++ {
		_, _ = cache.Get(fmt.Sprintf("nonexistent_%d", i))
	}

	stats := cache.GetStats()
	if stats.HitCount != 5 {
		t.Fatalf("Expected 5 hits, got %d", stats.HitCount)
	}

	if stats.MissCount != 3 {
		t.Fatalf("Expected 3 misses, got %d", stats.MissCount)
	}

	if stats.EntryCount != 5 {
		t.Fatalf("Expected 5 entries, got %d", stats.EntryCount)
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache ping operation verifies connectivity
func TestRedisCachePingConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Ping succeeds when running
	err := cache.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	_ = cache.Stop(ctx)

	// Property: Ping fails when not running
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	err = cache.Ping(pingCtx)
	if err == nil {
		t.Fatal("Expected Ping to fail when not running")
	}
}

// Property: Redis cache FlushDB operation clears database
func TestRedisCacheFlushDBConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: FlushDB removes all entries
	for i := 0; i < 10; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	_ = cache.FlushDB(context.Background())

	if cache.GetKeyCount() != 0 {
		t.Fatalf("Expected 0 keys after FlushDB, got %d", cache.GetKeyCount())
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache handles concurrent operations safely
func TestRedisCacheConcurrentOperationsProperty(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Concurrent sets and gets maintain consistency
	done := make(chan bool, 20)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 10; j++ {
				entry := &CacheEntry{
					Key:   fmt.Sprintf("key_%d_%d", idx, j),
					Value: fmt.Sprintf("value_%d_%d", idx, j),
					TTL:   3600,
				}
				_ = cache.Set(entry)
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 10; j++ {
				_, _ = cache.Get(fmt.Sprintf("key_%d_%d", idx, j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	if cache.GetKeyCount() != 100 {
		t.Fatalf("Expected 100 keys, got %d", cache.GetKeyCount())
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache handles error conditions gracefully
func TestRedisCacheErrorHandling(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Get with empty key returns error
	_, err := cache.Get("")
	if err == nil {
		t.Fatal("Expected error for empty key")
	}

	// Property: Set with nil entry returns error
	err = cache.Set(nil)
	if err == nil {
		t.Fatal("Expected error for nil entry")
	}

	// Property: Delete with empty key returns error
	err = cache.Delete("")
	if err == nil {
		t.Fatal("Expected error for empty key")
	}

	// Property: Expire with invalid TTL returns error
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}
	_ = cache.Set(entry)

	err = cache.Expire("test_key", -1)
	if err == nil {
		t.Fatal("Expected error for negative TTL")
	}

	_ = cache.Stop(ctx)
}

// Property: Redis cache maintains entry count accuracy
func TestRedisCacheEntryCountAccuracy(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewRedisCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL:  "http://localhost:8545",
		DataPullerType:     "https-jsonrpc",
		APIPort:            8080,
		CacheConnectionURL: "redis://localhost:6379",
	}

	_ = cache.Initialize(config)
	_ = cache.Start(ctx)

	// Property: Entry count increases with Set
	for i := 0; i < 10; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)

		if cache.GetKeyCount() != int64(i+1) {
			t.Fatalf("Expected %d keys, got %d", i+1, cache.GetKeyCount())
		}
	}

	// Property: Entry count decreases with Delete
	for i := 0; i < 10; i++ {
		_ = cache.Delete(fmt.Sprintf("key_%d", i))

		if cache.GetKeyCount() != int64(10-i-1) {
			t.Fatalf("Expected %d keys, got %d", 10-i-1, cache.GetKeyCount())
		}
	}

	_ = cache.Stop(ctx)
}
