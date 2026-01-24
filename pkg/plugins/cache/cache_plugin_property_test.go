package cache

import (
	"chainpulse/pkg/core"
	"fmt"
	"testing"
	"time"
)

// Property 12: Cache Hit Return
// Validates: Requirements 3.2
// Data found in cache should be returned immediately
func TestCachePluginHitReturn(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: All set values are retrievable
	entries := make([]*core.CacheEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = &core.CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: []byte(fmt.Sprintf("value_%d", i)),
			TTL:   3600,
		}
		_ = cache.Set(entries[i])
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

		if string(retrieved.Value) != fmt.Sprintf("value_%d", i) {
			t.Fatalf("Expected value_%d, got %v", i, string(retrieved.Value))
		}
	}
}

// Property: Cache hits are recorded
func TestCachePluginHitRecording(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Each successful get increments hit count
	entry := &core.CacheEntry{
		Key:   "test_key",
		Value: []byte("test_value"),
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

// Property 14: Cache Expiration
// Validates: Requirements 3.4
// Expired data should be evicted and not returned
func TestCachePluginExpirationConsistency(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}
	_ = cache.Start()
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Expired entries are not returned
	entry := &core.CacheEntry{
		Key:   "test_key",
		Value: []byte("test_value"),
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
}

// Property: Eviction count increases for expired entries
func TestCachePluginEvictionTracking(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(config); err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}
	if err := cache.Start(); err != nil {
		t.Fatalf("failed to start cache: %v", err)
	}
	defer func() {
		_ = cache.Stop()
	}()

	// Property: Each expired entry increments eviction count
	for i := 0; i < 5; i++ {
		entry := &core.CacheEntry{
			Key:   fmt.Sprintf("key_%d", i),
			Value: []byte(fmt.Sprintf("value_%d", i)),
			TTL:   1, // 1 second
		}
  _ = cache.Set(entry)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Access all entries to trigger eviction
	for i := 0; i < 5; i++ {
		_, _ = cache.Get(fmt.Sprintf("key_%d", i))
	}

	if cache.GetEvictionCount() < 5 {
		t.Fatalf("Expected at least 5 evictions, got %d", cache.GetEvictionCount())
	}
}
