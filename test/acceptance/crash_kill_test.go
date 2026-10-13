//go:build integration

package acceptance

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	killChainID = "t05-kill"
	killBatch   = 3
	// The plan's E05 acceptance asks for 10 kill runs per boundary.
	killIterations = 10
	// Held open inside the commit transaction so a signal can be delivered at a
	// known point. Without it a jittered timer usually lands between
	// transactions, and the matrix would prove nothing about the commit.
	killTxDelayMS = 400
)

// workload is a running graceful-workload process whose stdout we read.
type workload struct {
	cmd *exec.Cmd
	sc  *bufio.Scanner
}

func startWorkload(t *testing.T, bin, dsn string, env ...string) *workload {
	t.Helper()

	cmd := exec.Command(bin)
	cmd.Env = append(append(os.Environ(), env...), "CHAINPULSE_TEST_DATABASE_URL="+dsn)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	return &workload{cmd: cmd, sc: sc}
}

// waitFor reads until it has seen `count` lines starting with prefix.
func (w *workload) waitFor(t *testing.T, prefix string, count int) {
	t.Helper()

	seen := 0
	for w.sc.Scan() {
		if strings.HasPrefix(w.sc.Text(), prefix) {
			seen++
			if seen >= count {
				return
			}
		}
	}
	t.Fatalf("the workload exited before printing %d line(s) starting with %q", count, prefix)
}

// waitForBeginTx blocks until the workload has entered the transaction for a
// specific block. Between this line and the commit the process sleeps for
// killTxDelayMS, so a signal sent now is guaranteed to arrive inside the
// transaction with the events already written to it.
func (w *workload) waitForBeginTx(t *testing.T, block uint64) {
	t.Helper()

	want := fmt.Sprintf("begin tx %d", block)
	for w.sc.Scan() {
		if strings.HasPrefix(w.sc.Text(), want) {
			return
		}
	}
	t.Fatalf("the workload exited before %q", want)
}

func (w *workload) kill(t *testing.T) {
	t.Helper()
	require.NoError(t, w.cmd.Process.Kill(), "SIGKILL must be deliverable")
	_ = w.cmd.Wait() // a killed process is expected to exit non-zero
}

func (w *workload) interrupt(t *testing.T) {
	t.Helper()
	require.NoError(t, w.cmd.Process.Signal(os.Interrupt))
	require.NoError(t, w.cmd.Wait(), "the workload must exit 0 on SIGINT")
}

// durableState is what the database holds after a crash. A restart has to
// reason about it as a whole, so it is read as a whole.
type durableState struct {
	hasCheckpoint bool
	checkpoint    uint64
	events        int64
	blockCount    int64
	minBlock      sql.NullInt64
	maxBlock      sql.NullInt64
	fullBlocks    int64 // blocks holding exactly killBatch events
}

func readDurableState(t *testing.T, dsn string) durableState {
	t.Helper()
	db := openDB(t, dsn)

	var s durableState
	err := db.QueryRow(
		"SELECT last_indexed_block FROM indexing_state WHERE chain_id = $1", killChainID,
	).Scan(&s.checkpoint)
	switch err {
	case nil:
		s.hasCheckpoint = true
	case sql.ErrNoRows:
	default:
		t.Fatalf("read checkpoint: %v", err)
	}

	require.NoError(t, db.QueryRow(`
		SELECT count(*), count(DISTINCT block_number), min(block_number), max(block_number)
		FROM blockchain_events WHERE chain_id = $1`, killChainID,
	).Scan(&s.events, &s.blockCount, &s.minBlock, &s.maxBlock))

	require.NoError(t, db.QueryRow(`
		SELECT count(*) FROM (
			SELECT block_number FROM blockchain_events
			WHERE chain_id = $1
			GROUP BY block_number
			HAVING count(*) = $2
		) AS full_blocks`, killChainID, killBatch,
	).Scan(&s.fullBlocks))

	return s
}

// assertConsistentPrefix is the invariant E05 is about.
//
// A batch and its checkpoint advance commit in ONE transaction, so whatever
// moment the process dies at, the durable state must be a prefix of the batch
// stream: blocks 0..checkpoint present, each complete, nothing beyond. Anything
// else is a half-commit (events without the checkpoint advance, or the reverse)
// or a gap.
func assertConsistentPrefix(t *testing.T, dsn string, when string) {
	t.Helper()
	s := readDurableState(t, dsn)

	if !s.hasCheckpoint {
		assert.Zero(t, s.events,
			"%s: no checkpoint but %d event(s) are durable — events landed without their checkpoint", when, s.events)
		return
	}

	assert.Equal(t, int64(s.checkpoint+1)*killBatch, s.events,
		"%s: durable events must equal (checkpoint+1)*batch (no half-commit, no gap)", when)
	assert.Equal(t, int64(s.checkpoint+1), s.blockCount,
		"%s: blocks 0..%d must each be present exactly once", when, s.checkpoint)
	assert.True(t, s.minBlock.Valid && s.minBlock.Int64 == 0,
		"%s: the lowest durable block must be 0, got %v", when, s.minBlock)
	assert.True(t, s.maxBlock.Valid && s.maxBlock.Int64 == int64(s.checkpoint),
		"%s: the highest durable block must be the checkpoint %d, got %v", when, s.checkpoint, s.maxBlock)
	assert.Equal(t, int64(s.checkpoint+1), s.fullBlocks,
		"%s: every durable block must hold exactly %d events", when, killBatch)
}

func resetIndexingState(t *testing.T, dsn, chainID string) {
	t.Helper()
	db := openDB(t, dsn)
	_, err := db.Exec("DELETE FROM indexing_state WHERE chain_id = $1", chainID)
	require.NoError(t, err)
}

// TestCrashInsideCommitLeavesNothingBehind is the "before the commit" boundary.
//
// The process is killed while the commit transaction is open and the events are
// already written to it. Nothing from that transaction may survive: the events
// and the checkpoint advance are in the same transaction, so a surviving event
// with a stale checkpoint (or the reverse) is exactly the half-commit the plan
// forbids.
func TestCrashInsideCommitLeavesNothingBehind(t *testing.T) {
	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	bin := buildWorkload(t)

	for i := 0; i < killIterations; i++ {
		t.Run(fmt.Sprintf("kill_%02d", i), func(t *testing.T) {
			resetState(t, dsn)
			resetIndexingState(t, dsn, killChainID)

			w := startWorkload(t, bin, dsn,
				"CHAINPULSE_GRACEFUL_CHAIN_ID="+killChainID,
				fmt.Sprintf("CHAINPULSE_GRACEFUL_BATCH_SIZE=%d", killBatch),
				fmt.Sprintf("CHAINPULSE_GRACEFUL_TX_DELAY_MS=%d", killTxDelayMS),
			)

			// Block 0 has committed; we are now inside the transaction for
			// block 1, which will stay open for killTxDelayMS.
			w.waitFor(t, "committed block ", 1)
			w.waitForBeginTx(t, 1)
			w.kill(t)

			s := readDurableState(t, dsn)
			require.True(t, s.hasCheckpoint, "block 0 committed, so a checkpoint must exist")
			assert.Equal(t, uint64(0), s.checkpoint,
				"the in-flight transaction for block 1 must not have advanced the checkpoint")
			assert.Equal(t, int64(killBatch), s.events,
				"only block 0's events may be durable; the in-flight block 1 must be gone entirely")
			assert.True(t, s.maxBlock.Valid && s.maxBlock.Int64 == 0,
				"no durable event may belong to the in-flight block, got max block %v", s.maxBlock)
			assertConsistentPrefix(t, dsn, "after a kill inside the commit")
		})
	}
}

// TestCrashAfterCommitKeepsTheBlock is the "after the commit" boundary, and it
// also covers the restart: the workload resumes from the durable checkpoint, so
// a process that died after committing block N continues at N+1 rather than
// replaying from zero.
func TestCrashAfterCommitKeepsTheBlock(t *testing.T) {
	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	bin := buildWorkload(t)

	for i := 0; i < killIterations; i++ {
		t.Run(fmt.Sprintf("kill_%02d", i), func(t *testing.T) {
			resetState(t, dsn)
			resetIndexingState(t, dsn, killChainID)

			w := startWorkload(t, bin, dsn,
				"CHAINPULSE_GRACEFUL_CHAIN_ID="+killChainID,
				fmt.Sprintf("CHAINPULSE_GRACEFUL_BATCH_SIZE=%d", killBatch),
			)

			w.waitFor(t, "committed block ", 3)
			w.kill(t)

			assertConsistentPrefix(t, dsn, "after a kill following commits")
			crashed := readDurableState(t, dsn)
			require.GreaterOrEqual(t, crashed.checkpoint, uint64(2),
				"three commits were observed, so the checkpoint must be at least 2")

			resumed := startWorkload(t, bin, dsn,
				"CHAINPULSE_GRACEFUL_CHAIN_ID="+killChainID,
				fmt.Sprintf("CHAINPULSE_GRACEFUL_BATCH_SIZE=%d", killBatch),
				"CHAINPULSE_GRACEFUL_START_FROM_CHECKPOINT=1",
			)
			resumed.waitFor(t, "committed block ", 2)
			resumed.interrupt(t)

			after := readDurableState(t, dsn)
			assert.Greater(t, after.checkpoint, crashed.checkpoint,
				"the resumed run must continue past %d rather than replay from zero", crashed.checkpoint)
			assertConsistentPrefix(t, dsn, "after resuming from the checkpoint")
		})
	}
}
