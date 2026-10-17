//go:build integration

package acceptance

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildNonAtomicProbe compiles the deliberately non-atomic control workload.
func buildNonAtomicProbe(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("../..")
	require.NoError(t, err)

	bin := filepath.Join(t.TempDir(), "nonatomic-probe")
	cmd := exec.Command("go", "build", "-o", bin, "./test/acceptance/tools/nonatomic-probe")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)
	return bin
}

// TestCrashInsideCommitNegativeControl is the control for
// TestCrashInsideCommitLeavesNothingBehind.
//
// It runs the same kill in the same window against the same workload with ONE
// change: the events and the checkpoint advance commit in two separate
// transactions. If the atomic test's assertion had no discriminating power, this
// control would look identical. Instead the control must observe the
// half-commit — events durable while the checkpoint still points at the previous
// block — which is precisely the state the atomic committer makes impossible.
//
// Without this control, "a kill inside the commit leaves nothing behind" would
// be indistinguishable from "the kill did not land inside the commit".
func TestCrashInsideCommitNegativeControl(t *testing.T) {
	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	bin := buildNonAtomicProbe(t)

	resetState(t, dsn)
	resetIndexingState(t, dsn, killChainID)

	w := startWorkload(t, bin, dsn,
		"CHAINPULSE_GRACEFUL_CHAIN_ID="+killChainID,
		fmt.Sprintf("CHAINPULSE_GRACEFUL_BATCH_SIZE=%d", killBatch),
		fmt.Sprintf("CHAINPULSE_GRACEFUL_TX_DELAY_MS=%d", killTxDelayMS),
	)

	// Block 0 is fully committed (both transactions). Wait for block 1's EVENTS
	// transaction to commit: at that point its events are durable and its
	// checkpoint is not, and the process is about to sit in betweenTxDelay before
	// writing the checkpoint. Killing now is killing in the split.
	w.waitFor(t, "committed block ", 1)
	w.waitFor(t, "events committed ", 1)
	w.kill(t)

	s := readDurableState(t, dsn)
	require.True(t, s.hasCheckpoint, "block 0 committed, so a checkpoint must exist")

	// The control's whole point: the split commit leaves block 1's events behind
	// while the checkpoint still says 0. Asserted as "not a consistent prefix"
	// rather than pinning the exact counts, so the control does not depend on
	// how far the process got.
	consistent := s.events == int64(s.checkpoint+1)*killBatch &&
		s.blockCount == int64(s.checkpoint+1)
	assert.False(t, consistent,
		"the non-atomic control produced a consistent prefix (%d events, checkpoint %d); "+
			"the atomic test's assertion would then have no discriminating power",
		s.events, s.checkpoint)

	assert.Greater(t, s.events, int64(s.checkpoint+1)*killBatch,
		"the control must leave events beyond its checkpoint — that IS the half-commit")
	assert.True(t, s.maxBlock.Valid && s.maxBlock.Int64 > int64(s.checkpoint),
		"the control must leave an event whose block is past the checkpoint, got max %v vs checkpoint %d",
		s.maxBlock, s.checkpoint)

	t.Logf("control observed the half-commit: %d durable event(s), checkpoint %d, max block %v",
		s.events, s.checkpoint, s.maxBlock)
}
