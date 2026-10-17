package processor

import (
	"context"
	"errors"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/require"
)

func TestCanonicalValidationRunsInsideFenceBeforeAnyEffect(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	committer := newCommitter(t, store, nil)
	committer.SetBlockValidator(func(context.Context, *blockchain.Block) error {
		require.Len(t, store.txs, 1, "the lease transaction must already exist")
		require.Empty(t, store.txs[0].events)
		require.Nil(t, store.txs[0].checkpoint)
		return errors.New("canonical branch changed")
	})
	err := committer.CommitCompleteBlock(context.Background(), &blockchain.Block{Number: 10}, testEvents(1))
	require.ErrorContains(t, err, "canonical branch changed")
	require.Empty(t, store.committed)
	require.True(t, store.txs[0].rolledBack)
}
