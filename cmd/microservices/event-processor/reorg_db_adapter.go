package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// reorgDBAdapter adapts existing stores to core.DatabasePlugin
// for use by ReorgHandler in the event-processor microservice.
type reorgDBAdapter struct {
	EventStore    *query.MongoDBEventStore
	MetadataStore *query.PostgreSQLEventMetadataStore
	DB            *sql.DB
}

func newReorgDBAdapter(
	eventStore *query.MongoDBEventStore,
	metadataStore *query.PostgreSQLEventMetadataStore,
	db *sql.DB,
) *reorgDBAdapter {
	return &reorgDBAdapter{EventStore: eventStore, MetadataStore: metadataStore, DB: db}
}

func (a *reorgDBAdapter) Name() string                   { return "reorg-adapter" }
func (a *reorgDBAdapter) Version() string                { return "1.0.0" }
func (a *reorgDBAdapter) Initialize(_ core.Config) error { return nil }
func (a *reorgDBAdapter) Start() error                   { return nil }
func (a *reorgDBAdapter) Stop() error                    { return nil }
func (a *reorgDBAdapter) Health() error                  { return nil }

// EventReader
func (a *reorgDBAdapter) GetEvent(_ context.Context, _ string) (*blockchain.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgDBAdapter) QueryEvents(_ context.Context, _ any) ([]any, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgDBAdapter) GetAllEvents(_ context.Context) ([]*blockchain.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgDBAdapter) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error) {
	return a.EventStore.GetEventsByBlockRange(ctx, fromBlock, toBlock)
}

// EventWriter
func (a *reorgDBAdapter) StoreEvent(_ context.Context, _ any) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgDBAdapter) BatchStoreEvents(_ context.Context, _ []any) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgDBAdapter) DeleteEvent(_ context.Context, _ string) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgDBAdapter) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return a.EventStore.DeleteEventsByBlockRange(ctx, fromBlock, toBlock)
}

func (a *reorgDBAdapter) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return a.EventStore.MarkEventsAsReorged(ctx, fromBlock, toBlock)
}

// BlockReader
func (a *reorgDBAdapter) GetBlock(_ context.Context, blockNum uint64) (*blockchain.Block, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("reorg adapter: no database connection")
	}
	row := a.DB.QueryRow("SELECT block_hash, block_number FROM blockchain_events WHERE block_number = $1 LIMIT 1", blockNum)
	var hash string
	var number uint64
	if err := row.Scan(&hash, &number); err != nil {
		return nil, fmt.Errorf("reorg adapter: block %d not found: %w", blockNum, err)
	}
	return &blockchain.Block{Number: number, Hash: common.HexToHash(hash)}, nil
}

func (a *reorgDBAdapter) GetLatestBlock(_ context.Context) (uint64, error) {
	if a.DB == nil {
		return 0, fmt.Errorf("reorg adapter: no database connection")
	}
	var maxBlock uint64
	err := a.DB.QueryRow("SELECT COALESCE(MAX(block_number), 0) FROM blockchain_events").Scan(&maxBlock)
	if err != nil {
		return 0, fmt.Errorf("reorg adapter: failed to query latest block: %w", err)
	}
	return maxBlock, nil
}

func (a *reorgDBAdapter) GetAllBlocks(_ context.Context) ([]*blockchain.Block, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

// ReorgStatsProvider
func (a *reorgDBAdapter) GetReorgStats(_ context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}
