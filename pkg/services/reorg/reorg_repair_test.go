package reorg

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// repairRecorder captures the ORDER of repair steps, which is the property
// under test.
type repairRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *repairRecorder) record(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, s)
}

func (r *repairRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

type fakeRollback struct {
	rec *repairRecorder
	n   int64
	err error
}

func (f *fakeRollback) RollbackEvents(_ context.Context, from, to uint64) (int64, error) {
	f.rec.record(fmt.Sprintf("events:%d-%d", from, to))
	return f.n, f.err
}

type fakeCheckpoints struct {
	rec *repairRecorder
	to  uint64
	err error
}

func (f *fakeCheckpoints) RewindCheckpoint(_ context.Context, chainID string, toBlock uint64) error {
	f.rec.record(fmt.Sprintf("checkpoint:%s:%d", chainID, toBlock))
	f.to = toBlock
	return f.err
}

type fakeDerived struct {
	rec        *repairRecorder
	generation int64
	rangeSeen  string
	err        error
}

func (f *fakeDerived) InvalidateRange(_ context.Context, chainID string, generation int64, from, to uint64) error {
	f.rec.record(fmt.Sprintf("derived:%s:gen%d:%d-%d", chainID, generation, from, to))
	f.generation = generation
	f.rangeSeen = fmt.Sprintf("%d-%d", from, to)
	return f.err
}

type repairHarness struct {
	rec         *repairRecorder
	jobs        *fakeJobStore
	events      *fakeRollback
	checkpoints *fakeCheckpoints
	derived     *fakeDerived
	repair      *Repair
}

func newRepairHarness(t *testing.T) *repairHarness {
	t.Helper()

	rec := &repairRecorder{}
	h := &repairHarness{
		rec:         rec,
		jobs:        &fakeJobStore{},
		events:      &fakeRollback{rec: rec, n: 7},
		checkpoints: &fakeCheckpoints{rec: rec},
		derived:     &fakeDerived{rec: rec},
	}

	repair, err := NewRepair(h.jobs, h.events, h.checkpoints, h.derived, core.NewDefaultLogger(core.LogLevelError))
	require.NoError(t, err)
	h.repair = repair
	return h
}

func repairJobFixture() Job {
	return Job{
		ID:         "ethereum:100:110",
		ChainID:    "ethereum",
		Generation: 4,
		FromBlock:  100,
		ToBlock:    110,
		Status:     JobPending,
	}
}

// TestRepairJobRunsStepsInOrder pins the ordering: events, then the checkpoint,
// then derived state, then completion. Derived state is rebuildable, so it must
// not run before the data it derives from is consistent.
func TestRepairJobRunsStepsInOrder(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	require.NoError(t, h.repair.RepairJob(context.Background(), repairJobFixture()))

	assert.Equal(t, []string{
		"events:100-110",
		"checkpoint:ethereum:99", // the block BEFORE the repaired range
		"derived:ethereum:gen4:100-110",
	}, h.rec.snapshot())

	assert.Equal(t, []string{"ethereum:100:110"}, h.jobs.completed)
}

// TestRepairJobPassesGenerationToInvalidation is the generation-fence property:
// a repair must tell the invalidator which generation it is repairing, so state
// cached under a lower generation is not served.
func TestRepairJobPassesGenerationToInvalidation(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	require.NoError(t, h.repair.RepairJob(context.Background(), repairJobFixture()))

	assert.Equal(t, int64(4), h.derived.generation)
	assert.Equal(t, "100-110", h.derived.rangeSeen)
}

func TestRepairJobSkipsDoneJob(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	job := repairJobFixture()
	job.Status = JobDone

	require.NoError(t, h.repair.RepairJob(context.Background(), job))
	assert.Empty(t, h.rec.snapshot(), "a done job must not be re-applied")
	assert.Empty(t, h.jobs.completed)
}

// TestRepairJobLeavesJobPendingOnFailure is the resumability property: the job
// is marked done only after every step succeeded.
func TestRepairJobLeavesJobPendingOnFailure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		break_ func(*repairHarness)
	}{
		{"events fail", func(h *repairHarness) { h.events.err = errors.New("boom") }},
		{"checkpoint fails", func(h *repairHarness) { h.checkpoints.err = errors.New("boom") }},
		{"derived state fails", func(h *repairHarness) { h.derived.err = errors.New("boom") }},
		{"complete fails", func(h *repairHarness) { h.jobs.completeErr = errors.New("boom") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newRepairHarness(t)
			tc.break_(h)

			require.Error(t, h.repair.RepairJob(context.Background(), repairJobFixture()))
			assert.Empty(t, h.jobs.completed, "a failed repair must stay pending and resumable")
		})
	}
}

func TestRepairJobRejectsInvertedRange(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	job := repairJobFixture()
	job.FromBlock, job.ToBlock = 110, 100

	require.Error(t, h.repair.RepairJob(context.Background(), job))
	assert.Empty(t, h.rec.snapshot())
}

func TestRepairJobRejectsMissingChain(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	job := repairJobFixture()
	job.ChainID = ""

	require.Error(t, h.repair.RepairJob(context.Background(), job))
	assert.Empty(t, h.rec.snapshot())
}

// TestRepairResumeRepairsEveryPendingJob is the "resume after restart" path.
func TestRepairResumeRepairsEveryPendingJob(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	h.jobs.pending = []Job{
		{ID: "a", ChainID: "ethereum", Generation: 4, FromBlock: 100, ToBlock: 105, Status: JobPending},
		{ID: "b", ChainID: "ethereum", Generation: 4, FromBlock: 106, ToBlock: 110, Status: JobPending},
	}

	repaired, err := h.repair.Resume(context.Background(), "ethereum", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, repaired)
	assert.Equal(t, []string{"a", "b"}, h.jobs.completed)
}

// TestRepairResumeStopsOnFailure keeps the remaining jobs pending so the next
// pass picks them up.
func TestRepairResumeStopsOnFailure(t *testing.T) {
	t.Parallel()

	h := newRepairHarness(t)
	h.events.err = errors.New("boom")
	h.jobs.pending = []Job{
		{ID: "a", ChainID: "ethereum", FromBlock: 100, ToBlock: 105, Status: JobPending},
		{ID: "b", ChainID: "ethereum", FromBlock: 106, ToBlock: 110, Status: JobPending},
	}

	repaired, err := h.repair.Resume(context.Background(), "ethereum", 10)
	require.Error(t, err)
	assert.Zero(t, repaired)
	assert.Empty(t, h.jobs.completed, "nothing is marked done when the pass fails")
}
