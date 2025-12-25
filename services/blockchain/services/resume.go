package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	"chainpulse/shared/database"
	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ResumeService handles breakpoint resume and event replay functionality
type ResumeService struct {
	client     *ethclient.Client
	db         *database.DB
	mu         sync.Mutex
	lastBlock  *big.Int
}

// NewResumeService creates a new resume service
func NewResumeService(client *ethclient.Client, db *database.DB) *ResumeService {
	return &ResumeService{
		client: client,
		db:     db,
	}
}

// GetLastProcessedBlock returns the last block number that was successfully processed
func (rs *ResumeService) GetLastProcessedBlock() (*big.Int, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// If we have it cached, return it
	if rs.lastBlock != nil {
		return rs.lastBlock, nil
	}

	// Otherwise, get from database
	blockNum, err := rs.db.GetLastProcessedBlock()
	if err != nil {
		return nil, err
	}

	rs.lastBlock = blockNum
	return rs.lastBlock, nil
}

// SaveLastProcessedBlock saves the last processed block number to database and cache
func (rs *ResumeService) SaveLastProcessedBlock(blockNum *big.Int) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	err := rs.db.SaveLastProcessedBlock(blockNum)
	if err != nil {
		return err
	}

	rs.lastBlock = blockNum
	return nil
}

// ReplayEvents replays events from a specific block range
func (rs *ResumeService) ReplayEvents(ctx context.Context, fromBlock, toBlock *big.Int) error {
	log.Printf("Starting event replay from block %s to %s", fromBlock.String(), toBlock.String())

	// Calculate the range
	current := new(big.Int).Set(fromBlock)
	
	// Process in batches to avoid overwhelming the system
	batchSize := big.NewInt(1000) // Process 1000 blocks at a time
	
	for current.Cmp(toBlock) <= 0 {
		endBlock := new(big.Int).Add(current, batchSize)
		if endBlock.Cmp(toBlock) > 0 {
			endBlock = toBlock
		}
		
		log.Printf("Processing batch: %s to %s", current.String(), endBlock.String())
		
		// Get logs for this batch
		query := ethereum.FilterQuery{
			FromBlock: current,
			ToBlock:   endBlock,
			Addresses: []common.Address{}, // This will be filled with specific contract addresses
		}
		
		logs, err := rs.client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to get logs for batch %s-%s: %v", current.String(), endBlock.String(), err)
		}
		
		// Process each log
		for _, vLog := range logs {
			event := &types.Event{
				BlockNumber: vLog.BlockNumber,
				TxHash:      vLog.TxHash.Hex(),
				Address:     vLog.Address.Hex(),
				Topics:      make([]string, len(vLog.Topics)),
				Data:        fmt.Sprintf("0x%x", vLog.Data),
				BlockHash:   vLog.BlockHash.Hex(),
				TxIndex:     uint(vLog.TxIndex),
				LogIndex:    uint(vLog.Index),
			}
			
			for i, topic := range vLog.Topics {
				event.Topics[i] = topic.Hex()
			}
			
			// Store the event in the database
			if err := rs.db.StoreEvent(event); err != nil {
				return fmt.Errorf("failed to store event: %v", err)
			}
		}
		
		// Update the last processed block after each batch
		if err := rs.SaveLastProcessedBlock(endBlock); err != nil {
			return fmt.Errorf("failed to save last processed block: %v", err)
		}
		
		// Move to next batch
		current = new(big.Int).Add(endBlock, big.NewInt(1))
		
		// Small delay to prevent overwhelming the node
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	log.Printf("Completed event replay from block %s to %s", fromBlock.String(), toBlock.String())
	return nil
}

// ResumeFromLastBlock resumes indexing from the last processed block
func (rs *ResumeService) ResumeFromLastBlock(ctx context.Context, addresses []common.Address) error {
	lastBlock, err := rs.GetLastProcessedBlock()
	if err != nil {
		log.Printf("Could not get last processed block, starting from block 0: %v", err)
		lastBlock = big.NewInt(0)
	} else {
		// Start from the next block after the last processed one
		lastBlock = new(big.Int).Add(lastBlock, big.NewInt(1))
	}
	
	// Get the current latest block
	latestBlock, err := rs.client.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %v", err)
	}
	
	log.Printf("Resuming from block %s to latest block %s", lastBlock.String(), latestBlock.Number().String())
	
	// Process events from last processed block to current
	query := ethereum.FilterQuery{
		FromBlock: lastBlock,
		ToBlock:   latestBlock.Number(),
		Addresses: addresses,
	}
	
	logs, err := rs.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get logs: %v", err)
	}
	
	// Process each log
	for _, vLog := range logs {
		event := &types.Event{
			BlockNumber: vLog.BlockNumber,
			TxHash:      vLog.TxHash.Hex(),
			Address:     vLog.Address.Hex(),
			Topics:      make([]string, len(vLog.Topics)),
			Data:        fmt.Sprintf("0x%x", vLog.Data),
			BlockHash:   vLog.BlockHash.Hex(),
			TxIndex:     uint(vLog.TxIndex),
			LogIndex:    uint(vLog.Index),
		}
		
		for i, topic := range vLog.Topics {
			event.Topics[i] = topic.Hex()
		}
		
		// Store the event in the database
		if err := rs.db.StoreEvent(event); err != nil {
			return fmt.Errorf("failed to store event: %v", err)
		}
		
		// Update the last processed block
		if err := rs.SaveLastProcessedBlock(big.NewInt(int64(vLog.BlockNumber))); err != nil {
			return fmt.Errorf("failed to save last processed block: %v", err)
		}
	}
	
	return nil
}

// ExportEvents exports events to a JSON file for backup or transfer
func (rs *ResumeService) ExportEvents(ctx context.Context, fromBlock, toBlock *big.Int, filePath string) error {
	events, err := rs.db.GetEventsByBlockRange(fromBlock, toBlock)
	if err != nil {
		return fmt.Errorf("failed to get events for export: %v", err)
	}
	
	file, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal events to JSON: %v", err)
	}
	
	// In a real implementation, you would write this to the actual file
	// For now, we'll just return the JSON as a string for demonstration
	log.Printf("Exported %d events to JSON format", len(events))
	
	return nil
}

// ImportEvents imports events from a JSON file
func (rs *ResumeService) ImportEvents(ctx context.Context, filePath string) error {
	// In a real implementation, you would read the JSON file and import events
	// For now, this is a placeholder
	log.Printf("Importing events from file: %s", filePath)
	
	return nil
}