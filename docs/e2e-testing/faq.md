# E2E Testing Framework FAQ

## General Questions

### Q: What is the E2E testing framework?

A: The E2E testing framework provides comprehensive end-to-end testing capabilities for the ChainPulse Web3 Indexer. It validates the entire pipeline from blockchain event collection through indexing and querying.

### Q: What does the framework test?

A: The framework tests:
- Event collection from blockchain
- Event processing and indexing
- Query execution and validation
- Error handling and recovery
- Performance under load
- Multi-chain scenarios
- Concurrent processing

### Q: How long do tests take to run?

A: Test duration depends on the scenario:
- Happy path: ~5-10 minutes
- Error scenarios: ~10-15 minutes
- Performance tests: ~15-30 minutes
- Multi-chain tests: ~20-40 minutes
- Full suite: ~30-60 minutes

### Q: Can I run tests in parallel?

A: Yes, tests can run in parallel. Set `TEST_PARALLEL` environment variable:
```bash
export TEST_PARALLEL=8
go test ./test/e2e/... -v
```

### Q: What are the system requirements?

A: Minimum requirements:
- Go 1.21+
- Docker and Docker Compose
- 4GB RAM
- 10GB disk space
- Anvil (for blockchain simulation)

## Setup and Configuration

### Q: How do I set up the test environment?

A: Follow these steps:

1. Start services:
   ```bash
   docker-compose -f docker-compose.test.yml up -d
   ```

2. Set environment variables:
   ```bash
   export ANVIL_RPC_URL=http://localhost:8545
   export POSTGRES_URL=postgres://test:test@localhost:5432/chainpulse_test
   export REDIS_URL=redis://localhost:6379
   ```

3. Run tests:
   ```bash
   go test ./test/e2e/... -v
   ```

### Q: Can I use a different database?

A: The framework is designed for PostgreSQL. To use a different database, you would need to implement a custom `DatabaseManager`.

### Q: Can I run tests without Docker?

A: You can run tests without Docker if you have Anvil, PostgreSQL, and Redis running locally. Set the appropriate connection strings in environment variables.

### Q: How do I configure test timeouts?

A: Set the `TEST_TIMEOUT` environment variable:
```bash
export TEST_TIMEOUT=60m
go test ./test/e2e/... -v -timeout 60m
```

## Running Tests

### Q: How do I run a specific test?

A: Use the `-run` flag:
```bash
go test ./test/e2e -run TestHappyPath -v
```

### Q: How do I run tests with coverage?

A: Use the `-cover` flag:
```bash
go test ./test/e2e/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Q: How do I run tests with verbose output?

A: Use the `-v` flag:
```bash
go test ./test/e2e/... -v
```

### Q: How do I run tests with race detection?

A: Use the `-race` flag:
```bash
go test ./test/e2e/... -v -race
```

### Q: How do I profile test execution?

A: Use profiling flags:
```bash
go test ./test/e2e/... -v -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
```

## Test Scenarios

### Q: What test scenarios are available?

A: The framework includes:
- **Happy Path**: Normal operation flow
- **Error Scenarios**: Error handling and recovery
- **Performance**: Throughput and latency validation
- **Multi-Chain**: Cross-chain indexing
- **Concurrent**: Concurrent event processing

### Q: Can I create custom test scenarios?

A: Yes, implement the `Scenario` interface:
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

### Q: How do I add error injection to tests?

A: Use the error injection utilities:
```go
// Inject network error
if err := InjectNetworkError(ctx, 5*time.Second); err != nil {
    return err
}

// Verify error handling
// ...
```

### Q: How do I measure performance?

A: Use the performance measurement utilities:
```go
latency, err := MeasureLatency(ctx, func() error {
    return indexer.ProcessEvent(ctx, event)
})

throughput, err := MeasureThroughput(ctx, func() error {
    return indexer.ProcessEvent(ctx, event)
}, 10*time.Second)
```

## Performance and Optimization

### Q: What are the performance targets?

A: The framework validates:
- Event collection latency: < 2 seconds
- Event processing throughput: > 1000 events/second
- Query response time: < 500ms
- Code coverage: ≥ 80%
- Test pass rate: 100%

### Q: How do I improve test performance?

A: Try these optimizations:
1. Increase parallelism: `export TEST_PARALLEL=8`
2. Reduce test scope: Run specific tests
3. Increase resource allocation: More CPU/memory
4. Use SSD storage: Faster disk I/O
5. Reduce event count: `export PERF_EVENT_COUNT=5000`

### Q: Why are tests running slowly?

A: Common causes:
- Insufficient system resources
- High system load
- Slow disk I/O
- Network latency
- Database performance

Check system resources:
```bash
docker stats
top -l 1 | head -20
```

### Q: How do I profile performance bottlenecks?

A: Use profiling tools:
```bash
go test ./test/e2e/... -v -cpuprofile=cpu.prof
go tool pprof cpu.prof

go test ./test/e2e/... -v -memprofile=mem.prof
go tool pprof mem.prof
```

## Troubleshooting

### Q: Tests fail with connection errors. What should I do?

A: Check that all services are running:
```bash
docker-compose -f docker-compose.test.yml ps
```

Verify environment variables:
```bash
echo $ANVIL_RPC_URL
echo $POSTGRES_URL
echo $REDIS_URL
```

Test connectivity:
```bash
curl $ANVIL_RPC_URL
psql $POSTGRES_URL -c "SELECT 1"
redis-cli -u $REDIS_URL ping
```

### Q: Tests timeout. What should I do?

A: Increase timeouts:
```bash
export TEST_TIMEOUT=60m
export INDEXER_TIMEOUT=60s
```

Check system resources:
```bash
docker stats
```

Run tests sequentially:
```bash
export TEST_PARALLEL=1
```

### Q: Tests fail intermittently. What should I do?

A: Use fixed random seed:
```bash
export TEST_SEED=12345
```

Check for race conditions:
```bash
go test ./test/e2e/... -v -race
```

Run tests multiple times:
```bash
for i in {1..5}; do
  go test ./test/e2e/... -v
done
```

### Q: How do I debug test failures?

A: Enable verbose logging:
```bash
export LOG_LEVEL=debug
export TEST_VERBOSE=true
go test ./test/e2e/... -v
```

Capture output:
```bash
go test ./test/e2e/... -v > test-output.log 2>&1
```

Use debugger:
```bash
dlv test ./test/e2e -- -test.run TestName
```

### Q: Where can I find test logs?

A: Test logs are printed to stdout. Capture them:
```bash
go test ./test/e2e/... -v > test-output.log 2>&1
```

Service logs are available via Docker:
```bash
docker-compose -f docker-compose.test.yml logs -f
```

## CI/CD Integration

### Q: How do I run tests in GitHub Actions?

A: Create `.github/workflows/test.yml`:
```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      anvil:
        image: ghcr.io/foundry-rs/foundry:latest
        options: --health-cmd "curl -f http://localhost:8545" --health-interval 10s --health-timeout 5s --health-retries 5
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
      - run: go test ./test/e2e/... -v
```

### Q: How do I collect metrics in CI/CD?

A: Add metrics collection step:
```yaml
- name: Collect metrics
  run: |
    go test ./test/e2e/... -v -metrics
    ./scripts/collect-metrics.sh
```

### Q: How do I generate coverage reports?

A: Add coverage step:
```yaml
- name: Generate coverage
  run: |
    go test ./test/e2e/... -v -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
```

## Best Practices

### Q: What are best practices for writing tests?

A: Follow these guidelines:
1. Use descriptive test names
2. Test one scenario per test
3. Include error cases
4. Add performance assertions
5. Document test purpose
6. Clean up resources
7. Use fixtures for test data
8. Validate results thoroughly

### Q: How should I organize test code?

A: Organize by scenario:
```
test/e2e/
├── happy_path_scenarios.go
├── error_scenarios.go
├── performance_scenarios.go
├── multi_chain_scenarios.go
└── concurrent_scenarios.go
```

### Q: How do I handle test data?

A: Use fixtures:
```go
fixtures := e2e.NewFixtureManager(config)
events, err := fixtures.GenerateEvents(ctx, 100)
```

### Q: How do I validate test results?

A: Use validators:
```go
validator := e2e.NewValidator(config)
if err := validator.ValidateEventData(event); err != nil {
    return err
}
```

## Advanced Topics

### Q: How do I extend the framework?

A: Implement custom interfaces:
1. Custom scenarios: Implement `Scenario`
2. Custom managers: Extend `ComponentManager`
3. Custom validators: Implement validation logic
4. Custom metrics: Add metric collection

### Q: Can I use the framework for load testing?

A: Yes, use performance scenarios:
```bash
export PERF_EVENT_COUNT=100000
export PERF_CONCURRENT=50
go test ./test/e2e -run TestPerformance -v
```

### Q: How do I integrate with monitoring systems?

A: Export metrics:
```bash
go test ./test/e2e/... -v -metrics
./scripts/export-metrics.sh
```

## Support and Resources

### Q: Where can I find more information?

A: Check these resources:
- [Architecture Guide](./architecture.md)
- [Components Reference](./components.md)
- [Configuration Guide](./configuration.md)
- [Examples](./examples/)
- [Troubleshooting Guide](./troubleshooting.md)
- [API Reference](./api-reference.md)

### Q: How do I report issues?

A: Report issues with:
1. Test name and command
2. Error message and logs
3. Environment details
4. Steps to reproduce
5. Expected vs actual behavior

### Q: How do I contribute improvements?

A: Contributions welcome! Please:
1. Follow existing patterns
2. Add tests for new features
3. Update documentation
4. Submit pull request
5. Include test results

## Related Documentation

- [Architecture Guide](./architecture.md)
- [Components Reference](./components.md)
- [Configuration Guide](./configuration.md)
- [Examples](./examples/)
- [Troubleshooting Guide](./troubleshooting.md)
- [API Reference](./api-reference.md)
