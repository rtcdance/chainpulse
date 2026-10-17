//go:build migration

package migration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestQueryRevisionsTrackCommittedMutationsAndRollback(t *testing.T) {
	t.Parallel()
	base := os.Getenv("MIGRATION_TEST_DSN")
	require.NotEmpty(t, base)
	name := "chainpulse_query_revision_" + randomSuffix()
	createDB(t, base, name)
	defer dropDB(t, base, name)
	dsn := replaceDBName(base, name)
	m, err := migrate.New("file://"+filepath.Join(findProjectRoot(), "migrations"), dsn)
	require.NoError(t, err)
	defer func() { a, b := m.Close(); require.NoError(t, a); require.NoError(t, b) }()
	require.NoError(t, m.Migrate(19))
	store := database.NewPostgreSQLPluginAdapter(testhelpers.NewTestLogger(), core.NewDefaultMetricsCollector()).WithChainScope("1")
	require.NoError(t, store.Initialize(context.Background(), core.Config{DatabaseURL: dsn}))
	require.NoError(t, store.Start(context.Background()))
	defer func() { require.NoError(t, store.Stop(context.Background())) }()
	require.ErrorContains(t, store.ValidateCanonicalAggregateSchema(context.Background()), "schema20")
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,event_data,timestamp,status,created_at) VALUES('rev-a','1',1,'0xa','0xb',0,'0xc','Transfer','{"raw":"AAE="}',1700000000,'confirmed','2023-11-14 22:13:20')`)
	require.NoError(t, err)
	snapshot := func() string {
		t.Helper()
		var value string
		require.NoError(t, db.QueryRow(`SELECT row_to_json(e)::text FROM blockchain_events e WHERE id='rev-a'`).Scan(&value))
		return value
	}
	before := snapshot()
	require.NoError(t, m.Migrate(20))
	require.NoError(t, store.ValidateCanonicalAggregateSchema(context.Background()))
	_, err = db.Exec(`ALTER TABLE blockchain_events DISABLE TRIGGER bump_canonical_query_revision`)
	require.NoError(t, err)
	require.ErrorContains(t, store.ValidateCanonicalAggregateSchema(context.Background()), "mutation triggers")
	_, err = db.Exec(`ALTER TABLE blockchain_events ENABLE TRIGGER bump_canonical_query_revision`)
	require.NoError(t, err)
	require.NoError(t, store.ValidateCanonicalAggregateSchema(context.Background()))
	require.Equal(t, before, snapshot())
	version := func(chain string) (int64, string) {
		t.Helper()
		var revision int64
		var epoch string
		require.NoError(t, db.QueryRow(`SELECT revision,epoch FROM canonical_query_revisions WHERE chain_id=$1`, chain).Scan(&revision, &epoch))
		return revision, epoch
	}
	r0, epoch := version("1")
	require.Positive(t, r0)
	require.Len(t, epoch, 32)
	_, err = db.Exec(`INSERT INTO blockchain_events SELECT * FROM blockchain_events WHERE id='rev-a' ON CONFLICT DO NOTHING`)
	require.NoError(t, err)
	r1, sameEpoch := version("1")
	require.Equal(t, r0, r1, "duplicate writes must not change the data version")
	require.Equal(t, epoch, sameEpoch)
	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`UPDATE blockchain_events SET event_name='RolledBack' WHERE id='rev-a'`)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	r1, _ = version("1")
	require.Equal(t, r0, r1, "revision and data roll back together")
	_, err = db.Exec(`UPDATE blockchain_events SET chain_id='2' WHERE id='rev-a'`)
	require.NoError(t, err)
	r1, _ = version("1")
	r2, _ := version("2")
	require.Greater(t, r1, r0)
	require.Positive(t, r2, "both scopes change for a cross-chain mutation")
	_, err = db.Exec(`UPDATE blockchain_events SET status='reorged' WHERE id='rev-a'`)
	require.NoError(t, err)
	r3, _ := version("2")
	require.Greater(t, r3, r2)
	_, err = db.Exec(`DELETE FROM blockchain_events WHERE id='rev-a'`)
	require.NoError(t, err)
	r4, _ := version("2")
	require.Greater(t, r4, r3)
	_, err = db.Exec(`TRUNCATE blockchain_events`)
	require.NoError(t, err)
	r5, _ := version("2")
	require.Greater(t, r5, r4)
	// Keep a complete physical row present while removing/reapplying the schema.
	_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,event_data,timestamp,status,created_at) VALUES('rev-a','1',1,'0xa','0xb',0,'0xc','Transfer','{"raw":"AAE="}',1700000000,'confirmed','2023-11-14 22:13:20')`)
	require.NoError(t, err)
	require.Equal(t, before, snapshot())
	require.NoError(t, m.Steps(-1))
	require.Equal(t, before, snapshot(), "down20 preserves every event column")
	var absent sql.NullString
	require.NoError(t, db.QueryRow(`SELECT to_regclass('canonical_query_revisions')::text`).Scan(&absent))
	require.False(t, absent.Valid)
	require.NoError(t, m.Migrate(20))
	_, newEpoch := version("1")
	require.NotEqual(t, epoch, newEpoch, "reinstall cannot reuse stale Redis namespaces")
	require.Equal(t, before, snapshot())
}
