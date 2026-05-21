// Package processing contains infrastructure-level event processing types.
//
// Deprecated: The canonical implementations have moved to pkg/services/processor/
// (event_processor.go) and pkg/services/query/ (event_store.go). This package is
// retained for backward compatibility and is currently consumed by
// pkg/infrastructure/blockchain/blockchain_cluster.go. New code should use the
// service-layer equivalents.
package processing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"golang.org/x/sync/errgroup"
)

// Event represents a blockchain event to be processed
type Event struct {
	ID              string         `json:"id"`
	EventHash       string         `json:"event_hash"`
	BlockNumber     uint64         `json:"block_number"`
	TransactionHash string         `json:"transaction_hash"`
	LogIndex        uint64         `json:"log_index"`
	ContractAddress string         `json:"contract_address"`
	EventName       string         `json:"event_name"`
	EventData       map[string]any `json:"event_data"`
	ChainID         string         `json:"chain_id"`
	Timestamp       time.Time      `json:"timestamp"`
	ProcessedAt     time.Time      `json:"processed_at"`
	Status          string         `json:"status"` // pending, processed, failed
}

// EventValidationError represents validation errors
type EventValidationError struct {
	EventID string
	Reason  string
	Err     error // underlying error, if any
}

// Error implements the error interface
func (e *EventValidationError) Error() string {
	return fmt.Sprintf("validation failed for event %s: %s", e.EventID, e.Reason)
}

// Unwrap returns the underlying error, enabling errors.Is() and errors.As()
func (e *EventValidationError) Unwrap() error { return e.Err }

// EventProcessor processes blockchain events
type EventProcessor struct {
	mu                sync.RWMutex
	id                string
	chainID           string
	processedCount    int64
	failedCount       int64
	validationErrors  int64
	batchSize         int
	processingTimeout time.Duration
	metrics           *ProcessorMetrics
}

// ProcessorMetrics tracks processor metrics
type ProcessorMetrics struct {
	mu                  sync.RWMutex
	EventsProcessed     int64
	EventsFailed        int64
	ValidationErrors    int64
	BatchesProcessed    int64
	AverageLatency      time.Duration
	LastProcessedTime   time.Time
	TotalProcessingTime time.Duration
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(id, chainID string, batchSize int) *EventProcessor {
	return &EventProcessor{
		id:                id,
		chainID:           chainID,
		batchSize:         batchSize,
		processingTimeout: 30 * time.Second,
		metrics: &ProcessorMetrics{
			LastProcessedTime: time.Now(),
		},
	}
}

// ProcessEvent processes a single event
func (ep *EventProcessor) ProcessEvent(ctx context.Context, event *Event) error {
	start := time.Now()
	defer func() {
		ep.recordLatency(time.Since(start))
	}()

	// Validate event
	if err := ep.validateEvent(event); err != nil {
		ep.mu.Lock()
		ep.validationErrors++
		ep.mu.Unlock()
		return fmt.Errorf("validation error: %w", err)
	}

	// Normalize event
	ep.normalizeEvent(event)

	// Generate event hash for idempotency
	event.EventHash = ep.generateEventHash(event)

	// Mark as processed
	event.Status = "processed"
	event.ProcessedAt = time.Now()

	ep.mu.Lock()
	ep.processedCount++
	ep.mu.Unlock()

	return nil
}

// ProcessBatch processes a batch of events with bounded concurrency
func (ep *EventProcessor) ProcessBatch(ctx context.Context, events []*Event) error {
	start := time.Now()
	defer func() {
		ep.recordBatchLatency(time.Since(start))
	}()

	if len(events) == 0 {
		return nil
	}

	// Sort events by (block_number, log_index) to guarantee strict ordering
	// even though processing is concurrent. This ensures events from earlier
	// blocks enter the processing pipeline before later blocks.
	sort.Slice(events, func(i, j int) bool {
		if events[i].BlockNumber != events[j].BlockNumber {
			return events[i].BlockNumber < events[j].BlockNumber
		}
		return events[i].LogIndex < events[j].LogIndex
	})

	// Use errgroup with bounded concurrency to avoid spawning unlimited goroutines
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	var mu sync.Mutex // protects failedCount updates
	for i := range events {
		event := events[i]
		g.Go(func() error {
			if err := ep.ProcessEvent(gCtx, event); err != nil {
				mu.Lock()
				ep.failedCount++
				mu.Unlock()
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("batch processing failed: %w", err)
	}

	ep.mu.Lock()
	ep.metrics.BatchesProcessed++
	ep.mu.Unlock()

	return nil
}

// validateEvent validates event structure and content
func (ep *EventProcessor) validateEvent(event *Event) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	if event.ID == "" {
		return fmt.Errorf("event ID is empty")
	}

	if event.ChainID == "" {
		return fmt.Errorf("chain ID is empty")
	}

	if event.ContractAddress == "" {
		return fmt.Errorf("contract address is empty")
	}

	if event.EventName == "" {
		return fmt.Errorf("event name is empty")
	}

	if event.TransactionHash == "" {
		return fmt.Errorf("transaction hash is empty")
	}

	if event.EventData == nil {
		event.EventData = make(map[string]any)
	}

	return nil
}

// normalizeEvent normalizes event data
func (ep *EventProcessor) normalizeEvent(event *Event) {
	// Normalize addresses to lowercase
	event.ContractAddress = normalizeAddress(event.ContractAddress)
	event.TransactionHash = normalizeHash(event.TransactionHash)

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	}

// generateEventHash generates a deterministic hash for the event using the
// canonical natural key (chain_id, block_number, transaction_hash, log_index).
// This produces the same result as core.ComputeEventHash for equivalent fields.
func (ep *EventProcessor) generateEventHash(event *Event) string {
	// Same formula as core.ComputeEventHash: chain_id:block_number:tx_hash:log_index
	hashInput := fmt.Sprintf("%s:%d:%s:%d",
		event.ChainID,
		event.BlockNumber,
		event.TransactionHash,
		event.LogIndex,
	)

	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// GetMetrics returns processor metrics
func (ep *EventProcessor) GetMetrics() ProcessorMetrics {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	ep.metrics.mu.RLock()
	defer ep.metrics.mu.RUnlock()

	return ProcessorMetrics{
		EventsProcessed:     ep.metrics.EventsProcessed,
		EventsFailed:        ep.metrics.EventsFailed,
		ValidationErrors:    ep.metrics.ValidationErrors,
		BatchesProcessed:    ep.metrics.BatchesProcessed,
		AverageLatency:      ep.metrics.AverageLatency,
		LastProcessedTime:   ep.metrics.LastProcessedTime,
		TotalProcessingTime: ep.metrics.TotalProcessingTime,
	}
}

// recordLatency records event processing latency
func (ep *EventProcessor) recordLatency(latency time.Duration) {
	ep.metrics.mu.Lock()
	defer ep.metrics.mu.Unlock()

	ep.metrics.EventsProcessed++
	ep.metrics.TotalProcessingTime += latency

	if ep.metrics.EventsProcessed > 0 {
		ep.metrics.AverageLatency = ep.metrics.TotalProcessingTime / time.Duration(ep.metrics.EventsProcessed)
	}

	ep.metrics.LastProcessedTime = time.Now()
}

// recordBatchLatency records batch processing latency
func (ep *EventProcessor) recordBatchLatency(latency time.Duration) {
	ep.metrics.mu.Lock()
	defer ep.metrics.mu.Unlock()

	ep.metrics.TotalProcessingTime += latency
}

// GetID returns processor ID
func (ep *EventProcessor) GetID() string {
	return ep.id
}

// GetChainID returns chain ID
func (ep *EventProcessor) GetChainID() string {
	return ep.chainID
}

// GetProcessedCount returns number of processed events
func (ep *EventProcessor) GetProcessedCount() int64 {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.processedCount
}

// GetFailedCount returns number of failed events
func (ep *EventProcessor) GetFailedCount() int64 {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return ep.failedCount
}

// Health returns processor health status
func (ep *EventProcessor) Health() core.HealthStatus {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	status := core.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]any{
			"processor_id":      ep.id,
			"chain_id":          ep.chainID,
			"processed_count":   ep.processedCount,
			"failed_count":      ep.failedCount,
			"validation_errors": ep.validationErrors,
		},
	}

	// Mark as degraded if failure rate is high
	if ep.processedCount > 0 {
		failureRate := float64(ep.failedCount) / float64(ep.processedCount+ep.failedCount)
		if failureRate > 0.1 { // 10% failure rate
			status.Status = "degraded"
		}
	}

	return status
}

// normalizeAddress normalizes blockchain addresses to lowercase with 0x prefix
func normalizeAddress(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	if len(addr) > 2 && addr[:2] == "0x" {
		return addr
	}
	return "0x" + addr
}

// normalizeHash normalizes transaction hashes to lowercase with 0x prefix
func normalizeHash(hash string) string {
	hash = strings.ToLower(strings.TrimSpace(hash))
	if len(hash) > 2 && hash[:2] == "0x" {
		return hash
	}
	return "0x" + hash
}
