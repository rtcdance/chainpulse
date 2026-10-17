//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq" // postgres driver

	"github.com/rtcdance/chainpulse/pkg/plugins/database"
)

// resetLeases clears the lease table so lease tests start from "no owner".
func resetLeases(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("TRUNCATE chain_leases")
	require.NoError(t, err)
}

// TestPostgresLeaseTakeoverRejectsOldOwner is the T05 acceptance test for owner
// fencing: after a Kafka rebalance hands a chain to a new worker, the previous
// worker's commits must be rejected.
func TestPostgresLeaseTakeoverRejectsOldOwner(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetLeases(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-fencing"

	// Worker A takes the lease with a short TTL.
	genA, err := a.AcquireLease(ctx, chainID, "worker-A", 100*time.Millisecond)
	require.NoError(t, err, "first acquisition must succeed")
	assert.Equal(t, int64(1), genA)

	// A may commit while it owns the lease.
	txA, err := a.BeginFencedTx(ctx, chainID, "worker-A")
	require.NoError(t, err, "the owner may open a fenced transaction")
	require.NoError(t, txA.StoreEvent(ctx, txTestEvent(chainID, 1)))
	require.NoError(t, txA.SaveCheckpoint(ctx, chainID, 1, "0xaa"))
	require.NoError(t, txA.Commit(ctx))

	// A's lease expires; the chain is handed to worker B (rebalance).
	time.Sleep(150 * time.Millisecond)
	genB, err := a.AcquireLease(ctx, chainID, "worker-B", time.Minute)
	require.NoError(t, err, "an expired lease may be taken over")
	assert.Equal(t, int64(2), genB, "takeover must bump the generation")

	// The old owner must now be rejected.
	_, err = a.BeginFencedTx(ctx, chainID, "worker-A")
	require.ErrorIs(t, err, database.ErrLeaseLost, "the old owner must be fenced out")

	// The new owner works.
	txB, err := a.BeginFencedTx(ctx, chainID, "worker-B")
	require.NoError(t, err, "the new owner may open a fenced transaction")
	require.NoError(t, txB.StoreEvent(ctx, txTestEvent(chainID, 2)))
	require.NoError(t, txB.SaveCheckpoint(ctx, chainID, 2, "0xbb"))
	require.NoError(t, txB.Commit(ctx))
}

// TestPostgresLeaseCannotBeStolenWhileLive proves the takeover is conditional:
// a live lease is not stealable, so two workers cannot both believe they own
// the chain.
func TestPostgresLeaseCannotBeStolenWhileLive(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetLeases(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-fencing-live"

	_, err := a.AcquireLease(ctx, chainID, "worker-A", time.Minute)
	require.NoError(t, err)

	_, err = a.AcquireLease(ctx, chainID, "worker-B", time.Minute)
	require.ErrorIs(t, err, database.ErrLeaseLost, "a live lease must not be stolen")

	// Re-acquiring as the same owner is fine (idempotent renewal) and still
	// bumps the generation, so stale tokens stay stale.
	gen, err := a.AcquireLease(ctx, chainID, "worker-A", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(2), gen)

	owner, generation, ok, err := a.CurrentLease(ctx, chainID)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "worker-A", owner)
	assert.Equal(t, int64(2), generation)
}

// TestPostgresLeaseReleaseAllowsHandover covers the graceful path: a worker that
// shuts down releases the lease so the next one can start immediately instead of
// waiting for the TTL.
func TestPostgresLeaseReleaseAllowsHandover(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetLeases(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t05-fencing-release"

	_, err := a.AcquireLease(ctx, chainID, "worker-A", time.Minute)
	require.NoError(t, err)
	require.NoError(t, a.ReleaseLease(ctx, chainID, "worker-A"))

	// A non-owner cannot release someone else's lease.
	_, err = a.AcquireLease(ctx, chainID, "worker-B", time.Minute)
	require.NoError(t, err)
	require.NoError(t, a.ReleaseLease(ctx, chainID, "worker-A"), "releasing a lease we no longer own is a no-op")

	owner, _, ok, err := a.CurrentLease(ctx, chainID)
	require.NoError(t, err)
	assert.True(t, ok, "worker-B's lease must survive worker-A's release attempt")
	assert.Equal(t, "worker-B", owner)
}
