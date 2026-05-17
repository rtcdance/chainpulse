package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestAdvancedCacheInitialize(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewAdvancedInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	err := cache.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if cache.GetMaxSize() != 1024*1024*100 {
		t.Fatalf("Expected max size 100MB, got %d", cache.GetMaxSize())
	}

	if cache.GetMaxEntries() != 10000 {
		t.Fatalf("Expected max entries 10000, got %d", cache.GetMaxEntries())
	}

	// Test double initialization
	err = cache.Initialize(config)
	if err == nil {
		t.Fatal("Expected error on double initialization")
	}
}

func TestAdvancedCacheLifecycle(t *testing.T) {
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

	// Start
	err := cache.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check health
	health := cache.Health()
	if health.Status != "healthy" {
		t.Fatalf("Expected healthy status, got %s", health.Status)
	}

	// Stop
	err = cache.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Check health after stop
	health = cache.Health()
	if health.Status != "unhealthy" {
		t.Fatalf("Expected unhealthy status after stop, got %s", health.Status)
	}
}

func TestAdvancedCacheSetAndGet(t *testing.T) {
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

	// Set a value
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	err := cache.Set(entry)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value
	retrieved, err := cache.Get("test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve value")
	}

	if retrieved.Value != "test_value" {
		t.Fatalf("Expected test_value, got %v", retrieved.Value)
	}

	_ = cache.Stop()
}

func TestAdvancedCacheMaxSizeEviction(t *testing.T) {
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

	// Set max size to small value
	_ = cache.SetMaxSize(1000)

	// Add entries until eviction occurs
	for i := 0; i < 20; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d_with_some_extra_data", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Verify size is within limit
	if cache.GetCurrentSize() > 1000 {
		t.Fatalf("Expected size <= 1000, got %d", cache.GetCurrentSize())
	}

	_ = cache.Stop()
}

func TestAdvancedCacheMaxEntriesEviction(t *testing.T) {
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

	// Set max entries to small value
	if err := cache.SetMaxEntries(10); err != nil {
		t.Fatalf("failed to set max entries: %v", err)
	}

	// Add more entries than max
	for i := 0; i < 20; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	// Verify entries count is within limit
	if cache.GetCurrentEntries() > 10 {
		t.Fatalf("Expected entries <= 10, got %d", cache.GetCurrentEntries())
	}

	_ = cache.Stop()
}

func TestAdvancedCacheLRUEviction(t *testing.T) {
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

	// Set max entries to 5
	if err := cache.SetMaxEntries(5); err != nil {
		t.Fatalf("failed to set max entries: %v", err)
	}

	// Add 5 entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		if err := cache.Set(entry); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Access key_0 to make it recently used
	_, _ = cache.Get("key_0")

	// Add a new entry (should evict key_1 as it's least recently used)
	entry := &CacheEntry{
		Key:   "key_5",
		Value: "value_5",
		TTL:   3600,
	}
	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify key_0 still exists (was accessed)
	retrieved, _ := cache.Get("key_0")
	if retrieved == nil {
		t.Fatal("Expected key_0 to still exist")
	}

	_ = cache.Stop()
}

func TestAdvancedCacheExpiration(t *testing.T) {
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

	// Set entry with short TTL
	entry := &CacheEntry{
		Key:   "expire_key",
		Value: "expire_value",
		TTL:   1,
	}

	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	retrieved, _ := cache.Get("expire_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist immediately")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should not exist after expiration
	retrieved, _ = cache.Get("expire_key")
	if retrieved != nil {
		t.Fatal("Expected value to be expired")
	}

	_ = cache.Stop()
}

func TestAdvancedCacheDelete(t *testing.T) {
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

	// Set a value
	entry := &CacheEntry{
		Key:   "delete_key",
		Value: "delete_value",
		TTL:   3600,
	}

	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it exists
	retrieved, _ := cache.Get("delete_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist")
	}

	// Delete it
	_ = cache.Delete("delete_key")

	// Verify it's gone
	retrieved, _ = cache.Get("delete_key")
	if retrieved != nil {
		t.Fatal("Expected value to be deleted")
	}

	_ = cache.Stop()
}

func TestAdvancedCacheClear(t *testing.T) {
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

	// Add multiple entries
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

	_ = cache.Stop()
}

func TestAdvancedCacheStats(t *testing.T) {
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

	// Add entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		if err := cache.Set(entry); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
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

	_ = cache.Stop()
}

func TestAdvancedCacheSetMaxSize(t *testing.T) {
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

	// Set initial max size
	_ = cache.SetMaxSize(5000)

	if cache.GetMaxSize() != 5000 {
		t.Fatalf("Expected max size 5000, got %d", cache.GetMaxSize())
	}

	// Test invalid size
	err := cache.SetMaxSize(-1)
	if err == nil {
		t.Fatal("Expected error for negative size")
	}

	_ = cache.Stop()
}

func TestAdvancedCacheSetMaxEntries(t *testing.T) {
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

	// Set max entries
	_ = cache.SetMaxEntries(100)

	if cache.GetMaxEntries() != 100 {
		t.Fatalf("Expected max entries 100, got %d", cache.GetMaxEntries())
	}

	// Test invalid count
	err := cache.SetMaxEntries(0)
	if err == nil {
		t.Fatal("Expected error for zero entries")
	}

	_ = cache.Stop()
}

func TestAdvancedCacheEvictionPolicy(t *testing.T) {
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

	// Test default policy
	if cache.GetEvictionPolicy() != "LRU" {
		t.Fatalf("Expected LRU policy, got %s", cache.GetEvictionPolicy())
	}

	// Test setting policy
	_ = cache.SetEvictionPolicy("FIFO")
	if cache.GetEvictionPolicy() != "FIFO" {
		t.Fatalf("Expected FIFO policy, got %s", cache.GetEvictionPolicy())
	}

	// Test invalid policy
	err := cache.SetEvictionPolicy("INVALID")
	if err == nil {
		t.Fatal("Expected error for invalid policy")
	}

	_ = cache.Stop()
}

func TestAdvancedCacheConcurrentOperations(t *testing.T) {
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

	// Concurrent set operations
	done := make(chan bool, 20)

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

	_ = cache.Stop()
}

func TestAdvancedCacheCurrentSize(t *testing.T) {
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

	// Initial size should be 0
	if cache.GetCurrentSize() != 0 {
		t.Fatalf("Expected initial size 0, got %d", cache.GetCurrentSize())
	}

	// Add entry and check size increases
	entry := &CacheEntry{
		Key:   "test_key",
		Value: "test_value",
		TTL:   3600,
	}

	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if cache.GetCurrentSize() <= 0 {
		t.Fatalf("Expected size > 0 after set, got %d", cache.GetCurrentSize())
	}

	_ = cache.Stop()
}

func TestAdvancedCacheCurrentEntries(t *testing.T) {
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

	// Initial entries should be 0
	if cache.GetCurrentEntries() != 0 {
		t.Fatalf("Expected initial entries 0, got %d", cache.GetCurrentEntries())
	}

	// Add entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: fmt.Sprintf("value_%d", i),
			TTL:   3600,
		}
		_ = cache.Set(entry)
	}

	if cache.GetCurrentEntries() != 5 {
		t.Fatalf("Expected 5 entries, got %d", cache.GetCurrentEntries())
	}

	_ = cache.Stop()
}
