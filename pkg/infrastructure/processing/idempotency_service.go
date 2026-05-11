package processing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// IdempotencyService manages event idempotency.
// Blockchain events have a permanent natural key (chain_id, block_number, tx_hash, log_index)
// that will never repeat, so processed records are never evicted.
// Database-level unique constraints provide the ultimate dedup guarantee.
type IdempotencyService struct {
	mu              sync.RWMutex
	processedEvents map[string]*ProcessedEventRecord
	duplicateCount  int64
	checkCount      int64
	counterMu       sync.Mutex // Separate mutex for counter updates
}

// ProcessedEventRecord tracks a processed event
type ProcessedEventRecord struct {
	EventHash     string
	ProcessedAt   time.Time
	ChainID       string
	TransactionID string
	BlockNumber   uint64 // stored for range-based invalidation on reorg
	Status        string // "success", "failed", "pending"
}

// NewIdempotencyService creates a new idempotency service
func NewIdempotencyService() *IdempotencyService {
	return &IdempotencyService{
		processedEvents: make(map[string]*ProcessedEventRecord),
	}
}

// IsDuplicate checks if an event has already been processed
func (is *IdempotencyService) IsDuplicate(ctx context.Context, eventHash string) (bool, error) {
	if eventHash == "" {
		return false, fmt.Errorf("event hash is empty")
	}

	is.counterMu.Lock()
	is.checkCount++
	is.counterMu.Unlock()

	is.mu.RLock()
	defer is.mu.RUnlock()

	_, exists := is.processedEvents[eventHash]
	if !exists {
		return false, nil
	}

	is.counterMu.Lock()
	is.duplicateCount++
	is.counterMu.Unlock()

	return true, nil
}

// MarkProcessed marks an event as processed
func (is *IdempotencyService) MarkProcessed(ctx context.Context, eventHash, chainID, txID string, blockNumber uint64, status string) error {
	if eventHash == "" {
		return fmt.Errorf("event hash is empty")
	}

	if status == "" {
		status = "success"
	}

	is.mu.Lock()
	defer is.mu.Unlock()

	record := &ProcessedEventRecord{
		EventHash:     eventHash,
		ProcessedAt:   time.Now(),
		ChainID:       chainID,
		TransactionID: txID,
		BlockNumber:   blockNumber,
		Status:        status,
	}

	is.processedEvents[eventHash] = record

	return nil
}

// GetProcessedRecord retrieves a processed event record
func (is *IdempotencyService) GetProcessedRecord(ctx context.Context, eventHash string) (*ProcessedEventRecord, error) {
	if eventHash == "" {
		return nil, fmt.Errorf("event hash is empty")
	}

	is.mu.RLock()
	defer is.mu.RUnlock()

	record, exists := is.processedEvents[eventHash]
	if !exists {
		return nil, fmt.Errorf("record not found")
	}

	return record, nil
}

// GetDuplicateCount returns the number of duplicates detected
func (is *IdempotencyService) GetDuplicateCount() int64 {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return is.duplicateCount
}

// GetCheckCount returns the number of duplicate checks performed
func (is *IdempotencyService) GetCheckCount() int64 {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return is.checkCount
}

// GetProcessedCount returns the number of processed events
func (is *IdempotencyService) GetProcessedCount() int64 {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return int64(len(is.processedEvents))
}

// GetDuplicateRate returns the duplicate detection rate
func (is *IdempotencyService) GetDuplicateRate() float64 {
	is.mu.RLock()
	defer is.mu.RUnlock()

	if is.checkCount == 0 {
		return 0
	}

	return float64(is.duplicateCount) / float64(is.checkCount)
}

// Reset clears all processed records
func (is *IdempotencyService) Reset() {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.counterMu.Lock()
	defer is.counterMu.Unlock()

	is.processedEvents = make(map[string]*ProcessedEventRecord)
	is.duplicateCount = 0
	is.checkCount = 0
}

// InvalidateRange removes idempotency entries for events in the given block range.
// This is called after a reorg so that re-indexed events are not rejected as duplicates.
func (is *IdempotencyService) InvalidateRange(fromBlock, toBlock uint64) int {
	is.mu.Lock()
	defer is.mu.Unlock()

	removed := 0
	for hash, record := range is.processedEvents {
		if record.BlockNumber >= fromBlock && record.BlockNumber <= toBlock {
			delete(is.processedEvents, hash)
			removed++
		}
	}
	return removed
}

// GetMetrics returns idempotency metrics
func (is *IdempotencyService) GetMetrics() map[string]interface{} {
	is.mu.RLock()
	defer is.mu.RUnlock()

	return map[string]interface{}{
		"processed_count": int64(len(is.processedEvents)),
		"duplicate_count": is.duplicateCount,
		"check_count":     is.checkCount,
		"duplicate_rate":  is.GetDuplicateRate(),
	}
}

// ValidateIdempotency validates that an event can be safely retried
func (is *IdempotencyService) ValidateIdempotency(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	if event.EventHash == "" {
		return fmt.Errorf("event hash is empty")
	}

	// Check if duplicate
	isDuplicate, err := is.IsDuplicate(ctx, event.EventHash)
	if err != nil {
		return fmt.Errorf("duplicate check failed: %w", err)
	}

	if isDuplicate {
		return fmt.Errorf("event is duplicate")
	}

	return nil
}

// BatchValidateIdempotency validates a batch of events for idempotency
func (is *IdempotencyService) BatchValidateIdempotency(ctx context.Context, events []*Event) ([]bool, error) {
	if len(events) == 0 {
		return []bool{}, nil
	}

	results := make([]bool, len(events))

	is.mu.RLock()
	defer is.mu.RUnlock()

	for i, event := range events {
		if event == nil || event.EventHash == "" {
			results[i] = false
			continue
		}

		_, exists := is.processedEvents[event.EventHash]
		if !exists {
			results[i] = false
			continue
		}

		results[i] = true
	}

	return results, nil
}
