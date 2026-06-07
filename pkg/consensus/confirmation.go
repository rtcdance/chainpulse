package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// ─── Confirmation Depth Gates ────────────────────────────────────────────────

// Default confirmation thresholds. These can be overridden per chain.
const (
	// DefaultConfirmBlocks is the number of additional blocks needed before
	// an event transitions from Pending to Confirmed.
	// On Ethereum mainnet, 12 blocks (~2.4 min) is a common safe threshold.
	DefaultConfirmBlocks uint64 = 12

	// DefaultFinalizeEpochs is the number of epochs (each 32 slots / ~6.4 min)
	// after which a block is considered finalized by the Casper FFG protocol.
	// In practice, finality usually happens within 2–3 epochs.
	DefaultFinalizeEpochs uint64 = 2
)

// ConfirmationConfig holds the chain-specific confirmation thresholds.
type ConfirmationConfig struct {
	ConfirmBlocks   uint64 // blocks after which Pending → Confirmed
	FinalizeEpochs  uint64 // epochs after which Confirmed → Finalized
	SlotsPerEpoch   uint64 // typically 32 for Ethereum
	SlotDurationSec int64  // typically 12 seconds for Ethereum
}

// DefaultConfirmationConfig returns the default Ethereum mainnet confirmation config.
func DefaultConfirmationConfig() ConfirmationConfig {
	return ConfirmationConfig{
		ConfirmBlocks:   DefaultConfirmBlocks,
		FinalizeEpochs:  DefaultFinalizeEpochs,
		SlotsPerEpoch:   SlotsPerEpoch,
		SlotDurationSec: int64(SlotDuration.Seconds()),
	}
}

// BlocksToFinalize returns the approximate number of blocks needed for finalization.
func (c ConfirmationConfig) BlocksToFinalize() uint64 {
	return c.FinalizeEpochs * c.SlotsPerEpoch
}

// TimeToConfirm returns the estimated wall-clock duration until confirmation.
func (c ConfirmationConfig) TimeToConfirm() time.Duration {
	return time.Duration(c.ConfirmBlocks) * time.Duration(c.SlotDurationSec) * time.Second
}

// TimeToFinalize returns the estimated wall-clock duration until finalization.
func (c ConfirmationConfig) TimeToFinalize() time.Duration {
	return time.Duration(c.BlocksToFinalize()) * time.Duration(c.SlotDurationSec) * time.Second
}

// ─── Pending Event Tracking ─────────────────────────────────────────────────

// pendingEvent tracks a single event awaiting confirmation depth.
type pendingEvent struct {
	EventHash   string
	BlockNumber uint64
	BlockHash   string
	Status      blockchain.EventStatus
	QueuedAt    time.Time
}

// FinalityChecker is an optional dependency that queries the chain for
// the actual finalized block number. If set, ConfirmationTracker will
// periodically reconcile on-chain finality with its local state.
type FinalityChecker interface {
	// GetFinalizedBlockNumber returns the latest finalized block number for the chain.
	GetFinalizedBlockNumber(ctx context.Context, chainID string) (uint64, error)

	// IsBlockFinalized returns true if the given block number is at or below
	// the finalized block for the chain.
	IsBlockFinalized(ctx context.Context, chainID string, blockNumber uint64) (bool, error)
}

// ConfirmationTracker tracks events through the Pending → Confirmed → Finalized
// lifecycle, using block depth as the gate for state transitions.
type ConfirmationTracker struct {
	mu           sync.RWMutex
	config       ConfirmationConfig
	pending      map[string]*pendingEvent // eventHash → pending info
	currentBlock uint64
	chainID      string // chain identifier for on-chain finality queries

	// Optional on-chain finality checker
	finalityChecker   FinalityChecker
	reconcileInterval uint64 // check on-chain finality every N blocks (default 64)
	blocksSinceCheck  uint64 // counter since last on-chain check

	// Goroutine lifecycle
	wg       sync.WaitGroup
	done     chan struct{}
	stopOnce sync.Once

	// Rate limiting: at most one reconciliation RPC at a time
	reconcileSem chan struct{}

	// Callbacks for state transitions
	OnConfirmed func(eventHash string)
	OnFinalized func(eventHash string)
}

// NewConfirmationTracker creates a new tracker with the given configuration.
func NewConfirmationTracker(config ConfirmationConfig, chainID string) *ConfirmationTracker {
	return &ConfirmationTracker{
		config:            config,
		chainID:           chainID,
		pending:           make(map[string]*pendingEvent),
		reconcileInterval: 64, // check on-chain finality every 64 blocks
		reconcileSem:      make(chan struct{}, 1),
		done:              make(chan struct{}),
	}
}

// SetFinalityChecker sets the optional on-chain finality checker.
func (t *ConfirmationTracker) SetFinalityChecker(fc FinalityChecker) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.finalityChecker = fc
}

// Stop signals the reconciliation goroutine to exit and waits for it to finish.
// Safe to call multiple times.
func (t *ConfirmationTracker) Stop() {
	t.stopOnce.Do(func() {
		close(t.done)
	})
	t.wg.Wait()
}

// Track adds an event to the confirmation tracking system.
// The event starts in blockchain.EventStatusPending.
func (t *ConfirmationTracker) Track(eventHash string, blockNumber uint64, blockHash string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.pending[eventHash]; exists {
		return // already tracking
	}

	t.pending[eventHash] = &pendingEvent{
		EventHash:   eventHash,
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		Status:      blockchain.EventStatusPending,
		QueuedAt:    time.Now(),
	}
}

// AdvanceBlock is called when a new block is imported. It checks all pending
// events and promotes those that have reached the confirmation or finalization
// depth threshold.
func (t *ConfirmationTracker) AdvanceBlock(blockNumber uint64) []blockchain.EventStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.currentBlock = blockNumber
	t.blocksSinceCheck++
	var transitions []blockchain.EventStatus

	// Periodically reconcile with on-chain finality
	if t.finalityChecker != nil && t.blocksSinceCheck >= t.reconcileInterval {
		t.blocksSinceCheck = 0
		// Reconcile in background to avoid blocking AdvanceBlock.
		// Rate-limited to at most one in-flight RPC by the reconcileSem channel.
		select {
		case t.reconcileSem <- struct{}{}:
			t.wg.Add(1)
			go func() {
				defer t.wg.Done()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("goroutine panic recovered", "panic", r)
					}
				}()
				defer func() { <-t.reconcileSem }()
				select {
				case <-t.done:
					return
				default:
					if _, err := t.ReconcileFinality(); err != nil {
						// Log but don't fail — local depth gates still work
						slog.Warn("on-chain finality reconciliation failed, local depth gates still apply",
							"chain_id", t.chainID, "error", err)
					}
				}
			}()
		default:
			// A reconciliation is already in-flight; skip this round.
			// This prevents unbounded goroutine accumulation when blocks
			// advance faster than the RPC round-trip time.
		}
	}

	for hash, pe := range t.pending {
		// Guard against uint64 underflow: if the chain head moved backward
		// (reorg), blockNumber < pe.BlockNumber and subtraction would wrap to
		// ~2^64, instantly satisfying the confirmation threshold.
		if blockNumber < pe.BlockNumber {
			continue
		}
		blocksSince := blockNumber - pe.BlockNumber

		// Pending → Confirmed
		if pe.Status == blockchain.EventStatusPending && blocksSince >= t.config.ConfirmBlocks {
			pe.Status = blockchain.EventStatusConfirmed
			transitions = append(transitions, blockchain.EventStatusConfirmed)
			if t.OnConfirmed != nil {
				t.OnConfirmed(hash)
			}
		}

		// Confirmed → Finalized (may happen in the same AdvanceBlock call)
		if pe.Status == blockchain.EventStatusConfirmed {
			blocksToFinalize := t.config.BlocksToFinalize()
			if blocksSince >= blocksToFinalize {
				pe.Status = blockchain.EventStatusFinalized
				transitions = append(transitions, blockchain.EventStatusFinalized)
				if t.OnFinalized != nil {
					t.OnFinalized(hash)
				}
			}
		}
	}

	return transitions
}

// ReconcileFinality queries the on-chain finality checker (if configured)
// and promotes any events whose block number is below the finalized block.
// This ensures the tracker stays consistent with the chain even after restarts.
func (t *ConfirmationTracker) ReconcileFinality() (uint64, error) {
	if t.finalityChecker == nil {
		return 0, fmt.Errorf("no finality checker configured")
	}

	// Phase 1: RPC call WITHOUT holding the lock.
	// Holding a mutex across a network call is a liveness risk — a slow RPC
	// (3-10s) would block all AdvanceBlock and Track calls.
	finalizedBlock, err := t.finalityChecker.GetFinalizedBlockNumber(context.Background(), t.chainID)
	if err != nil {
		return 0, fmt.Errorf("failed to get finalized block: %w", err)
	}

	// Phase 2: Update pending event statuses UNDER the lock (fast, no I/O).
	t.mu.Lock()
	promoted := uint64(0)
	for hash, pe := range t.pending {
		if pe.BlockNumber <= finalizedBlock && pe.Status != blockchain.EventStatusFinalized {
			pe.Status = blockchain.EventStatusFinalized
			promoted++
			if t.OnFinalized != nil {
				t.OnFinalized(hash)
			}
		}
	}
	t.mu.Unlock()

	return promoted, nil
}

// pendingEventJSON is the JSON representation of a pendingEvent for persistence.
type pendingEventJSON struct {
	EventHash   string           `json:"event_hash"`
	BlockNumber uint64           `json:"block_number"`
	BlockHash   string           `json:"block_hash"`
	Status      blockchain.EventStatus `json:"status"`
	QueuedAt    time.Time        `json:"queued_at"`
}

// Persist serializes the tracker's pending events to JSON bytes.
// This can be stored in a CheckpointStore for restart recovery.
func (t *ConfirmationTracker) Persist() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	events := make([]pendingEventJSON, 0, len(t.pending))
	for _, pe := range t.pending {
		events = append(events, pendingEventJSON{
			EventHash:   pe.EventHash,
			BlockNumber: pe.BlockNumber,
			BlockHash:   pe.BlockHash,
			Status:      pe.Status,
			QueuedAt:    pe.QueuedAt,
		})
	}

	data, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pending events: %w", err)
	}
	return data, nil
}

// Load restores the tracker's pending events from JSON bytes.
// This should be called on startup before any Track() calls.
func (t *ConfirmationTracker) Load(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var events []pendingEventJSON
	if err := json.Unmarshal(data, &events); err != nil {
		return fmt.Errorf("failed to unmarshal pending events: %w", err)
	}

	for _, e := range events {
		t.pending[e.EventHash] = &pendingEvent{
			EventHash:   e.EventHash,
			BlockNumber: e.BlockNumber,
			BlockHash:   e.BlockHash,
			Status:      e.Status,
			QueuedAt:    e.QueuedAt,
		}
	}

	return nil
}

// GetStatus returns the current confirmation status of an event.
// Returns blockchain.EventStatusPending if the event is not being tracked.
func (t *ConfirmationTracker) GetStatus(eventHash string) blockchain.EventStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	pe, exists := t.pending[eventHash]
	if !exists {
		return blockchain.EventStatusPending
	}
	return pe.Status
}

// PendingCount returns the number of events still awaiting confirmation.
func (t *ConfirmationTracker) PendingCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, pe := range t.pending {
		if pe.Status == blockchain.EventStatusPending {
			count++
		}
	}
	return count
}

// ConfirmedCount returns the number of confirmed but not yet finalized events.
func (t *ConfirmationTracker) ConfirmedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, pe := range t.pending {
		if pe.Status == blockchain.EventStatusConfirmed {
			count++
		}
	}
	return count
}

// FinalizedCount returns the number of finalized events.
func (t *ConfirmationTracker) FinalizedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, pe := range t.pending {
		if pe.Status == blockchain.EventStatusFinalized {
			count++
		}
	}
	return count
}

// TotalTracked returns the total number of tracked events across all states.
func (t *ConfirmationTracker) TotalTracked() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.pending)
}

// RemoveFinalized evicts all finalized events from the tracker to free memory.
// Returns the number of events removed.
func (t *ConfirmationTracker) RemoveFinalized() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	removed := 0
	for hash, pe := range t.pending {
		if pe.Status == blockchain.EventStatusFinalized {
			delete(t.pending, hash)
			removed++
		}
	}
	return removed
}

// MarkReorged marks all events at the given block hash as reorged and removes
// them from tracking. Returns the number of affected events.
func (t *ConfirmationTracker) MarkReorged(blockHash string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	reorged := 0
	for hash, pe := range t.pending {
		if pe.BlockHash == blockHash {
			pe.Status = blockchain.EventStatusReorged
			delete(t.pending, hash)
			reorged++
		}
	}
	return reorged
}

// BlocksUntilConfirmed returns how many more blocks are needed for the event
// to transition to Confirmed. Returns 0 if already confirmed or finalized,
// or an error if the event is not tracked.
func (t *ConfirmationTracker) BlocksUntilConfirmed(eventHash string) (uint64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	pe, exists := t.pending[eventHash]
	if !exists {
		return 0, fmt.Errorf("event %s not tracked", eventHash)
	}

	if pe.Status != blockchain.EventStatusPending {
		return 0, nil
	}

	// Guard against uint64 underflow: if the chain head moved backward (reorg),
	// currentBlock < pe.BlockNumber and the event is not yet confirmable.
	if t.currentBlock < pe.BlockNumber {
		return t.config.ConfirmBlocks, nil
	}
	blocksSince := t.currentBlock - pe.BlockNumber
	if blocksSince >= t.config.ConfirmBlocks {
		return 0, nil
	}

	return t.config.ConfirmBlocks - blocksSince, nil
}
