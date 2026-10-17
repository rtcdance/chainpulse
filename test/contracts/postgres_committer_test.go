//go:build integration

package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
)

// TestPostgresBatchCommitter exercises the shared commit contract (F05) end to
// end against a real database: an empty block and an all-duplicate block must
// both advance the checkpoint, and draining must stop new batches.
func TestPostgresBatchCommitter(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)
	resetLeases(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-committer"

	gate := bootstrap.NewDrainController()
	committer, err := processor.NewBatchCommitter(a, gate, chainID, "worker-A", time.Minute)
	require.NoError(t, err)

	// 1. A normal block.
	require.NoError(t, committer.CommitBlock(ctx, 1, "0xblock1", []*blockchain.BlockchainEvent{
		txTestEvent(chainID, 1),
	}))
	assert.Equal(t, 1, countChainEvents(t, dsn, chainID))
	block, hash := readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(1), block)
	assert.Equal(t, "0xblock1", hash)

	// 2. An empty block: no events, but the checkpoint must still advance.
	require.NoError(t, committer.CommitBlock(ctx, 2, "0xblock2", nil))
	assert.Equal(t, 1, countChainEvents(t, dsn, chainID), "an empty block adds no events")
	block, hash = readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(2), block, "an empty block must still advance the checkpoint")
	assert.Equal(t, "0xblock2", hash)

	// 3. An all-duplicate block: same event as block 1, so ON CONFLICT DO
	//    NOTHING suppresses it, but the checkpoint must still advance.
	require.NoError(t, committer.CommitBlock(ctx, 3, "0xblock3", []*blockchain.BlockchainEvent{
		txTestEvent(chainID, 1),
	}))
	assert.Equal(t, 1, countChainEvents(t, dsn, chainID), "the duplicate must not be stored twice")
	block, _ = readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(3), block, "a duplicate-only block must still advance the checkpoint")

	// 4. Draining stops new batches.
	require.True(t, gate.Drain(time.Second))
	err = committer.CommitBlock(ctx, 4, "0xblock4", nil)
	require.ErrorIs(t, err, processor.ErrDraining)
	block, _ = readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(3), block, "a rejected batch must not move the checkpoint")

	// 5. A rebalanced-away owner is fenced out: worker-A releases the lease,
	//    worker-B takes over, and worker-A can no longer commit.
	require.NoError(t, committer.Release(ctx))

	other := newAdapter(t, dsn)
	_, err = other.AcquireLease(ctx, chainID, "worker-B", time.Minute)
	require.NoError(t, err, "another worker may take over once the lease is released")

	stale, err := processor.NewBatchCommitter(a, nil, chainID, "worker-A", time.Minute)
	require.NoError(t, err)
	err = stale.CommitBlock(ctx, 5, "0xblock5", nil)
	require.Error(t, err, "the fenced-out worker must not commit")
	block, _ = readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(3), block, "a fenced-out commit must not move the checkpoint")

	require.NoError(t, other.ReleaseLease(ctx, chainID, "worker-B"))
}
