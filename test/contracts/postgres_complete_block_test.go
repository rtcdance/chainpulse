//go:build integration

package contracts

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/stretchr/testify/require"
)

func TestPostgresCompleteHeaderAndLogsShareCommit(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a := newAdapter(t, dsn)
	chain := "complete-block-delivery"
	committer, err := processor.NewBatchCommitter(a, nil, chain, "complete-block-owner", time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, committer.Release(context.Background())) })
	header := &types.Header{Number: big.NewInt(1), Time: 1700000000, ParentHash: common.HexToHash("0x1234")}
	block := &blockchain.Block{Number: 1, Hash: header.Hash(), ParentHash: header.ParentHash, Timestamp: int64(header.Time)}
	event := txTestEvent(chain, 1)
	event.BlockHash = block.Hash
	event.ID = core.ComputeEventHash(event)
	event.EventHash = event.ID
	require.NoError(t, committer.CommitCompleteBlock(ctx, block, []*blockchain.BlockchainEvent{event}))
	stored, err := a.GetBlockForChain(ctx, chain, 1)
	require.NoError(t, err)
	require.Equal(t, block.ParentHash, stored.ParentHash)
	require.Equal(t, block.Timestamp, stored.Timestamp)
	bad := &blockchain.Block{Number: 2, Hash: common.HexToHash("0x22"), ParentHash: common.HexToHash("0xff"), Timestamp: 1700000001}
	event = txTestEvent(chain, 2)
	event.BlockHash = bad.Hash
	event.ID = core.ComputeEventHash(event)
	event.EventHash = event.ID
	require.Error(t, committer.CommitCompleteBlock(ctx, bad, []*blockchain.BlockchainEvent{event}), "wrong parent must roll back events and checkpoint together")
	height, hash := readCheckpoint(t, dsn, chain)
	require.Equal(t, uint64(1), height)
	require.Equal(t, block.Hash.Hex(), hash)
	require.Equal(t, 1, countChainEvents(t, dsn, chain))
	bad.ParentHash = block.Hash
	require.NoError(t, committer.CommitCompleteBlock(ctx, bad, nil), "empty block persists full header and advances checkpoint")
	stored, err = a.GetBlockForChain(ctx, chain, 2)
	require.NoError(t, err)
	require.Equal(t, block.Hash, stored.ParentHash)
}
