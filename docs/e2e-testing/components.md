# E2E Testing Components Reference

## Overview

This document provides a comprehensive reference for all E2E testing framework components, their interfaces, and usage patterns.

## Component Managers

### Orchestrator

Manages the overall test lifecycle and scenario execution.

**Interface:**
```go
type Orchestrator interface {
    // Initialize sets up the test environment
    Initialize(ctx context.Context) error
    
    // ExecuteScenario runs a test scenario
    ExecuteScenario(ctx context.Context, scenario Scenario) (Result, error)
    
    // Cleanup tears down the test environment
    Cleanup(ctx context.Context) error
    
    // GetMetrics returns collected metrics
    GetMetrics() Metrics
}
```

**Usage:**
```go
orchestrator := e2e.NewOrchestrator(config)
defer orchestrator.Cleanup(ctx)

if err := orchestrator.Initialize(ctx); err != nil {
    log.Fatal(err)
}

result, err := orchestrator.ExecuteScenario(ctx, scenario)
if err != nil {
    log.Fatal(err)
}

metrics := orchestrator.GetMetrics()
```

### Blockchain Manager

Manages Anvil blockchain instance and contract deployment.

**Interface:**
```go
type BlockchainManager interface {
    // Start starts the Anvil instance
    Start(ctx context.Context) error
    
    // Stop stops the Anvil instance
    Stop(ctx context.Context) error
    
    // DeployContract deploys a contract
    DeployContract(ctx context.Context, contract Contract) (Address, error)
    
    // EmitEvent emits a blockchain event
    EmitEvent(ctx context.Context, event Event) (TxHash, error)
    
    // GetBalance returns account balance
    GetBalance(ctx context.Context, address Address) (BigInt, error)
    
    // GetBlockNumber returns current block number
    GetBlockNumber(ctx context.Context) (uint64, error)
}
```

**Usage:**
```go
blockchain := e2e.NewBlockchainManager(config)
if err := blockchain.Start(ctx); err != nil {
    log.Fatal(err)
}
defer blockchain.Stop(ctx)

address, err := blockchain.DeployContract(ctx, contract)
if err != nil {
    log.Fatal(err)
}

txHash, err := blockchain.EmitEvent(ctx, event)
if err != nil {
    log.Fatal(err)
}
```

### Indexer Manager

Manages the indexer service during testing.

**Interface:**
```go
type IndexerManager interface {
    // Start starts the indexer service
    Start(ctx context.Context) error
    
    // Stop stops the indexer service
    Stop(ctx context.Context) error
    
    // IsHealthy checks if indexer is healthy
    IsHealthy(ctx context.Context) (bool, error)
    
    // GetStatus returns indexer status
    GetStatus(ctx context.Context) (Status, error)
    
    // GetMetrics returns indexer metrics
    GetMetrics(ctx context.Context) (Metrics, error)
}
```

**Usage:**
```go
indexer := e2e.NewIndexerManager(config)
if err := indexer.Start(ctx); err != nil {
    log.Fatal(err)
}
defer indexer.Stop(ctx)

healthy, err := indexer.IsHealthy(ctx)
if !healthy {
    log.Fatal("Indexer not healthy")
}

status, err := indexer.GetStatus(ctx)
log.Printf("Indexer status: %+v", status)
```

### Database Manager

Manages PostgreSQL database for testing.

**Interface:**
```go
type DatabaseManager interface {
    // Start starts the database
    Start(ctx context.Context) error
    
    // Stop stops the database
    Stop(ctx context.Context) error
    
    // Migrate runs database migrations
    Migrate(ctx context.Context) error
    
    // Query executes a query
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    
    // Exec executes a statement
    Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
    
    // GetEventCount returns count of indexed events
    GetEventCount(ctx context.Context) (int64, error)
}
```

**Usage:**
```go
database := e2e.NewDatabaseManager(config)
if err := database.Start(ctx); err != nil {
    log.Fatal(err)
}
defer database.Stop(ctx)

if err := database.Migrate(ctx); err != nil {
    log.Fatal(err)
}

count, err := database.GetEventCount(ctx)
log.Printf("Event count: %d", count)
```

### API Manager

Manages API gateway for testing.

**Interface:**
```go
type APIManager interface {
    // Start starts the API gateway
    Start(ctx context.Context) error
    
    // Stop stops the API gateway
    Stop(ctx context.Context) error
    
    // QueryEvents queries indexed events
    QueryEvents(ctx context.Context, query Query) (Events, error)
    
    // GetEventByID retrieves event by ID
    GetEventByID(ctx context.Context, id string) (Event, error)
    
    // GetHealth returns API health status
    GetHealth(ctx context.Context) (HealthStatus, error)
}
```

**Usage:**
```go
api := e2e.NewAPIManager(config)
if err := api.Start(ctx); err != nil {
    log.Fatal(err)
}
defer api.Stop(ctx)

events, err := api.QueryEvents(ctx, query)
if err != nil {
    log.Fatal(err)
}

log.Printf("Found %d events", len(events))
```

### Cache Manager

Manages Redis cache for testing.

**Interface:**
```go
type CacheManager interface {
    // Start starts the cache
    Start(ctx context.Context) error
    
    // Stop stops the cache
    Stop(ctx context.Context) error
    
    // Get retrieves a value
    Get(ctx context.Context, key string) (string, error)
    
    // Set sets a value
    Set(ctx context.Context, key string, value string) error
    
    // Delete deletes a value
    Delete(ctx context.Context, key string) error
    
    // Flush clears all cache
    Flush(ctx context.Context) error
}
```

**Usage:**
```go
cache := e2e.NewCacheManager(config)
if err := cache.Start(ctx); err != nil {
    log.Fatal(err)
}
defer cache.Stop(ctx)

if err := cache.Set(ctx, "key", "value"); err != nil {
    log.Fatal(err)
}

value, err := cache.Get(ctx, "key")
log.Printf("Cache value: %s", value)
```

## Test Utilities

### Error Injection

Simulates various error conditions for testing error handling.

**Interface:**
```go
type ErrorInjector interface {
    // InjectNetworkError simulates network failure
    InjectNetworkError(ctx context.Context, duration time.Duration) error
    
    // InjectDatabaseError simulates database error
    InjectDatabaseError(ctx context.Context, duration time.Duration) error
    
    // InjectBlockchainError simulates blockchain error
    InjectBlockchainError(ctx context.Context, duration time.Duration) error
    
    // InjectTimeout simulates timeout
    InjectTimeout(ctx context.Context, duration time.Duration) error
}
```

**Usage:**
```go
injector := e2e.NewErrorInjector(config)

// Simulate network failure for 5 seconds
if err := injector.InjectNetworkError(ctx, 5*time.Second); err != nil {
    log.Fatal(err)
}

// Verify error handling
// ...
```

### Performance Measurement

Measures performance metrics during test execution.

**Interface:**
```go
type PerformanceMeasurer interface {
    // MeasureLatency measures operation latency
    MeasureLatency(ctx context.Context, operation func() error) (time.Duration, error)
    
    // MeasureThroughput measures operations per second
    MeasureThroughput(ctx context.Context, operation func() error, duration time.Duration) (float64, error)
    
    // MeasureResourceUsage measures CPU and memory
    MeasureResourceUsage(ctx context.Context) (ResourceUsage, error)
    
    // GetMetrics returns collected metrics
    GetMetrics() Metrics
}
```

**Usage:**
```go
measurer := e2e.NewPerformanceMeasurer()

latency, err := measurer.MeasureLatency(ctx, func() error {
    return indexer.ProcessEvent(ctx, event)
})
log.Printf("Latency: %v", latency)

throughput, err := measurer.MeasureThroughput(ctx, func() error {
    return indexer.ProcessEvent(ctx, event)
}, 10*time.Second)
log.Printf("Throughput: %.2f events/sec", throughput)
```

### Fixture Management

Manages test data and fixtures.

**Interface:**
```go
type FixtureManager interface {
    // GenerateEvents generates test events
    GenerateEvents(ctx context.Context, count int) (Events, error)
    
    // GenerateContract generates test contract
    GenerateContract(ctx context.Context) (Contract, error)
    
    // LoadFixture loads fixture from file
    LoadFixture(ctx context.Context, name string) (Fixture, error)
    
    // SaveFixture saves fixture to file
    SaveFixture(ctx context.Context, name string, fixture Fixture) error
}
```

**Usage:**
```go
fixtures := e2e.NewFixtureManager(config)

events, err := fixtures.GenerateEvents(ctx, 100)
if err != nil {
    log.Fatal(err)
}

contract, err := fixtures.GenerateContract(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Validation Helpers

Provides validation functions for test assertions.

**Interface:**
```go
type Validator interface {
    // ValidateEventData validates event data
    ValidateEventData(event Event) error
    
    // ValidateQueryResult validates query result
    ValidateQueryResult(result QueryResult) error
    
    // ValidatePerformance validates performance metrics
    ValidatePerformance(metrics Metrics) error
    
    // ValidateConsistency validates data consistency
    ValidateConsistency(ctx context.Context) error
}
```

**Usage:**
```go
validator := e2e.NewValidator(config)

if err := validator.ValidateEventData(event); err != nil {
    log.Fatal(err)
}

if err := validator.ValidatePerformance(metrics); err != nil {
    log.Fatal(err)
}
```

## Data Models

### Event

Represents a blockchain event.

```go
type Event struct {
    // Event ID
    ID string
    
    // Contract address
    ContractAddress string
    
    // Event signature
    EventSignature string
    
    // Event data
    Data map[string]interface{}
    
    // Block number
    BlockNumber uint64
    
    // Transaction hash
    TxHash string
    
    // Log index
    LogIndex uint
    
    // Timestamp
    Timestamp time.Time
}
```

### Metrics

Represents collected performance metrics.

```go
type Metrics struct {
    // Event collection latency
    CollectionLatency time.Duration
    
    // Event processing throughput
    ProcessingThroughput float64
    
    // Query response time
    QueryLatency time.Duration
    
    // Resource usage
    ResourceUsage ResourceUsage
    
    // Test duration
    Duration time.Duration
    
    // Timestamp
    Timestamp time.Time
}
```

### Result

Represents test execution result.

```go
type Result struct {
    // Test name
    Name string
    
    // Pass/fail status
    Passed bool
    
    // Error message
    Error string
    
    // Metrics
    Metrics Metrics
    
    // Duration
    Duration time.Duration
    
    // Timestamp
    Timestamp time.Time
}
```

## Related Documentation

- [Architecture Guide](./architecture.md)
- [Configuration Guide](./configuration.md)
- [Examples](./examples/)
- [API Reference](./api-reference.md)
