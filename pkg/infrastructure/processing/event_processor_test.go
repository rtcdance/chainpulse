package processing

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewEventProcessor tests EventProcessor creation
func TestNewEventProcessor(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	assert.NotNil(t, ep)
	assert.Equal(t, "processor-1", ep.GetID())
	assert.Equal(t, "ethereum", ep.GetChainID())
	assert.Equal(t, 100, ep.batchSize)
	assert.Equal(t, 30*time.Second, ep.processingTimeout)
	assert.NotNil(t, ep.metrics)
}

// TestProcessEventSuccess tests successful event processing
func TestProcessEventSuccess(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{"from": "0x123", "to": "0x456"},
		ChainID:         "ethereum",
		Timestamp:       time.Now(),
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.NoError(t, err)
	assert.Equal(t, "processed", event.Status)
	assert.NotEmpty(t, event.EventHash)
	assert.False(t, event.ProcessedAt.IsZero())
	assert.Equal(t, int64(1), ep.GetProcessedCount())
}

// TestProcessEventNilEvent tests processing nil event
func TestProcessEventNilEvent(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event is nil")
}

// TestProcessEventMissingID tests processing event with missing ID
func TestProcessEventMissingID(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event ID is empty")
}

// TestProcessEventMissingChainID tests processing event with missing chain ID
func TestProcessEventMissingChainID(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chain ID is empty")
}

// TestProcessEventMissingContractAddress tests processing event with missing contract address
func TestProcessEventMissingContractAddress(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contract address is empty")
}

// TestProcessEventMissingEventName tests processing event with missing event name
func TestProcessEventMissingEventName(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event name is empty")
}

// TestProcessEventMissingTransactionHash tests processing event with missing transaction hash
func TestProcessEventMissingTransactionHash(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction hash is empty")
}

// TestProcessEventNilEventData tests processing event with nil event data
func TestProcessEventNilEventData(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       nil,
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.NoError(t, err)
	assert.NotNil(t, event.EventData)
}

// TestProcessEventNormalization tests event normalization
func TestProcessEventNormalization(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
		Timestamp:       time.Time{}, // Zero timestamp
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)

	assert.NoError(t, err)
	assert.False(t, event.Timestamp.IsZero())
}

// TestProcessBatchSuccess tests successful batch processing
func TestProcessBatchSuccess(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	events := make([]*Event, 5)
	for i := 0; i < 5; i++ {
		events[i] = &Event{
			ID:              fmt.Sprintf("event-%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: fmt.Sprintf("0xabc%d", i),
			LogIndex:        uint64(i),
			ContractAddress: "0xdef456",
			EventName:       "Transfer",
			EventData:       map[string]interface{}{},
			ChainID:         "ethereum",
		}
	}

	ctx := context.Background()
	err := ep.ProcessBatch(ctx, events)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), ep.GetProcessedCount())

	for _, event := range events {
		assert.Equal(t, "processed", event.Status)
		assert.NotEmpty(t, event.EventHash)
	}
}

// TestProcessBatchEmpty tests processing empty batch
func TestProcessBatchEmpty(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	ctx := context.Background()
	err := ep.ProcessBatch(ctx, []*Event{})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), ep.GetProcessedCount())
}

// TestProcessBatchWithErrors tests batch processing with errors
func TestProcessBatchWithErrors(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	events := make([]*Event, 3)
	events[0] = &Event{
		ID:              "event-0",
		BlockNumber:     12345,
		TransactionHash: "0xabc0",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}
	events[1] = &Event{
		ID:              "", // Invalid - missing ID
		BlockNumber:     12346,
		TransactionHash: "0xabc1",
		LogIndex:        1,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}
	events[2] = &Event{
		ID:              "event-2",
		BlockNumber:     12347,
		TransactionHash: "0xabc2",
		LogIndex:        2,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessBatch(ctx, events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch processing failed")
	assert.Equal(t, int64(2), ep.GetProcessedCount())
	assert.Equal(t, int64(1), ep.GetFailedCount())
}

// TestGenerateEventHash tests event hash generation
func TestGenerateEventHash(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	hash1 := ep.generateEventHash(event)
	hash2 := ep.generateEventHash(event)

	// Same event should generate same hash (deterministic)
	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA256 hex is 64 characters
}

// TestGenerateEventHashDifferent tests that events with different natural keys
// generate different hashes. Events that differ only in non-key fields (ID,
// EventName, ContractAddress) should produce the SAME hash since they share
// the same on-chain identity.
func TestGenerateEventHashDifferent(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	// Same natural key, different ID → same hash (same on-chain event)
	event1 := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	event2 := &Event{
		ID:              "event-2",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	hash1 := ep.generateEventHash(event1)
	hash2 := ep.generateEventHash(event2)
	assert.Equal(t, hash1, hash2, "events with same natural key should have same hash")

	// Different block number → different hash
	event3 := &Event{
		ID:              "event-3",
		BlockNumber:     99999,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}
	hash3 := ep.generateEventHash(event3)
	assert.NotEqual(t, hash1, hash3, "events with different block numbers should have different hashes")
}

// TestNormalizeAddress tests address normalization
func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0xabc123", "0xabc123"},
		{"abc123", "abc123"},
		{"0x", "0x"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeAddress(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeHash tests hash normalization
func TestNormalizeHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0xabc123", "0xabc123"},
		{"abc123", "abc123"},
		{"0x", "0x"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeHash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetMetrics tests metrics retrieval
func TestGetMetrics(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)
	assert.NoError(t, err)

	metrics := ep.GetMetrics()

	assert.Equal(t, int64(1), metrics.EventsProcessed)
	assert.Greater(t, metrics.AverageLatency, time.Duration(0))
	assert.False(t, metrics.LastProcessedTime.IsZero())
}

// TestHealth tests health status
func TestHealth(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	health := ep.Health()

	assert.Equal(t, "healthy", health.Status)
	assert.False(t, health.Timestamp.IsZero())
	assert.Equal(t, "processor-1", health.Details["processor_id"])
	assert.Equal(t, "ethereum", health.Details["chain_id"])
}

// TestHealthDegraded tests degraded health status
func TestHealthDegraded(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	// Manually set high failure rate
	ep.mu.Lock()
	ep.processedCount = 100
	ep.failedCount = 20 // 20% failure rate
	ep.mu.Unlock()

	health := ep.Health()

	assert.Equal(t, "degraded", health.Status)
}

// TestConcurrentProcessing tests concurrent event processing
func TestConcurrentProcessing(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 10

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			ctx := context.Background()

			for i := 0; i < eventsPerGoroutine; i++ {
				event := &Event{
					ID:              fmt.Sprintf("event-%d-%d", goroutineID, i),
					BlockNumber:     uint64(12345 + i),
					TransactionHash: fmt.Sprintf("0xabc%d%d", goroutineID, i),
					LogIndex:        uint64(i),
					ContractAddress: "0xdef456",
					EventName:       "Transfer",
					EventData:       map[string]interface{}{},
					ChainID:         "ethereum",
				}

				err := ep.ProcessEvent(ctx, event)
				assert.NoError(t, err)
			}
		}(g)
	}

	wg.Wait()

	expectedCount := int64(numGoroutines * eventsPerGoroutine)
	assert.Equal(t, expectedCount, ep.GetProcessedCount())
}

// TestProcessEventWithContext tests event processing with context
func TestProcessEventWithContext(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ep.ProcessEvent(ctx, event)

	assert.NoError(t, err)
	assert.Equal(t, "processed", event.Status)
}

// TestProcessEventLatencyRecording tests latency recording
func TestProcessEventLatencyRecording(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)
	assert.NoError(t, err)

	metrics := ep.GetMetrics()

	assert.Greater(t, metrics.AverageLatency, time.Duration(0))
	assert.Greater(t, metrics.TotalProcessingTime, time.Duration(0))
}

// TestProcessBatchLatencyRecording tests batch latency recording
func TestProcessBatchLatencyRecording(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	events := make([]*Event, 5)
	for i := 0; i < 5; i++ {
		events[i] = &Event{
			ID:              fmt.Sprintf("event-%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: fmt.Sprintf("0xabc%d", i),
			LogIndex:        uint64(i),
			ContractAddress: "0xdef456",
			EventName:       "Transfer",
			EventData:       map[string]interface{}{},
			ChainID:         "ethereum",
		}
	}

	ctx := context.Background()
	err := ep.ProcessBatch(ctx, events)
	assert.NoError(t, err)

	metrics := ep.GetMetrics()

	assert.Equal(t, int64(1), metrics.BatchesProcessed)
	assert.Greater(t, metrics.TotalProcessingTime, time.Duration(0))
}

// TestGetID tests ID retrieval
func TestGetID(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)
	assert.Equal(t, "processor-1", ep.GetID())
}

// TestGetChainID tests chain ID retrieval
func TestGetChainID(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)
	assert.Equal(t, "ethereum", ep.GetChainID())
}

// TestGetProcessedCount tests processed count retrieval
func TestGetProcessedCount(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	event := &Event{
		ID:              "event-1",
		BlockNumber:     12345,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessEvent(ctx, event)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), ep.GetProcessedCount())
}

// TestGetFailedCount tests failed count retrieval
func TestGetFailedCount(t *testing.T) {
	ep := NewEventProcessor("processor-1", "ethereum", 100)

	// Create a batch with one invalid event to trigger failure
	events := make([]*Event, 2)
	events[0] = &Event{
		ID:              "event-0",
		BlockNumber:     12345,
		TransactionHash: "0xabc0",
		LogIndex:        0,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}
	events[1] = &Event{
		ID:              "", // Invalid - missing ID
		BlockNumber:     12346,
		TransactionHash: "0xabc1",
		LogIndex:        1,
		ContractAddress: "0xdef456",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{},
		ChainID:         "ethereum",
	}

	ctx := context.Background()
	err := ep.ProcessBatch(ctx, events)
	assert.Error(t, err)

	assert.Equal(t, int64(1), ep.GetFailedCount())
}
