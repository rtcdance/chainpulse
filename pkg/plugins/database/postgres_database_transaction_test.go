package database

import (
	"chainpulse/pkg/core"
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// TestTransactionIsolation tests transaction isolation
func TestTransactionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: "chainpulse",
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Create test event
	event := &core.BlockchainEvent{
		ID:              "test-tx-isolation-1",
		BlockNumber:     1,
		TransactionHash: common.HexToHash("0x1234567890abcdef"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x123"),
		EventName:       "Transfer",
		EventData:       []byte("data1"),
		BlockTimestamp:  time.Now().Unix(),
	}

	// Test write in transaction
	ctx := context.Background()
	tx, err := db.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert event
	_, err = tx.ExecContext(ctx, `
		INSERT INTO blockchain_events (id, block_number, transaction_hash, log_index, contract_address, event_name, event_data, block_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.BlockNumber, event.TransactionHash, event.LogIndex, event.ContractAddress, event.EventName, event.EventData, event.BlockTimestamp)

	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to insert: %v", err)
	}

	// Commit
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify insert
	retrieved, err := db.GetEventByHash(event.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Event not found")
	}

	t.Log("Transaction isolation test passed")
}

// TestRollbackOnError tests rollback when error occurs
func TestRollbackOnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: "chainpulse",
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Create test events with duplicate hash
	events := []core.BlockchainEvent{
		{
			ID:              "test-rollback-1",
			BlockNumber:     2,
			TransactionHash: common.HexToHash("0x1234567890abcdef"),
			LogIndex:        0,
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
			BlockTimestamp:  time.Now().Unix(),
		},
		{
			ID:              "test-rollback-1", // Duplicate hash - will cause error
			BlockNumber:     3,
			TransactionHash: common.HexToHash("0x2345678901bcdef0"),
			LogIndex:        0,
			ContractAddress: common.HexToAddress("0x123"),
			EventName:       "Transfer",
			EventData:       []byte("data1"),
			BlockTimestamp:  time.Now().Unix(),
		},
	}

	// Try to write events (should fail on duplicate)
	err = db.WriteEvents(events)
	if err == nil {
		t.Fatal("Expected error on duplicate hash")
	}

	// Verify first event was rolled back
	retrieved, err := db.GetEventByHash("test-rollback-1")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if retrieved != nil {
		t.Fatal("Event should have been rolled back")
	}

	t.Log("Rollback on error test passed")
}

// TestConcurrentTransactions tests concurrent transaction handling
func TestConcurrentTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: "chainpulse",
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Run concurrent transactions
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			event := &core.BlockchainEvent{
				ID:              fmt.Sprintf("test-concurrent-%d", id),
				BlockNumber:     uint64(id),
				TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", id)),
				LogIndex:        uint(id),
				ContractAddress: common.HexToAddress("0x123"),
				EventName:       "Transfer",
				EventData:       []byte("data1"),
				BlockTimestamp:  time.Now().Unix(),
			}

			err := db.WriteEvent(event)
			if err != nil {
				t.Errorf("Failed to write event %d: %v", id, err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	t.Log("Concurrent transactions test passed")
}

// TestTransactionConsistency tests ACID compliance
func TestTransactionConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &core.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "chainpulse",
		PostgresPassword: "chainpulse",
		PostgresDB:       "chainpulse",
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	db := NewPostgreSQLDatabase(logger, metrics)
	err := db.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() { _ = db.Stop() }()

	// Write event
	event := &core.BlockchainEvent{
		ID:              "test-consistency-1",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0x1234567890abcdef"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x123"),
		EventName:       "Transfer",
		EventData:       []byte("data1"),
		BlockTimestamp:  time.Now().Unix(),
	}

	err = db.WriteEvent(event)
	if err != nil {
		t.Fatalf("Failed to write event: %v", err)
	}

	// Read event multiple times - should get same result
	for i := 0; i < 3; i++ {
		retrieved, err := db.GetEventByHash(event.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve event (attempt %d): %v", i, err)
		}

		if retrieved == nil {
			t.Fatalf("Event not found (attempt %d)", i)
		}

		if retrieved.ID != event.ID {
			t.Fatalf("Hash mismatch (attempt %d): expected %s, got %s", i, event.ID, retrieved.ID)
		}
	}

	t.Log("Transaction consistency test passed")
}
