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

func TestPostgresFinalityProofIsFreshScopedAndReorgProtected(t *testing.T) {
	t.Parallel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const chain = "t07-finality-proof"
	db := openContractDBForReturn(t, dsn)
	for _, table := range []string{"chain_finality_observations", "blockchain_events", "blocks", "indexing_state", "chain_leases", "reorg_generations", "reorg_jobs"} {
		_, err := db.ExecContext(ctx, "DELETE FROM "+table+" WHERE chain_id=$1", chain)
		require.NoError(t, err)
	}
	store := newAdapter(t, dsn).WithChainScope(chain)
	committer, err := processor.NewBatchCommitter(store, nil, chain, "t07-finality-owner", time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, committer.Release(context.Background())) })
	headers := make(map[uint64]*types.Header)
	events := make(map[uint64]*blockchain.BlockchainEvent)
	parent := common.HexToHash("0x1234")
	for height := uint64(1); height <= 3; height++ {
		header := &types.Header{Number: new(big.Int).SetUint64(height), Time: 1700000000 + height, ParentHash: parent}
		headers[height] = header
		event := txTestEvent(chain, height)
		event.BlockHash = header.Hash()
		event.ID = core.ComputeEventHash(event)
		event.EventHash = event.ID
		events[height] = event
		require.NoError(t, committer.CommitCompleteBlock(ctx, &blockchain.Block{Number: height, Hash: header.Hash(), ParentHash: parent, Timestamp: int64(header.Time)}, []*blockchain.BlockchainEvent{event}))
		parent = header.Hash()
	}
	proof := &blockchain.FinalityEvidence{Number: 2, Hash: headers[2].Hash(), GenesisHash: common.HexToHash("0xabcd"), ObservedAt: time.Now().UTC()}
	read := func(height uint64) *blockchain.BlockchainEvent {
		t.Helper()
		event, err := store.GetCanonicalEvent(ctx, events[height].ID)
		require.NoError(t, err)
		require.NotNil(t, event)
		return event
	}
	require.NoError(t, store.RecordFinalityObservation(ctx, chain, "safe", proof))
	require.Equal(t, "safe", read(1).State.Finality.State)
	require.Equal(t, "rpc:safe", read(1).State.Finality.Source)
	require.Equal(t, "unknown", read(3).State.Finality.State, "an event above the tagged head is not safe")
	var rawState string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT event_state#>>'{finality,state}' FROM blockchain_events WHERE id=$1`, events[1].ID).Scan(&rawState))
	require.Equal(t, "unknown", rawState, "proof projection never rewrites event history")
	for _, tc := range []struct {
		name   string
		change func(*blockchain.FinalityEvidence)
	}{
		{"stale", func(p *blockchain.FinalityEvidence) { p.ObservedAt = time.Now().Add(-61 * time.Second) }},
		{"future", func(p *blockchain.FinalityEvidence) { p.ObservedAt = time.Now().Add(6 * time.Second) }},
		{"wrong header", func(p *blockchain.FinalityEvidence) { p.Hash = common.HexToHash("0xbad") }},
		{"wrong genesis", func(p *blockchain.FinalityEvidence) { p.GenesisHash = common.HexToHash("0xbad") }},
		{"ahead of commit", func(p *blockchain.FinalityEvidence) { p.Number = 4 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := *proof
			tc.change(&bad)
			require.Error(t, store.RecordFinalityObservation(ctx, chain, "safe", &bad))
		})
	}
	require.Error(t, store.RecordFinalityObservation(ctx, "another-chain", "safe", proof))
	require.Error(t, store.RecordFinalityObservation(ctx, chain, "latest", proof))
	_, err = db.ExecContext(ctx, `UPDATE chain_finality_observations SET proof_at=CURRENT_TIMESTAMP-interval '61 seconds' WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	require.Equal(t, "unknown", read(1).State.Finality.State)
	require.Equal(t, "degraded", read(1).State.Finality.Confidence)
	proof.ObservedAt = time.Now().UTC()
	require.NoError(t, store.RecordFinalityObservation(ctx, chain, "safe", proof))
	require.NoError(t, store.RecordFinalityObservation(ctx, chain, "finalized", proof))
	require.Equal(t, "finalized", read(1).State.Finality.State)
	require.Equal(t, "finalized", read(2).State.Finality.Confidence)
	regression := *proof
	regression.Number = 1
	regression.Hash = headers[1].Hash()
	require.ErrorContains(t, store.RecordFinalityObservation(ctx, chain, "finalized", &regression), "regressed")
	require.NoError(t, store.RecordFinalityObservation(ctx, chain, "safe", nil))
	require.NoError(t, store.RecordFinalityObservation(ctx, chain, "finalized", nil))
	require.Equal(t, "unknown", read(1).State.Finality.State)
	require.Equal(t, "degraded", read(1).State.Finality.Confidence)
	for _, statement := range []string{
		`UPDATE blockchain_events SET status='reorged' WHERE chain_id=$1 AND block_number=1`,
		`DELETE FROM blocks WHERE chain_id=$1 AND number=1`,
		`UPDATE indexing_state SET last_indexed_block=1 WHERE chain_id=$1`,
		`DELETE FROM indexing_state WHERE chain_id=$1`,
		`UPDATE indexing_state SET last_indexed_block=NULL WHERE chain_id=$1`,
		`UPDATE indexing_state SET last_indexed_hash=NULL WHERE chain_id=$1`,
		`UPDATE blockchain_events SET event_data=jsonb_set(event_data,'{raw}','"forged"') WHERE chain_id=$1 AND block_number=1`,
		`UPDATE blocks SET parent_hash='0xdead' WHERE chain_id=$1 AND number=1`,
	} {
		_, err := db.ExecContext(ctx, statement, chain)
		require.Error(t, err, "schema protection must survive rollback to a previous application binary: %s", statement)
	}
	_, err = store.CommitReorg(ctx, chain, 2, 3)
	require.ErrorContains(t, err, "verified finalized boundary", "provider outage must not erase a verified rollback boundary")
	_, err = store.CommitReorg(ctx, chain, 3, 3)
	require.NoError(t, err)
	require.NoError(t, store.RecordFinalityObservation(ctx, chain, "finalized", proof))
	require.Equal(t, "finalized", read(1).State.Finality.State, "a fresh proof can be rebound after a safe reorg above its head")
	var generation int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT generation FROM chain_finality_observations WHERE chain_id=$1 AND tag='finalized'`, chain).Scan(&generation))
	require.Equal(t, int64(1), generation)
	require.Equal(t, "canonical", read(1).State.Canonicality)
	require.Equal(t, "processed", read(1).State.Processing)
}
