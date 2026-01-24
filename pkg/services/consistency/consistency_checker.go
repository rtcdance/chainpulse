package consistency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// ConsistencyChecker verifies and repairs data consistency
type ConsistencyChecker struct {
	database core.DatabasePlugin
	logger   core.Logger
	mu       sync.RWMutex
}

// ConsistencyReport contains consistency check results
type ConsistencyReport struct {
	CheckedAt           time.Time
	TotalEvents         int64
	DuplicateEvents     int64
	MissingEvents       int64
	InvalidSequences    int64
	InconsistentBlocks  int64
	Status              string
	Issues              []string
	RepairAttempts      int64
	SuccessfulRepairs   int64
	FailedRepairs       int64
}

// EventConsistency represents consistency info for an event
type EventConsistency struct {
	EventID       string
	BlockNumber   uint64
	TransactionHash string
	IsDuplicate   bool
	IsOrphaned    bool
	IsValid       bool
	Issues        []string
}

// BlockConsistency represents consistency info for a block
type BlockConsistency struct {
	BlockNumber   uint64
	IsValid       bool
	HasValidParent bool
	EventCount    int64
	Issues        []string
}

// NewConsistencyChecker creates a new consistency checker
func NewConsistencyChecker(
	database core.DatabasePlugin,
	logger core.Logger,
) *ConsistencyChecker {
	return &ConsistencyChecker{
		database: database,
		logger:   logger,
	}
}

// CheckConsistency performs a comprehensive consistency check
func (cc *ConsistencyChecker) CheckConsistency(ctx context.Context) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		CheckedAt: time.Now(),
		Issues:    []string{},
	}

	// Get all events
	events, err := cc.database.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	report.TotalEvents = int64(len(events))

	// Check for duplicates
	duplicates, err := cc.checkDuplicates(ctx, events)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicates: %w", err)
	}
	report.DuplicateEvents = int64(len(duplicates))
	if len(duplicates) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("found %d duplicate events", len(duplicates)))
	}

	// Check event sequence
	sequenceIssues, err := cc.VerifyEventSequence(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify event sequence: %w", err)
	}
	report.InvalidSequences = int64(len(sequenceIssues))
	if len(sequenceIssues) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("found %d sequence issues", len(sequenceIssues)))
	}

	// Check block sequence
	blockIssues, err := cc.VerifyBlockSequence(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify block sequence: %w", err)
	}
	report.InconsistentBlocks = int64(len(blockIssues))
	if len(blockIssues) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("found %d block issues", len(blockIssues)))
	}

	// Determine status
	if len(report.Issues) == 0 {
		report.Status = "healthy"
	} else if report.DuplicateEvents > 0 || report.InvalidSequences > 0 {
		report.Status = "degraded"
	} else {
		report.Status = "unhealthy"
	}

	cc.logger.Info(
		"Consistency check completed",
		map[string]interface{}{
			"total_events": report.TotalEvents,
			"duplicates": report.DuplicateEvents,
			"sequence_issues": report.InvalidSequences,
			"block_issues": report.InconsistentBlocks,
			"status": report.Status,
		},
	)

	return report, nil
}

// checkDuplicates finds duplicate events
func (cc *ConsistencyChecker) checkDuplicates(ctx context.Context, events []*core.BlockchainEvent) ([]*core.BlockchainEvent, error) {
	seen := make(map[string]bool)
	var duplicates []*core.BlockchainEvent

	for _, event := range events {
		key := fmt.Sprintf("%s-%d-%s", event.TransactionHash.Hex(), event.LogIndex, event.EventSignature.Hex())
		if seen[key] {
			duplicates = append(duplicates, event)
		}
		seen[key] = true
	}

	return duplicates, nil
}

// VerifyEventSequence verifies that events form a valid sequence
func (cc *ConsistencyChecker) VerifyEventSequence(ctx context.Context) ([]string, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	var issues []string

	// Get all events ordered by block and log index
	events, err := cc.database.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	// Check for gaps in block numbers
	blockMap := make(map[uint64]int)
	for _, event := range events {
		blockMap[event.BlockNumber]++
	}

	if len(blockMap) > 0 {
		minBlock := uint64(^uint64(0))
		maxBlock := uint64(0)

		for block := range blockMap {
			if block < minBlock {
				minBlock = block
			}
			if block > maxBlock {
				maxBlock = block
			}
		}

		// Check for gaps
		for block := minBlock; block <= maxBlock; block++ {
			if _, exists := blockMap[block]; !exists {
				issues = append(issues, fmt.Sprintf("missing events for block %d", block))
			}
		}
	}

	// Check for duplicate events
	seen := make(map[string]bool)
	for _, event := range events {
		key := fmt.Sprintf("%s-%d-%s", event.TransactionHash.Hex(), event.LogIndex, event.EventSignature.Hex())
		if seen[key] {
			issues = append(issues, fmt.Sprintf("duplicate event: %s", key))
		}
		seen[key] = true
	}

	return issues, nil
}

// VerifyBlockSequence verifies that blocks form a valid sequence
func (cc *ConsistencyChecker) VerifyBlockSequence(ctx context.Context) ([]string, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	var issues []string

	// Get all blocks
	blocks, err := cc.database.GetAllBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all blocks: %w", err)
	}

	if len(blocks) == 0 {
		return issues, nil
	}

	// Sort blocks by number
	blockMap := make(map[uint64]*core.Block)
	minBlock := uint64(^uint64(0))
	maxBlock := uint64(0)

	for _, block := range blocks {
		blockMap[block.Number] = block
		if block.Number < minBlock {
			minBlock = block.Number
		}
		if block.Number > maxBlock {
			maxBlock = block.Number
		}
	}

	// Check for gaps
	for block := minBlock; block < maxBlock; block++ {
		if _, exists := blockMap[block]; !exists {
			issues = append(issues, fmt.Sprintf("missing block %d", block))
		}
	}

	// Check parent-child relationships
	for block := minBlock + 1; block <= maxBlock; block++ {
		currentBlock := blockMap[block]
		parentBlock := blockMap[block-1]

		if currentBlock == nil || parentBlock == nil {
			continue
		}

		if currentBlock.ParentHash != parentBlock.Hash {
			issues = append(issues, fmt.Sprintf(
				"block %d parent hash mismatch: expected %s, got %s",
				block,
				parentBlock.Hash.Hex(),
				currentBlock.ParentHash.Hex(),
			))
		}
	}

	return issues, nil
}

// RepairInconsistencies attempts to repair found inconsistencies
func (cc *ConsistencyChecker) RepairInconsistencies(ctx context.Context) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		CheckedAt: time.Now(),
		Issues:    []string{},
	}

	// Get all events
	events, err := cc.database.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	report.TotalEvents = int64(len(events))

	// Repair duplicates
	duplicates, err := cc.checkDuplicates(ctx, events)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicates: %w", err)
	}

	for _, duplicate := range duplicates {
		err := cc.database.DeleteEvent(ctx, duplicate.ID)
		if err != nil {
			report.FailedRepairs++
			cc.logger.Error(
				"Failed to delete duplicate event",
				map[string]interface{}{
					"event_id": duplicate.ID,
					"error": err.Error(),
				},
			)
		} else {
			report.SuccessfulRepairs++
		}
		report.RepairAttempts++
	}

	report.DuplicateEvents = int64(len(duplicates))

	// Verify sequences after repair
	sequenceIssues, err := cc.VerifyEventSequence(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify event sequence: %w", err)
	}
	report.InvalidSequences = int64(len(sequenceIssues))

	blockIssues, err := cc.VerifyBlockSequence(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify block sequence: %w", err)
	}
	report.InconsistentBlocks = int64(len(blockIssues))

	// Determine status
	if report.DuplicateEvents == 0 && report.InvalidSequences == 0 && report.InconsistentBlocks == 0 {
		report.Status = "healthy"
	} else if report.DuplicateEvents > 0 || report.InvalidSequences > 0 {
		report.Status = "degraded"
	} else {
		report.Status = "unhealthy"
	}

	cc.logger.Info(
		"Consistency repair completed",
		map[string]interface{}{
			"repair_attempts": report.RepairAttempts,
			"successful_repairs": report.SuccessfulRepairs,
			"failed_repairs": report.FailedRepairs,
			"status": report.Status,
		},
	)

	return report, nil
}

// GetEventConsistency checks consistency of a specific event
func (cc *ConsistencyChecker) GetEventConsistency(ctx context.Context, eventID string) (*EventConsistency, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	consistency := &EventConsistency{
		EventID: eventID,
		Issues:  []string{},
		IsValid: true,
	}

	// Get event
	event, err := cc.database.GetEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if event == nil {
		consistency.IsValid = false
		consistency.Issues = append(consistency.Issues, "event not found")
		return consistency, nil
	}

	consistency.BlockNumber = event.BlockNumber
	consistency.TransactionHash = event.TransactionHash.Hex()

	// Check for duplicates
	events, err := cc.database.GetEventsByBlockRange(ctx, event.BlockNumber, event.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	duplicateCount := 0
	for _, e := range events {
		if e.TransactionHash == event.TransactionHash && e.LogIndex == event.LogIndex {
			duplicateCount++
		}
	}

	if duplicateCount > 1 {
		consistency.IsDuplicate = true
		consistency.IsValid = false
		consistency.Issues = append(consistency.Issues, fmt.Sprintf("duplicate event found (%d total)", duplicateCount))
	}

	// Check if block exists
	block, err := cc.database.GetBlock(ctx, event.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	if block == nil {
		consistency.IsOrphaned = true
		consistency.IsValid = false
		consistency.Issues = append(consistency.Issues, "block not found")
	}

	return consistency, nil
}

// GetBlockConsistency checks consistency of a specific block
func (cc *ConsistencyChecker) GetBlockConsistency(ctx context.Context, blockNumber uint64) (*BlockConsistency, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	consistency := &BlockConsistency{
		BlockNumber: blockNumber,
		Issues:      []string{},
		IsValid:     true,
	}

	// Get block
	block, err := cc.database.GetBlock(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	if block == nil {
		consistency.IsValid = false
		consistency.Issues = append(consistency.Issues, "block not found")
		return consistency, nil
	}

	// Check parent block
	if blockNumber > 0 {
		parentBlock, err := cc.database.GetBlock(ctx, blockNumber-1)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent block: %w", err)
		}

		if parentBlock == nil {
			consistency.HasValidParent = false
			consistency.Issues = append(consistency.Issues, "parent block not found")
		} else if block.ParentHash != parentBlock.Hash {
			consistency.HasValidParent = false
			consistency.Issues = append(consistency.Issues, "parent hash mismatch")
		} else {
			consistency.HasValidParent = true
		}
	} else {
		consistency.HasValidParent = true
	}

	// Count events
	events, err := cc.database.GetEventsByBlockRange(ctx, blockNumber, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	consistency.EventCount = int64(len(events))

	if len(consistency.Issues) > 0 {
		consistency.IsValid = false
	}

	return consistency, nil
}
