//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	_ "github.com/lib/pq" // postgres driver

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
)

func resetReorg(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("TRUNCATE reorg_outbox, reorg_jobs")
	require.NoError(t, err)
}

func testReorgJob(id string) database.ReorgJob {
	return database.ReorgJob{
		ID:         id,
		ChainID:    "ethereum",
		Generation: 3,
		FromBlock:  100,
		ToBlock:    110,
		Status:     database.ReorgJobPending,
	}
}

func testNotifications(jobID string) []database.ReorgNotification {
	return []database.ReorgNotification{
		{JobID: jobID, Topic: "reorg.detected", Payload: []byte(`{"from":100,"to":110}`)},
		{JobID: jobID, Topic: "reorg.rollback", Payload: []byte(`{"chain":"ethereum"}`)},
	}
}

// TestPostgresReorgOutboxIsAtomic is the T06 / F07 acceptance test: the repair
// job and the notifications it owes are recorded together, and an interrupted
// reorg is discoverable afterwards.
func TestPostgresReorgOutboxIsAtomic(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetReorg(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	job := testReorgJob("reorg-1")

	require.NoError(t, a.EnqueueReorgJob(ctx, job, testNotifications(job.ID)))

	// The job is discoverable, so a restart can resume it.
	jobs, err := a.PendingReorgJobs(ctx, "ethereum", 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, job.ID, jobs[0].ID)
	assert.Equal(t, job.FromBlock, jobs[0].FromBlock)
	assert.Equal(t, job.Generation, jobs[0].Generation)

	// Both notifications are owed.
	pending, err := a.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.Equal(t, job.ID, pending[0].JobID)
	assert.Equal(t, job.Generation, pending[0].Generation, "the generation must travel with the notification")
	assert.Equal(t, "reorg.detected", pending[0].Topic)

	// Draining one leaves the other.
	require.NoError(t, a.MarkReorgNotificationPublished(ctx, pending[0].ID))
	pending, err = a.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	// Finishing the job removes it from the resume set.
	require.NoError(t, a.CompleteReorgJob(ctx, job.ID))
	jobs, err = a.PendingReorgJobs(ctx, "ethereum", 10)
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

// TestPostgresReorgOutboxEnqueueIsIdempotent covers a retry of the enqueue
// itself (the caller crashed after committing but before recording success).
func TestPostgresReorgOutboxEnqueueIsIdempotent(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetReorg(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	job := testReorgJob("reorg-dup")

	require.NoError(t, a.EnqueueReorgJob(ctx, job, testNotifications(job.ID)))
	require.NoError(t, a.EnqueueReorgJob(ctx, job, testNotifications(job.ID)))

	jobs, err := a.PendingReorgJobs(ctx, "ethereum", 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "the job must not be duplicated")

	pending, err := a.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 2, "notifications must not be duplicated")
}

// TestPostgresReorgOutboxSurvivesRestart is the "crash during repair" case: the
// process dies after the rollback committed but before anything was published.
// A fresh process must find both the job and its notifications.
func TestPostgresReorgOutboxSurvivesRestart(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetReorg(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First process: records the job, then dies (nothing published).
	first := newAdapter(t, dsn)
	job := testReorgJob("reorg-crash")
	require.NoError(t, first.EnqueueReorgJob(ctx, job, testNotifications(job.ID)))
	require.NoError(t, first.Stop(ctx))

	// Second process: finds the unfinished work and drains it.
	second := newAdapter(t, dsn)

	jobs, err := second.PendingReorgJobs(ctx, "ethereum", 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1, "the interrupted job must be resumable")
	assert.Equal(t, job.ID, jobs[0].ID)

	pending, err := second.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2, "the notifications must still be owed")

	for _, n := range pending {
		require.NoError(t, second.MarkReorgNotificationPublished(ctx, n.ID))
	}
	require.NoError(t, second.CompleteReorgJob(ctx, job.ID))

	pending, err = second.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, pending)

	jobs, err = second.PendingReorgJobs(ctx, "ethereum", 10)
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

// TestPostgresRewindCheckpoint is the "回退 checkpoint" half of a reorg repair:
// the checkpoint must move BACK to the block before the repaired range, or
// re-indexing would silently skip it.
func TestPostgresRewindCheckpoint(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	truncateIndexingTables(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	chainID := "t06-rewind"

	// A known block at the rewind target, so the hash can be restored.
	require.NoError(t, a.WriteBlock(ctx, chainID, &blockchain.Block{
		Number: 99,
		Hash:   common.HexToHash("0x99"),
	}))

	// Advance the checkpoint past the range we are about to repair.
	tx, err := a.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.SaveCheckpoint(ctx, chainID, 110, "0x110"))
	require.NoError(t, tx.Commit(ctx))

	block, _ := readCheckpoint(t, dsn, chainID)
	require.Equal(t, uint64(110), block)

	require.NoError(t, a.RewindCheckpoint(ctx, chainID, 99))

	block, hash := readCheckpoint(t, dsn, chainID)
	assert.Equal(t, uint64(99), block, "the checkpoint must move back")
	assert.Equal(t, common.HexToHash("0x99").Hex(), hash, "the hash must come from the known block")
}

// TestPostgresReorgOutboxMarkIsIdempotent covers the at-least-once window: the
// drainer published, then crashed before marking. Re-marking must not error.
func TestPostgresReorgOutboxMarkIsIdempotent(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	resetReorg(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := newAdapter(t, dsn)
	job := testReorgJob("reorg-at-least-once")
	require.NoError(t, a.EnqueueReorgJob(ctx, job, testNotifications(job.ID)))

	pending, err := a.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)

	require.NoError(t, a.MarkReorgNotificationPublished(ctx, pending[0].ID))
	require.NoError(t, a.MarkReorgNotificationPublished(ctx, pending[0].ID), "re-marking must be a no-op")

	pending, err = a.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	// A failed attempt is visible rather than silently retried forever.
	require.NoError(t, a.RecordReorgNotificationFailure(ctx, pending[0].ID))
	pending, err = a.PendingReorgNotifications(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, 1, pending[0].Attempts)
}
