package cache

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkInMemoryCache_Set(b *testing.B) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.Set(ctx, key, []byte("value"), 3600)
	}
}

func BenchmarkInMemoryCache_Get(b *testing.B) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	c.Set(ctx, "bench-key", []byte("bench-value"), 3600)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ctx, "bench-key")
	}
}

func BenchmarkInMemoryCache_Delete(b *testing.B) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("del-%d", i)
		c.Set(ctx, key, []byte("v"), 60)
		c.Delete(ctx, key)
	}
}

func BenchmarkInMemoryCache_ParallelReads(b *testing.B) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	c.Set(ctx, "shared", []byte("data"), 3600)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get(ctx, "shared")
		}
	})
}

func BenchmarkInMemoryCache_ParallelWrites(b *testing.B) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("w-%d", i)
			c.Set(ctx, key, []byte("v"), 60)
			i++
		}
	})
}

func BenchmarkInMemoryCache_GetStats(b *testing.B) {
	c := NewInMemoryCache()
	_ = c.Start()
	defer c.Stop()

	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("k-%d", i)
		c.Set(ctx, key, []byte("v"), 3600)
		c.Get(ctx, key)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetStats()
	}
}
