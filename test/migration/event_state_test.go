//go:build migration

package migration

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

func TestEventStateUpgradeRollbackPreservesLegacyEvidence(t *testing.T) {
	t.Parallel()
	base := os.Getenv("MIGRATION_TEST_DSN")
	require.NotEmpty(t, base, "migration lane requires an explicit isolated PostgreSQL DSN")
	name := "chainpulse_state_migration_" + randomSuffix()
	createDB(t, base, name)
	defer dropDB(t, base, name)
	dsn := replaceDBName(base, name)
	m, err := migrate.New("file://"+filepath.Join(findProjectRoot(), "migrations"), dsn)
	require.NoError(t, err)
	defer func() { sourceErr, dbErr := m.Close(); require.NoError(t, sourceErr); require.NoError(t, dbErr) }()
	require.NoError(t, m.Migrate(15))
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	for i, status := range []string{"pending", "confirmed", "finalized", "failed", "reorged"} {
		_, err = db.Exec(`INSERT INTO blockchain_events(id,chain_id,block_number,block_hash,transaction_hash,log_index,contract_address,event_name,event_data,timestamp,status) VALUES($1,'1',1,'0x1','0x2',$2,'0x3','Transfer','{"raw":"AAE="}',1700000000,$1)`, status, i)
		require.NoError(t, err)
	}
	legacy := func() string {
		var raw string
		require.NoError(t, db.QueryRow(`SELECT jsonb_agg(jsonb_build_array(id,status,event_data,block_hash,transaction_hash,log_index) ORDER BY id)::text FROM blockchain_events`).Scan(&raw))
		return raw
	}
	before := legacy()
	for cycle := 0; cycle < 2; cycle++ {
		require.NoError(t, m.Steps(1))
		require.Equal(t, before, legacy())
		rows, err := db.Query(`SELECT status,event_state FROM blockchain_events ORDER BY id`)
		require.NoError(t, err)
		count := 0
		for rows.Next() {
			var status string
			var raw []byte
			var state map[string]any
			require.NoError(t, rows.Scan(&status, &raw))
			require.NoError(t, json.Unmarshal(raw, &state))
			finality := state["finality"].(map[string]any)
			require.Equal(t, "unknown", finality["state"], "historical finalized cannot manufacture evidence")
			if status == "reorged" {
				require.Equal(t, "orphaned", state["canonicality"])
			} else {
				require.Equal(t, "canonical", state["canonicality"])
			}
			count++
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		require.Equal(t, 5, count)
		require.NoError(t, m.Steps(-1))
		require.Equal(t, before, legacy(), "downgrade must preserve old application columns and raw identity")
	}
}
