# E2E Testing API Reference

## Overview

This document provides a comprehensive reference for all public interfaces, data models, and functions in the E2E testing framework.

## Core Interfaces

### Orchestrator

Main interface for managing test lifecycle.

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
    
    // GetStatus returns current status
    GetStatus() Status
}
```

**Example:**
```go
orchestrator := e2e.NewOrchestrator(config)
defer orchestrator.Cleanup(ctx)

if err := orchestrator.Initialize(ctx); err != nil {
    return err
}

result, err := orchestrator.ExecuteScenario(ctx, scenario)
if err != nil {
    return err
}

metrics := orchestrator.GetMetrics()
```

### Scenario

Interface for test scenarios.

```go
type Scenario interface {
    // Name returns the scenario name
    Name() string
    
    // Description returns scenario description
    Description() string
    
    // Execute runs the scenario
    Execute(ctx context.Context, managers ComponentManagers) error
    
    // Validate checks scenario results
    Validate(ctx context.Context) error
    
    // Cleanup cleans up scenario resources
    Cleanup(ctx context.Context) error
}
```

**Example:**
```go
type CustomScenario struct {
    name string
}

func (s *CustomScenario) Name() string {
    return s.name
}

func (s *CustomScenario) Execute(ctx context.Context, managers ComponentManagers) error {
    // Implement scenario logic
    return nil
}

func (s *CustomScenario) Validate(ctx context.Context) error {
    // Implement validation logic
    return nil
}
```

### ComponentManagers

Container for all component managers.

```go
type ComponentManagers struct {
    // Blockchain manager
    Blockchain BlockchainManager
    
    // Indexer manager
    Indexer IndexerManager
    
    // Database manager
    Database DatabaseManager
    
    // API manager
    API APIManager
    
    // Cache manager
    Cache CacheManager
    
    // Event manager
    Event EventManager
}
```

## Component Managers

### BlockchainManager

Manages blockchain interactions.

```go
type BlockchainManager interface {
    // Start starts the blockchain
    Start(ctx context.Context) error
    
    // Stop stops the blockchain
    Stop(ctx context.Context) error
    
    // DeployContract deploys a contract
    DeployContract(ctx context.Context, contract Contract) (Address, error)
    
    // EmitEvent emits a blockchain event
    EmitEvent(ctx context.Context, event Event) (TxHash, error)
    
    // GetBalance returns account balance
    GetBalance(ctx context.Context, address Address) (BigInt, error)
    
    // GetBlockNumber returns current block number
    GetBlockNumber(ctx context.Context) (uint64, error)
    
    // GetLogs retrieves logs for a contract
    GetLogs(ctx context.Context, filter LogFilter) ([]Log, error)
}
```

**Example:**
```go
// Deploy contract
address, err := blockchain.DeployContract(ctx, contract)
if err != nil {
    return err
}

// Emit event
txHash, err := blockchain.EmitEvent(ctx, event)
if err != nil {
    return err
}

// Get logs
logs, err := blockchain.GetLogs(ctx, filter)
if err != nil {
    return err
}
```

### IndexerManager

Manages indexer service.

```go
type IndexerManager interface {
    // Start starts the indexer
    Start(ctx context.Context) error
    
    // Stop stops the indexer
    Stop(ctx context.Context) error
    
    // IsHealthy checks if indexer is healthy
    IsHealthy(ctx context.Context) (bool, error)
    
    // GetStatus returns indexer status
    GetStatus(ctx context.Context) (Status, error)
    
    // GetMetrics returns indexer metrics
    GetMetrics(ctx context.Context) (Metrics, error)
    
    // WaitForSync waits for indexer to sync
    WaitForSync(ctx context.Context, timeout time.Duration) error
}
```

**Example:**
```go
// Start indexer
if err := indexer.Start(ctx); err != nil {
    return err
}

// Wait for sync
if err := indexer.WaitForSync(ctx, 5*time.Minute); err != nil {
    return err
}

// Get status
status, err := indexer.GetStatus(ctx)
if err != nil {
    return err
}
```

### DatabaseManager

Manages database operations.

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
    
    // GetEvent retrieves event by ID
    GetEvent(ctx context.Context, id string) (Event, error)
}
```

**Example:**
```go
// Get event count
count, err := database.GetEventCount(ctx)
if err != nil {
    return err
}

// Get specific event
event, err := database.GetEvent(ctx, eventID)
if err != nil {
    return err
}

// Execute query
rows, err := database.Query(ctx, "SELECT * FROM events WHERE contract = $1", contractAddr)
if err != nil {
    return err
}
```

### APIManager

Manages API gateway.

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
    
    // ExecuteGraphQL executes GraphQL query
    ExecuteGraphQL(ctx context.Context, query string) (interface{}, error)
}
```

**Example:**
```go
// Query events
events, err := api.QueryEvents(ctx, query)
if err != nil {
    return err
}

// Get specific event
event, err := api.GetEventByID(ctx, eventID)
if err != nil {
    return err
}

// Execute GraphQL
result, err := api.ExecuteGraphQL(ctx, graphqlQuery)
if err != nil {
    return err
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

Represents performance metrics.

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

### Status

Represents component status.

```go
type Status struct {
    // Component name
    Name string
    
    // Health status
    Healthy bool
    
    // Last update time
    LastUpdate time.Time
    
    // Status message
    Message string
    
    // Additional details
    Details map[string]interface{}
}
```

### Query

Represents a database query.

```go
type Query struct {
    // Contract address filter
    ContractAddress string
    
    // Event signature filter
    EventSignature string
    
    // Block range
    FromBlock uint64
    ToBlock uint64
    
    // Limit results
    Limit int
    
    // Offset for pagination
    Offset int
}
```

## Utility Functions

### Error Injection

```go
// Inject network error
func InjectNetworkError(ctx context.Context, duration time.Duration) error

// Inject database error
func InjectDatabaseError(ctx context.Context, duration time.Duration) error

// Inject blockchain error
func InjectBlockchainError(ctx context.Context, duration time.Duration) error

// Inject timeout
func InjectTimeout(ctx context.Context, duration time.Duration) error
```

### Performance Measurement

```go
// Measure operation latency
func MeasureLatency(ctx context.Context, operation func() error) (time.Duration, error)

// Measure throughput
func MeasureThroughput(ctx context.Context, operation func() error, duration time.Duration) (float64, error)

// Measure resource usage
func MeasureResourceUsage(ctx context.Context) (ResourceUsage, error)
```

### Fixture Management

```go
// Generate test events
func GenerateEvents(ctx context.Context, count int) (Events, error)

// Generate test contract
func GenerateContract(ctx context.Context) (Contract, error)

// Load fixture from file
func LoadFixture(ctx context.Context, name string) (Fixture, error)

// Save fixture to file
func SaveFixture(ctx context.Context, name string, fixture Fixture) error
```

### Validation

```go
// Validate event data
func ValidateEventData(event Event) error

// Validate query result
func ValidateQueryResult(result QueryResult) error

// Validate performance metrics
func ValidatePerformance(metrics Metrics) error

// Validate data consistency
func ValidateConsistency(ctx context.Context) error
```

## Error Types

### Common Errors

```go
// Connection error
type ConnectionError struct {
    Service string
    Reason  string
}

// Timeout error
type TimeoutError struct {
    Operation string
    Duration  time.Duration
}

// Validation error
type ValidationError struct {
    Field   string
    Message string
}

// Performance error
type PerformanceError struct {
    Metric   string
    Expected float64
    Actual   float64
}
```

## Constants

### Performance Targets

```go
const (
    // Event collection latency target
    TargetCollectionLatency = 2 * time.Second
    
    // Event processing throughput target
    TargetProcessingThroughput = 1000.0 // events/sec
    
    // Query response time target
    TargetQueryLatency = 500 * time.Millisecond
    
    // Code coverage target
    TargetCodeCoverage = 0.80 // 80%
)
```

### Timeouts

```go
const (
    // Default operation timeout
    DefaultTimeout = 30 * time.Second
    
    // Default test timeout
    DefaultTestTimeout = 30 * time.Minute
    
    // Default sync timeout
    DefaultSyncTimeout = 5 * time.Minute
)
```

## Related Documentation

- [Architecture Guide](./architecture.md)
- [Components Reference](./components.md)
- [Configuration Guide](./configuration.md)
- [Examples](./examples/)
- [Troubleshooting](./troubleshooting.md)
- [FAQ](./faq.md)
