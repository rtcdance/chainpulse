package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultValidationManager implements ValidationManager
type DefaultValidationManager struct {
	mu sync.RWMutex
}

// NewValidationManager creates a new validation manager
func NewValidationManager() ValidationManager {
	return &DefaultValidationManager{}
}

// ValidateEventCollection validates that all events were collected
func (vm *DefaultValidationManager) ValidateEventCollection(ctx context.Context, emitted []*EventEmission, indexed []*IndexedEvent) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if len(emitted) != len(indexed) {
		return fmt.Errorf("event count mismatch: emitted %d, indexed %d", len(emitted), len(indexed))
	}

	// Create a map of indexed events by TxHash and LogIndex for quick lookup
	indexedMap := make(map[string]*IndexedEvent)
	for _, event := range indexed {
		key := fmt.Sprintf("%s_%d", event.TxHash, event.LogIndex)
		indexedMap[key] = event
	}

	// Verify all emitted events are indexed
	for _, emission := range emitted {
		key := fmt.Sprintf("%s_%d", emission.TxHash, emission.LogIndex)
		if _, exists := indexedMap[key]; !exists {
			return fmt.Errorf("emitted event not found in indexed events: %s", key)
		}
	}

	return nil
}

// ValidateEventDecoding validates event decoding accuracy
func (vm *DefaultValidationManager) ValidateEventDecoding(ctx context.Context, event *IndexedEvent) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if event == nil {
		return fmt.Errorf("event is nil")
	}

	if event.ID == "" {
		return fmt.Errorf("event ID is empty")
	}

	if event.ContractAddress == "" {
		return fmt.Errorf("contract address is empty")
	}

	if event.EventName == "" {
		return fmt.Errorf("event name is empty")
	}

	if event.TxHash == "" {
		return fmt.Errorf("transaction hash is empty")
	}

	if event.DecodedData == nil {
		return fmt.Errorf("decoded data is nil")
	}

	return nil
}

// ValidateEventOrdering validates event ordering
func (vm *DefaultValidationManager) ValidateEventOrdering(ctx context.Context, events []*IndexedEvent) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if len(events) == 0 {
		return nil
	}

	// Verify events are ordered by block number and log index
	for i := 1; i < len(events); i++ {
		prev := events[i-1]
		curr := events[i]

		if curr.BlockNumber < prev.BlockNumber {
			return fmt.Errorf("block number ordering violation: event %d has block %d, event %d has block %d",
				i-1, prev.BlockNumber, i, curr.BlockNumber)
		}

		if curr.BlockNumber == prev.BlockNumber && curr.LogIndex < prev.LogIndex {
			return fmt.Errorf("log index ordering violation in block %d: event %d has index %d, event %d has index %d",
				curr.BlockNumber, i-1, prev.LogIndex, i, curr.LogIndex)
		}
	}

	return nil
}

// ValidateAPIResponse validates API query response
func (vm *DefaultValidationManager) ValidateAPIResponse(ctx context.Context, response *APIResponse, expectedEvents []*IndexedEvent) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if response == nil {
		return fmt.Errorf("API response is nil")
	}

	if len(response.Events) != len(expectedEvents) {
		return fmt.Errorf("event count mismatch: expected %d, got %d", len(expectedEvents), len(response.Events))
	}

	// Verify response contains expected events
	expectedMap := make(map[string]*IndexedEvent)
	for _, event := range expectedEvents {
		key := fmt.Sprintf("%s_%d", event.TxHash, event.LogIndex)
		expectedMap[key] = event
	}

	for _, event := range response.Events {
		key := fmt.Sprintf("%s_%d", event.TxHash, event.LogIndex)
		if _, exists := expectedMap[key]; !exists {
			return fmt.Errorf("unexpected event in response: %s", key)
		}
	}

	// Verify pagination metadata
	if response.Total < len(response.Events) {
		return fmt.Errorf("total count %d is less than returned events %d", response.Total, len(response.Events))
	}

	return nil
}

// ValidatePerformance validates performance metrics
func (vm *DefaultValidationManager) ValidatePerformance(ctx context.Context, metrics PerformanceMetrics) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	// Validate average duration (should be < 2 seconds)
	if metrics.AverageDuration > 2*time.Second {
		return fmt.Errorf("average duration %v exceeds 2 second limit", metrics.AverageDuration)
	}

	// Validate throughput (should be > 1000 events/second)
	if metrics.ThroughputOpsPerSec < 1000.0 {
		return fmt.Errorf("throughput %.2f ops/sec is below 1000 ops/sec minimum", metrics.ThroughputOpsPerSec)
	}

	// Validate success rate (should be > 95%)
	if metrics.TotalOperations > 0 {
		successRate := float64(metrics.SuccessfulOps) / float64(metrics.TotalOperations) * 100
		if successRate < 95.0 {
			return fmt.Errorf("success rate %.2f%% is below 95%% minimum", successRate)
		}
	}

	return nil
}

// ValidateEventDeduplication validates that duplicate events are handled correctly
func (vm *DefaultValidationManager) ValidateEventDeduplication(ctx context.Context, events []*IndexedEvent) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	// Create a map to track unique events
	seen := make(map[string]int)

	for _, event := range events {
		key := fmt.Sprintf("%s_%d", event.TxHash, event.LogIndex)
		seen[key]++
	}

	// Check for duplicates
	for key, count := range seen {
		if count > 1 {
			return fmt.Errorf("duplicate event found: %s (count: %d)", key, count)
		}
	}

	return nil
}

// ValidateEventDataIntegrity validates that event data is intact
func (vm *DefaultValidationManager) ValidateEventDataIntegrity(ctx context.Context, emitted *EventEmission, indexed *IndexedEvent) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if emitted == nil || indexed == nil {
		return fmt.Errorf("emitted or indexed event is nil")
	}

	if emitted.ContractAddress != indexed.ContractAddress {
		return fmt.Errorf("contract address mismatch: emitted %s, indexed %s", emitted.ContractAddress, indexed.ContractAddress)
	}

	if emitted.EventName != indexed.EventName {
		return fmt.Errorf("event name mismatch: emitted %s, indexed %s", emitted.EventName, indexed.EventName)
	}

	if emitted.TxHash != indexed.TxHash {
		return fmt.Errorf("transaction hash mismatch: emitted %s, indexed %s", emitted.TxHash, indexed.TxHash)
	}

	if emitted.BlockNumber != indexed.BlockNumber {
		return fmt.Errorf("block number mismatch: emitted %d, indexed %d", emitted.BlockNumber, indexed.BlockNumber)
	}

	if emitted.LogIndex != indexed.LogIndex {
		return fmt.Errorf("log index mismatch: emitted %d, indexed %d", emitted.LogIndex, indexed.LogIndex)
	}

	return nil
}
