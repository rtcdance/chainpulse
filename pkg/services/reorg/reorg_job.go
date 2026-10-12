package reorg

import (
	"context"
	"strconv"
	"time"
)

// Job statuses.
const (
	JobPending = "pending"
	JobDone    = "done"
)

// Job is a persistent repair job: which block range of which chain must be
// repaired, at which ownership generation.
//
// Recording it is what makes an interrupted reorg resumable: without a job row
// there is nothing to find after a restart.
type Job struct {
	ID         string
	ChainID    string
	Generation int64
	FromBlock  uint64
	ToBlock    uint64
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Notification is one message owed for a job. ChainID and Generation travel
// with it so a consumer can recognise one from a superseded generation without
// a second lookup.
type Notification struct {
	ID         int64
	JobID      string
	ChainID    string
	Generation int64
	Topic      string
	Payload    []byte
	Attempts   int
}

// JobStore persists repair work and the notifications it owes.
//
// EnqueueJob MUST write the job and its notifications in ONE transaction. That
// is the whole point (finding F07): it removes the state "rollback committed,
// notification lost", leaving only "job recorded, notification still pending"
// (which a drainer retries) or "both committed".
type JobStore interface {
	EnqueueJob(ctx context.Context, job Job, notifications []Notification) error
	PendingJobs(ctx context.Context, chainID string, limit int) ([]Job, error)
	CompleteJob(ctx context.Context, jobID string) error
}

// OutboxDrainer publishes owed notifications. The outbox is drained at least
// once, so a consumer may see a notification twice; job id and generation let
// it deduplicate.
type OutboxDrainer interface {
	PendingNotifications(ctx context.Context, limit int) ([]Notification, error)
	MarkPublished(ctx context.Context, id int64) error
	RecordFailure(ctx context.Context, id int64) error
}

// ReorgJobID derives the stable id for a reorg of one chain over a block range.
// It is deterministic so that a retried HandleReorg for the same reorg does not
// create a second job.
func ReorgJobID(chainID string, fromBlock, toBlock uint64) string {
	return chainID + ":" + strconv.FormatUint(fromBlock, 10) + ":" + strconv.FormatUint(toBlock, 10)
}
