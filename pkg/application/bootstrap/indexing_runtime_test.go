package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
)

func TestBuildMonolithicIndexingRuntimeRequiresLogger(t *testing.T) {
	runtime, err := buildMonolithicIndexingRuntimeWithDeps(nil, &runtimeTestDatabasePlugin{}, &runtimeTestCachePlugin{}, []string{"ethereum"}, defaultMonolithicIndexingRuntimeDeps())
	if err == nil {
		t.Fatal("expected logger validation error")
	}
	if runtime != nil {
		t.Fatal("expected nil runtime on validation error")
	}
}

func TestBuildMonolithicIndexingRuntimeRequiresChains(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	runtime, err := buildMonolithicIndexingRuntimeWithDeps(logger, &runtimeTestDatabasePlugin{}, &runtimeTestCachePlugin{}, []string{" ", ""}, defaultMonolithicIndexingRuntimeDeps())
	if err == nil {
		t.Fatal("expected chain validation error")
	}
	if runtime != nil {
		t.Fatal("expected nil runtime on validation error")
	}
}

func TestBuildMonolithicIndexingRuntimePropagatesConstructorFailure(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	expectedErr := errors.New("constructor boom")

	runtime, err := buildMonolithicIndexingRuntimeWithDeps(logger, &runtimeTestDatabasePlugin{}, &runtimeTestCachePlugin{}, []string{"ethereum"}, monolithicIndexingRuntimeDeps{
		newRuntime: func(deps appindexing.RuntimeDeps) (*appindexing.SharedRuntime, error) {
			return nil, expectedErr
		},
		newSink: func(database core.DatabasePlugin, cache core.CachePlugin, logger core.Logger) (appindexing.EventSink, error) {
			return &capturingSink{}, nil
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected constructor failure, got %v", err)
	}
	if runtime != nil {
		t.Fatal("expected nil runtime on constructor failure")
	}
}

func TestBuildMonolithicIndexingRuntimeRequiresDatabase(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	runtime, err := buildMonolithicIndexingRuntimeWithDeps(logger, nil, &runtimeTestCachePlugin{}, []string{"ethereum"}, defaultMonolithicIndexingRuntimeDeps())
	if err == nil {
		t.Fatal("expected database validation error")
	}
	if runtime != nil {
		t.Fatal("expected nil runtime on validation error")
	}
}

func TestBuildMonolithicIndexingRuntimePropagatesSinkFailure(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	expectedErr := errors.New("sink boom")

	runtime, err := buildMonolithicIndexingRuntimeWithDeps(logger, &runtimeTestDatabasePlugin{}, &runtimeTestCachePlugin{}, []string{"ethereum"}, monolithicIndexingRuntimeDeps{
		newRuntime: appindexing.NewSharedRuntime,
		newSink: func(database core.DatabasePlugin, cache core.CachePlugin, logger core.Logger) (appindexing.EventSink, error) {
			return nil, expectedErr
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected sink failure, got %v", err)
	}
	if runtime != nil {
		t.Fatal("expected nil runtime on sink failure")
	}
}

func TestBuildMonolithicIndexingRuntimeSuccess(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	db := &runtimeTestDatabasePlugin{}
	cache := &runtimeTestCachePlugin{}

	runtime, err := BuildMonolithicIndexingRuntime(logger, db, cache, []string{" ethereum ", "", "polygon"})
	if err != nil {
		t.Fatalf("expected runtime build success, got %v", err)
	}

	if err := runtime.Initialize(context.Background()); err != nil {
		t.Fatalf("expected initialize success, got %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("expected start success, got %v", err)
	}

	status := runtime.Status()
	if status.State != "running" {
		t.Fatalf("expected running state, got %s", status.State)
	}
	if len(status.Chains) != 2 {
		t.Fatalf("expected two normalized chains, got %d", len(status.Chains))
	}
	if status.Chains[0] != "ethereum" || status.Chains[1] != "polygon" {
		t.Fatalf("unexpected chain list: %#v", status.Chains)
	}

	checkpoint, err := runtime.LoadCheckpoint(context.Background(), "ethereum")
	if err != nil {
		t.Fatalf("expected checkpoint load success, got %v", err)
	}
	if checkpoint.ChainID != "ethereum" {
		t.Fatalf("expected checkpoint chain id ethereum, got %s", checkpoint.ChainID)
	}

	replay, err := runtime.LoadReplayBatch(context.Background(), "ethereum", appindexing.Checkpoint{ChainID: "ethereum"})
	if err != nil {
		t.Fatalf("expected replay load success, got %v", err)
	}
	if len(replay) != 0 {
		t.Fatalf("expected empty replay batch, got %+v", replay)
	}
}

func TestBuildInMemoryIndexingRuntimeSuccess(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	runtime, err := BuildInMemoryIndexingRuntime(logger, capturingSink{}, []string{"ethereum"})
	if err != nil {
		t.Fatalf("expected in-memory runtime build success, got %v", err)
	}
	if err := runtime.Initialize(context.Background()); err != nil {
		t.Fatalf("expected initialize success, got %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("expected start success, got %v", err)
	}

	status := runtime.Status()
	if !status.CheckpointingEnabled || !status.IdempotencyEnabled || !status.FailureRoutingEnabled || !status.ReplayEnabled {
		t.Fatalf("expected in-memory runtime ports enabled: %+v", status)
	}
}

func TestBuildMonolithicIndexingRuntimeRoutesFailuresIntoReplayJournal(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	db := &runtimeTestDatabasePlugin{}
	cache := &runtimeTestCachePlugin{}

	runtime, err := buildMonolithicIndexingRuntimeWithDeps(logger, db, cache, []string{"ethereum"}, monolithicIndexingRuntimeDeps{
		newRuntime: appindexing.NewSharedRuntime,
		newSink: func(database core.DatabasePlugin, cache core.CachePlugin, logger core.Logger) (appindexing.EventSink, error) {
			return failingRuntimeSink{err: errors.New("persist failed")}, nil
		},
	})
	if err != nil {
		t.Fatalf("expected runtime build success, got %v", err)
	}
	if err := runtime.Initialize(context.Background()); err != nil {
		t.Fatalf("expected initialize success, got %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("expected start success, got %v", err)
	}

	err = runtime.ProcessBatch(context.Background(), "ethereum", []appindexing.EventEnvelope{
		{
			EventKey:         "evt-1",
			ChainID:          "ethereum",
			BlockNumber:      10,
			CheckpointCursor: "10:1",
			ReceivedAt:       time.Unix(1700000000, 0),
		},
	})
	if err == nil {
		t.Fatal("expected process batch failure")
	}

	replay, err := runtime.LoadReplayBatch(context.Background(), "ethereum", appindexing.Checkpoint{ChainID: "ethereum"})
	if err != nil {
		t.Fatalf("expected replay load success, got %v", err)
	}
	if len(replay) != 1 || replay[0].EventKey != "evt-1" {
		t.Fatalf("unexpected replay batch: %+v", replay)
	}

	status := runtime.Status()
	if !status.FailureRoutingEnabled || !status.ReplayEnabled {
		t.Fatalf("expected replay and failure routing enabled: %+v", status)
	}
	if status.RoutedFailures != 1 {
		t.Fatalf("unexpected routed failure count: %+v", status)
	}
}

type capturingSink struct{}

func (capturingSink) Persist(ctx context.Context, events []appindexing.EventEnvelope) error {
	return nil
}

type failingRuntimeSink struct {
	err error
}

func (s failingRuntimeSink) Persist(ctx context.Context, events []appindexing.EventEnvelope) error {
	return s.err
}

type runtimeTestDatabasePlugin struct{}

func (d *runtimeTestDatabasePlugin) Name() string                        { return "runtime-test-db" }
func (d *runtimeTestDatabasePlugin) Version() string                     { return "1.0.0" }
func (d *runtimeTestDatabasePlugin) Initialize(config core.Config) error { return nil }
func (d *runtimeTestDatabasePlugin) Start() error                        { return nil }
func (d *runtimeTestDatabasePlugin) Stop() error                         { return nil }
func (d *runtimeTestDatabasePlugin) Health() error                       { return nil }
func (d *runtimeTestDatabasePlugin) StoreEvent(ctx context.Context, event interface{}) error {
	return nil
}

func (d *runtimeTestDatabasePlugin) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (d *runtimeTestDatabasePlugin) QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error) {
	return nil, nil
}

func (d *runtimeTestDatabasePlugin) BatchStoreEvents(ctx context.Context, events []interface{}) error {
	return nil
}

func (d *runtimeTestDatabasePlugin) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (d *runtimeTestDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	return nil, nil
}

func (d *runtimeTestDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error {
	return nil
}

func (d *runtimeTestDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (d *runtimeTestDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	return nil, nil
}

func (d *runtimeTestDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (d *runtimeTestDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (d *runtimeTestDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

type runtimeTestCachePlugin struct{}

func (c *runtimeTestCachePlugin) Name() string                        { return "runtime-test-cache" }
func (c *runtimeTestCachePlugin) Version() string                     { return "1.0.0" }
func (c *runtimeTestCachePlugin) Initialize(config core.Config) error { return nil }
func (c *runtimeTestCachePlugin) Start() error                        { return nil }
func (c *runtimeTestCachePlugin) Stop() error                         { return nil }
func (c *runtimeTestCachePlugin) Health() error                       { return nil }
func (c *runtimeTestCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}

func (c *runtimeTestCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	return nil
}
func (c *runtimeTestCachePlugin) Delete(ctx context.Context, key string) error { return nil }
func (c *runtimeTestCachePlugin) GetStats() core.CacheStats                    { return core.CacheStats{} }
