package eventproc

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// ReorgDBAdapter adapts existing stores to core.DatabasePlugin
// for use by ReorgHandler in the event-processor microservice.
type ReorgDBAdapter struct {
	EventStore    *query.MongoDBEventStore
	MetadataStore *query.PostgreSQLEventMetadataStore
	DB            *sql.DB
}

func NewReorgDBAdapter(
	eventStore *query.MongoDBEventStore,
	metadataStore *query.PostgreSQLEventMetadataStore,
	db *sql.DB,
) *ReorgDBAdapter {
	return &ReorgDBAdapter{EventStore: eventStore, MetadataStore: metadataStore, DB: db}
}

func (a *ReorgDBAdapter) Name() string                   { return "reorg-adapter" }
func (a *ReorgDBAdapter) Version() string                { return "1.0.0" }
func (a *ReorgDBAdapter) Initialize(_ core.Config) error { return nil }
func (a *ReorgDBAdapter) Start() error                   { return nil }
func (a *ReorgDBAdapter) Stop() error                    { return nil }
func (a *ReorgDBAdapter) Health() error                  { return nil }

// EventReader
func (a *ReorgDBAdapter) GetEvent(_ context.Context, _ string) (*core.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *ReorgDBAdapter) QueryEvents(_ context.Context, _ any) ([]any, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *ReorgDBAdapter) GetAllEvents(_ context.Context) ([]*core.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *ReorgDBAdapter) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	return a.EventStore.GetEventsByBlockRange(ctx, fromBlock, toBlock)
}

// EventWriter
func (a *ReorgDBAdapter) StoreEvent(_ context.Context, _ any) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *ReorgDBAdapter) BatchStoreEvents(_ context.Context, _ []any) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *ReorgDBAdapter) DeleteEvent(_ context.Context, _ string) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *ReorgDBAdapter) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return a.EventStore.DeleteEventsByBlockRange(ctx, fromBlock, toBlock)
}

func (a *ReorgDBAdapter) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return a.EventStore.MarkEventsAsReorged(ctx, fromBlock, toBlock)
}

// BlockReader
func (a *ReorgDBAdapter) GetBlock(_ context.Context, blockNum uint64) (*core.Block, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("reorg adapter: no database connection")
	}
	row := a.DB.QueryRow("SELECT block_hash, block_number FROM blockchain_events WHERE block_number = $1 LIMIT 1", blockNum)
	var hash string
	var number uint64
	if err := row.Scan(&hash, &number); err != nil {
		return nil, fmt.Errorf("reorg adapter: block %d not found: %w", blockNum, err)
	}
	return &core.Block{Number: number, Hash: common.HexToHash(hash)}, nil
}

func (a *ReorgDBAdapter) GetLatestBlock(_ context.Context) (uint64, error) {
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

func (a *ReorgDBAdapter) GetAllBlocks(_ context.Context) ([]*core.Block, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

// ReorgStatsProvider
func (a *ReorgDBAdapter) GetReorgStats(_ context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}
