package reorg

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// failingBlockHashProvider simulates an RPC/DB failure while scanning for a reorg.
type failingBlockHashProvider struct{ err error }

func (p failingBlockHashProvider) GetBlockHash(context.Context, uint64) (common.Hash, error) {
	return common.Hash{}, p.err
}

// stubCheckpointStore records successful checkpoint writes.
type stubCheckpointStore struct{ saved int }

func (s *stubCheckpointStore) GetLastIndexedBlock(context.Context, string) (uint64, string, error) {
	return 0, "", nil
}

func (s *stubCheckpointStore) SaveLastIndexedBlock(context.Context, string, uint64, string) error {
	s.saved++
	return nil
}

func (s *stubCheckpointStore) GetBlockHash(context.Context, string, uint64) (string, error) {
	return "", nil
}

// TestDetectReorg_ScanErrorIsNotReportedAsReorg is the regression test for the
// silent-failure defect: when the block-hash scan errors out, DetectReorg used
// to return (true, currentBlock, nil), i.e. it *invented* a reorg at the chain
// head. A failed scan must surface the error and must not claim a reorg.
func TestDetectReorg_ScanErrorIsNotReportedAsReorg(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())
	handler.SetBlockHashProvider(failingBlockHashProvider{err: errors.New("rpc unavailable")})

	// Seed the cache so the next call takes the comparison path.
	if _, _, err := handler.DetectReorg(context.Background(), 100, common.HexToHash("0xaaa")); err != nil {
		t.Fatalf("seed call must succeed: %v", err)
	}

	reorg, reorgBlock, err := handler.DetectReorg(context.Background(), 100, common.HexToHash("0xbbb"))
	if err == nil {
		t.Fatal("expected an error when the block-hash provider fails")
	}
	if reorg {
		t.Errorf("must not report a reorg when the scan failed (reorgBlock=%d)", reorgBlock)
	}
}

// TestUpdateBlockHash_CleansCacheWhenCheckpointPersisted pins the invariant that
// cache cleanup must not be disabled by checkpoint persistence.
//
// Note on severity: with the production default (checkpointInterval = 10) the
// guard is skipped on at most one call in ten, so the cache merely overshoots
// reorgThreshold by ~1 entry before the next call cleans it — this is a latent
// footgun, not a live unbounded-growth bug. checkpointInterval is unexported and
// has no setter, so 0 is not reachable in production today; the test uses it to
// pin the invariant so a future change to the cadence cannot silently disable
// cleanup.
func TestUpdateBlockHash_CleansCacheWhenCheckpointPersisted(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())
	store := &stubCheckpointStore{}
	handler.WithCheckpointStore(store)
	handler.checkpointInterval = 0 // persist on every block (worst case)

	const blocks = 50
	for i := uint64(1); i <= blocks; i++ {
		handler.UpdateBlockHash(i, common.BigToHash(new(big.Int).SetUint64(i)))
	}

	if store.saved == 0 {
		t.Fatal("precondition: checkpoints should have been persisted")
	}

	// reorgThreshold is 12 in newTestHandler; the cache must stay bounded.
	if got := len(handler.lastKnownBlocks); got > 20 {
		t.Errorf("block cache grew to %d entries (reorgThreshold=12): cleanup was skipped on checkpoint writes", got)
	}
}
