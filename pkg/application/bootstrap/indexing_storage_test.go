package bootstrap

import (
	"context"
	"errors"
	"testing"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
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
func (s *stubDatabasePlugin) Initialize(_ context.Context, config core.Config) error {
	s.initCalled = true
	return s.initErr
}

func (s *stubDatabasePlugin) Start(_ context.Context) error {
	if s.startErr == nil {
		s.started = true
	}
	return s.startErr
}
func (s *stubDatabasePlugin) Stop(_ context.Context) error   { s.stopped = true; return nil }
func (s *stubDatabasePlugin) Health(_ context.Context) error { return nil }
func (s *stubDatabasePlugin) StoreEvent(ctx context.Context, event any) error {
	return nil
}

func (s *stubDatabasePlugin) GetEvent(ctx context.Context, id string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (s *stubDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	return nil, nil
}

func (s *stubDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error {
	return nil
}

func (s *stubDatabasePlugin) GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (s *stubDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error { return nil }
func (s *stubDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (s *stubDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*blockchain.Block, error) {
	return nil, nil
}
func (s *stubDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) { return 0, nil }
func (s *stubDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (s *stubDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (s *stubDatabasePlugin) GetReorgStats(ctx context.Context) (*portcontract.ReorgStats, error) {
	return &portcontract.ReorgStats{}, nil
}

type stubCachePlugin struct {
	initErr    error
	startErr   error
	initCalled bool
	started    bool
}

func (s *stubCachePlugin) Name() string    { return "stub-cache" }
func (s *stubCachePlugin) Version() string { return "1.0.0" }
func (s *stubCachePlugin) Initialize(_ context.Context, config core.Config) error {
	s.initCalled = true
	return s.initErr
}

func (s *stubCachePlugin) Start(_ context.Context) error {
	if s.startErr == nil {
		s.started = true
	}
	return s.startErr
}
func (s *stubCachePlugin) Stop(_ context.Context) error                        { return nil }
func (s *stubCachePlugin) Health(_ context.Context) error                      { return nil }
func (s *stubCachePlugin) HealthCheck(ctx context.Context) error               { return nil }
func (s *stubCachePlugin) Get(ctx context.Context, key string) ([]byte, error) { return nil, nil }
func (s *stubCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	return nil
}
func (s *stubCachePlugin) Delete(ctx context.Context, key string) error { return nil }
func (s *stubCachePlugin) GetStats() portcontract.CacheStats            { return portcontract.CacheStats{} }

func TestBuildMonolithicIndexingStorageRequiresLogger(t *testing.T) {
	t.Parallel()
	db, cache, err := buildMonolithicIndexingStorageWithDeps(context.Background(), nil, core.Config{}, defaultMonolithicIndexingStorageDeps())
	if err == nil {
		t.Fatal("expected logger validation error")
	}
	if db != nil || cache != nil {
		t.Fatal("expected nil storage on validation error")
	}
}

type legacyConfiguredCache struct{ stubCachePlugin }

func (c *legacyConfiguredCache) Initialize(*core.Config) error { c.initCalled = true; return nil }
func (c *legacyConfiguredCache) Start(context.Context) error {
	if !c.initCalled {
		return errors.New("legacy cache was not initialized")
	}
	c.started = true
	return nil
}

func TestStorageInitializesLegacyPointerConfigCache(t *testing.T) {
	t.Parallel()
	cache := &legacyConfiguredCache{}
	_, _, err := buildMonolithicIndexingStorageWithDeps(context.Background(), core.NewDefaultLogger(core.LogLevelError), core.Config{}, monolithicIndexingStorageDeps{
		newDatabase: func(portcontract.Logger, core.Config) portcontract.DatabasePlugin { return &stubDatabasePlugin{} },
		newCache:    func(core.Config) portcontract.CachePlugin { return cache },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cache.initCalled || !cache.started {
		t.Fatal("legacy cache lifecycle was skipped")
	}
}

func TestBuildMonolithicIndexingStorageSuccess(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	db, cache, err := BuildMonolithicIndexingStorage(context.Background(), logger, core.Config{})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if db == nil || cache == nil {
		t.Fatal("expected non-nil storage adapters")
	}
}

func TestNewMonolithicIndexingDatabaseForMode(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	monolithicDB := newMonolithicIndexingDatabaseForMode(logger, core.Config{DeploymentMode: "monolithic"})
	if monolithicDB.Name() != "monolithic-memory-database" {
		t.Fatalf("expected monolithic-memory-database, got %s", monolithicDB.Name())
	}

	microserviceDB := newMonolithicIndexingDatabaseForMode(logger, core.Config{DeploymentMode: "microservice"})
	if microserviceDB.Name() != "mock-db" {
		t.Fatalf("expected mock-db, got %s", microserviceDB.Name())
	}
}

func TestSnapshotCompatibleDatabaseStoresBlockSnapshots(t *testing.T) {
	t.Parallel()
	db := newSnapshotCompatibleDatabase(&stubDatabasePlugin{})

	hash := common.HexToHash("0xabc")
	block := &blockchain.Block{Number: 12, Hash: hash}
	if err := db.StoreBlockSnapshot(context.Background(), block); err != nil {
		t.Fatalf("store block snapshot: %v", err)
	}

	got, err := db.GetBlock(context.Background(), 12)
	if err != nil {
		t.Fatalf("get block: %v", err)
	}
	if got == nil || got.Hash != hash {
		t.Fatalf("expected block hash 0xabc, got %+v", got)
	}
}

func TestSnapshotCompatibleDatabase_GetAllBlocks_FallbackToLocal(t *testing.T) {
	t.Parallel()
	sdb := newSnapshotCompatibleDatabase(&stubDatabasePlugin{})

	// Store blocks to local cache (delegate returns nil)
	for i := uint64(1); i <= 3; i++ {
		err := sdb.StoreBlockSnapshot(context.Background(), &blockchain.Block{Number: i})
		if err != nil {
			t.Fatalf("store block %d: %v", i, err)
		}
	}

	blocks, err := sdb.GetAllBlocks(context.Background())
	if err != nil {
		t.Fatalf("get all blocks: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
}

type stubDatabasePluginWithBlocks struct {
	stubDatabasePlugin
	blocks []*blockchain.Block
}

func (s *stubDatabasePluginWithBlocks) GetAllBlocks(_ context.Context) ([]*blockchain.Block, error) {
	return s.blocks, nil
}

func TestSnapshotCompatibleDatabase_GetAllBlocks_FromDelegate(t *testing.T) {
	t.Parallel()
	delegate := &stubDatabasePluginWithBlocks{
		blocks: []*blockchain.Block{{Number: 42}, {Number: 99}},
	}
	sdb := newSnapshotCompatibleDatabase(delegate)

	blocks, err := sdb.GetAllBlocks(context.Background())
	if err != nil {
		t.Fatalf("get all blocks: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks from delegate, got %d", len(blocks))
	}
	if blocks[0].Number != 42 {
		t.Fatalf("expected block 42 first, got %d", blocks[0].Number)
	}
}

func TestBuildMonolithicIndexingStorageStopsDatabaseIfCacheStartFails(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	db := &stubDatabasePlugin{}
	cache := &stubCachePlugin{startErr: errors.New("cache start boom")}

	_, _, err := buildMonolithicIndexingStorageWithDeps(context.Background(), logger, core.Config{}, monolithicIndexingStorageDeps{
		newDatabase: func(logger portcontract.Logger, config core.Config) portcontract.DatabasePlugin { return db },
		newCache:    func(core.Config) portcontract.CachePlugin { return cache },
	})
	if err == nil {
		t.Fatal("expected cache start failure")
	}
	if !db.stopped {
		t.Fatal("expected database stop on cache start failure")
	}
}

// Regression (F27): the snapshotCompatibleDatabase wrapper embeds only the
// portcontract.DatabasePlugin interface, so the ConfigurablePlugin/LifecyclePlugin type
// assertions in buildMonolithicIndexingStorageWithDeps silently failed for every
// wrapped database. Initialize/Start were never called — for the PostgreSQL
// adapter this left p.db nil and every event store failed at runtime with
// "database not initialized", while the starter logged "indexing storage
// started" (a false pass). Observed live on 2026-10-13: the deployed container
// received a real Transfer event through the port stack and dropped it with
// exactly that error.
func TestSnapshotCompatibleDatabaseForwardsLifecycleToWrappedPlugin(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	inner := &stubDatabasePlugin{}

	_, _, err := buildMonolithicIndexingStorageWithDeps(context.Background(), logger, core.Config{}, monolithicIndexingStorageDeps{
		newDatabase: func(portcontract.Logger, core.Config) portcontract.DatabasePlugin {
			return newSnapshotCompatibleDatabase(inner)
		},
		newCache: func(core.Config) portcontract.CachePlugin { return &stubCachePlugin{} },
	})
	if err != nil {
		t.Fatalf("build indexing storage: %v", err)
	}
	if !inner.initCalled {
		t.Fatal("Initialize was not forwarded to the wrapped database plugin")
	}
	if !inner.started {
		t.Fatal("Start was not forwarded to the wrapped database plugin")
	}
}
