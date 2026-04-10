package indexing

import (
	"context"
	"fmt"
	"sync"

	"chainpulse/pkg/core"
)

// MonolithicMemoryDatabase provides a debug-friendly in-memory implementation
// of the indexing database contract used by monolithic mode.
type MonolithicMemoryDatabase struct {
	mu      sync.RWMutex
	started bool
	events  map[string]*core.BlockchainEvent
	blocks  map[uint64]*core.Block
	logger  core.Logger
}

// NewMonolithicMemoryDatabase creates a new in-memory database adapter.
func NewMonolithicMemoryDatabase(logger core.Logger) *MonolithicMemoryDatabase {
	return &MonolithicMemoryDatabase{
		events: make(map[string]*core.BlockchainEvent),
		blocks: make(map[uint64]*core.Block),
		logger: logger,
	}
}

func (db *MonolithicMemoryDatabase) Name() string    { return "monolithic-memory-database" }
func (db *MonolithicMemoryDatabase) Version() string { return "1.0.0" }
func (db *MonolithicMemoryDatabase) Initialize(config core.Config) error {
	return nil
}

func (db *MonolithicMemoryDatabase) Start() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.started = true
	return nil
}

func (db *MonolithicMemoryDatabase) Stop() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.started = false
	db.events = make(map[string]*core.BlockchainEvent)
	db.blocks = make(map[uint64]*core.Block)
	return nil
}

func (db *MonolithicMemoryDatabase) Health() error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if !db.started {
		return fmt.Errorf("database not started")
	}
	return nil
}

func (db *MonolithicMemoryDatabase) StoreEvent(ctx context.Context, event interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.started {
		return fmt.Errorf("database not started")
	}

	blockchainEvent, ok := event.(*core.BlockchainEvent)
	if !ok || blockchainEvent == nil {
		return fmt.Errorf("event must be *core.BlockchainEvent")
	}

	db.events[blockchainEvent.ID] = blockchainEvent
	return nil
}

func (db *MonolithicMemoryDatabase) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.events[id], nil
}

func (db *MonolithicMemoryDatabase) QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	results := make([]interface{}, 0, len(db.events))
	for _, event := range db.events {
		results = append(results, event)
	}
	return results, nil
}

func (db *MonolithicMemoryDatabase) BatchStoreEvents(ctx context.Context, events []interface{}) error {
	for _, event := range events {
		if err := db.StoreEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (db *MonolithicMemoryDatabase) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	results := make([]*core.BlockchainEvent, 0, len(db.events))
	for _, event := range db.events {
		results = append(results, event)
	}
	return results, nil
}

func (db *MonolithicMemoryDatabase) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	results := make([]*core.Block, 0, len(db.blocks))
	for _, block := range db.blocks {
		results = append(results, block)
	}
	return results, nil
}

func (db *MonolithicMemoryDatabase) DeleteEvent(ctx context.Context, eventID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	delete(db.events, eventID)
	return nil
}

func (db *MonolithicMemoryDatabase) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	results := make([]*core.BlockchainEvent, 0)
	for _, event := range db.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			results = append(results, event)
		}
	}
	return results, nil
}

func (db *MonolithicMemoryDatabase) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.blocks[blockNumber], nil
}

func (db *MonolithicMemoryDatabase) GetLatestBlock(ctx context.Context) (uint64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var maxBlock uint64
	for blockNumber := range db.blocks {
		if blockNumber > maxBlock {
			maxBlock = blockNumber
		}
	}
	for _, event := range db.events {
		if event.BlockNumber > maxBlock {
			maxBlock = event.BlockNumber
		}
	}
	return maxBlock, nil
}

func (db *MonolithicMemoryDatabase) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var deleted int64
	for id, event := range db.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			delete(db.events, id)
			deleted++
		}
	}
	return deleted, nil
}

func (db *MonolithicMemoryDatabase) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

// StoreBlockSnapshot records a minimal canonical block snapshot for monolithic
// runtime reorg detection.
func (db *MonolithicMemoryDatabase) StoreBlockSnapshot(ctx context.Context, block *core.Block) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.started {
		return fmt.Errorf("database not started")
	}
	if block == nil {
		return fmt.Errorf("block is nil")
	}

	copyBlock := *block
	db.blocks[block.Number] = &copyBlock
	return nil
}

// MonolithicMemoryCache provides a debug-friendly in-memory cache for the
// indexing contract used by monolithic mode.
type MonolithicMemoryCache struct {
	mu      sync.RWMutex
	started bool
	data    map[string][]byte
}

// NewMonolithicMemoryCache creates a new in-memory cache adapter.
func NewMonolithicMemoryCache() *MonolithicMemoryCache {
	return &MonolithicMemoryCache{
		data: make(map[string][]byte),
	}
}

func (c *MonolithicMemoryCache) Name() string    { return "monolithic-memory-cache" }
func (c *MonolithicMemoryCache) Version() string { return "1.0.0" }
func (c *MonolithicMemoryCache) Initialize(config core.Config) error {
	return nil
}

func (c *MonolithicMemoryCache) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started = true
	return nil
}

func (c *MonolithicMemoryCache) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.started = false
	return nil
}

func (c *MonolithicMemoryCache) Health() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.started {
		return fmt.Errorf("cache not started")
	}
	return nil
}

func (c *MonolithicMemoryCache) HealthCheck(ctx context.Context) error {
	return c.Health()
}

func (c *MonolithicMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key], nil
}

func (c *MonolithicMemoryCache) Set(ctx context.Context, key string, value []byte, ttl int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return fmt.Errorf("cache not started")
	}
	c.data[key] = append([]byte(nil), value...)
	return nil
}

func (c *MonolithicMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *MonolithicMemoryCache) GetStats() core.CacheStats {
	return core.CacheStats{}
}
