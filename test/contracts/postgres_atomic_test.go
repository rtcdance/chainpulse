//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq" // postgres driver

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// truncateIndexingTables empties the events and checkpoint tables so a test
// asserting "nothing was written" is not confused by earlier rows.
func truncateIndexingTables(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("TRUNCATE blockchain_events")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE indexing_state")
	require.NoError(t, err)
}

// readCheckpoint returns the persisted checkpoint for a chain (0, "" if unset).
func readCheckpoint(t *testing.T, dsn, chainID string) (uint64, string) {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var block uint64
	var hash sql.NullString
	err = db.QueryRow(
		"SELECT last_indexed_block, last_indexed_hash FROM indexing_state WHERE chain_id = $1",
		chainID,
	).Scan(&block, &hash)
	if err == sql.ErrNoRows {
		return 0, ""
	}
	require.NoError(t, err)
	return block, hash.String
}

func countChainEvents(t *testing.T, dsn, chainID string) int {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var n int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM blockchain_events WHERE chain_id = $1", chainID,
	).Scan(&n))
	return n
}

func newAdapter(t *testing.T, dsn string) *database.PostgreSQLPluginAdapter {
	t.Helper()
	ctx := context.Background()
	a := database.NewPostgreSQLPluginAdapter(
		testhelpers.NewTestLogger(),
		testhelpers.NewTestMetricsCollector(),
	)
	require.NoError(t, a.Initialize(ctx, core.Config{DatabaseURL: dsn}), "Initialize")
	require.NoError(t, a.Start(ctx), "Start")
	t.Cleanup(func() { _ = a.Stop(context.Background()) })
	return a
}

func txTestEvent(chainID string, blockNumber uint64) *blockchain.BlockchainEvent {
	e := &blockchain.BlockchainEvent{
		ChainID:         chainID,
		BlockNumber:     blockNumber,
		BlockHash:       common.HexToHash("0xfeed0000000000000000000000000000000000000000000000000000000000f1"),
		TransactionHash: common.HexToHash("0xfeed0000000000000000000000000000000000000000000000000000000000f2"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000bb"),
		EventName:       "Transfer",
		BlockTimestamp:  1700000300,
	}
	e.EventHash = core.ComputeEventHash(e)
	e.ID = e.EventHash
	return e
}

// TestPostgresAtomicCommit is the T05 acceptance test for the happy path: an
// event batch and the checkpoint advance land together.
func TestPostgresAtomicCommit(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-commit"

	tx, err := a.BeginTx(ctx)
	require.NoError(t, err, "BeginTx")

	require.NoError(t, tx.StoreEvent(ctx, txTestEvent(chainID, 42)))
	require.NoError(t, tx.SaveCheckpoint(ctx, chainID, 42, "0xfeed"))
	require.NoError(t, tx.Commit(ctx), "Commit")

	assert.Equal(t, 1, countChainEvents(t, dsn, chainID), "the event must be committed")
	block, hash := readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(42), block, "the checkpoint must be committed")
	assert.Equal(t, "0xfeed", hash)
}

// TestPostgresAtomicRollback is the T05 acceptance test for the failure path:
// rolling back must leave NEITHER the events nor the checkpoint. This is the
// "no half-commit" property — the crash window between "events written" and
// "checkpoint advanced" must not exist.
func TestPostgresAtomicRollback(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-rollback"

	tx, err := a.BeginTx(ctx)
	require.NoError(t, err)

	require.NoError(t, tx.StoreEvent(ctx, txTestEvent(chainID, 99)))
	require.NoError(t, tx.SaveCheckpoint(ctx, chainID, 99, "0xdead"))
	require.NoError(t, tx.Rollback(ctx), "Rollback")

	assert.Equal(t, 0, countChainEvents(t, dsn, chainID), "rolled-back events must not persist")
	block, hash := readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(0), block, "rolled-back checkpoint must not persist")
	assert.Equal(t, "", hash)
}

// TestPostgresRollbackIsIdempotent covers the `defer tx.Rollback(ctx)` pattern:
// rolling back after a successful commit must be a harmless no-op, not a second
// error that would mask the real result.
func TestPostgresRollbackIsIdempotent(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-idem"

	tx, err := a.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.StoreEvent(ctx, txTestEvent(chainID, 7)))
	require.NoError(t, tx.SaveCheckpoint(ctx, chainID, 7, "0xbeef"))
	require.NoError(t, tx.Commit(ctx))

	require.NoError(t, tx.Rollback(ctx), "Rollback after Commit must be a no-op")
	assert.Equal(t, 1, countChainEvents(t, dsn, chainID), "committed data must survive the deferred rollback")
	block, _ := readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(7), block)

	// A second Commit must be rejected rather than silently re-committing.
	require.Error(t, tx.Commit(ctx), "Commit after Commit must fail")
}
