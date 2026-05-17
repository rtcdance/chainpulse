package reorg

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
)

// ReorgHandler detects and recovers from blockchain reorganizations.
//
// Renaming would break many external uses.
type ReorgHandler struct {
	database core.DatabasePlugin
	logger   core.Logger
	mu       sync.RWMutex

	// Chain identity and event publishing
	chainID  string
	eventBus core.EventBus

	// Persistent checkpoint store (optional — falls back to in-memory if nil)
	checkpointStore core.CheckpointStore

	// Canonical chain hash provider — compares locally-indexed hashes
	// against the live chain. Defaults to database (backward compat);
	// production should inject an RPC-backed provider.
	blockHashProvider core.BlockHashProvider

	// Idempotency invalidator — clears idempotency entries for reorged
	// block ranges so that re-indexed events are not rejected as duplicates.
	idempotencyInvalidator core.IdempotencyInvalidator

	// State tracking
	lastKnownBlocks map[uint64]common.Hash // block number -> block hash (LRU cache backed by DB)
	reorgThreshold  uint64                 // blocks to keep for reorg detection
	maxRollback     uint64                 // maximum blocks to rollback
}

// ReorgEvent represents a detected reorganization.
//
// Renaming would break many external uses.
type ReorgEvent struct {
	DetectedAt       time.Time
	ReorgBlock       uint64
	BlocksAffected   uint64
	OldBlockHash     common.Hash
	NewBlockHash     common.Hash
	EventsRolledBack int64
}

// NewReorgHandler creates a new reorg handler
func NewReorgHandler(
	database core.DatabasePlugin,
	logger core.Logger,
	reorgThreshold uint64,
	maxRollback uint64,
) *ReorgHandler {
	rh := &ReorgHandler{
		database:          database,
		logger:            logger,
		lastKnownBlocks:   make(map[uint64]common.Hash),
		reorgThreshold:    reorgThreshold,
		maxRollback:       maxRollback,
		blockHashProvider: &DatabaseBlockHashProvider{db: database},
	}
	return rh
}

// SetBlockHashProvider sets the provider used to fetch canonical chain block hashes.
// Production code should inject an RPC-backed provider so reorg detection compares
// local hashes against the live chain. Defaults to database lookup.
func (rh *ReorgHandler) SetBlockHashProvider(provider core.BlockHashProvider) {
	rh.mu.Lock()
	rh.blockHashProvider = provider
	rh.mu.Unlock()
}

// SetIdempotencyInvalidator sets the invalidator used to clear idempotency entries
// for reorged block ranges. Without this, re-indexed events after a reorg are
// incorrectly rejected as duplicates.
func (rh *ReorgHandler) SetIdempotencyInvalidator(invalidator core.IdempotencyInvalidator) {
	rh.mu.Lock()
	rh.idempotencyInvalidator = invalidator
	rh.mu.Unlock()
}

// DatabaseBlockHashProvider fetches block hashes from the local database.
// This is the default (backward-compatible) provider but is insufficient for
// reorg detection because the database contains pre-reorg data.
type DatabaseBlockHashProvider struct {
	db core.DatabasePlugin
}

func (p *DatabaseBlockHashProvider) GetBlockHash(ctx context.Context, blockNumber uint64) (common.Hash, error) {
	block, err := p.db.GetBlock(ctx, blockNumber)
	if err != nil {
		return common.Hash{}, err
	}
	if block == nil {
		return common.Hash{}, nil
	}
	return block.Hash, nil
}

// WithChainID sets the chain identifier for reorg event publishing.
func (rh *ReorgHandler) WithChainID(chainID string) *ReorgHandler {
	rh.chainID = chainID
	return rh
}

// WithEventBus sets the event bus for publishing reorg events.
func (rh *ReorgHandler) WithEventBus(eventBus core.EventBus) *ReorgHandler {
	rh.eventBus = eventBus
	return rh
}

// WithCheckpointStore sets the checkpoint store for persisting block hashes across restarts.
func (rh *ReorgHandler) WithCheckpointStore(store core.CheckpointStore) *ReorgHandler {
	rh.checkpointStore = store
	return rh
}

// DetectReorg detects if a reorg has occurred
func (rh *ReorgHandler) DetectReorg(
	ctx context.Context,
	currentBlock uint64,
	newBlockHash common.Hash,
) (bool, uint64, error) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	// Get the stored hash for this block (check in-memory cache first)
	storedHash, exists := rh.lastKnownBlocks[currentBlock]
	if !exists && rh.checkpointStore != nil {
		// Fall back to DB-backed checkpoint store
		dbHash, err := rh.checkpointStore.GetBlockHash(ctx, rh.chainID, currentBlock)
		if err == nil && dbHash != "" {
			storedHash = common.HexToHash(dbHash)
			exists = true
			// Cache it for next time
			rh.lastKnownBlocks[currentBlock] = storedHash
		}
	}
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
	reorgBlock, err := rh.findReorgBlock(ctx, currentBlock)
	if err != nil {
		rh.logger.Warn("reorg scan failed", core.LogKeyError, err)
		return true, currentBlock, nil // Report reorg at current block as best guess
	}
	if reorgBlock == 0 {
		return false, 0, fmt.Errorf("failed to find reorg block")
	}

	rh.logger.Warn(
		"Reorg detected",
		core.LogKeyReorgBlock, reorgBlock,
		core.LogKeyCurrentBlock, currentBlock,
		core.LogKeyBlocksAffected, currentBlock-reorgBlock+1,
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
	eventsRolledBack, err := rh.RollbackEvents(ctx, reorgBlock, currentBlock)
	if err != nil {
		return fmt.Errorf("failed to rollback events: %w", err)
	}

	// Clean up block cache
	rh.cleanupBlockCache(reorgBlock)

	// Publish reorg event if event bus is configured
	if rh.eventBus != nil {
		reorgEvt := &ReorgEvent{
			DetectedAt:       time.Now(),
			ReorgBlock:       reorgBlock,
			BlocksAffected:   blocksToRollback,
			EventsRolledBack: eventsRolledBack,
		}
		if err := rh.eventBus.Publish(ctx, core.TopicReorgDetected, reorgEvt); err != nil {
			rh.logger.Error("Failed to publish reorg event", core.LogKeyError, err)
		}

		// Publish re-index trigger event so the puller can re-pull affected blocks
		reindexEvt := &core.ReorgRollbackEvent{
			ChainID:    rh.chainID,
			FromBlock:  reorgBlock,
			ToBlock:    currentBlock,
			DetectedAt: time.Now(),
		}
		if err := rh.eventBus.Publish(ctx, core.TopicReorgRollback, reindexEvt); err != nil {
			rh.logger.Error("Failed to publish reorg rollback event", core.LogKeyError, err)
		}
	}

	rh.logger.Info(
		"Reorg handled successfully",
		core.LogKeyReorgBlock, reorgBlock,
		core.LogKeyBlocksRolledBack, blocksToRollback,
		core.LogKeyEventsRolledBack, eventsRolledBack,
	)

	return nil
}

// RollbackEvents marks events as reorged (soft delete) instead of hard-deleting them.
// currentBlock is the highest known block at the time of reorg detection — only events
// in [fromBlock, currentBlock] are marked, not future blocks.
func (rh *ReorgHandler) RollbackEvents(ctx context.Context, fromBlock, currentBlock uint64) (int64, error) {
	// Mark events as reorged instead of deleting
	count, err := rh.database.MarkEventsAsReorged(ctx, fromBlock, currentBlock)
	if err != nil {
		return 0, fmt.Errorf("failed to mark events as reorged: %w", err)
	}

	// Clear idempotency entries for the reorged range so that
	// re-indexed events are not rejected as duplicates.
	if rh.idempotencyInvalidator != nil {
		invalidated := rh.idempotencyInvalidator.InvalidateRange(fromBlock, currentBlock)
		rh.logger.Info(
			"Idempotency entries invalidated for reorged range",
			core.LogKeyFromBlock, fromBlock,
			core.LogKeyCurrentBlock, currentBlock,
			core.LogKeyInvalidated, invalidated,
		)
	}

	rh.logger.Info(
		"Events marked as reorged",
		core.LogKeyFromBlock, fromBlock,
		core.LogKeyCurrentBlock, currentBlock,
		core.LogKeyCount, count,
	)

	return count, nil
}

// findReorgBlock finds the block where reorg occurred.
// Uses binary search for efficiency (O(log n) vs O(n) linear scan),
// falling back to linear scan on RPC errors.
func (rh *ReorgHandler) findReorgBlock(ctx context.Context, currentBlock uint64) (uint64, error) {
	maxScanDepth := uint64(256) // Prevent unbounded scan
	if rh.maxRollback > 0 && rh.maxRollback < maxScanDepth {
		maxScanDepth = rh.maxRollback
	}

	// Determine the lower bound for the search
	lowerBound := uint64(1)
	if currentBlock > maxScanDepth {
		lowerBound = currentBlock - maxScanDepth
	}

	// Try binary search first
	if reorgBlock, err := rh.binarySearchReorg(ctx, lowerBound, currentBlock); err == nil {
		return reorgBlock, nil
	}

	// Fallback to linear scan if binary search fails (e.g., RPC errors)
	rh.logger.Warn("binary search reorg detection failed, falling back to linear scan", "error", "binary_search_unavailable")
	return rh.linearScanReorg(ctx, currentBlock, maxScanDepth)
}

// binarySearchReorg performs a binary search between lowerBound and currentBlock
// to find the exact reorg divergence point. For each candidate block, it compares
// the locally-indexed hash against the canonical chain hash (via blockHashProvider).
// Time complexity: O(log n).
func (rh *ReorgHandler) binarySearchReorg(ctx context.Context, lowerBound, currentBlock uint64) (uint64, error) {
	lo, hi := lowerBound, currentBlock
	result := currentBlock // Default: assume reorg at current block

	for lo <= hi {
		select {
		case <-ctx.Done():
			return currentBlock, ctx.Err()
		default:
		}

		mid := lo + (hi-lo)/2

		storedHash, exists := rh.lastKnownBlocks[mid]
		if !exists {
			// No stored hash for this block — move to higher blocks
			lo = mid + 1
			continue
		}

		canonicalHash, err := rh.blockHashProvider.GetBlockHash(ctx, mid)
		if err != nil {
			// Can't verify — return error to trigger linear fallback
			return 0, fmt.Errorf("block hash lookup failed at block %d: %w", mid, err)
		}

		if canonicalHash == (common.Hash{}) {
			// No block on canonical chain — move to higher blocks
			lo = mid + 1
			continue
		}

		if storedHash != canonicalHash {
			// Reorg at or before this block
			result = mid
			hi = mid - 1 // Search lower half for earlier divergence
		} else {
			// No reorg at this block — search upper half
			lo = mid + 1
		}
	}

	return result, nil
}

// linearScanReorg scans backwards from currentBlock to find the reorg point (O(n)).
func (rh *ReorgHandler) linearScanReorg(ctx context.Context, currentBlock, maxScanDepth uint64) (uint64, error) {
	scanned := uint64(0)
	for block := currentBlock; block > 0 && scanned < maxScanDepth; block-- {
		scanned++

		select {
		case <-ctx.Done():
			return currentBlock, ctx.Err()
		default:
		}

		storedHash, exists := rh.lastKnownBlocks[block]
		if !exists {
			continue
		}

		canonicalHash, err := rh.blockHashProvider.GetBlockHash(ctx, block)
		if err != nil {
			rh.logger.Error(
				"Failed to get canonical block hash",
				core.LogKeyBlockNumber, block,
				core.LogKeyError, err,
			)
			continue
		}

		if canonicalHash == (common.Hash{}) {
			continue
		}

		if storedHash != canonicalHash {
			return block, nil
		}
	}

	if scanned >= maxScanDepth {
		return currentBlock, fmt.Errorf("reorg scan exceeded max depth %d without finding divergence point", maxScanDepth)
	}

	return currentBlock, nil
}

// cleanupBlockCache removes old blocks from cache
func (rh *ReorgHandler) cleanupBlockCache(fromBlock uint64) {
	// Remove blocks older than reorg threshold
	var cutoff uint64
	if fromBlock > rh.reorgThreshold {
		cutoff = fromBlock - rh.reorgThreshold
	}
	// If fromBlock <= reorgThreshold, cutoff stays 0 — nothing is evicted
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
