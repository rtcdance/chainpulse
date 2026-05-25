package cache

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestCachePluginInitialize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	err := cache.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = cache.Initialize(ctx, config)
	if err == nil {
		t.Fatal("Expected error on double initialization")
	}
}

func TestCachePluginLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := cache.Health(ctx); err != nil {
		t.Fatalf("Expected healthy status, got error: %v", err)
	}

	if err := cache.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if err := cache.Health(ctx); err == nil {
		t.Fatalf("Expected unhealthy status after stop")
	}
}

func TestCachePluginSetAndGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := cache.Set(ctx, "test_key", []byte("test_value"), 3600); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := cache.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected to retrieve value")
	}
	if string(retrieved) != "test_value" {
		t.Fatalf("Expected value 'test_value', got %v", string(retrieved))
	}

	stats := cache.GetStats()
	if stats.HitCount != 1 {
		t.Fatalf("Expected hit count 1, got %d", stats.HitCount)
	}
}

func TestCachePluginMiss(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	retrieved, err := cache.Get(ctx, "non_existent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved != nil {
		t.Fatal("Expected nil for non-existent key")
	}

	stats := cache.GetStats()
	if stats.MissCount != 1 {
		t.Fatalf("Expected miss count 1, got %d", stats.MissCount)
	}
}

func TestCachePluginDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := cache.Set(ctx, "test_key", []byte("test_value"), 3600); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, _ := cache.Get(ctx, "test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist")
	}

	if err := cache.Delete(ctx, "test_key"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, _ = cache.Get(ctx, "test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be deleted")
	}
}

func TestCachePluginClear(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		key := "key_" + string(rune('0'+i))
		if err := cache.Set(ctx, key, []byte("value"), 3600); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	stats := cache.GetStats()
	if stats.EntryCount != 0 {
		t.Fatalf("Expected 0 entries after clear, got %d", stats.EntryCount)
	}
}

func TestCachePluginExpiration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test-cache", "1.0.0", logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := cache.Initialize(ctx, config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := cache.Set(ctx, "test_key", []byte("test_value"), 1); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, _ := cache.Get(ctx, "test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist immediately")
	}

	time.Sleep(2 * time.Second)

	retrieved, _ = cache.Get(ctx, "test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be expired")
	}
}

func TestNewBaseCachePlugin(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	p := NewBaseCachePlugin("test", "2.0.0", logger, metrics)

	if p == nil {
		t.Fatal("Expected non-nil plugin")
	}
	if p.Name() != "test" {
		t.Fatalf("Expected name 'test', got '%s'", p.Name())
	}
	if p.Version() != "2.0.0" {
		t.Fatalf("Expected version '2.0.0', got '%s'", p.Version())
	}
}

func TestBaseCachePluginName(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("my-plugin", "1.0.0", nil, nil)
	if p.Name() != "my-plugin" {
		t.Fatalf("Expected 'my-plugin', got '%s'", p.Name())
	}
}

func TestBaseCachePluginVersion(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("test", "3.1.4", nil, nil)
	if p.Version() != "3.1.4" {
		t.Fatalf("Expected '3.1.4', got '%s'", p.Version())
	}
}

func TestBaseCachePluginRecordHit(t *testing.T) {
	t.Parallel()
	metrics := core.NewDefaultMetricsCollector()
	p := NewBaseCachePlugin("test", "1.0.0", nil, metrics)
	p.RecordHit()
	if p.hitCount != 1 {
		t.Fatalf("Expected hitCount 1, got %d", p.hitCount)
	}
}

func TestBaseCachePluginRecordMiss(t *testing.T) {
	t.Parallel()
	metrics := core.NewDefaultMetricsCollector()
	p := NewBaseCachePlugin("test", "1.0.0", nil, metrics)
	p.RecordMiss()
	if p.missCount != 1 {
		t.Fatalf("Expected missCount 1, got %d", p.missCount)
	}
}

func TestBaseCachePluginRecordEviction(t *testing.T) {
	t.Parallel()
	metrics := core.NewDefaultMetricsCollector()
	p := NewBaseCachePlugin("test", "1.0.0", nil, metrics)
	p.RecordEviction()
	if p.evictionCount != 1 {
		t.Fatalf("Expected evictionCount 1, got %d", p.evictionCount)
	}
}

func TestBaseCachePluginRecordHitCount(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	p.RecordHit()
	p.RecordHit()
	if p.recordHitCount() != 2 {
		t.Fatalf("Expected hitCount 2, got %d", p.recordHitCount())
	}
}

func TestBaseCachePluginRecordMissCount(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	p.RecordMiss()
	if p.recordMissCount() != 1 {
		t.Fatalf("Expected missCount 1, got %d", p.recordMissCount())
	}
}

func TestBaseCachePluginRecordEvictionCount(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	p.RecordEviction()
	p.RecordEviction()
	p.RecordEviction()
	if p.recordEvictionCount() != 3 {
		t.Fatalf("Expected evictionCount 3, got %d", p.recordEvictionCount())
	}
}

func TestBaseCachePluginStartNotInitialized(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("Expected error starting uninitialized plugin")
	}
}

func TestBaseCachePluginStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	cfg := core.Config{}
	if err := p.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err := p.Start(ctx)
	if err == nil {
		t.Fatal("Expected error starting already running plugin")
	}
}

func TestBaseCachePluginStopNotRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	cfg := core.Config{}
	if err := p.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	err := p.Stop(ctx)
	if err == nil {
		t.Fatal("Expected error stopping not-running plugin")
	}
}

func TestBaseCachePluginHealthNotInitialized(t *testing.T) {
	t.Parallel()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("Expected error for Health on uninitialized plugin")
	}
}

func TestBaseCachePluginHealthNotRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	cfg := core.Config{}
	if err := p.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	err := p.Health(ctx)
	if err == nil {
		t.Fatal("Expected error for Health on not-running plugin")
	}
}

func TestBaseCachePluginHealthCheck(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := NewBaseCachePlugin("test", "1.0.0", nil, nil)
	cfg := core.Config{}
	if err := p.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := p.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestNewDefaultInMemoryCachePlugin(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("inmem", "1.0.0", logger, metrics)
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
	if cache.Name() != "inmem" {
		t.Fatalf("Expected name 'inmem', got '%s'", cache.Name())
	}
}

func TestDefaultInMemoryCachePluginGetEmptyKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	_, err := cache.Get(ctx, "")
	if err == nil {
		t.Fatal("Expected error for empty key")
	}
}

func TestDefaultInMemoryCachePluginSetEmptyKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err := cache.Set(ctx, "", []byte("value"), 3600)
	if err == nil {
		t.Fatal("Expected error for empty key")
	}
}

func TestDefaultInMemoryCachePluginDeleteEmptyKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err := cache.Delete(ctx, "")
	if err == nil {
		t.Fatal("Expected error for empty key")
	}
}

func TestDefaultInMemoryCachePluginClearNotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	err := cache.Clear()
	if err == nil {
		t.Fatal("Expected error clearing not-running cache")
	}
}

func TestDefaultInMemoryCachePluginGetNotRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	_, err := cache.Get(ctx, "key")
	if err == nil {
		t.Fatal("Expected error for Get on not-running cache")
	}
}

func TestDefaultInMemoryCachePluginSetNotRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	err := cache.Set(ctx, "key", []byte("value"), 3600)
	if err == nil {
		t.Fatal("Expected error for Set on not-running cache")
	}
}

func TestDefaultInMemoryCachePluginDeleteNotRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	err := cache.Delete(ctx, "key")
	if err == nil {
		t.Fatal("Expected error for Delete on not-running cache")
	}
}

func TestDefaultInMemoryCacheZeroTTLUsesDefault(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	cache := NewDefaultInMemoryCachePlugin("test", "1.0.0", logger, metrics)
	cfg := core.Config{}
	if err := cache.Initialize(ctx, cfg); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := cache.Set(ctx, "zero_ttl", []byte("data"), 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	retrieved, _ := cache.Get(ctx, "zero_ttl")
	if retrieved == nil {
		t.Fatal("Expected value to exist with default TTL")
	}
}
