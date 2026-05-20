package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestCachePluginHitReturn(t *testing.T) {
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
		t.Fatalf("failed to initialize cache: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("failed to start cache: %v", err)
	}
	defer func() { _ = cache.Stop(ctx) }()

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		if err := cache.Set(ctx, key, []byte(fmt.Sprintf("value_%d", i)), 3600); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		retrieved, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if retrieved == nil {
			t.Fatalf("Expected to retrieve %s, got nil", key)
		}
		if string(retrieved) != fmt.Sprintf("value_%d", i) {
			t.Fatalf("Expected value_%d, got %v", i, string(retrieved))
		}
	}
}

func TestCachePluginHitRecording(t *testing.T) {
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
		t.Fatalf("failed to initialize cache: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("failed to start cache: %v", err)
	}
	defer func() { _ = cache.Stop(ctx) }()

	if err := cache.Set(ctx, "test_key", []byte("test_value"), 3600); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	for i := 0; i < 10; i++ {
		_, _ = cache.Get(ctx, "test_key")
	}

	stats := cache.GetStats()
	if stats.HitCount != 10 {
		t.Fatalf("Expected 10 hits, got %d", stats.HitCount)
	}
}

func TestCachePluginExpirationConsistency(t *testing.T) {
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
		t.Fatalf("failed to initialize cache: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("failed to start cache: %v", err)
	}
	defer func() { _ = cache.Stop(ctx) }()

	if err := cache.Set(ctx, "test_key", []byte("test_value"), 1); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, _ := cache.Get(ctx, "test_key")
	if retrieved == nil {
		t.Fatal("Expected value to exist immediately after set")
	}

	time.Sleep(2 * time.Second)

	retrieved, _ = cache.Get(ctx, "test_key")
	if retrieved != nil {
		t.Fatal("Expected value to be expired")
	}
}

func TestCachePluginEvictionTracking(t *testing.T) {
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
		t.Fatalf("failed to initialize cache: %v", err)
	}
	if err := cache.Start(ctx); err != nil {
		t.Fatalf("failed to start cache: %v", err)
	}
	defer func() { _ = cache.Stop(ctx) }()

	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		if err := cache.Set(ctx, key, []byte(fmt.Sprintf("value_%d", i)), 1); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	time.Sleep(2 * time.Second)

	for i := 0; i < 5; i++ {
		_, _ = cache.Get(ctx, fmt.Sprintf("key_%d", i))
	}

	stats := cache.GetStats()
	if stats.EvictionCount < 5 {
		t.Fatalf("Expected at least 5 evictions, got %d", stats.EvictionCount)
	}
}
