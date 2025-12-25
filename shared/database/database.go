package database

import (
	"context"
	"math/big"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"chainpulse/shared/types"
)

type Database struct {
	DB *gorm.DB
}

// DB is an alias for Database to maintain compatibility
type DB = Database

func NewDatabase(dsn string) (*Database, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&types.IndexedEvent{}, &types.LastProcessedBlock{}, &types.ProcessedEvent{}, &types.Contract{})
	if err != nil {
		return nil, err
	}

	return &Database{
		DB: db,
	}, nil
}

func (d *Database) SaveEvent(event *types.IndexedEvent) error {
	return d.DB.Create(event).Error
}

func (d *Database) SaveContract(contract *types.Contract) error {
	return d.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(contract).Error
}

func (d *Database) GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error) {
	var events []types.IndexedEvent
	query := d.DB.Model(&types.IndexedEvent{})

	if filter.Contract != "" {
		query = query.Where("contract = ?", filter.Contract)
	}

	if filter.EventType != "" {
		query = query.Where("event_name = ?", filter.EventType)
	}

	if filter.FromBlock != nil {
		query = query.Where("block_number >= ?", filter.FromBlock)
	}

	if filter.ToBlock != nil {
		query = query.Where("block_number <= ?", filter.ToBlock)
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	query = query.Order("block_number DESC, created_at DESC")

	err := query.Find(&events).Error
	return events, err
}

func (d *Database) GetEventByID(id uint) (*types.IndexedEvent, error) {
	var event types.IndexedEvent
	err := d.DB.First(&event, id).Error
	return &event, err
}

func (d *Database) GetLatestBlockProcessed() (*types.IndexedEvent, error) {
	var event types.IndexedEvent
	err := d.DB.Order("block_number DESC").First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (d *Database) GetLastProcessedBlock() (*big.Int, error) {
	var lastBlock types.LastProcessedBlock
	err := d.DB.Order("created_at DESC").First(&lastBlock).Error
	if err != nil {
		// If no record is found, return 0
		if err == gorm.ErrRecordNotFound {
			return big.NewInt(0), nil
		}
		return nil, err
	}
	return lastBlock.BlockNumber, nil
}

func (d *Database) GetEvents(limitNum, offset int) ([]types.IndexedEvent, error) {
	var events []types.IndexedEvent
	err := d.DB.Limit(limitNum).Offset(offset).Order("block_number DESC, created_at DESC").Find(&events).Error
	return events, err
}

func (d *Database) GetEventByTxHash(txHash string) (*types.IndexedEvent, error) {
	var event types.IndexedEvent
	err := d.DB.Where("tx_hash = ?", txHash).First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

func (d *Database) GetEventsByBlockNumber(blockNumber int64) ([]types.IndexedEvent, error) {
	var events []types.IndexedEvent
	err := d.DB.Where("block_number = ?", blockNumber).Find(&events).Error
	return events, err
}

func (d *Database) GetContracts() ([]types.Contract, error) {
	var contracts []types.Contract
	err := d.DB.Find(&contracts).Error
	return contracts, err
}

func (d *Database) GetContractByAddress(address string) (*types.Contract, error) {
	var contract types.Contract
	err := d.DB.Where("address = ?", address).First(&contract).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &contract, nil
}

func (d *Database) GetStats() (*types.Stats, error) {
	var stats types.Stats
	
	// Count total events
	var eventCount int64
	err := d.DB.Model(&types.IndexedEvent{}).Count(&eventCount).Error
	if err != nil {
		return nil, err
	}
	
	// Get latest block processed
	var latestEvent types.IndexedEvent
	err = d.DB.Model(&types.IndexedEvent{}).Order("block_number DESC").First(&latestEvent).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Count total contracts
	var contractCount int64
	err = d.DB.Model(&types.Contract{}).Count(&contractCount).Error
	if err != nil {
		return nil, err
	}
	
	stats.TotalEvents = eventCount
	stats.TotalContracts = contractCount
	stats.LatestBlock = latestEvent.BlockNumber.Int64()
	
	return &stats, nil
}

func (d *Database) GetLastProcessedBlockByNumber(blockNumber *big.Int) (*types.LastProcessedBlock, error) {
	var lastBlock types.LastProcessedBlock
	err := d.DB.Where("block_number = ?", blockNumber).First(&lastBlock).Error
	if err != nil {
		return nil, err
	}
	return &lastBlock, nil
}

func (d *Database) SaveLastProcessedBlock(blockNum *big.Int) error {
	// Try to find an existing record for the same chain
	var existing types.LastProcessedBlock
	err := d.DB.Where("chain_id = ?", "ethereum_mainnet").First(&existing).Error // Using default chain ID
	
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	
	// If record exists, update it; otherwise, create a new one
	if err == nil {
		existing.BlockNumber = blockNum
		return d.DB.Save(&existing).Error
	} else {
		// Create a new record
		lastBlock := &types.LastProcessedBlock{
			BlockNumber: blockNum,
			ChainID:     "ethereum_mainnet", // Default chain ID, should be configurable
		}
		return d.DB.Create(lastBlock).Error
	}
}

func (d *Database) UpdateLastProcessedBlockWithHash(blockNum *big.Int, blockHash string) error {
	// Try to find an existing record for the same chain
	var existing types.LastProcessedBlock
	err := d.DB.Where("chain_id = ?", "ethereum_mainnet").First(&existing).Error
	
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	
	// If record exists, update it; otherwise, create a new one
	if err == nil {
		existing.BlockNumber = blockNum
		existing.BlockHash = blockHash
		return d.DB.Save(&existing).Error
	} else {
		// Create a new record
		lastBlock := &types.LastProcessedBlock{
			BlockNumber: blockNum,
			BlockHash:   blockHash,
			ChainID:     "ethereum_mainnet", // Default chain ID, should be configurable
		}
		return d.DB.Create(lastBlock).Error
	}
}

func (d *Database) DeleteEventsFromBlock(blockNumber *big.Int) error {
	return d.DB.Where("block_number >= ?", blockNumber).Delete(&types.IndexedEvent{}).Error
}

func (d *Database) DeleteProcessedEventsFromBlock(blockNumber *big.Int) error {
	return d.DB.Where("block_number >= ?", blockNumber).Delete(&types.ProcessedEvent{}).Error
}

func (d *Database) GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error) {
	var events []types.IndexedEvent
	err := d.DB.Where("block_number >= ? AND block_number <= ?", fromBlock, toBlock).
		Order("block_number ASC").
		Find(&events).Error
	return events, err
}

func (d *Database) EventExists(eventKey string) (bool, error) {
	var count int64
	err := d.DB.Model(&types.ProcessedEvent{}).
		Where("event_key = ?", eventKey).
		Count(&count).Error
	
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

func (d *Database) MarkEventAsProcessed(eventKey string) error {
	processedEvent := &types.ProcessedEvent{
		EventKey:  eventKey,
		Processed: true,
		Timestamp: time.Now(),
	}
	
	return d.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(processedEvent).Error
}

func (d *Database) MarkEventAsProcessedWithTx(tx *gorm.DB, eventKey string) error {
	processedEvent := &types.ProcessedEvent{
		EventKey:  eventKey,
		Processed: true,
		Timestamp: time.Now(),
	}
	
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(processedEvent).Error
}

// Ping checks if the database connection is alive
func (d *Database) Ping(ctx context.Context) error {
	db, err := d.DB.DB()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}