package core

import (
	"context"
	"sync"
	"testing"
)

// BenchmarkEventBusPublish measures the throughput of publishing events
// to a single subscriber. This is the common case — one publisher, one indexer.
func BenchmarkEventBusPublish(b *testing.B) {
	logger := &benchEventBusLogger{}
	eb := NewEventBus(logger)

	received := make(chan struct{}, 1)
	_, _ = eb.Subscribe(context.Background(), "test-topic", func(_ any) {
		select {
		case received <- struct{}{}:
		default:
		}
	})

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = eb.Publish(ctx, "test-topic", "hello")
		<-received // wait for handler to complete
	}
}

// BenchmarkEventBusPublishMultiSubscriber measures throughput with 10 subscribers.
func BenchmarkEventBusPublishMultiSubscriber(b *testing.B) {
	logger := &benchEventBusLogger{}
	eb := NewEventBus(logger)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		_, _ = eb.Subscribe(context.Background(), "test-topic", func(_ any) {
			wg.Done()
		})
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wg.Add(10)
		_ = eb.Publish(ctx, "test-topic", "hello")
		wg.Wait()
	}
}

// BenchmarkEventBusPublishSync measures synchronous publish throughput.
func BenchmarkEventBusPublishSync(b *testing.B) {
	logger := &benchEventBusLogger{}
	eb := NewEventBus(logger)

	_, _ = eb.Subscribe(context.Background(), "test-topic", func(_ any) {})

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = eb.PublishSync(ctx, "test-topic", "hello")
	}
}

// benchEventBusLogger is a no-op logger for benchmarks
type benchEventBusLogger struct{}

func (l *benchEventBusLogger) Debug(_ string, _ ...any)          {}
func (l *benchEventBusLogger) Info(_ string, _ ...any)           {}
func (l *benchEventBusLogger) Warn(_ string, _ ...any)           {}
func (l *benchEventBusLogger) Error(_ string, _ ...any)          {}
func (l *benchEventBusLogger) Fatal(_ string, _ ...any)          {}
func (l *benchEventBusLogger) WithCorrelationID(_ string) Logger { return l }
