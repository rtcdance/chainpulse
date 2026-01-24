# Comprehensive E2E Testing Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [Test Execution](#test-execution)
3. [Writing Tests](#writing-tests)
4. [Test Scenarios](#test-scenarios)
5. [Performance Testing](#performance-testing)
6. [Debugging Tests](#debugging-tests)
7. [CI/CD Integration](#cicd-integration)
8. [Best Practices](#best-practices)

## Getting Started

### Prerequisites

Before running E2E tests, ensure you have:

- Go 1.21 or later
- Docker and Docker Compose
- Anvil (Foundry)
- PostgreSQL 14+
- Redis 7+
- 4GB+ RAM available
- 10GB+ disk space

### Initial Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/chainpulse/indexer.git
   cd indexer
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Start test services:**
   ```bash
   docker-compose -f docker-compose.test.yml up -d
   ```

4. **Verify services are running:**
   ```bash
   docker-compose -f docker-compose.test.yml ps
   ```

5. **Set environment variables:**
   ```bash
   export ANVIL_RPC_URL=http://localhost:8545
   export POSTGRES_URL=postgres://test:test@localhost:5432/chainpulse_test
   export REDIS_URL=redis://localhost:6379
   export LOG_LEVEL=info
   ```

6. **Run your first test:**
   ```bash
   go test ./test/e2e -run TestHappyPath -v
   ```

## Test Execution

### Running All Tests

```bash
# Run all E2E tests
go test ./test/e2e/... -v

# Run with timeout
go test ./test/e2e/... -v -timeout 60m

# Run with coverage
go test ./test/e2e/... -v -cover -coverprofile=coverage.out
```

### Running Specific Tests

```bash
# Run specific test
go test ./test/e2e -run TestHappyPath -v

# Run tests matching pattern
go test ./test/e2e -run Test.*Performance -v

# Run tests in specific package
go test ./test/e2e/scenarios -v
```

### Running Tests in Parallel

```bash
# Run tests in parallel (default: 4 workers)
go test ./test/e2e/... -v -parallel 8

# Run tests sequentially
go test ./test/e2e/... -v -parallel 1
```

### Running Tests with Specific Configuration

```bash
# Development configuration
export LOG_LEVEL=debug
export TEST_VERBOSE=true
go test ./test/e2e/... -v

# Performance testing configuration
export PERF_EVENT_COUNT=50000
export PERF_DURATION=120s
export PERF_CONCURRENT=50
go test ./test/e2e -run TestPerformance -v

# CI/CD configuration
export LOG_LEVEL=info
export TEST_TIMEOUT=60m
go test ./test/e2e/... -v -timeout 60m
```

## Writing Tests

### Test Structure

All E2E tests follow this structure:

```go
package e2e

import (
    "context"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestMyScenario(t *testing.T) {
    // 1. Setup
    ctx := context.Background()
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(ctx)
    
    require.NoError(t, orchestrator.Initialize(ctx))
    
    // 2. Execute
    scenario := &MyScenario{}
    result, err := orchestrator.ExecuteScenario(ctx, scenario)
    
    // 3. Validate
    require.NoError(t, err)
    require.True(t, result.Passed)
    require.NotNil(t, result.Metrics)
}
```

### Creating Custom Scenarios

Implement the `Scenario` interface:

```go
type MyScenario struct {
    // Scenario configuration
    eventCount int
    timeout    time.Duration
}

func (s *MyScenario) Name() string {
    return "My Custom Scenario"
}

func (s *MyScenario) Description() string {
    return "Tests my custom functionality"
}

func (s *MyScenario) Execute(ctx context.Context, managers ComponentManagers) error {
    // 1. Deploy contracts
    address, err := managers.Blockchain.DeployContract(ctx, contract)
    if err != nil {
        return err
    }
    
    // 2. Emit events
    for i := 0; i < s.eventCount; i++ {
        event := createTestEvent(i)
        _, err := managers.Blockchain.EmitEvent(ctx, event)
        if err != nil {
            return err
        }
    }
    
    // 3. Wait for indexing
    if err := managers.Indexer.WaitForSync(ctx, s.timeout); err != nil {
        return err
    }
    
    // 4. Query results
    events, err := managers.API.QueryEvents(ctx, query)
    if err != nil {
        return err
    }
    
    // 5. Store results for validation
    s.results = events
    return nil
}

func (s *MyScenario) Validate(ctx context.Context) error {
    if len(s.results) != s.eventCount {
        return fmt.Errorf("expected %d events, got %d", s.eventCount, len(s.results))
    }
    return nil
}

func (s *MyScenario) Cleanup(ctx context.Context) error {
    // Clean up scenario resources
    return nil
}
```

### Using Test Utilities

#### Error Injection

```go
func TestErrorHandling(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(ctx)
    
    require.NoError(t, orchestrator.Initialize(ctx))
    
    // Inject network error for 5 seconds
    injector := NewErrorInjector(orchestrator.Config())
    go func() {
        injector.InjectNetworkError(ctx, 5*time.Second)
    }()
    
    // Execute scenario while error is injected
    scenario := &MyScenario{}
    result, err := orchestrator.ExecuteScenario(ctx, scenario)
    
    // Verify error handling
    require.NoError(t, err)
    require.True(t, result.Passed)
}
```

#### Performance Measurement

```go
func TestPerformance(t *testing.T) {
    ctx := context.Background()
    measurer := NewPerformanceMeasurer()
    
    // Measure latency
    latency, err := measurer.MeasureLatency(ctx, func() error {
        return indexer.ProcessEvent(ctx, event)
    })
    require.NoError(t, err)
    require.Less(t, latency, 2*time.Second)
    
    // Measure throughput
    throughput, err := measurer.MeasureThroughput(ctx, func() error {
        return indexer.ProcessEvent(ctx, event)
    }, 10*time.Second)
    require.NoError(t, err)
    require.Greater(t, throughput, 1000.0) // events/sec
}
```

#### Fixture Management

```go
func TestWithFixtures(t *testing.T) {
    ctx := context.Background()
    fixtures := NewFixtureManager(defaultConfig)
    
    // Generate test events
    events, err := fixtures.GenerateEvents(ctx, 100)
    require.NoError(t, err)
    require.Len(t, events, 100)
    
    // Generate test contract
    contract, err := fixtures.GenerateContract(ctx)
    require.NoError(t, err)
    require.NotNil(t, contract)
}
```

## Test Scenarios

### Happy Path Scenario

Tests the normal operation flow:

```go
func TestHappyPath(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(ctx)
    
    require.NoError(t, orchestrator.Initialize(ctx))
    
    scenario := &HappyPathScenario{
        eventCount: 100,
    }
    
    result, err := orchestrator.ExecuteScenario(ctx, scenario)
    require.NoError(t, err)
    require.True(t, result.Passed)
    
    // Verify metrics
    require.Less(t, result.Metrics.CollectionLatency, 2*time.Second)
    require.Greater(t, result.Metrics.ProcessingThroughput, 1000.0)
}
```

### Error Scenario

Tests error handling and recovery:

```go
func TestErrorScenario(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(ctx)
    
    require.NoError(t, orchestrator.Initialize(ctx))
    
    scenario := &ErrorScenario{
        errorType: "network_failure",
        duration:  5 * time.Second,
    }
    
    result, err := orchestrator.ExecuteScenario(ctx, scenario)
    require.NoError(t, err)
    require.True(t, result.Passed)
}
```

### Performance Scenario

Tests performance targets:

```go
func TestPerformanceScenario(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(ctx)
    
    require.NoError(t, orchestrator.Initialize(ctx))
    
    scenario := &PerformanceScenario{
        eventCount: 10000,
        duration:   60 * time.Second,
    }
    
    result, err := orchestrator.ExecuteScenario(ctx, scenario)
    require.NoError(t, err)
    require.True(t, result.Passed)
    
    // Verify performance targets
    require.Greater(t, result.Metrics.ProcessingThroughput, 1000.0)
    require.Less(t, result.Metrics.QueryLatency, 500*time.Millisecond)
}
```

### Multi-Chain Scenario

Tests multi-chain support:

```go
func TestMultiChainScenario(t *testing.T) {
    ctx := context.Background()
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(ctx)
    
    require.NoError(t, orchestrator.Initialize(ctx))
    
    scenario := &MultiChainScenario{
        chains: []string{"ethereum", "polygon", "arbitrum"},
    }
    
    result, err := orchestrator.ExecuteScenario(ctx, scenario)
    require.NoError(t, err)
    require.True(t, result.Passed)
}
```

## Performance Testing

### Setting Performance Targets

```bash
# Configure performance test parameters
export PERF_EVENT_COUNT=50000      # Number of events
export PERF_DURATION=120s          # Test duration
export PERF_CONCURRENT=50          # Concurrent operations
```

### Running Performance Tests

```bash
# Run performance tests
go test ./test/e2e -run TestPerformance -v

# Run with metrics collection
go test ./test/e2e -run TestPerformance -v -metrics

# Run with profiling
go test ./test/e2e -run TestPerformance -v \
  -cpuprofile=cpu.prof \
  -memprofile=mem.prof
```

### Analyzing Performance Results

```bash
# View CPU profile
go tool pprof cpu.prof

# View memory profile
go tool pprof mem.prof

# Generate performance report
./scripts/analyze-performance.sh
```

## Debugging Tests

### Enable Verbose Logging

```bash
export LOG_LEVEL=debug
export TEST_VERBOSE=true
go test ./test/e2e -run TestName -v
```

### Run with Race Detector

```bash
go test ./test/e2e/... -v -race
```

### Run with Debugger

```bash
dlv test ./test/e2e -- -test.run TestName
```

### Inspect Test State

```bash
# Connect to database
psql $POSTGRES_URL

# Query indexed events
SELECT * FROM events LIMIT 10;

# Check event count
SELECT COUNT(*) FROM events;
```

### Capture Test Output

```bash
# Save output to file
go test ./test/e2e/... -v > test-output.log 2>&1

# View output
tail -f test-output.log

# Search for errors
grep -i error test-output.log
```

## CI/CD Integration

### GitHub Actions Workflow

Create `.github/workflows/test.yml`:

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      anvil:
        image: ghcr.io/foundry-rs/foundry:latest
        options: --health-cmd "curl -f http://localhost:8545"
      postgres:
        image: postgres:14-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: chainpulse_test
      redis:
        image: redis:7-alpine
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run E2E tests
        env:
          ANVIL_RPC_URL: http://anvil:8545
          POSTGRES_URL: postgres://test:test@postgres:5432/chainpulse_test
          REDIS_URL: redis://redis:6379
        run: go test ./test/e2e/... -v -timeout 60m
      
      - name: Generate coverage
        run: |
          go test ./test/e2e/... -v -coverprofile=coverage.out
          go tool cover -html=coverage.out -o coverage.html
      
      - name: Upload coverage
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: coverage.html
```

### Running Tests Locally with Docker

```bash
# Build test image
docker build -f Dockerfile.test -t chainpulse-e2e-test .

# Run tests in container
docker run --rm \
  --network chainpulse-network \
  -e ANVIL_RPC_URL=http://anvil:8545 \
  -e POSTGRES_URL=postgres://test:test@postgres:5432/chainpulse_test \
  -e REDIS_URL=redis://redis:6379 \
  chainpulse-e2e-test
```

## Best Practices

### 1. Test Isolation

Each test should be independent:

```go
func TestIsolation(t *testing.T) {
    // Each test gets its own orchestrator
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(context.Background())
    
    // Tests don't interfere with each other
}
```

### 2. Meaningful Assertions

Use descriptive assertions:

```go
// Good
require.Equal(t, expectedCount, actualCount, "event count should match")

// Bad
require.Equal(t, 100, len(events))
```

### 3. Cleanup Resources

Always clean up resources:

```go
func TestCleanup(t *testing.T) {
    orchestrator := NewOrchestrator(defaultConfig)
    defer orchestrator.Cleanup(context.Background()) // Always cleanup
    
    // Test code
}
```

### 4. Use Fixtures for Test Data

```go
// Good
fixtures := NewFixtureManager(config)
events, _ := fixtures.GenerateEvents(ctx, 100)

// Bad
events := make([]Event, 100)
for i := 0; i < 100; i++ {
    events[i] = Event{...} // Manual creation
}
```

### 5. Test Performance Targets

```go
// Always verify performance targets
require.Less(t, latency, 2*time.Second, "latency should be < 2s")
require.Greater(t, throughput, 1000.0, "throughput should be > 1000 events/sec")
```

### 6. Document Test Purpose

```go
// Good - Clear purpose
func TestEventCollectionLatency(t *testing.T) {
    // Tests that event collection latency is < 2 seconds
    // as per requirement 1.1
}

// Bad - Unclear purpose
func TestLatency(t *testing.T) {
    // Tests latency
}
```

### 7. Use Table-Driven Tests

```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name       string
        eventCount int
        expected   int
    }{
        {"small", 10, 10},
        {"medium", 100, 100},
        {"large", 1000, 1000},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test code
        })
    }
}
```

### 8. Handle Timeouts Gracefully

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := orchestrator.ExecuteScenario(ctx, scenario)
if err == context.DeadlineExceeded {
    t.Fatal("test timed out")
}
```

## Related Documentation

- [Architecture Guide](./architecture.md)
- [Components Reference](./components.md)
- [Configuration Guide](./configuration.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [API Reference](./api-reference.md)
- [FAQ](./faq.md)
