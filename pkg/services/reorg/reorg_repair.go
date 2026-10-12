package reorg

import (
	"context"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/logkeys"
)

// EventRollerBacker soft-deletes the events of a block range.
type EventRollerBacker interface {
	RollbackEvents(ctx context.Context, fromBlock, toBlock uint64) (int64, error)
}

// CheckpointRewinder moves a chain's checkpoint back to a block.
type CheckpointRewinder interface {
	RewindCheckpoint(ctx context.Context, chainID string, toBlock uint64) error
}

// DerivedStateInvalidator drops derived/cached state for a block range.
//
// The generation is passed explicitly because it is the fence: state written
// under an older generation must not be served, and a repair must be able to
// say which generation it is repairing. Anything cached under a lower
// generation is stale by definition.
type DerivedStateInvalidator interface {
	InvalidateRange(ctx context.Context, chainID string, generation int64, fromBlock, toBlock uint64) error
}

// Repair carries out the derived-state part of a reorg job: roll the events
// back, rewind the checkpoint, and invalidate derived state — then mark the job
// done.
//
// The order matters. Events first, so the data is consistent before the
// checkpoint moves; the checkpoint next, so a crash leaves a resume point that
// is at or below the repaired range (re-running the repair is safe, skipping it
// is not); derived state last, because it is rebuildable from the other two.
type Repair struct {
	jobs        JobStore
	events      EventRollerBacker
	checkpoints CheckpointRewinder
	derived     DerivedStateInvalidator
	logger      core.Logger
}

// NewRepair creates the repair runner.
func NewRepair(
	jobs JobStore,
	events EventRollerBacker,
	checkpoints CheckpointRewinder,
	derived DerivedStateInvalidator,
	logger core.Logger,
) (*Repair, error) {
	if jobs == nil {
		return nil, fmt.Errorf("job store is required")
	}
	if events == nil {
		return nil, fmt.Errorf("event rollback is required")
	}
	if checkpoints == nil {
		return nil, fmt.Errorf("checkpoint rewind is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &Repair{jobs: jobs, events: events, checkpoints: checkpoints, derived: derived, logger: logger}, nil
}

// RepairJob runs the repair for one job.
//
// It is idempotent: a job already marked done is skipped, and a partially
// applied repair can be re-run because every step is a range-scoped idempotent
// operation. The job is marked done only after every step succeeded, so a
// failure leaves it pending and therefore resumable.
func (r *Repair) RepairJob(ctx context.Context, job Job) error {
	if job.Status == JobDone {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if job.ChainID == "" {
		return fmt.Errorf("reorg job %s has no chain id", job.ID)
	}
	if job.ToBlock < job.FromBlock {
		return fmt.Errorf("reorg job %s has an inverted range %d..%d", job.ID, job.FromBlock, job.ToBlock)
	}

	rolledBack, err := r.events.RollbackEvents(ctx, job.FromBlock, job.ToBlock)
	if err != nil {
		return fmt.Errorf("reorg job %s: rollback events %d..%d: %w", job.ID, job.FromBlock, job.ToBlock, err)
	}

	// The checkpoint goes to the block BEFORE the repaired range: the first
	// block that must be re-indexed is FromBlock itself.
	rewindTo := job.FromBlock
	if rewindTo > 0 {
		rewindTo--
	}
	if err := r.checkpoints.RewindCheckpoint(ctx, job.ChainID, rewindTo); err != nil {
		return fmt.Errorf("reorg job %s: rewind checkpoint to %d: %w", job.ID, rewindTo, err)
	}

	if r.derived != nil {
		if err := r.derived.InvalidateRange(ctx, job.ChainID, job.Generation, job.FromBlock, job.ToBlock); err != nil {
			return fmt.Errorf("reorg job %s: invalidate derived state: %w", job.ID, err)
		}
	}

	if err := r.jobs.CompleteJob(ctx, job.ID); err != nil {
		return fmt.Errorf("reorg job %s: complete: %w", job.ID, err)
	}

	r.logger.Info("Reorg job repaired",
		"job_id", job.ID,
		"chain_id", job.ChainID,
		"from_block", job.FromBlock,
		"to_block", job.ToBlock,
		logkeys.LogKeyEventsRolledBack, rolledBack)

	return nil
}

// Resume repairs every unfinished job for a chain and returns how many were
// repaired. An empty chainID means every chain.
//
// A failure stops the pass and is returned with the count repaired so far: the
// remaining jobs stay pending, which is the resumable state.
func (r *Repair) Resume(ctx context.Context, chainID string, limit int) (int, error) {
	jobs, err := r.jobs.PendingJobs(ctx, chainID, limit)
	if err != nil {
		return 0, fmt.Errorf("load pending reorg jobs: %w", err)
	}

	repaired := 0
	for _, job := range jobs {
		if err := r.RepairJob(ctx, job); err != nil {
			return repaired, err
		}
		repaired++
	}
	return repaired, nil
}
