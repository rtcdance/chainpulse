package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ─── Backfill Coordinator ────────────────────────────────────────────────────

// BackfillStatus represents the state of a backfill job.
type BackfillStatus string

const (
	BackfillPending   BackfillStatus = "pending"
	BackfillRunning   BackfillStatus = "running"
	BackfillCompleted BackfillStatus = "completed"
	BackfillFailed    BackfillStatus = "failed"
	BackfillCancelled BackfillStatus = "cancelled"
)

// BackfillRange represents a block range to be indexed.
type BackfillRange struct {
	FromBlock uint64 `json:"from_block"`
	ToBlock   uint64 `json:"to_block"`
}

// BackfillJob tracks the progress of a single backfill operation.
type BackfillJob struct {
	ID         string         `json:"id"`
	ChainID    string         `json:"chain_id"`
	Range      BackfillRange  `json:"range"`
	Status     BackfillStatus `json:"status"`
	Progress   uint64         `json:"progress"` // last successfully indexed block
	Error      string         `json:"error,omitempty"`
	StartedAt  time.Time      `json:"started_at,omitempty"`
	FinishedAt time.Time      `json:"finished_at,omitempty"`
}

// PercentComplete returns the completion percentage (0–100).
func (j *BackfillJob) PercentComplete() float64 {
	total := j.Range.ToBlock - j.Range.FromBlock
	if total == 0 {
		return 100
	}
	indexed := j.Progress - j.Range.FromBlock
	return float64(indexed) / float64(total) * 100
}

// IsFinished returns true if the job is in a terminal state.
func (j *BackfillJob) IsFinished() bool {
	return j.Status == BackfillCompleted || j.Status == BackfillFailed || j.Status == BackfillCancelled
}

// BackfillConfig holds configuration for backfill operations.
type BackfillConfig struct {
	// MaxConcurrency is the maximum number of concurrent block-range workers.
	MaxConcurrency int `json:"max_concurrency"`
	// ChunkSize is the number of blocks per worker chunk.
	ChunkSize uint64 `json:"chunk_size"`
	// MaxRetries is the maximum number of retries per chunk on failure.
	MaxRetries int `json:"max_retries"`
	// RetryDelay is the delay between retries.
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultBackfillConfig returns sensible defaults.
func DefaultBackfillConfig() BackfillConfig {
	return BackfillConfig{
		MaxConcurrency: 4,
		ChunkSize:      1000,
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
	}
}

// BackfillCoordinator manages historical block range indexing.
// It partitions a large block range into chunks and processes them
// concurrently with retry support.
type BackfillCoordinator struct {
	mu          sync.RWMutex
	config      BackfillConfig
	jobs        map[string]*BackfillJob // job ID → job
	sem         chan struct{}            // concurrency semaphore
	jobCounter  int64                    // monotonic ID generator
}

// NewBackfillCoordinator creates a new coordinator.
func NewBackfillCoordinator(config BackfillConfig) *BackfillCoordinator {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 4
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1000
	}
	return &BackfillCoordinator{
		config: config,
		jobs:   make(map[string]*BackfillJob),
		sem:    make(chan struct{}, config.MaxConcurrency),
	}
}

// PartitionRange splits a block range into chunks of the configured size.
func (bc *BackfillCoordinator) PartitionRange(r BackfillRange) []BackfillRange {
	if r.FromBlock > r.ToBlock {
		return nil
	}

	var chunks []BackfillRange
	for start := r.FromBlock; start <= r.ToBlock; start += bc.config.ChunkSize {
		end := start + bc.config.ChunkSize - 1
		if end > r.ToBlock {
			end = r.ToBlock
		}
		chunks = append(chunks, BackfillRange{FromBlock: start, ToBlock: end})
	}
	return chunks
}

// CreateJob creates a new backfill job for the given chain and range.
func (bc *BackfillCoordinator) CreateJob(chainID string, r BackfillRange) *BackfillJob {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.jobCounter++
	job := &BackfillJob{
		ID:      fmt.Sprintf("bf_%d", bc.jobCounter),
		ChainID: chainID,
		Range:   r,
		Status:  BackfillPending,
		Progress: r.FromBlock,
	}
	bc.jobs[job.ID] = job
	return job
}

// StartJob marks a job as running.
func (bc *BackfillCoordinator) StartJob(jobID string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	job, exists := bc.jobs[jobID]
	if !exists {
		return fmt.Errorf("backfill job %s not found", jobID)
	}
	if job.Status != BackfillPending {
		return fmt.Errorf("job %s is %s, cannot start", jobID, job.Status)
	}
	job.Status = BackfillRunning
	job.StartedAt = time.Now()
	return nil
}

// UpdateProgress updates the progress of a running job.
func (bc *BackfillCoordinator) UpdateProgress(jobID string, lastIndexedBlock uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if job, exists := bc.jobs[jobID]; exists && job.Status == BackfillRunning {
		if lastIndexedBlock > job.Progress {
			job.Progress = lastIndexedBlock
		}
	}
}

// CompleteJob marks a job as completed.
func (bc *BackfillCoordinator) CompleteJob(jobID string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if job, exists := bc.jobs[jobID]; exists {
		job.Status = BackfillCompleted
		job.Progress = job.Range.ToBlock
		job.FinishedAt = time.Now()
	}
}

// FailJob marks a job as failed.
func (bc *BackfillCoordinator) FailJob(jobID string, err error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if job, exists := bc.jobs[jobID]; exists {
		job.Status = BackfillFailed
		job.Error = err.Error()
		job.FinishedAt = time.Now()
	}
}

// CancelJob cancels a running job.
func (bc *BackfillCoordinator) CancelJob(jobID string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if job, exists := bc.jobs[jobID]; exists && !job.IsFinished() {
		job.Status = BackfillCancelled
		job.FinishedAt = time.Now()
	}
}

// GetJob returns a copy of the job.
func (bc *BackfillCoordinator) GetJob(jobID string) (*BackfillJob, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	job, exists := bc.jobs[jobID]
	if !exists {
		return nil, false
	}
	cp := *job
	return &cp, true
}

// ActiveJobs returns all non-terminal jobs.
func (bc *BackfillCoordinator) ActiveJobs() []*BackfillJob {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	var active []*BackfillJob
	for _, job := range bc.jobs {
		if !job.IsFinished() {
			cp := *job
			active = append(active, &cp)
		}
	}
	return active
}

// AcquireSlot acquires a concurrency slot (blocks if at capacity).
func (bc *BackfillCoordinator) AcquireSlot() {
	bc.sem <- struct{}{}
}

// ReleaseSlot releases a concurrency slot.
func (bc *BackfillCoordinator) ReleaseSlot() {
	<-bc.sem
}

// TryAcquireSlot tries to acquire a slot without blocking.
func (bc *BackfillCoordinator) TryAcquireSlot() bool {
	select {
	case bc.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// ─── Head Tracker ───────────────────────────────────────────────────────────

// HeadTracker monitors the chain head and detects gaps between polls.
// When a gap is detected (e.g., poll returns block N, next poll returns N+3),
// it emits the missing blocks for backfill.
type HeadTracker struct {
	mu          sync.RWMutex
	chainID     string
	lastHead    uint64
	gapDetected func(chainID string, fromBlock, toBlock uint64)
}

// NewHeadTracker creates a new head tracker.
func NewHeadTracker(chainID string) *HeadTracker {
	return &HeadTracker{
		chainID: chainID,
	}
}

// OnGapDetected registers a callback for when a gap is detected.
func (h *HeadTracker) OnGapDetected(fn func(chainID string, fromBlock, toBlock uint64)) {
	h.gapDetected = fn
}

// AdvanceHead updates the head block number and detects any gaps.
// Returns the missing block range if a gap was detected, or nil otherwise.
func (h *HeadTracker) AdvanceHead(newHead uint64) *BackfillRange {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.lastHead == 0 {
		// First head — no gap possible
		h.lastHead = newHead
		return nil
	}

	if newHead <= h.lastHead {
		// Stale or reorg — no forward gap
		return nil
	}

	if newHead == h.lastHead+1 {
		// Normal progression — no gap
		h.lastHead = newHead
		return nil
	}

	// Gap detected: blocks [lastHead+1, newHead-1] were missed
	gap := &BackfillRange{
		FromBlock: h.lastHead + 1,
		ToBlock:   newHead - 1,
	}

	h.lastHead = newHead

	if h.gapDetected != nil {
		h.gapDetected(h.chainID, gap.FromBlock, gap.ToBlock)
	}

	return gap
}

// LastHead returns the last known head block.
func (h *HeadTracker) LastHead() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastHead
}

// ─── Checkpoint Manager ─────────────────────────────────────────────────────

// CheckpointManager manages high-water mark tracking and auto-persist.
type CheckpointManager struct {
	mu       sync.RWMutex
	store    CheckpointStore
	watermarks map[string]uint64 // chainID → last persisted block
	dirty    map[string]bool     // chainID → needs persist
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(store CheckpointStore) *CheckpointManager {
	return &CheckpointManager{
		store:      store,
		watermarks: make(map[string]uint64),
		dirty:      make(map[string]bool),
	}
}

// Advance updates the high-water mark for a chain. It does NOT persist immediately.
func (cm *CheckpointManager) Advance(chainID string, blockNumber uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	current, exists := cm.watermarks[chainID]
	if !exists || blockNumber > current {
		cm.watermarks[chainID] = blockNumber
		cm.dirty[chainID] = true
	}
}

// Flush persists all dirty watermarks to the store.
func (cm *CheckpointManager) Flush(ctx context.Context) error {
	cm.mu.Lock()
	dirty := make(map[string]uint64)
	for chainID, isDirty := range cm.dirty {
		if isDirty {
			dirty[chainID] = cm.watermarks[chainID]
			cm.dirty[chainID] = false
		}
	}
	cm.mu.Unlock()

	for chainID, block := range dirty {
		if err := cm.store.SaveLastIndexedBlock(ctx, chainID, block, ""); err != nil {
			return fmt.Errorf("failed to flush checkpoint for %s: %w", chainID, err)
		}
	}
	return nil
}

// Load restores the high-water mark from the store.
func (cm *CheckpointManager) Load(ctx context.Context, chainID string) (uint64, error) {
	block, _, err := cm.store.GetLastIndexedBlock(ctx, chainID)
	if err != nil {
		return 0, err
	}
	cm.mu.Lock()
	cm.watermarks[chainID] = block
	cm.dirty[chainID] = false
	cm.mu.Unlock()
	return block, nil
}

// Watermark returns the current high-water mark for a chain.
func (cm *CheckpointManager) Watermark(chainID string) uint64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.watermarks[chainID]
}

// IsDirty returns whether a chain's watermark needs to be persisted.
func (cm *CheckpointManager) IsDirty(chainID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.dirty[chainID]
}
