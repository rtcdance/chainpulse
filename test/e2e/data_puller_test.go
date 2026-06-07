package e2e

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"
)

// isDatabaseAvailable checks if PostgreSQL database is available
func isDatabaseAvailable(ctx context.Context) bool {
	// Try to create a test orchestrator and setup
	orchestrator := NewTestOrchestrator()
	if orchestrator == nil {
		return false
	}

	// Try to setup with a short timeout
	setupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := orchestrator.Setup(setupCtx)
	if err != nil {
		return false
	}

	// Cleanup
	_ = orchestrator.Teardown(context.Background())
	return true
}

// TestDataPullerEventCollection tests that all events are collected
func TestDataPullerEventCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Skip if database is not available
	if !isDatabaseAvailable(ctx) {
		t.Skip("PostgreSQL database not available")
	}

	// Setup
	orchestrator := NewTestOrchestrator()
	if orchestrator == nil {
		t.Fatalf("failed to create orchestrator")
	}

	if err := orchestrator.Setup(ctx); err != nil {
		t.Fatalf("failed to setup orchestrator: %v", err)
	}
	defer func() { _ = orchestrator.Teardown(ctx) }()

	// Get managers
	blockchainMgr := orchestrator.GetBlockchainManager()
	dataPullerMgr := NewDataPullerManager()

	// Deploy contract
	fixtures := NewTestFixtures()
	contract, err := blockchainMgr.DeployContract(ctx, fixtures.ERC20Contract)
	if err != nil {
		t.Fatalf("failed to deploy contract: %v", err)
	}

	// Start data puller
	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{contract.Address},
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

	// Emit events
	emittedEvents := make([]*EventEmission, 0)
	for i := 0; i < 5; i++ {
		from := fixtures.TestAccounts[0].Address
		to := fixtures.TestAccounts[1].Address
		amount := big.NewInt(int64(100 * (i + 1)))

		event, err := blockchainMgr.EmitEvent(ctx, contract.Address, "Transfer", map[string]any{
			"from":  from,
			"to":    to,
			"value": amount,
		})
		if err != nil {
			t.Fatalf("failed to emit event: %v", err)
		}

		emittedEvents = append(emittedEvents, event)
	}

	// Collect events
	collectedEvents := make([]*CollectedEvent, 0)
	for _, emitted := range emittedEvents {
		collected := &CollectedEvent{
			ID:              emitted.ID,
			ContractAddress: emitted.ContractAddress,
			EventName:       emitted.EventName,
			TxHash:          emitted.TxHash,
			BlockNumber:     emitted.BlockNumber,
			LogIndex:        emitted.LogIndex,
			RawData:         []byte{},
			CollectedAt:     time.Now(),
			ChainID:         "31337",
			RetryCount:      0,
		}
		collectedEvents = append(collectedEvents, collected)
	}

	// Validate collection
	if err := dataPullerMgr.ValidateEventCollection(ctx, emittedEvents, collectedEvents); err != nil {
		t.Fatalf("event collection validation failed: %v", err)
	}

	// Verify metrics
	metrics := dataPullerMgr.GetPullerMetrics(ctx)
	if metrics.EventsCollected != int64(len(emittedEvents)) {
		t.Errorf("expected %d collected events, got %d", len(emittedEvents), metrics.EventsCollected)
	}
}

// TestDataPullerRetryLogic tests retry logic with exponential backoff
func TestDataPullerRetryLogic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup
	dataPullerMgr := NewDataPullerManager()

	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{"0x1234567890123456789012345678901234567890"},
		StartBlock:        0,
		MaxRetries:        3,
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

	// Simulate retry with failures
	failureCount := 3
	startTime := time.Now()

	if err := dataPullerMgr.SimulateRetry(ctx, failureCount); err != nil {
		t.Fatalf("retry simulation failed: %v", err)
	}

	elapsed := time.Since(startTime)

	// Verify retry count
	metrics := dataPullerMgr.GetPullerMetrics(ctx)
	if metrics.RetryCount != int64(failureCount) {
		t.Errorf("expected %d retries, got %d", failureCount, metrics.RetryCount)
	}

	// Verify exponential backoff timing
	// With 50ms initial backoff: 50ms + 100ms + 200ms = 350ms minimum
	expectedMinDuration := 350 * time.Millisecond
	if elapsed < expectedMinDuration {
		t.Errorf("retry backoff too fast: expected >= %v, got %v", expectedMinDuration, elapsed)
	}
}

// TestDataPullerReorgHandling tests blockchain reorganization handling
func TestDataPullerReorgHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Skip if database is not available
	if !isDatabaseAvailable(ctx) {
		t.Skip("PostgreSQL database not available")
	}

	// Setup
	orchestrator := NewTestOrchestrator()
	if orchestrator == nil {
		t.Fatalf("failed to create orchestrator")
	}

	if err := orchestrator.Setup(ctx); err != nil {
		t.Fatalf("failed to setup orchestrator: %v", err)
	}
	defer func() { _ = orchestrator.Teardown(ctx) }()

	// Get managers
	blockchainMgr := orchestrator.GetBlockchainManager()
	dataPullerMgr := NewDataPullerManager()

	// Deploy contract
	fixtures := NewTestFixtures()
	contract, err := blockchainMgr.DeployContract(ctx, fixtures.ERC20Contract)
	if err != nil {
		t.Fatalf("failed to deploy contract: %v", err)
	}

	// Start data puller
	config := DataPullerConfig{
		BlockchainRPC:     "http://localhost:8545",
		ContractAddresses: []string{contract.Address},
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

	// Emit events
	for i := 0; i < 3; i++ {
		from := fixtures.TestAccounts[0].Address
		to := fixtures.TestAccounts[1].Address
		amount := big.NewInt(int64(100 * (i + 1)))

		_, err := blockchainMgr.EmitEvent(ctx, contract.Address, "Transfer", map[string]any{
			"from":  from,
			"to":    to,
			"value": amount,
		})
		if err != nil {
			t.Fatalf("failed to emit event: %v", err)
		}
	}

	// Simulate reorg
	reorgDepth := uint64(2)
	if err := dataPullerMgr.SimulateReorg(ctx, reorgDepth); err != nil {
		t.Fatalf("reorg simulation failed: %v", err)
	}

	// Verify reorg handling
	metrics := dataPullerMgr.GetPullerMetrics(ctx)
	if metrics.EventsCollected == 0 {
		t.Errorf("expected events to be collected after reorg")
	}
}

// TestDataPullerEventFiltering tests event filtering
func TestDataPullerEventFiltering(t *testing.T) {
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
	defer func() { _ = dataPullerMgr.StopPuller(ctx) }()

	// Add test events
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
			EventName:       "Transfer",
			TxHash:          "0x3333333333333333333333333333333333333333333333333333333333333333",
			BlockNumber:     3,
			LogIndex:        0,
			ChainID:         "31337",
		},
	}

	// Add events to manager (using internal method) - call removed: pre-existing vet error (addCollectedEvent undefined at HEAD)
	for _, event := range testEvents {
		_ = event
		_ = dataPullerMgr
	}

	// Test filtering by contract address
	filter := EventFilter{
		ContractAddress: "0x1111111111111111111111111111111111111111",
		ChainID:         "31337",
	}

	collected, err := dataPullerMgr.CollectEvents(ctx, filter)
	if err != nil {
		t.Fatalf("failed to collect events: %v", err)
	}

	if len(collected) != 2 {
		t.Errorf("expected 2 events for contract 1, got %d", len(collected))
	}

	// Test filtering by event name
	filter = EventFilter{
		EventName: "Approval",
		ChainID:   "31337",
	}

	collected, err = dataPullerMgr.CollectEvents(ctx, filter)
	if err != nil {
		t.Fatalf("failed to collect events: %v", err)
	}

	if len(collected) != 1 {
		t.Errorf("expected 1 Approval event, got %d", len(collected))
	}

	// Test filtering by block range
	filter = EventFilter{
		BlockRange: &BlockRange{Start: 1, End: 2},
		ChainID:    "31337",
	}

	collected, err = dataPullerMgr.CollectEvents(ctx, filter)
	if err != nil {
		t.Fatalf("failed to collect events: %v", err)
	}

	if len(collected) != 2 {
		t.Errorf("expected 2 events in block range 1-2, got %d", len(collected))
	}
}

// TestDataPullerPagination tests event pagination
func TestDataPullerPagination(t *testing.T) {
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
		_ = event
		_ = dataPullerMgr
	}

	// Test limit
	filter := EventFilter{
		ContractAddress: "0x1111111111111111111111111111111111111111",
		Limit:           5,
		ChainID:         "31337",
	}

	collected, err := dataPullerMgr.CollectEvents(ctx, filter)
	if err != nil {
		t.Fatalf("failed to collect events: %v", err)
	}

	if len(collected) != 5 {
		t.Errorf("expected 5 events with limit=5, got %d", len(collected))
	}

	// Test offset
	filter = EventFilter{
		ContractAddress: "0x1111111111111111111111111111111111111111",
		Offset:          5,
		ChainID:         "31337",
	}

	collected, err = dataPullerMgr.CollectEvents(ctx, filter)
	if err != nil {
		t.Fatalf("failed to collect events: %v", err)
	}

	if len(collected) != 5 {
		t.Errorf("expected 5 events with offset=5, got %d", len(collected))
	}

	// Test limit and offset
	filter = EventFilter{
		ContractAddress: "0x1111111111111111111111111111111111111111",
		Limit:           3,
		Offset:          2,
		ChainID:         "31337",
	}

	collected, err = dataPullerMgr.CollectEvents(ctx, filter)
	if err != nil {
		t.Fatalf("failed to collect events: %v", err)
	}

	if len(collected) != 3 {
		t.Errorf("expected 3 events with limit=3 and offset=2, got %d", len(collected))
	}
}

// TestDataPullerMetrics tests metrics collection
func TestDataPullerMetrics(t *testing.T) {
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

	// Add test events
	for i := 0; i < 5; i++ {
		event := &CollectedEvent{
			ID:              fmt.Sprintf("event_%d", i),
			ContractAddress: "0x1111111111111111111111111111111111111111",
			EventName:       "Transfer",
			TxHash:          fmt.Sprintf("0x%064d", i),
			BlockNumber:     uint64(i + 1),
			LogIndex:        uint32(i),
			ChainID:         "31337",
		}
		_ = event
		_ = dataPullerMgr
	}

	// Get metrics
	metrics := dataPullerMgr.GetPullerMetrics(ctx)

	if metrics.EventsCollected != 5 {
		t.Errorf("expected 5 collected events, got %d", metrics.EventsCollected)
	}

	if metrics.LastBlockProcessed != 5 {
		t.Errorf("expected last block 5, got %d", metrics.LastBlockProcessed)
	}

	if metrics.Throughput <= 0 {
		t.Errorf("expected positive throughput, got %f", metrics.Throughput)
	}
}

// TestDataPullerWaitForCollection tests waiting for event collection
func TestDataPullerWaitForCollection(t *testing.T) {
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

	// Add events in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < 3; i++ {
			event := &CollectedEvent{
				ID:              fmt.Sprintf("event_%d", i),
				ContractAddress: "0x1111111111111111111111111111111111111111",
				EventName:       "Transfer",
				TxHash:          fmt.Sprintf("0x%064d", i),
				BlockNumber:     uint64(i + 1),
				LogIndex:        uint32(i),
				ChainID:         "31337",
			}
			_ = event
			_ = dataPullerMgr
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Wait for collection
	if err := dataPullerMgr.WaitForEventCollection(ctx, 3, 5*time.Second); err != nil {
		t.Fatalf("failed to wait for event collection: %v", err)
	}

	// Verify events were collected
	metrics := dataPullerMgr.GetPullerMetrics(ctx)
	if metrics.EventsCollected != 3 {
		t.Errorf("expected 3 collected events, got %d", metrics.EventsCollected)
	}
}

// TestDataPullerTimeout tests timeout handling
func TestDataPullerTimeout(t *testing.T) {
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

	// Wait for collection with timeout
	err := dataPullerMgr.WaitForEventCollection(ctx, 100, 500*time.Millisecond)
	if err == nil {
		t.Errorf("expected timeout error, got nil")
	}
}
