package processor

import (
	"context"
	"fmt"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
)

// mockBenchStorage is a no-op storage for benchmarking processor logic.
type mockBenchStorage struct{}

func (m *mockBenchStorage) WriteEvent(_ context.Context, _ *core.BlockchainEvent) error { return nil }

func BenchmarkProcessEvent(b *testing.B) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)

	processor := NewDefaultEventProcessor(
		logger,
		metrics,
		idempotency,
		nil, // no cache
		&mockBenchStorage{},
		nil, // no event bus
	)
	_ = processor.Initialize(&core.Config{
		ServiceName: "bench",
	})
	_ = processor.Start()

	event := &core.BlockchainEvent{
		ID:              "bench-event-1",
		ChainID:         "1",
		BlockNumber:     100,
		BlockHash:       common.HexToHash("0xabc"),
		TransactionHash: common.HexToHash("0xdef"),
		ContractAddress: common.HexToAddress("0x1234"),
		EventName:       "Transfer",
		EventSignature:  common.HexToHash("0xddf252ad"),
		LogIndex:        0,
		Network:         "ethereum",
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		event.ID = common.HexToHash(fmt.Sprintf("0x%x", i)).Hex()
		_ = processor.ProcessEvent(ctx, event)
	}
}
