# Design: Web3 Indexer E2E Testing Framework

## Overview

The E2E testing framework provides a comprehensive testing solution for the ChainPulse blockchain event indexer. It combines Anvil (deterministic Ethereum test node), Hardhat (smart contract deployment), and Go testing libraries (testify, gopter) to validate the complete event indexing pipeline.

The framework is organized into three layers:
1. **Blockchain Layer**: Anvil-based test environment with smart contracts
2. **Indexer Layer**: ChainPulse indexer components (puller, processor, storage)
3. **Validation Layer**: Test assertions and property-based tests

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    E2E Test Suite                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Test Orchestrator                           │  │
│  │  - Manages test lifecycle                           │  │
│  │  - Coordinates Anvil + Indexer                      │  │
│  │  - Collects metrics and logs                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                  │
│         ┌────────────────┼────────────────┐                │
│         │                │                │                │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌─────▼──────┐         │
│  │  Blockchain │  │   Indexer   │  │ Validation │         │
│  │   Manager   │  │   Manager   │  │  Manager   │         │
│  └─────────────┘  └─────────────┘  └────────────┘         │
│         │                │                │                │
│         │                │                │                │
│  ┌──────▼──────────────────────────────────▼──────┐        │
│  │         Test Fixtures & Utilities              │        │
│  │  - Smart contracts (ERC20, ERC721)             │        │
│  │  - Event generators                           │        │
│  │  - State snapshots                            │        │
│  │  - Assertion helpers                          │        │
│  └───────────────────────────────────────────────┘        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
    ┌─────────┐          ┌─────────┐         ┌──────────┐
    │  Anvil  │          │ Indexer │         │ Database │
    │ (Port   │          │ (Port   │         │ (Port    │
    │ 8545)   │          │ 8080)   │         │ 5432)    │
    └─────────┘          └─────────┘         └──────────┘
```

## Components and Interfaces

### 1. Test Orchestrator

Manages the complete test lifecycle and coordinates all components.

```go
type TestOrchestrator interface {
    // Setup initializes all test infrastructure
    Setup(ctx context.Context) error
    
    // Teardown cleans up resources
    Teardown(ctx context.Context) error
    
    // GetBlockchainManager returns the blockchain manager
    GetBlockchainManager() BlockchainManager
    
    // GetIndexerManager returns the indexer manager
    GetIndexerManager() IndexerManager
    
    // GetValidationManager returns the validation manager
    GetValidationManager() ValidationManager
    
    // CollectMetrics returns collected test metrics
    CollectMetrics() TestMetrics
}
```

### 2. Blockchain Manager

Manages Anvil instance and smart contract deployment.

```go
type BlockchainManager interface {
    // StartAnvil starts the Anvil test node
    StartAnvil(ctx context.Context) error
    
    // StopAnvil stops the Anvil instance
    StopAnvil(ctx context.Context) error
    
    // DeployContract deploys a smart contract
    DeployContract(ctx context.Context, contract ContractDefinition) (*DeployedContract, error)
    
    // EmitEvent triggers an event emission
    EmitEvent(ctx context.Context, contractAddr string, eventName string, params map[string]interface{}) (*EventEmission, error)
    
    // GetBlockNumber returns current block number
    GetBlockNumber(ctx context.Context) (uint64, error)
    
    // CreateSnapshot creates a blockchain state snapshot
    CreateSnapshot(ctx context.Context) (string, error)
    
    // RestoreSnapshot restores blockchain to a previous state
    RestoreSnapshot(ctx context.Context, snapshotID string) error
}
```

### 3. Indexer Manager

Manages ChainPulse indexer components.

```go
type IndexerManager interface {
    // StartIndexer starts the indexer service
    StartIndexer(ctx context.Context, config IndexerConfig) error
    
    // StopIndexer stops the indexer
    StopIndexer(ctx context.Context) error
    
    // WaitForIndexing waits for events to be indexed
    WaitForIndexing(ctx context.Context, expectedCount int, timeout time.Duration) error
    
    // GetIndexedEvents returns indexed events
    GetIndexedEvents(ctx context.Context, filter EventFilter) ([]*IndexedEvent, error)
    
    // GetIndexerMetrics returns indexer performance metrics
    GetIndexerMetrics(ctx context.Context) IndexerMetrics
}
```

### 4. Validation Manager

Validates test results and assertions.

```go
type ValidationManager interface {
    // ValidateEventCollection validates that all events were collected
    ValidateEventCollection(ctx context.Context, emitted []*EventEmission, indexed []*IndexedEvent) error
    
    // ValidateEventDecoding validates event decoding accuracy
    ValidateEventDecoding(ctx context.Context, event *IndexedEvent) error
    
    // ValidateEventOrdering validates event ordering
    ValidateEventOrdering(ctx context.Context, events []*IndexedEvent) error
    
    // ValidateAPIResponse validates API query response
    ValidateAPIResponse(ctx context.Context, response *APIResponse, expectedEvents []*IndexedEvent) error
    
    // ValidatePerformance validates performance metrics
    ValidatePerformance(ctx context.Context, metrics PerformanceMetrics) error
}
```

### 5. Test Fixtures

Pre-configured test data and utilities.

```go
type TestFixtures struct {
    // ERC20 contract definition
    ERC20Contract ContractDefinition
    
    // ERC721 contract definition
    ERC721Contract ContractDefinition
    
    // Test accounts with pre-funded balances
    TestAccounts []TestAccount
    
    // Event generators for creating test scenarios
    EventGenerators map[string]EventGenerator
    
    // Assertion helpers
    Assertions AssertionHelpers
}
```

## Data Models

### EventEmission

Represents an event emitted on the blockchain.

```go
type EventEmission struct {
    // Unique identifier
    ID string
    
    // Contract address
    ContractAddress string
    
    // Event name
    EventName string
    
    // Transaction hash
    TxHash string
    
    // Block number
    BlockNumber uint64
    
    // Log index
    LogIndex uint32
    
    // Event parameters
    Parameters map[string]interface{}
    
    // Emission timestamp
    Timestamp time.Time
}
```

### IndexedEvent

Represents an event after indexing.

```go
type IndexedEvent struct {
    // Unique identifier
    ID string
    
    // Contract address
    ContractAddress string
    
    // Event name
    EventName string
    
    // Transaction hash
    TxHash string
    
    // Block number
    BlockNumber uint64
    
    // Log index
    LogIndex uint32
    
    // Decoded event data
    DecodedData map[string]interface{}
    
    // Indexing timestamp
    IndexedAt time.Time
    
    // Chain identifier
    ChainID string
}
```

### TestMetrics

Collected metrics during test execution.

```go
type TestMetrics struct {
    // Event collection latency (milliseconds)
    CollectionLatency []int64
    
    // Event processing latency (milliseconds)
    ProcessingLatency []int64
    
    // API query latency (milliseconds)
    QueryLatency []int64
    
    // Events per second throughput
    Throughput float64
    
    // Error count
    ErrorCount int
    
    // Memory usage (bytes)
    MemoryUsage int64
    
    // CPU usage (percentage)
    CPUUsage float64
}
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Event Collection Completeness

**For any** set of events emitted on the blockchain, all events SHALL be collected by the data puller without loss or duplication.

**Validates: Requirements 3.1, 3.2, 3.4**

### Property 2: Event Ordering Preservation

**For any** sequence of events emitted in order, the indexed events SHALL maintain the same ordering (by block number and log index).

**Validates: Requirements 3.4, 4.4, 5.3**

### Property 3: Event Decoding Accuracy

**For any** blockchain event, the decoded event data SHALL match the original event parameters according to the contract ABI.

**Validates: Requirements 4.1, 4.2**

### Property 4: Duplicate Prevention

**For any** event, storing it multiple times SHALL result in only one copy in the database (idempotency).

**Validates: Requirements 4.3, 5.2**

### Property 5: API Query Consistency

**For any** query filter, the API response SHALL contain exactly the events matching the filter criteria, in correct order.

**Validates: Requirements 6.1, 6.3**

### Property 6: Multi-Chain Isolation

**For any** events from different chains, the indexer SHALL maintain separate collections and correctly tag events with chain identifiers.

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 7: Concurrent Processing Safety

**For any** set of concurrent events, the indexer SHALL process all events without race conditions or data corruption.

**Validates: Requirements 8.1, 8.3**

### Property 8: Latency Bounds

**For any** event emission, the end-to-end latency from emission to API availability SHALL be less than 2 seconds.

**Validates: Requirements 9.1**

### Property 9: Throughput Minimum

**For any** batch of events, the indexer SHALL maintain throughput of at least 1000 events per second.

**Validates: Requirements 9.2**

### Property 10: Error Recovery

**For any** transient error, the indexer SHALL implement exponential backoff retry and eventually recover.

**Validates: Requirements 10.1, 10.2**

## Error Handling

### Error Classification

1. **Transient Errors**: Network timeouts, temporary database unavailability
   - Recovery: Exponential backoff retry (max 5 attempts)
   - Timeout: 30 seconds

2. **Permanent Errors**: Invalid contract ABI, malformed events
   - Recovery: Log error and skip event
   - Action: Alert operator

3. **Critical Errors**: Database corruption, indexer crash
   - Recovery: Checkpoint and restart
   - Action: Immediate alert

### Error Handling Strategy

- Implement circuit breaker for blockchain connections
- Queue events during database unavailability
- Log all errors with context for debugging
- Implement graceful degradation

## Testing Strategy

### Test Organization

```
test/e2e/
├── orchestrator_test.go          # Test orchestrator tests
├── blockchain_manager_test.go    # Blockchain manager tests
├── indexer_manager_test.go       # Indexer manager tests
├── validation_manager_test.go    # Validation manager tests
├── fixtures/
│   ├── contracts/                # Smart contract definitions
│   ├── events/                   # Event generators
│   └── data/                     # Test data
└── scenarios/
    ├── happy_path_test.go        # Normal operation
    ├── error_handling_test.go    # Error scenarios
    ├── concurrent_test.go        # Concurrent processing
    ├── performance_test.go       # Performance validation
    └── multi_chain_test.go       # Multi-chain scenarios
```

### Test Execution

1. **Setup Phase**: Initialize Anvil, deploy contracts, start indexer
2. **Execution Phase**: Emit events, collect metrics
3. **Validation Phase**: Verify results against properties
4. **Teardown Phase**: Clean up resources

### Property-Based Testing

Use gopter for property-based tests:
- Generate random event sequences
- Validate properties hold for all generated inputs
- Minimum 100 iterations per property test

### Performance Testing

- Measure end-to-end latency
- Measure throughput (events/second)
- Measure memory and CPU usage
- Validate against performance requirements

## Implementation Notes

### Dependencies

- `github.com/ethereum/go-ethereum`: Ethereum client library
- `github.com/stretchr/testify`: Assertion library
- `github.com/leanovate/gopter`: Property-based testing
- `testcontainers-go`: Docker-based test infrastructure
- `github.com/go-redis/redis`: Redis client
- `github.com/lib/pq`: PostgreSQL driver

### Configuration

Test configuration via environment variables:
- `E2E_ANVIL_PORT`: Anvil port (default: 8545)
- `E2E_INDEXER_PORT`: Indexer API port (default: 8080)
- `E2E_DATABASE_URL`: Database connection string
- `E2E_TIMEOUT`: Test timeout (default: 5 minutes)
- `E2E_PARALLEL`: Number of parallel tests (default: 1)

### Logging

- Structured JSON logging for all components
- Log levels: DEBUG, INFO, WARN, ERROR
- Correlation IDs for tracing requests
- Metrics exported to Prometheus format

## Deployment

### Local Development

```bash
# Run all E2E tests
go test ./test/e2e/...

# Run specific test
go test ./test/e2e/ -run TestEventCollection

# Run with verbose output
go test -v ./test/e2e/...
```

### CI/CD Integration

- Run E2E tests on every commit
- Generate coverage reports
- Collect performance metrics
- Archive test logs and artifacts

## Success Criteria

- All E2E tests pass consistently
- Event collection latency < 2 seconds
- Event processing throughput > 1000 events/second
- API query latency < 500ms
- Zero event loss or duplication
- Proper error handling and recovery
- Memory usage < 500MB for 100,000 events
- All properties validated with 100+ iterations
