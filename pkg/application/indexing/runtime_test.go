package indexing

import (
	"context"
	"fmt"
	"testing"

	"chainpulse/pkg/core"
)

type stubEventSource struct{}

func (stubEventSource) Fetch(context.Context, string) ([]EventEnvelope, error) { return nil, nil }

type stubEventSink struct {
	persisted [][]EventEnvelope
	err       error
}

func (s *stubEventSink) Persist(_ context.Context, events []EventEnvelope) error {
	if s.err != nil {
		return s.err
	}
	copied := append([]EventEnvelope(nil), events...)
	s.persisted = append(s.persisted, copied)
	return nil
}

type stubCheckpointStore struct {
	loaded map[string]Checkpoint
	saved  []Checkpoint
	err    error
}

func (s *stubCheckpointStore) Load(_ context.Context, chainID string) (Checkpoint, error) {
	if s.err != nil {
		return Checkpoint{}, s.err
	}
	return s.loaded[chainID], nil
}

func (s *stubCheckpointStore) Save(_ context.Context, checkpoint Checkpoint) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, checkpoint)
	return nil
}

type stubIdempotencyStore struct {
	processed map[string]bool
	markErr   map[string]error
}

func (s *stubIdempotencyStore) IsProcessed(_ context.Context, key string) (bool, error) {
	return s.processed[key], nil
}

func (s *stubIdempotencyStore) MarkProcessed(_ context.Context, key string) error {
	if err := s.markErr[key]; err != nil {
		return err
	}
	s.processed[key] = true
	return nil
}

type routedFailure struct {
	failure ProcessingFailure
	event   EventEnvelope
}

type stubFailureRouter struct {
	routed []routedFailure
	err    error
}

func (s *stubFailureRouter) Route(_ context.Context, failure ProcessingFailure, event EventEnvelope) error {
	if s.err != nil {
		return s.err
	}
	s.routed = append(s.routed, routedFailure{failure: failure, event: event})
	return nil
}

type stubReplaySource struct{}

func (stubReplaySource) Replay(context.Context, string, Checkpoint) ([]EventEnvelope, error) {
	return []EventEnvelope{{EventKey: "replay-1", ChainID: "ethereum", BlockNumber: 9, CheckpointCursor: "9:0"}}, nil
}

func validRuntimeDeps() RuntimeDeps {
	sink := &stubEventSink{}
	checkpoints := &stubCheckpointStore{loaded: map[string]Checkpoint{
		"ethereum": {ChainID: "ethereum", Cursor: "8:0", BlockNumber: 8},
	}}
	idempotency := &stubIdempotencyStore{processed: map[string]bool{}, markErr: map[string]error{}}
	failures := &stubFailureRouter{}

	return RuntimeDeps{
		Logger:          core.NewDefaultLogger(core.LogLevelInfo),
		Source:          stubEventSource{},
		Sink:            sink,
		CheckpointStore: checkpoints,
		Idempotency:     idempotency,
		FailureRouter:   failures,
		ReplaySource:    stubReplaySource{},
		Chains:          []string{"ethereum"},
	}
}

func TestNewSharedRuntimeRejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mut  func(*RuntimeDeps)
		want string
	}{
		{name: "missing logger", mut: func(d *RuntimeDeps) { d.Logger = nil }, want: "logger is required"},
		{name: "missing source", mut: func(d *RuntimeDeps) { d.Source = nil }, want: "event source is required"},
		{name: "missing sink", mut: func(d *RuntimeDeps) { d.Sink = nil }, want: "event sink is required"},
		{name: "missing checkpoint store", mut: func(d *RuntimeDeps) { d.CheckpointStore = nil }, want: "checkpoint store is required"},
		{name: "missing idempotency", mut: func(d *RuntimeDeps) { d.Idempotency = nil }, want: "idempotency store is required"},
		{name: "missing failure router", mut: func(d *RuntimeDeps) { d.FailureRouter = nil }, want: "failure router is required"},
		{name: "missing chains", mut: func(d *RuntimeDeps) { d.Chains = nil }, want: "at least one chain is required"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := validRuntimeDeps()
			tc.mut(&deps)

			_, err := NewSharedRuntime(deps)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected error %q, got %v", tc.want, err)
			}
		})
	}
}

func TestSharedRuntimeProcessBatchSuccess(t *testing.T) {
	t.Parallel()

	deps := validRuntimeDeps()
	rt, err := NewSharedRuntime(deps)
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}
	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start error = %v", err)
	}

	err = rt.ProcessBatch(context.Background(), "ethereum", []EventEnvelope{
		{EventKey: "evt-1", ChainID: "ethereum", BlockNumber: 10, CheckpointCursor: "10:1"},
		{EventKey: "evt-2", ChainID: "ethereum", BlockNumber: 11, CheckpointCursor: "11:0"},
	})
	if err != nil {
		t.Fatalf("ProcessBatch error = %v", err)
	}

	sink := deps.Sink.(*stubEventSink)
	if len(sink.persisted) != 1 || len(sink.persisted[0]) != 2 {
		t.Fatalf("unexpected persisted batches: %+v", sink.persisted)
	}

	checkpoints := deps.CheckpointStore.(*stubCheckpointStore)
	if len(checkpoints.saved) != 1 {
		t.Fatalf("expected one checkpoint save, got %d", len(checkpoints.saved))
	}
	if checkpoints.saved[0].BlockNumber != 11 || checkpoints.saved[0].Cursor != "11:0" {
		t.Fatalf("unexpected checkpoint: %+v", checkpoints.saved[0])
	}

	status := rt.Status()
	if status.ProcessedEvents != 2 || status.RoutedFailures != 0 {
		t.Fatalf("unexpected runtime status: %+v", status)
	}
	if !status.CheckpointingEnabled || !status.IdempotencyEnabled || !status.FailureRoutingEnabled || !status.ReplayEnabled {
		t.Fatalf("expected runtime ports to be marked as enabled: %+v", status)
	}
	if status.LastCheckpointChainID != "ethereum" || status.LastCheckpointCursor != "11:0" || status.LastCheckpointBlock != 11 {
		t.Fatalf("unexpected last checkpoint status: %+v", status)
	}
}

func TestSharedRuntimeLoadCheckpoint(t *testing.T) {
	t.Parallel()

	deps := validRuntimeDeps()
	rt, err := NewSharedRuntime(deps)
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}
	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}

	checkpoint, err := rt.LoadCheckpoint(context.Background(), "ethereum")
	if err != nil {
		t.Fatalf("LoadCheckpoint error = %v", err)
	}
	if checkpoint.BlockNumber != 8 || checkpoint.Cursor != "8:0" {
		t.Fatalf("unexpected checkpoint: %+v", checkpoint)
	}
}

func TestSharedRuntimeLoadReplayBatch(t *testing.T) {
	t.Parallel()

	deps := validRuntimeDeps()
	rt, err := NewSharedRuntime(deps)
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}
	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}

	events, err := rt.LoadReplayBatch(context.Background(), "ethereum", Checkpoint{ChainID: "ethereum", Cursor: "8:0", BlockNumber: 8})
	if err != nil {
		t.Fatalf("LoadReplayBatch error = %v", err)
	}
	if len(events) != 1 || events[0].EventKey != "replay-1" {
		t.Fatalf("unexpected replay batch: %+v", events)
	}
}

func TestSharedRuntimeLoadCheckpointRequiresInitialize(t *testing.T) {
	t.Parallel()

	rt, err := NewSharedRuntime(validRuntimeDeps())
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}

	_, err = rt.LoadCheckpoint(context.Background(), "ethereum")
	if err == nil || err.Error() != "runtime must be initialized before loading checkpoints" {
		t.Fatalf("unexpected LoadCheckpoint error: %v", err)
	}
}

func TestSharedRuntimeLoadReplayBatchRequiresReplaySource(t *testing.T) {
	t.Parallel()

	deps := validRuntimeDeps()
	deps.ReplaySource = nil
	rt, err := NewSharedRuntime(deps)
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}
	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}

	_, err = rt.LoadReplayBatch(context.Background(), "ethereum", Checkpoint{ChainID: "ethereum"})
	if err == nil || err.Error() != "replay source is not configured" {
		t.Fatalf("unexpected LoadReplayBatch error: %v", err)
	}
}

func TestSharedRuntimeProcessBatchSkipsDuplicates(t *testing.T) {
	t.Parallel()

	deps := validRuntimeDeps()
	idempotency := deps.Idempotency.(*stubIdempotencyStore)
	idempotency.processed["evt-1"] = true

	rt, err := NewSharedRuntime(deps)
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}
	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start error = %v", err)
	}

	err = rt.ProcessBatch(context.Background(), "ethereum", []EventEnvelope{
		{EventKey: "evt-1", ChainID: "ethereum", BlockNumber: 10, CheckpointCursor: "10:1"},
		{EventKey: "evt-2", ChainID: "ethereum", BlockNumber: 11, CheckpointCursor: "11:0"},
	})
	if err != nil {
		t.Fatalf("ProcessBatch error = %v", err)
	}

	sink := deps.Sink.(*stubEventSink)
	if len(sink.persisted) != 1 || len(sink.persisted[0]) != 1 || sink.persisted[0][0].EventKey != "evt-2" {
		t.Fatalf("unexpected persisted batches: %+v", sink.persisted)
	}

	status := rt.Status()
	if status.ProcessedEvents != 1 {
		t.Fatalf("unexpected processed events count: %+v", status)
	}
	if status.SkippedDuplicates != 1 {
		t.Fatalf("unexpected skipped duplicates count: %+v", status)
	}
}

func TestSharedRuntimeProcessBatchRoutesPersistFailure(t *testing.T) {
	t.Parallel()

	deps := validRuntimeDeps()
	deps.Sink.(*stubEventSink).err = fmt.Errorf("persist boom")

	rt, err := NewSharedRuntime(deps)
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}
	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start error = %v", err)
	}

	err = rt.ProcessBatch(context.Background(), "ethereum", []EventEnvelope{
		{EventKey: "evt-1", ChainID: "ethereum", BlockNumber: 10, CheckpointCursor: "10:1"},
	})
	if err == nil {
		t.Fatal("expected persist failure")
	}

	failures := deps.FailureRouter.(*stubFailureRouter)
	if len(failures.routed) != 1 {
		t.Fatalf("expected one routed failure, got %d", len(failures.routed))
	}
	if failures.routed[0].failure.Reason != "persist failed: persist boom" {
		t.Fatalf("unexpected routed failure: %+v", failures.routed[0])
	}

	status := rt.Status()
	if status.RoutedFailures != 1 {
		t.Fatalf("unexpected routed failures count: %+v", status)
	}
}

func TestSharedRuntimeLifecycleAndHealth(t *testing.T) {
	t.Parallel()

	rt, err := NewSharedRuntime(validRuntimeDeps())
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}

	status := rt.Status()
	if status.State != "created" || status.Initialized || status.Started {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	if !status.CheckpointingEnabled || !status.IdempotencyEnabled || !status.FailureRoutingEnabled || !status.ReplayEnabled {
		t.Fatalf("expected runtime ports enabled in initial status: %+v", status)
	}

	if err := rt.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}

	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start error = %v", err)
	}

	health, err := rt.Health(context.Background())
	if err != nil {
		t.Fatalf("Health error = %v", err)
	}
	if health.Status != "healthy" {
		t.Fatalf("expected healthy status, got %q", health.Status)
	}

	if err := rt.Stop(context.Background()); err != nil {
		t.Fatalf("Stop error = %v", err)
	}

	status = rt.Status()
	if status.State != "stopped" || status.Started {
		t.Fatalf("unexpected stopped status: %+v", status)
	}
}

func TestSharedRuntimeStartRequiresInitialize(t *testing.T) {
	t.Parallel()

	rt, err := NewSharedRuntime(validRuntimeDeps())
	if err != nil {
		t.Fatalf("NewSharedRuntime error = %v", err)
	}

	err = rt.Start(context.Background())
	if err == nil || err.Error() != "runtime must be initialized before start" {
		t.Fatalf("unexpected start error: %v", err)
	}
}
