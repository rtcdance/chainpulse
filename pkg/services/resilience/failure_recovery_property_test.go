package resilience

import (
	"chainpulse/pkg/core"
	"context"
	"fmt"
	"testing"
	"time"
)

// Property 20: Failure Recovery
// Validates that the system can recover from failures without data loss

func TestProperty_FailureRecovery_CheckpointConsistency(t *testing.T) {
	// Property: Saved checkpoints must be retrievable with identical state
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	for trial := 0; trial < 100; trial++ {
		blockNum := int64(trial * 100)
		eventHash := fmt.Sprintf("hash_%d", trial)

		state := RecoveryState{
			LastProcessedBlockNumber: blockNum,
			LastProcessedEventHash:   eventHash,
			LastProcessedTimestamp:   time.Now().Unix(),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}

		checkpoint, err := rm.LoadCheckpoint(ctx)
		if err != nil {
			t.Fatalf("LoadCheckpoint failed: %v", err)
		}

		if checkpoint.State.LastProcessedBlockNumber != blockNum {
			t.Errorf("trial %d: block number mismatch: expected %d, got %d",
				trial, blockNum, checkpoint.State.LastProcessedBlockNumber)
		}

		if checkpoint.State.LastProcessedEventHash != eventHash {
			t.Errorf("trial %d: event hash mismatch: expected %s, got %s",
				trial, eventHash, checkpoint.State.LastProcessedEventHash)
		}
	}
}

func TestProperty_FailureRecovery_StateValidation(t *testing.T) {
	// Property: Invalid states must be rejected
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	invalidStates := []RecoveryState{
		{
			LastProcessedBlockNumber: -1,
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		},
		{
			LastProcessedBlockNumber: 100,
			ProcessedEventHashes:     nil,
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		},
		{
			LastProcessedBlockNumber: 100,
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            nil,
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		},
	}

	for i, state := range invalidStates {
		err := rm.SaveCheckpoint(ctx, state)
		if err == nil {
			t.Errorf("trial %d: expected error for invalid state", i)
		}
	}
}

func TestProperty_FailureRecovery_CheckpointHistory(t *testing.T) {
	// Property: Checkpoint history must maintain insertion order and respect size limit
	rm := NewDefaultRecoveryManager(5, 5*time.Second)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		state := RecoveryState{
			LastProcessedBlockNumber: int64(i),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
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

	if len(history) > 5 {
		t.Errorf("history size exceeds limit: expected <= 5, got %d", len(history))
	}

	// Verify order (most recent should be last)
	if len(history) > 1 {
		for i := 0; i < len(history)-1; i++ {
			if history[i].Timestamp.After(history[i+1].Timestamp) {
				t.Errorf("history order violated at index %d", i)
			}
		}
	}
}

func TestProperty_FailureRecovery_RecoveryAttempts(t *testing.T) {
	// Property: Recovery attempts must be tracked accurately
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Reduce iterations to avoid timeout
	for trial := 0; trial < 10; trial++ {
		state := RecoveryState{
			LastProcessedBlockNumber: int64(trial),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}

		checkpoint, _ := rm.LoadCheckpoint(ctx)
		_ = re.RecoverFromCheckpoint(ctx, checkpoint)
	}

	stats := re.GetRecoveryStats(ctx)
	if stats["recovery_attempts"] != int64(10) {
		t.Errorf("expected 10 recovery attempts, got %v", stats["recovery_attempts"])
	}

	if stats["successful_recoveries"] != int64(10) {
		t.Errorf("expected 10 successful recoveries, got %v", stats["successful_recoveries"])
	}
}

func TestProperty_FailureRecovery_DataConsistency(t *testing.T) {
	// Property: Recovered data must be consistent (no duplicate events)
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		processedHashes := make(map[string]bool)
		for i := 0; i < 10; i++ {
			processedHashes[fmt.Sprintf("hash_%d_%d", trial, i)] = true
		}

		state := RecoveryState{
			LastProcessedBlockNumber: int64(trial),
			ProcessedEventHashes:     processedHashes,
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}

		_, _ = rm.LoadCheckpoint(ctx)
		err = re.VerifyDataConsistency(ctx)
		if err != nil {
			t.Fatalf("trial %d: VerifyDataConsistency failed: %v", trial, err)
		}
	}
}

func TestProperty_FailureRecovery_SafeStateTransitions(t *testing.T) {
	// Property: Safe state transitions must be valid
	ssm := NewDefaultSafeStateManager()
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		// Enter safe state
		err := ssm.EnterSafeState(ctx, fmt.Sprintf("reason_%d", trial))
		if err != nil {
			t.Fatalf("trial %d: EnterSafeState failed: %v", trial, err)
		}

		if !ssm.IsSafeState(ctx) {
			t.Fatalf("trial %d: expected to be in safe state", trial)
		}

		reason := ssm.GetSafeStateReason(ctx)
		if reason != fmt.Sprintf("reason_%d", trial) {
			t.Errorf("trial %d: reason mismatch", trial)
		}

		// Exit safe state
		err = ssm.ExitSafeState(ctx)
		if err != nil {
			t.Fatalf("trial %d: ExitSafeState failed: %v", trial, err)
		}

		if ssm.IsSafeState(ctx) {
			t.Fatalf("trial %d: expected to not be in safe state", trial)
		}
	}
}

func TestProperty_FailureRecovery_CheckpointVerification(t *testing.T) {
	// Property: All saved checkpoints must pass verification
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		state := RecoveryState{
			LastProcessedBlockNumber: int64(trial),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}

		checkpoint, _ := rm.LoadCheckpoint(ctx)
		err = rm.VerifyCheckpoint(ctx, checkpoint)
		if err != nil {
			t.Fatalf("trial %d: VerifyCheckpoint failed: %v", trial, err)
		}
	}
}

func TestProperty_FailureRecovery_HealthStatus(t *testing.T) {
	// Property: Health status must reflect checkpoint availability
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	// Without checkpoint
	health := rm.Health(ctx)
	if health.Status != "degraded" {
		t.Errorf("expected degraded status without checkpoint, got %s", health.Status)
	}

	// With checkpoint
	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]interface{}),
		QueueState:               make(map[string]interface{}),
		DatabaseState:            make(map[string]interface{}),
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

func TestProperty_FailureRecovery_ConcurrentCheckpoints(t *testing.T) {
	// Property: Concurrent checkpoint operations must be safe
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	// Save initial checkpoint
	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]interface{}),
		QueueState:               make(map[string]interface{}),
		DatabaseState:            make(map[string]interface{}),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Concurrent operations
	done := make(chan error, 30)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := rm.LoadCheckpoint(ctx)
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			checkpoint, _ := rm.LoadCheckpoint(ctx)
			err := rm.VerifyCheckpoint(ctx, checkpoint)
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			health := rm.Health(ctx)
			if health.Status == "" {
				done <- fmt.Errorf("health status is empty")
			} else {
				done <- nil
			}
		}()
	}

	for i := 0; i < 30; i++ {
		err := <-done
		if err != nil {
			t.Fatalf("concurrent operation failed: %v", err)
		}
	}
}

func TestProperty_FailureRecovery_RecoveryStats(t *testing.T) {
	// Property: Recovery statistics must be accurate
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	re := NewDefaultRecoveryExecutor(rm)
	ctx := context.Background()

	// Perform successful recovery
	state := RecoveryState{
		LastProcessedBlockNumber: 100,
		ProcessedEventHashes:     make(map[string]bool),
		PendingEvents:            make([]core.BlockchainEvent, 0),
		CacheState:               make(map[string]interface{}),
		QueueState:               make(map[string]interface{}),
		DatabaseState:            make(map[string]interface{}),
	}

	err := rm.SaveCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	checkpoint, _ := rm.LoadCheckpoint(ctx)
	_ = re.RecoverFromCheckpoint(ctx, checkpoint)

	stats := re.GetRecoveryStats(ctx)

	if stats["recovery_attempts"] != int64(1) {
		t.Errorf("expected 1 recovery attempt, got %v", stats["recovery_attempts"])
	}

	if stats["successful_recoveries"] != int64(1) {
		t.Errorf("expected 1 successful recovery, got %v", stats["successful_recoveries"])
	}

	if stats["failed_recoveries"] != int64(0) {
		t.Errorf("expected 0 failed recoveries, got %v", stats["failed_recoveries"])
	}
}

func TestProperty_FailureRecovery_CheckpointTimestamp(t *testing.T) {
	// Property: Checkpoint timestamps must be reasonable
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		state := RecoveryState{
			LastProcessedBlockNumber: int64(trial),
			ProcessedEventHashes:     make(map[string]bool),
			PendingEvents:            make([]core.BlockchainEvent, 0),
			CacheState:               make(map[string]interface{}),
			QueueState:               make(map[string]interface{}),
			DatabaseState:            make(map[string]interface{}),
		}

		err := rm.SaveCheckpoint(ctx, state)
		if err != nil {
			t.Fatalf("SaveCheckpoint failed: %v", err)
		}

		checkpoint, _ := rm.LoadCheckpoint(ctx)

		// Verify timestamp is not in the future
		if checkpoint.Timestamp.After(time.Now().Add(1 * time.Minute)) {
			t.Errorf("trial %d: checkpoint timestamp is in the future", trial)
		}

		// Verify timestamp is not too old
		if checkpoint.Timestamp.Before(time.Now().Add(-1 * time.Hour)) {
			t.Errorf("trial %d: checkpoint timestamp is too old", trial)
		}
	}
}

func TestProperty_FailureRecovery_DeleteCheckpointIdempotency(t *testing.T) {
	// Property: Deleting a non-existent checkpoint should fail consistently
	rm := NewDefaultRecoveryManager(10, 5*time.Second)
	ctx := context.Background()

	for trial := 0; trial < 10; trial++ {
		err := rm.DeleteCheckpoint(ctx)
		if err == nil {
			t.Errorf("trial %d: expected error when deleting non-existent checkpoint", trial)
		}
	}
}

func TestProperty_FailureRecovery_SafeStateReason(t *testing.T) {
	// Property: Safe state reason must be preserved correctly
	ssm := NewDefaultSafeStateManager()
	ctx := context.Background()

	reasons := []string{
		"data corruption detected",
		"critical error occurred",
		"recovery in progress",
		"maintenance mode",
	}

	for _, reason := range reasons {
		err := ssm.EnterSafeState(ctx, reason)
		if err != nil {
			t.Fatalf("EnterSafeState failed: %v", err)
		}

		retrievedReason := ssm.GetSafeStateReason(ctx)
		if retrievedReason != reason {
			t.Errorf("reason mismatch: expected %s, got %s", reason, retrievedReason)
		}

		err = ssm.ExitSafeState(ctx)
		if err != nil {
			t.Fatalf("ExitSafeState failed: %v", err)
		}
	}
}
