package bootstrap

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

func makeTestEvent(id string, block uint64) *core.BlockchainEvent {
	return &core.BlockchainEvent{
		ID:              id,
		EventName:       "Transfer",
		ChainID:         "1",
		BlockNumber:     block,
		BlockHash:       common.HexToHash("0x" + id),
		TransactionHash: common.HexToHash("0xabc"),
		ContractAddress: common.HexToAddress("0x123"),
	}
}

func TestMonolithicMemoryDatabase_Lifecycle(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(nil)

	if db.Name() != "monolithic-memory-database" {
		t.Errorf("Name() = %q", db.Name())
	}
	if db.Version() != "1.0.0" {
		t.Errorf("Version() = %q", db.Version())
	}

	if err := db.Initialize(core.Config{}); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
	if err := db.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := db.Health(); err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if err := db.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestMonolithicMemoryDatabase_HealthNotStarted(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(nil)
	if err := db.Health(); err == nil {
		t.Error("expected error from Health() when not started")
	}
}

func TestMonolithicMemoryDatabase_StoreAndGetEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	event := makeTestEvent("evt1", 100)
	if err := db.StoreEvent(ctx, event); err != nil {
		t.Fatalf("StoreEvent() error: %v", err)
	}

	got, err := db.GetEvent(ctx, "evt1")
	if err != nil {
		t.Fatalf("GetEvent() error: %v", err)
	}
	if got == nil || got.ID != "evt1" {
		t.Fatalf("GetEvent() returned %v, expected evt1", got)
	}
}

func TestMonolithicMemoryDatabase_StoreEventErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)

	_ = db.Initialize(core.Config{})
	if err := db.StoreEvent(ctx, "not-an-event"); err == nil {
		t.Error("expected error for non-event type")
	}
	if err := db.StoreEvent(ctx, nil); err == nil {
		t.Error("expected error for nil event")
	}

	_ = db.Start()
	defer db.Stop()
	if err := db.StoreEvent(ctx, nil); err == nil {
		t.Error("expected error for nil event after start")
	}
}

func TestMonolithicMemoryDatabase_BatchStoreEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	events := []any{makeTestEvent("b1", 10), makeTestEvent("b2", 20)}
	if err := db.BatchStoreEvents(ctx, events); err != nil {
		t.Fatalf("BatchStoreEvents() error: %v", err)
	}
	if _, err := db.GetEvent(ctx, "b1"); err != nil {
		t.Error("expected event b1 to exist after batch")
	}
}

func TestMonolithicMemoryDatabase_GetAllEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreEvent(ctx, makeTestEvent("a1", 1))
	_ = db.StoreEvent(ctx, makeTestEvent("a2", 2))

	all, err := db.GetAllEvents(ctx)
	if err != nil {
		t.Fatalf("GetAllEvents() error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 events, got %d", len(all))
	}
}

func TestMonolithicMemoryDatabase_GetEventsByBlockRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreEvent(ctx, makeTestEvent("r1", 5))
	_ = db.StoreEvent(ctx, makeTestEvent("r2", 10))
	_ = db.StoreEvent(ctx, makeTestEvent("r3", 15))

	results, err := db.GetEventsByBlockRange(ctx, 8, 12)
	if err != nil {
		t.Fatalf("GetEventsByBlockRange() error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "r2" {
		t.Errorf("expected 1 event (r2) in range [8,12], got %d events", len(results))
	}
}

func TestMonolithicMemoryDatabase_DeleteEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreEvent(ctx, makeTestEvent("del1", 1))
	if err := db.DeleteEvent(ctx, "del1"); err != nil {
		t.Fatalf("DeleteEvent() error: %v", err)
	}
	got, _ := db.GetEvent(ctx, "del1")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMonolithicMemoryDatabase_DeleteEventsByBlockRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreEvent(ctx, makeTestEvent("x1", 1))
	_ = db.StoreEvent(ctx, makeTestEvent("x2", 5))
	_ = db.StoreEvent(ctx, makeTestEvent("x3", 10))

	deleted, err := db.DeleteEventsByBlockRange(ctx, 2, 8)
	if err != nil {
		t.Fatalf("DeleteEventsByBlockRange() error: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

func TestMonolithicMemoryDatabase_MarkEventsAsReorged(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreEvent(ctx, makeTestEvent("re1", 10))
	count, err := db.MarkEventsAsReorged(ctx, 5, 15)
	if err != nil {
		t.Fatalf("MarkEventsAsReorged() error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 reorged, got %d", count)
	}
	event, _ := db.GetEvent(ctx, "re1")
	if event == nil || event.Status != core.EventStatusReorged {
		t.Error("expected event status to be Reorged")
	}
}

func TestMonolithicMemoryDatabase_StoreAndGetBlock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	block := &core.Block{Number: 42, Hash: common.HexToHash("0xbeef")}
	if err := db.StoreBlockSnapshot(ctx, block); err != nil {
		t.Fatalf("StoreBlockSnapshot() error: %v", err)
	}

	got, err := db.GetBlock(ctx, 42)
	if err != nil {
		t.Fatalf("GetBlock() error: %v", err)
	}
	if got == nil || got.Number != 42 {
		t.Fatal("expected block 42")
	}
}

func TestMonolithicMemoryDatabase_StoreBlockNilError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	if err := db.StoreBlockSnapshot(ctx, nil); err == nil {
		t.Error("expected error for nil block")
	}
}

func TestMonolithicMemoryDatabase_GetLatestBlock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreBlockSnapshot(ctx, &core.Block{Number: 100})
	_ = db.StoreBlockSnapshot(ctx, &core.Block{Number: 200})
	_ = db.StoreEvent(ctx, makeTestEvent("latevt", 150))

	latest, err := db.GetLatestBlock(ctx)
	if err != nil {
		t.Fatalf("GetLatestBlock() error: %v", err)
	}
	if latest != 200 {
		t.Errorf("expected latest block 200, got %d", latest)
	}
}

func TestMonolithicMemoryDatabase_QueryEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreEvent(ctx, makeTestEvent("q1", 1))
	_ = db.StoreEvent(ctx, makeTestEvent("q2", 2))

	results, err := db.QueryEvents(ctx, nil)
	if err != nil {
		t.Fatalf("QueryEvents() error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestMonolithicMemoryDatabase_StoreEventNotStarted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	if err := db.StoreEvent(ctx, makeTestEvent("ns", 1)); err == nil {
		t.Error("expected error when storing before Start()")
	}
}

func TestMonolithicMemoryCache_Lifecycle(t *testing.T) {
	t.Parallel()
	c := NewMonolithicMemoryCache()

	if c.Name() != "monolithic-memory-cache" {
		t.Errorf("Name() = %q", c.Name())
	}
	if c.Version() != "1.0.0" {
		t.Errorf("Version() = %q", c.Version())
	}

	_ = c.Initialize(core.Config{})
	if err := c.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := c.Health(); err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if err := c.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error: %v", err)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestMonolithicMemoryCache_SetGetDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := NewMonolithicMemoryCache()
	_ = c.Initialize(core.Config{})
	_ = c.Start()
	defer c.Stop()

	if err := c.Set(ctx, "k1", []byte("v1"), 60); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	val, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if string(val) != "v1" {
		t.Errorf("Get() = %q, want %q", string(val), "v1")
	}

	if err := c.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	val, _ = c.Get(ctx, "k1")
	if val != nil {
		t.Error("expected nil after delete")
	}
}

func TestMonolithicMemoryCache_SetNotStarted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := NewMonolithicMemoryCache()
	if err := c.Set(ctx, "k", []byte("v"), 60); err == nil {
		t.Error("expected error when setting before Start()")
	}
}

func TestMonolithicMemoryCache_GetStats(t *testing.T) {
	t.Parallel()
	c := NewMonolithicMemoryCache()
	stats := c.GetStats()
	if stats.HitCount != 0 {
		t.Errorf("expected 0 hit count, got %d", stats.HitCount)
	}
}

func TestMonolithicMemoryDatabase_GetReorgStats(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(nil)
	stats, err := db.GetReorgStats(context.Background())
	if err != nil {
		t.Fatalf("GetReorgStats() error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}

func TestMonolithicMemoryDatabase_GetAllBlocks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := NewMonolithicMemoryDatabase(nil)
	_ = db.Initialize(core.Config{})
	_ = db.Start()
	defer db.Stop()

	_ = db.StoreBlockSnapshot(ctx, &core.Block{Number: 1})
	_ = db.StoreBlockSnapshot(ctx, &core.Block{Number: 2})

	blocks, err := db.GetAllBlocks(ctx)
	if err != nil {
		t.Fatalf("GetAllBlocks() error: %v", err)
	}
	if len(blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocks))
	}
}
