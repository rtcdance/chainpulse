# Web3 + Go E2E Testing: Industry Best Practices

## Executive Summary

This document outlines industry-standard practices for end-to-end testing of blockchain indexers using Go and Web3 tools. The approach combines deterministic blockchain simulation (Anvil), smart contract testing (Hardhat), and property-based testing (gopter) to achieve comprehensive test coverage.

## 1. Blockchain Testing Infrastructure

### 1.1 Anvil: Deterministic Test Node

**Why Anvil?**
- Deterministic block production (no randomness)
- Fast block time (1 second)
- Forking support for mainnet testing
- Snapshot/restore for test isolation
- No external dependencies

**Setup Pattern:**
```go
// Start Anvil with deterministic configuration
anvil := NewAnvilInstance(&AnvilConfig{
    Port:           8545,
    BlockTime:      1,
    Deterministic:  true,
    ChainID:        31337,
    Accounts:       10,
    Balance:        "1000000000000000000", // 1 ETH
})

err := anvil.Start(ctx)
defer anvil.Stop(ctx)
```

**Best Practices:**
- Use fixed block time for predictable test timing
- Pre-fund test accounts with sufficient balance
- Use deterministic account generation
- Implement snapshot/restore for test isolation
- Clean up resources in defer statements

### 1.2 Smart Contract Deployment

**Standard Contracts for Testing:**

1. **ERC20 Token Contract**
   - Transfer events for basic testing
   - Mint/Burn for supply changes
   - Approval events for allowance testing

2. **ERC721 NFT Contract**
   - Mint/Burn events
   - Transfer events
   - Approval events

3. **Custom Domain Contracts**
   - Application-specific events
   - Complex event parameters
   - Multi-indexed events

**Deployment Pattern:**
```go
// Deploy contract with ABI capture
contract, err := blockchain.DeployContract(ctx, &ContractDefinition{
    Name:     "TestToken",
    Bytecode: erc20Bytecode,
    ABI:      erc20ABI,
    Constructor: []interface{}{
        "Test Token",
        "TEST",
        big.NewInt(1000000),
    },
})

// Store ABI for event decoding
abiRegistry.Register(contract.Address, erc20ABI)
```

**Best Practices:**
- Store contract ABIs for event decoding
- Use deterministic deployment addresses
- Implement contract factory pattern
- Validate deployment success
- Capture deployment gas usage

## 2. Event Generation and Emission

### 2.1 Event Generation Patterns

**Property-Based Event Generation:**
```go
// Generate random event sequences
gen := gopter.CombineGens(
    gen.UInt64Range(1, 1000),           // Block number
    gen.SliceOfN(10, gen.UInt32()),     // Log indices
    gen.MapOf(gen.Const("from"), gen.AlphaString()),  // Parameters
)

properties.Property("event_ordering", gen.Property(func(events []*Event) bool {
    // Validate property holds for generated events
    return validateEventOrdering(events)
}))
```

**Event Emission Pattern:**
```go
// Emit events with captured metadata
emission, err := blockchain.EmitEvent(ctx, &EventEmission{
    ContractAddress: tokenAddr,
    EventName:       "Transfer",
    Parameters: map[string]interface{}{
        "from":   senderAddr,
        "to":     recipientAddr,
        "amount": big.NewInt(1000),
    },
})

// Capture emission metadata for validation
emissions = append(emissions, &EmissionRecord{
    Emission:  emission,
    Timestamp: time.Now(),
    BlockNum:  emission.BlockNumber,
})
```

**Best Practices:**
- Use property-based generation for comprehensive coverage
- Capture emission metadata for validation
- Implement event ordering guarantees
- Handle multi-indexed events correctly
- Validate event parameters before emission

### 2.2 Event Scenarios

**Scenario 1: Normal Operation**
- Emit events in sequence
- Validate collection and indexing
- Verify API query results

**Scenario 2: Concurrent Events**
- Emit events from multiple goroutines
- Validate no race conditions
- Verify ordering within blocks

**Scenario 3: Blockchain Reorg**
- Emit events on main chain
- Trigger reorg (fork)
- Validate reorg detection and recovery

**Scenario 4: Large Batch**
- Emit 1000+ events
- Validate throughput
- Monitor memory usage

## 3. Data Puller Integration

### 3.1 Event Collection Validation

**Collection Pattern:**
```go
// Start data puller
puller := NewDataPuller(indexerConfig)
err := puller.Start(ctx)
defer puller.Stop(ctx)

// Emit events
emissions := emitTestEvents(ctx, blockchain, 100)

// Wait for collection
err = puller.WaitForEvents(ctx, len(emissions), 5*time.Second)

// Validate collection
collected, err := indexer.QueryEvents(ctx, &EventFilter{})
assert.Equal(t, len(emissions), len(collected))
```

**Best Practices:**
- Implement timeout for collection waiting
- Validate event count matches emission count
- Check for duplicates
- Verify event data integrity
- Monitor collection latency

### 3.2 Retry Logic Testing

**Retry Pattern:**
```go
// Simulate transient errors
blockchain.SimulateError(ctx, &ErrorSimulation{
    Type:     "network_timeout",
    Duration: 2 * time.Second,
    Retries:  3,
})

// Verify retry logic
err := puller.CollectEvents(ctx)
assert.NoError(t, err)

// Validate retry metrics
metrics := puller.GetMetrics()
assert.Equal(t, 3, metrics.RetryCount)
assert.True(t, metrics.EventuallySucceeded)
```

**Best Practices:**
- Test exponential backoff
- Validate maximum retry attempts
- Verify eventual success
- Monitor retry latency
- Test circuit breaker patterns

## 4. Event Processing Pipeline

### 4.1 Event Decoding

**Decoding Pattern:**
```go
// Decode event using ABI
decoder := NewEventDecoder(abiRegistry)
decoded, err := decoder.Decode(ctx, &RawEvent{
    Topics: event.Topics,
    Data:   event.Data,
})

// Validate decoded data
assert.Equal(t, expectedFrom, decoded.Parameters["from"])
assert.Equal(t, expectedTo, decoded.Parameters["to"])
assert.Equal(t, expectedAmount, decoded.Parameters["amount"])
```

**Best Practices:**
- Use contract ABI for accurate decoding
- Validate parameter types
- Handle indexed vs non-indexed parameters
- Test edge cases (large numbers, special characters)
- Implement error handling for malformed events

### 4.2 Duplicate Detection

**Idempotency Pattern:**
```go
// Store event
err := processor.ProcessEvent(ctx, event)
assert.NoError(t, err)

// Store same event again
err = processor.ProcessEvent(ctx, event)
assert.NoError(t, err)

// Verify only one copy stored
events, err := database.QueryEvents(ctx, &EventFilter{
    TxHash:   event.TxHash,
    LogIndex: event.LogIndex,
})
assert.Equal(t, 1, len(events))
```

**Best Practices:**
- Use (TxHash, LogIndex) as unique key
- Implement idempotent storage
- Test duplicate detection
- Validate no data loss
- Monitor duplicate rate

## 5. Database Persistence

### 5.1 Multi-Database Support

**Database Abstraction Pattern:**
```go
// Support multiple databases
type EventStore interface {
    StoreEvent(ctx context.Context, event *Event) error
    QueryEvents(ctx context.Context, filter *EventFilter) ([]*Event, error)
    Close(ctx context.Context) error
}

// PostgreSQL implementation
pgStore := NewPostgresEventStore(pgConfig)

// MongoDB implementation
mongoStore := NewMongoEventStore(mongoConfig)
```

**Best Practices:**
- Use interface-based abstraction
- Test with multiple databases
- Implement connection pooling
- Handle database-specific features
- Validate data consistency

### 5.2 Testcontainers Integration

**Container Pattern:**
```go
// Start PostgreSQL container
container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: testcontainers.ContainerRequest{
        Image:        "postgres:15",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_PASSWORD": "password",
        },
    },
    Started: true,
})
defer container.Terminate(ctx)

// Get connection string
host, _ := container.Host(ctx)
port, _ := container.MappedPort(ctx, "5432")
connStr := fmt.Sprintf("postgres://user:password@%s:%s/testdb", host, port)
```

**Best Practices:**
- Use testcontainers for database testing
- Implement automatic cleanup
- Support multiple database versions
- Test connection pooling
- Validate data persistence

## 6. API Query Validation

### 6.1 REST API Testing

**API Test Pattern:**
```go
// Query events via REST API
resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/v1/events?contract=%s&limit=10", contractAddr))
assert.Equal(t, http.StatusOK, resp.StatusCode)

// Parse response
var events []*APIEvent
json.NewDecoder(resp.Body).Decode(&events)

// Validate response
assert.Equal(t, 10, len(events))
assert.Equal(t, contractAddr, events[0].ContractAddress)
```

**Best Practices:**
- Test all query parameters
- Validate response format
- Check HTTP status codes
- Test error responses
- Validate pagination

### 6.2 Query Filter Validation

**Filter Pattern:**
```go
// Test various filters
filters := []EventFilter{
    {ContractAddress: contractAddr},
    {EventName: "Transfer"},
    {BlockRange: &BlockRange{Start: 100, End: 200}},
    {Limit: 50, Offset: 100},
}

for _, filter := range filters {
    events, err := api.QueryEvents(ctx, filter)
    assert.NoError(t, err)
    validateFilterResults(t, events, filter)
}
```

**Best Practices:**
- Test all filter combinations
- Validate filter logic
- Test pagination
- Check sorting order
- Validate result accuracy

## 7. Multi-Chain Testing

### 7.1 Chain Isolation

**Multi-Chain Pattern:**
```go
// Start multiple Anvil instances
chains := make(map[string]*AnvilInstance)
for _, chainID := range []string{"ethereum", "polygon", "arbitrum"} {
    chains[chainID] = NewAnvilInstance(&AnvilConfig{
        Port:    8545 + len(chains),
        ChainID: chainIDMap[chainID],
    })
    chains[chainID].Start(ctx)
}

// Deploy contracts on each chain
for chainID, anvil := range chains {
    contract, _ := anvil.DeployContract(ctx, erc20Def)
    contractMap[chainID] = contract
}

// Emit events on each chain
for chainID, anvil := range chains {
    anvil.EmitEvent(ctx, &EventEmission{
        ContractAddress: contractMap[chainID].Address,
        EventName:       "Transfer",
        Parameters:      params,
    })
}

// Validate chain isolation
events := indexer.QueryEvents(ctx, &EventFilter{ChainID: "ethereum"})
assert.True(t, allEventsFromChain(events, "ethereum"))
```

**Best Practices:**
- Use separate Anvil instances per chain
- Tag events with chain identifier
- Validate chain isolation
- Test cross-chain consistency
- Monitor per-chain metrics

## 8. Concurrent Processing

### 8.1 Race Condition Detection

**Concurrency Pattern:**
```go
// Emit events concurrently
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        blockchain.EmitEvent(ctx, &EventEmission{
            ContractAddress: contractAddr,
            EventName:       "Transfer",
            Parameters: map[string]interface{}{
                "from":   senders[id%len(senders)],
                "to":     recipients[id%len(recipients)],
                "amount": big.NewInt(int64(id)),
            },
        })
    }(i)
}
wg.Wait()

// Validate no race conditions
events, _ := indexer.QueryEvents(ctx, &EventFilter{})
assert.Equal(t, 100, len(events))
validateNoCorruption(t, events)
```

**Best Practices:**
- Use goroutines for concurrent emission
- Implement race detector
- Validate data consistency
- Test with high concurrency
- Monitor for deadlocks

## 9. Performance Testing

### 9.1 Latency Measurement

**Latency Pattern:**
```go
// Measure end-to-end latency
start := time.Now()
emission, _ := blockchain.EmitEvent(ctx, eventDef)
indexed, _ := indexer.WaitForEvent(ctx, emission.TxHash, 5*time.Second)
latency := time.Since(start)

// Validate latency bounds
assert.Less(t, latency, 2*time.Second)

// Collect latency metrics
metrics.RecordLatency(latency)
```

**Best Practices:**
- Measure end-to-end latency
- Collect latency percentiles (p50, p95, p99)
- Validate against SLOs
- Monitor latency trends
- Alert on latency violations

### 9.2 Throughput Measurement

**Throughput Pattern:**
```go
// Measure throughput
start := time.Now()
for i := 0; i < 1000; i++ {
    blockchain.EmitEvent(ctx, eventDef)
}
duration := time.Since(start)

// Calculate throughput
throughput := 1000.0 / duration.Seconds()

// Validate throughput
assert.Greater(t, throughput, 1000.0) // 1000 events/second
```

**Best Practices:**
- Measure events per second
- Test with various batch sizes
- Monitor throughput trends
- Validate against requirements
- Test under load

## 10. Error Handling and Recovery

### 10.1 Error Injection

**Error Injection Pattern:**
```go
// Inject transient error
blockchain.SimulateError(ctx, &ErrorSimulation{
    Type:     "network_timeout",
    Duration: 2 * time.Second,
})

// Verify recovery
err := puller.CollectEvents(ctx)
assert.NoError(t, err)

// Validate retry metrics
metrics := puller.GetMetrics()
assert.Greater(t, metrics.RetryCount, 0)
```

**Best Practices:**
- Implement error injection framework
- Test transient errors
- Test permanent errors
- Validate recovery mechanisms
- Monitor error rates

### 10.2 Graceful Degradation

**Degradation Pattern:**
```go
// Simulate database unavailability
database.SimulateUnavailability(ctx, 5*time.Second)

// Verify graceful degradation
err := indexer.ProcessEvents(ctx, events)
assert.NoError(t, err)

// Validate events queued
queued := indexer.GetQueuedEvents()
assert.Equal(t, len(events), len(queued))

// Verify recovery after database available
time.Sleep(6 * time.Second)
err = indexer.ProcessQueuedEvents(ctx)
assert.NoError(t, err)
```

**Best Practices:**
- Implement event queuing
- Test graceful degradation
- Validate eventual consistency
- Monitor queue depth
- Alert on queue overflow

## 11. Test Organization and Execution

### 11.1 Test Structure

```
test/e2e/
├── orchestrator/
│   ├── orchestrator.go           # Test orchestrator
│   ├── orchestrator_test.go      # Orchestrator tests
│   └── orchestrator_property_test.go  # Property tests
├── blockchain/
│   ├── manager.go                # Blockchain manager
│   ├── manager_test.go           # Manager tests
│   └── fixtures.go               # Test fixtures
├── indexer/
│   ├── manager.go                # Indexer manager
│   ├── manager_test.go           # Manager tests
│   └── scenarios.go              # Test scenarios
├── validation/
│   ├── validator.go              # Validation logic
│   ├── validator_test.go         # Validator tests
│   └── assertions.go             # Assertion helpers
└── scenarios/
    ├── happy_path_test.go        # Happy path tests
    ├── error_handling_test.go    # Error scenario tests
    ├── concurrent_test.go        # Concurrent tests
    ├── performance_test.go       # Performance tests
    └── multi_chain_test.go       # Multi-chain tests
```

### 11.2 Test Execution

**Sequential Execution:**
```bash
# Run all E2E tests
go test ./test/e2e/...

# Run specific test
go test ./test/e2e/ -run TestEventCollection

# Run with verbose output
go test -v ./test/e2e/...

# Run with coverage
go test -cover ./test/e2e/...
```

**Parallel Execution:**
```bash
# Run tests in parallel
go test -parallel 4 ./test/e2e/...

# Run with race detector
go test -race ./test/e2e/...
```

**Best Practices:**
- Use table-driven tests
- Implement test helpers
- Use subtests for organization
- Implement cleanup in defer
- Use t.Parallel() for parallel tests

## 12. CI/CD Integration

### 12.1 GitHub Actions Workflow

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      
      - name: Run E2E Tests
        run: go test -v -race ./test/e2e/...
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
      
      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test-results/
```

**Best Practices:**
- Run tests on every commit
- Generate coverage reports
- Archive test artifacts
- Implement test result reporting
- Alert on test failures

## 13. Monitoring and Observability

### 13.1 Metrics Collection

**Metrics Pattern:**
```go
// Collect metrics
metrics := &TestMetrics{
    CollectionLatency: []int64{},
    ProcessingLatency: []int64{},
    QueryLatency:      []int64{},
}

// Record latencies
start := time.Now()
// ... operation ...
metrics.CollectionLatency = append(metrics.CollectionLatency, time.Since(start).Milliseconds())

// Export metrics
report := metrics.GenerateReport()
fmt.Println(report)
```

**Best Practices:**
- Collect latency percentiles
- Track throughput
- Monitor error rates
- Collect resource usage
- Generate reports

## 14. Troubleshooting Guide

### Common Issues

1. **Anvil Connection Timeout**
   - Verify Anvil is running
   - Check port availability
   - Increase timeout

2. **Event Not Collected**
   - Verify contract deployment
   - Check event emission
   - Validate ABI registration

3. **Database Connection Error**
   - Verify database is running
   - Check connection string
   - Validate credentials

4. **Race Conditions**
   - Run with race detector
   - Increase concurrency
   - Check synchronization

## Conclusion

This framework provides a comprehensive approach to E2E testing for blockchain indexers. By combining deterministic blockchain simulation, property-based testing, and industry-standard tools, you can achieve high confidence in your indexer's correctness and performance.

Key takeaways:
- Use Anvil for deterministic blockchain testing
- Implement property-based tests for comprehensive coverage
- Test with multiple databases and configurations
- Monitor performance and latency
- Implement proper error handling and recovery
- Integrate with CI/CD for continuous validation
