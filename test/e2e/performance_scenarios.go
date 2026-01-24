package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PerformanceScenario represents a performance testing scenario
type PerformanceScenario struct {
	name        string
	description string
	execute     func(ctx context.Context, orch *Orchestrator) error
}

// PerformanceMetrics holds performance measurement data
type PerformanceMetrics struct {
	OperationName       string
	TotalOperations     int
	SuccessfulOps       int
	FailedOps           int
	TotalDuration       time.Duration
	AverageDuration     time.Duration
	MinDuration         time.Duration
	MaxDuration         time.Duration
	ThroughputOpsPerSec float64
	EndToEndLatency     time.Duration
	Throughput          float64
	MemoryUsage         int64
	CPUUsage            float64
}

// NewPerformanceScenarios returns all performance scenarios
func NewPerformanceScenarios() []PerformanceScenario {
	return []PerformanceScenario{
		{
			name:        "BlockchainReadThroughput",
			description: "Test blockchain read operation throughput",
			execute:     executeBlockchainReadThroughput,
		},
		{
			name:        "DatabaseWriteThroughput",
			description: "Test database write operation throughput",
			execute:     executeDatabaseWriteThroughput,
		},
		{
			name:        "ConcurrentBlockchainReads",
			description: "Test concurrent blockchain read operations",
			execute:     executeConcurrentBlockchainReads,
		},
		{
			name:        "ConcurrentDatabaseWrites",
			description: "Test concurrent database write operations",
			execute:     executeConcurrentDatabaseWrites,
		},
		{
			name:        "EndToEndLatency",
			description: "Test end-to-end operation latency",
			execute:     executeEndToEndLatency,
		},
	}
}

// executeBlockchainReadThroughput tests blockchain read throughput
func executeBlockchainReadThroughput(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	const numOperations = 100
	start := time.Now()
	var successCount int

	for i := 0; i < numOperations; i++ {
		_, err := blockchain.GetBlockNumber(ctx)
		if err == nil {
			successCount++
		}
	}

	duration := time.Since(start)
	throughput := float64(successCount) / duration.Seconds()

	if successCount < numOperations*90/100 { // Allow 10% failure rate
		return fmt.Errorf("blockchain read throughput too low: %d/%d successful, throughput: %.2f ops/sec",
			successCount, numOperations, throughput)
	}

	return nil
}

// executeDatabaseWriteThroughput tests database write throughput
func executeDatabaseWriteThroughput(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	database := orch.GetDatabaseManager()
	if database == nil {
		return fmt.Errorf("database manager is nil")
	}

	// Create test table
	schema := `CREATE TABLE IF NOT EXISTS perf_test (
		id SERIAL PRIMARY KEY,
		data VARCHAR(255),
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	const numOperations = 100
	start := time.Now()
	var successCount int

	for i := 0; i < numOperations; i++ {
		_, err := database.ExecuteCommand(ctx,
			"INSERT INTO perf_test (data) VALUES ($1)", fmt.Sprintf("data_%d", i))
		if err == nil {
			successCount++
		}
	}

	duration := time.Since(start)
	throughput := float64(successCount) / duration.Seconds()

	// Cleanup
	_ = database.DropTable(ctx, "perf_test")

	if successCount < numOperations*90/100 { // Allow 10% failure rate
		return fmt.Errorf("database write throughput too low: %d/%d successful, throughput: %.2f ops/sec",
			successCount, numOperations, throughput)
	}

	return nil
}

// executeConcurrentBlockchainReads tests concurrent blockchain reads
func executeConcurrentBlockchainReads(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	blockchain := orch.GetBlockchainManager()
	if blockchain == nil {
		return fmt.Errorf("blockchain manager is nil")
	}

	const numGoroutines = 10
	const opsPerGoroutine = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int
	var errorCount int

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				_, err := blockchain.GetBlockNumber(ctx)
				mu.Lock()
				if err == nil {
					successCount++
				} else {
					errorCount++
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	totalOps := numGoroutines * opsPerGoroutine
	throughput := float64(successCount) / duration.Seconds()

	if successCount < totalOps*90/100 { // Allow 10% failure rate
		return fmt.Errorf("concurrent blockchain reads too slow: %d/%d successful, throughput: %.2f ops/sec",
			successCount, totalOps, throughput)
	}

	return nil
}

// executeConcurrentDatabaseWrites tests concurrent database writes
func executeConcurrentDatabaseWrites(ctx context.Context, orch *Orchestrator) error {
	if orch == nil {
		return fmt.Errorf("orchestrator is nil")
	}

	database := orch.GetDatabaseManager()
	if database == nil {
		return fmt.Errorf("database manager is nil")
	}

	// Create test table
	schema := `CREATE TABLE IF NOT EXISTS concurrent_perf_test (
		id SERIAL PRIMARY KEY,
		goroutine_id INT,
		operation_id INT,
		data VARCHAR(255),
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	const numGoroutines = 10
	const opsPerGoroutine = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int
	var errorCount int

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				_, err := database.ExecuteCommand(ctx,
					"INSERT INTO concurrent_perf_test (goroutine_id, operation_id, data) VALUES ($1, $2, $3)",
					goroutineID, i, fmt.Sprintf("data_%d_%d", goroutineID, i))
				mu.Lock()
				if err == nil {
					successCount++
				} else {
					errorCount++
				}
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOps := numGoroutines * opsPerGoroutine
	throughput := float64(successCount) / duration.Seconds()

	// Cleanup
	_ = database.DropTable(ctx, "concurrent_perf_test")

	if successCount < totalOps*90/100 { // Allow 10% failure rate
		return fmt.Errorf("concurrent database writes too slow: %d/%d successful, throughput: %.2f ops/sec",
			successCount, totalOps, throughput)
	}

	return nil
}

// executeEndToEndLatency tests end-to-end operation latency
func executeEndToEndLatency(ctx context.Context, orch *Orchestrator) error {
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

	// Create test table
	schema := `CREATE TABLE IF NOT EXISTS latency_test (
		id SERIAL PRIMARY KEY,
		block_number BIGINT,
		created_at TIMESTAMP DEFAULT NOW()
	)`
	if err := database.CreateTable(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	const numOperations = 50
	var durations []time.Duration
	var successCount int

	for i := 0; i < numOperations; i++ {
		start := time.Now()

		// Get blockchain state
		blockNum, err := blockchain.GetBlockNumber(ctx)
		if err != nil {
			continue
		}

		// Insert into database
		_, err = database.ExecuteCommand(ctx,
			"INSERT INTO latency_test (block_number) VALUES ($1)", blockNum)
		if err != nil {
			continue
		}

		duration := time.Since(start)
		durations = append(durations, duration)
		successCount++
	}

	// Cleanup
	_ = database.DropTable(ctx, "latency_test")

	if successCount < numOperations*90/100 { // Allow 10% failure rate
		return fmt.Errorf("end-to-end latency test failed: %d/%d successful",
			successCount, numOperations)
	}

	// Calculate average latency
	var totalDuration time.Duration
	for _, d := range durations {
		totalDuration += d
	}
	avgLatency := totalDuration / time.Duration(len(durations))

	// Latency should be reasonable (< 1 second per operation)
	if avgLatency > 1*time.Second {
		return fmt.Errorf("end-to-end latency too high: %.2f ms average",
			float64(avgLatency.Milliseconds()))
	}

	return nil
}

// RunPerformanceScenario runs a single performance scenario
func RunPerformanceScenario(ctx context.Context, orch *Orchestrator, scenario PerformanceScenario) error {
	return scenario.execute(ctx, orch)
}

// RunAllPerformanceScenarios runs all performance scenarios
func RunAllPerformanceScenarios(ctx context.Context, orch *Orchestrator) map[string]error {
	scenarios := NewPerformanceScenarios()
	results := make(map[string]error)

	for _, scenario := range scenarios {
		results[scenario.name] = RunPerformanceScenario(ctx, orch, scenario)
	}

	return results
}
