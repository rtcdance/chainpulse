package reorg

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// mockReorgDatabase implements core.DatabasePlugin for testing.
type mockReorgDatabase struct {
	blocks      map[uint64]*core.Block
	events      []*core.BlockchainEvent
	latestBlock uint64

	deleteCalled bool
	deleteFrom   uint64
	deleteCount  int64
}

func newMockReorgDatabase() *mockReorgDatabase {
	return &mockReorgDatabase{
		blocks: make(map[uint64]*core.Block),
	}
}

func (m *mockReorgDatabase) addBlock(number uint64, hash, parentHash common.Hash) {
	m.blocks[number] = &core.Block{
		Number:     number,
		Hash:       hash,
		ParentHash: parentHash,
	}
	if number > m.latestBlock {
		m.latestBlock = number
	}
}

func (m *mockReorgDatabase) addEvent(blockNumber uint64) {
	m.events = append(m.events, &core.BlockchainEvent{
		BlockNumber: blockNumber,
	})
}

// EventReader
func (m *mockReorgDatabase) GetEvent(_ context.Context, _ string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockReorgDatabase) QueryEvents(_ context.Context, _ any) ([]any, error) {
	return nil, nil
}

func (m *mockReorgDatabase) GetAllEvents(_ context.Context) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *mockReorgDatabase) GetEventsByBlockRange(_ context.Context, fromBlock, _ uint64) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	for _, e := range m.events {
		if uint64(e.BlockNumber) >= fromBlock {
			result = append(result, e)
		}
	}
	return result, nil
}

// EventWriter
func (m *mockReorgDatabase) StoreEvent(_ context.Context, _ any) error { return nil }

func (m *mockReorgDatabase) BatchStoreEvents(_ context.Context, _ []any) error { return nil }

func (m *mockReorgDatabase) DeleteEvent(_ context.Context, _ string) error { return nil }

func (m *mockReorgDatabase) DeleteEventsByBlockRange(_ context.Context, fromBlock, _ uint64) (int64, error) {
	m.deleteCalled = true
	m.deleteFrom = fromBlock
	count := m.deleteCount
	m.deleteCount = 0
	return count, nil
}

func (m *mockReorgDatabase) MarkEventsAsReorged(_ context.Context, fromBlock, toBlock uint64) (int64, error) {
	return m.DeleteEventsByBlockRange(context.Background(), fromBlock, toBlock)
}

// BlockReader
func (m *mockReorgDatabase) GetBlock(_ context.Context, number uint64) (*core.Block, error) {
	b, ok := m.blocks[number]
	if !ok {
		return nil, fmt.Errorf("block %d not found", number)
	}
	return b, nil
}

func (m *mockReorgDatabase) GetLatestBlock(_ context.Context) (uint64, error) {
	return m.latestBlock, nil
}

func (m *mockReorgDatabase) GetAllBlocks(_ context.Context) ([]*core.Block, error) {
	return nil, nil
}

// ReorgStatsProvider
func (m *mockReorgDatabase) GetReorgStats(_ context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

// Plugin interface
func (m *mockReorgDatabase) Name() string                   { return "mock-reorg-db" }
func (m *mockReorgDatabase) Version() string                { return "test" }
func (m *mockReorgDatabase) Initialize(_ context.Context, _ core.Config) error { return nil }
func (m *mockReorgDatabase) Start(_ context.Context) error                   { return nil }
func (m *mockReorgDatabase) Stop(_ context.Context) error                    { return nil }
func (m *mockReorgDatabase) Health(_ context.Context) error                  { return nil }

// --- Helpers ---

func newTestHandler(db *mockReorgDatabase) *ReorgHandler {
	return NewReorgHandler(db, core.NewDefaultLogger(core.LogLevelError), 12, 120)
}

func newTestHandlerWithMaxRollback(db *mockReorgDatabase, maxRollback uint64) *ReorgHandler {
	return NewReorgHandler(db, core.NewDefaultLogger(core.LogLevelError), 12, maxRollback)
}

// --- Tests ---

func TestDetectReorgFirstBlockNoReorg(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	reorg, reorgBlock, err := handler.DetectReorg(context.Background(), 100, common.HexToHash("0xabc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reorg {
		t.Error("first block should not trigger reorg")
	}
	if reorgBlock != 0 {
		t.Errorf("expected reorgBlock 0, got %d", reorgBlock)
	}
}

func TestDetectReorgSameHashNoReorg(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	hash := common.HexToHash("0xabc")
	handler.UpdateBlockHash(100, hash)

	reorg, _, err := handler.DetectReorg(context.Background(), 100, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reorg {
		t.Error("same hash should not trigger reorg")
	}
}

func TestDetectReorgDifferentHashTriggersReorg(t *testing.T) {
	db := newMockReorgDatabase()
	db.addBlock(99, common.HexToHash("0x99"), common.HexToHash("0x98"))
	db.addBlock(100, common.HexToHash("0xaa"), common.HexToHash("0x99"))

	handler := newTestHandler(db)
	handler.UpdateBlockHash(100, common.HexToHash("0xaa"))

	reorg, reorgBlock, err := handler.DetectReorg(context.Background(), 100, common.HexToHash("0xbb"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reorg {
		t.Error("different hash should trigger reorg")
	}
	if reorgBlock == 0 {
		t.Error("expected non-zero reorg block")
	}
}

func TestHandleReorgInvalidBlock(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	err := handler.HandleReorg(context.Background(), 0)
	if err == nil {
		t.Error("expected error for block 0")
	}
}

func TestHandleReorgExceedsMaxRollback(t *testing.T) {
	db := newMockReorgDatabase()
	db.addBlock(200, common.HexToHash("0xc8"), common.HexToHash("0xc7"))
	handler := newTestHandlerWithMaxRollback(db, 10)

	err := handler.HandleReorg(context.Background(), 100)
	if err == nil {
		t.Error("expected error for reorg exceeding max rollback")
	}
}

func TestHandleReorgSuccess(t *testing.T) {
	db := newMockReorgDatabase()
	db.addBlock(105, common.HexToHash("0x69"), common.HexToHash("0x68"))
	db.addEvent(100)
	db.addEvent(101)
	db.addEvent(102)
	db.deleteCount = 3

	handler := newTestHandler(db)

	err := handler.HandleReorg(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !db.deleteCalled {
		t.Error("expected DeleteEventsByBlockRange to be called")
	}
	if db.deleteFrom != 100 {
		t.Errorf("expected delete from block 100, got %d", db.deleteFrom)
	}
}

func TestRollbackEventsEmptyRange(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	count, err := handler.RollbackEvents(context.Background(), 100, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 events rolled back, got %d", count)
	}
}

func TestRollbackEventsWithEvents(t *testing.T) {
	db := newMockReorgDatabase()
	db.addEvent(50)
	db.addEvent(51)
	db.addEvent(52)
	db.deleteCount = 3

	handler := newTestHandler(db)

	count, err := handler.RollbackEvents(context.Background(), 50, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 events rolled back, got %d", count)
	}
	if !db.deleteCalled {
		t.Error("expected DeleteEventsByBlockRange to be called")
	}
}

func TestUpdateBlockHash(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	hash1 := common.HexToHash("0xaaa")
	handler.UpdateBlockHash(100, hash1)

	block, hash, ok := handler.GetLastKnownBlock()
	if !ok {
		t.Error("expected block to exist")
	}
	if block != 100 {
		t.Errorf("expected block 100, got %d", block)
	}
	if hash != hash1 {
		t.Errorf("expected hash %s, got %s", hash1.Hex(), hash.Hex())
	}
}

func TestGetLastKnownBlockEmpty(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	_, _, ok := handler.GetLastKnownBlock()
	if ok {
		t.Error("expected no block when empty")
	}
}

func TestVerifyBlockSequenceValid(t *testing.T) {
	db := newMockReorgDatabase()
	hash100 := common.HexToHash("0x100")
	hash101 := common.HexToHash("0x101")
	db.addBlock(100, hash100, common.HexToHash("0x0ff"))
	db.addBlock(101, hash101, hash100)

	handler := newTestHandler(db)

	err := handler.VerifyBlockSequence(context.Background(), 100, 101)
	if err != nil {
		t.Fatalf("unexpected error for valid sequence: %v", err)
	}
}

func TestVerifyBlockSequenceBroken(t *testing.T) {
	db := newMockReorgDatabase()
	db.addBlock(100, common.HexToHash("0x100"), common.HexToHash("0x0ff"))
	db.addBlock(101, common.HexToHash("0x101"), common.HexToHash("0xbad"))

	handler := newTestHandler(db)

	err := handler.VerifyBlockSequence(context.Background(), 100, 101)
	if err == nil {
		t.Error("expected error for broken sequence")
	}
}

func TestReset(t *testing.T) {
	handler := newTestHandler(newMockReorgDatabase())

	handler.UpdateBlockHash(100, common.HexToHash("0xaaa"))
	handler.Reset()

	_, _, ok := handler.GetLastKnownBlock()
	if ok {
		t.Error("expected no block after reset")
	}
}

func TestHandleReorgPublishesEvent(t *testing.T) {
	db := newMockReorgDatabase()
	db.addBlock(105, common.HexToHash("0x69"), common.HexToHash("0x68"))
	db.addEvent(100)
	db.deleteCount = 2

	published := make(chan *ReorgEvent, 1)
	bus := core.NewEventBus(core.NewDefaultLogger(core.LogLevelError))
	_, _ = bus.Subscribe(context.Background(), "reorg-detected", func(_ context.Context, payload any) error {
		published <- payload.(*ReorgEvent)
		return nil
	})

	handler := NewReorgHandler(db, core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithEventBus(bus)

	err := handler.HandleReorg(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt := <-published:
		if evt.ReorgBlock != 100 {
			t.Errorf("expected reorg block 100, got %d", evt.ReorgBlock)
		}
		if evt.EventsRolledBack != 2 {
			t.Errorf("expected 2 events rolled back, got %d", evt.EventsRolledBack)
		}
	case <-time.After(time.Second):
		t.Error("expected reorg event to be published within 1s")
	}
}

func TestHandleReorgNoEventBus(t *testing.T) {
	db := newMockReorgDatabase()
	db.addBlock(105, common.HexToHash("0x69"), common.HexToHash("0x68"))
	db.addEvent(100)
	db.deleteCount = 1

	handler := newTestHandler(db)

	// Should not panic when eventBus is nil
	err := handler.HandleReorg(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
