//go:build integration

package contracts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/require"
)

func TestPostgresEventStateSeparatesCanonicalProcessingAndFinality(t *testing.T) {
	t.Parallel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	db := openContractDBForReturn(t, dsn)
	const chain = "t04-state"
	_, err := db.ExecContext(ctx, `DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	store := newAdapter(t, dsn).WithChainScope(chain)
	event := &blockchain.BlockchainEvent{ID: chain, ChainID: chain, BlockNumber: 1, BlockHash: common.HexToHash("0x1"), TransactionHash: common.HexToHash("0x2"), EventData: []byte{0, 1, 0, 2}, Status: blockchain.EventStatusPending}
	require.NoError(t, store.StoreEvent(ctx, event))
	read := func() map[string]any {
		t.Helper()
		var raw []byte
		require.NoError(t, db.QueryRowContext(ctx, `SELECT event_state FROM blockchain_events WHERE id=$1`, chain).Scan(&raw))
		var state map[string]any
		require.NoError(t, json.Unmarshal(raw, &state))
		return state
	}
	initial := read()
	require.Equal(t, "canonical", initial["canonicality"])
	require.Equal(t, "pending", initial["processing"])
	for _, step := range []struct{ status, canonicality, processing string }{
		{"confirmed", "canonical", "processed"},
		{"reorged", "orphaned", "processed"},
		{"pending", "canonical", "processed"},
		{"finalized", "canonical", "processed"},
	} {
		_, err := db.ExecContext(ctx, `UPDATE blockchain_events SET status=$2 WHERE id=$1`, chain, step.status)
		require.NoError(t, err)
		state := read()
		require.Equal(t, step.canonicality, state["canonicality"])
		require.Equal(t, step.processing, state["processing"])
		finality := state["finality"].(map[string]any)
		require.Equal(t, "unknown", finality["state"], "legacy status is not RPC finality evidence")
	}
	got, err := store.GetEvent(ctx, chain)
	require.NoError(t, err)
	require.Equal(t, event.EventData, got.EventData)
	require.NotNil(t, got.State)
	require.Equal(t, "processed", got.State.Processing)
	require.Equal(t, "unknown", got.State.Finality.State)
	require.Equal(t, blockchain.EventStatusFinalized, got.Status, "legacy API status remains readable")
	_, err = db.ExecContext(ctx, `UPDATE blockchain_events SET event_state=jsonb_set(event_state,'{finality}','{"state":"finalized","confidence":"finalized","source":"rpc:finalized"}') WHERE id=$1`, chain)
	require.Error(t, err, "finality without observation/head evidence must be rejected")
}
