package consensus

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfirmationConfig(t *testing.T) {
	t.Parallel()

	c := DefaultConfirmationConfig()

	if c.ConfirmBlocks != DefaultConfirmBlocks {
		t.Errorf("ConfirmBlocks = %d, want %d", c.ConfirmBlocks, DefaultConfirmBlocks)
	}
	if c.FinalizeEpochs != DefaultFinalizeEpochs {
		t.Errorf("FinalizeEpochs = %d, want %d", c.FinalizeEpochs, DefaultFinalizeEpochs)
	}
	if c.SlotsPerEpoch != SlotsPerEpoch {
		t.Errorf("SlotsPerEpoch = %d, want %d", c.SlotsPerEpoch, SlotsPerEpoch)
	}
	if c.SlotDurationSec != int64(SlotDuration.Seconds()) {
		t.Errorf("SlotDurationSec = %d", c.SlotDurationSec)
	}
}

func TestConfirmationConfig_BlocksToFinalize(t *testing.T) {
	t.Parallel()

	c := ConfirmationConfig{FinalizeEpochs: 2, SlotsPerEpoch: 32}
	if got := c.BlocksToFinalize(); got != 64 {
		t.Errorf("BlocksToFinalize() = %d, want 64", got)
	}

	c2 := ConfirmationConfig{FinalizeEpochs: 3, SlotsPerEpoch: 32}
	if got := c2.BlocksToFinalize(); got != 96 {
		t.Errorf("BlocksToFinalize() = %d, want 96", got)
	}
}

func TestConfirmationConfig_TimeToConfirm(t *testing.T) {
	t.Parallel()

	c := ConfirmationConfig{ConfirmBlocks: 12, SlotDurationSec: 12}
	want := 12 * 12 * time.Second
	if got := c.TimeToConfirm(); got != want {
		t.Errorf("TimeToConfirm() = %v, want %v", got, want)
	}
}

func TestConfirmationConfig_TimeToFinalize(t *testing.T) {
	t.Parallel()

	c := ConfirmationConfig{FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	want := 2 * 32 * 12 * time.Second
	if got := c.TimeToFinalize(); got != want {
		t.Errorf("TimeToFinalize() = %v, want %v", got, want)
	}
}

func TestNewConfirmationTracker(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	if ct == nil {
		t.Fatal("NewConfirmationTracker returned nil")
	}
	if ct.chainID != "ethereum" {
		t.Errorf("chainID = %s, want ethereum", ct.chainID)
	}
	if ct.pending == nil {
		t.Error("pending map is nil")
	}
}

func TestConfirmationTracker_SetFinalityChecker(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	if ct.finalityChecker != nil {
		t.Error("finalityChecker should be nil initially")
	}

	mockFC := &mockFinalityChecker{}
	ct.SetFinalityChecker(mockFC)

	if fc, ok := ct.finalityChecker.(*mockFinalityChecker); !ok || fc != mockFC {
		t.Error("finalityChecker was not set correctly")
	}
}

func TestConfirmationTracker_Stop(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")
	ct.Stop()
	ct.Stop()
}

func TestConfirmationTracker_Track(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 100, "blockhash1")
	ct.Track("hash2", 101, "blockhash2")
	ct.Track("hash1", 105, "blockhash3")

	if ct.TotalTracked() != 2 {
		t.Errorf("TotalTracked = %d, want 2", ct.TotalTracked())
	}
}

func TestConfirmationTracker_GetStatus(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 100, "blockhash1")

	status := ct.GetStatus("hash1")
	if status != "pending" {
		t.Errorf("GetStatus = %s, want pending", status)
	}

	statusUnknown := ct.GetStatus("nonexistent")
	if statusUnknown != "pending" {
		t.Errorf("GetStatus(unknown) = %s, want pending", statusUnknown)
	}
}

func TestConfirmationTracker_AdvanceBlock_CompleteLifecycle(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 0, "blockhash1")

	if ct.PendingCount() != 1 {
		t.Errorf("PendingCount = %d, want 1", ct.PendingCount())
	}

	transitions := ct.AdvanceBlock(12)
	if len(transitions) != 1 {
		t.Errorf("expected 1 transition, got %d", len(transitions))
	}

	if ct.GetStatus("hash1") != "confirmed" {
		t.Errorf("status = %s, want confirmed", ct.GetStatus("hash1"))
	}
	if ct.ConfirmedCount() != 1 {
		t.Errorf("ConfirmedCount = %d, want 1", ct.ConfirmedCount())
	}
	if ct.PendingCount() != 0 {
		t.Errorf("PendingCount = %d, want 0", ct.PendingCount())
	}

	transitions = ct.AdvanceBlock(64)
	if len(transitions) != 1 {
		t.Errorf("expected 1 finalization transition, got %d", len(transitions))
	}

	if ct.GetStatus("hash1") != "finalized" {
		t.Errorf("status = %s, want finalized", ct.GetStatus("hash1"))
	}
	if ct.FinalizedCount() != 1 {
		t.Errorf("FinalizedCount = %d, want 1", ct.FinalizedCount())
	}
}

func TestConfirmationTracker_AdvanceBlock_NotEnoughBlocks(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 0, "blockhash1")
	ct.AdvanceBlock(5)

	if ct.PendingCount() != 1 {
		t.Errorf("PendingCount = %d, want 1", ct.PendingCount())
	}
	if ct.GetStatus("hash1") != "pending" {
		t.Errorf("status = %s, want pending", ct.GetStatus("hash1"))
	}
}

func TestConfirmationTracker_RemoveFinalized(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 0, "blockhash1")
	ct.Track("hash2", 0, "blockhash2")
	ct.AdvanceBlock(64)

	removed := ct.RemoveFinalized()
	if removed != 2 {
		t.Errorf("RemoveFinalized = %d, want 2", removed)
	}
	if ct.TotalTracked() != 0 {
		t.Errorf("TotalTracked = %d, want 0", ct.TotalTracked())
	}
}

func TestConfirmationTracker_MarkReorged(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 100, "blockhash_A")
	ct.Track("hash2", 101, "blockhash_A")
	ct.Track("hash3", 102, "blockhash_B")

	reorged := ct.MarkReorged("blockhash_A")
	if reorged != 2 {
		t.Errorf("MarkReorged = %d, want 2", reorged)
	}
	if ct.TotalTracked() != 1 {
		t.Errorf("TotalTracked = %d, want 1", ct.TotalTracked())
	}
}

func TestConfirmationTracker_BlocksUntilConfirmed(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 100, "blockhash")

	ct.AdvanceBlock(105)
	remaining, err := ct.BlocksUntilConfirmed("hash1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != 7 {
		t.Errorf("BlocksUntilConfirmed = %d, want 7", remaining)
	}

	ct.AdvanceBlock(112)
	remaining, err = ct.BlocksUntilConfirmed("hash1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != 0 {
		t.Errorf("BlocksUntilConfirmed = %d, want 0 (confirmed)", remaining)
	}

	_, err = ct.BlocksUntilConfirmed("nonexistent")
	if err == nil {
		t.Error("expected error for unknown event")
	}
}

func TestConfirmationTracker_PersistAndLoad(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	ct.Track("hash1", 100, "blockhash1")
	ct.Track("hash2", 200, "blockhash2")

	data, err := ct.Persist()
	if err != nil {
		t.Fatalf("Persist failed: %v", err)
	}

	ct2 := NewConfirmationTracker(config, "ethereum")
	err = ct2.Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if ct2.TotalTracked() != 2 {
		t.Errorf("TotalTracked = %d, want 2", ct2.TotalTracked())
	}
	if ct2.GetStatus("hash1") != "pending" {
		t.Errorf("GetStatus(hash1) = %s, want pending", ct2.GetStatus("hash1"))
	}
}

func TestConfirmationTracker_PersistEmpty(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	data, err := ct.Persist()
	if err != nil {
		t.Fatalf("Persist empty failed: %v", err)
	}

	err = ct.Load(data)
	if err != nil {
		t.Fatalf("Load empty failed: %v", err)
	}
}

func TestConfirmationTracker_ReconcileFinality_NoChecker(t *testing.T) {
	t.Parallel()

	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	_, err := ct.ReconcileFinality()
	if err == nil {
		t.Error("expected error when no finality checker configured")
	}
}

type mockFinalityChecker struct {
	finalizedBlock uint64
	err            error
}

func (m *mockFinalityChecker) GetFinalizedBlockNumber(_ context.Context, _ string) (uint64, error) {
	return m.finalizedBlock, m.err
}

func (m *mockFinalityChecker) IsBlockFinalized(_ context.Context, _ string, blockNumber uint64) (bool, error) {
	return blockNumber <= m.finalizedBlock, m.err
}

func TestConfirmationTracker_ReconcileFinality_Success(t *testing.T) {
	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")
	ct.SetFinalityChecker(&mockFinalityChecker{finalizedBlock: 150})

	ct.Track("hash1", 100, "blockhash1")
	ct.Track("hash2", 200, "blockhash2")

	promoted, err := ct.ReconcileFinality()
	if err != nil {
		t.Fatalf("ReconcileFinality failed: %v", err)
	}
	if promoted != 1 {
		t.Errorf("promoted = %d, want 1", promoted)
	}
	if ct.FinalizedCount() != 1 {
		t.Errorf("FinalizedCount = %d, want 1", ct.FinalizedCount())
	}
}

func TestConfirmationTracker_OnConfirmedCallback(t *testing.T) {
	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	var confirmedHash string
	ct.OnConfirmed = func(eventHash string) {
		confirmedHash = eventHash
	}

	ct.Track("hash1", 0, "blockhash1")
	ct.AdvanceBlock(12)

	if confirmedHash != "hash1" {
		t.Errorf("OnConfirmed hash = %s, want hash1", confirmedHash)
	}
}

func TestConfirmationTracker_OnFinalizedCallback(t *testing.T) {
	config := ConfirmationConfig{ConfirmBlocks: 12, FinalizeEpochs: 2, SlotsPerEpoch: 32, SlotDurationSec: 12}
	ct := NewConfirmationTracker(config, "ethereum")

	var finalizedHash string
	ct.OnFinalized = func(eventHash string) {
		finalizedHash = eventHash
	}

	ct.Track("hash1", 0, "blockhash1")
	ct.AdvanceBlock(64)

	if finalizedHash != "hash1" {
		t.Errorf("OnFinalized hash = %s, want hash1", finalizedHash)
	}
}
