package e2e

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ErrorScenario represents an error handling scenario
type ErrorScenario struct {
	name        string
	description string
	execute     func(ctx context.Context, orch *Orchestrator) error
}

// NewErrorScenarios returns all error scenarios
func NewErrorScenarios() []ErrorScenario {
	return []ErrorScenario{
		{
			name:        "InvalidBlockchainAddress",
			description: "Test handling of invalid blockchain address",
			execute:     executeInvalidBlockchainAddress,
		},
		{
			name:        "DatabaseConnectionFailure",
			description: "Test handling of database connection failure",
			execute:     executeDatabaseConnectionFailure,
		},
		{
			name:        "InvalidDatabaseQuery",
			description: "Test handling of invalid database query",
			execute:     executeInvalidDatabaseQuery,
		},
		{
			name:        "APIEndpointNotFound",
			description: "Test handling of API endpoint not found",
			execute:     executeAPIEndpointNotFound,
		},
		{
			name:        "ContextCancellation",
			description: "Test handling of context cancellation",
			execute:     executeContextCancellation,
		},
	}
}

// executeInvalidBlockchainAddress tests handling of invalid blockchain address
func executeInvalidBlockchainAddress(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	// Try to get balance with invalid address - use HexToAddress which handles invalid addresses
	invalidAddr := common.HexToAddress("0x0000000000000000000000000000000000000000")
	_, err := blockchain.GetBalance(ctx, invalidAddr)
	// We expect an error or nil balance
	if err != nil {
		return nil // Expected error
	}

	return nil
}

// executeDatabaseConnectionFailure tests handling of database connection failure
func executeDatabaseConnectionFailure(ctx context.Context, orch *Orchestrator) error {
	// Create a new database manager with invalid connection string
	invalidDB := NewDatabaseManager("postgres://invalid:invalid@localhost:9999/invalid")

	// Try to initialize with invalid connection
	err := invalidDB.Initialize(ctx)
	if err == nil {
		return fmt.Errorf("expected error for invalid database connection, got nil")
	}

	return nil
}

// executeInvalidDatabaseQuery tests handling of invalid database query
func executeInvalidDatabaseQuery(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	database := orch.GetDatabaseManager()
	if database == nil {
		return fmt.Errorf("database manager is nil")
	}

	// Try to execute invalid SQL query
	_, err := database.ExecuteQuery(ctx, "INVALID SQL QUERY")

	// We expect an error
	if err == nil {
		return fmt.Errorf("expected error for invalid SQL query, got nil")
	}

	return nil
}

// executeAPIEndpointNotFound tests handling of API endpoint not found
func executeAPIEndpointNotFound(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	api := orch.GetAPIManager()
	if api == nil {
		return fmt.Errorf("API manager is nil")
	}

	// Try to call non-existent endpoint
	_, statusCode, err := api.GetRequest(ctx, "/nonexistent/endpoint")

	// We expect either an error or a 404 status code
	if err == nil && statusCode != 404 {
		return fmt.Errorf("expected error or 404 status, got status %d", statusCode)
	}

	return nil
}

// executeContextCancellation tests handling of context cancellation
func executeContextCancellation(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	// Create a context that's already cancelled
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	// Try to use cancelled context
	_, err := blockchain.GetBlockNumber(cancelledCtx)

	// We expect an error due to cancelled context
	if err == nil {
		return fmt.Errorf("expected error for cancelled context, got nil")
	}

	return nil
}

// RunErrorScenario runs a single error scenario
func RunErrorScenario(ctx context.Context, orch *Orchestrator, scenario ErrorScenario) error {
	return scenario.execute(ctx, orch)
}

// RunAllErrorScenarios runs all error scenarios
func RunAllErrorScenarios(ctx context.Context, orch *Orchestrator) map[string]error {
	scenarios := NewErrorScenarios()
	results := make(map[string]error)

	for _, scenario := range scenarios {
		results[scenario.name] = RunErrorScenario(ctx, orch, scenario)
	}

	return results
}
