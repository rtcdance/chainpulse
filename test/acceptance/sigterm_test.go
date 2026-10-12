//go:build integration

// Package acceptance holds cross-cutting acceptance tests for the delivery
// plan. Unlike test/contracts (which pins per-adapter behaviour), these drive a
// real process and assert end-to-end properties.
package acceptance

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	_ "github.com/lib/pq" // postgres driver

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gracefulChainID = "t05-sigterm"
	gracefulBatch   = 3
	// The plan's F24 acceptance says "10 SIGTERM runs". Kept at 10: each run is
	// a few hundred milliseconds.
	gracefulIterations = 10
)

func acceptanceDSN() string {
	if os.Getenv("CHAINPULSE_RUN_POSTGRES_INTEGRATION") != "1" {
		return ""
	}
	return os.Getenv("CHAINPULSE_TEST_DATABASE_URL")
}

func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func resetState(t *testing.T, dsn string) {
	t.Helper()
	db := openDB(t, dsn)
	_, err := db.Exec("TRUNCATE blockchain_events")
	require.NoError(t, err)
	_, err = db.Exec("TRUNCATE indexing_state")
	require.NoError(t, err)
}

// readCheckpoint returns (blockNumber, true) when a checkpoint row exists.
func readCheckpoint(t *testing.T, dsn string) (uint64, bool) {
	t.Helper()
	db := openDB(t, dsn)

	var block uint64
	err := db.QueryRow(
		"SELECT last_indexed_block FROM indexing_state WHERE chain_id = $1", gracefulChainID,
	).Scan(&block)
	if err == sql.ErrNoRows {
		return 0, false
	}
	require.NoError(t, err)
	return block, true
}

func countEvents(t *testing.T, dsn string) int64 {
	t.Helper()
	db := openDB(t, dsn)

	var n int64
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM blockchain_events WHERE chain_id = $1", gracefulChainID,
	).Scan(&n))
	return n
}

// buildWorkload compiles the SIGTERM workload once per test run.
func buildWorkload(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("../..")
	require.NoError(t, err)

	bin := filepath.Join(t.TempDir(), "graceful-workload")
	cmd := exec.Command("go", "build", "-o", bin, "./test/acceptance/tools/graceful")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)
	return bin
}

// TestSigtermGracefulShutdown is the T05 / F24 acceptance test.
//
// For each run: start the workload, let it commit a few batches, send SIGTERM
// (not SIGKILL — Docker stop and rolling deploys use SIGTERM), and assert:
//
//   - exit code 0 (it shut down cleanly rather than being killed);
//   - the checkpoint still lands on a batch boundary;
//   - the number of committed events equals the checkpoint exactly, i.e. there
//     is no half-commit and no gap.
func TestSigtermGracefulShutdown(t *testing.T) {
	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	bin := buildWorkload(t)

	for i := 0; i < gracefulIterations; i++ {
		t.Run(fmt.Sprintf("run_%02d", i), func(t *testing.T) {
			resetState(t, dsn)

			cmd := exec.Command(bin)
			cmd.Env = append(os.Environ(),
				"CHAINPULSE_TEST_DATABASE_URL="+dsn,
				"CHAINPULSE_GRACEFUL_CHAIN_ID="+gracefulChainID,
				fmt.Sprintf("CHAINPULSE_GRACEFUL_BATCH_SIZE=%d", gracefulBatch),
			)
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			cmd.Stderr = os.Stderr
			require.NoError(t, cmd.Start())

			// Let it commit a few batches so there is something in flight.
			sc := bufio.NewScanner(stdout)
			committed := 0
			for sc.Scan() {
				if strings.HasPrefix(sc.Text(), "committed block ") {
					committed++
					if committed >= 3 {
						break
					}
				}
			}
			require.GreaterOrEqual(t, committed, 3, "workload must commit before we signal it")

			require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
			require.NoError(t, cmd.Wait(), "process must exit 0 on SIGTERM")

			cp, ok := readCheckpoint(t, dsn)
			require.True(t, ok, "a checkpoint must exist after shutdown")

			// Every batch writes exactly gracefulBatch events and advances the
			// checkpoint by one, so committed blocks 0..cp give (cp+1)*batch
			// events. Any other count means a half-commit or a gap.
			assert.Equal(t, int64((cp+1)*gracefulBatch), countEvents(t, dsn),
				"event count must match the checkpoint exactly (no half-commit, no gap)")
		})
	}
}
