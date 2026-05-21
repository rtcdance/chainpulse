package processor

import (
	"context"
	"fmt"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func BenchmarkIsDuplicate(b *testing.B) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	svc := NewDefaultIdempotencyService(logger, metrics)
	ctx := context.Background()

	// Pre-populate 10K entries
	for i := 0; i < 10000; i++ {
		hash := fmt.Sprintf("event-hash-%d", i)
		_ = svc.MarkProcessed(ctx, hash)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("event-hash-%d", i%10000)
		_, _ = svc.IsDuplicate(ctx, hash)
	}
}

func BenchmarkMarkProcessed(b *testing.B) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	svc := NewDefaultIdempotencyService(logger, metrics)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("event-hash-%d", i)
		_ = svc.MarkProcessed(ctx, hash)
	}
}
