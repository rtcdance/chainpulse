package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestInMemoryCache_NameAndVersion(t *testing.T) {
	c := NewInMemoryCache()
	if c.Name() != "inmemory-cache" {
		t.Fatalf("expected 'inmemory-cache', got '%s'", c.Name())
	}
	if c.Version() != "1.0.0" {
		t.Fatalf("expected '1.0.0', got '%s'", c.Version())
	}
}

func TestInMemoryCache_Initialize(t *testing.T) {
	c := NewInMemoryCache()
	if err := c.Initialize(core.Config{}); err != nil {
		t.Fatalf("Initialize should never fail, got: %v", err)
	}
}

func TestInMemoryCache_HealthNotStarted(t *testing.T) {
	c := NewInMemoryCache()
	if err := c.Health(); err == nil {
		t.Fatal("Health on not-started cache should return error")
	}
	if err := c.HealthCheck(context.Background()); err == nil {
		t.Fatal("HealthCheck on not-started cache should return error")
	}
}

func TestInMemoryCache_StartStop(t *testing.T) {
	c := NewInMemoryCache()
	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := c.Health(); err != nil {
		t.Fatalf("Health after Start should be nil: %v", err)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestInMemoryCache_SetAndGet(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "key1", []byte("value1"), 60)
	v, err := c.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(v) != "value1" {
		t.Fatalf("expected 'value1', got '%s'", string(v))
	}
}

func TestInMemoryCache_GetMissing(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	v, err := c.Get(context.Background(), "no-such-key")
	if err != nil {
		t.Fatalf("Get for missing key should not error: %v", err)
	}
	if v != nil {
		t.Fatalf("expected nil for missing key, got '%s'", string(v))
	}
}

func TestInMemoryCache_SetAndDelete(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "key1", []byte("value1"), 60)
	_ = c.Delete(ctx, "key1")
	v, _ := c.Get(ctx, "key1")
	if v != nil {
		t.Fatalf("expected nil after delete, got '%s'", string(v))
	}
}

func TestInMemoryCache_SetOverwrites(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "key", []byte("first"), 60)
	_ = c.Set(ctx, "key", []byte("second"), 60)
	v, _ := c.Get(ctx, "key")
	if string(v) != "second" {
		t.Fatalf("expected 'second' after override, got '%s'", string(v))
	}
}

func TestInMemoryCache_TTLExpiration(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "short-lived", []byte("data"), 0)
	v, _ := c.Get(ctx, "short-lived")
	if v != nil {
		t.Fatal("expected nil for expired item with TTL=0")
	}
}

func TestInMemoryCache_GetStats(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "k1", []byte("v1"), 60)
	_ = c.Set(ctx, "k2", []byte("v2"), 60)
	c.Get(ctx, "k1")
	c.Get(ctx, "k1")
	c.Get(ctx, "k2")
	c.Get(ctx, "no-key")

	stats := c.GetStats()
	if stats.HitCount != 3 {
		t.Fatalf("expected hit count 3, got %d", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Fatalf("expected miss count 1, got %d", stats.MissCount)
	}
}

func TestInMemoryCache_ConcurrentReadWrite(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "shared", []byte("init"), 3600)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.Get(ctx, "shared")
				c.Set(ctx, "shared", []byte("updated"), 3600)
			}
		}(i)
	}
	wg.Wait()

	v, _ := c.Get(ctx, "shared")
	if v == nil {
		t.Fatal("shared key should still exist after concurrent operations")
	}
}

func TestInMemoryCache_StopClearsData(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	c.Set(context.Background(), "persist", []byte("will-clear"), 3600)
	c.Stop()

	// Data should be cleared after Stop
	c.mu.RLock()
	size := len(c.data)
	c.mu.RUnlock()
	if size != 0 {
		t.Fatalf("expected empty data after Stop, got %d entries", size)
	}
}

func TestInMemoryCache_StopCleansUpGoroutine(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	c.Stop()

	// done channel should be closed by Stop — verify no panic on double close
	// by checking started is false (cleanup goroutine should have exited)
	select {
	case <-c.done:
		// done channel is closed as expected
	default:
		t.Fatal("done channel should be closed after Stop")
	}
}

func TestInMemoryCache_StopDoubleCall(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	c.Stop()

	// Second Stop should not panic — close(c.done) is safe because
	// c.done is closed in the first call and cleanup goroutine has exited.
	// However, close(c.done) twice would panic, so we verify this is safe.
	c.Stop()
}

func TestInMemoryCache_HealthAfterStop(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	c.Stop()

	if err := c.Health(); err == nil {
		t.Fatal("Health after Stop should return error")
	}
}

func TestInMemoryCache_EvictionByCleanup(t *testing.T) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	_ = c.Set(ctx, "expired", []byte("old"), 0)
	_ = c.Set(ctx, "alive", []byte("new"), 3600)

	// Force cleanup
	c.mu.Lock()
	now := time.Now()
	for key, item := range c.data {
		if now.After(item.expiresAt) {
			delete(c.data, key)
			c.stats.EvictionCount++
		}
	}
	c.mu.Unlock()

	v, _ := c.Get(ctx, "expired")
	if v != nil {
		t.Fatal("expired key should be nil after eviction")
	}
	v, _ = c.Get(ctx, "alive")
	if string(v) != "new" {
		t.Fatalf("alive key should survive eviction, got '%s'", string(v))
	}
}
