//go:build integration

package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/stretchr/testify/require"
)

func TestFencedCheckpointRejectsGapsAndUnscheduledBranchChanges(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a := newAdapter(t, dsn)
	chain := "checkpoint-order-delivery"
	committer, err := processor.NewBatchCommitter(a, nil, chain, "checkpoint-order-owner", time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, committer.Release(context.Background())) })
	require.NoError(t, committer.CommitBlock(ctx, 1, "0xbranch-a", nil))
	require.Error(t, committer.CommitBlock(ctx, 3, "0xgap", nil), "block 2 has not completed")
	require.Error(t, committer.CommitBlock(ctx, 1, "0xbranch-b", nil), "a branch change requires a durable reorg transaction")
	block, hash := readCheckpoint(t, dsn, chain)
	require.Equal(t, uint64(1), block)
	require.Equal(t, "0xbranch-a", hash)
	require.NoError(t, committer.CommitBlock(ctx, 1, "0xbranch-a", nil), "same branch replay is idempotent")
	require.NoError(t, committer.CommitBlock(ctx, 2, "0xnext", nil))
	require.NoError(t, committer.CommitBlock(ctx, 1, "0xbranch-a", nil), "late canonical redelivery must not rewind the checkpoint")
	block, hash = readCheckpoint(t, dsn, chain)
	require.Equal(t, uint64(2), block)
	require.Equal(t, "0xnext", hash)
}
