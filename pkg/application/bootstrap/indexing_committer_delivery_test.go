package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestMonolithicCommitterRejectsMixedChainBatch(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	c, err := BuildMonolithicBatchCommitter(store, NewDrainController(), []string{"ethereum", "polygon"})
	require.NoError(t, err)
	err = c.CommitBlock(context.Background(), 10, "hash", []*blockchain.BlockchainEvent{
		newTestChainEvent("ethereum", 10), newTestChainEvent("polygon", 10),
	})
	require.Error(t, err)
	require.Empty(t, store.acquired, "mixed-chain input must not acquire a lease or write")
	require.False(t, store.tx.committed)
}

func TestMonolithicCommitterRoutesEmptyBlockExplicitly(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	c, err := BuildMonolithicBatchCommitter(store, NewDrainController(), []string{"ethereum", "polygon"})
	require.NoError(t, err)
	require.NoError(t, c.CommitChainBlock(context.Background(), "polygon", 10, "hash", nil))
	require.Equal(t, []string{"polygon:10:hash"}, store.tx.checkpoints)
	require.Empty(t, store.tx.events)
	require.True(t, store.tx.committed)
}
