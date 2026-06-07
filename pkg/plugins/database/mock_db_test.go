package database

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestNewMockDB(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if db == nil {
		t.Fatal("expected non-nil MockDB")
	}
	if db.Name() != "mock-db" {
		t.Fatalf("expected name 'mock-db', got %q", db.Name())
	}
	if db.Version() != "1.0.0" {
		t.Fatalf("expected version '1.0.0', got %q", db.Version())
	}
	if len(db.events) != 0 {
		t.Fatal("expected empty events map")
	}
	if len(db.blocks) != 0 {
		t.Fatal("expected empty blocks map")
	}
}

func TestMockDB_Name(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if db.Name() != "mock-db" {
		t.Fatalf("expected 'mock-db', got %q", db.Name())
	}
}

func TestMockDB_Version(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if db.Version() != "1.0.0" {
		t.Fatalf("expected '1.0.0', got %q", db.Version())
	}
}

func TestMockDB_Initialize(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if err := db.Initialize(t.Context(), core.Config{}); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestMockDB_Start(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if err := db.Start(t.Context()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !db.started {
		t.Fatal("expected started to be true")
	}
}

func TestMockDB_Stop(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if err := db.Start(t.Context()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := db.Stop(t.Context()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if db.started {
		t.Fatal("expected started to be false after stop")
	}
}

func TestMockDB_Health_NotStarted(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	err := db.Health(t.Context())
	if err == nil {
		t.Fatal("expected error for health check when not started")
	}
}

func TestMockDB_Health_Started(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if err := db.Start(t.Context()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := db.Health(t.Context()); err != nil {
		t.Fatalf("Health failed: %v", err)
	}
}

func TestMockDB_StoreEvent(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	event := &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100}
	if err := db.StoreEvent(t.Context(), event); err != nil {
		t.Fatalf("StoreEvent failed: %v", err)
	}
	if len(db.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(db.events))
	}
}

func TestMockDB_StoreEvent_NonEvent(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if err := db.StoreEvent(t.Context(), "not-an-event"); err != nil {
		t.Fatalf("StoreEvent failed: %v", err)
	}
	if len(db.events) != 0 {
		t.Fatal("expected 0 events for non-event input")
	}
}

func TestMockDB_GetEvent_Exists(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	event := &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100}
	_ = db.StoreEvent(t.Context(), event)

	result, err := db.GetEvent(t.Context(), "evt-1")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil event")
	}
	if result.ID != "evt-1" {
		t.Fatalf("expected event ID 'evt-1', got %q", result.ID)
	}
}

func TestMockDB_GetEvent_NotFound(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	result, err := db.GetEvent(t.Context(), "nonexistent")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if result != nil {
		t.Fatal("expected nil for non-existent event")
	}
}

func TestMockDB_QueryEvents(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-2", BlockNumber: 200})

	results, err := db.QueryEvents(t.Context(), nil)
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 events, got %d", len(results))
	}
}

func TestMockDB_QueryEvents_Empty(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	results, err := db.QueryEvents(t.Context(), nil)
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 events, got %d", len(results))
	}
}

func TestMockDB_BatchStoreEvents(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	events := []any{
		&blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100},
		&blockchain.BlockchainEvent{ID: "evt-2", BlockNumber: 200},
	}
	if err := db.BatchStoreEvents(t.Context(), events); err != nil {
		t.Fatalf("BatchStoreEvents failed: %v", err)
	}
	if len(db.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(db.events))
	}
}

func TestMockDB_BatchStoreEvents_WithNonEvent(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	events := []any{
		&blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100},
		"not-an-event",
	}
	if err := db.BatchStoreEvents(t.Context(), events); err != nil {
		t.Fatalf("BatchStoreEvents failed: %v", err)
	}
	if len(db.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(db.events))
	}
}

func TestMockDB_GetAllEvents(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-2", BlockNumber: 200})

	events, err := db.GetAllEvents(t.Context())
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestMockDB_GetAllEvents_Empty(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	events, err := db.GetAllEvents(t.Context())
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestMockDB_GetAllBlocks(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	db.mu.Lock()
	db.blocks[100] = &blockchain.Block{Number: 100, Hash: common.HexToHash("0xabc")}
	db.mu.Unlock()

	blocks, err := db.GetAllBlocks(t.Context())
	if err != nil {
		t.Fatalf("GetAllBlocks failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestMockDB_GetAllBlocks_Empty(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	blocks, err := db.GetAllBlocks(t.Context())
	if err != nil {
		t.Fatalf("GetAllBlocks failed: %v", err)
	}
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestMockDB_DeleteEvent(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100})
	if err := db.DeleteEvent(t.Context(), "evt-1"); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}
	if len(db.events) != 0 {
		t.Fatal("expected 0 events after delete")
	}
}

func TestMockDB_DeleteEvent_NonExistent(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	if err := db.DeleteEvent(t.Context(), "nonexistent"); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}
}

func TestMockDB_GetEventsByBlockRange(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-2", BlockNumber: 150})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-3", BlockNumber: 200})

	events, err := db.GetEventsByBlockRange(t.Context(), 100, 150)
	if err != nil {
		t.Fatalf("GetEventsByBlockRange failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events in range, got %d", len(events))
	}
}

func TestMockDB_GetEventsByBlockRange_Empty(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	events, err := db.GetEventsByBlockRange(t.Context(), 1, 10)
	if err != nil {
		t.Fatalf("GetEventsByBlockRange failed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestMockDB_GetBlock_Exists(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	db.mu.Lock()
	db.blocks[42] = &blockchain.Block{Number: 42, Hash: common.HexToHash("0xdef")}
	db.mu.Unlock()

	block, err := db.GetBlock(t.Context(), 42)
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}
	if block == nil {
		t.Fatal("expected non-nil block")
	}
	if block.Number != 42 {
		t.Fatalf("expected block 42, got %d", block.Number)
	}
}

func TestMockDB_GetBlock_NotFound(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	block, err := db.GetBlock(t.Context(), 999)
	if err != nil {
		t.Fatalf("GetBlock failed: %v", err)
	}
	if block != nil {
		t.Fatal("expected nil for non-existent block")
	}
}

func TestMockDB_GetLatestBlock_Empty(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	latest, err := db.GetLatestBlock(t.Context())
	if err != nil {
		t.Fatalf("GetLatestBlock failed: %v", err)
	}
	if latest != 0 {
		t.Fatalf("expected 0 for empty blocks, got %d", latest)
	}
}

func TestMockDB_GetLatestBlock_WithBlocks(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	db.mu.Lock()
	db.blocks[10] = &blockchain.Block{Number: 10}
	db.blocks[50] = &blockchain.Block{Number: 50}
	db.blocks[30] = &blockchain.Block{Number: 30}
	db.mu.Unlock()

	latest, err := db.GetLatestBlock(t.Context())
	if err != nil {
		t.Fatalf("GetLatestBlock failed: %v", err)
	}
	if latest != 50 {
		t.Fatalf("expected 50, got %d", latest)
	}
}

func TestMockDB_DeleteEventsByBlockRange(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-2", BlockNumber: 150})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-3", BlockNumber: 200})

	count, err := db.DeleteEventsByBlockRange(t.Context(), 100, 150)
	if err != nil {
		t.Fatalf("DeleteEventsByBlockRange failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 deleted events, got %d", count)
	}
	if len(db.events) != 1 {
		t.Fatalf("expected 1 remaining event, got %d", len(db.events))
	}
}

func TestMockDB_DeleteEventsByBlockRange_EmptyRange(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	count, err := db.DeleteEventsByBlockRange(t.Context(), 1, 10)
	if err != nil {
		t.Fatalf("DeleteEventsByBlockRange failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 deleted events, got %d", count)
	}
}

func TestMockDB_MarkEventsAsReorged(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-1", BlockNumber: 100})
	_ = db.StoreEvent(t.Context(), &blockchain.BlockchainEvent{ID: "evt-2", BlockNumber: 200})

	count, err := db.MarkEventsAsReorged(t.Context(), 100, 150)
	if err != nil {
		t.Fatalf("MarkEventsAsReorged failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 reorged event, got %d", count)
	}

	event, _ := db.GetEvent(t.Context(), "evt-1")
	if event.Status != blockchain.EventStatusReorged {
		t.Fatalf("expected event status reorged, got %q", event.Status)
	}
}

func TestMockDB_MarkEventsAsReorged_EmptyRange(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	count, err := db.MarkEventsAsReorged(t.Context(), 1, 10)
	if err != nil {
		t.Fatalf("MarkEventsAsReorged failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 reorged events, got %d", count)
	}
}

func TestMockDB_GetReorgStats(t *testing.T) {
	t.Parallel()
	db := NewMockDB()
	stats, err := db.GetReorgStats(t.Context())
	if err != nil {
		t.Fatalf("GetReorgStats failed: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}
