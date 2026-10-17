package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

func TestDatabasePluginInitialize(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	err := db.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test double initialization
	err = db.Initialize(context.Background(), config)
	if err == nil {
		t.Fatal("Expected error on double initialization")
	}
}

func TestDatabasePluginLifecycle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	_ = db.Initialize(context.Background(), config)

	// Start
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check health
	if err := db.Health(context.Background()); err != nil {
		t.Fatalf("Expected healthy, got error: %v", err)
	}

	// Stop
	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Check health after stop — should return error
	if err := db.Health(context.Background()); err == nil {
		t.Fatal("Expected unhealthy status after stop, got nil error")
	}
}

func TestDatabasePluginWriteEvent(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write event
	event := &blockchain.BlockchainEvent{
		EventHash:       "0xabc123",
		BlockNumber:     12345,
		TransactionHash: common.HexToHash("0xdef456"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0xghi789"),
		EventName:       "Transfer",
		ChainID:         "1",
		BlockTimestamp:  time.Now().Unix(),
	}

	if err := db.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Verify event count
	if db.GetEventCount() != 1 {
		t.Fatalf("Expected 1 event, got %d", db.GetEventCount())
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginWriteEvents(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write multiple events
	events := make([]blockchain.BlockchainEvent, 5)
	for i := 0; i < 5; i++ {
		events[i] = blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xhash%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0xtx%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0xcontract"),
			EventName:       "Transfer",
			ChainID:         "1",
			BlockTimestamp:  time.Now().Unix(),
		}
	}

	if err := db.WriteEvents(context.Background(), events); err != nil {
		t.Fatalf("WriteEvents failed: %v", err)
	}

	// Verify event count
	if db.GetEventCount() != 5 {
		t.Fatalf("Expected 5 events, got %d", db.GetEventCount())
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginQueryEvents(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write events
	for i := 0; i < 10; i++ {
		event := &blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xhash%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0xtx%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0xcontract"),
			EventName:       "Transfer",
			ChainID:         "1",
			BlockTimestamp:  time.Now().Unix(),
		}
		if err := db.WriteEvent(context.Background(), event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	// Query all events
	filter := &core.EventFilter{
		Limit: 100,
	}

	result, err := db.QueryEvents(filter)
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}

	if result.Total != 10 {
		t.Fatalf("Expected 10 events, got %d", result.Total)
	}

	if len(result.Events) != 10 {
		t.Fatalf("Expected 10 events in result, got %d", len(result.Events))
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginQueryEventsWithFilter(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write events with different contract addresses
	for i := 0; i < 5; i++ {
		event := &blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xhash%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0xtx%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			EventName:       "Transfer",
			ChainID:         "1",
			BlockTimestamp:  time.Now().Unix(),
		}
		if err := db.WriteEvent(context.Background(), event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	for i := 5; i < 10; i++ {
		event := &blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xhash%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0xtx%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"),
			EventName:       "Approval",
			ChainID:         "1",
			BlockTimestamp:  time.Now().Unix(),
		}
		if err := db.WriteEvent(context.Background(), event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	// Query by contract address
	filter := &core.EventFilter{
		ContractAddress: []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")},
		Limit:           100,
	}

	result, err := db.QueryEvents(filter)
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}

	if result.Total != 5 {
		t.Fatalf("Expected 5 events, got %d", result.Total)
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginGetEventByHash(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write event
	event := &blockchain.BlockchainEvent{
		EventHash:       "0xabc123",
		BlockNumber:     12345,
		TransactionHash: common.HexToHash("0xdef456"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0xghi789"),
		EventName:       "Transfer",
		ChainID:         "1",
		BlockTimestamp:  time.Now().Unix(),
	}

	if err := db.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Get event by hash
	retrieved, err := db.GetEventByHash("0xabc123")
	if err != nil {
		t.Fatalf("GetEventByHash failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve event")
	}

	if retrieved.EventHash != "0xabc123" {
		t.Fatalf("Expected hash 0xabc123, got %s", retrieved.EventHash)
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginDeleteEvent(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write event
	event := &blockchain.BlockchainEvent{
		EventHash:       "0xabc123",
		BlockNumber:     12345,
		TransactionHash: common.HexToHash("0xdef456"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0xghi789"),
		EventName:       "Transfer",
		ChainID:         "1",
		BlockTimestamp:  time.Now().Unix(),
	}

	if err := db.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Verify event exists
	retrieved, _ := db.GetEventByHash("0xabc123")
	if retrieved == nil {
		t.Fatal("Expected event to exist")
	}

	// Delete event
	if err := db.DeleteEvent(context.Background(), "0xabc123"); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	// Verify event is deleted
	retrieved, _ = db.GetEventByHash("0xabc123")
	if retrieved != nil {
		t.Fatal("Expected event to be deleted")
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginStats(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write events
	for i := 0; i < 5; i++ {
		event := &blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xhash%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0xtx%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0xcontract"),
			EventName:       "Transfer",
			ChainID:         "1",
			BlockTimestamp:  time.Now().Unix(),
		}
		if err := db.WriteEvent(context.Background(), event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	// Query events
	filter := &core.EventFilter{Limit: 100}
	if _, err := db.QueryEvents(filter); err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}

	// Delete event
	if err := db.DeleteEvent(context.Background(), "0xhash0"); err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	// Get stats
	stats := db.GetStats()
	if stats.WriteCount != 5 {
		t.Fatalf("Expected 5 writes, got %d", stats.WriteCount)
	}

	if stats.ReadCount != 1 {
		t.Fatalf("Expected 1 read, got %d", stats.ReadCount)
	}

	if stats.DeleteCount != 1 {
		t.Fatalf("Expected 1 delete, got %d", stats.DeleteCount)
	}

	if stats.EventCount != 4 {
		t.Fatalf("Expected 4 events, got %d", stats.EventCount)
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginConcurrentOperations(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Concurrent write operations
	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 10; j++ {
				event := &blockchain.BlockchainEvent{
					EventHash:       fmt.Sprintf("0xhash_%d_%d", idx, j),
					BlockNumber:     uint64(12345 + idx*10 + j),
					TransactionHash: common.HexToHash(fmt.Sprintf("0xtx_%d_%d", idx, j)),
					LogIndex:        uint64(j),
					ContractAddress: common.HexToAddress("0xcontract"),
					EventName:       "Transfer",
					ChainID:         "1",
					BlockTimestamp:  time.Now().Unix(),
				}
				if err := db.WriteEvent(context.Background(), event); err != nil {
					t.Logf("WriteEvent failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func(idx int) {
			filter := &core.EventFilter{Limit: 100}
			if _, err := db.QueryEvents(filter); err != nil {
				t.Logf("QueryEvents failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	if db.GetEventCount() != 100 {
		t.Fatalf("Expected 100 events, got %d", db.GetEventCount())
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginErrorHandling(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test write with nil event
	err := db.WriteEvent(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil event")
	}

	// Test write with empty hash
	event := &blockchain.BlockchainEvent{
		BlockNumber: 12345,
	}
	err = db.WriteEvent(context.Background(), event)
	if err == nil {
		t.Fatal("Expected error for empty hash")
	}

	// Test query with nil filter
	_, err = db.QueryEvents(nil)
	if err == nil {
		t.Fatal("Expected error for nil filter")
	}

	// Test get with empty hash
	_, err = db.GetEventByHash("")
	if err == nil {
		t.Fatal("Expected error for empty hash")
	}

	// Test delete with empty hash
	err = db.DeleteEvent(context.Background(), "")
	if err == nil {
		t.Fatal("Expected error for empty hash")
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDatabasePluginQueryWithPagination(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	db := NewDefaultInMemoryDatabasePlugin(logger, metrics)

	config := core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}

	if err := db.Initialize(context.Background(), config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := db.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Write 20 events
	for i := 0; i < 20; i++ {
		event := &blockchain.BlockchainEvent{
			EventHash:       fmt.Sprintf("0xhash%d", i),
			BlockNumber:     uint64(12345 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0xtx%d", i)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0xcontract"),
			EventName:       "Transfer",
			ChainID:         "1",
			BlockTimestamp:  time.Now().Unix(),
		}
		if err := db.WriteEvent(context.Background(), event); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	// Query with limit
	filter := &core.EventFilter{
		Limit: 5,
	}

	result, err := db.QueryEvents(filter)
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}

	if len(result.Events) != 5 {
		t.Fatalf("Expected 5 events in result, got %d", len(result.Events))
	}

	if result.Total != 20 {
		t.Fatalf("Expected total 20, got %d", result.Total)
	}

	if err := db.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
