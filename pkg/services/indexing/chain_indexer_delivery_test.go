package indexing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

type atomicDeliveryRuntime struct {
	stubSharedBatchRuntime
}

func (*atomicDeliveryRuntime) AtomicCommitsEnabled() bool { return true }

func TestIndexerAtomicFailureDoesNotFallBackToLegacyWrite(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	indexer := NewDefaultChainIndexer("ethereum", db, NewMockCachePlugin(), NewMockLogger(), nil)
	runtime := &atomicDeliveryRuntime{stubSharedBatchRuntime: stubSharedBatchRuntime{err: errors.New("commit failed")}}
	indexer.SetSharedRuntime(runtime, nil)
	err := indexer.IndexEvents(context.Background(), []*blockchain.BlockchainEvent{{ID: "a", ChainID: "ethereum", BlockNumber: 10}})
	require.ErrorContains(t, err, "commit failed")
	require.Zero(t, db.GetStoreCount())
}

func TestIndexerAtomicSuccessDoesNotWriteTwice(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	indexer := NewDefaultChainIndexer("ethereum", db, NewMockCachePlugin(), NewMockLogger(), nil)
	indexer.SetSharedRuntime(&atomicDeliveryRuntime{}, nil)
	require.NoError(t, indexer.IndexEvents(context.Background(), []*blockchain.BlockchainEvent{{ID: "a", ChainID: "ethereum", BlockNumber: 10}}))
	require.Zero(t, db.GetStoreCount(), "the durable committer owns the write")
}
