# Code Coverage Guide

## Overview

This guide explains how to measure, improve, and maintain code coverage in ChainPulse.

## Quick Start

### Run All Tests with Coverage
```bash
go test -cover ./...
```

### Generate HTML Coverage Report
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
# or
xdg-open coverage.html  # Linux
```

### Check Coverage by Package
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep pkg/
```

## Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| `pkg/core` | 90% | - |
| `pkg/infrastructure` | 80% | - |
| `pkg/services` | 85% | - |
| `pkg/plugins` | 75% | - |
| `pkg/integrations` | 70% | - |
| `pkg/observability` | 80% | - |
| **Overall** | **60%** | - |

## Running Tests Locally

### Unit Tests Only
```bash
go test -short -cover ./pkg/...
```

### Integration Tests
```bash
# Requires PostgreSQL and Redis running
go test -cover ./test/integration/...
```

### E2E Tests
```bash
# Requires Anvil, PostgreSQL, and Redis running
go test -cover ./test/e2e/...
```

### All Tests with Coverage
```bash
# Start dependencies
docker-compose -f docker/docker-compose.yml up -d

# Run all tests
go test -coverprofile=coverage.out -covermode=atomic ./...

# Generate report
go tool cover -html=coverage.out
```

## Improving Coverage

### 1. Identify Uncovered Code
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Look for red areas in the HTML report.

### 2. Write Tests for Uncovered Code
```go
func TestMyFunction(t *testing.T) {
    // Test the uncovered code path
    result := MyFunction(input)
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

### 3. Verify Coverage Improved
```bash
go test -coverprofile=coverage.out ./pkg/mypackage/...
go tool cover -func=coverage.out | grep MyFunction
```

## Coverage by Component

### Core Package
- **Target:** 90%
- **Focus:** Foundation interfaces and types
- **Strategy:** Comprehensive unit tests for all types and interfaces

### Infrastructure Package
- **Target:** 80%
- **Focus:** Database, deployment, and infrastructure utilities
- **Strategy:** Integration tests with real services

### Services Package
- **Target:** 85%
- **Focus:** Business logic and event processing
- **Strategy:** Unit tests + integration tests

### Plugins Package
- **Target:** 75%
- **Focus:** Plugin implementations
- **Strategy:** Plugin-specific tests + integration tests

### Integrations Package
- **Target:** 70%
- **Focus:** External integrations
- **Strategy:** Mock-based tests + integration tests

### Observability Package
- **Target:** 80%
- **Focus:** Monitoring and tracing
- **Strategy:** Unit tests for metrics and logging

## CI/CD Coverage Checks

### GitHub Actions Workflow
The coverage workflow runs automatically on:
- Push to main, develop, v2 branches
- Pull requests to main, develop, v2 branches
- Manual trigger via workflow_dispatch

### Coverage Report
After each run, coverage reports are available in:
- **Artifacts:** GitHub Actions artifacts section
- **PR Comments:** Automatic coverage summary
- **Codecov:** https://codecov.io/

### Minimum Coverage
- **Overall:** 60%
- **Failure:** Coverage below 60% will fail the build

## Best Practices

### 1. Write Tests First
- Use TDD (Test-Driven Development)
- Write tests before implementing features
- Ensures better coverage from the start

### 2. Test Edge Cases
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        {"normal case", input1, expected1, false},
        {"edge case", input2, expected2, false},
        {"error case", input3, nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### 3. Use Table-Driven Tests
- More comprehensive coverage
- Easier to add new test cases
- Better readability

### 4. Test Error Paths
```go
func TestErrorHandling(t *testing.T) {
    _, err := MyFunction(invalidInput)
    if err == nil {
        t.Error("expected error, got nil")
    }
}
```

### 5. Use Mocks for External Dependencies
```go
type MockDatabase struct {
    mock.Mock
}

func (m *MockDatabase) Query(ctx context.Context, sql string) (interface{}, error) {
    args := m.Called(ctx, sql)
    return args.Get(0), args.Error(1)
}
```

## Troubleshooting

### Coverage Report Shows 0%
**Cause:** Tests not running or coverage file not generated  
**Solution:**
1. Verify tests run: `go test -v ./...`
2. Check coverage file: `ls -la coverage.out`
3. Verify coverage mode: `head -1 coverage.out` should show `mode: atomic`

### Coverage Lower Than Expected
**Cause:** Uncovered code paths or skipped tests  
**Solution:**
1. Generate HTML report: `go tool cover -html=coverage.out`
2. Identify red areas (uncovered code)
3. Write tests for those paths

### Tests Timeout
**Cause:** Long-running tests or hanging processes  
**Solution:**
1. Increase timeout: `go test -timeout 30m ./...`
2. Check for goroutine leaks: `go test -race ./...`
3. Profile tests: `go test -cpuprofile=cpu.prof ./...`

## Tools and Resources

### Go Coverage Tools
- `go test -cover` - Basic coverage
- `go tool cover` - HTML reports
- `gocovmerge` - Merge coverage files

### External Tools
- [Codecov](https://codecov.io/) - Coverage tracking
- [Coveralls](https://coveralls.io/) - Coverage history
- [SonarQube](https://www.sonarqube.org/) - Code quality

### Documentation
- [Go Testing Package](https://golang.org/pkg/testing/)
- [Go Coverage Documentation](https://golang.org/doc/effective_go#testing)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)

## Coverage Metrics

### How Coverage is Calculated
- Coverage = (Covered Statements / Total Statements) × 100%
- Measured at the statement level
- Includes all code paths

### Coverage Modes
- `set` - Did each statement run?
- `count` - How many times did each statement run?
- `atomic` - Like count, but safe for concurrent programs

## Next Steps

1. **Check Current Coverage:** `go test -cover ./...`
2. **Identify Gaps:** `go tool cover -html=coverage.out`
3. **Write Tests:** Add tests for uncovered code
4. **Verify:** Re-run coverage check
5. **Monitor:** Track coverage trends over time

---

**Last Updated:** January 22, 2026  
**Coverage Target:** 60% overall, 80%+ for core packages
