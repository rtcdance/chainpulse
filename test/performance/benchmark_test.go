package performance

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/cache"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
)

func BenchmarkMemoryMQ_Publish(b *testing.B) {
	mq := mq.NewMemoryMQ()
	_ = mq.Initialize(core.Config{})
	_ = mq.Start()
	defer func() { _ = mq.Stop() }()

	ctx := context.Background()
	msg := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mq.Publish(ctx, "bench-topic", msg)
	}
}

func BenchmarkMemoryMQ_Subscribe(b *testing.B) {
	mq := mq.NewMemoryMQ()
	_ = mq.Initialize(core.Config{})
	_ = mq.Start()
	defer func() { _ = mq.Stop() }()

	ctx := context.Background()
	received := 0

	_ = mq.Subscribe(ctx, "bench-topic", func(msg []byte) {
		received++
	})

	msg := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mq.Publish(ctx, "bench-topic", msg)
	}
}

func BenchmarkInMemoryCache_Set(b *testing.B) {
	cache := cache.NewInMemoryCache()
	_ = cache.Initialize(core.Config{})
	_ = cache.Start()
	defer func() { _ = cache.Stop() }()

	ctx := context.Background()
	value := []byte("benchmark value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Set(ctx, "key", value, 60)
	}
}

func BenchmarkInMemoryCache_Get(b *testing.B) {
	cache := cache.NewInMemoryCache()
	_ = cache.Initialize(core.Config{})
	_ = cache.Start()
	defer func() { _ = cache.Stop() }()

	ctx := context.Background()
	_ = cache.Set(ctx, "key", []byte("value"), 60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "key")
	}
}

func BenchmarkMockDB_StoreEvent(b *testing.B) {
	db := database.NewMockDB()
	_ = db.Initialize(context.Background(), core.Config{})
	_ = db.Start(context.Background())
	defer func() { _ = db.Stop(context.Background()) }()

	ctx := context.Background()
	event := &core.BlockchainEvent{
		ID:          "bench-event",
		BlockNumber: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.StoreEvent(ctx, event)
	}
}

func BenchmarkMockDB_GetEvent(b *testing.B) {
	db := database.NewMockDB()
	_ = db.Initialize(context.Background(), core.Config{})
	_ = db.Start(context.Background())
	defer func() { _ = db.Stop(context.Background()) }()

	ctx := context.Background()
	event := &core.BlockchainEvent{
		ID:          "bench-event",
		BlockNumber: 1000,
	}
	_ = db.StoreEvent(ctx, event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = db.GetEvent(ctx, "bench-event")
	}
}

func BenchmarkMockDB_BatchStore(b *testing.B) {
	db := database.NewMockDB()
	_ = db.Initialize(context.Background(), core.Config{})
	_ = db.Start(context.Background())
	defer func() { _ = db.Stop(context.Background()) }()

	ctx := context.Background()
	events := make([]any, 100)
	for i := 0; i < 100; i++ {
		events[i] = &core.BlockchainEvent{
			ID:          "event-" + string(rune(i)),
			BlockNumber: uint64(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.BatchStoreEvents(ctx, events)
	}
}
