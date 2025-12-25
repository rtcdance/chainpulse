package database

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"chainpulse/shared/cache"
	"chainpulse/shared/types"

	"gorm.io/gorm"
)

// CachedDatabase wraps the database with caching functionality
type CachedDatabase struct {
	DB    *Database
	Cache *cache.Cache
}

// NewCachedDatabase creates a new database instance with caching
func NewCachedDatabase(dsn string, cache *cache.Cache) (*CachedDatabase, error) {
	db, err := NewDatabase(dsn)
	if err != nil {
		return nil, err
	}

	return &CachedDatabase{
		DB:    db,
		Cache: cache,
	}, nil
}

// GetEventByTxHash retrieves an event by transaction hash with caching
func (cd *CachedDatabase) GetEventByTxHash(txHash string) (*types.IndexedEvent, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("event:tx_hash:%s", txHash)

	// Try to get from cache first
	var event types.IndexedEvent
	err := cd.Cache.Get(ctx, cacheKey, &event)
	if err == nil {
		return &event, nil
	}

	// Cache miss, get from database
	dbEvent, err := cd.DB.GetEventByTxHash(txHash)
	if err != nil {
		return nil, err
	}

	if dbEvent != nil {
		// Cache the result for 5 minutes
		go func() {
			if err := cd.Cache.Set(ctx, cacheKey, dbEvent, 5*time.Minute); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Error caching event: %v\n", err)
			}
		}()
	}

	return dbEvent, nil
}

// GetContractByAddress retrieves a contract by address with caching
func (cd *CachedDatabase) GetContractByAddress(address string) (*types.Contract, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("contract:address:%s", address)

	// Try to get from cache first
	var contract types.Contract
	err := cd.Cache.Get(ctx, cacheKey, &contract)
	if err == nil {
		return &contract, nil
	}

	// Cache miss, get from database
	dbContract, err := cd.DB.GetContractByAddress(address)
	if err != nil {
		return nil, err
	}

	if dbContract != nil {
		// Cache the result for 10 minutes
		go func() {
			if err := cd.Cache.Set(ctx, cacheKey, dbContract, 10*time.Minute); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Error caching contract: %v\n", err)
			}
		}()
	}

	return dbContract, nil
}

// GetStats retrieves statistics with caching
func (cd *CachedDatabase) GetStats() (*types.Stats, error) {
	ctx := context.Background()
	cacheKey := "stats:overview"

	// Try to get from cache first
	var stats types.Stats
	err := cd.Cache.Get(ctx, cacheKey, &stats)
	if err == nil {
		return &stats, nil
	}

	// Cache miss, get from database
	dbStats, err := cd.DB.GetStats()
	if err != nil {
		return nil, err
	}

	// Cache the result for 1 minute (shorter for stats as they change frequently)
	go func() {
		if err := cd.Cache.Set(ctx, cacheKey, dbStats, 1*time.Minute); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Error caching stats: %v\n", err)
		}
	}()

	return dbStats, nil
}

// GetEvents retrieves events with optional caching
func (cd *CachedDatabase) GetEvents(limitNum, offset int) ([]types.IndexedEvent, error) {
	// For events, we don't cache the entire result set as it changes frequently
	// Instead, we'll just pass through to the database
	return cd.DB.GetEvents(limitNum, offset)
}

// GetEventsByBlockNumber retrieves events by block number with caching
func (cd *CachedDatabase) GetEventsByBlockNumber(blockNumber int64) ([]types.IndexedEvent, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("events:block_number:%d", blockNumber)

	// Try to get from cache first
	var events []types.IndexedEvent
	err := cd.Cache.Get(ctx, cacheKey, &events)
	if err == nil {
		return events, nil
	}

	// Cache miss, get from database
	dbEvents, err := cd.DB.GetEventsByBlockNumber(blockNumber)
	if err != nil {
		return nil, err
	}

	if len(dbEvents) > 0 {
		// Cache the result for 2 minutes
		go func() {
			if err := cd.Cache.Set(ctx, cacheKey, dbEvents, 2*time.Minute); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Error caching events by block: %v\n", err)
			}
		}()
	}

	return dbEvents, nil
}

// GetLatestBlockProcessed retrieves the latest processed block with caching
func (cd *CachedDatabase) GetLatestBlockProcessed() (*types.IndexedEvent, error) {
	ctx := context.Background()
	cacheKey := "event:latest_block"

	// Try to get from cache first
	var event types.IndexedEvent
	err := cd.Cache.Get(ctx, cacheKey, &event)
	if err == nil {
		return &event, nil
	}

	// Cache miss, get from database
	dbEvent, err := cd.DB.GetLatestBlockProcessed()
	if err != nil {
		return nil, err
	}

	if dbEvent != nil {
		// Cache for 30 seconds (since this changes frequently)
		go func() {
			if err := cd.Cache.Set(ctx, cacheKey, dbEvent, 30*time.Second); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Error caching latest block: %v\n", err)
			}
		}()
	}

	return dbEvent, nil
}

// GetLastProcessedBlock retrieves the last processed block with caching
func (cd *CachedDatabase) GetLastProcessedBlock() (*big.Int, error) {
	ctx := context.Background()
	cacheKey := "block:last_processed"

	// Try to get from cache first
	var blockNumber big.Int
	err := cd.Cache.Get(ctx, cacheKey, &blockNumber)
	if err == nil {
		return &blockNumber, nil
	}

	// Cache miss, get from database
	dbBlock, err := cd.DB.GetLastProcessedBlock()
	if err != nil {
		return nil, err
	}

	// Cache for 1 minute
	go func() {
		if err := cd.Cache.Set(ctx, cacheKey, dbBlock, 1*time.Minute); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Error caching last processed block: %v\n", err)
		}
	}()

	return dbBlock, nil
}

// InvalidateEventCache removes cached event data
func (cd *CachedDatabase) InvalidateEventCache(txHash string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("event:tx_hash:%s", txHash)
	return cd.Cache.Delete(ctx, cacheKey)
}

// InvalidateContractCache removes cached contract data
func (cd *CachedDatabase) InvalidateContractCache(address string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("contract:address:%s", address)
	return cd.Cache.Delete(ctx, cacheKey)
}

// InvalidateBlockCache removes cached block data
func (cd *CachedDatabase) InvalidateBlockCache(blockNumber int64) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("events:block_number:%d", blockNumber)
	return cd.Cache.Delete(ctx, cacheKey)
}

// All other database methods that don't need caching just pass through to the underlying DB
func (cd *CachedDatabase) SaveEvent(event *types.IndexedEvent) error {
	return cd.DB.SaveEvent(event)
}

func (cd *CachedDatabase) SaveContract(contract *types.Contract) error {
	err := cd.DB.SaveContract(contract)
	if err == nil {
		// Invalidate the contract cache when saving
		go func() {
			if err := cd.InvalidateContractCache(contract.Address); err != nil {
				fmt.Printf("Error invalidating contract cache: %v\n", err)
			}
		}()
	}
	return err
}

func (cd *CachedDatabase) GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error) {
	return cd.DB.GetEvents(filter)
}

func (cd *CachedDatabase) GetEventByID(id uint) (*types.IndexedEvent, error) {
	// For GetEventByID, we could implement caching, but for now we'll just pass through
	return cd.DB.GetEventByID(id)
}

func (cd *CachedDatabase) GetContracts() ([]types.Contract, error) {
	// For now, just pass through to the database
	return cd.DB.GetContracts()
}

func (cd *CachedDatabase) GetLastProcessedBlockByNumber(blockNumber *big.Int) (*types.LastProcessedBlock, error) {
	return cd.DB.GetLastProcessedBlockByNumber(blockNumber)
}

func (cd *CachedDatabase) SaveLastProcessedBlock(blockNum *big.Int) error {
	err := cd.DB.SaveLastProcessedBlock(blockNum)
	if err == nil {
		// Invalidate the last processed block cache
		go func() {
			if err := cd.Cache.Delete(context.Background(), "block:last_processed"); err != nil {
				fmt.Printf("Error invalidating last processed block cache: %v\n", err)
			}
		}()
	}
	return err
}

func (cd *CachedDatabase) UpdateLastProcessedBlockWithHash(blockNum *big.Int, blockHash string) error {
	err := cd.DB.UpdateLastProcessedBlockWithHash(blockNum, blockHash)
	if err == nil {
		// Invalidate the last processed block cache
		go func() {
			if err := cd.Cache.Delete(context.Background(), "block:last_processed"); err != nil {
				fmt.Printf("Error invalidating last processed block cache: %v\n", err)
			}
		}()
	}
	return err
}

func (cd *CachedDatabase) DeleteEventsFromBlock(blockNumber *big.Int) error {
	err := cd.DB.DeleteEventsFromBlock(blockNumber)
	if err == nil {
		// Invalidate any cached data for this block range
		go func() {
			blockNum := blockNumber.Int64()
			if err := cd.InvalidateBlockCache(blockNum); err != nil {
				fmt.Printf("Error invalidating block cache: %v\n", err)
			}
		}()
	}
	return err
}

func (cd *CachedDatabase) DeleteProcessedEventsFromBlock(blockNumber *big.Int) error {
	return cd.DB.DeleteProcessedEventsFromBlock(blockNumber)
}

func (cd *CachedDatabase) GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error) {
	return cd.DB.GetEventsByBlockRange(fromBlock, toBlock)
}

func (cd *CachedDatabase) EventExists(eventKey string) (bool, error) {
	return cd.DB.EventExists(eventKey)
}

func (cd *CachedDatabase) MarkEventAsProcessed(eventKey string) error {
	return cd.DB.MarkEventAsProcessed(eventKey)
}

func (cd *CachedDatabase) MarkEventAsProcessedWithTx(tx *gorm.DB, eventKey string) error {
	return cd.DB.MarkEventAsProcessedWithTx(tx, eventKey)
}

func (cd *CachedDatabase) Ping(ctx context.Context) error {
	return cd.DB.Ping(ctx)
}