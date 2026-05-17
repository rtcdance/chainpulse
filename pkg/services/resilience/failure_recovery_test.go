package resilience

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

func TestDefaultRecoveryManager_SaveCheckpoint(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		LastProcessedEventHash:   "hash123",
		LastProcessedTimestamp:   time.Now().Unix(),
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	if rm.currentCheckpoint == nil {
		t.Fatal("currentCheckpoint is nil after save")
	}

	if rm.currentCheckpoint.State.LastProcessedBlockNumber != 100 {
		t.Errorf("expected block number 100, got %d", rm.currentCheckpoint.State.LastProcessedBlockNumber)
	}
}

func TestDefaultRecoveryManager_SaveCheckpoint_InvalidState(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	// Invalid state with negative block number
	state := RecoveryState{
		LastProcessedBlockNumber: -1,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestDefaultRecoveryManager_LoadCheckpoint(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		LastProcessedEventHash:   "hash123",
		LastProcessedTimestamp:   time.Now().Unix(),
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	checkpoint, err := rm.LoadCheckpoint(ctx)
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if checkpoint == nil {
		t.Fatal("checkpoint is nil")
	}

	if checkpoint.State.LastProcessedBlockNumber != 100 {
		t.Errorf("expected block number 100, got %d", checkpoint.State.LastProcessedBlockNumber)
	}
}

func TestDefaultRecoveryManager_LoadCheckpoint_NoCheckpoint(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	checkpoint, err := rm.LoadCheckpoint(ctx)
	if err == nil {
		t.Fatal("expected error when no checkpoint available")
	}

	if checkpoint != nil {
		t.Fatal("checkpoint should be nil")
	}
}

func TestDefaultRecoveryManager_VerifyCheckpoint(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	checkpoint, _ := rm.LoadCheckpoint(ctx)

	err = rm.VerifyCheckpoint(ctx, checkpoint)
	if err != nil {
		t.Fatalf("VerifyCheckpoint failed: %v", err)
	}
}

func TestDefaultRecoveryManager_VerifyCheckpoint_Nil(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	err := rm.VerifyCheckpoint(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil checkpoint")
	}
}

func TestDefaultRecoveryManager_DeleteCheckpoint(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	err = rm.DeleteCheckpoint(ctx)
	if err != nil {
		t.Fatalf("DeleteCheckpoint failed: %v", err)
	}

	if rm.currentCheckpoint != nil {
		t.Fatal("currentCheckpoint should be nil after delete")
	}
}

func TestDefaultRecoveryManager_GetCheckpointHistory(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	// Save multiple checkpoints
	for i := 0; i < 3; i++ {
		state := RecoveryState{
			LastProcessedBlockNumber: int64(100 + i),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]any),
			QueueState:               make(map[string]any),
			DatabaseState:            make(map[string]any),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}
	}

	history, err := rm.GetCheckpointHistory(ctx)
	if err != nil {
		t.Fatalf("GetCheckpointHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 checkpoints, got %d", len(history))
	}
}

func TestDefaultRecoveryManager_Health(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	health := rm.Health(ctx)
	if health.Status != "degraded" {
		t.Errorf("expected degraded status without checkpoint, got %s", health.Status)
	}

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	health = rm.Health(ctx)
	if health.Status != "healthy" {
		t.Errorf("expected healthy status with checkpoint, got %s", health.Status)
	}
}

func TestDefaultRecoveryExecutor_RecoverFromCheckpoint(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	checkpoint, _ := rm.LoadCheckpoint(ctx)

	err = re.RecoverFromCheckpoint(ctx, checkpoint)
	if err != nil {
		t.Fatalf("RecoverFromCheckpoint failed: %v", err)
	}
}

func TestDefaultRecoveryExecutor_RecoverFromCheckpoint_Nil(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	err := re.RecoverFromCheckpoint(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil checkpoint")
	}
}

func TestDefaultRecoveryExecutor_VerifyDataConsistency(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	err = re.VerifyDataConsistency(ctx)
	if err != nil {
		t.Fatalf("VerifyDataConsistency failed: %v", err)
	}
}

func TestDefaultRecoveryExecutor_VerifyDataConsistency_DuplicateEvent(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     map[string]bool{"hash1": true},
		PendingEvents: []core.BlockchainEvent{
			{
				EventHash:   "hash1",
				BlockNumber: 100,
				LogIndex:    0,
			},
		},
		CacheState:    make(map[string]any),
		QueueState:    make(map[string]any),
		DatabaseState: make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	err = re.VerifyDataConsistency(ctx)
	if err == nil {
		t.Fatal("expected error for duplicate event")
	}
}

func TestDefaultRecoveryExecutor_GetRecoveryStats(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	checkpoint, _ := rm.LoadCheckpoint(ctx)
	_ = re.RecoverFromCheckpoint(ctx, checkpoint)

	stats := re.GetRecoveryStats(ctx)
	if stats == nil {
		t.Fatal("stats is nil")
	}

	if stats["recovery_attempts"] != int64(1) {
		t.Errorf("expected 1 recovery attempt, got %v", stats["recovery_attempts"])
	}

	if stats["successful_recoveries"] != int64(1) {
		t.Errorf("expected 1 successful recovery, got %v", stats["successful_recoveries"])
	}
}

func TestDefaultSafeStateManager_EnterSafeState(t *testing.T) {
	t.Parallel()
	ssm := NewDefaultSafeStateManager()
	ctx := context.Background()

	err := ssm.EnterSafeState(ctx, "test reason")
	if err != nil {
		t.Fatalf("EnterSafeState failed: %v", err)
	}

	if !ssm.IsSafeState(ctx) {
		t.Fatal("expected to be in safe state")
	}

	reason := ssm.GetSafeStateReason(ctx)
	if reason != "test reason" {
		t.Errorf("expected reason 'test reason', got '%s'", reason)
	}
}

func TestDefaultSafeStateManager_EnterSafeState_AlreadyInSafeState(t *testing.T) {
	t.Parallel()
	ssm := NewDefaultSafeStateManager()
	ctx := context.Background()

	err := ssm.EnterSafeState(ctx, "reason1")
	if err != nil {
		t.Fatalf("EnterSafeState failed: %v", err)
	}

	err = ssm.EnterSafeState(ctx, "reason2")
	if err == nil {
		t.Fatal("expected error when already in safe state")
	}
}

func TestDefaultSafeStateManager_ExitSafeState(t *testing.T) {
	t.Parallel()
	ssm := NewDefaultSafeStateManager()
	ctx := context.Background()

	err := ssm.EnterSafeState(ctx, "test reason")
	if err != nil {
		t.Fatalf("EnterSafeState failed: %v", err)
	}

	err = ssm.ExitSafeState(ctx)
	if err != nil {
		t.Fatalf("ExitSafeState failed: %v", err)
	}

	if ssm.IsSafeState(ctx) {
		t.Fatal("expected to not be in safe state")
	}
}

func TestDefaultSafeStateManager_ExitSafeState_NotInSafeState(t *testing.T) {
	t.Parallel()
	ssm := NewDefaultSafeStateManager()
	ctx := context.Background()

	err := ssm.ExitSafeState(ctx)
	if err == nil {
		t.Fatal("expected error when not in safe state")
	}
}

func TestRecoveryManager_ConcurrentOperations(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	// Save initial checkpoint
	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]any),
		QueueState:               make(map[string]any),
		DatabaseState:            make(map[string]any),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Concurrent load operations
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := rm.LoadCheckpoint(ctx)
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Fatalf("concurrent LoadCheckpoint failed: %v", err)
		}
	}
}

func TestRecoveryManager_CheckpointHistoryLimit(t *testing.T) {
	t.Parallel()
	rm := NewDefaultRecoveryManager(3, 5*time.Second)
	ctx := context.Background()

	// Save more checkpoints than the limit
	for i := 0; i < 5; i++ {
		state := RecoveryState{
			LastProcessedBlockNumber: int64(100 + i),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]any),
			QueueState:               make(map[string]any),
			DatabaseState:            make(map[string]any),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}
	}

	history, err := rm.GetCheckpointHistory(ctx)
	if err != nil {
		t.Fatalf("GetCheckpointHistory failed: %v", err)
	}

	if len(history) > 3 {
		t.Errorf("expected at most 3 checkpoints, got %d", len(history))
	}
}
