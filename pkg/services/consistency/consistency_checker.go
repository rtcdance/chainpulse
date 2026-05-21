package consistency

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// ConsistencyChecker verifies and repairs data consistency.
//
// Renaming would break many external uses.
type ConsistencyChecker struct {
	database core.DatabasePlugin
	logger   core.Logger
	mu       sync.RWMutex
}

// ConsistencyReport contains consistency check results.
//
// Renaming would break many external uses.
type ConsistencyReport struct {
	CheckedAt          time.Time
	TotalEvents        int64
	DuplicateEvents    int64
	MissingEvents      int64
	InvalidSequences   int64
	InconsistentBlocks int64
	Status             string
	Issues             []string
	RepairAttempts     int64
	SuccessfulRepairs  int64
	FailedRepairs      int64
}

// EventConsistency represents consistency info for an event
type EventConsistency struct {
	EventID         string
	BlockNumber     uint64
	TransactionHash string
	IsDuplicate     bool
	IsOrphaned      bool
	IsValid         bool
	Issues          []string
}

// BlockConsistency represents consistency info for a block
type BlockConsistency struct {
	BlockNumber    uint64
	IsValid        bool
	HasValidParent bool
	EventCount     int64
	Issues         []string
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

// CheckConsistency performs a comprehensive consistency check.
// Optimized to load data once and reuse across all checks.
func (cc *ConsistencyChecker) CheckConsistency(ctx context.Context) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		CheckedAt: time.Now(),
		Issues:    []string{},
	}

	// Load all events once
	events, err := cc.database.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}
	report.TotalEvents = int64(len(events))

	// Check for duplicates (single pass over events)
	duplicates := findDuplicates(events)
	report.DuplicateEvents = int64(len(duplicates))
	if len(duplicates) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("found %d duplicate events", len(duplicates)))
	}

	// Build block map from events for sequence checks
	eventBlockMap := buildBlockSetFromEvents(events)
	sequenceIssues := findEventSequenceGaps(eventBlockMap)
	report.InvalidSequences = int64(len(sequenceIssues))
	if len(sequenceIssues) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("found %d sequence issues", len(sequenceIssues)))
	}

	// Load all blocks once for block checks
	blocks, err := cc.database.GetAllBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all blocks: %w", err)
	}

	blockIssues := findBlockSequenceGaps(blocks)
	report.InconsistentBlocks = int64(len(blockIssues))
	if len(blockIssues) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("found %d block issues", len(blockIssues)))
	}

	// Determine status
	report.Status = computeStatus(report)

	cc.logger.Info(
		"Consistency check completed",
		map[string]any{
			"total_events":    report.TotalEvents,
			"duplicates":      report.DuplicateEvents,
			"sequence_issues": report.InvalidSequences,
			"block_issues":    report.InconsistentBlocks,
			"status":          report.Status,
		},
	)

	return report, nil
}

// findDuplicates finds duplicate events in O(n) using a map.
func findDuplicates(events []*core.BlockchainEvent) []*core.BlockchainEvent {
	seen := make(map[string]bool, len(events))
	var duplicates []*core.BlockchainEvent

	for _, event := range events {
		key := fmt.Sprintf("%s-%d-%s", event.TransactionHash.Hex(), event.LogIndex, event.EventSignature.Hex())
		if seen[key] {
			duplicates = append(duplicates, event)
		}
		seen[key] = true
	}

	return duplicates
}

// buildBlockSetFromEvents creates a set of block numbers from events.
func buildBlockSetFromEvents(events []*core.BlockchainEvent) map[uint64]bool {
	blockSet := make(map[uint64]bool, len(events))
	for _, event := range events {
		blockSet[event.BlockNumber] = true
	}
	return blockSet
}

// findEventSequenceGaps finds gaps in event block coverage using set operations
// instead of iterating from minBlock to maxBlock (which can be O(billions)).
func findEventSequenceGaps(blockSet map[uint64]bool) []string {
	if len(blockSet) == 0 {
		return nil
	}

	blocks := make([]uint64, 0, len(blockSet))
	for block := range blockSet {
		blocks = append(blocks, block)
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i] < blocks[j] })

	var issues []string
	for i := 1; i < len(blocks); i++ {
		if blocks[i] != blocks[i-1]+1 {
			gapStart := blocks[i-1] + 1
			gapEnd := blocks[i] - 1
			if gapStart == gapEnd {
				issues = append(issues, fmt.Sprintf("missing events for block %d", gapStart))
			} else {
				issues = append(issues, fmt.Sprintf("missing events for blocks %d-%d (%d blocks)", gapStart, gapEnd, gapEnd-gapStart+1))
			}
		}
	}

	return issues
}

// findBlockSequenceGaps finds gaps and parent hash mismatches in blocks
// using sorted block list instead of iterating the entire range.
func findBlockSequenceGaps(blocks []*core.Block) []string {
	if len(blocks) == 0 {
		return nil
	}

	// Sort blocks by number
	sorted := make([]*core.Block, len(blocks))
	copy(sorted, blocks)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Number < sorted[j].Number })

	blockMap := make(map[uint64]*core.Block, len(sorted))
	for _, block := range sorted {
		blockMap[block.Number] = block
	}

	var issues []string

	// Find gaps between consecutive stored blocks
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Number != sorted[i-1].Number+1 {
			gapStart := sorted[i-1].Number + 1
			gapEnd := sorted[i].Number - 1
			if gapStart == gapEnd {
				issues = append(issues, fmt.Sprintf("missing block %d", gapStart))
			} else {
				issues = append(issues, fmt.Sprintf("missing blocks %d-%d (%d blocks)", gapStart, gapEnd, gapEnd-gapStart+1))
			}
		}
	}

	// Check parent-child relationships using sorted order (O(n))
	for i := 1; i < len(sorted); i++ {
		currentBlock := sorted[i]
		parentBlock := sorted[i-1]

		if currentBlock.Number == parentBlock.Number+1 {
			if currentBlock.ParentHash != parentBlock.Hash {
				issues = append(issues, fmt.Sprintf(
					"block %d parent hash mismatch: expected %s, got %s",
					currentBlock.Number,
					parentBlock.Hash.Hex(),
					currentBlock.ParentHash.Hex(),
				))
			}
		}
	}

	return issues
}

// RepairInconsistencies attempts to repair found inconsistencies
// including duplicate deletion, sequence gap detection, orphaned event removal,
// and block hash mismatches from reorgs.
func (cc *ConsistencyChecker) RepairInconsistencies(ctx context.Context) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		CheckedAt: time.Now(),
		Issues:    []string{},
	}

	// Get all events once
	events, err := cc.database.GetAllEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}
	report.TotalEvents = int64(len(events))

	// Phase 1: Repair duplicate events
	duplicates := findDuplicates(events)
	report.DuplicateEvents = int64(len(duplicates))

	for _, duplicate := range duplicates {
		report.RepairAttempts++
		err := cc.database.DeleteEvent(ctx, duplicate.ID)
		if err != nil {
			report.FailedRepairs++
			cc.logger.Error(
				"Failed to delete duplicate event",
				map[string]any{
					"event_id": duplicate.ID,
					"error":    err.Error(),
				},
			)
		} else {
			report.SuccessfulRepairs++
			cc.logger.Info("Deleted duplicate event", "event_id", duplicate.ID)
		}
	}

	// Phase 2: Remove orphaned events (events whose blocks don't exist)
	orphaned, err := cc.findOrphanedEvents(ctx, events)
	if err != nil {
		return nil, fmt.Errorf("failed to find orphaned events: %w", err)
	}

	for _, orphan := range orphaned {
		report.RepairAttempts++
		err := cc.database.DeleteEvent(ctx, orphan.ID)
		if err != nil {
			report.FailedRepairs++
			cc.logger.Error("Failed to delete orphaned event", "event_id", orphan.ID, "error", err.Error())
		} else {
			report.SuccessfulRepairs++
			cc.logger.Info("Deleted orphaned event", "event_id", orphan.ID, "block", orphan.BlockNumber)
		}
	}

	// Phase 3: Verify and report sequence gaps (cannot auto-fill without re-indexing)
	eventBlockMap := buildBlockSetFromEvents(events)
	sequenceIssues := findEventSequenceGaps(eventBlockMap)
	report.InvalidSequences = int64(len(sequenceIssues))
	report.MissingEvents = int64(len(sequenceIssues))

	// Check block sequence
	blocks, err := cc.database.GetAllBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all blocks: %w", err)
	}
	blockIssues := findBlockSequenceGaps(blocks)
	report.InconsistentBlocks = int64(len(blockIssues))

	// Determine status based on remaining issues after repair
	report.Status = computeStatus(report)

	if len(sequenceIssues) > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("sequence gaps detected (%d issues) — requires re-indexing", len(sequenceIssues)))
	}
	for _, issue := range blockIssues {
		report.Issues = append(report.Issues, "block issue: "+issue)
	}

	cc.logger.Info(
		"Consistency repair completed",
		map[string]any{
			"total_events":       report.TotalEvents,
			"duplicates_removed": report.DuplicateEvents,
			"orphaned_removed":   len(orphaned),
			"repair_attempts":    report.RepairAttempts,
			"successful_repairs": report.SuccessfulRepairs,
			"failed_repairs":     report.FailedRepairs,
			"status":             report.Status,
		},
	)

	return report, nil
}

// findOrphanedEvents returns events whose associated blocks don't exist in the database.
func (cc *ConsistencyChecker) findOrphanedEvents(ctx context.Context, events []*core.BlockchainEvent) ([]*core.BlockchainEvent, error) {
	seenBlocks := make(map[uint64]bool)
	var orphaned []*core.BlockchainEvent

	for _, event := range events {
		if _, checked := seenBlocks[event.BlockNumber]; !checked {
			block, err := cc.database.GetBlock(ctx, event.BlockNumber)
			if err != nil {
				cc.logger.Warn("Failed to check block for orphan detection", "block", event.BlockNumber, "error", err.Error())
				continue
			}
			seenBlocks[event.BlockNumber] = block != nil
		}

		if !seenBlocks[event.BlockNumber] {
			orphaned = append(orphaned, event)
		}
	}

	return orphaned, nil
}

// computeStatus determines the health status from a report.
// After repairs, the status should reflect remaining issues.
func computeStatus(report *ConsistencyReport) string {
	hasRemainingIssues := report.InvalidSequences > 0 || report.InconsistentBlocks > 0
	if hasRemainingIssues {
		if report.InvalidSequences > 0 {
			return "degraded"
		}
		return "unhealthy"
	}
	return "healthy"
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
