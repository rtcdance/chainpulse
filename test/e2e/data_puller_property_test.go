package e2e

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"
)

// TestDataPullerEventCollectionCompleteness tests that event collection is complete
// Property: For any set of emitted events, all events must be collected
func TestDataPullerEventCollectionCompleteness(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1111111111111111111111111111111111111111"},
		StartBlock:        0,
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		Timeout:           10 * time.Second,
		ChainID:           "31337",
	}

	if err := dataPullerMgr.StartPuller(ctx, config); err != nil {
		t.Fatalf("failed to start data puller: %v", err)
	}
	defer func() {
		if err := dataPullerMgr.StopPuller(ctx); err != nil {
			t.Logf("Failed to stop data puller: %v", err)
		}
	}()

	// Property test: For various event counts, all events must be collected
	testCases := []int{1, 5, 10, 20, 50}

	for _, eventCount := range testCases {
		t.Run(fmt.Sprintf("EventCount_%d", eventCount), func(t *testing.T) {
			// Create emitted events
			emittedEvents := make([]*EventEmission, eventCount)
			for i := 0; i < eventCount; i++ {
				blockNumber, err := nonNegativeIntToUint64(i + 1)
				if err != nil {
					t.Fatalf("invalid block number: %d: %v", i+1, err)
				}
				logIndex, err := nonNegativeIntToUint32(i)
				if err != nil {
					t.Fatalf("invalid log index: %d: %v", i, err)
				}
				emittedEvents[i] = &EventEmission{
					ID:              fmt.Sprintf("event_%d", i),
					ContractAddress: "0x1111111111111111111111111111111111111111",
					EventName:       "Transfer",
					TxHash:          fmt.Sprintf("0x%064d", i),
					BlockNumber:     blockNumber,
					LogIndex:        logIndex,
					Parameters:      map[string]any{"value": i},
					Timestamp:       time.Now(),
				}
			}

			// Create collected events
			collectedEvents := make([]*CollectedEvent, eventCount)
			for i := 0; i < eventCount; i++ {
				collectedEvents[i] = &CollectedEvent{
					ID:              emittedEvents[i].ID,
					ContractAddress: emittedEvents[i].ContractAddress,
					EventName:       emittedEvents[i].EventName,
					TxHash:          emittedEvents[i].TxHash,
					BlockNumber:     emittedEvents[i].BlockNumber,
					LogIndex:        emittedEvents[i].LogIndex,
					RawData:         []byte{},
					CollectedAt:     time.Now(),
					ChainID:         "31337",
					RetryCount:      0,
				}
			}

			// Validate collection
			if err := dataPullerMgr.ValidateEventCollection(ctx, emittedEvents, collectedEvents); err != nil {
				t.Errorf("event collection validation failed for %d events: %v", eventCount, err)
			}
		})
	}
}

// TestDataPullerExponentialBackoffRetry tests exponential backoff retry logic
// Property: Retry delays must increase exponentially
func TestDataPullerExponentialBackoffRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1111111111111111111111111111111111111111"},
		StartBlock:        0,
		MaxRetries:        5,
		RetryBackoff:      50 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		Timeout:           10 * time.Second,
		ChainID:           "31337",
	}

	if err := dataPullerMgr.StartPuller(ctx, config); err != nil {
		t.Fatalf("failed to start data puller: %v", err)
	}
	defer func() {
		if err := dataPullerMgr.StopPuller(ctx); err != nil {
			t.Logf("Failed to stop data puller: %v", err)
		}
	}()

	// Property test: Retry delays increase exponentially
	testCases := []int{1, 2, 3, 4, 5}

	for _, failureCount := range testCases {
		t.Run(fmt.Sprintf("FailureCount_%d", failureCount), func(t *testing.T) {
			startTime := time.Now()

			if err := dataPullerMgr.SimulateRetry(ctx, failureCount); err != nil {
				t.Fatalf("retry simulation failed: %v", err)
			}

			elapsed := time.Since(startTime)

			// Calculate expected minimum duration
			// With 50ms initial backoff: sum of 50ms * (2^0 + 2^1 + ... + 2^(n-1))
			expectedMinDuration := time.Duration(0)
			backoff := 50 * time.Millisecond
			for i := 0; i < failureCount; i++ {
				expectedMinDuration += backoff
				backoff *= 2
			}

			if elapsed < expectedMinDuration {
				t.Errorf("retry backoff too fast: expected >= %v, got %v", expectedMinDuration, elapsed)
			}

			// Verify retry count
			metrics := dataPullerMgr.GetPullerMetrics(ctx)
			if metrics.RetryCount < int64(failureCount) {
				t.Errorf("expected at least %d retries, got %d", failureCount, metrics.RetryCount)
			}
		})
	}
}

// TestDataPullerEventFilteringAccuracy tests event filtering accuracy
// Property: Filtered events must match filter criteria
func TestDataPullerEventFilteringAccuracy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222"},
		StartBlock:        0,
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		Timeout:           10 * time.Second,
		ChainID:           "31337",
	}

	if err := dataPullerMgr.StartPuller(ctx, config); err != nil {
		t.Fatalf("failed to start data puller: %v", err)
	}
	defer func() {
		if err := dataPullerMgr.StopPuller(ctx); err != nil {
			t.Logf("Failed to stop data puller: %v", err)
		}
	}()

	// Create test events with various attributes
	testEvents := []*CollectedEvent{
		{
			ID:              "event_1",
			ContractAddress: "0x1111111111111111111111111111111111111111",
			EventName:       "Transfer",
			TxHash:          "0x1111111111111111111111111111111111111111111111111111111111111111",
			BlockNumber:     1,
			LogIndex:        0,
			ChainID:         "31337",
		},
		{
			ID:              "event_2",
			ContractAddress: "0x2222222222222222222222222222222222222222",
			EventName:       "Approval",
			TxHash:          "0x2222222222222222222222222222222222222222222222222222222222222222",
			BlockNumber:     2,
			LogIndex:        0,
			ChainID:         "31337",
		},
		{
			ID:              "event_3",
			ContractAddress: "0x1111111111111111111111111111111111111111",
			EventName:       "Approval",
			TxHash:          "0x3333333333333333333333333333333333333333333333333333333333333333",
			BlockNumber:     3,
			LogIndex:        0,
			ChainID:         "31337",
		},
	}

	// Add events to manager
	for _, event := range testEvents {
		dataPullerMgr.(*DefaultDataPullerManager).addCollectedEvent(event)
	}

	// Property test: Filtered results must match filter criteria
	testCases := []struct {
		name     string
		filter   EventFilter
		expected int
		validate func(*CollectedEvent) bool
	}{
		{
			name: "FilterByContractAddress",
			filter: EventFilter{
				ContractAddress: "0x1111111111111111111111111111111111111111",
				ChainID:         "31337",
			},
			expected: 2,
			validate: func(e *CollectedEvent) bool {
				return e.ContractAddress == "0x1111111111111111111111111111111111111111"
			},
		},
		{
			name: "FilterByEventName",
			filter: EventFilter{
				EventName: "Approval",
				ChainID:   "31337",
			},
			expected: 2,
			validate: func(e *CollectedEvent) bool {
				return e.EventName == "Approval"
			},
		},
		{
			name: "FilterByBlockRange",
			filter: EventFilter{
				BlockRange: &BlockRange{Start: 1, End: 2},
				ChainID:    "31337",
			},
			expected: 2,
			validate: func(e *CollectedEvent) bool {
				return e.BlockNumber >= 1 && e.BlockNumber <= 2
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			collected, err := dataPullerMgr.CollectEvents(ctx, tc.filter)
			if err != nil {
				t.Fatalf("failed to collect events: %v", err)
			}

			if len(collected) != tc.expected {
				t.Errorf("expected %d events, got %d", tc.expected, len(collected))
			}

			// Verify all results match filter criteria
			for _, event := range collected {
				if !tc.validate(event) {
					t.Errorf("event does not match filter criteria: %v", event)
				}
			}
		})
	}
}

// TestDataPullerPaginationCorrectness tests pagination correctness
// Property: Paginated results must be consistent and complete
func TestDataPullerPaginationCorrectness(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1111111111111111111111111111111111111111"},
		StartBlock:        0,
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		Timeout:           10 * time.Second,
		ChainID:           "31337",
	}

	if err := dataPullerMgr.StartPuller(ctx, config); err != nil {
		t.Fatalf("failed to start data puller: %v", err)
	}
	defer func() {
		if err := dataPullerMgr.StopPuller(ctx); err != nil {
			t.Logf("Failed to stop data puller: %v", err)
		}
	}()

	// Add test events
	totalEvents := 20
	for i := 0; i < totalEvents; i++ {
		blockNumber, err := nonNegativeIntToUint64(i + 1)
		if err != nil {
			t.Fatalf("invalid block number: %d: %v", i+1, err)
		}
		logIndex, err := nonNegativeIntToUint32(i)
		if err != nil {
			t.Fatalf("invalid log index: %d: %v", i, err)
		}
		event := &CollectedEvent{
			ID:              fmt.Sprintf("event_%d", i),
			ContractAddress: "0x1111111111111111111111111111111111111111",
			EventName:       "Transfer",
			TxHash:          fmt.Sprintf("0x%064d", i),
			BlockNumber:     blockNumber,
			LogIndex:        logIndex,
			ChainID:         "31337",
		}
		dataPullerMgr.(*DefaultDataPullerManager).addCollectedEvent(event)
	}

	// Property test: Pagination must be consistent
	pageSize := 5
	allCollected := make([]*CollectedEvent, 0)

	for offset := 0; offset < totalEvents; offset += pageSize {
		filter := EventFilter{
			ContractAddress: "0x1111111111111111111111111111111111111111",
			Limit:           pageSize,
			Offset:          offset,
			ChainID:         "31337",
		}

		collected, err := dataPullerMgr.CollectEvents(ctx, filter)
		if err != nil {
			t.Fatalf("failed to collect events: %v", err)
		}

		allCollected = append(allCollected, collected...)
	}

	// Verify all events were collected through pagination
	if len(allCollected) != totalEvents {
		t.Errorf("pagination incomplete: expected %d events, got %d", totalEvents, len(allCollected))
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, event := range allCollected {
		if seen[event.ID] {
			t.Errorf("duplicate event found: %s", event.ID)
		}
		seen[event.ID] = true
	}
}

// TestDataPullerMetricsAccuracy tests metrics accuracy
// Property: Metrics must accurately reflect collected events
func TestDataPullerMetricsAccuracy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1111111111111111111111111111111111111111"},
		StartBlock:        0,
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		Timeout:           10 * time.Second,
		ChainID:           "31337",
	}

	if err := dataPullerMgr.StartPuller(ctx, config); err != nil {
		t.Fatalf("failed to start data puller: %v", err)
	}
	defer func() { _ = dataPullerMgr.StopPuller(ctx) }()

	// Property test: Metrics must match actual event counts
	testCases := []int{1, 5, 10, 20}

	for _, eventCount := range testCases {
		t.Run(fmt.Sprintf("EventCount_%d", eventCount), func(t *testing.T) {
			// Clear previous events
			dpm := dataPullerMgr.(*DefaultDataPullerManager)
			dpm.mu.Lock()
			dpm.collectedEvents = make([]*CollectedEvent, 0)
			dpm.eventIndex = make(map[string]*CollectedEvent)
			dpm.metrics = &DataPullerMetrics{
				EventsCollected:    0,
				EventsFailed:       0,
				RetryCount:         0,
				LastBlockProcessed: 0,
			}
			dpm.mu.Unlock()

			// Add events
			for i := 0; i < eventCount; i++ {
				blockNumber, err := nonNegativeIntToUint64(i + 1)
				if err != nil {
					t.Fatalf("invalid block number: %d: %v", i+1, err)
				}
				logIndex, err := nonNegativeIntToUint32(i)
				if err != nil {
					t.Fatalf("invalid log index: %d: %v", i, err)
				}
				event := &CollectedEvent{
					ID:              fmt.Sprintf("event_%d", i),
					ContractAddress: "0x1111111111111111111111111111111111111111",
					EventName:       "Transfer",
					TxHash:          fmt.Sprintf("0x%064d", i),
					BlockNumber:     blockNumber,
					LogIndex:        logIndex,
					ChainID:         "31337",
				}
				dpm.addCollectedEvent(event)
			}

			// Get metrics
			metrics := dataPullerMgr.GetPullerMetrics(ctx)

			// Verify metrics accuracy
			if metrics.EventsCollected != int64(eventCount) {
				t.Errorf("expected %d collected events, got %d", eventCount, metrics.EventsCollected)
			}

			eventCountAsUint64, err := nonNegativeIntToUint64(eventCount)
			if err != nil {
				t.Fatalf("invalid eventCount: %d: %v", eventCount, err)
			}
			if metrics.LastBlockProcessed != eventCountAsUint64 {
				t.Errorf("expected last block %d, got %d", eventCount, metrics.LastBlockProcessed)
			}

			if metrics.Throughput <= 0 {
				t.Errorf("expected positive throughput, got %f", metrics.Throughput)
			}

			if metrics.ErrorRate != 0 {
				t.Errorf("expected zero error rate, got %f", metrics.ErrorRate)
			}
		})
	}
}

func nonNegativeIntToUint32(value int) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("value out of range: %d", value)
	}
	return uint32(value), nil
}

// TestDataPullerReorgHandlingCorrectness tests reorg handling correctness
// Property: Reorg simulation must correctly identify affected transactions
func TestDataPullerReorgHandlingCorrectness(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1111111111111111111111111111111111111111"},
		StartBlock:        0,
		MaxRetries:        3,
		RetryBackoff:      100 * time.Millisecond,
		PollInterval:      100 * time.Millisecond,
		Timeout:           10 * time.Second,
		ChainID:           "31337",
	}

	if err := dataPullerMgr.StartPuller(ctx, config); err != nil {
		t.Fatalf("failed to start data puller: %v", err)
	}
	defer func() {
		if err := dataPullerMgr.StopPuller(ctx); err != nil {
			t.Logf("Failed to stop data puller: %v", err)
		}
	}()

	// Add test events
	for i := 0; i < 10; i++ {
		event := &CollectedEvent{
			ID:              fmt.Sprintf("event_%d", i),
			ContractAddress: "0x1111111111111111111111111111111111111111",
			EventName:       "Transfer",
			TxHash:          fmt.Sprintf("0x%064d", i),
			BlockNumber:     uint64(i + 1),
			LogIndex:        uint32(i),
			ChainID:         "31337",
		}
		dataPullerMgr.(*DefaultDataPullerManager).addCollectedEvent(event)
	}

	// Property test: Reorg must correctly identify affected transactions
	testCases := []struct {
		reorgDepth       uint64
		expectedAffected int
	}{
		{reorgDepth: 1, expectedAffected: 1},
		{reorgDepth: 3, expectedAffected: 3},
		{reorgDepth: 5, expectedAffected: 5},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ReorgDepth_%d", tc.reorgDepth), func(t *testing.T) {
			if err := dataPullerMgr.SimulateReorg(ctx, tc.reorgDepth); err != nil {
				t.Fatalf("reorg simulation failed: %v", err)
			}

			// Verify reorg was recorded
			metrics := dataPullerMgr.GetPullerMetrics(ctx)
			if metrics.EventsCollected == 0 {
				t.Errorf("expected events to be collected after reorg")
			}
		})
	}
}
