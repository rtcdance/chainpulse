package e2e

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// MultiChainScenario represents a multi-chain testing scenario
type MultiChainScenario struct {
	name        string
	description string
	execute     func(ctx context.Context, orch *Orchestrator) error
}

// ChainState represents the state of a blockchain
type ChainState struct {
	ChainID    int
	BlockNum   int64
	Timestamp  time.Time
	DataHash   string
}

// NewMultiChainScenarios returns all multi-chain scenarios
func NewMultiChainScenarios() []MultiChainScenario {
	return []MultiChainScenario{
		{
			name:        "MultiChainStateSync",
			description: "Test state synchronization across multiple chains",
			execute:     executeMultiChainStateSync,
		},
		{
			name:        "CrossChainDataConsistency",
			description: "Test data consistency across chains",
			execute:     executeCrossChainDataConsistency,
		},
		{
			name:        "MultiChainConcurrentOperations",
			description: "Test concurrent operations on multiple chains",
			execute:     executeMultiChainConcurrentOperations,
		},
		{
			name:        "MultiChainFailover",
			description: "Test failover between chains",
			execute:     executeMultiChainFailover,
		},
		{
			name:        "MultiChainDataAggregation",
			description: "Test data aggregation from multiple chains",
			execute:     executeMultiChainDataAggregation,
		},
	}
}

// executeMultiChainStateSync tests state synchronization across chains
func executeMultiChainStateSync(ctx context.Context, orch *Orchestrator) error {
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

	// Create table for chain states
	schema := `CREATE TABLE IF NOT EXISTS chain_states (
		id SERIAL PRIMARY KEY,
		chain_id INT,
		block_number BIGINT,
		timestamp TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Get state from primary chain
	blockNum, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	// Store state for chain 1
	_, err = database.ExecuteCommand(ctx,
		"INSERT INTO chain_states (chain_id, block_number) VALUES ($1, $2)",
		1, blockNum)
	if err != nil {
		return fmt.Errorf("failed to store chain state: %w", err)
	}

	// Simulate state from chain 2 (would be from different blockchain in real scenario)
	_, err = database.ExecuteCommand(ctx,
		"INSERT INTO chain_states (chain_id, block_number) VALUES ($1, $2)",
		2, blockNum+1)
	if err != nil {
		return fmt.Errorf("failed to store chain 2 state: %w", err)
	}

	// Verify both states are stored
	var count int
	err = database.ExecuteRow(ctx, "SELECT COUNT(*) FROM chain_states").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count chain states: %w", err)
	}

	if count != 2 {
		return fmt.Errorf("expected 2 chain states, got %d", count)
	}

	// Cleanup
	_ = database.DropTable(ctx, "chain_states")

	return nil
}

// executeCrossChainDataConsistency tests data consistency across chains
func executeCrossChainDataConsistency(ctx context.Context, orch *Orchestrator) error {
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

	// Create table for cross-chain data
	schema := `CREATE TABLE IF NOT EXISTS cross_chain_data (
		id SERIAL PRIMARY KEY,
		source_chain INT,
		dest_chain INT,
		data_hash VARCHAR(255),
		verified BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Get blockchain state
	blockNum, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	// Create data hash from blockchain state
	dataHash := fmt.Sprintf("hash_%d", blockNum)

	// Insert cross-chain data
	_, err = database.ExecuteCommand(ctx,
		"INSERT INTO cross_chain_data (source_chain, dest_chain, data_hash, verified) VALUES ($1, $2, $3, $4)",
		1, 2, dataHash, true)
	if err != nil {
		return fmt.Errorf("failed to insert cross-chain data: %w", err)
	}

	// Verify data consistency
	var verified bool
	var storedHash string
	err = database.ExecuteRow(ctx,
		"SELECT verified, data_hash FROM cross_chain_data WHERE source_chain = $1 AND dest_chain = $2",
		1, 2).Scan(&verified, &storedHash)
	if err != nil {
		return fmt.Errorf("failed to query cross-chain data: %w", err)
	}

	if !verified || storedHash != dataHash {
		return fmt.Errorf("cross-chain data consistency check failed: verified=%v, hash=%s", verified, storedHash)
	}

	// Cleanup
	_ = database.DropTable(ctx, "cross_chain_data")

	return nil
}

// executeMultiChainConcurrentOperations tests concurrent operations on multiple chains
func executeMultiChainConcurrentOperations(ctx context.Context, orch *Orchestrator) error {
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

	// Create table for concurrent operations
	schema := `CREATE TABLE IF NOT EXISTS concurrent_chain_ops (
		id SERIAL PRIMARY KEY,
		chain_id INT,
		operation_id INT,
		status VARCHAR(50),
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	const numChains = 3
	const opsPerChain = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int
	var errorCount int

	// Simulate concurrent operations on multiple chains
	for chainID := 1; chainID <= numChains; chainID++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			for opID := 0; opID < opsPerChain; opID++ {
				// Get blockchain state
				_, err := blockchain.GetBlockNumber(ctx)
				if err != nil {
					mu.Lock()
					errorCount++
					mu.Unlock()
					continue
				}

				// Record operation in database
				_, err = database.ExecuteCommand(ctx,
					"INSERT INTO concurrent_chain_ops (chain_id, operation_id, status) VALUES ($1, $2, $3)",
					cid, opID, "completed")
				mu.Lock()
				if err == nil {
					successCount++
				} else {
					errorCount++
				}
				mu.Unlock()
			}
		}(chainID)
	}

	wg.Wait()

	totalOps := numChains * opsPerChain
	if successCount < totalOps*90/100 { // Allow 10% failure rate
		return fmt.Errorf("concurrent chain operations failed: %d/%d successful",
			successCount, totalOps)
	}

	// Cleanup
	_ = database.DropTable(ctx, "concurrent_chain_ops")

	return nil
}

// executeMultiChainFailover tests failover between chains
func executeMultiChainFailover(ctx context.Context, orch *Orchestrator) error {
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

	// Create table for failover tracking
	schema := `CREATE TABLE IF NOT EXISTS failover_tracking (
		id SERIAL PRIMARY KEY,
		primary_chain INT,
		backup_chain INT,
		failover_reason VARCHAR(255),
		timestamp TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Try to get state from primary chain
	_, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		// Primary chain failed, record failover
		_, err = database.ExecuteCommand(ctx,
			"INSERT INTO failover_tracking (primary_chain, backup_chain, failover_reason) VALUES ($1, $2, $3)",
			1, 2, "primary_unavailable")
		if err != nil {
			return fmt.Errorf("failed to record failover: %w", err)
		}
	} else {
		// Primary chain is healthy, record successful operation
		_, err = database.ExecuteCommand(ctx,
			"INSERT INTO failover_tracking (primary_chain, backup_chain, failover_reason) VALUES ($1, $2, $3)",
			1, 2, "primary_healthy")
		if err != nil {
			return fmt.Errorf("failed to record primary health: %w", err)
		}
	}

	// Verify failover tracking
	var count int
	err = database.ExecuteRow(ctx, "SELECT COUNT(*) FROM failover_tracking").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count failover records: %w", err)
	}

	if count != 1 {
		return fmt.Errorf("expected 1 failover record, got %d", count)
	}

	// Cleanup
	_ = database.DropTable(ctx, "failover_tracking")

	return nil
}

// executeMultiChainDataAggregation tests data aggregation from multiple chains
func executeMultiChainDataAggregation(ctx context.Context, orch *Orchestrator) error {
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

	// Create table for aggregated data
	schema := `CREATE TABLE IF NOT EXISTS aggregated_chain_data (
		id SERIAL PRIMARY KEY,
		chain_id INT,
		block_number BIGINT,
		event_count INT,
		aggregated_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Get data from primary chain
	blockNum, err := blockchain.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %w", err)
	}

	// Aggregate data from multiple chains
	const numChains = 3
	for chainID := 1; chainID <= numChains; chainID++ {
		eventCount := chainID * 10 // Simulate different event counts per chain
		if blockNum > math.MaxUint64-uint64(chainID) {
			return fmt.Errorf("aggregated block number overflow for chain %d", chainID)
		}

		_, err = database.ExecuteCommand(ctx,
			"INSERT INTO aggregated_chain_data (chain_id, block_number, event_count) VALUES ($1, $2, $3)",
			chainID, blockNum+uint64(chainID), eventCount)
		if err != nil {
			return fmt.Errorf("failed to insert aggregated data: %w", err)
		}
	}

	// Verify aggregation
	var totalEvents int
	err = database.ExecuteRow(ctx,
		"SELECT SUM(event_count) FROM aggregated_chain_data").Scan(&totalEvents)
	if err != nil {
		return fmt.Errorf("failed to sum aggregated data: %w", err)
	}

	expectedTotal := 10 + 20 + 30 // 1*10 + 2*10 + 3*10
	if totalEvents != expectedTotal {
		return fmt.Errorf("aggregation mismatch: expected %d, got %d", expectedTotal, totalEvents)
	}

	// Cleanup
	_ = database.DropTable(ctx, "aggregated_chain_data")

	return nil
}

// RunMultiChainScenario runs a single multi-chain scenario
func RunMultiChainScenario(ctx context.Context, orch *Orchestrator, scenario MultiChainScenario) error {
	return scenario.execute(ctx, orch)
}

// RunAllMultiChainScenarios runs all multi-chain scenarios
func RunAllMultiChainScenarios(ctx context.Context, orch *Orchestrator) map[string]error {
	scenarios := NewMultiChainScenarios()
	results := make(map[string]error)

	for _, scenario := range scenarios {
		results[scenario.name] = RunMultiChainScenario(ctx, orch, scenario)
	}

	return results
}
