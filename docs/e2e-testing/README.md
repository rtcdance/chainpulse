# E2E Testing Framework Guide

## Overview

The ChainPulse Web3 Indexer E2E testing framework provides comprehensive end-to-end testing capabilities for validating the entire indexing pipeline. This guide covers framework architecture, usage patterns, and best practices.

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Anvil (for local blockchain simulation)
- PostgreSQL 14+
- Redis 7+

### Running Your First Test

```bash
# Set up test environment
export ANVIL_RPC_URL=http://localhost:8545
export POSTGRES_URL=postgres://user:pass@localhost:5432/chainpulse_test
export REDIS_URL=redis://localhost:6379

# Run E2E tests
go test ./test/e2e/... -v -timeout 30m

# Run specific test scenario
go test ./test/e2e -run TestHappyPath -v
```

### Test Scenarios

The framework includes several pre-built test scenarios:

- **Happy Path**: Basic event collection and indexing flow
- **Error Scenarios**: Error handling and recovery
- **Performance**: Throughput and latency validation
- **Multi-Chain**: Cross-chain indexing scenarios
- **Concurrent**: Concurrent event processing

## Key Features

### Comprehensive Test Coverage

- Event collection from blockchain
- Event processing and indexing
- Query execution and validation
- Error handling and recovery
- Performance under load
- Multi-chain scenarios

### Flexible Test Scenarios

- Pre-built scenario templates
- Custom scenario support
- Parameterized test execution
- Fixture management
- Error injection capabilities

### Performance Metrics

- Event collection latency
- Event processing throughput
- Query response time
- Resource utilization
- Concurrent processing capacity

### Production Validation

- Test coverage verification
- Performance baseline validation
- Documentation completeness check
- Deployment readiness assessment

## Documentation Structure

- **[Architecture](./architecture.md)** - Framework design and components
- **[Components](./components.md)** - Component reference and interfaces
- **[Configuration](./configuration.md)** - Configuration options and setup
- **[Examples](./examples/)** - Code examples and patterns
- **[Troubleshooting](./troubleshooting.md)** - Common issues and solutions
- **[API Reference](./api-reference.md)** - Public interfaces and data models
- **[FAQ](./faq.md)** - Frequently asked questions

## Common Tasks

### Running Tests

```bash
# Run all E2E tests
go test ./test/e2e/... -v

# Run with coverage
go test ./test/e2e/... -v -coverprofile=coverage.out

# Run specific test
go test ./test/e2e -run TestName -v

# Run with timeout
go test ./test/e2e/... -v -timeout 60m
```

### Debugging Tests

```bash
# Run with verbose logging
go test ./test/e2e/... -v -run TestName

# Run with race detector
go test ./test/e2e/... -race

# Run with pprof profiling
go test ./test/e2e/... -cpuprofile=cpu.prof -memprofile=mem.prof
```

### Performance Testing

```bash
# Run performance tests
go test ./test/e2e -run TestPerformance -v

# Collect metrics
go test ./test/e2e -run TestPerformance -v -metrics

# Compare with baseline
go test ./test/e2e -run TestPerformance -v -baseline
```

## Performance Targets

The framework validates these performance targets:

- **Event Collection Latency**: < 2 seconds
- **Event Processing Throughput**: > 1000 events/second
- **Query Response Time**: < 500ms
- **Code Coverage**: ≥ 80%
- **Test Pass Rate**: 100%

## Next Steps

1. Read the [Architecture Guide](./architecture.md) to understand framework design
2. Review [Components](./components.md) for available test utilities
3. Check [Examples](./examples/) for common test patterns
4. See [Configuration](./configuration.md) for setup options
5. Consult [Troubleshooting](./troubleshooting.md) for common issues

## Support

For issues or questions:

1. Check the [FAQ](./faq.md) for common questions
2. Review [Troubleshooting](./troubleshooting.md) for known issues
3. Check test logs for error details
4. Review [API Reference](./api-reference.md) for interface details

## Contributing

When adding new tests:

1. Follow existing test patterns
2. Include comprehensive error cases
3. Add performance assertions
4. Document test scenarios
5. Update this guide with new patterns

## Related Documentation

- [Production Deployment Guide](../deployment/production-checklist.md)
- [Monitoring and Alerting](../deployment/monitoring.md)
- [Operations Guide](../deployment/operations.md)
