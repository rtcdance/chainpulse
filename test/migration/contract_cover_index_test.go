//go:build migration

package migration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
)

func TestContractCoverIndexUpgradeRollbackPreservesAuthority(t *testing.T) {
	t.Parallel()
	base := os.Getenv("MIGRATION_TEST_DSN")
	require.NotEmpty(t, base)
	name := "chainpulse_contract_cover_" + randomSuffix()
	createDB(t, base, name)
	defer dropDB(t, base, name)
	dsn := replaceDBName(base, name)
	m, err := migrate.New("file://"+filepath.Join(findProjectRoot(), "migrations"), dsn)
	require.NoError(t, err)
	defer func() { a, b := m.Close(); require.NoError(t, a); require.NoError(t, b) }()
	require.NoError(t, m.Migrate(20))
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,event_data,timestamp,status)
VALUES('cover-a','1',1,'0xa','0xb',0,'0xAB','Transfer','{"raw":"AAE="}',1700000000,'confirmed'),
('cover-b','2',1,'0xc','0xd',0,'0xab','Swap','{}',1700000000,'reorged')`)
	require.NoError(t, err)
	snapshot := func() string {
		t.Helper()
		var value string
		require.NoError(t, db.QueryRow(`SELECT jsonb_build_object(
'events',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM blockchain_events e),
'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY chain_id) FROM canonical_query_revisions r))::text`).Scan(&value))
		return value
	}
	before := snapshot()
	for range 2 {
		require.NoError(t, m.Migrate(21))
		var valid bool
		require.NoError(t, db.QueryRow(`SELECT indisvalid AND indisready FROM pg_index WHERE indexrelid='idx_events_canonical_contract_cover'::regclass`).Scan(&valid))
		require.True(t, valid)
		require.Equal(t, before, snapshot(), "adding an index cannot alter event data or revision authority")
		var total int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM blockchain_events WHERE chain_id='1' AND lower(contract_address)='0xab' AND status<>'reorged' AND event_state->>'canonicality'='canonical'`).Scan(&total))
		require.Equal(t, 1, total)
		require.NoError(t, m.Steps(-1))
		require.Equal(t, before, snapshot(), "rollback retains data, epochs and revision numbers")
		var absent sql.NullString
		require.NoError(t, db.QueryRow(`SELECT to_regclass('idx_events_canonical_contract_cover')::text`).Scan(&absent))
		require.False(t, absent.Valid)
	}
}
