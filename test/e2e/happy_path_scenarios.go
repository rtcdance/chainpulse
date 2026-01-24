package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// HappyPathScenario represents a successful workflow scenario
type HappyPathScenario struct {
	name        string
	description string
	execute     func(ctx context.Context, orch *Orchestrator) error
}

// NewHappyPathScenarios returns all happy path scenarios
func NewHappyPathScenarios() []HappyPathScenario {
	return []HappyPathScenario{
		{
			name:        "BlockchainTransaction",
			description: "Test successful blockchain transaction execution",
			execute:     executeBlockchainTransaction,
		},
		{
			name:        "DatabaseInsertAndQuery",
			description: "Test successful database insert and query",
			execute:     executeDatabaseInsertAndQuery,
		},
		{
			name:        "APIEndpointCall",
			description: "Test successful API endpoint call",
			execute:     executeAPIEndpointCall,
		},
		{
			name:        "CompleteWorkflow",
			description: "Test complete workflow across all services",
			execute:     executeCompleteWorkflow,
		},
		{
			name:        "DataConsistency",
			description: "Test data consistency across blockchain and database",
			execute:     executeDataConsistency,
		},
	}
}

// executeBlockchainTransaction tests successful blockchain transaction
func executeBlockchainTransaction(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	// Get initial block number
	blockNum, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	if blockNum == 0 {
		return fmt.Errorf("invalid block number: %d", blockNum)
	}

	// Get account balance using proper address conversion
	testAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	balance, err := blockchain.GetBalance(ctx, testAddr)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if balance == nil {
		return fmt.Errorf("balance is nil")
	}

	return nil
}

// executeDatabaseInsertAndQuery tests successful database operations
func executeDatabaseInsertAndQuery(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	database := orch.GetDatabaseManager()
	if database == nil {
		return fmt.Errorf("database manager is nil")
	}

	// Create test table
	schema := `CREATE TABLE IF NOT EXISTS test_events (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Insert test data
	rowsAffected, err := database.ExecuteCommand(ctx,
		"INSERT INTO test_events (name) VALUES ($1)", "test_event")
	if err != nil {
		return fmt.Errorf("failed to insert data: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	// Query inserted data
	var count int
	err = database.ExecuteRow(ctx, "SELECT COUNT(*) FROM test_events").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query data: %w", err)
	}

	if count != 1 {
		return fmt.Errorf("expected 1 row, got %d", count)
	}

	// Cleanup
	if err := database.DropTable(ctx, "test_events"); err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	return nil
}

// executeAPIEndpointCall tests successful API endpoint call
func executeAPIEndpointCall(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	api := orch.GetAPIManager()
	if api == nil {
		return fmt.Errorf("API manager is nil")
	}

	// Check API health
	if !api.IsHealthy(ctx) {
		return fmt.Errorf("API is not healthy")
	}

	// Make GET request to health endpoint
	body, statusCode, err := api.GetRequest(ctx, "/health")
	if err != nil {
		return fmt.Errorf("failed to make GET request: %w", err)
	}

	if statusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", statusCode)
	}

	if len(body) == 0 {
		return fmt.Errorf("response body is empty")
	}

	return nil
}

// executeCompleteWorkflow tests complete workflow across all services
func executeCompleteWorkflow(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	database := orch.GetDatabaseManager()
	if database == nil {
		return fmt.Errorf("database manager is nil")
	}

	api := orch.GetAPIManager()
	if api == nil {
		return fmt.Errorf("API manager is nil")
	}

	// Step 1: Get blockchain state
	blockNum, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("step 1 failed - get block number: %w", err)
	}

	// Step 2: Create database table
	schema := `CREATE TABLE IF NOT EXISTS workflow_events (
		id SERIAL PRIMARY KEY,
		block_number BIGINT,
		event_data VARCHAR(255),
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("step 2 failed - create table: %w", err)
	}

	// Step 3: Insert blockchain state into database
	rowsAffected, err := database.ExecuteCommand(ctx,
		"INSERT INTO workflow_events (block_number, event_data) VALUES ($1, $2)",
		blockNum, "workflow_test")
	if err != nil {
		return fmt.Errorf("step 3 failed - insert data: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("step 3 failed - expected 1 row affected, got %d", rowsAffected)
	}

	// Step 4: Query data from database
	var storedBlockNum uint64
	err = database.ExecuteRow(ctx,
		"SELECT block_number FROM workflow_events WHERE event_data = $1",
		"workflow_test").Scan(&storedBlockNum)
	if err != nil {
		return fmt.Errorf("step 4 failed - query data: %w", err)
	}

	if storedBlockNum != blockNum {
		return fmt.Errorf("step 4 failed - block number mismatch: %d != %d", storedBlockNum, blockNum)
	}

	// Step 5: Call API to verify system is responsive
	if !api.IsHealthy(ctx) {
		return fmt.Errorf("step 5 failed - API not healthy")
	}

	// Step 6: Cleanup
	if err := database.DropTable(ctx, "workflow_events"); err != nil {
		return fmt.Errorf("step 6 failed - drop table: %w", err)
	}

	return nil
}

// executeDataConsistency tests data consistency across services
func executeDataConsistency(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	database := orch.GetDatabaseManager()
	if database == nil {
		return fmt.Errorf("database manager is nil")
	}

	// Get blockchain state
	blockNum1, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial block number: %w", err)
	}

	// Create table and insert blockchain state
	schema := `CREATE TABLE IF NOT EXISTS consistency_test (
		id SERIAL PRIMARY KEY,
		block_number BIGINT,
		timestamp TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	rowsAffected, err := database.ExecuteCommand(ctx,
		"INSERT INTO consistency_test (block_number) VALUES ($1)", blockNum1)
	if err != nil {
		return fmt.Errorf("failed to insert block number: %w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	// Wait a moment for potential state changes
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Get blockchain state again
	blockNum2, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get final block number: %w", err)
	}

	// Query database
	var storedBlockNum uint64
	err = database.ExecuteRow(ctx,
		"SELECT block_number FROM consistency_test ORDER BY id DESC LIMIT 1").Scan(&storedBlockNum)
	if err != nil {
		return fmt.Errorf("failed to query block number: %w", err)
	}

	if storedBlockNum != blockNum1 {
		return fmt.Errorf("data consistency check failed: stored=%d, initial=%d", storedBlockNum, blockNum1)
	}

	// Verify blockchain state is consistent
	if blockNum2 < blockNum1 {
		return fmt.Errorf("blockchain state went backwards: %d -> %d", blockNum1, blockNum2)
	}

	// Cleanup
	if err := database.DropTable(ctx, "consistency_test"); err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	return nil
}

// RunHappyPathScenario runs a single happy path scenario
func RunHappyPathScenario(ctx context.Context, orch *Orchestrator, scenario HappyPathScenario) error {
	return scenario.execute(ctx, orch)
}

// RunAllHappyPathScenarios runs all happy path scenarios
func RunAllHappyPathScenarios(ctx context.Context, orch *Orchestrator) map[string]error {
	scenarios := NewHappyPathScenarios()
	results := make(map[string]error)

	for _, scenario := range scenarios {
		results[scenario.name] = RunHappyPathScenario(ctx, orch, scenario)
	}

	return results
}
