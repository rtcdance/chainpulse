package cache

import (
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// Property 12: Cache Hit Return
// Validates: Requirements 3.2
// Data found in advanced cache should be returned immediately
func TestAdvancedCacheHitReturn(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

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
}

// Property: Cache hits are recorded in advanced cache
func TestAdvancedCacheHitRecording(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

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

	if cache.GetHitCount() != 10 {
		t.Fatalf("Expected 10 hits, got %d", cache.GetHitCount())
	}
}

// Property: Cache misses are recorded in advanced cache
func TestAdvancedCacheMissRecording(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Each failed get increments miss count
	for i := 0; i < 10; i++ {
		_, _ = cache.Get(fmt.Sprintf("nonexistent_key_%d", i))
	}

	if cache.GetMissCount() != 10 {
		t.Fatalf("Expected 10 misses, got %d", cache.GetMissCount())
	}
}

// Property 14: Cache Expiration
// Validates: Requirements 3.4
// Expired data in advanced cache should be evicted and not returned
func TestAdvancedCacheExpirationConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Property: Expired entries are not returned
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   1, // 1 second
	}

	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

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
}

// Property: LRU eviction maintains cache size limits
func TestAdvancedCacheLRUEvictionConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Property: Cache respects max entries limit with LRU eviction
	_ = cache.SetMaxEntries(5)

	// Add 10 entries
	for i := 0; i < 10; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		if err := cache.Set(entry); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Verify entries count is within limit
	if cache.GetCurrentEntries() > 5 {
		t.Fatalf("Expected entries <= 5, got %d", cache.GetCurrentEntries())
	}
}

// Property: Size-based eviction maintains cache size limits
func TestAdvancedCacheSizeEvictionConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Cache respects max size limit
	_ = cache.SetMaxSize(500)

	// Add entries until size limit is reached
	for i := 0; i < 20; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d_with_extra_data", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Verify size is within limit
	if cache.GetCurrentSize() > 500 {
		t.Fatalf("Expected size <= 500, got %d", cache.GetCurrentSize())
	}
}

// Property: Recently accessed entries are not evicted
func TestAdvancedCacheRecentAccessPreservation(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Recently accessed entries survive eviction
	_ = cache.SetMaxEntries(3)

	// Add 3 entries
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Access key_0 to make it recently used
	_, _ = cache.Get("key_0")

	// Add new entry (should evict key_1 or key_2, not key_0)
	entry := &CacheEntry{
		Key:   "key_3",
		Value: "value_3",
		TTL:   3600,
	}
	_ = cache.Set(entry)

	// Verify key_0 still exists
	retrieved, _ := cache.Get("key_0")
	if retrieved == nil {
		t.Fatal("Expected key_0 to still exist after access")
	}
}

// Property: Delete operation removes entries
func TestAdvancedCacheDeleteConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

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
}

// Property: Clear operation removes all entries
func TestAdvancedCacheClearConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

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
	if cache.GetCurrentEntries() != 10 {
		t.Fatalf("Expected 10 entries, got %d", cache.GetCurrentEntries())
	}

	// Clear all
	_ = cache.Clear()

	// Verify all are gone
	if cache.GetCurrentEntries() != 0 {
		t.Fatalf("Expected 0 entries after clear, got %d", cache.GetCurrentEntries())
	}

	defer func() { _ = cache.Stop() }()
}

// Property: Statistics are accurate
func TestAdvancedCacheStatsAccuracy(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

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
}

// Property: Max size configuration is respected
func TestAdvancedCacheMaxSizeConfiguration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: SetMaxSize updates the limit
	_ = cache.SetMaxSize(2000)

	if cache.GetMaxSize() != 2000 {
		t.Fatalf("Expected max size 2000, got %d", cache.GetMaxSize())
	}
}

// Property: Max entries configuration is respected
func TestAdvancedCacheMaxEntriesConfiguration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: SetMaxEntries updates the limit
	_ = cache.SetMaxEntries(50)

	if cache.GetMaxEntries() != 50 {
		t.Fatalf("Expected max entries 50, got %d", cache.GetMaxEntries())
	}
}

// Property: Eviction policy configuration is respected
func TestAdvancedCacheEvictionPolicyConfiguration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: SetEvictionPolicy updates the policy
	_ = cache.SetEvictionPolicy("FIFO")

	if cache.GetEvictionPolicy() != "FIFO" {
		t.Fatalf("Expected FIFO policy, got %s", cache.GetEvictionPolicy())
	}
}

// Property: Concurrent operations maintain consistency
func TestAdvancedCacheConcurrentConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Property: Concurrent operations don't lose data
	done := make(chan bool, 20)

	// 20 goroutines, each setting 10 entries
	for g := 0; g < 20; g++ {
		go func(goroutineID int) {
			for i := 0; i < 10; i++ {
				entry := &CacheEntry{
					Key:   fmt.Sprintf("key_%d_%d", goroutineID, i),
					Value: fmt.Sprintf("value_%d_%d", goroutineID, i),
					TTL:   3600,
				}
				_ = cache.Set(entry)
			}
			done <- true
		}(g)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify entries (may be less due to eviction)
	if cache.GetCurrentEntries() <= 0 {
		t.Fatalf("Expected some entries, got %d", cache.GetCurrentEntries())
	}
}

// Property: Entry count accuracy
func TestAdvancedCacheEntryCountAccuracy(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Property: Entry count increases with Set
	for i := 0; i < 10; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)

		if cache.GetCurrentEntries() != int64(i+1) {
			t.Fatalf("Expected %d entries, got %d", i+1, cache.GetCurrentEntries())
		}
	}

	// Property: Entry count decreases with Delete
	for i := 0; i < 10; i++ {
		_ = cache.Delete(fmt.Sprintf("key_%d", i))

		if cache.GetCurrentEntries() != int64(10-i-1) {
			t.Fatalf("Expected %d entries, got %d", 10-i-1, cache.GetCurrentEntries())
		}
	}
}

// Property: Size tracking accuracy
func TestAdvancedCacheSizeTrackingAccuracy(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Property: Size increases with Set
	initialSize := cache.GetCurrentSize()

	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value_with_data",
		TTL:   3600,
	}

	_ = cache.Set(entry)

	if cache.GetCurrentSize() <= initialSize {
		t.Fatalf("Expected size to increase, got %d", cache.GetCurrentSize())
	}

	// Property: Size decreases with Delete
	sizeBeforeDelete := cache.GetCurrentSize()
	_ = cache.Delete("test_key")

	if cache.GetCurrentSize() >= sizeBeforeDelete {
		t.Fatalf("Expected size to decrease, got %d", cache.GetCurrentSize())
	}
}

// Property: Eviction count increases when entries are evicted
func TestAdvancedCacheEvictionTracking(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = cache.Initialize(config)
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Eviction count increases when entries are evicted
	_ = cache.SetMaxEntries(3)

	// Add 3 entries
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		if err := cache.Set(entry); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	initialEvictions := cache.GetEvictionCount()

	// Add more entries to trigger eviction
	for i := 3; i < 6; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	if cache.GetEvictionCount() <= initialEvictions {
		t.Fatalf("Expected eviction count to increase, got %d", cache.GetEvictionCount())
	}
}
