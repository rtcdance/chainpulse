package performance

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/plugins/cache"
	"chainpulse/pkg/plugins/database"
	"chainpulse/pkg/plugins/mq"
	"chainpulse/pkg/core"
)

func BenchmarkMemoryMQ_Publish(b *testing.B) {
	mq := mq.NewMemoryMQ()
	mq.Initialize(core.Config{})
	mq.Start()
	defer mq.Stop()

	ctx := context.Background()
	msg := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mq.Publish(ctx, "bench-topic", msg)
	}
}

func BenchmarkMemoryMQ_Subscribe(b *testing.B) {
	mq := mq.NewMemoryMQ()
	mq.Initialize(core.Config{})
	mq.Start()
	defer mq.Stop()

	ctx := context.Background()
	received := 0

	mq.Subscribe(ctx, "bench-topic", func(msg []byte) {
		received++
	})

	msg := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mq.Publish(ctx, "bench-topic", msg)
	}
}

func BenchmarkInMemoryCache_Set(b *testing.B) {
	cache := cache.NewInMemoryCache()
	cache.Initialize(core.Config{})
	cache.Start()
	defer cache.Stop()

	ctx := context.Background()
	value := []byte("benchmark value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "key", value, 60)
	}
}

func BenchmarkInMemoryCache_Get(b *testing.B) {
	cache := cache.NewInMemoryCache()
	cache.Initialize(core.Config{})
	cache.Start()
	defer cache.Stop()

	ctx := context.Background()
	cache.Set(ctx, "key", []byte("value"), 60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "key")
	}
}

func BenchmarkMockDB_StoreEvent(b *testing.B) {
	db := database.NewMockDB()
	db.Initialize(core.Config{})
	db.Start()
	defer db.Stop()

	ctx := context.Background()
	event := &core.BlockchainEvent{
		ID:          "bench-event",
		BlockNumber: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.StoreEvent(ctx, event)
	}
}

func BenchmarkMockDB_GetEvent(b *testing.B) {
	db := database.NewMockDB()
	db.Initialize(core.Config{})
	db.Start()
	defer db.Stop()

	ctx := context.Background()
	event := &core.BlockchainEvent{
		ID:          "bench-event",
		BlockNumber: 1000,
	}
	db.StoreEvent(ctx, event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetEvent(ctx, "bench-event")
	}
}

func BenchmarkMockDB_BatchStore(b *testing.B) {
	db := database.NewMockDB()
	db.Initialize(core.Config{})
	db.Start()
	defer db.Stop()

	ctx := context.Background()
	events := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		events[i] = &core.BlockchainEvent{
			ID:          "event-" + string(rune(i)),
			BlockNumber: uint64(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.BatchStoreEvents(ctx, events)
	}
}
