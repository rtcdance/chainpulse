package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/reorg"
)

// mockReorgIntegrationDB implements core.DatabasePlugin for reorg integration testing.
type mockReorgIntegrationDB struct {
	events      []*core.BlockchainEvent
	latestBlock uint64
}

func newMockReorgIntegrationDB() *mockReorgIntegrationDB {
	return &mockReorgIntegrationDB{events: make([]*core.BlockchainEvent, 0)}
}

func (m *mockReorgIntegrationDB) addEvent(blockNumber uint64, eventName string) {
	m.events = append(m.events, &core.BlockchainEvent{
		ID:              string(rune(len(m.events) + 1)),
		BlockNumber:     blockNumber,
		TransactionHash: common.HexToHash("0xtx"),
		ContractAddress: common.HexToAddress("0xcontract"),
		EventName:       eventName,
		ChainID:         "1",
	})
	if blockNumber > m.latestBlock {
		m.latestBlock = blockNumber
	}
}

// Plugin interface
func (m *mockReorgIntegrationDB) Name() string { return "mock-reorg-db" }

func (m *mockReorgIntegrationDB) Version() string { return "test" }

func (m *mockReorgIntegrationDB) Initialize(config core.Config) error { return nil }

func (m *mockReorgIntegrationDB) Start() error { return nil }

func (m *mockReorgIntegrationDB) Stop() error { return nil }

func (m *mockReorgIntegrationDB) Health() error { return nil }

// EventReader
func (m *mockReorgIntegrationDB) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockReorgIntegrationDB) QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error) {
	return nil, nil
}

func (m *mockReorgIntegrationDB) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	return m.events, nil
}

func (m *mockReorgIntegrationDB) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	for _, e := range m.events {
		if e.BlockNumber >= fromBlock && e.BlockNumber <= toBlock {
			result = append(result, e)
		}
	}
	return result, nil
}

// EventWriter
func (m *mockReorgIntegrationDB) StoreEvent(ctx context.Context, event interface{}) error { return nil }

func (m *mockReorgIntegrationDB) BatchStoreEvents(ctx context.Context, events []interface{}) error {
	return nil
}

func (m *mockReorgIntegrationDB) DeleteEvent(ctx context.Context, eventID string) error { return nil }

func (m *mockReorgIntegrationDB) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	var remaining []*core.BlockchainEvent
	deleted := int64(0)
	for _, e := range m.events {
		if e.BlockNumber >= fromBlock && e.BlockNumber <= toBlock {
			deleted++
		} else {
			remaining = append(remaining, e)
		}
	}
	m.events = remaining
	// Update latestBlock after deletion
	m.latestBlock = 0
	for _, e := range m.events {
		if e.BlockNumber > m.latestBlock {
			m.latestBlock = e.BlockNumber
		}
	}
	return deleted, nil
}

func (m *mockReorgIntegrationDB) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	var remaining []*core.BlockchainEvent
	var count int64
	for _, e := range m.events {
		if e.BlockNumber >= fromBlock && e.BlockNumber <= toBlock {
			count++
		} else {
			remaining = append(remaining, e)
		}
	}
	m.events = remaining
	// Update latestBlock
	m.latestBlock = 0
	for _, e := range m.events {
		if e.BlockNumber > m.latestBlock {
			m.latestBlock = e.BlockNumber
		}
	}
	return count, nil
}

// BlockReader
func (m *mockReorgIntegrationDB) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	return nil, nil
}

func (m *mockReorgIntegrationDB) GetLatestBlock(ctx context.Context) (uint64, error) {
	return m.latestBlock, nil
}

func (m *mockReorgIntegrationDB) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	return nil, nil
}

// ReorgStatsProvider
func (m *mockReorgIntegrationDB) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return nil, nil
}

// TestReorgDetectionAndRollback simulates a complete reorg scenario:
// 1. Index blocks 100-110 with events
// 2. Detect reorg at block 105
// 3. Verify ReorgHandler rolls back blocks 105-110
// 4. Verify events are removed from storage
func TestReorgDetectionAndRollback(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	db := newMockReorgIntegrationDB()

	// Step 1: Index blocks 100-110
	for i := uint64(100); i <= 110; i++ {
		db.addEvent(i, "Transfer")
	}

	if len(db.events) != 11 {
		t.Fatalf("expected 11 events after initial indexing, got %d", len(db.events))
	}

	// Step 2: Create ReorgHandler and simulate reorg at block 105
	handler := reorg.NewReorgHandler(db, logger, 12, 120)
	if err := handler.HandleReorg(context.Background(), 105); err != nil {
		t.Fatalf("HandleReorg failed: %v", err)
	}

	// Step 3: Verify events from blocks 105-110 were rolled back
	remaining := len(db.events)
	if remaining != 5 { // blocks 100-104 remain
		t.Errorf("expected 5 events after reorg rollback (blocks 100-104), got %d", remaining)
	}

	// Verify no events from blocks >= 105 remain
	for _, e := range db.events {
		if e.BlockNumber >= 105 {
			t.Errorf("found event from block %d after reorg rollback, should have been deleted", e.BlockNumber)
		}
	}
}

// TestReorgWithEventBusPublishing verifies that reorg events are published
// to the event bus when configured.
func TestReorgWithEventBusPublishing(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	db := newMockReorgIntegrationDB()

	for i := uint64(100); i <= 110; i++ {
		db.addEvent(i, "Transfer")
	}

	bus := core.NewEventBus(logger)

	published := make(chan *reorg.ReorgEvent, 1)
	bus.Subscribe(context.Background(), "reorg-detected", func(payload interface{}) {
		if evt, ok := payload.(*reorg.ReorgEvent); ok {
			published <- evt
		}
	})

	handler := reorg.NewReorgHandler(db, logger, 12, 120).
		WithChainID("1").
		WithEventBus(bus)

	if err := handler.HandleReorg(context.Background(), 105); err != nil {
		t.Fatalf("HandleReorg failed: %v", err)
	}

	select {
	case evt := <-published:
		if evt.ReorgBlock != 105 {
			t.Errorf("expected reorg block 105, got %d", evt.ReorgBlock)
		}
		if evt.DetectedAt.IsZero() {
			t.Error("expected non-zero detected timestamp")
		}
		if evt.EventsRolledBack == 0 {
			t.Error("expected some events to be rolled back")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for reorg event to be published")
	}
}

// TestReorgRollbackAndReindex verifies the full cycle:
// index -> reorg -> rollback -> re-index with new data.
func TestReorgRollbackAndReindex(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	db := newMockReorgIntegrationDB()

	// Index original blocks 100-110
	for i := uint64(100); i <= 110; i++ {
		db.addEvent(i, "Transfer")
	}

	handler := reorg.NewReorgHandler(db, logger, 12, 120)
	if err := handler.HandleReorg(context.Background(), 105); err != nil {
		t.Fatalf("HandleReorg failed: %v", err)
	}

	if len(db.events) != 5 {
		t.Fatalf("expected 5 events after rollback, got %d", len(db.events))
	}

	// Re-index blocks 105-110 with new chain data (post-reorg)
	for i := uint64(105); i <= 110; i++ {
		db.addEvent(i, "Transfer")
		db.addEvent(i, "Approval") // extra events from new chain
	}

	if len(db.events) != 17 { // 5 (100-104) + 12 (105-110 x 2 events each)
		t.Errorf("expected 17 events after re-index, got %d", len(db.events))
	}
}
