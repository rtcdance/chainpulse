package database

import (
	"context"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type MockDB struct {
	name    string
	version string
	events  map[string]*core.BlockchainEvent
	blocks  map[uint64]*core.Block
	mu      sync.RWMutex
	started bool
}

func NewMockDB() *MockDB {
	return &MockDB{
		name:    "mock-db",
		version: "1.0.0",
		events:  make(map[string]*core.BlockchainEvent),
		blocks:  make(map[uint64]*core.Block),
	}
}

func (m *MockDB) Name() string    { return m.name }
func (m *MockDB) Version() string { return m.version }

func (m *MockDB) Initialize(_ context.Context, _ core.Config) error {
	return nil
}

func (m *MockDB) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	return nil
}

func (m *MockDB) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	m.events = make(map[string]*core.BlockchainEvent)
	m.blocks = make(map[uint64]*core.Block)
	return nil
}

func (m *MockDB) Health(_ context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.started {
		return core.NewSystemError(core.ErrorTypeCritical, core.ErrorCodeInternalError, "mock db not started", nil)
	}
	return nil
}

func (m *MockDB) StoreEvent(ctx context.Context, event any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := event.(*core.BlockchainEvent); ok {
		m.events[e.ID] = e
	}
	return nil
}

func (m *MockDB) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	event, exists := m.events[id]
	if !exists {
		return nil, nil
	}
	return event, nil
}

func (m *MockDB) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]any, 0, len(m.events))
	for _, event := range m.events {
		results = append(results, event)
	}
	return results, nil
}

func (m *MockDB) BatchStoreEvents(ctx context.Context, events []any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range events {
		if e, ok := event.(*core.BlockchainEvent); ok {
			m.events[e.ID] = e
		}
	}
	return nil
}

func (m *MockDB) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]*core.BlockchainEvent, 0, len(m.events))
	for _, event := range m.events {
		events = append(events, event)
	}
	return events, nil
}

func (m *MockDB) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blocks := make([]*core.Block, 0, len(m.blocks))
	for _, block := range m.blocks {
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (m *MockDB) DeleteEvent(ctx context.Context, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.events, eventID)
	return nil
}

func (m *MockDB) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]*core.BlockchainEvent, 0)
	for _, event := range m.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			events = append(events, event)
		}
	}
	return events, nil
}

func (m *MockDB) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	block, exists := m.blocks[blockNumber]
	if !exists {
		return nil, nil
	}
	return block, nil
}

func (m *MockDB) GetLatestBlock(ctx context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest uint64
	for blockNum := range m.blocks {
		if blockNum > latest {
			latest = blockNum
		}
	}
	return latest, nil
}

func (m *MockDB) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for id, event := range m.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			delete(m.events, id)
			count++
		}
	}
	return count, nil
}

func (m *MockDB) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for id, event := range m.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			event.Status = core.EventStatusReorged
			m.events[id] = event
			count++
		}
	}
	return count, nil
}

func (m *MockDB) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}
