package pullers

import (
	"context"
	"fmt"
	"sync"
	"time"
	"chainpulse/pkg/core"
)

// ReorgHandler handles blockchain reorganizations
type ReorgHandler struct {
	mu                    sync.RWMutex
	logger                core.Logger
	metricsCollector      core.MetricsCollector
	eventBus              core.EventBus
	dataPuller            core.DataPullerPlugin
	lastConfirmedBlock    uint64
	confirmationThreshold uint64
	reorgDetectionWindow  uint64
	reorgHistory          map[uint64]bool // block number -> is reorg
	affectedBlocks        map[uint64][]core.BlockchainEvent
	reorgCount            int64
	recoveryCount         int64
	lastReorgTime         time.Time
	lastReorgBlock        uint64
	isProcessing          bool
}

// ReorgDetectionResult represents the result of reorg detection
type ReorgDetectionResult struct {
	IsReorg        bool
	ReorgBlockNum  uint64
	AffectedBlocks []uint64
	NewChainHead   uint64
	Timestamp      time.Time
}

// NewReorgHandler creates a new reorg handler
func NewReorgHandler(
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
	dataPuller core.DataPullerPlugin,
) *ReorgHandler {
	return &ReorgHandler{
		logger:                logger,
		metricsCollector:      metricsCollector,
		eventBus:              eventBus,
		dataPuller:            dataPuller,
		lastConfirmedBlock:    0,
		confirmationThreshold: 12, // Ethereum standard: 12 blocks
		reorgDetectionWindow:  256, // Look back up to 256 blocks
		reorgHistory:          make(map[uint64]bool),
		affectedBlocks:        make(map[uint64][]core.BlockchainEvent),
		reorgCount:            0,
		recoveryCount:         0,
		isProcessing:          false,
	}
}

// DetectReorg detects blockchain reorganizations
func (h *ReorgHandler) DetectReorg(ctx context.Context, currentBlock, previousBlock uint64, currentHash, previousHash string) (*ReorgDetectionResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isProcessing {
		return nil, fmt.Errorf("reorg detection already in progress")
	}

	h.isProcessing = true
	defer func() { h.isProcessing = false }()

	result := &ReorgDetectionResult{
		IsReorg:        false,
		ReorgBlockNum:  0,
		AffectedBlocks: make([]uint64, 0),
		NewChainHead:   currentBlock,
		Timestamp:      time.Now().UTC(),
	}

	// Check if current block is less than previous block (reorg detected)
	if currentBlock < previousBlock {
		h.logger.Info("reorg detected: block number decreased", "current", currentBlock, "previous", previousBlock)
		result.IsReorg = true
		result.ReorgBlockNum = currentBlock
		result.AffectedBlocks = h.findAffectedBlocks(currentBlock, previousBlock)
		h.reorgCount++
		h.lastReorgTime = time.Now().UTC()
		h.lastReorgBlock = currentBlock

		h.metricsCollector.RecordCounter("reorg_detected", 1, map[string]string{})
		h.metricsCollector.RecordGauge("reorg_block_number", float64(currentBlock), map[string]string{})
		h.metricsCollector.RecordGauge("reorg_affected_blocks", float64(len(result.AffectedBlocks)), map[string]string{})

		h.logger.Warn("blockchain reorganization detected", "reorg_block", currentBlock, "affected_blocks", len(result.AffectedBlocks))

		return result, nil
	}

	// Check if hash changed for the same block number (reorg detected)
	if currentBlock == previousBlock && currentHash != previousHash {
		h.logger.Info("reorg detected: hash changed for same block", "block", currentBlock)
		result.IsReorg = true
		result.ReorgBlockNum = currentBlock
		result.AffectedBlocks = []uint64{currentBlock}
		h.reorgCount++
		h.lastReorgTime = time.Now().UTC()
		h.lastReorgBlock = currentBlock

		h.metricsCollector.RecordCounter("reorg_detected", 1, map[string]string{})
		h.metricsCollector.RecordGauge("reorg_block_number", float64(currentBlock), map[string]string{})

		h.logger.Warn("blockchain reorganization detected", "reorg_block", currentBlock, "affected_blocks", 1)

		return result, nil
	}

	// No reorg detected
	h.metricsCollector.RecordCounter("reorg_check_passed", 1, map[string]string{})

	return result, nil
}

// HandleReorg handles a detected reorganization
func (h *ReorgHandler) HandleReorg(ctx context.Context, result *ReorgDetectionResult) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !result.IsReorg {
		return nil
	}

	h.logger.Info("handling blockchain reorganization", "reorg_block", result.ReorgBlockNum, "affected_blocks", len(result.AffectedBlocks))

	// Store reorg information
	h.reorgHistory[result.ReorgBlockNum] = true

	// Store affected blocks for reprocessing
	for _, blockNum := range result.AffectedBlocks {
		if _, exists := h.affectedBlocks[blockNum]; !exists {
			h.affectedBlocks[blockNum] = make([]core.BlockchainEvent, 0)
		}
	}

	// Publish reorg event
	reorgEvent := map[string]interface{}{
		"type":             "reorg",
		"reorg_block":      result.ReorgBlockNum,
		"affected_blocks":  result.AffectedBlocks,
		"new_chain_head":   result.NewChainHead,
		"timestamp":        result.Timestamp,
	}

	if err := h.eventBus.Publish(ctx, "blockchain_reorg", reorgEvent); err != nil {
		h.logger.Error("failed to publish reorg event", "error", err.Error())
	}

	h.logger.Info("reorg event published", "reorg_block", result.ReorgBlockNum)

	return nil
}

// ReprocessAffectedBlocks reprocesses blocks affected by a reorganization
func (h *ReorgHandler) ReprocessAffectedBlocks(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("reprocessing affected blocks", "from_block", fromBlock, "to_block", toBlock)

	events := make([]core.BlockchainEvent, 0)

	// Pull events from the affected block range
	pulledEvents, err := h.dataPuller.PullEvents(ctx, fromBlock, toBlock)
	if err != nil {
		h.logger.Error("failed to pull events for reprocessing", "error", err.Error(), "from_block", fromBlock, "to_block", toBlock)
		h.metricsCollector.RecordCounter("reorg_reprocess_errors", 1, map[string]string{})
		return nil, err
	}

	// Validate and store events
	for _, event := range pulledEvents {
		// Validate event structure
		if event.BlockNumber == 0 || event.EventName == "" {
			h.logger.Warn("invalid event during reprocessing", "error", "missing required fields", "event_id", event.ID)
			continue
		}

		events = append(events, event)

		// Store in affected blocks map
		if _, exists := h.affectedBlocks[event.BlockNumber]; exists {
			h.affectedBlocks[event.BlockNumber] = append(h.affectedBlocks[event.BlockNumber], event)
		}
	}

	h.recoveryCount++
	h.metricsCollector.RecordGauge("reorg_reprocess_blocks", float64(toBlock-fromBlock+1), map[string]string{})
	h.metricsCollector.RecordGauge("reorg_reprocess_events", float64(len(events)), map[string]string{})
	h.metricsCollector.RecordGauge("reorg_recovery_count", float64(h.recoveryCount), map[string]string{})

	h.logger.Info("affected blocks reprocessed", "count", len(events), "from_block", fromBlock, "to_block", toBlock)

	return events, nil
}

// GetAffectedBlocks returns the blocks affected by a reorganization
func (h *ReorgHandler) GetAffectedBlocks(reorgBlock uint64) []uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	affectedBlocks := make([]uint64, 0)

	for blockNum := range h.affectedBlocks {
		if blockNum >= reorgBlock {
			affectedBlocks = append(affectedBlocks, blockNum)
		}
	}

	return affectedBlocks
}

// GetReorgHistory returns the reorg history
func (h *ReorgHandler) GetReorgHistory() map[uint64]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	history := make(map[uint64]bool)
	for k, v := range h.reorgHistory {
		history[k] = v
	}

	return history
}

// ClearReorgHistory clears the reorg history
func (h *ReorgHandler) ClearReorgHistory() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.reorgHistory = make(map[uint64]bool)
	h.affectedBlocks = make(map[uint64][]core.BlockchainEvent)

	h.logger.Info("reorg history cleared")
}

// SetConfirmationThreshold sets the confirmation threshold
func (h *ReorgHandler) SetConfirmationThreshold(threshold uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.confirmationThreshold = threshold
	h.logger.Info("confirmation threshold set", "threshold", threshold)
}

// SetReorgDetectionWindow sets the reorg detection window
func (h *ReorgHandler) SetReorgDetectionWindow(window uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.reorgDetectionWindow = window
	h.logger.Info("reorg detection window set", "window", window)
}

// GetStats returns statistics about the reorg handler
func (h *ReorgHandler) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"last_confirmed_block":      h.lastConfirmedBlock,
		"confirmation_threshold":    h.confirmationThreshold,
		"reorg_detection_window":    h.reorgDetectionWindow,
		"reorg_count":               h.reorgCount,
		"recovery_count":            h.recoveryCount,
		"last_reorg_time":           h.lastReorgTime,
		"last_reorg_block":          h.lastReorgBlock,
		"reorg_history_size":        len(h.reorgHistory),
		"affected_blocks_size":      len(h.affectedBlocks),
		"is_processing":             h.isProcessing,
	}
}

// IsReorgConfirmed checks if a block is confirmed (not subject to reorg)
func (h *ReorgHandler) IsReorgConfirmed(blockNumber, currentBlock uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// A block is confirmed if it's at least confirmationThreshold blocks behind the current block
	return (currentBlock - blockNumber) >= h.confirmationThreshold
}

// findAffectedBlocks finds the blocks affected by a reorganization
func (h *ReorgHandler) findAffectedBlocks(reorgBlock, previousBlock uint64) []uint64 {
	affectedBlocks := make([]uint64, 0)

	// All blocks from reorgBlock to previousBlock are affected
	for i := reorgBlock; i <= previousBlock; i++ {
		affectedBlocks = append(affectedBlocks, i)
	}

	return affectedBlocks
}

// Monitor monitors for reorganizations
func (h *ReorgHandler) Monitor(ctx context.Context, checkInterval time.Duration) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	var lastBlock uint64
	var lastHash string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Get current block
			currentBlock, err := h.dataPuller.GetLatestBlock(ctx)
			if err != nil {
				h.logger.Error("failed to get latest block for reorg monitoring", "error", err.Error())
				continue
			}

			// Detect reorg
			result, err := h.DetectReorg(ctx, currentBlock, lastBlock, "", lastHash)
			if err != nil {
				h.logger.Error("failed to detect reorg", "error", err.Error())
				continue
			}

			// Handle reorg if detected
			if result.IsReorg {
				if err := h.HandleReorg(ctx, result); err != nil {
					h.logger.Error("failed to handle reorg", "error", err.Error())
					continue
				}

				// Reprocess affected blocks
				if len(result.AffectedBlocks) > 0 {
					minBlock := result.AffectedBlocks[0]
					maxBlock := result.AffectedBlocks[len(result.AffectedBlocks)-1]

					_, err := h.ReprocessAffectedBlocks(ctx, minBlock, maxBlock)
					if err != nil {
						h.logger.Error("failed to reprocess affected blocks", "error", err.Error())
						continue
					}
				}
			}

			lastBlock = currentBlock
		}
	}
}

// UpdateLastConfirmedBlock updates the last confirmed block
func (h *ReorgHandler) UpdateLastConfirmedBlock(blockNumber uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if blockNumber > h.lastConfirmedBlock {
		h.lastConfirmedBlock = blockNumber
		h.logger.Info("last confirmed block updated", "block_number", blockNumber)
	}
}

// GetLastConfirmedBlock returns the last confirmed block
func (h *ReorgHandler) GetLastConfirmedBlock() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.lastConfirmedBlock
}
