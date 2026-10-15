//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/services/processor"

	"github.com/stretchr/testify/require"
)

func TestOriginalBranchCanBecomeCanonicalAgain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	const chain = "t04-return"
	db := openContractDBForReturn(t, dsn)
	for _, table := range []string{"blockchain_events", "blocks", "indexing_state", "chain_leases", "reorg_jobs", "reorg_generations"} {
		_, err := db.ExecContext(ctx, "DELETE FROM "+table+" WHERE chain_id=$1", chain)
		require.NoError(t, err)
	}
	store := newAdapter(t, dsn).WithChainScope(chain)
	committer, err := processor.NewBatchCommitter(store, nil, chain, "return-owner", time.Minute)
	require.NoError(t, err)
	defer committer.Release(context.Background())
	first := &types.Header{Number: big.NewInt(1), Time: 1700000000}
	base := &blockchain.Block{Number: 1, Hash: first.Hash(), Timestamp: int64(first.Time)}
	require.NoError(t, committer.CommitCompleteBlock(ctx, base, nil))
	original := &types.Header{Number: big.NewInt(2), ParentHash: base.Hash, Time: 1700000001}
	alternate := &types.Header{Number: big.NewInt(2), ParentHash: base.Hash, Time: 1700000002}
	commit := func(header *types.Header) *blockchain.BlockchainEvent {
		block := &blockchain.Block{Number: 2, Hash: header.Hash(), ParentHash: header.ParentHash, Timestamp: int64(header.Time)}
		event := &blockchain.BlockchainEvent{ChainID: chain, BlockNumber: 2, BlockHash: block.Hash, TransactionHash: common.HexToHash("0x111"), ContractAddress: common.HexToAddress("0xabc"), EventName: "Transfer", Status: blockchain.EventStatusPending}
		event.ID = core.ComputeEventHash(event)
		event.EventHash = event.ID
		err := committer.CommitCompleteBlock(ctx, block, []*blockchain.BlockchainEvent{event})
		if errors.Is(err, database.ErrLeaseLost) {
			err = committer.CommitCompleteBlock(ctx, block, []*blockchain.BlockchainEvent{event})
		}
		require.NoError(t, err)
		return event
	}
	event := commit(original)
	_, err = store.CommitReorg(ctx, chain, 2, 2)
	require.NoError(t, err)
	alternateEvent := commit(alternate)
	// Neither an unfenced duplicate nor an unscheduled old branch may
	// resurrect the orphan. Its attempted effects must remain rolled back.
	require.NoError(t, store.StoreEvent(ctx, event))
	orphan, err := store.GetCanonicalEvent(ctx, event.ID)
	require.NoError(t, err)
	require.Nil(t, orphan)
	oldBlock := &blockchain.Block{Number: 2, Hash: original.Hash(), ParentHash: original.ParentHash, Timestamp: int64(original.Time)}
	require.Error(t, committer.CommitCompleteBlock(ctx, oldBlock, []*blockchain.BlockchainEvent{event}))
	orphan, err = store.GetCanonicalEvent(ctx, event.ID)
	require.NoError(t, err)
	require.Nil(t, orphan)
	_, err = store.CommitReorg(ctx, chain, 2, 2)
	require.NoError(t, err)
	commit(original)
	got, err := store.GetCanonicalEvent(ctx, event.ID)
	require.NoError(t, err)
	require.NotNil(t, got, "the original physical identity must become canonical again after A to B to A")
	require.NotNil(t, got.State)
	require.Equal(t, "canonical", got.State.Canonicality)
	require.Equal(t, "processed", got.State.Processing)
	require.Equal(t, "unknown", got.State.Finality.State)
	other, err := store.GetCanonicalEvent(ctx, alternateEvent.ID)
	require.NoError(t, err)
	require.Nil(t, other, "the alternate branch must remain available only as orphan history")
	orphanHistory, err := store.GetEvent(ctx, alternateEvent.ID)
	require.NoError(t, err)
	require.Equal(t, "orphaned", orphanHistory.State.Canonicality)
	require.Equal(t, "processed", orphanHistory.State.Processing)
	var physical, canonical int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER (WHERE status<>'reorged') FROM blockchain_events WHERE chain_id=$1`, chain).Scan(&physical, &canonical))
	require.Equal(t, 2, physical)
	require.Equal(t, 1, canonical)

	require.NoError(t, committer.Release(ctx))
	_, err = store.AcquireLease(ctx, chain, "cross-chain-owner", time.Minute)
	require.NoError(t, err)
	tx, err := store.BeginFencedTx(ctx, chain, "cross-chain-owner")
	require.NoError(t, err)
	foreign := *event
	foreign.ChainID = "another-chain"
	require.ErrorContains(t, tx.StoreEvent(ctx, &foreign), "event chain does not match lease")
	require.NoError(t, tx.Rollback(ctx))
	require.NoError(t, store.ReleaseLease(ctx, chain, "cross-chain-owner"))
}

func openContractDBForReturn(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}
