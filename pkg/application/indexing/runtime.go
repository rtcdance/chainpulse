package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// EventEnvelope is the shared unit of work passed through indexing runtime
// stages before persistence and downstream publication.
type EventEnvelope struct {
	EventKey         string
	ChainID          string
	BlockNumber      uint64
	TransactionHash  string
	LogIndex         uint
	Payload          interface{}
	ReceivedAt       time.Time
	CheckpointCursor string
}

// Checkpoint captures chain progress for restart/replay aware runtimes.
type Checkpoint struct {
	ChainID     string
	Cursor      string
	BlockNumber uint64
	UpdatedAt   time.Time
}

// ProcessingFailure classifies an event handling failure and the action taken.
type ProcessingFailure struct {
	EventKey   string
	ChainID    string
	Retryable  bool
	Reason     string
	OccurredAt time.Time
}

// RuntimeStatus reports additive indexing runtime health and counters.
type RuntimeStatus struct {
	State                 string
	Initialized           bool
	Started               bool
	Chains                []string
	CheckpointingEnabled  bool
	IdempotencyEnabled    bool
	FailureRoutingEnabled bool
	ReplayEnabled         bool
	ProcessedEvents       int64
	SkippedDuplicates     int64
	RoutedFailures        int64
	LastCheckpointChainID string
	LastCheckpointCursor  string
	LastCheckpointBlock   uint64
	LastUpdatedAt         time.Time
}

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
		Details: map[string]interface{}{
			"state":                   rt.status.State,
			"chains":                  append([]string(nil), rt.status.Chains...),
			"initialized":             rt.status.Initialized,
			"started":                 rt.status.Started,
			"checkpointing_enabled":   rt.status.CheckpointingEnabled,
			"idempotency_enabled":     rt.status.IdempotencyEnabled,
			"failure_routing_enabled": rt.status.FailureRoutingEnabled,
			"replay_enabled":          rt.status.ReplayEnabled,
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
