package database

import (
	"math/big"
	"os"
	"testing"

	"chainpulse/shared/types"
)

func TestDatabase_SaveLastProcessedBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database test in short mode")
	}

	// Use a test database URL or skip if not available
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://chainpulse:password@localhost:5432/chainpulse_test?sslmode=disable"
	}

	db, err := NewDatabase(dsn)
	if err != nil {
		t.Skipf("skipping test: could not connect to database: %v", err)
	}

	// Test saving a block number
	blockNum := big.NewInt(12345)
	err = db.SaveLastProcessedBlock(blockNum)
	if err != nil {
		t.Fatalf("Failed to save last processed block: %v", err)
	}

	// Retrieve the saved block number
	retrievedBlock, err := db.GetLastProcessedBlock()
	if err != nil {
		t.Fatalf("Failed to get last processed block: %v", err)
	}

	if retrievedBlock.Cmp(blockNum) != 0 {
		t.Errorf("Expected block number %s, got %s", blockNum.String(), retrievedBlock.String())
	}
}

func TestDatabase_GetLastProcessedBlockDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database test in short mode")
	}

	// Use a test database URL or skip if not available
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://chainpulse:password@localhost:5432/chainpulse_test?sslmode=disable"
	}

	db, err := NewDatabase(dsn)
	if err != nil {
		t.Skipf("skipping test: could not connect to database: %v", err)
	}

	// Before saving any block, it should return 0
	defaultBlock, err := db.GetLastProcessedBlock()
	if err != nil {
		t.Fatalf("Failed to get default last processed block: %v", err)
	}

	if defaultBlock.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Expected default block number 0, got %s", defaultBlock.String())
	}
}

func TestDatabase_GetEventsByBlockRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database test in short mode")
	}

	// Use a test database URL or skip if not available
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://chainpulse:password@localhost:5432/chainpulse_test?sslmode=disable"
	}

	db, err := NewDatabase(dsn)
	if err != nil {
		t.Skipf("skipping test: could not connect to database: %v", err)
	}

	// First, clean up any existing events for this test
	// In a real test environment, you might want to use a separate test schema

	// Create some test events
	events := []*types.IndexedEvent{
		{
			BlockNumber: big.NewInt(100),
			TxHash:      "0x1",
			EventName:   "TestEvent",
			Contract:    "0xContract1",
		},
		{
			BlockNumber: big.NewInt(150),
			TxHash:      "0x2",
			EventName:   "TestEvent",
			Contract:    "0xContract2",
		},
		{
			BlockNumber: big.NewInt(200),
			TxHash:      "0x3",
			EventName:   "TestEvent",
			Contract:    "0xContract3",
		},
		{
			BlockNumber: big.NewInt(250),
			TxHash:      "0x4",
			EventName:   "TestEvent",
			Contract:    "0xContract4",
		},
	}

	// Save the test events
	for _, event := range events {
		err := db.SaveEvent(event)
		if err != nil {
			t.Fatalf("Failed to save test event: %v", err)
		}
	}

	// Test getting events in a specific range
	fromBlock := big.NewInt(125)
	toBlock := big.NewInt(225)
	
	resultEvents, err := db.GetEventsByBlockRange(fromBlock, toBlock)
	if err != nil {
		t.Fatalf("Failed to get events by block range: %v", err)
	}

	// We expect 2 events (block 150 and 200)
	if len(resultEvents) != 2 {
		t.Errorf("Expected 2 events, got %d", len(resultEvents))
	}

	// Verify the correct events were returned
	expectedBlocks := []*big.Int{big.NewInt(150), big.NewInt(200)}
	for i, expectedBlock := range expectedBlocks {
		if resultEvents[i].BlockNumber.Cmp(expectedBlock) != 0 {
			t.Errorf("Expected event with block %s at index %d, got %s", 
				expectedBlock.String(), i, resultEvents[i].BlockNumber.String())
		}
	}
}