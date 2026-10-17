package processor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchCommitterRotatesOwnershipWithoutAcceptingCancelledWork(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	committer := newCommitter(t, store, nil)
	require.NoError(t, committer.EnsureLease(context.Background()))
	require.NoError(t, committer.RotateOwner(context.Background(), "next-generation"))
	_, err := store.BeginFencedTx(context.Background(), "1", "worker-A")
	require.Error(t, err)
	revoked, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, committer.CommitBlock(revoked, 10, "hash", testEvents(1)), context.Canceled)
	require.Empty(t, store.committed)
	require.NoError(t, committer.CommitBlock(context.Background(), 10, "hash", testEvents(1)))
	require.Equal(t, "next-generation", store.owner)
	require.Len(t, store.committed, 1)
	_, err = store.BeginFencedTx(context.Background(), "1", "worker-A")
	require.Error(t, err)
}
