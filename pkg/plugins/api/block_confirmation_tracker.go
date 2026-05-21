package api

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
	"github.com/rtcdance/chainpulse/pkg/services/reorg"
)

// BlockConfirmationTracker monitors chain heads and transitions pending events
// to confirmed after N additional blocks, or to reorged if the block hash changes.
// When a FinalityChecker is available, it uses PoS finalized blocks instead of
// depth-based confirmation counting for superior accuracy.
type BlockConfirmationTracker struct {
	db              *sql.DB
	logger          core.Logger
	metrics         core.MetricsCollector
	subscriptionHub *SubscriptionHub               // for buffering replay events
	reorgHandlers   map[string]*reorg.ReorgHandler // chainID -> reorg handler
	finalityChecker finality.FinalityChecker       // optional: PoS finality support
	confirmationMap map[uint64]int                 // chainID (numeric) -> confirmation blocks
	mu              sync.RWMutex
	running         bool
	cancel          context.CancelFunc
	checkInterval   time.Duration
}

// NewBlockConfirmationTracker creates a new confirmation tracker
func NewBlockConfirmationTracker(
	db *sql.DB,
	logger core.Logger,
	metrics core.MetricsCollector,
	hub *SubscriptionHub,
	confirmationMap map[uint64]int,
) *BlockConfirmationTracker {
	if confirmationMap == nil {
		confirmationMap = map[uint64]int{
			1:        12,  // Ethereum
			137:      128, // Polygon
			56:       15,  // BSC
			97:       15,  // BSC Testnet
			42161:    12,  // Arbitrum One
			421614:   12,  // Arbitrum Sepolia
			10:       12,  // Optimism
			11155420: 12,  // Optimism Sepolia
			8453:     12,  // Base
			84532:    12,  // Base Sepolia
			43114:    12,  // Avalanche C-Chain
			43113:    12,  // Avalanche Fuji
		}
	}
	return &BlockConfirmationTracker{
		db:              db,
		logger:          logger,
		metrics:         metrics,
		subscriptionHub: hub,
		reorgHandlers:   make(map[string]*reorg.ReorgHandler),
		confirmationMap: confirmationMap,
		checkInterval:   30 * time.Second,
	}
}

// RegisterReorgHandler registers a reorg handler for a specific chain
func (t *BlockConfirmationTracker) RegisterReorgHandler(chainID string, handler *reorg.ReorgHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reorgHandlers[chainID] = handler
}

// SetCheckInterval configures how often the tracker polls for confirmations
func (t *BlockConfirmationTracker) SetCheckInterval(d time.Duration) {
	t.checkInterval = d
}

// SetFinalityChecker sets the finality checker for PoS chain support.
// When set, the tracker will use finalized block numbers instead of
// depth-based confirmation counting for chains that support it.
func (t *BlockConfirmationTracker) SetFinalityChecker(fc finality.FinalityChecker) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.finalityChecker = fc
}

// Start begins the periodic confirmation check loop
func (t *BlockConfirmationTracker) Start(ctx context.Context) error {
	if t.db == nil {
		return fmt.Errorf("database not configured")
	}
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("tracker already running")
	}
	ctx, t.cancel = context.WithCancel(ctx)
	t.running = true
	t.mu.Unlock()

	go t.runLoop(ctx)
	t.logger.Info("BlockConfirmationTracker started")
	return nil
}

// Stop halts the confirmation tracker
func (t *BlockConfirmationTracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
	}
	t.running = false
}

func (t *BlockConfirmationTracker) runLoop(ctx context.Context) {
	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	// Run once immediately
	t.checkConfirmations(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.checkConfirmations(ctx)
		}
	}
}

// checkConfirmations queries pending events and transitions them based on chain depth
func (t *BlockConfirmationTracker) checkConfirmations(ctx context.Context) {
	// Get current chain heads from indexing_state
	rows, err := t.db.QueryContext(ctx,
		`SELECT chain_id, last_indexed_block FROM indexing_state`,
	)
	if err != nil {
		t.logger.Error("failed to query indexing state", "error", err)
		return
	}

	type chainHead struct {
		chainID string
		block   int64
	}
	var heads []chainHead
	for rows.Next() {
		var h chainHead
		if err := rows.Scan(&h.chainID, &h.block); err != nil {
			continue
		}
		heads = append(heads, h)
	}
	if err := rows.Err(); err != nil {
		t.logger.Error("error iterating indexing state rows", "error", err)
		return
	}
	rows.Close() //nolint:errcheck

	for _, head := range heads {
		confirmations := t.getConfirmationBlocks(head.chainID)
		t.transitionPendingEvents(ctx, head.chainID, head.block, int64(confirmations))
		t.detectReorgs(ctx, head.chainID)
	}
}

// getConfirmationBlocks returns the number of confirmations required for a chain
func (t *BlockConfirmationTracker) getConfirmationBlocks(chainID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try parsing as numeric chain ID first
	if id, err := strconv.ParseUint(chainID, 10, 64); err == nil {
		if n, ok := t.confirmationMap[id]; ok {
			return n
		}
	}

	// Fallback for named chains
	namedChains := map[string]uint64{
		"ethereum": 1, "polygon": 137, "bsc": 56, "arbitrum": 42161,
		"optimism": 10, "base": 8453, "avalanche": 43114,
	}
	if id, ok := namedChains[chainID]; ok {
		if n, ok2 := t.confirmationMap[id]; ok2 {
			return n
		}
	}

	t.logger.Warn("unknown chain ID, using default confirmation count", "chainId", chainID, "default", 12)
	return 12 // default
}

// transitionPendingEvents moves events from pending to confirmed when their block
// is deep enough relative to the current chain head. When a FinalityChecker is
// available and returns a finalized block number, events at or below that block
// are confirmed immediately (true PoS finality), bypassing depth-based counting.
func (t *BlockConfirmationTracker) transitionPendingEvents(ctx context.Context, chainID string, headBlock, confirmations int64) {
	cutoffBlock := headBlock - confirmations

	// Try PoS finality first — events below the finalized block are irrevocably confirmed
	t.mu.RLock()
	fc := t.finalityChecker
	t.mu.RUnlock()

	if fc != nil {
		finalizedBlock, err := fc.GetFinalizedBlockNumber(ctx, chainID)
		if err == nil && int64(finalizedBlock) > cutoffBlock {
			// Finalized block is further ahead than depth-based cutoff; use it
			cutoffBlock = int64(finalizedBlock)
		} else if err != nil {
			// Log but don't block — fall back to depth-based confirmation
			t.logger.Warn("finality check failed, using depth-based confirmation",
				"chainId", chainID, "error", err.Error())
		}
	}

	if cutoffBlock <= 0 {
		return
	}

	result, err := t.db.ExecContext(ctx,
		`UPDATE events SET status = 'confirmed' 
		 WHERE chain_id = $1 AND status = 'pending' AND block_number <= $2`,
		chainID, cutoffBlock,
	)
	if err != nil {
		t.logger.Error("failed to transition events", "chainId", chainID, "error", err)
		t.metrics.RecordCounter("confirmation.transition_failed", 1, nil)
		return
	}

	n, _ := result.RowsAffected()
	if n > 0 {
		t.logger.Info("transitioned events to confirmed",
			"chainId", chainID, "count", n, "cutoffBlock", cutoffBlock)
		t.metrics.RecordCounter("confirmation.events_confirmed", n, nil)

		// Notify subscribers via the subscription hub
		if t.subscriptionHub != nil {
			t.notifyConfirmedEvents(ctx, chainID, cutoffBlock)
		}
	}
}

// notifyConfirmedEvents fetches newly confirmed events and buffers them for replay
func (t *BlockConfirmationTracker) notifyConfirmedEvents(ctx context.Context, chainID string, cutoffBlock int64) {
	rows, err := t.db.QueryContext(ctx,
		`SELECT id, block_number, contract_address, event_name, event_data 
		 FROM events WHERE chain_id = $1 AND status = 'confirmed' AND block_number <= $2
		 ORDER BY block_number DESC LIMIT 100`,
		chainID, cutoffBlock,
	)
	if err != nil {
		return
	}
	defer rows.Close() //nolint:errcheck // defer close

	for rows.Next() {
		var id string
		var blockNumber int64
		var contractAddress, eventName string
		var eventData []byte

		if err := rows.Scan(&id, &blockNumber, &contractAddress, &eventName, &eventData); err != nil {
			continue
		}

		payload := map[string]any{
			"type":            "confirmed",
			"eventId":         id,
			"chainId":         chainID,
			"contractAddress": contractAddress,
			"eventName":       eventName,
			"blockNumber":     blockNumber,
			"timestamp":       time.Now().Unix(),
		}
		t.subscriptionHub.BufferEvent("event:confirmed", payload)
	}
	if err := rows.Err(); err != nil {
		t.logger.Error("error iterating confirmed events rows", "error", err)
	}
}

// detectReorgs checks if any confirmed blocks have different hashes than what
// we indexed. Uses the registered ReorgHandler to detect and handle chain reorganizations.
func (t *BlockConfirmationTracker) detectReorgs(ctx context.Context, chainID string) {
	t.mu.RLock()
	handler, exists := t.reorgHandlers[chainID]
	t.mu.RUnlock()
	if !exists || handler == nil {
		return
	}

	// Get the latest block number and hash from the blocks table
	var headBlock uint64
	var headHash string
	err := t.db.QueryRowContext(ctx,
		`SELECT number, hash FROM blocks WHERE chain_id = $1 ORDER BY number DESC LIMIT 1`,
		chainID,
	).Scan(&headBlock, &headHash)
	if err != nil {
		if err != sql.ErrNoRows {
			t.logger.Error("failed to query latest block for reorg check", "chainId", chainID, "error", err)
		}
		return
	}

	reorgDetected, reorgBlock, err := handler.DetectReorg(ctx, headBlock, common.HexToHash(headHash))
	if err != nil {
		t.logger.Warn("reorg detection failed", "chainId", chainID, "block", headBlock, "error", err)
		return
	}

	if reorgDetected {
		t.logger.Info("reorg detected, initiating rollback",
			"chainId", chainID, "reorgBlock", reorgBlock, "headBlock", headBlock)
		t.metrics.RecordCounter("confirmation.reorg_detected", 1, map[string]string{"chain_id": chainID})

		if err := handler.HandleReorg(ctx, reorgBlock); err != nil {
			t.logger.Error("reorg rollback failed", "chainId", chainID, "reorgBlock", reorgBlock, "error", err)
			t.metrics.RecordCounter("confirmation.reorg_rollback_failed", 1, map[string]string{"chain_id": chainID})
			return
		}

		t.logger.Info("reorg rollback completed", "chainId", chainID, "reorgBlock", reorgBlock)
		t.metrics.RecordCounter("confirmation.reorg_rollback_completed", 1, map[string]string{"chain_id": chainID})

		// Notify subscribers
		if t.subscriptionHub != nil {
			t.subscriptionHub.BufferEvent("event:reorged", map[string]any{
				"type":       "reorged",
				"chainId":    chainID,
				"reorgBlock": reorgBlock,
				"headBlock":  headBlock,
				"timestamp":  time.Now().Unix(),
			})
		}
	}
}

// GetConfirmationStats returns current confirmation statistics
func (t *BlockConfirmationTracker) GetConfirmationStats(ctx context.Context) (map[string]any, error) {
	if t.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	stats := make(map[string]any)

	// Count pending events
	var pendingCount int64
	if err := t.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM events WHERE status = 'pending'`,
	).Scan(&pendingCount); err != nil {
		return nil, fmt.Errorf("count pending events: %w", err)
	}
	stats["pendingEvents"] = pendingCount

	// Count confirmed events
	var confirmedCount int64
	if err := t.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM events WHERE status = 'confirmed'`,
	).Scan(&confirmedCount); err != nil {
		return nil, fmt.Errorf("count confirmed events: %w", err)
	}
	stats["confirmedEvents"] = confirmedCount

	// Count reorged events
	var reorgedCount int64
	if err := t.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM events WHERE status = 'reorged'`,
	).Scan(&reorgedCount); err != nil {
		return nil, fmt.Errorf("count reorged events: %w", err)
	}
	stats["reorgedEvents"] = reorgedCount

	t.mu.RLock()
	stats["confirmationMap"] = t.confirmationMap
	t.mu.RUnlock()

	return stats, nil
}
