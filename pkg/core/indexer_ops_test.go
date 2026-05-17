package core

import (
	"context"
	"errors"
	"testing"
)

// ─── Backfill Coordinator Tests ──────────────────────────────────────────────

func TestBackfillCoordinator_PartitionRange(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(BackfillConfig{ChunkSize: 100})

	chunks := bc.PartitionRange(BackfillRange{FromBlock: 0, ToBlock: 249})
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0].FromBlock != 0 || chunks[0].ToBlock != 99 {
		t.Errorf("chunk 0 = %+v, want [0,99]", chunks[0])
	}
	if chunks[1].FromBlock != 100 || chunks[1].ToBlock != 199 {
		t.Errorf("chunk 1 = %+v, want [100,199]", chunks[1])
	}
	if chunks[2].FromBlock != 200 || chunks[2].ToBlock != 249 {
		t.Errorf("chunk 2 = %+v, want [200,249]", chunks[2])
	}
}

func TestBackfillCoordinator_PartitionRange_ExactFit(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(BackfillConfig{ChunkSize: 100})
	chunks := bc.PartitionRange(BackfillRange{FromBlock: 0, ToBlock: 99})
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestBackfillCoordinator_PartitionRange_Inverted(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(BackfillConfig{ChunkSize: 100})
	chunks := bc.PartitionRange(BackfillRange{FromBlock: 200, ToBlock: 100})
	if chunks != nil {
		t.Errorf("inverted range should return nil, got %v", chunks)
	}
}

func TestBackfillCoordinator_JobLifecycle(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(DefaultBackfillConfig())

	job := bc.CreateJob("ethereum", BackfillRange{FromBlock: 1000, ToBlock: 5000})
	if job.Status != BackfillPending {
		t.Errorf("initial status = %s, want pending", job.Status)
	}

	// Start
	if err := bc.StartJob(job.ID); err != nil {
		t.Fatalf("StartJob failed: %v", err)
	}
	got, _ := bc.GetJob(job.ID)
	if got.Status != BackfillRunning {
		t.Errorf("status after start = %s, want running", got.Status)
	}

	// Progress
	bc.UpdateProgress(job.ID, 2000)
	got, _ = bc.GetJob(job.ID)
	if got.Progress != 2000 {
		t.Errorf("progress = %d, want 2000", got.Progress)
	}

	// Complete
	bc.CompleteJob(job.ID)
	got, _ = bc.GetJob(job.ID)
	if got.Status != BackfillCompleted {
		t.Errorf("status after complete = %s, want completed", got.Status)
	}
	if got.PercentComplete() != 100 {
		t.Errorf("percent = %f, want 100", got.PercentComplete())
	}
}

func TestBackfillCoordinator_FailJob(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(DefaultBackfillConfig())
	job := bc.CreateJob("ethereum", BackfillRange{FromBlock: 0, ToBlock: 100})
	bc.StartJob(job.ID)
	bc.FailJob(job.ID, errors.New("connection refused"))

	got, _ := bc.GetJob(job.ID)
	if got.Status != BackfillFailed {
		t.Errorf("status = %s, want failed", got.Status)
	}
	if got.Error != "connection refused" {
		t.Errorf("error = %s, want connection refused", got.Error)
	}
}

func TestBackfillCoordinator_CancelJob(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(DefaultBackfillConfig())
	job := bc.CreateJob("ethereum", BackfillRange{FromBlock: 0, ToBlock: 100})
	bc.StartJob(job.ID)
	bc.CancelJob(job.ID)

	got, _ := bc.GetJob(job.ID)
	if got.Status != BackfillCancelled {
		t.Errorf("status = %s, want cancelled", got.Status)
	}
}

func TestBackfillCoordinator_ActiveJobs(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(DefaultBackfillConfig())
	bc.CreateJob("eth", BackfillRange{0, 100})
	job2 := bc.CreateJob("eth", BackfillRange{200, 300})
	bc.StartJob(job2.ID)
	bc.CompleteJob(job2.ID)

	active := bc.ActiveJobs()
	if len(active) != 1 {
		t.Errorf("active jobs = %d, want 1", len(active))
	}
}

func TestBackfillCoordinator_ConcurrencySemaphore(t *testing.T) {
	t.Parallel()
	bc := NewBackfillCoordinator(BackfillConfig{MaxConcurrency: 2})

	if !bc.TryAcquireSlot() {
		t.Error("first acquire should succeed")
	}
	if !bc.TryAcquireSlot() {
		t.Error("second acquire should succeed")
	}
	if bc.TryAcquireSlot() {
		t.Error("third acquire should fail (at capacity)")
	}
	bc.ReleaseSlot()
	if !bc.TryAcquireSlot() {
		t.Error("acquire after release should succeed")
	}
	bc.ReleaseSlot()
	bc.ReleaseSlot()
}

func TestBackfillJob_PercentComplete(t *testing.T) {
	t.Parallel()
	job := &BackfillJob{
		Range:    BackfillRange{FromBlock: 100, ToBlock: 200},
		Progress: 150,
	}
	if pct := job.PercentComplete(); pct != 50.0 {
		t.Errorf("percent = %f, want 50", pct)
	}
}

// ─── Head Tracker Tests ─────────────────────────────────────────────────────

func TestHeadTracker_NoGapOnFirst(t *testing.T) {
	t.Parallel()
	ht := NewHeadTracker("ethereum")
	gap := ht.AdvanceHead(100)
	if gap != nil {
		t.Errorf("first head should not produce gap, got %+v", gap)
	}
	if ht.LastHead() != 100 {
		t.Errorf("lastHead = %d, want 100", ht.LastHead())
	}
}

func TestHeadTracker_NormalProgression(t *testing.T) {
	t.Parallel()
	ht := NewHeadTracker("ethereum")
	ht.AdvanceHead(100)
	gap := ht.AdvanceHead(101)
	if gap != nil {
		t.Errorf("normal progression should not produce gap, got %+v", gap)
	}
}

func TestHeadTracker_GapDetection(t *testing.T) {
	t.Parallel()
	ht := NewHeadTracker("ethereum")
	ht.AdvanceHead(100)

	gap := ht.AdvanceHead(105)
	if gap == nil {
		t.Fatal("expected gap detection")
	}
	if gap.FromBlock != 101 || gap.ToBlock != 104 {
		t.Errorf("gap = [%d, %d], want [101, 104]", gap.FromBlock, gap.ToBlock)
	}
	if ht.LastHead() != 105 {
		t.Errorf("lastHead = %d, want 105", ht.LastHead())
	}
}

func TestHeadTracker_StaleBlock(t *testing.T) {
	t.Parallel()
	ht := NewHeadTracker("ethereum")
	ht.AdvanceHead(100)
	gap := ht.AdvanceHead(99)
	if gap != nil {
		t.Errorf("stale block should not produce gap, got %+v", gap)
	}
}

func TestHeadTracker_Callback(t *testing.T) {
	t.Parallel()
	ht := NewHeadTracker("ethereum")
	var calledWith struct{ from, to uint64 }
	ht.OnGapDetected(func(chainID string, fromBlock, toBlock uint64) {
		calledWith.from = fromBlock
		calledWith.to = toBlock
	})

	ht.AdvanceHead(100)
	ht.AdvanceHead(110)

	if calledWith.from != 101 || calledWith.to != 109 {
		t.Errorf("callback got [%d, %d], want [101, 109]", calledWith.from, calledWith.to)
	}
}

// ─── Checkpoint Manager Tests ────────────────────────────────────────────────

type mockCheckpointStore struct {
	data   map[string]uint64
	err    error
	writes int
}

func newMockCheckpointStore() *mockCheckpointStore {
	return &mockCheckpointStore{data: make(map[string]uint64)}
}

func (m *mockCheckpointStore) SaveLastIndexedBlock(_ context.Context, chainID string, blockNumber uint64, _ string) error {
	m.writes++
	if m.err != nil {
		return m.err
	}
	m.data[chainID] = blockNumber
	return nil
}

func (m *mockCheckpointStore) GetLastIndexedBlock(_ context.Context, chainID string) (uint64, string, error) {
	if m.err != nil {
		return 0, "", m.err
	}
	return m.data[chainID], "", nil
}

func (m *mockCheckpointStore) GetBlockHash(_ context.Context, _ string, _ uint64) (string, error) {
	return "", nil
}

func TestCheckpointManager_AdvanceAndFlush(t *testing.T) {
	t.Parallel()
	store := newMockCheckpointStore()
	cm := NewCheckpointManager(store)

	cm.Advance("ethereum", 100)
	cm.Advance("ethereum", 200)
	cm.Advance("ethereum", 150) // stale — should not update

	if !cm.IsDirty("ethereum") {
		t.Error("should be dirty after advance")
	}
	if cm.Watermark("ethereum") != 200 {
		t.Errorf("watermark = %d, want 200", cm.Watermark("ethereum"))
	}

	if err := cm.Flush(context.Background()); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
	if cm.IsDirty("ethereum") {
		t.Error("should not be dirty after flush")
	}
	if store.data["ethereum"] != 200 {
		t.Errorf("persisted = %d, want 200", store.data["ethereum"])
	}
	if store.writes != 1 {
		t.Errorf("writes = %d, want 1 (single batch)", store.writes)
	}
}

func TestCheckpointManager_Load(t *testing.T) {
	t.Parallel()
	store := newMockCheckpointStore()
	store.data["ethereum"] = 500

	cm := NewCheckpointManager(store)
	block, err := cm.Load(context.Background(), "ethereum")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if block != 500 {
		t.Errorf("loaded = %d, want 500", block)
	}
	if cm.IsDirty("ethereum") {
		t.Error("should not be dirty after load")
	}
}

func TestCheckpointManager_FlushError(t *testing.T) {
	t.Parallel()
	store := newMockCheckpointStore()
	store.err = errors.New("disk full")

	cm := NewCheckpointManager(store)
	cm.Advance("ethereum", 100)

	if err := cm.Flush(context.Background()); err == nil {
		t.Fatal("expected flush error")
	}
}

func TestCheckpointManager_MultipleChains(t *testing.T) {
	t.Parallel()
	store := newMockCheckpointStore()
	cm := NewCheckpointManager(store)

	cm.Advance("ethereum", 100)
	cm.Advance("polygon", 200)

	if err := cm.Flush(context.Background()); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if store.data["ethereum"] != 100 {
		t.Errorf("ethereum = %d, want 100", store.data["ethereum"])
	}
	if store.data["polygon"] != 200 {
		t.Errorf("polygon = %d, want 200", store.data["polygon"])
	}
}
