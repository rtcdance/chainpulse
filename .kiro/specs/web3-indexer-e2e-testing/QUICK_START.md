# Quick Start: Web3 Indexer E2E Testing

## 5-Minute Setup

### Prerequisites
```bash
# Install Go 1.25+
go version

# Install Anvil (Foundry)
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Verify installation
anvil --version
```

### Project Structure
```
test/e2e/
├── orchestrator.go          # Test orchestrator
├── blockchain_manager.go    # Blockchain management
├── indexer_manager.go       # Indexer management
├── validation_manager.go    # Validation logic
├── fixtures.go              # Test fixtures
└── scenarios_test.go        # Test scenarios
```

## Running Tests

### All E2E Tests
```bash
cd test/e2e
go test -v ./...
```

### Specific Test
```bash
go test -v -run TestEventCollection
```

### With Race Detector
```bash
go test -race ./...
```

### With Coverage
```bash
go test -cover ./...
```

## Key Components

### 1. Test Orchestrator
Manages complete test lifecycle.

```go
orchestrator := NewTestOrchestrator()
err := orchestrator.Setup(ctx)
defer orchestrator.Teardown(ctx)

// Use orchestrator
blockchain := orchestrator.GetBlockchainManager()
indexer := orchestrator.GetIndexerManager()
validator := orchestrator.GetValidationManager()
```

### 2. Blockchain Manager
Manages Anvil and smart contracts.

```go
// Start Anvil
err := blockchain.StartAnvil(ctx)

// Deploy contract
contract, err := blockchain.DeployContract(ctx, erc20Definition)

// Emit event
emission, err := blockchain.EmitEvent(ctx, &EventEmission{
    ContractAddress: contract.Address,
    EventName:       "Transfer",
    Parameters:      params,
})
```

### 3. Indexer Manager
Manages indexer components.

```go
// Start indexer
err := indexer.StartIndexer(ctx, config)

// Wait for indexing
err := indexer.WaitForIndexing(ctx, expectedCount, timeout)

// Query events
events, err := indexer.GetIndexedEvents(ctx, filter)
```

### 4. Validation Manager
Validates test results.

```go
// Validate event collection
err := validator.ValidateEventCollection(ctx, emitted, indexed)

// Validate event decoding
err := validator.ValidateEventDecoding(ctx, event)

// Validate API response
err := validator.ValidateAPIResponse(ctx, response, expected)
```

## Common Test Patterns

### Pattern 1: Happy Path
```go
func TestEventCollection(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewTestOrchestrator()
    orchestrator.Setup(ctx)
    defer orchestrator.Teardown(ctx)
    
    blockchain := orchestrator.GetBlockchainManager()
    indexer := orchestrator.GetIndexerManager()
    validator := orchestrator.GetValidationManager()
    
    // Deploy contract
    contract, _ := blockchain.DeployContract(ctx, erc20Definition)
    
    // Emit events
    emissions := make([]*EventEmission, 10)
    for i := 0; i < 10; i++ {
        emission, _ := blockchain.EmitEvent(ctx, &EventEmission{
            ContractAddress: contract.Address,
            EventName:       "Transfer",
            Parameters:      params,
        })
        emissions[i] = emission
    }
    
    // Wait for indexing
    indexer.WaitForIndexing(ctx, 10, 5*time.Second)
    
    // Query events
    indexed, _ := indexer.GetIndexedEvents(ctx, &EventFilter{})
    
    // Validate
    validator.ValidateEventCollection(ctx, emissions, indexed)
}
```

### Pattern 2: Error Handling
```go
func TestErrorRecovery(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewTestOrchestrator()
    orchestrator.Setup(ctx)
    defer orchestrator.Teardown(ctx)
    
    blockchain := orchestrator.GetBlockchainManager()
    indexer := orchestrator.GetIndexerManager()
    
    // Simulate error
    blockchain.SimulateError(ctx, &ErrorSimulation{
        Type:     "network_timeout",
        Duration: 2 * time.Second,
    })
    
    // Emit events (should retry)
    contract, _ := blockchain.DeployContract(ctx, erc20Definition)
    blockchain.EmitEvent(ctx, &EventEmission{
        ContractAddress: contract.Address,
        EventName:       "Transfer",
        Parameters:      params,
    })
    
    // Verify recovery
    indexer.WaitForIndexing(ctx, 1, 10*time.Second)
    events, _ := indexer.GetIndexedEvents(ctx, &EventFilter{})
    assert.Equal(t, 1, len(events))
}
```

### Pattern 3: Concurrent Processing
```go
func TestConcurrentProcessing(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewTestOrchestrator()
    orchestrator.Setup(ctx)
    defer orchestrator.Teardown(ctx)
    
    blockchain := orchestrator.GetBlockchainManager()
    indexer := orchestrator.GetIndexerManager()
    
    contract, _ := blockchain.DeployContract(ctx, erc20Definition)
    
    // Emit events concurrently
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            blockchain.EmitEvent(ctx, &EventEmission{
                ContractAddress: contract.Address,
                EventName:       "Transfer",
                Parameters:      params,
            })
        }(i)
    }
    wg.Wait()
    
    // Verify all events indexed
    indexer.WaitForIndexing(ctx, 100, 10*time.Second)
    events, _ := indexer.GetIndexedEvents(ctx, &EventFilter{})
    assert.Equal(t, 100, len(events))
}
```

### Pattern 4: Performance Testing
```go
func TestPerformance(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewTestOrchestrator()
    orchestrator.Setup(ctx)
    defer orchestrator.Teardown(ctx)
    
    blockchain := orchestrator.GetBlockchainManager()
    indexer := orchestrator.GetIndexerManager()
    
    contract, _ := blockchain.DeployContract(ctx, erc20Definition)
    
    // Measure latency
    start := time.Now()
    emission, _ := blockchain.EmitEvent(ctx, &EventEmission{
        ContractAddress: contract.Address,
        EventName:       "Transfer",
        Parameters:      params,
    })
    indexed, _ := indexer.WaitForEvent(ctx, emission.TxHash, 5*time.Second)
    latency := time.Since(start)
    
    // Validate latency
    assert.Less(t, latency, 2*time.Second)
    
    // Measure throughput
    start = time.Now()
    for i := 0; i < 1000; i++ {
        blockchain.EmitEvent(ctx, &EventEmission{
            ContractAddress: contract.Address,
            EventName:       "Transfer",
            Parameters:      params,
        })
    }
    duration := time.Since(start)
    throughput := 1000.0 / duration.Seconds()
    
    // Validate throughput
    assert.Greater(t, throughput, 1000.0)
}
```

## Property-Based Testing

### Using gopter
```go
func TestEventOrderingProperty(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    // Generate random event sequences
    gen := gopter.CombineGens(
        gen.UInt64Range(1, 1000),
        gen.SliceOfN(10, gen.UInt32()),
    )
    
    properties.Property("event_ordering", gen.Property(func(blockNum uint64, logIndices []uint32) bool {
        // Create events
        events := make([]*Event, len(logIndices))
        for i, logIdx := range logIndices {
            events[i] = &Event{
                BlockNumber: blockNum,
                LogIndex:    logIdx,
            }
        }
        
        // Validate ordering
        return validateEventOrdering(events)
    }))
    
    properties.TestingRun(t)
}
```

## Debugging Tips

### Enable Verbose Logging
```bash
go test -v -run TestEventCollection
```

### Run Single Test
```bash
go test -run TestEventCollection -v
```

### Check Anvil Logs
```bash
# In separate terminal
anvil --verbose
```

### Inspect Database
```bash
# Connect to PostgreSQL
psql -h localhost -U postgres -d chainpulse

# Query events
SELECT * FROM events LIMIT 10;
```

### Monitor Indexer
```bash
# Check health
curl http://localhost:8080/health

# Query events
curl http://localhost:8080/api/v1/events?limit=10
```

## Environment Variables

```bash
# Anvil configuration
export E2E_ANVIL_PORT=8545
export E2E_ANVIL_BLOCK_TIME=1

# Indexer configuration
export E2E_INDEXER_PORT=8080
export E2E_INDEXER_LOG_LEVEL=DEBUG

# Database configuration
export E2E_DATABASE_URL="postgres://user:password@localhost:5432/chainpulse"

# Test configuration
export E2E_TIMEOUT=300s
export E2E_PARALLEL=1
```

## Troubleshooting

### Anvil Connection Error
```bash
# Check if Anvil is running
lsof -i :8545

# Start Anvil manually
anvil --port 8545
```

### Database Connection Error
```bash
# Check PostgreSQL
psql -h localhost -U postgres

# Create test database
createdb chainpulse
```

### Event Not Indexed
```bash
# Check indexer logs
tail -f indexer.log

# Verify contract deployment
curl http://localhost:8080/api/v1/contracts

# Check event emission
curl http://localhost:8080/api/v1/events
```

### Race Condition
```bash
# Run with race detector
go test -race ./test/e2e/...

# Increase concurrency
go test -race -parallel 4 ./test/e2e/...
```

## Next Steps

1. **Review Requirements**: Read `requirements.md` for detailed requirements
2. **Study Design**: Review `design.md` for architecture and components
3. **Implement Tasks**: Follow `tasks.md` for implementation steps
4. **Learn Best Practices**: Study `INDUSTRY_BEST_PRACTICES.md` for patterns
5. **Run Tests**: Execute tests and validate results

## Resources

- [Anvil Documentation](https://book.getfoundry.sh/anvil/)
- [Hardhat Documentation](https://hardhat.org/docs)
- [Go Testing](https://golang.org/pkg/testing/)
- [gopter Documentation](https://github.com/leanovate/gopter)
- [testify Documentation](https://github.com/stretchr/testify)

## Support

For issues or questions:
1. Check troubleshooting section
2. Review test logs
3. Check Anvil output
4. Verify environment setup
5. Open an issue with logs and configuration
