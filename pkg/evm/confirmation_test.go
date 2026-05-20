package evm

import (
	"testing"
)

func TestConfirmationConfig_Defaults(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfirmationConfig()
	if cfg.ConfirmBlocks != 12 {
		t.Errorf("ConfirmBlocks = %d, want 12", cfg.ConfirmBlocks)
	}
	if cfg.FinalizeEpochs != 2 {
		t.Errorf("FinalizeEpochs = %d, want 2", cfg.FinalizeEpochs)
	}
	if cfg.BlocksToFinalize() != 64 { // 2 * 32
		t.Errorf("BlocksToFinalize = %d, want 64", cfg.BlocksToFinalize())
	}
}

func TestConfirmationTracker_PendingToConfirmed(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  3,
		FinalizeEpochs: 2,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	// Track event at block 100
	tracker.Track("hash1", 100, "blockhash100")

	if status := tracker.GetStatus("hash1"); status != EventStatusPending {
		t.Errorf("status = %s, want pending", status)
	}

	// Block 101: not enough depth yet
	tracker.AdvanceBlock(101)
	if status := tracker.GetStatus("hash1"); status != EventStatusPending {
		t.Errorf("status at 101 = %s, want pending", status)
	}

	// Block 102: still not enough (need 3 blocks = 103)
	tracker.AdvanceBlock(102)
	if status := tracker.GetStatus("hash1"); status != EventStatusPending {
		t.Errorf("status at 102 = %s, want pending", status)
	}

	// Block 103: 103 - 100 = 3 blocks = confirmed!
	transitions := tracker.AdvanceBlock(103)
	if status := tracker.GetStatus("hash1"); status != EventStatusConfirmed {
		t.Errorf("status at 103 = %s, want confirmed", status)
	}
	if len(transitions) != 1 || transitions[0] != EventStatusConfirmed {
		t.Errorf("transitions = %v, want [confirmed]", transitions)
	}
}

func TestConfirmationTracker_ConfirmedToFinalized(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  3,
		FinalizeEpochs: 2,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("hash1", 100, "blockhash100")

	// Advance to confirmed
	tracker.AdvanceBlock(103)
	if status := tracker.GetStatus("hash1"); status != EventStatusConfirmed {
		t.Errorf("status = %s, want confirmed", status)
	}

	// Not yet finalized: need 64 blocks (2 epochs * 32 slots)
	tracker.AdvanceBlock(163)
	if status := tracker.GetStatus("hash1"); status != EventStatusConfirmed {
		t.Errorf("status at 163 = %s, want confirmed (not yet finalized)", status)
	}

	// Block 164: 164 - 100 = 64 = finalized!
	tracker.AdvanceBlock(164)
	if status := tracker.GetStatus("hash1"); status != EventStatusFinalized {
		t.Errorf("status at 164 = %s, want finalized", status)
	}
}

func TestConfirmationTracker_Callbacks(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  2,
		FinalizeEpochs: 1,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	var confirmedHashes []string
	var finalizedHashes []string
	tracker.OnConfirmed = func(hash string) { confirmedHashes = append(confirmedHashes, hash) }
	tracker.OnFinalized = func(hash string) { finalizedHashes = append(finalizedHashes, hash) }

	tracker.Track("h1", 100, "b1")
	tracker.Track("h2", 100, "b1")

	tracker.AdvanceBlock(102) // confirmed
	if len(confirmedHashes) != 2 {
		t.Errorf("confirmed callbacks = %d, want 2", len(confirmedHashes))
	}

	tracker.AdvanceBlock(132) // finalized (100 + 32)
	if len(finalizedHashes) != 2 {
		t.Errorf("finalized callbacks = %d, want 2", len(finalizedHashes))
	}
}

func TestConfirmationTracker_Counts(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  5,
		FinalizeEpochs: 2,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("h1", 100, "b1")
	tracker.Track("h2", 101, "b1")
	tracker.Track("h3", 102, "b1")

	if tracker.PendingCount() != 3 {
		t.Errorf("pending = %d, want 3", tracker.PendingCount())
	}
	if tracker.TotalTracked() != 3 {
		t.Errorf("total = %d, want 3", tracker.TotalTracked())
	}

	// Advance to confirm h1 only (105 - 100 = 5)
	tracker.AdvanceBlock(105)
	if tracker.PendingCount() != 2 {
		t.Errorf("pending = %d, want 2", tracker.PendingCount())
	}
	if tracker.ConfirmedCount() != 1 {
		t.Errorf("confirmed = %d, want 1", tracker.ConfirmedCount())
	}
}

func TestConfirmationTracker_RemoveFinalized(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  2,
		FinalizeEpochs: 1,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("h1", 100, "b1")
	tracker.Track("h2", 100, "b1")

	// Advance to finalized
	tracker.AdvanceBlock(132)

	if tracker.FinalizedCount() != 2 {
		t.Errorf("finalized = %d, want 2", tracker.FinalizedCount())
	}

	removed := tracker.RemoveFinalized()
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	if tracker.TotalTracked() != 0 {
		t.Errorf("total after remove = %d, want 0", tracker.TotalTracked())
	}
}

func TestConfirmationTracker_MarkReorged(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  5,
		FinalizeEpochs: 2,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("h1", 100, "blockA")
	tracker.Track("h2", 100, "blockA")
	tracker.Track("h3", 101, "blockB")

	reorged := tracker.MarkReorged("blockA")
	if reorged != 2 {
		t.Errorf("reorged = %d, want 2", reorged)
	}
	if tracker.TotalTracked() != 1 {
		t.Errorf("total after reorg = %d, want 1", tracker.TotalTracked())
	}
	if status := tracker.GetStatus("h3"); status != EventStatusPending {
		t.Errorf("h3 status = %s, want pending (not reorged)", status)
	}
}

func TestConfirmationTracker_BlocksUntilConfirmed(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{
		ConfirmBlocks:  12,
		FinalizeEpochs: 2,
		SlotsPerEpoch:  32,
	}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("h1", 100, "b1")
	tracker.AdvanceBlock(105)

	blocks, err := tracker.BlocksUntilConfirmed("h1")
	if err != nil {
		t.Fatalf("BlocksUntilConfirmed error: %v", err)
	}
	// 12 - (105 - 100) = 7
	if blocks != 7 {
		t.Errorf("blocks until confirmed = %d, want 7", blocks)
	}

	// After confirmed, returns 0
	tracker.AdvanceBlock(112)
	blocks, err = tracker.BlocksUntilConfirmed("h1")
	if err != nil {
		t.Fatalf("BlocksUntilConfirmed error: %v", err)
	}
	if blocks != 0 {
		t.Errorf("blocks after confirmed = %d, want 0", blocks)
	}

	// Unknown event
	_, err = tracker.BlocksUntilConfirmed("unknown")
	if err == nil {
		t.Fatal("expected error for unknown event")
	}
}

func TestConfirmationTracker_DuplicateTrack(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{ConfirmBlocks: 5, FinalizeEpochs: 2, SlotsPerEpoch: 32}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("h1", 100, "b1")
	tracker.Track("h1", 100, "b1") // duplicate should be no-op

	if tracker.TotalTracked() != 1 {
		t.Errorf("total = %d, want 1 (duplicate ignored)", tracker.TotalTracked())
	}
}

func TestConfirmationTracker_GetStatus_Untracked(t *testing.T) {
	t.Parallel()
	cfg := ConfirmationConfig{ConfirmBlocks: 5, FinalizeEpochs: 2, SlotsPerEpoch: 32}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	if status := tracker.GetStatus("unknown"); status != EventStatusPending {
		t.Errorf("untracked status = %s, want pending", status)
	}
}

func TestBlockchainEvent_IsFinalized(t *testing.T) {
	t.Parallel()
	ev := &BlockchainEvent{Status: EventStatusFinalized}
	if !ev.IsFinalized() {
		t.Error("expected IsFinalized = true")
	}

	ev.Status = EventStatusConfirmed
	if ev.IsFinalized() {
		t.Error("confirmed should not be finalized")
	}
}

func TestConfirmationTracker_Uint64UnderflowOnReorg(t *testing.T) {
	t.Parallel()
	// If the chain head moves backward (reorg), blockNumber < pe.BlockNumber.
	// Without the underflow guard, subtraction wraps to ~2^64 and instantly
	// satisfies the confirmation threshold, promoting events to Finalized.
	cfg := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	// Track an event at block 100, then advance to 105
	tracker.Track("h1", 100, "b1")
	tracker.AdvanceBlock(105)
	if status := tracker.GetStatus("h1"); status != EventStatusPending {
		t.Fatalf("before reorg: status = %s, want pending", status)
	}

	// Reorg moves the chain head back to block 95
	tracker.AdvanceBlock(95)

	// The event must NOT be promoted — it's still pending
	status := tracker.GetStatus("h1")
	if status != EventStatusPending {
		t.Errorf("after reorg head=95 < event=100: status = %s, want pending", status)
	}
}

func TestConfirmationTracker_BlocksUntilConfirmed_UnderflowOnReorg(t *testing.T) {
	t.Parallel()
	// After reorg, currentBlock < event.BlockNumber.
	// BlocksUntilConfirmed must not underflow and must return ConfirmBlocks
	// (full wait still needed, not 0).
	cfg := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32}
	tracker := NewConfirmationTracker(cfg, "ethereum")

	tracker.Track("h1", 100, "b1")
	tracker.AdvanceBlock(105)

	blocks, err := tracker.BlocksUntilConfirmed("h1")
	if err != nil {
		t.Fatalf("BlocksUntilConfirmed error: %v", err)
	}
	// 12 - (105-100) = 7
	if blocks != 7 {
		t.Errorf("before reorg: blocks = %d, want 7", blocks)
	}

	// Reorg moves head back to 95
	tracker.AdvanceBlock(95)
	blocks, err = tracker.BlocksUntilConfirmed("h1")
	if err != nil {
		t.Fatalf("BlocksUntilConfirmed after reorg error: %v", err)
	}
	if blocks != cfg.ConfirmBlocks {
		t.Errorf("after reorg head=95 < event=100: blocks = %d, want %d (full ConfirmBlocks)", blocks, cfg.ConfirmBlocks)
	}
}
