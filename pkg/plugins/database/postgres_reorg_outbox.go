package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Reorg job statuses.
const (
	ReorgJobPending = "pending"
	ReorgJobDone    = "done"
)

// ReorgJob is a persistent repair job: which block range of which chain must be
// repaired, and how far ownership has advanced (generation). Recording it makes
// an interrupted reorg discoverable and resumable after a restart.
type ReorgJob struct {
	ID         string
	ChainID    string
	Generation int64
	FromBlock  uint64
	ToBlock    uint64
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ReorgNotification is one message owed for a reorg job. ChainID and Generation
// travel with it so a consumer can tell a stale notification (from a superseded
// generation) from a current one without a second lookup.
type ReorgNotification struct {
	ID         int64
	JobID      string
	ChainID    string
	Generation int64
	Topic      string
	Payload    []byte
	Attempts   int
}

// EnqueueReorgJob records the repair job AND its notifications in ONE
// transaction.
//
// This is the fix for F07: previously the rollback committed, then
// notifications were published best-effort and a failure was only logged, so a
// crash could leave a completed rollback with nobody told. Now the possible
// states are "job recorded, notification still pending" (which a drainer
// retries) or "both committed" — never "rollback done, notification lost".
//
// The write is idempotent per job id: re-enqueueing the same job does not
// duplicate the job row or its notifications.
func (a *PostgreSQLPluginAdapter) EnqueueReorgJob(ctx context.Context, job ReorgJob, notifications []ReorgNotification) error {
	if job.ID == "" || job.ChainID == "" {
		return fmt.Errorf("reorg job id and chain id are required")
	}
	if job.Status == "" {
		job.Status = ReorgJobPending
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		a.RecordError()
		return fmt.Errorf("database not initialized")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		a.RecordError()
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existing string
	err = tx.QueryRowContext(ctx, "SELECT id FROM reorg_jobs WHERE id = $1 FOR UPDATE", job.ID).Scan(&existing)
	switch {
	case err == nil:
		// Already recorded: nothing more to do (idempotent).
		return tx.Commit()
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("failed to check reorg job: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO reorg_jobs (id, chain_id, generation, from_block, to_block, status, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, now(), now())`,
		job.ID, job.ChainID, job.Generation, job.FromBlock, job.ToBlock, job.Status)
	if err != nil {
		return fmt.Errorf("failed to record reorg job: %w", err)
	}

	for i, n := range notifications {
		payload, err := outboxPayload(n.Payload)
		if err != nil {
			return fmt.Errorf("notification %d: %w", i, err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO reorg_outbox (job_id, topic, payload)
	VALUES ($1, $2, $3)`, job.ID, n.Topic, payload); err != nil {
			return fmt.Errorf("failed to enqueue notification %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reorg job: %w", err)
	}
	return nil
}

// outboxPayload guarantees the value stored in the jsonb column is valid JSON.
// Raw bytes that are already JSON are stored as-is; anything else is stored as a
// JSON string so the row is still written rather than rejected.
func outboxPayload(payload []byte) (string, error) {
	if len(payload) == 0 {
		return "{}", nil
	}
	if json.Valid(payload) {
		return string(payload), nil
	}
	encoded, err := json.Marshal(string(payload))
	if err != nil {
		return "", fmt.Errorf("payload is not encodable: %w", err)
	}
	return string(encoded), nil
}

// PendingReorgJobs returns jobs that are not done yet, oldest first. An empty
// chainID means "all chains". This is what a restarted process calls to resume.
func (a *PostgreSQLPluginAdapter) PendingReorgJobs(ctx context.Context, chainID string, limit int) ([]ReorgJob, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.QueryContext(ctx, `
SELECT id, chain_id, generation, from_block, to_block, status, created_at, updated_at
	FROM reorg_jobs
	WHERE status <> 'done' AND ($1 = '' OR chain_id = $1)
	ORDER BY created_at ASC, id ASC
	LIMIT $2`, chainID, limit)
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to query pending reorg jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var jobs []ReorgJob
	for rows.Next() {
		var j ReorgJob
		if err := rows.Scan(&j.ID, &j.ChainID, &j.Generation, &j.FromBlock, &j.ToBlock,
			&j.Status, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan reorg job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate reorg jobs: %w", err)
	}
	return jobs, nil
}

// CompleteReorgJob marks a repair job done. It is idempotent.
func (a *PostgreSQLPluginAdapter) CompleteReorgJob(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("job id is required")
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if _, err := db.ExecContext(ctx,
		"UPDATE reorg_jobs SET status = $2, updated_at = now() WHERE id = $1 AND status <> $2",
		jobID, ReorgJobDone); err != nil {
		return fmt.Errorf("failed to complete reorg job: %w", err)
	}
	return nil
}

// PendingReorgNotifications returns notifications that have not been published,
// oldest first. The outbox is drained at least once, so a consumer may see a
// notification twice; job id and generation let it deduplicate.
func (a *PostgreSQLPluginAdapter) PendingReorgNotifications(ctx context.Context, limit int) ([]ReorgNotification, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.QueryContext(ctx, `
SELECT o.id, o.job_id, j.chain_id, j.generation, o.topic, o.payload, o.attempts
	FROM reorg_outbox o
	JOIN reorg_jobs j ON j.id = o.job_id
	WHERE o.published_at IS NULL
	ORDER BY o.id ASC
	LIMIT $1`, limit)
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ReorgNotification
	for rows.Next() {
		var n ReorgNotification
		var payload []byte
		if err := rows.Scan(&n.ID, &n.JobID, &n.ChainID, &n.Generation, &n.Topic, &payload, &n.Attempts); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		n.Payload = payload
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate notifications: %w", err)
	}
	return out, nil
}

// MarkReorgNotificationPublished marks one notification delivered. It is
// idempotent: marking an already-published notification is a no-op, which
// matters because the drainer retries after a crash that happened between
// publishing and marking.
func (a *PostgreSQLPluginAdapter) MarkReorgNotificationPublished(ctx context.Context, id int64) error {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if _, err := db.ExecContext(ctx,
		"UPDATE reorg_outbox SET published_at = now() WHERE id = $1 AND published_at IS NULL", id); err != nil {
		return fmt.Errorf("failed to mark notification published: %w", err)
	}
	return nil
}

// RecordReorgNotificationFailure bumps the attempt counter so a poison
// notification is visible rather than retried forever.
func (a *PostgreSQLPluginAdapter) RecordReorgNotificationFailure(ctx context.Context, id int64) error {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if _, err := db.ExecContext(ctx,
		"UPDATE reorg_outbox SET attempts = attempts + 1 WHERE id = $1", id); err != nil {
		return fmt.Errorf("failed to record notification failure: %w", err)
	}
	return nil
}
