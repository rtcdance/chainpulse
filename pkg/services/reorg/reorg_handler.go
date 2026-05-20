package reorg

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
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

	// Confirmation depth: number of blocks after which a block is considered
	// "safe" from reorg. Varies by chain: Ethereum ~12, BSC ~15, Polygon ~128.
	confirmationDepth uint64

	// Current chain head for IsConfirmed checks
	chainHead uint64

	// Checkpoint batching: persist hashes to checkpointStore every N blocks
	// to survive restarts without overwhelming the store.
	lastPersistedBlock uint64
	checkpointInterval uint64 // blocks between checkpoint writes (default: 10)
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
		checkpointInterval: 10,
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

// WithConfirmationDepth sets the confirmation depth for finality checks.
// Typical values: Ethereum 12, BSC 15, Polygon 128, Arbitrum 0 (instant finality).
func (rh *ReorgHandler) WithConfirmationDepth(depth uint64) *ReorgHandler {
	rh.confirmationDepth = depth
	return rh
}

// UpdateChainHead updates the known chain head for confirmation depth checks.
func (rh *ReorgHandler) UpdateChainHead(head uint64) {
	rh.mu.Lock()
	rh.chainHead = head
	rh.mu.Unlock()
}

// IsConfirmed returns true if the given block number has at least
// confirmationDepth blocks built on top of it (i.e., is safe from reorg).
func (rh *ReorgHandler) IsConfirmed(blockNumber uint64) bool {
	rh.mu.RLock()
	chainHead := rh.chainHead
	depth := rh.confirmationDepth
	rh.mu.RUnlock()

	if depth == 0 {
		return true
	}

	return blockNumber+depth <= chainHead
}

// ConfirmationDepth returns the configured confirmation depth.
func (rh *ReorgHandler) ConfirmationDepth() uint64 {
	rh.mu.RLock()
	defer rh.mu.RUnlock()
	return rh.confirmationDepth
}

// DetectReorg detects if a reorg has occurred
func (rh *ReorgHandler) DetectReorg(
	ctx context.Context,
	currentBlock uint64,
	newBlockHash common.Hash,
) (bool, uint64, error) {
	rh.mu.Lock()
	storedHash, exists := rh.lastKnownBlocks[currentBlock]
	if !exists && rh.checkpointStore != nil {
		dbHash, err := rh.checkpointStore.GetBlockHash(ctx, rh.chainID, currentBlock)
		if err == nil && dbHash != "" {
			storedHash = common.HexToHash(dbHash)
			exists = true
			rh.lastKnownBlocks[currentBlock] = storedHash
		}
	}
	if !exists {
		rh.lastKnownBlocks[currentBlock] = newBlockHash
		rh.mu.Unlock()
		return false, 0, nil
	}

	if storedHash == newBlockHash {
		rh.mu.Unlock()
		return false, 0, nil
	}

	knownBlocks := make(map[uint64]common.Hash, len(rh.lastKnownBlocks))
	for k, v := range rh.lastKnownBlocks {
		knownBlocks[k] = v
	}
	provider := rh.blockHashProvider
	maxRollback := rh.maxRollback
	rh.mu.Unlock()

	reorgBlock, err := rh.findReorgBlock(ctx, currentBlock, knownBlocks, provider, maxRollback)
	if err != nil {
		rh.logger.Warn("reorg scan failed", core.LogKeyError, err)
		return true, currentBlock, nil
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
	if reorgBlock == 0 {
		return fmt.Errorf("invalid reorg block: 0")
	}

	currentBlock, err := rh.database.GetLatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	blocksToRollback := currentBlock - reorgBlock + 1

	rh.mu.RLock()
	maxR := rh.maxRollback
	db := rh.database
	invalidator := rh.idempotencyInvalidator
	eventBus := rh.eventBus
	chainID := rh.chainID
	rh.mu.RUnlock()

	if blocksToRollback > maxR {
		return fmt.Errorf("reorg too large: %d blocks (max: %d)", blocksToRollback, maxR)
	}

	eventsRolledBack, err := rh.rollbackWithSnapshots(ctx, reorgBlock, currentBlock, db, invalidator)
	if err != nil {
		return fmt.Errorf("failed to rollback events: %w", err)
	}

	rh.mu.Lock()
	rh.cleanupBlockCache(reorgBlock)
	rh.mu.Unlock()

	if eventBus != nil {
		reorgEvt := &ReorgEvent{
			DetectedAt:       time.Now(),
			ReorgBlock:       reorgBlock,
			BlocksAffected:   blocksToRollback,
			EventsRolledBack: eventsRolledBack,
		}
		if err := eventBus.Publish(ctx, core.TopicReorgDetected, reorgEvt); err != nil {
			rh.logger.Error("Failed to publish reorg event", core.LogKeyError, err)
		}

		reindexEvt := &core.ReorgRollbackEvent{
			ChainID:    chainID,
			FromBlock:  reorgBlock,
			ToBlock:    currentBlock,
			DetectedAt: time.Now(),
		}
		if err := eventBus.Publish(ctx, core.TopicReorgRollback, reindexEvt); err != nil {
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
func (rh *ReorgHandler) RollbackEvents(ctx context.Context, fromBlock, currentBlock uint64) (int64, error) {
	rh.mu.RLock()
	db := rh.database
	invalidator := rh.idempotencyInvalidator
	rh.mu.RUnlock()
	return rh.rollbackWithSnapshots(ctx, fromBlock, currentBlock, db, invalidator)
}

func (rh *ReorgHandler) rollbackWithSnapshots(ctx context.Context, fromBlock, currentBlock uint64, db core.DatabasePlugin, invalidator core.IdempotencyInvalidator) (int64, error) {
	count, err := db.MarkEventsAsReorged(ctx, fromBlock, currentBlock)
	if err != nil {
		return 0, fmt.Errorf("failed to mark events as reorged: %w", err)
	}

	if invalidator != nil {
		invalidated := invalidator.InvalidateRange(fromBlock, currentBlock)
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
func (rh *ReorgHandler) findReorgBlock(ctx context.Context, currentBlock uint64, knownBlocks map[uint64]common.Hash, provider core.BlockHashProvider, maxRollback uint64) (uint64, error) {
	maxScanDepth := uint64(256)
	if maxRollback > 0 && maxRollback < maxScanDepth {
		maxScanDepth = maxRollback
	}

	lowerBound := uint64(1)
	if currentBlock > maxScanDepth {
		lowerBound = currentBlock - maxScanDepth
	}

	if reorgBlock, err := rh.binarySearchReorg(ctx, lowerBound, currentBlock, knownBlocks, provider); err == nil {
		return reorgBlock, nil
	}

	rh.logger.Warn("binary search reorg detection failed, falling back to linear scan", "error", "binary_search_unavailable")
	return rh.linearScanReorg(ctx, currentBlock, maxScanDepth, knownBlocks, provider)
}

// binarySearchReorg performs a binary search between lowerBound and currentBlock
// to find the exact reorg divergence point. For each candidate block, it compares
// the locally-indexed hash against the canonical chain hash (via blockHashProvider).
// Time complexity: O(log n).
func (rh *ReorgHandler) binarySearchReorg(ctx context.Context, lowerBound, currentBlock uint64, knownBlocks map[uint64]common.Hash, provider core.BlockHashProvider) (uint64, error) {
	lo, hi := lowerBound, currentBlock
	result := currentBlock

	for lo <= hi {
		select {
		case <-ctx.Done():
			return currentBlock, ctx.Err()
		default:
		}

		mid := lo + (hi-lo)/2

		storedHash, exists := knownBlocks[mid]
		if !exists {
			lo = mid + 1
			continue
		}

		canonicalHash, err := provider.GetBlockHash(ctx, mid)
		if err != nil {
			return 0, fmt.Errorf("block hash lookup failed at block %d: %w", mid, err)
		}

		if canonicalHash == (common.Hash{}) {
			lo = mid + 1
			continue
		}

		if storedHash != canonicalHash {
			result = mid
			hi = mid - 1
		} else {
			lo = mid + 1
		}
	}

	return result, nil
}

func (rh *ReorgHandler) linearScanReorg(ctx context.Context, currentBlock, maxScanDepth uint64, knownBlocks map[uint64]common.Hash, provider core.BlockHashProvider) (uint64, error) {
	lowerBound := uint64(1)
	if currentBlock > maxScanDepth {
		lowerBound = currentBlock - maxScanDepth
	}

	for block := currentBlock; block >= lowerBound; block-- {
		select {
		case <-ctx.Done():
			return currentBlock, ctx.Err()
		default:
		}

		storedHash, exists := knownBlocks[block]
		if !exists {
			continue
		}

		canonicalHash, err := provider.GetBlockHash(ctx, block)
		if err != nil {
			rh.logger.Warn("block hash lookup failed during reorg scan", core.LogKeyBlockNumber, block, core.LogKeyError, err)
			continue
		}

		if canonicalHash == (common.Hash{}) {
			continue
		}

		if storedHash == canonicalHash {
			return block, nil
		}
	}

	return currentBlock, fmt.Errorf("no matching block found in range %d-%d", lowerBound, currentBlock)
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

// UpdateBlockHash updates the known hash for a block and periodically persists
// the latest block hash to checkpointStore for restart survival. The in-memory
// map holds the full reorg window; only the tip is persisted.
func (rh *ReorgHandler) UpdateBlockHash(blockNumber uint64, blockHash common.Hash) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	rh.lastKnownBlocks[blockNumber] = blockHash

	// Persist latest hash every checkpointInterval blocks (single write, not batch loop)
	if rh.checkpointStore != nil && blockNumber > rh.lastPersistedBlock+rh.checkpointInterval {
		_ = rh.checkpointStore.SaveLastIndexedBlock(context.Background(), rh.chainID, blockNumber, blockHash.Hex())
		rh.lastPersistedBlock = blockNumber
	}

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
	db := rh.database
	rh.mu.RUnlock()

	for block := fromBlock; block < toBlock; block++ {
		currentBlock, err := db.GetBlock(ctx, block)
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", block, err)
		}

		if currentBlock == nil {
			return fmt.Errorf("block %d not found", block)
		}

		nextBlock, err := db.GetBlock(ctx, block+1)
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
