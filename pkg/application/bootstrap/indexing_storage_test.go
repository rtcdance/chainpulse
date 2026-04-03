package bootstrap

import (
	"context"
	"errors"
	"testing"

	"chainpulse/pkg/core"
)

type stubDatabasePlugin struct {
	initErr    error
	startErr   error
	initCalled bool
	started    bool
	stopped    bool
}

func (s *stubDatabasePlugin) Name() string    { return "stub-db" }
func (s *stubDatabasePlugin) Version() string { return "1.0.0" }
func (s *stubDatabasePlugin) Initialize(config core.Config) error {
	s.initCalled = true
	return s.initErr
}
func (s *stubDatabasePlugin) Start() error {
	if s.startErr == nil {
		s.started = true
	}
	return s.startErr
}
func (s *stubDatabasePlugin) Stop() error   { s.stopped = true; return nil }
func (s *stubDatabasePlugin) Health() error { return nil }
func (s *stubDatabasePlugin) StoreEvent(ctx context.Context, event interface{}) error {
	return nil
}
func (s *stubDatabasePlugin) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) BatchStoreEvents(ctx context.Context, events []interface{}) error {
	return nil
}
func (s *stubDatabasePlugin) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error { return nil }
func (s *stubDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) { return 0, nil }
func (s *stubDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}
func (s *stubDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

type stubCachePlugin struct {
	initErr    error
	startErr   error
	initCalled bool
	started    bool
}

func (s *stubCachePlugin) Name() string    { return "stub-cache" }
func (s *stubCachePlugin) Version() string { return "1.0.0" }
func (s *stubCachePlugin) Initialize(config core.Config) error {
	s.initCalled = true
	return s.initErr
}
func (s *stubCachePlugin) Start() error {
	if s.startErr == nil {
		s.started = true
	}
	return s.startErr
}
func (s *stubCachePlugin) Stop() error                                         { return nil }
func (s *stubCachePlugin) Health() error                                       { return nil }
func (s *stubCachePlugin) Get(ctx context.Context, key string) ([]byte, error) { return nil, nil }
func (s *stubCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	return nil
}
func (s *stubCachePlugin) Delete(ctx context.Context, key string) error { return nil }
func (s *stubCachePlugin) GetStats() core.CacheStats                    { return core.CacheStats{} }

func TestBuildMonolithicIndexingStorageRequiresLogger(t *testing.T) {
	db, cache, err := buildMonolithicIndexingStorageWithDeps(nil, core.Config{}, defaultMonolithicIndexingStorageDeps())
	if err == nil {
		t.Fatal("expected logger validation error")
	}
	if db != nil || cache != nil {
		t.Fatal("expected nil storage on validation error")
	}
}

func TestBuildMonolithicIndexingStorageSuccess(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	db, cache, err := BuildMonolithicIndexingStorage(logger, core.Config{})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if db == nil || cache == nil {
		t.Fatal("expected non-nil storage adapters")
	}
}

func TestBuildMonolithicIndexingStorageStopsDatabaseIfCacheStartFails(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	db := &stubDatabasePlugin{}
	cache := &stubCachePlugin{startErr: errors.New("cache start boom")}

	_, _, err := buildMonolithicIndexingStorageWithDeps(logger, core.Config{}, monolithicIndexingStorageDeps{
		newDatabase: func(logger core.Logger) core.DatabasePlugin { return db },
		newCache:    func() core.CachePlugin { return cache },
	})
	if err == nil {
		t.Fatal("expected cache start failure")
	}
	if !db.stopped {
		t.Fatal("expected database stop on cache start failure")
	}
}
