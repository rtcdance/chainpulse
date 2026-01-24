# E2E Testing Framework Architecture

## Overview

The E2E testing framework provides a comprehensive testing infrastructure for validating the entire ChainPulse Web3 Indexer pipeline. The architecture is modular, extensible, and designed for both local development and CI/CD environments.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    E2E Testing Framework                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         Test Orchestration Layer                        │  │
│  │  - Scenario Manager                                     │  │
│  │  - Test Lifecycle Management                            │  │
│  │  - Fixture Management                                   │  │
│  │  - Metrics Collection                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          │                                      │
│  ┌──────────────────────▼──────────────────────────────────┐  │
│  │         Component Managers                             │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │ Blockchain   │  │ Indexer      │  │ Database     │  │  │
│  │  │ Manager      │  │ Manager      │  │ Manager      │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │ API Manager  │  │ Cache Manager│  │ Event Manager│  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          │                                      │
│  ┌──────────────────────▼──────────────────────────────────┐  │
│  │         Test Utilities                                 │  │
│  │  - Error Injection                                     │  │
│  │  - Performance Measurement                             │  │
│  │  - Fixture Generation                                  │  │
│  │  - Validation Helpers                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                          │                                      │
│  ┌──────────────────────▼──────────────────────────────────┐  │
│  │         External Services                              │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │ Anvil        │  │ PostgreSQL   │  │ Redis        │  │  │
│  │  │ (Blockchain) │  │ (Metadata)   │  │ (Cache)      │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Test Orchestration Layer

Manages test lifecycle, scenario execution, and metrics collection.

**Responsibilities:**
- Initialize test environment
- Execute test scenarios
- Manage test fixtures
- Collect performance metrics
- Generate test reports

**Key Interfaces:**
```go
type Orchestrator interface {
    // Initialize sets up the test environment
    Initialize(ctx context.Context) error
    
    // ExecuteScenario runs a test scenario
    ExecuteScenario(ctx context.Context, scenario Scenario) (Result, error)
    
    // Cleanup tears down the test environment
    Cleanup(ctx context.Context) error
}

type Scenario interface {
    // Name returns the scenario name
    Name() string
    
    // Execute runs the scenario
    Execute(ctx context.Context, managers ComponentManagers) error
    
    // Validate checks scenario results
    Validate(ctx context.Context) error
}
```

### 2. Component Managers

Manage individual system components during testing.

**Blockchain Manager:**
- Manages Anvil instance
- Deploys test contracts
- Simulates blockchain events
- Handles chain reorganizations

**Indexer Manager:**
- Starts/stops indexer service
- Monitors indexer health
- Collects indexer metrics
- Validates indexing results

**Database Manager:**
- Manages PostgreSQL instance
- Handles schema migrations
- Provides data access
- Validates data consistency

**API Manager:**
- Manages API gateway
- Executes API queries
- Validates API responses
- Measures API performance

**Cache Manager:**
- Manages Redis instance
- Validates cache behavior
- Measures cache performance
- Handles cache invalidation

**Event Manager:**
- Generates test events
- Tracks event flow
- Validates event processing
- Measures event latency

### 3. Test Utilities

Provides helper functions and utilities for testing.

**Error Injection:**
- Simulate network failures
- Simulate database errors
- Simulate blockchain errors
- Simulate timeout scenarios

**Performance Measurement:**
- Measure event latency
- Measure throughput
- Measure query performance
- Measure resource usage

**Fixture Management:**
- Generate test data
- Create test scenarios
- Manage test state
- Clean up test data

**Validation Helpers:**
- Validate event data
- Validate query results
- Validate performance metrics
- Validate system state

## Test Scenarios

### Happy Path Scenario

Tests the normal operation flow:
1. Deploy test contract
2. Emit test events
3. Collect events
4. Index events
5. Query indexed data
6. Validate results

### Error Scenario

Tests error handling:
1. Simulate network failure
2. Verify error handling
3. Verify recovery
4. Validate data consistency

### Performance Scenario

Tests performance targets:
1. Generate high event volume
2. Measure collection latency
3. Measure processing throughput
4. Validate performance targets

### Multi-Chain Scenario

Tests multi-chain support:
1. Deploy contracts on multiple chains
2. Emit events on each chain
3. Collect events from all chains
4. Index events from all chains
5. Query cross-chain data
6. Validate results

### Concurrent Scenario

Tests concurrent processing:
1. Generate concurrent events
2. Process events concurrently
3. Measure concurrent throughput
4. Validate data consistency

## Data Flow

```
┌──────────────┐
│ Test Scenario│
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│ Blockchain Manager   │
│ - Deploy contracts   │
│ - Emit events        │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Indexer Manager      │
│ - Collect events     │
│ - Process events     │
│ - Index data         │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Database Manager     │
│ - Store indexed data │
│ - Validate data      │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ API Manager          │
│ - Query data         │
│ - Validate results   │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Metrics Collection   │
│ - Latency            │
│ - Throughput         │
│ - Resource usage     │
└──────────────────────┘
```

## Execution Flow

```
1. Setup Phase
   ├─ Initialize Anvil
   ├─ Initialize PostgreSQL
   ├─ Initialize Redis
   ├─ Start Indexer
   └─ Verify readiness

2. Execution Phase
   ├─ Deploy test contracts
   ├─ Emit test events
   ├─ Collect events
   ├─ Process events
   ├─ Index data
   └─ Collect metrics

3. Validation Phase
   ├─ Query indexed data
   ├─ Validate results
   ├─ Validate performance
   └─ Validate consistency

4. Cleanup Phase
   ├─ Stop Indexer
   ├─ Stop PostgreSQL
   ├─ Stop Redis
   ├─ Stop Anvil
   └─ Generate report
```

## Configuration

The framework is configured through environment variables and configuration files:

**Environment Variables:**
- `ANVIL_RPC_URL` - Anvil RPC endpoint
- `POSTGRES_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `INDEXER_RPC_URL` - Indexer RPC endpoint
- `API_URL` - API gateway URL

**Configuration Files:**
- `test/e2e/config.yaml` - Test configuration
- `test/e2e/scenarios.yaml` - Scenario definitions
- `test/e2e/fixtures.yaml` - Fixture definitions

## Performance Targets

The framework validates these performance targets:

| Metric | Target | Measurement |
|--------|--------|-------------|
| Event Collection Latency | < 2 seconds | Time from event emission to collection |
| Event Processing Throughput | > 1000 events/sec | Events processed per second |
| Query Response Time | < 500ms | Time to execute query |
| Code Coverage | ≥ 80% | Percentage of code covered by tests |
| Test Pass Rate | 100% | Percentage of tests passing |

## Extension Points

The framework provides extension points for custom testing:

1. **Custom Scenarios** - Implement `Scenario` interface
2. **Custom Managers** - Extend `ComponentManager` interface
3. **Custom Validators** - Implement validation logic
4. **Custom Metrics** - Add custom metric collection

## Related Documentation

- [Components Reference](./components.md)
- [Configuration Guide](./configuration.md)
- [Examples](./examples/)
- [Troubleshooting](./troubleshooting.md)
