package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

func TestBaseDatabasePlugin_Start_NotInitialized(t *testing.T) {
	t.Parallel()
	p := &BaseDatabasePlugin{}
	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected error starting uninitialized plugin")
	}
}

func TestBaseDatabasePlugin_Start_DoubleStart(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	p := NewBaseDatabasePlugin(logger, metrics)
	_ = p.Initialize(context.Background(), core.Config{})
	_ = p.Start(context.Background())
	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected error on double start")
	}
}

func TestBaseDatabasePlugin_Stop_NotRunning(t *testing.T) {
	t.Parallel()
	p := &BaseDatabasePlugin{}
	err := p.Stop(context.Background())
	if err == nil {
		t.Fatal("expected error stopping not-running plugin")
	}
}

func TestBaseDatabasePlugin_Health_NotInitialized(t *testing.T) {
	t.Parallel()
	p := &BaseDatabasePlugin{}
	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for uninitialized health check")
	}
}

func TestBaseDatabasePlugin_Health_NotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	p := NewBaseDatabasePlugin(logger, nil)
	_ = p.Initialize(context.Background(), core.Config{})
	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for not-running health check")
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteBatch(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)

	events := []*blockchain.BlockchainEvent{
		{EventHash: "0xhash1", BlockNumber: 1, ContractAddress: common.HexToAddress("0x1111"), ChainID: "1"},
		{EventHash: "0xhash2", BlockNumber: 2, ContractAddress: common.HexToAddress("0x2222"), ChainID: "1"},
	}
	if err := db.WriteBatch(context.Background(), events); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}
	if db.GetEventCount() != 2 {
		t.Fatalf("expected 2 events, got %d", db.GetEventCount())
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteBatch_Empty(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	err := db.WriteBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("WriteBatch(nil) should succeed: %v", err)
	}
	err = db.WriteBatch(context.Background(), []*blockchain.BlockchainEvent{})
	if err != nil {
		t.Fatalf("WriteBatch(empty) should succeed: %v", err)
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteBatch_NilElements(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	events := []*blockchain.BlockchainEvent{
		{EventHash: "0xvalid", BlockNumber: 1, ChainID: "1"},
		nil,
		{EventHash: "0xvalid2", BlockNumber: 2, ChainID: "1"},
	}
	if err := db.WriteBatch(context.Background(), events); err != nil {
		t.Fatalf("WriteBatch with nil element: %v", err)
	}
	if db.GetEventCount() != 2 {
		t.Fatalf("expected 2 events (nil skipped), got %d", db.GetEventCount())
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteBatch_MissingHash(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	events := []*blockchain.BlockchainEvent{
		{BlockNumber: 1},
	}
	err := db.WriteBatch(context.Background(), events)
	if err == nil {
		t.Fatal("expected error for missing event hash")
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteBatch_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	err := db.WriteBatch(context.Background(), []*blockchain.BlockchainEvent{{EventHash: "0xh"}})
	if err == nil {
		t.Fatal("expected error when plugin not running")
	}
}

func TestDefaultInMemoryDatabasePlugin_GetAllEvents(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	writeNEvents(t, db, 5)

	events, err := db.GetAllEvents(context.Background())
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
}

func TestDefaultInMemoryDatabasePlugin_GetAllEvents_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetAllEvents(context.Background())
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// GetAllBlocks
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetAllBlocks(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	blocks, err := db.GetAllBlocks(context.Background())
	if err != nil {
		t.Fatalf("GetAllBlocks failed: %v", err)
	}
	if blocks == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestDefaultInMemoryDatabasePlugin_GetAllBlocks_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetAllBlocks(context.Background())
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// GetEventsByBlockRange
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetEventsByBlockRange(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	// Write events at blocks 10, 20, 30, 40, 50
	for i := 0; i < 5; i++ {
		ev := &blockchain.BlockchainEvent{
			EventHash:   fmt.Sprintf("0xbev%d", i),
			BlockNumber: uint64(10 + i*10),
			ChainID:     "1",
		}
		if err := db.WriteEvent(context.Background(), ev); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
	}

	tests := []struct {
		name      string
		from, to  uint64
		wantCount int
	}{
		{"full range", 5, 55, 5},
		{"mid range", 15, 35, 2},
		{"single block", 20, 20, 1},
		{"no match", 100, 200, 0},
		{"from only", 30, 999, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GetEventsByBlockRange(context.Background(), tt.from, tt.to)
			if err != nil {
				t.Fatalf("GetEventsByBlockRange: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("len = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestDefaultInMemoryDatabasePlugin_GetEventsByBlockRange_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetEventsByBlockRange(context.Background(), 0, 100)
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// GetBlock
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetBlock(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	// In-memory impl does not store blocks – returns nil, nil
	block, err := db.GetBlock(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}
	if block != nil {
		t.Fatal("expected nil block from in-memory plugin")
	}
}

func TestDefaultInMemoryDatabasePlugin_GetBlock_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetBlock(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// GetLatestBlock
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetLatestBlock(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	latest, err := db.GetLatestBlock(context.Background())
	if err != nil {
		t.Fatalf("GetLatestBlock failed: %v", err)
	}
	if latest != 0 {
		t.Fatalf("expected 0, got %d", latest)
	}
}

func TestDefaultInMemoryDatabasePlugin_GetLatestBlock_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetLatestBlock(context.Background())
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// DeleteEventsByBlockRange
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_DeleteEventsByBlockRange(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	writeNEventRange(t, db, []uint64{10, 20, 30, 40, 50})

	count, err := db.DeleteEventsByBlockRange(context.Background(), 15, 45)
	if err != nil {
		t.Fatalf("DeleteEventsByBlockRange: %v", err)
	}
	if count != 3 { // 20, 30, 40
		t.Fatalf("expected 3 deleted, got %d", count)
	}
	if db.GetEventCount() != 2 {
		t.Fatalf("expected 2 remaining, got %d", db.GetEventCount())
	}
}

func TestDefaultInMemoryDatabasePlugin_DeleteEventsByBlockRange_NoMatch(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	writeNEventRange(t, db, []uint64{1, 2, 3})

	count, err := db.DeleteEventsByBlockRange(context.Background(), 100, 200)
	if err != nil {
		t.Fatalf("DeleteEventsByBlockRange: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestDefaultInMemoryDatabasePlugin_DeleteEventsByBlockRange_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.DeleteEventsByBlockRange(context.Background(), 0, 100)
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// MarkEventsAsReorged
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_MarkEventsAsReorged(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	writeNEventRange(t, db, []uint64{10, 20, 30})

	count, err := db.MarkEventsAsReorged(context.Background(), 15, 25)
	if err != nil {
		t.Fatalf("MarkEventsAsReorged: %v", err)
	}
	if count != 1 { // only block 20
		t.Fatalf("expected 1 reorged, got %d", count)
	}

	// Verify status was updated
	ev, err := db.GetEventByHash("0xev_20")
	if err != nil {
		t.Fatalf("GetEventByHash: %v", err)
	}
	if ev == nil {
		t.Fatal("expected event to exist")
	}
	if ev.Status != blockchain.EventStatusReorged {
		t.Fatalf("expected reorged status, got %v", ev.Status)
	}

	// Verify unmarked event still has default status
	ev2, _ := db.GetEventByHash("0xev_10")
	if ev2.Status == blockchain.EventStatusReorged {
		t.Fatal("expected event 10 to NOT be reorged")
	}
}

func TestDefaultInMemoryDatabasePlugin_MarkEventsAsReorged_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.MarkEventsAsReorged(context.Background(), 0, 100)
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// GetReorgStats
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetReorgStats(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	stats, err := db.GetReorgStats(context.Background())
	if err != nil {
		t.Fatalf("GetReorgStats: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}

func TestDefaultInMemoryDatabasePlugin_GetReorgStats_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetReorgStats(context.Background())
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// WriteEvents – edge cases
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_WriteEvents_Empty(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	err := db.WriteEvents(context.Background(), []blockchain.BlockchainEvent{})
	if err == nil {
		t.Fatal("expected error for empty list")
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteEvents_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	err := db.WriteEvents(context.Background(), []blockchain.BlockchainEvent{{EventHash: "0xh"}})
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

func TestDefaultInMemoryDatabasePlugin_WriteEvents_MissingHash(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	events := []blockchain.BlockchainEvent{
		{BlockNumber: 1},
	}
	err := db.WriteEvents(context.Background(), events)
	if err == nil {
		t.Fatal("expected error for missing hash")
	}
}

// ---------------------------------------------------------------------------
// GetEventByHash – not found path
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetEventByHash_NotFound(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	ev, err := db.GetEventByHash("0xnonexistent")
	if err != nil {
		t.Fatalf("GetEventByHash: %v", err)
	}
	if ev != nil {
		t.Fatal("expected nil for non-existent hash")
	}
}

func TestDefaultInMemoryDatabasePlugin_GetEventByHash_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.GetEventByHash("0xanything")
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

// ---------------------------------------------------------------------------
// DeleteEvent – non-existent
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_DeleteEvent_NotFound(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	err := db.DeleteEvent(context.Background(), "0xnonexistent")
	if err != nil {
		t.Fatalf("DeleteEvent non-existent: %v", err)
	}
}

// ---------------------------------------------------------------------------
// QueryEvents – edge cases: offset, block range filter
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_QueryEvents_WithOffsetAndBlockRange(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	for i := 0; i < 10; i++ {
		ev := &blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xq%d", i),
			BlockNumber:     uint64(100 + i),
			ContractAddress: common.HexToAddress("0xabc"),
			ChainID:         "1",
		}
		if err := db.WriteEvent(context.Background(), ev); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
	}

	filter := &core.EventFilter{
		FromBlock: 103,
		ToBlock:   107,
		Limit:     3,
		Offset:    1,
	}
	result, err := db.QueryEvents(filter)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if result.Total != 5 {
		t.Fatalf("expected total 5, got %d", result.Total)
	}
	// Limit=3, Offset=1 → we get len(events[1:4]) = 3 results
	if len(result.Events) != 3 {
		t.Fatalf("expected 3 events after offset+limit, got %d", len(result.Events))
	}
}

func TestDefaultInMemoryDatabasePlugin_QueryEvents_NilFilter(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	_, err := db.QueryEvents(nil)
	if err == nil {
		t.Fatal("expected error for nil filter")
	}
}

func TestDefaultInMemoryDatabasePlugin_QueryEvents_NotRunning(t *testing.T) {
	t.Parallel()
	db := newInitedDB(t)
	_, err := db.QueryEvents(&core.EventFilter{Limit: 10})
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

func TestDefaultInMemoryDatabasePlugin_QueryEvents_ContractFilter(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)

	addrA := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	addrB := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	for i := 0; i < 3; i++ {
		writeEvent(t, db, fmt.Sprintf("0xca%d", i), uint64(i), addrA)
	}
	for i := 0; i < 2; i++ {
		writeEvent(t, db, fmt.Sprintf("0xcb%d", i), uint64(10+i), addrB)
	}

	// Filter by addrA
	result, err := db.QueryEvents(&core.EventFilter{
		ContractAddress: []common.Address{addrA},
		Limit:           100,
	})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected 3 events for addrA, got %d", result.Total)
	}

	// Filter by both
	result, err = db.QueryEvents(&core.EventFilter{
		ContractAddress: []common.Address{addrA, addrB},
		Limit:           100,
	})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if result.Total != 5 {
		t.Fatalf("expected 5 events for both, got %d", result.Total)
	}
}

// ---------------------------------------------------------------------------
// GetStats – edge cases (zero values, averages)
// ---------------------------------------------------------------------------

func TestDefaultInMemoryDatabasePlugin_GetStats_Empty(t *testing.T) {
	t.Parallel()
	db := newStartedDB(t)
	stats := db.GetStats()
	if stats.WriteCount != 0 || stats.ReadCount != 0 {
		t.Fatalf("expected zero counts on empty db")
	}
	if stats.AvgWriteTimeMs != 0.0 || stats.AvgReadTimeMs != 0.0 {
		t.Fatal("expected zero averages on empty db")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newStartedDB creates, initializes, and starts a DefaultInMemoryDatabasePlugin.
func newStartedDB(t *testing.T) *DefaultInMemoryDatabasePlugin {
	t.Helper()
	db := newInitedDB(t)
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	return db
}

// newInitedDB creates and initializes, but does NOT start.
func newInitedDB(t *testing.T) *DefaultInMemoryDatabasePlugin {
	t.Helper()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)
	if err := db.Initialize(context.Background(), core.Config{}); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	return db
}

func writeNEvents(t *testing.T, db *DefaultInMemoryDatabasePlugin, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		ev := &blockchain.BlockchainEvent{
			EventHash:   fmt.Sprintf("0xh%d", i),
			BlockNumber: uint64(i),
			ChainID:     "1",
		}
		if err := db.WriteEvent(context.Background(), ev); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
	}
}

func writeNEventRange(t *testing.T, db *DefaultInMemoryDatabasePlugin, blocks []uint64) {
	t.Helper()
	for _, b := range blocks {
		ev := &blockchain.BlockchainEvent{
			EventHash:   fmt.Sprintf("0xev_%d", b),
			BlockNumber: b,
			ChainID:     "1",
		}
		if err := db.WriteEvent(context.Background(), ev); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
	}
}

func writeEvent(t *testing.T, db *DefaultInMemoryDatabasePlugin, hash string, block uint64, addr common.Address) {
	t.Helper()
	ev := &blockchain.BlockchainEvent{
		EventHash:       hash,
		BlockNumber:     block,
		ContractAddress: addr,
		ChainID:         "1",
	}
	if err := db.WriteEvent(context.Background(), ev); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}
}
