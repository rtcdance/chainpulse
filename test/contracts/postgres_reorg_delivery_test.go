//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostgresReorgCommitsCheckpointAndNotificationsTogether(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	a := newAdapter(t, dsn)
	ctx := context.Background()
	const chain = "reorg-delivery"
	_, err = db.Exec(`DELETE FROM reorg_jobs WHERE chain_id=$1;`, chain)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM reorg_generations WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO indexing_state(chain_id,last_indexed_block,last_indexed_hash) VALUES($1,12,'old') ON CONFLICT(chain_id) DO UPDATE SET last_indexed_block=12,last_indexed_hash='old'`, chain)
	require.NoError(t, err)
	_, err = a.CommitReorg(ctx, chain, 10, 12)
	require.NoError(t, err)
	var height, generation int64
	require.NoError(t, db.QueryRow(`SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1`, chain).Scan(&height))
	require.Equal(t, int64(9), height)
	require.NoError(t, db.QueryRow(`SELECT generation FROM reorg_generations WHERE chain_id=$1`, chain).Scan(&generation))
	require.Equal(t, int64(1), generation)
	var notices int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM reorg_outbox o JOIN reorg_jobs j ON o.job_id=j.id WHERE j.chain_id=$1`, chain).Scan(&notices))
	require.Equal(t, 2, notices)
	// A retry after the committed rewind must not bump generations or enqueue twice.
	_, err = a.CommitReorg(ctx, chain, 10, 12)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT generation FROM reorg_generations WHERE chain_id=$1`, chain).Scan(&generation))
	require.Equal(t, int64(1), generation)
}

func TestPostgresReorgFailureCannotPartiallyRevoke(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	a := newAdapter(t, dsn)
	ctx := context.Background()
	const chain = "reorg-delivery-atomic"
	_, err = db.Exec(`DELETE FROM reorg_jobs WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM reorg_generations WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	for _, scope := range []string{chain, chain + "-other"} {
		_, err = db.Exec(`DELETE FROM blockchain_events WHERE chain_id=$1`, scope)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO indexing_state(chain_id,last_indexed_block,last_indexed_hash) VALUES($1,12,'old') ON CONFLICT(chain_id) DO UPDATE SET last_indexed_block=12,last_indexed_hash='old'`, scope)
		require.NoError(t, err)
		for height := 9; height <= 12; height++ {
			_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,timestamp,status)
				VALUES($1,$2,$3,'old',$1,0,'contract','Transfer',1700000000,'pending')`, fmt.Sprintf("%s:%d", scope, height), scope, height)
			require.NoError(t, err)
		}
	}
	_, err = db.Exec(`CREATE OR REPLACE FUNCTION delivery_reject_notice() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN IF NEW.job_id LIKE 'reorg-delivery-atomic:%' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER delivery_reject_notice BEFORE INSERT ON reorg_outbox FOR EACH ROW EXECUTE FUNCTION delivery_reject_notice();`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Exec(`DROP TRIGGER IF EXISTS delivery_reject_notice ON reorg_outbox; DROP FUNCTION IF EXISTS delivery_reject_notice()`)
		require.NoError(t, err)
	})
	_, err = a.CommitReorg(ctx, chain, 10, 12)
	require.ErrorContains(t, err, "injected outbox failure")
	var height, count int
	require.NoError(t, db.QueryRow(`SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1`, chain).Scan(&height))
	require.Equal(t, 12, height)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM blockchain_events WHERE chain_id=$1 AND status='reorged'`, chain).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM reorg_generations WHERE chain_id=$1`, chain).Scan(&count))
	require.Zero(t, count)
	_, err = db.Exec(`DROP TRIGGER delivery_reject_notice ON reorg_outbox`)
	require.NoError(t, err)
	count64, err := a.CommitReorg(ctx, chain, 10, 12)
	require.NoError(t, err)
	require.Equal(t, int64(3), count64)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM blockchain_events WHERE chain_id=$1 AND status='pending'`, chain+"-other").Scan(&count))
	require.Equal(t, 4, count)
	jobs, err := a.PendingReorgJobs(ctx, chain, 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "revocation does not falsely complete canonical replay")
	require.NoError(t, a.CompleteReorgsThrough(ctx, chain, 12))
	jobs, err = a.PendingReorgJobs(ctx, chain, 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "an uncommitted replay cannot finish the job")
	_, err = db.Exec(`UPDATE indexing_state SET last_indexed_block=12,last_indexed_hash='new' WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	require.NoError(t, a.CompleteReorgsThrough(ctx, chain, 12))
	jobs, err = a.PendingReorgJobs(ctx, chain, 10)
	require.NoError(t, err)
	require.Empty(t, jobs)
}

func TestPostgresReorgRefusesFinalizedEffectsAndAdvancedCheckpoint(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	a := newAdapter(t, dsn)
	const chain = "reorg-delivery-finalized"
	_, err = db.Exec(`DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO indexing_state(chain_id,last_indexed_block,last_indexed_hash) VALUES($1,13,'old') ON CONFLICT(chain_id) DO UPDATE SET last_indexed_block=13`, chain)
	require.NoError(t, err)
	_, err = a.CommitReorg(context.Background(), chain, 10, 12)
	require.ErrorContains(t, err, "advanced")
	_, err = db.Exec(`UPDATE indexing_state SET last_indexed_block=12 WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,timestamp,status)
		VALUES($1,$1,10,'old','tx',0,'contract','Transfer',1700000000,'finalized')`, chain)
	require.NoError(t, err)
	_, err = a.CommitReorg(context.Background(), chain, 10, 12)
	require.ErrorContains(t, err, "finalized")
	var height int
	require.NoError(t, db.QueryRow(`SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1`, chain).Scan(&height))
	require.Equal(t, 12, height)
}
