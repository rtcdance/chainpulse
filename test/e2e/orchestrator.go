package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// placeholderDatabaseURL is the historical DSN baked into the test
	// orchestrator. It never connects; its only job is to keep the
	// "database unavailable → skip" path intact when TEST_DATABASE_URL is
	// not set (decision D16(a)).
	placeholderDatabaseURL = "postgres://user:password@localhost:5432/test"

	// scenarioDatabaseURL is the historical DSN used by scenario
	// orchestrators (docker-compose default credentials).
	scenarioDatabaseURL = "postgres://postgres:postgres@localhost:5432/chainpulse_test"
)

// testDatabaseURL returns TEST_DATABASE_URL from the environment when set,
// otherwise fallback (decision D16(a)): when a real CI/service container
// provisions the variable its database is used, while an absent variable
// keeps the historical placeholder so skip behavior is unchanged.
func testDatabaseURL(fallback string) string {
	if url := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); url != "" {
		return url
	}
	return fallback
}

// Orchestrator coordinates all E2E test managers
type Orchestrator struct {
	blockchain  *BlockchainManager
	database    *DatabaseManager
	api         *APIManager
	initialized bool
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(blockchainURL, databaseConnString, apiBaseURL string) *Orchestrator {
	return &Orchestrator{
		blockchain: NewBlockchainManager(blockchainURL),
		database:   NewDatabaseManager(databaseConnString),
		api:        NewAPIManager(apiBaseURL),
	}
}

// NewTestOrchestrator creates a new orchestrator with default test configuration
func NewTestOrchestrator() *Orchestrator {
	return &Orchestrator{
		blockchain: NewBlockchainManager("http://localhost:8545"),
		database:   NewDatabaseManager(testDatabaseURL(placeholderDatabaseURL)),
		api:        NewAPIManager("http://localhost:8080"),
	}
}

// Setup initializes the orchestrator for testing
func (o *Orchestrator) Setup(ctx context.Context) error {
	return o.Initialize(ctx)
}

// Teardown closes the orchestrator after testing
func (o *Orchestrator) Teardown(ctx context.Context) error {
	return o.Close()
}

// Initialize initializes all managers
func (o *Orchestrator) Initialize(ctx context.Context) error {
	if o.initialized {
		return fmt.Errorf("orchestrator already initialized")
	}

	// Initialize blockchain manager
	if err := o.blockchain.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize blockchain manager: %w", err)
	}

	// Initialize database manager
	if err := o.database.Initialize(ctx); err != nil {
		_ = o.blockchain.Close()
		return fmt.Errorf("failed to initialize database manager: %w", err)
	}

	// Initialize API manager
	if err := o.api.Initialize(ctx); err != nil {
		_ = o.blockchain.Close()
		_ = o.database.Close()
		return fmt.Errorf("failed to initialize API manager: %w", err)
	}

	o.initialized = true
	return nil
}

// Close closes all managers
func (o *Orchestrator) Close() error {
	if !o.initialized {
		return nil
	}

	var errs []error

	if err := o.blockchain.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close blockchain manager: %w", err))
	}

	if err := o.database.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close database manager: %w", err))
	}

	if err := o.api.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close API manager: %w", err))
	}

	o.initialized = false

	if len(errs) > 0 {
		return fmt.Errorf("errors closing orchestrator: %v", errs)
	}

	return nil
}

// GetBlockchainManager returns the blockchain manager
func (o *Orchestrator) GetBlockchainManager() *BlockchainManager {
	return o.blockchain
}

// GetDatabaseManager returns the database manager
func (o *Orchestrator) GetDatabaseManager() *DatabaseManager {
	return o.database
}

// GetAPIManager returns the API manager
func (o *Orchestrator) GetAPIManager() *APIManager {
	return o.api
}

// WaitForAllServices waits for all services to be ready
func (o *Orchestrator) WaitForAllServices(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for blockchain
	if err := o.blockchain.WaitForConnection(ctx, timeout); err != nil {
		return fmt.Errorf("blockchain not ready: %w", err)
	}

	// Wait for database
	if err := o.database.WaitForConnection(ctx, timeout); err != nil {
		return fmt.Errorf("database not ready: %w", err)
	}

	// Wait for API
	if err := o.api.WaitForHealthy(ctx, timeout); err != nil {
		return fmt.Errorf("API not ready: %w", err)
	}

	return nil
}

// IsReady checks if all services are ready
func (o *Orchestrator) IsReady(ctx context.Context) bool {
	if !o.initialized {
		return false
	}

	return o.blockchain.IsConnected(ctx) &&
		o.database.IsConnected(ctx) &&
		o.api.IsHealthy(ctx)
}

// Reset resets all services to a clean state
func (o *Orchestrator) Reset(ctx context.Context, tables []string) error {
	if !o.initialized {
		return fmt.Errorf("orchestrator not initialized")
	}

	// Clear database tables
	if err := o.database.ClearAllTables(ctx, tables); err != nil {
		return fmt.Errorf("failed to clear database tables: %w", err)
	}

	return nil
}

// GetStatus returns the status of all services
func (o *Orchestrator) GetStatus(ctx context.Context) map[string]bool {
	status := make(map[string]bool)

	if !o.initialized {
		status["initialized"] = false
		return status
	}

	status["initialized"] = true
	status["blockchain"] = o.blockchain.IsConnected(ctx)
	status["database"] = o.database.IsConnected(ctx)
	status["api"] = o.api.IsHealthy(ctx)

	return status
}
