package reorg

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

type benchMockDB struct {
	block *core.Block
}

func (m *benchMockDB) GetBlock(_ context.Context, blockNumber uint64) (*core.Block, error) {
	return m.block, nil
}

// Stub out the remaining DatabasePlugin methods
func (m *benchMockDB) Name() string                   { return "bench" }
func (m *benchMockDB) Version() string                { return "1.0" }
func (m *benchMockDB) Initialize(_ core.Config) error { return nil }
func (m *benchMockDB) Start() error                   { return nil }
func (m *benchMockDB) Stop() error                    { return nil }
func (m *benchMockDB) IsRunning() bool                { return true }
func (m *benchMockDB) Health() error                  { return nil }
func (m *benchMockDB) GetEvent(_ context.Context, _ string) (*core.BlockchainEvent, error) {
	return nil, nil
}
func (m *benchMockDB) QueryEvents(_ context.Context, _ any) ([]any, error) {
	return nil, nil
}
func (m *benchMockDB) GetAllEvents(_ context.Context) ([]*core.BlockchainEvent, error) {
	return nil, nil
}
func (m *benchMockDB) GetEventsByBlockRange(_ context.Context, _, _ uint64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}
func (m *benchMockDB) GetLatestBlock(_ context.Context) (uint64, error)      { return 0, nil }
func (m *benchMockDB) GetAllBlocks(_ context.Context) ([]*core.Block, error) { return nil, nil }
func (m *benchMockDB) GetReorgStats(_ context.Context) (*core.ReorgStats, error) {
	return nil, nil
}
func (m *benchMockDB) StoreEvent(_ context.Context, _ any) error         { return nil }
func (m *benchMockDB) BatchStoreEvents(_ context.Context, _ []any) error { return nil }
func (m *benchMockDB) DeleteEvent(_ context.Context, _ string) error     { return nil }
func (m *benchMockDB) DeleteEventsByBlockRange(_ context.Context, _, _ uint64) (int64, error) {
	return 0, nil
}
func (m *benchMockDB) MarkEventsAsReorged(_ context.Context, _, _ uint64) (int64, error) {
	return 0, nil
}

// benchMockLogger stubs core.Logger
type benchMockLogger struct{}

func (l *benchMockLogger) Debug(_ string, _ ...any)               {}
func (l *benchMockLogger) Info(_ string, _ ...any)                {}
func (l *benchMockLogger) Warn(_ string, _ ...any)                {}
func (l *benchMockLogger) Error(_ string, _ ...any)               {}
func (l *benchMockLogger) Fatal(_ string, _ ...any)               {}
func (l *benchMockLogger) WithCorrelationID(_ string) core.Logger { return l }

// BenchmarkBinarySearchReorg measures the performance of binary search reorg detection.
// With 1000 blocks, binary search should complete in ~10 iterations (log2(1000)).
func BenchmarkBinarySearchReorg(b *testing.B) {
	blockHash := common.HexToHash("0xabc123")
	mockDB := &benchMockDB{
		block: &core.Block{Number: 1, Hash: blockHash},
	}
	logger := &benchMockLogger{}
	rh := NewReorgHandler(mockDB, logger, 1000, 500)

	// Pre-populate lastKnownBlocks with matching hashes (no reorg)
	for i := uint64(1); i <= 1000; i++ {
		rh.lastKnownBlocks[i] = blockHash
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = rh.binarySearchReorg(ctx, 1, 1000)
	}
}

// BenchmarkBinarySearchReorg_WithReorg tests when a reorg exists at block 500.
func BenchmarkBinarySearchReorg_WithReorg(b *testing.B) {
	goodHash := common.HexToHash("0xabc123")
	badHash := common.HexToHash("0xdef456")
	mockDB := &benchMockDB{
		block: &core.Block{Number: 1, Hash: goodHash},
	}
	logger := &benchMockLogger{}
	rh := NewReorgHandler(mockDB, logger, 1000, 500)

	// Blocks 1-499 match, blocks 500+ are reorged (different hash)
	for i := uint64(1); i < 500; i++ {
		rh.lastKnownBlocks[i] = goodHash
	}
	for i := uint64(500); i <= 1000; i++ {
		rh.lastKnownBlocks[i] = badHash
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = rh.binarySearchReorg(ctx, 1, 1000)
	}
}
