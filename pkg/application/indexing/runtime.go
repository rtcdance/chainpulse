package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// EventEnvelope, Checkpoint, ProcessingFailure, and RuntimeStatus are type aliases
// for core definitions. They were moved to pkg/core to prevent domain-layer
// packages from depending on application/indexing (DDD layer inversion).
// These aliases preserve backward compatibility for existing importers.
//
// New code should prefer importing from core directly.
type (
	EventEnvelope     = core.EventEnvelope
	Checkpoint        = core.Checkpoint
	ProcessingFailure = core.ProcessingFailure
	RuntimeStatus     = core.RuntimeStatus
)

// EventSource emits normalized indexing envelopes from puller or replay flows.
type EventSource interface {
	Fetch(ctx context.Context, chainID string) ([]EventEnvelope, error)
}

// EventSink persists accepted envelopes and derived metadata.
type EventSink interface {
	Persist(ctx context.Context, events []EventEnvelope) error
}

// CheckpointStore loads and saves runtime checkpoints.
type CheckpointStore interface {
	Load(ctx context.Context, chainID string) (Checkpoint, error)
	Save(ctx context.Context, checkpoint Checkpoint) error
}

// IdempotencyStore prevents duplicate processing of stable event keys.
type IdempotencyStore interface {
	IsProcessed(ctx context.Context, eventKey string) (bool, error)
	MarkProcessed(ctx context.Context, eventKey string) error
}

// FailureRouter routes failed envelopes to retry or DLQ paths.
type FailureRouter interface {
	Route(ctx context.Context, failure ProcessingFailure, event EventEnvelope) error
}

// ReplaySource provides historical envelopes from checkpoint/DLQ recovery flow.
type ReplaySource interface {
	Replay(ctx context.Context, chainID string, from Checkpoint) ([]EventEnvelope, error)
}

// ReplayRangeSource provides bounded replay for operator-triggered DLQ replay.
type ReplayRangeSource interface {
	ReplayRange(ctx context.Context, chainID string, from, to Checkpoint, limit int) ([]EventEnvelope, error)
}

// ReplayAcknowledger removes replayed events from the underlying replay source
// once processing succeeds.
type ReplayAcknowledger interface {
	AcknowledgeReplay(ctx context.Context, chainID string, events []EventEnvelope) error
}

// Runtime exposes the additive lifecycle shared by monolith and microservices.
type Runtime interface {
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) (core.HealthStatus, error)
	Status() RuntimeStatus
}

// SharedRuntime is the first shared indexing runtime skeleton for later parity
// wiring between deployment modes.
type SharedRuntime struct {
	logger       core.Logger
	source       EventSource
	sink         EventSink
	checkpoints  CheckpointStore
	idempotency  IdempotencyStore
	failureRoute FailureRouter
	replay       ReplaySource
	chains       []string

	mu     sync.RWMutex
	status RuntimeStatus
}

// RuntimeDeps groups the required runtime ports.
type RuntimeDeps struct {
	Logger          core.Logger
	Source          EventSource
	Sink            EventSink
	CheckpointStore CheckpointStore
	Idempotency     IdempotencyStore
	FailureRouter   FailureRouter
	ReplaySource    ReplaySource
	Chains          []string
}

// NewSharedRuntime creates a shared indexing runtime skeleton.
func NewSharedRuntime(deps RuntimeDeps) (*SharedRuntime, error) {
	if deps.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if deps.Source == nil {
		return nil, fmt.Errorf("event source is required")
	}
	if deps.Sink == nil {
		return nil, fmt.Errorf("event sink is required")
	}
	if deps.CheckpointStore == nil {
		return nil, fmt.Errorf("checkpoint store is required")
	}
	if deps.Idempotency == nil {
		return nil, fmt.Errorf("idempotency store is required")
	}
	if deps.FailureRouter == nil {
		return nil, fmt.Errorf("failure router is required")
	}
	if len(deps.Chains) == 0 {
		return nil, fmt.Errorf("at least one chain is required")
	}

	now := time.Now()
	return &SharedRuntime{
		logger:       deps.Logger,
		source:       deps.Source,
		sink:         deps.Sink,
		checkpoints:  deps.CheckpointStore,
		idempotency:  deps.Idempotency,
		failureRoute: deps.FailureRouter,
		replay:       deps.ReplaySource,
		chains:       append([]string(nil), deps.Chains...),
		status: RuntimeStatus{
			State:                 "created",
			Chains:                append([]string(nil), deps.Chains...),
			CheckpointingEnabled:  deps.CheckpointStore != nil,
			IdempotencyEnabled:    deps.Idempotency != nil,
			FailureRoutingEnabled: deps.FailureRouter != nil,
			ReplayEnabled:         deps.ReplaySource != nil,
			RecoveryState:         "recovery-unobserved",
			LastUpdatedAt:         now,
		},
	}, nil
}

// Initialize marks runtime readiness for later mode-specific wiring.
func (rt *SharedRuntime) Initialize(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.status.Initialized = true
	rt.status.State = "initialized"
	rt.status.LastUpdatedAt = time.Now()
	return nil
}

// Start marks the runtime as started after initialization.
func (rt *SharedRuntime) Start(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	if !rt.status.Initialized {
		return fmt.Errorf("runtime must be initialized before start")
	}
	rt.status.Started = true
	rt.status.State = "running"
	rt.status.LastUpdatedAt = time.Now()
	return nil
}

// Stop marks runtime shutdown without tearing down ports yet.
func (rt *SharedRuntime) Stop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.status.Started = false
	rt.status.State = "stopped"
	rt.status.LastUpdatedAt = time.Now()
	return nil
}

// Health reports runtime lifecycle health only in this additive phase.
func (rt *SharedRuntime) Health(ctx context.Context) (core.HealthStatus, error) {
	select {
	case <-ctx.Done():
		return core.HealthStatus{}, ctx.Err()
	default:
	}

	rt.mu.RLock()
	defer rt.mu.RUnlock()

	status := "degraded"
	message := "runtime created but not started"
	if rt.status.Started {
		status = "healthy"
		message = "runtime running"
	} else if rt.status.Initialized {
		message = "runtime initialized but not started"
	}

	return core.HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details: map[string]any{
			"state":                      rt.status.State,
			"chains":                     append([]string(nil), rt.status.Chains...),
			"initialized":                rt.status.Initialized,
			"started":                    rt.status.Started,
			"checkpointing_enabled":      rt.status.CheckpointingEnabled,
			"idempotency_enabled":        rt.status.IdempotencyEnabled,
			"failure_routing_enabled":    rt.status.FailureRoutingEnabled,
			"replay_enabled":             rt.status.ReplayEnabled,
			"recovery_state":             rt.status.RecoveryState,
			"recovery_runs":              rt.status.RecoveryRuns,
			"recovery_failures":          rt.status.RecoveryFailures,
			"recovery_checkpoint_loads":  rt.status.RecoveryCheckpointLoads,
			"recovery_replayed_events":   rt.status.RecoveryReplayedEvents,
			"last_recovery_chain_id":     rt.status.LastRecoveryChainID,
			"last_recovery_cursor":       rt.status.LastRecoveryCursor,
			"last_recovery_block":        rt.status.LastRecoveryBlock,
			"last_recovery_replay_count": rt.status.LastRecoveryReplayCount,
			"last_recovery_error":        rt.status.LastRecoveryError,
		},
	}, nil
}

// Status returns a copy of runtime status.
func (rt *SharedRuntime) Status() RuntimeStatus {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	status := rt.status
	status.Chains = append([]string(nil), rt.status.Chains...)
	return status
}

// LoadCheckpoint loads the stored checkpoint for one chain after runtime
// initialization.
func (rt *SharedRuntime) LoadCheckpoint(ctx context.Context, chainID string) (Checkpoint, error) {
	select {
	case <-ctx.Done():
		return Checkpoint{}, ctx.Err()
	default:
	}

	if chainID == "" {
		return Checkpoint{}, fmt.Errorf("chain ID is required")
	}

	rt.mu.RLock()
	initialized := rt.status.Initialized
	rt.mu.RUnlock()
	if !initialized {
		return Checkpoint{}, fmt.Errorf("runtime must be initialized before loading checkpoints")
	}

	return rt.checkpoints.Load(ctx, chainID)
}

// LoadReplayBatch loads replay envelopes from the configured replay source.
func (rt *SharedRuntime) LoadReplayBatch(ctx context.Context, chainID string, from Checkpoint) ([]EventEnvelope, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if chainID == "" {
		return nil, fmt.Errorf("chain ID is required")
	}

	rt.mu.RLock()
	initialized := rt.status.Initialized
	rt.mu.RUnlock()
	if !initialized {
		return nil, fmt.Errorf("runtime must be initialized before loading replay batches")
	}
	if rt.replay == nil {
		return nil, fmt.Errorf("replay source is not configured")
	}

	return rt.replay.Replay(ctx, chainID, from)
}

// RecoverChain loads the latest checkpoint for one chain, replays any available
// recovery envelopes, and records additive recovery status facts.
//
//nolint:funlen // RecoverChain has many statements for checkpoint loading, replay, and status recording.
func (rt *SharedRuntime) RecoverChain(ctx context.Context, chainID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if chainID == "" {
		return fmt.Errorf("chain ID is required")
	}

	rt.mu.RLock()
	started := rt.status.Started
	rt.mu.RUnlock()
	if !started {
		return fmt.Errorf("runtime must be started before recovery")
	}

	checkpoint, err := rt.checkpoints.Load(ctx, chainID)
	if err != nil {
		rt.recordRecoveryFailure(chainID, Checkpoint{}, fmt.Errorf("load checkpoint: %w", err))
		return fmt.Errorf("load checkpoint: %w", err)
	}

	replayed := []EventEnvelope(nil)
	if rt.replay != nil {
		replayed, err = rt.replay.Replay(ctx, chainID, checkpoint)
		if err != nil {
			rt.recordRecoveryFailure(chainID, checkpoint, fmt.Errorf("load replay batch: %w", err))
			return fmt.Errorf("load replay batch: %w", err)
		}
	}

	if len(replayed) > 0 {
		if err := rt.ProcessBatch(ctx, chainID, replayed); err != nil {
			rt.recordRecoveryFailure(chainID, checkpoint, fmt.Errorf("process replay batch: %w", err))
			return fmt.Errorf("process replay batch: %w", err)
		}

		if err := rt.acknowledgeReplayBatch(ctx, chainID, replayed); err != nil {
			rt.recordRecoveryFailure(chainID, checkpoint, fmt.Errorf("ack replay batch: %w", err))

			return fmt.Errorf("ack replay batch: %w", err)
		}
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.status.RecoveryRuns++
	rt.status.RecoveryCheckpointLoads++
	rt.status.LastRecoveryChainID = chainID
	rt.status.LastRecoveryCursor = checkpoint.Cursor
	rt.status.LastRecoveryBlock = checkpoint.BlockNumber
	rt.status.LastRecoveryReplayCount = int64(len(replayed))
	rt.status.RecoveryReplayedEvents += int64(len(replayed))
	rt.status.LastRecoveryError = ""
	rt.status.LastRecoveryAt = time.Now()
	rt.status.LastUpdatedAt = rt.status.LastRecoveryAt
	if len(replayed) > 0 {
		rt.status.RecoveryState = "replay-applied"
	} else {
		rt.status.RecoveryState = "checkpoint-loaded"
	}
	return nil
}

// ReplayChainRange reprocesses a bounded replay window for one chain. This is
// the additive manual DLQ replay seam used by operator-driven recovery flows.
func (rt *SharedRuntime) ReplayChainRange(
	ctx context.Context,
	chainID string,
	from, to Checkpoint,
	limit int,
) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if chainID == "" {
		return 0, fmt.Errorf("chain ID is required")
	}

	rt.mu.RLock()
	started := rt.status.Started
	rt.mu.RUnlock()

	if !started {
		return 0, fmt.Errorf("runtime must be started before replay")
	}

	if rt.replay == nil {
		return 0, fmt.Errorf("replay source is not configured")
	}

	replayed, err := rt.loadReplayRange(ctx, chainID, from, to, limit)
	if err != nil {
		return 0, fmt.Errorf("load replay range: %w", err)
	}

	if len(replayed) == 0 {
		return 0, nil
	}

	if err := rt.ProcessBatch(ctx, chainID, replayed); err != nil {
		return 0, fmt.Errorf("process replay range: %w", err)
	}

	if err := rt.acknowledgeReplayBatch(ctx, chainID, replayed); err != nil {
		return 0, fmt.Errorf("ack replay range: %w", err)
	}

	return len(replayed), nil
}

// ProcessBatch applies the first shared indexing orchestration path for one
// chain: dedupe -> persist -> mark processed -> checkpoint.
func (rt *SharedRuntime) ProcessBatch(ctx context.Context, chainID string, events []EventEnvelope) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if chainID == "" {
		return fmt.Errorf("chain ID is required")
	}
	if len(events) == 0 {
		return nil
	}

	rt.mu.RLock()
	started := rt.status.Started
	rt.mu.RUnlock()
	if !started {
		return fmt.Errorf("runtime must be started before processing")
	}

	accepted := make([]EventEnvelope, 0, len(events))
	for _, event := range events {
		if event.ChainID == "" {
			event.ChainID = chainID
		}
		if event.ChainID != chainID {
			return fmt.Errorf("event chain ID mismatch: expected %s, got %s", chainID, event.ChainID)
		}
		if event.EventKey == "" {
			return fmt.Errorf("event key is required")
		}

		processed, err := rt.idempotency.IsProcessed(ctx, event.EventKey)
		if err != nil {
			return fmt.Errorf("idempotency check failed for %s: %w", event.EventKey, err)
		}
		if processed {
			rt.mu.Lock()
			rt.status.SkippedDuplicates++
			rt.status.LastUpdatedAt = time.Now()
			rt.mu.Unlock()
			continue
		}
		accepted = append(accepted, event)
	}

	if len(accepted) == 0 {
		return nil
	}

	if err := rt.sink.Persist(ctx, accepted); err != nil {
		rt.routeFailures(ctx, accepted, true, fmt.Sprintf("persist failed: %v", err))
		return fmt.Errorf("persist batch: %w", err)
	}

	for _, event := range accepted {
		if err := rt.idempotency.MarkProcessed(ctx, event.EventKey); err != nil {
			rt.routeFailures(ctx, []EventEnvelope{event}, true, fmt.Sprintf("mark processed failed: %v", err))
			return fmt.Errorf("mark processed for %s: %w", event.EventKey, err)
		}
	}

	last := accepted[len(accepted)-1]
	checkpoint := Checkpoint{
		ChainID:     chainID,
		Cursor:      last.CheckpointCursor,
		BlockNumber: last.BlockNumber,
		UpdatedAt:   time.Now(),
	}
	if err := rt.checkpoints.Save(ctx, checkpoint); err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}

	rt.mu.Lock()
	rt.status.ProcessedEvents += int64(len(accepted))
	rt.status.LastCheckpointChainID = checkpoint.ChainID
	rt.status.LastCheckpointCursor = checkpoint.Cursor
	rt.status.LastCheckpointBlock = checkpoint.BlockNumber
	rt.status.LastUpdatedAt = time.Now()
	rt.mu.Unlock()
	return nil
}

func (rt *SharedRuntime) routeFailures(ctx context.Context, events []EventEnvelope, retryable bool, reason string) {
	for _, event := range events {
		failure := ProcessingFailure{
			EventKey:   event.EventKey,
			ChainID:    event.ChainID,
			Retryable:  retryable,
			Reason:     reason,
			OccurredAt: time.Now(),
		}
		if err := rt.failureRoute.Route(ctx, failure, event); err != nil {
			rt.logger.Error("failed to route processing failure", "event_key", event.EventKey, "error", err.Error())
			continue
		}

		rt.mu.Lock()
		rt.status.RoutedFailures++
		rt.status.LastUpdatedAt = time.Now()
		rt.mu.Unlock()
	}
}

func (rt *SharedRuntime) loadReplayRange(
	ctx context.Context,
	chainID string,
	from, to Checkpoint,
	limit int,
) ([]EventEnvelope, error) {
	if rangeSource, ok := rt.replay.(ReplayRangeSource); ok {
		return rangeSource.ReplayRange(ctx, chainID, from, to, limit)
	}

	replayed, err := rt.replay.Replay(ctx, chainID, from)
	if err != nil {
		return nil, fmt.Errorf("replay events for chain %s: %w", chainID, err)
	}

	filtered := make([]EventEnvelope, 0, len(replayed))

	for _, event := range replayed {
		if !eventWithinReplayRange(event, from, to) {
			continue
		}

		filtered = append(filtered, event)

		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

func (rt *SharedRuntime) acknowledgeReplayBatch(ctx context.Context, chainID string, events []EventEnvelope) error {
	if len(events) == 0 {
		return nil
	}

	acknowledger, ok := rt.replay.(ReplayAcknowledger)
	if !ok {
		return nil
	}

	return acknowledger.AcknowledgeReplay(ctx, chainID, events)
}

func eventWithinReplayRange(event EventEnvelope, from, to Checkpoint) bool {
	if compareEventToCheckpoint(event, from) < 0 {
		return false
	}

	if checkpointIsZero(to) {
		return true
	}

	return compareEventToCheckpoint(event, to) <= 0
}

func compareEventToCheckpoint(event EventEnvelope, checkpoint Checkpoint) int {
	if checkpointIsZero(checkpoint) {
		return 1
	}

	switch {
	case event.BlockNumber < checkpoint.BlockNumber:
		return -1
	case event.BlockNumber > checkpoint.BlockNumber:
		return 1
	}

	if checkpoint.Cursor == "" || event.CheckpointCursor == "" {
		return 0
	}

	switch {
	case event.CheckpointCursor < checkpoint.Cursor:
		return -1
	case event.CheckpointCursor > checkpoint.Cursor:
		return 1
	default:
		return 0
	}
}

func checkpointIsZero(checkpoint Checkpoint) bool {
	return checkpoint.BlockNumber == 0 && checkpoint.Cursor == ""
}

func (rt *SharedRuntime) recordRecoveryFailure(chainID string, checkpoint Checkpoint, err error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.status.RecoveryFailures++
	rt.status.LastRecoveryChainID = chainID
	rt.status.LastRecoveryCursor = checkpoint.Cursor
	rt.status.LastRecoveryBlock = checkpoint.BlockNumber
	rt.status.LastRecoveryReplayCount = 0
	rt.status.LastRecoveryAt = time.Now()
	rt.status.LastUpdatedAt = rt.status.LastRecoveryAt
	rt.status.RecoveryState = "recovery-error"
	if err != nil {
		rt.status.LastRecoveryError = err.Error()
	}
}
