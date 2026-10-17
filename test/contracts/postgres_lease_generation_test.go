//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/stretchr/testify/require"
)

func TestPostgresReleasedLeaseRetainsMonotonicGeneration(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	const chain = "ownership-generation"
	_, err = db.Exec("DELETE FROM chain_leases WHERE chain_id=$1", chain)
	require.NoError(t, err)
	store := newAdapter(t, dsn)
	ctx := context.Background()
	first, err := store.AcquireLease(ctx, chain, "old-generation", time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.ReleaseLease(ctx, chain, "old-generation"))
	_, _, active, err := store.CurrentLease(ctx, chain)
	require.NoError(t, err)
	require.False(t, active)
	second, err := store.AcquireLease(ctx, chain, "new-generation", time.Minute)
	require.NoError(t, err)
	require.Greater(t, second, first, "release must not reset the fencing epoch")
	_, err = store.BeginFencedTx(ctx, chain, "old-generation")
	require.ErrorIs(t, err, database.ErrLeaseLost)
	// A delayed cleanup from the old generation cannot release the new owner.
	require.NoError(t, store.ReleaseLease(ctx, chain, "old-generation"))
	owner, generation, active, err := store.CurrentLease(ctx, chain)
	require.NoError(t, err)
	require.True(t, active)
	require.Equal(t, "new-generation", owner)
	require.Equal(t, second, generation)
}
