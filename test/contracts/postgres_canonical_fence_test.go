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

func TestPostgresExpiredLeaseCannotBeginTransaction(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	a := newAdapter(t, dsn)
	ctx := context.Background()
	const chain = "canonical-expired-fence"
	_, err = db.Exec("DELETE FROM chain_leases WHERE chain_id=$1", chain)
	require.NoError(t, err)
	_, err = a.AcquireLease(ctx, chain, "owner", time.Minute)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE chain_leases SET expires_at=clock_timestamp()-interval '1 second' WHERE chain_id=$1", chain)
	require.NoError(t, err)
	_, err = a.BeginFencedTx(ctx, chain, "owner")
	require.ErrorIs(t, err, database.ErrLeaseLost)
}

func TestPostgresStaleReorgDecisionCannotRevokeNewBranch(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	a := newAdapter(t, dsn)
	ctx := context.Background()
	const chain = "canonical-reorg-cas"
	_, err = db.Exec(`INSERT INTO indexing_state(chain_id,last_indexed_block,last_indexed_hash) VALUES($1,12,'replacement')
	 ON CONFLICT(chain_id) DO UPDATE SET last_indexed_block=12,last_indexed_hash='replacement'`, chain)
	require.NoError(t, err)
	_, err = a.CommitReorgAtCheckpoint(ctx, chain, 10, 12, "obsolete")
	require.ErrorContains(t, err, "checkpoint changed")
	var height int
	var hash string
	require.NoError(t, db.QueryRow("SELECT last_indexed_block,last_indexed_hash FROM indexing_state WHERE chain_id=$1", chain).Scan(&height, &hash))
	require.Equal(t, 12, height)
	require.Equal(t, "replacement", hash)
	var jobs int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM reorg_jobs WHERE chain_id=$1", chain).Scan(&jobs))
	require.Zero(t, jobs)
}

func TestPostgresOrphanBlockAuditIsTerminalAndIdempotent(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	a := newAdapter(t, dsn)
	ctx := context.Background()
	const chain = "canonical-orphan-audit"
	_, err = db.Exec("DELETE FROM dlq_events WHERE chain_id=$1", chain)
	require.NoError(t, err)
	for range 100 {
		require.NoError(t, a.RecordOrphanedBlock(ctx, chain, "0x01", []byte(`{"schema":"chainpulse.block.v1","events":[]}`)))
	}
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM dlq_events WHERE chain_id=$1 AND status='orphaned'", chain).Scan(&count))
	require.Equal(t, 1, count)
}
