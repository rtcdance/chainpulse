package reorg

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"chainpulse/pkg/core"
)

// ReorgHandler detects and recovers from blockchain reorganizations
type ReorgHandler struct {
	database core.DatabasePlugin
	logger   core.Logger
	mu       sync.RWMutex

	// State tracking
	lastKnownBlocks map[uint64]common.Hash // block number -> block hash
	reorgThreshold  uint64                  // blocks to keep for reorg detection
	maxRollback     uint64                  // maximum blocks to rollback
}

// ReorgEvent represents a detected reorganization
type ReorgEvent struct {
	DetectedAt    time.Time
	ReorgBlock    uint64
	BlocksAffected uint64
	OldBlockHash  common.Hash
	NewBlockHash  common.Hash
	EventsRolledBack int64
}

// NewReorgHandler creates a new reorg handler
func NewReorgHandler(
	database core.DatabasePlugin,
	logger core.Logger,
	reorgThreshold uint64,
	maxRollback uint64,
) *ReorgHandler {
	return &ReorgHandler{
		database:        database,
		logger:          logger,
		lastKnownBlocks: make(map[uint64]common.Hash),
		reorgThreshold:  reorgThreshold,
		maxRollback:     maxRollback,
	}
}

// DetectReorg detects if a reorg has occurred
func (rh *ReorgHandler) DetectReorg(
	ctx context.Context,
	currentBlock uint64,
	newBlockHash common.Hash,
) (bool, uint64, error) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	// Get the stored hash for this block
	storedHash, exists := rh.lastKnownBlocks[currentBlock]
	if !exists {
		// First time seeing this block, store it
		rh.lastKnownBlocks[currentBlock] = newBlockHash
		return false, 0, nil
	}

	// Check if hash matches
	if storedHash == newBlockHash {
		return false, 0, nil
	}

	// Reorg detected - find the reorg block
	reorgBlock := rh.findReorgBlock(ctx, currentBlock)
	if reorgBlock == 0 {
		return false, 0, fmt.Errorf("failed to find reorg block")
	}

	rh.logger.Warn(
		"Reorg detected",
		map[string]interface{}{
			"reorg_block": reorgBlock,
			"current_block": currentBlock,
			"blocks_affected": currentBlock - reorgBlock + 1,
		},
	)

	return true, reorgBlock, nil
}

// HandleReorg handles a detected reorganization
func (rh *ReorgHandler) HandleReorg(ctx context.Context, reorgBlock uint64) error {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	// Validate reorg block
	if reorgBlock == 0 {
		return fmt.Errorf("invalid reorg block: 0")
	}

	// Get current block from database
	currentBlock, err := rh.database.GetLatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	// Calculate blocks to rollback
	blocksToRollback := currentBlock - reorgBlock + 1
	if blocksToRollback > rh.maxRollback {
		return fmt.Errorf("reorg too large: %d blocks (max: %d)", blocksToRollback, rh.maxRollback)
	}

	// Rollback events
	eventsRolledBack, err := rh.RollbackEvents(ctx, reorgBlock)
	if err != nil {
		return fmt.Errorf("failed to rollback events: %w", err)
	}

	// Clean up block cache
	rh.cleanupBlockCache(reorgBlock)

	rh.logger.Info(
		"Reorg handled successfully",
		map[string]interface{}{
			"reorg_block": reorgBlock,
			"blocks_rolled_back": blocksToRollback,
			"events_rolled_back": eventsRolledBack,
		},
	)

	return nil
}

// RollbackEvents rolls back events from a specific block
func (rh *ReorgHandler) RollbackEvents(ctx context.Context, fromBlock uint64) (int64, error) {
	// Get events to rollback
	events, err := rh.database.GetEventsByBlockRange(ctx, fromBlock, ^uint64(0))
	if err != nil {
		return 0, fmt.Errorf("failed to get events: %w", err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	// Delete events
	count, err := rh.database.DeleteEventsByBlockRange(ctx, fromBlock, ^uint64(0))
	if err != nil {
		return 0, fmt.Errorf("failed to delete events: %w", err)
	}

	rh.logger.Info(
		"Events rolled back",
		map[string]interface{}{
			"from_block": fromBlock,
			"count": count,
		},
	)

	return count, nil
}

// findReorgBlock finds the block where reorg occurred
func (rh *ReorgHandler) findReorgBlock(ctx context.Context, currentBlock uint64) uint64 {
	// Start from current block and go backwards
	for block := currentBlock; block > 0; block-- {
		storedHash, exists := rh.lastKnownBlocks[block]
		if !exists {
			continue
		}

		// Get block from database
		dbBlock, err := rh.database.GetBlock(ctx, block)
		if err != nil {
			rh.logger.Error(
				"Failed to get block from database",
				map[string]interface{}{
					"block": block,
					"error": err.Error(),
				},
			)
			continue
		}

		if dbBlock == nil {
			continue
		}

		// Check if hash matches
		if storedHash != dbBlock.Hash {
			// Found the reorg point
			return block
		}
	}

	// If no mismatch found, reorg is at current block
	return currentBlock
}

// cleanupBlockCache removes old blocks from cache
func (rh *ReorgHandler) cleanupBlockCache(fromBlock uint64) {
	// Remove blocks older than reorg threshold
	cutoff := fromBlock - rh.reorgThreshold
	for block := range rh.lastKnownBlocks {
		if block < cutoff {
			delete(rh.lastKnownBlocks, block)
		}
	}
}

// UpdateBlockHash updates the known hash for a block
func (rh *ReorgHandler) UpdateBlockHash(blockNumber uint64, blockHash common.Hash) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.lastKnownBlocks[blockNumber] = blockHash

	// Cleanup old blocks
	thresholdWindow := rh.reorgThreshold
	if thresholdWindow <= math.MaxUint64/2 {
		thresholdWindow *= 2
	} else {
		thresholdWindow = math.MaxUint64
	}
	if uint64(len(rh.lastKnownBlocks)) > thresholdWindow {
		rh.cleanupBlockCache(blockNumber)
	}
}

// GetReorgStats returns reorg statistics
func (rh *ReorgHandler) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	// Get stats from database
	stats, err := rh.database.GetReorgStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get reorg stats: %w", err)
	}

	return stats, nil
}

// VerifyBlockSequence verifies that blocks form a valid sequence
func (rh *ReorgHandler) VerifyBlockSequence(ctx context.Context, fromBlock, toBlock uint64) error {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	for block := fromBlock; block < toBlock; block++ {
		currentBlock, err := rh.database.GetBlock(ctx, block)
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", block, err)
		}

		if currentBlock == nil {
			return fmt.Errorf("block %d not found", block)
		}

		nextBlock, err := rh.database.GetBlock(ctx, block+1)
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", block+1, err)
		}

		if nextBlock == nil {
			return fmt.Errorf("block %d not found", block+1)
		}

		// Verify parent hash
		if currentBlock.Hash != nextBlock.ParentHash {
			return fmt.Errorf(
				"block sequence broken at %d: hash %s != parent hash %s",
				block,
				currentBlock.Hash.Hex(),
				nextBlock.ParentHash.Hex(),
			)
		}
	}

	return nil
}

// GetLastKnownBlock returns the last known block hash
func (rh *ReorgHandler) GetLastKnownBlock() (uint64, common.Hash, bool) {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	var maxBlock uint64
	var maxHash common.Hash

	for block, hash := range rh.lastKnownBlocks {
		if block > maxBlock {
			maxBlock = block
			maxHash = hash
		}
	}

	return maxBlock, maxHash, maxBlock > 0
}

// Reset clears all reorg state
func (rh *ReorgHandler) Reset() {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.lastKnownBlocks = make(map[uint64]common.Hash)
}
