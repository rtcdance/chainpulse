package resilience

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// RecoveryState represents the state that can be persisted and recovered
type RecoveryState struct {
	// Checkpoint data
	LastProcessedBlockNumber int64                  `json:"last_processed_block_number"`
	LastProcessedEventHash   string                 `json:"last_processed_event_hash"`
	LastProcessedTimestamp   int64                  `json:"last_processed_timestamp"`
	ProcessedEventHashes     map[string]bool        `json:"processed_event_hashes"`
	PendingEvents            []core.BlockchainEvent `json:"pending_events"`
	CacheState               map[string]any         `json:"cache_state"`
	QueueState               map[string]any         `json:"queue_state"`
	DatabaseState            map[string]any         `json:"database_state"`
	Timestamp                int64                  `json:"timestamp"`
	Version                  int                    `json:"version"`
}

// RecoveryCheckpoint represents a saved checkpoint
type RecoveryCheckpoint struct {
	State     RecoveryState
	Timestamp time.Time
	Valid     bool
}

// RecoveryManager manages state persistence and recovery
type RecoveryManager interface {
	// SaveCheckpoint saves the current state
	SaveCheckpoint(ctx context.Context, state RecoveryState) error

	// LoadCheckpoint loads the last saved state
	LoadCheckpoint(ctx context.Context) (*RecoveryCheckpoint, error)

	// VerifyCheckpoint verifies checkpoint integrity
	VerifyCheckpoint(ctx context.Context, checkpoint *RecoveryCheckpoint) error

	// DeleteCheckpoint deletes a checkpoint
	DeleteCheckpoint(ctx context.Context) error

	// GetCheckpointHistory returns the history of checkpoints
	GetCheckpointHistory(ctx context.Context) ([]RecoveryCheckpoint, error)

	// Health returns the health status of the recovery manager
	Health(ctx context.Context) core.HealthStatus
}

// DefaultRecoveryManager implements RecoveryManager
type DefaultRecoveryManager struct {
	mu                    sync.RWMutex
	currentCheckpoint     *RecoveryCheckpoint
	checkpointHistory     []RecoveryCheckpoint
	maxHistorySize        int
	checkpointInterval    time.Duration
	lastCheckpointTime    time.Time
	checksumVerification  bool
	operationCount        int64
	successCount          int64
	errorCount            int64
	lastError             error
	lastErrorTime         time.Time
	recoveryAttempts      int64
	successfulRecoveries  int64
	failedRecoveries      int64
	dataConsistencyErrors int64
}

// NewDefaultRecoveryManager creates a new recovery manager
func NewDefaultRecoveryManager(maxHistorySize int, checkpointInterval time.Duration) *DefaultRecoveryManager {
	return &DefaultRecoveryManager{
		maxHistorySize:       maxHistorySize,
		checkpointInterval:   checkpointInterval,
		checksumVerification: true,
		checkpointHistory:    make([]RecoveryCheckpoint, 0, maxHistorySize),
	}
}

// SaveCheckpoint saves the current state
func (rm *DefaultRecoveryManager) SaveCheckpoint(ctx context.Context, state RecoveryState) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	atomic.AddInt64(&rm.operationCount, 1)

	// Validate state
	if err := rm.validateState(state); err != nil {
		atomic.AddInt64(&rm.errorCount, 1)
		rm.lastError = err
		rm.lastErrorTime = time.Now()
		return fmt.Errorf("invalid state: %w", err)
	}

	// Set timestamp
	state.Timestamp = time.Now().Unix()
	state.Version = 1

	// Create checkpoint
	checkpoint := RecoveryCheckpoint{
		State:     state,
		Timestamp: time.Now(),
		Valid:     true,
	}

	// Verify checkpoint
	if rm.checksumVerification {
		if err := rm.verifyCheckpointInternal(checkpoint); err != nil {
			atomic.AddInt64(&rm.errorCount, 1)
			rm.lastError = err
			rm.lastErrorTime = time.Now()
			return fmt.Errorf("checkpoint verification failed: %w", err)
		}
	}

	// Save checkpoint
	rm.currentCheckpoint = &checkpoint
	rm.lastCheckpointTime = time.Now()

	// Add to history
	rm.checkpointHistory = append(rm.checkpointHistory, checkpoint)
	if len(rm.checkpointHistory) > rm.maxHistorySize {
		rm.checkpointHistory = rm.checkpointHistory[1:]
	}

	atomic.AddInt64(&rm.successCount, 1)
	return nil
}

// LoadCheckpoint loads the last saved state
func (rm *DefaultRecoveryManager) LoadCheckpoint(ctx context.Context) (*RecoveryCheckpoint, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	atomic.AddInt64(&rm.operationCount, 1)

	if rm.currentCheckpoint == nil {
		return nil, fmt.Errorf("no checkpoint available")
	}

	// Verify checkpoint before returning
	if err := rm.verifyCheckpointInternal(*rm.currentCheckpoint); err != nil {
		atomic.AddInt64(&rm.errorCount, 1)
		rm.lastError = err
		rm.lastErrorTime = time.Now()
		return nil, fmt.Errorf("checkpoint verification failed: %w", err)
	}

	atomic.AddInt64(&rm.successCount, 1)
	return rm.currentCheckpoint, nil
}

// VerifyCheckpoint verifies checkpoint integrity
func (rm *DefaultRecoveryManager) VerifyCheckpoint(ctx context.Context, checkpoint *RecoveryCheckpoint) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	atomic.AddInt64(&rm.operationCount, 1)

	if checkpoint == nil {
		atomic.AddInt64(&rm.errorCount, 1)
		return fmt.Errorf("checkpoint is nil")
	}

	if err := rm.verifyCheckpointInternal(*checkpoint); err != nil {
		atomic.AddInt64(&rm.errorCount, 1)
		rm.lastError = err
		rm.lastErrorTime = time.Now()
		return fmt.Errorf("verify checkpoint internal: %w", err)
	}

	atomic.AddInt64(&rm.successCount, 1)
	return nil
}

// DeleteCheckpoint deletes a checkpoint
func (rm *DefaultRecoveryManager) DeleteCheckpoint(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	atomic.AddInt64(&rm.operationCount, 1)

	if rm.currentCheckpoint == nil {
		atomic.AddInt64(&rm.errorCount, 1)
		return fmt.Errorf("no checkpoint to delete")
	}

	rm.currentCheckpoint = nil
	rm.checkpointHistory = make([]RecoveryCheckpoint, 0, rm.maxHistorySize)

	atomic.AddInt64(&rm.successCount, 1)
	return nil
}

// GetCheckpointHistory returns the history of checkpoints
func (rm *DefaultRecoveryManager) GetCheckpointHistory(ctx context.Context) ([]RecoveryCheckpoint, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	atomic.AddInt64(&rm.operationCount, 1)

	if len(rm.checkpointHistory) == 0 {
		atomic.AddInt64(&rm.errorCount, 1)
		return nil, fmt.Errorf("no checkpoint history available")
	}

	// Return a copy of the history
	history := make([]RecoveryCheckpoint, len(rm.checkpointHistory))
	copy(history, rm.checkpointHistory)

	atomic.AddInt64(&rm.successCount, 1)
	return history, nil
}

// Health returns the health status of the recovery manager
func (rm *DefaultRecoveryManager) Health(ctx context.Context) core.HealthStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	status := core.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]any),
	}

	// Check if checkpoint is available
	if rm.currentCheckpoint == nil {
		status.Status = "degraded"
		status.Details["checkpoint_available"] = false
	} else {
		status.Details["checkpoint_available"] = true
		status.Details["last_checkpoint_time"] = rm.lastCheckpointTime
	}

	// Add statistics
	status.Details["operation_count"] = atomic.LoadInt64(&rm.operationCount)
	status.Details["success_count"] = atomic.LoadInt64(&rm.successCount)
	status.Details["error_count"] = atomic.LoadInt64(&rm.errorCount)
	status.Details["recovery_attempts"] = atomic.LoadInt64(&rm.recoveryAttempts)
	status.Details["successful_recoveries"] = atomic.LoadInt64(&rm.successfulRecoveries)
	status.Details["failed_recoveries"] = atomic.LoadInt64(&rm.failedRecoveries)
	status.Details["data_consistency_errors"] = atomic.LoadInt64(&rm.dataConsistencyErrors)
	status.Details["checkpoint_history_size"] = len(rm.checkpointHistory)

	if rm.lastError != nil {
		status.Details["last_error"] = rm.lastError.Error()
		status.Details["last_error_time"] = rm.lastErrorTime
	}

	return status
}

// validateState validates the recovery state
func (rm *DefaultRecoveryManager) validateState(state RecoveryState) error {
	if state.LastProcessedBlockNumber < 0 {
		return fmt.Errorf("invalid block number: %d", state.LastProcessedBlockNumber)
	}

	if state.ProcessedEventHashes == nil {
		return fmt.Errorf("processed event hashes is nil")
	}

	if state.PendingEvents == nil {
		return fmt.Errorf("pending events is nil")
	}

	if state.CacheState == nil {
		return fmt.Errorf("cache state is nil")
	}

	if state.QueueState == nil {
		return fmt.Errorf("queue state is nil")
	}

	if state.DatabaseState == nil {
		return fmt.Errorf("database state is nil")
	}

	return nil
}

// verifyCheckpointInternal verifies checkpoint integrity
func (rm *DefaultRecoveryManager) verifyCheckpointInternal(checkpoint RecoveryCheckpoint) error {
	if !checkpoint.Valid {
		return fmt.Errorf("checkpoint is marked as invalid")
	}

	// Verify state can be marshaled
	_, err := json.Marshal(checkpoint.State)
	if err != nil {
		return fmt.Errorf("checkpoint state cannot be marshaled: %w", err)
	}

	// Verify timestamp is reasonable
	if checkpoint.Timestamp.After(time.Now().Add(1 * time.Minute)) {
		return fmt.Errorf("checkpoint timestamp is in the future")
	}

	return nil
}

// RecoveryExecutor executes recovery operations
type RecoveryExecutor interface {
	// RecoverFromCheckpoint recovers the system from a checkpoint
	RecoverFromCheckpoint(ctx context.Context, checkpoint *RecoveryCheckpoint) error

	// VerifyDataConsistency verifies data consistency after recovery
	VerifyDataConsistency(ctx context.Context) error

	// GetRecoveryStats returns recovery statistics
	GetRecoveryStats(ctx context.Context) map[string]any
}

// DefaultRecoveryExecutor implements RecoveryExecutor
type DefaultRecoveryExecutor struct {
	mu                    sync.RWMutex
	recoveryManager       RecoveryManager
	recoveryAttempts      int64
	successfulRecoveries  int64
	failedRecoveries      int64
	dataConsistencyErrors int64
	lastRecoveryTime      time.Time
	lastRecoveryError     error
}

// NewDefaultRecoveryExecutor creates a new recovery executor
func NewDefaultRecoveryExecutor(recoveryManager RecoveryManager) *DefaultRecoveryExecutor {
	return &DefaultRecoveryExecutor{
		recoveryManager: recoveryManager,
	}
}

// RecoverFromCheckpoint recovers the system from a checkpoint
func (re *DefaultRecoveryExecutor) RecoverFromCheckpoint(ctx context.Context, checkpoint *RecoveryCheckpoint) error {
	atomic.AddInt64(&re.recoveryAttempts, 1)

	if checkpoint == nil {
		atomic.AddInt64(&re.failedRecoveries, 1)
		err := fmt.Errorf("checkpoint is nil")
		re.mu.Lock()
		re.lastRecoveryError = err
		re.lastRecoveryTime = time.Now()
		re.mu.Unlock()
		return fmt.Errorf("recover from checkpoint: %w", err)
	}

	// Verify checkpoint
	if err := re.recoveryManager.VerifyCheckpoint(ctx, checkpoint); err != nil {
		atomic.AddInt64(&re.failedRecoveries, 1)
		re.mu.Lock()
		re.lastRecoveryError = err
		re.lastRecoveryTime = time.Now()
		re.mu.Unlock()
		return fmt.Errorf("checkpoint verification failed: %w", err)
	}

	// Recover state
	state := checkpoint.State

	// Validate recovered state
	if state.LastProcessedBlockNumber < 0 {
		atomic.AddInt64(&re.failedRecoveries, 1)
		err := fmt.Errorf("invalid recovered block number: %d", state.LastProcessedBlockNumber)
		re.mu.Lock()
		re.lastRecoveryError = err
		re.lastRecoveryTime = time.Now()
		re.mu.Unlock()
		return fmt.Errorf("recover from checkpoint: %w", err)
	}

	// Verify data consistency (without holding lock to avoid deadlock)
	if err := re.verifyDataConsistencyUnlocked(ctx); err != nil {
		atomic.AddInt64(&re.dataConsistencyErrors, 1)
		atomic.AddInt64(&re.failedRecoveries, 1)
		re.mu.Lock()
		re.lastRecoveryError = err
		re.lastRecoveryTime = time.Now()
		re.mu.Unlock()
		return fmt.Errorf("data consistency verification failed: %w", err)
	}

	re.mu.Lock()
	re.lastRecoveryTime = time.Now()
	re.mu.Unlock()
	atomic.AddInt64(&re.successfulRecoveries, 1)
	return nil
}

// VerifyDataConsistency verifies data consistency after recovery
func (re *DefaultRecoveryExecutor) VerifyDataConsistency(ctx context.Context) error {
	re.mu.RLock()
	defer re.mu.RUnlock()

	return re.verifyDataConsistencyUnlocked(ctx)
}

// verifyDataConsistencyUnlocked verifies data consistency without holding lock
func (re *DefaultRecoveryExecutor) verifyDataConsistencyUnlocked(ctx context.Context) error {
	// Load checkpoint to verify consistency
	checkpoint, err := re.recoveryManager.LoadCheckpoint(ctx)
	if err != nil {
		atomic.AddInt64(&re.dataConsistencyErrors, 1)
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if checkpoint == nil {
		atomic.AddInt64(&re.dataConsistencyErrors, 1)
		return fmt.Errorf("checkpoint is nil")
	}

	// Verify state consistency
	state := checkpoint.State

	// Check for duplicate events
	if len(state.ProcessedEventHashes) > 0 && len(state.PendingEvents) > 0 {
		for _, event := range state.PendingEvents {
			if _, exists := state.ProcessedEventHashes[event.EventHash]; exists {
				atomic.AddInt64(&re.dataConsistencyErrors, 1)
				return fmt.Errorf("duplicate event detected: %s", event.EventHash)
			}
		}
	}

	// Verify timestamp consistency
	if state.Timestamp > time.Now().Unix() {
		atomic.AddInt64(&re.dataConsistencyErrors, 1)
		return fmt.Errorf("invalid timestamp: %d", state.Timestamp)
	}

	return nil
}

// GetRecoveryStats returns recovery statistics
func (re *DefaultRecoveryExecutor) GetRecoveryStats(ctx context.Context) map[string]any {
	re.mu.RLock()
	defer re.mu.RUnlock()

	stats := map[string]any{
		"recovery_attempts":       atomic.LoadInt64(&re.recoveryAttempts),
		"successful_recoveries":   atomic.LoadInt64(&re.successfulRecoveries),
		"failed_recoveries":       atomic.LoadInt64(&re.failedRecoveries),
		"data_consistency_errors": atomic.LoadInt64(&re.dataConsistencyErrors),
		"last_recovery_time":      re.lastRecoveryTime,
	}

	if re.lastRecoveryError != nil {
		stats["last_recovery_error"] = re.lastRecoveryError.Error()
	}

	return stats
}

// SafeStateManager manages safe state transitions
type SafeStateManager interface {
	// EnterSafeState enters a safe state to prevent data corruption
	EnterSafeState(ctx context.Context, reason string) error

	// ExitSafeState exits the safe state
	ExitSafeState(ctx context.Context) error

	// IsSafeState returns whether the system is in safe state
	IsSafeState(ctx context.Context) bool

	// GetSafeStateReason returns the reason for entering safe state
	GetSafeStateReason(ctx context.Context) string
}

// DefaultSafeStateManager implements SafeStateManager
type DefaultSafeStateManager struct {
	mu              sync.RWMutex
	inSafeState     bool
	safeStateReason string
	safeStateTime   time.Time
	exitTime        time.Time
}

// NewDefaultSafeStateManager creates a new safe state manager
func NewDefaultSafeStateManager() *DefaultSafeStateManager {
	return &DefaultSafeStateManager{}
}

// EnterSafeState enters a safe state to prevent data corruption
func (ssm *DefaultSafeStateManager) EnterSafeState(ctx context.Context, reason string) error {
	ssm.mu.Lock()
	defer ssm.mu.Unlock()

	if ssm.inSafeState {
		return fmt.Errorf("already in safe state: %s", ssm.safeStateReason)
	}

	ssm.inSafeState = true
	ssm.safeStateReason = reason
	ssm.safeStateTime = time.Now()

	return nil
}

// ExitSafeState exits the safe state
func (ssm *DefaultSafeStateManager) ExitSafeState(ctx context.Context) error {
	ssm.mu.Lock()
	defer ssm.mu.Unlock()

	if !ssm.inSafeState {
		return fmt.Errorf("not in safe state")
	}

	ssm.inSafeState = false
	ssm.safeStateReason = ""
	ssm.exitTime = time.Now()

	return nil
}

// IsSafeState returns whether the system is in safe state
func (ssm *DefaultSafeStateManager) IsSafeState(ctx context.Context) bool {
	ssm.mu.RLock()
	defer ssm.mu.RUnlock()

	return ssm.inSafeState
}

// GetSafeStateReason returns the reason for entering safe state
func (ssm *DefaultSafeStateManager) GetSafeStateReason(ctx context.Context) string {
	ssm.mu.RLock()
	defer ssm.mu.RUnlock()

	return ssm.safeStateReason
}
