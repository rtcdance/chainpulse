//go:build migration

package migration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

func TestFinalityObservationUpgradeRollbackPreservesEventIdentity(t *testing.T) {
	t.Parallel()
	base := os.Getenv("MIGRATION_TEST_DSN")
	require.NotEmpty(t, base, "migration lane requires an isolated PostgreSQL DSN")
	name := "chainpulse_finality_migration_" + randomSuffix()
	createDB(t, base, name)
	defer dropDB(t, base, name)
	dsn := replaceDBName(base, name)
	m, err := migrate.New("file://"+filepath.Join(findProjectRoot(), "migrations"), dsn)
	require.NoError(t, err)
	defer func() { a, b := m.Close(); require.NoError(t, a); require.NoError(t, b) }()
	require.NoError(t, m.Migrate(16))
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,event_data,timestamp,status) VALUES('legacy','1',1,'0x1','0x2',0,'0x3','Transfer','{"raw":"AAE="}',1700000000,'confirmed')`)
	require.NoError(t, err)
	snapshot := func() string {
		t.Helper()
		var value string
		require.NoError(t, db.QueryRow(`SELECT row_to_json(e)::text FROM blockchain_events e WHERE id='legacy'`).Scan(&value))
		return value
	}
	before := snapshot()
	for cycle := 0; cycle < 2; cycle++ {
		require.NoError(t, m.Migrate(17))
		require.Equal(t, before, snapshot())
		_, err = db.Exec(`INSERT INTO chain_finality_observations(chain_id,tag,status) VALUES('1','finalized','known')`)
		require.Error(t, err, "known finality requires complete proof")
		_, err = db.Exec(`INSERT INTO chain_finality_observations(chain_id,tag,status) VALUES('1','safe','degraded')`)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO chain_finality_observations(chain_id,tag,status) VALUES('1','latest','degraded')`)
		require.Error(t, err, "latest is never a proof tag")
		require.NoError(t, m.Steps(-1))
		require.Equal(t, before, snapshot(), "down17 must preserve raw identity and all existing state")
		var functions int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_proc WHERE proname IN ('chainpulse_guard_finality_events','chainpulse_guard_finality_blocks','chainpulse_guard_finality_checkpoint','chainpulse_verified_finalized_height')`).Scan(&functions))
		require.Zero(t, functions, "down without verified finalized proof must remove its added functions")

	}
}

func TestFinalitySchemaRemovalRefusesVerifiedBoundaryLoss(t *testing.T) {
	t.Parallel()
	base := os.Getenv("MIGRATION_TEST_DSN")
	require.NotEmpty(t, base, "migration lane requires isolated PostgreSQL DSN")
	name := "chainpulse_finality_guard_" + randomSuffix()
	createDB(t, base, name)
	defer dropDB(t, base, name)
	dsn := replaceDBName(base, name)
	m, err := migrate.New("file://"+filepath.Join(findProjectRoot(), "migrations"), dsn)
	require.NoError(t, err)
	defer func() { a, b := m.Close(); require.NoError(t, a); require.NoError(t, b) }()
	require.NoError(t, m.Migrate(17))
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.Exec(`INSERT INTO chain_finality_observations(chain_id,tag,status,proof_at,head_number,head_hash,genesis_hash,generation)
 VALUES('1','finalized','known',CURRENT_TIMESTAMP,1,'0x1111111111111111111111111111111111111111111111111111111111111111','0x2222222222222222222222222222222222222222222222222222222222222222',0)`)
	require.NoError(t, err)
	require.ErrorContains(t, m.Steps(-1), "retain schema17")
	var proofs, triggers int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM chain_finality_observations WHERE tag='finalized'`).Scan(&proofs))
	require.Equal(t, 1, proofs, "refused schema removal must retain proof")
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_trigger WHERE tgname IN ('guard_finality_events','guard_finality_blocks','guard_finality_checkpoint')`).Scan(&triggers))
	require.Equal(t, 3, triggers, "old applications retain database boundary protection")
	// This disposable DB intentionally ends with migrate's dirty marker from the
	// refused operation. Never force a production ledger past this guard.
}
