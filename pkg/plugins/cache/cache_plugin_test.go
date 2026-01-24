package cache

import (
	"chainpulse/pkg/core"
	"testing"
	"time"
)

func TestCachePluginInitialize(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	err := cache.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test double initialization
	err = cache.Initialize(config)
	if err == nil {
		t.Fatal("Expected error on double initialization")
	}

	// Test nil config
	cache2 := NewDefaultInMemoryCachePlugin(logger, metrics)
	err = cache2.Initialize(nil)
	if err == nil {
		t.Fatal("Expected error with nil config")
	}
}

func TestCachePluginLifecycle(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

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

func TestCachePluginSetAndGet(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

 _ = cache.Initialize(config)
 _ = cache.Start()

	// Set a value
	entry := &core.CacheEntry{
		Key:   "test_key",
		Value: []byte("test_value"),
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

	if retrieved.Key != "test_key" {
		t.Fatalf("Expected key test_key, got %s", retrieved.Key)
	}

	if string(retrieved.Value) != "test_value" {
		t.Fatalf("Expected value test_value, got %v", string(retrieved.Value))
	}

	// Check hit count
	if cache.GetHitCount() != 1 {
		t.Fatalf("Expected hit count 1, got %d", cache.GetHitCount())
	}
}

func TestCachePluginMiss(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

	config := &core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

 _ = cache.Initialize(config)
 _ = cache.Start()

	// Get non-existent value
	retrieved, err := cache.Get("non_existent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved != nil {
		t.Fatal("Expected nil for non-existent key")
	}

	// Check miss count
	if cache.GetMissCount() != 1 {
		t.Fatalf("Expected miss count 1, got %d", cache.GetMissCount())
	}
}

func TestCachePluginDelete(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

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
	entry := &core.CacheEntry{
		Key:   "test_key",
		Value: []byte("test_value"),
		TTL:   3600,
	}

	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it exists
	retrieved, _ := cache.Get("test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist")
	}

	// Delete it
	err := cache.Delete("test_key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	retrieved, _ = cache.Get("test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be deleted")
	}
}

func TestCachePluginClear(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

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

	// Set multiple values
	for i := 0; i < 5; i++ {
		entry := &core.CacheEntry{
			Key:   "key_" + string(rune(i)),
			Value: []byte("value"),
			TTL:   3600,
		}
		if err := cache.Set(entry); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Clear
	err := cache.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify all are gone
	stats := cache.GetStats()
	if stats.EntryCount != 0 {
		t.Fatalf("Expected 0 entries after clear, got %d", stats.EntryCount)
	}
}

func TestCachePluginExpiration(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin(logger, metrics)

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

	// Set a value with short TTL
	entry := &core.CacheEntry{
		Key:       "test_key",
		Value:     []byte("test_value"),
		TTL:       1, // 1 second
		ExpiresAt: time.Now().Add(1 * time.Second),
	}

	if err := cache.Set(entry); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	retrieved, _ := cache.Get("test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist immediately")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should not exist after expiration
	retrieved, _ = cache.Get("test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be expired")
	}
}
