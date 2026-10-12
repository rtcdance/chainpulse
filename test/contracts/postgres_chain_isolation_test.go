//go:build integration

package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/test/acceptance/tools/digest"
)

const allBlocks = uint64(1) << 62

// setCheckpoint commits a checkpoint for one chain.
func setCheckpoint(t *testing.T, a *database.PostgreSQLPluginAdapter, chainID string, block uint64, hash string) {
	t.Helper()

	ctx := context.Background()
	tx, err := a.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.SaveCheckpoint(ctx, chainID, block, hash))
	require.NoError(t, tx.Commit(ctx))
}

// chainDigest summarises one chain's events, blocks and checkpoint with the
// canonical oracle, so "unchanged" is asserted over CONTENT rather than row
// order (the oracle stable-sorts and dedups by physical identity).
func chainDigest(t *testing.T, a *database.PostgreSQLPluginAdapter, chainID string) string {
	t.Helper()

	ctx := context.Background()

	events, err := a.GetEventsByBlockRangeForChain(ctx, chainID, 0, allBlocks)
	require.NoError(t, err)
	digestEvents := make([]digest.Event, 0, len(events))
	for _, e := range events {
		digestEvents = append(digestEvents, digest.Event{
			Chain:     e.ChainID,
			BlockHash: e.BlockHash.Hex(),
			TxHash:    e.TransactionHash.Hex(),
			LogIndex:  uint32(e.LogIndex),
			// Status is part of the digest so a rollback shows up.
			Data: e.EventName + ":" + string(e.Status),
		})
	}

	blocks, err := a.GetAllBlocksForChain(ctx, chainID)
	require.NoError(t, err)
	digestBlocks := make([]digest.Block, 0, len(blocks)+1)
	for _, b := range blocks {
		digestBlocks = append(digestBlocks, digest.Block{Chain: chainID, Height: b.Number, BlockHash: b.Hash.Hex()})
	}

	cpBlock, cpHash, err := a.GetCheckpointForChain(ctx, chainID)
	require.NoError(t, err)
	digestBlocks = append(digestBlocks, digest.Block{Chain: chainID, Height: cpBlock, BlockHash: cpHash})

	return digest.MustCanonicalEvents(digestEvents) + "|" + digest.MustCanonicalBlocks(digestBlocks)
}

// TestPostgresChainIsolation is the T03 / I01 acceptance test.
//
// Two chains carry an event at the SAME height with the SAME transaction hash
// and log index. Rolling one chain back must leave the other chain's rows,
// block hash, checkpoint and digest byte-for-byte unchanged.
func TestPostgresChainIsolation(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)

	const (
		chainA = "chain-A"
		chainB = "chain-B"
		height = uint64(500)
	)

	// Deliberately identical tx hash and log index across chains: height alone
	// must not be treated as identity (I02), and chain must be part of every
	// lookup (I01).
	txHash := common.HexToHash("0x1234")
	blockHashA := common.HexToHash("0xaaaa0000000000000000000000000000000000000000000000000000000000aa")
	blockHashB := common.HexToHash("0xbbbb0000000000000000000000000000000000000000000000000000000000bb")

	writeChain := func(chainID string, blockHash common.Hash) *blockchain.BlockchainEvent {
		event := &blockchain.BlockchainEvent{
			ChainID:         chainID,
			BlockNumber:     height,
			BlockHash:       blockHash,
			TransactionHash: txHash,
			LogIndex:        0,
			ContractAddress: common.HexToAddress("0xcccc0000000000000000000000000000000000cc"),
			EventName:       "Transfer",
			BlockTimestamp:  1700000000,
		}
		event.EventHash = core.ComputeEventHash(event)
		event.ID = event.EventHash

		require.NoError(t, a.WriteEvent(ctx, event))
		require.NoError(t, a.WriteBlock(ctx, chainID, &blockchain.Block{Number: height, Hash: blockHash}))
		return event
	}

	writeChain(chainA, blockHashA)
	writeChain(chainB, blockHashB)
	setCheckpoint(t, a, chainA, height, blockHashA.Hex())
	setCheckpoint(t, a, chainB, height, blockHashB.Hex())

	beforeB := chainDigest(t, a, chainB)
	beforeA := chainDigest(t, a, chainA)

	// Roll chain A back only.
	affected, err := a.MarkEventsAsReorgedForChain(ctx, chainA, height, height)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "only chain A's row may be affected")

	// A changed...
	assert.NotEqual(t, beforeA, chainDigest(t, a, chainA))

	eventsA, err := a.GetEventsByBlockRangeForChain(ctx, chainA, height, height)
	require.NoError(t, err)
	require.Len(t, eventsA, 1)
	assert.Equal(t, blockchain.EventStatusReorged, eventsA[0].Status)

	// ...B did not: rows, block hash, checkpoint and digest all identical.
	eventsB, err := a.GetEventsByBlockRangeForChain(ctx, chainB, height, height)
	require.NoError(t, err)
	require.Len(t, eventsB, 1)
	assert.Equal(t, blockchain.EventStatusPending, eventsB[0].Status, "chain B must not be reorged")
	assert.Equal(t, beforeB, chainDigest(t, a, chainB),
		"chain B's rows/hash/checkpoint digest must be byte-identical after A's rollback")

	// Chain-scoped reads never mix chains.
	for _, e := range eventsA {
		assert.Equal(t, chainA, e.ChainID)
	}
	for _, e := range eventsB {
		assert.Equal(t, chainB, e.ChainID)
	}

	// The unscoped port method DOES see both chains. That is exactly why the
	// scoped API exists, and why the unscoped one is only safe single-chain.
	unscoped, err := a.GetEventsByBlockRange(ctx, height, height)
	require.NoError(t, err)
	assert.Len(t, unscoped, 2, "the unscoped port method mixes chains — use the chain-scoped API")

	// Chain-scoped block and checkpoint reads.
	blockA, err := a.GetBlockForChain(ctx, chainA, height)
	require.NoError(t, err)
	require.NotNil(t, blockA)
	assert.Equal(t, blockHashA.Hex(), blockA.Hash.Hex())

	blockB, err := a.GetBlockForChain(ctx, chainB, height)
	require.NoError(t, err)
	require.NotNil(t, blockB)
	assert.Equal(t, blockHashB.Hex(), blockB.Hash.Hex())

	cpBlockA, cpHashA, err := a.GetCheckpointForChain(ctx, chainA)
	require.NoError(t, err)
	assert.Equal(t, height, cpBlockA)
	assert.Equal(t, blockHashA.Hex(), cpHashA)

	cpBlockB, cpHashB, err := a.GetCheckpointForChain(ctx, chainB)
	require.NoError(t, err)
	assert.Equal(t, height, cpBlockB)
	assert.Equal(t, blockHashB.Hex(), cpHashB)
}

// TestPostgresChainScopedAccessorsRejectEmptyChain pins the guard: an empty
// chain key must be refused rather than silently matching every chain.
func TestPostgresChainScopedAccessorsRejectEmptyChain(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	ctx := context.Background()
	a := newAdapter(t, dsn)

	_, err := a.GetEventsByBlockRangeForChain(ctx, "", 0, 1)
	require.Error(t, err)
	_, err = a.GetBlockForChain(ctx, "", 1)
	require.Error(t, err)
	_, err = a.GetLatestBlockForChain(ctx, "")
	require.Error(t, err)
	_, err = a.GetAllBlocksForChain(ctx, "")
	require.Error(t, err)
	_, err = a.MarkEventsAsReorgedForChain(ctx, "", 0, 1)
	require.Error(t, err)
	_, _, err = a.GetCheckpointForChain(ctx, "")
	require.Error(t, err)
}
