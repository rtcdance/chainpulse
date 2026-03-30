---
name: "testing"
description: "Guidelines for writing tests in Go. Invoke when writing unit tests, integration tests, or benchmarks."
---

# Testing Guidelines

## Purpose
Ensure comprehensive test coverage and reliable testing practices.

## When to Invoke
- Writing unit tests
- Writing integration tests
- Creating test fixtures
- Setting up test environments

## Test File Convention

```
pkg/
├── service.go
└── service_test.go    # Test file alongside source
```

## Test Structure

```go
package pkg_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestService_DoSomething(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "success case",
            input:   "valid",
            want:    "result",
            wantErr: false,
        },
        {
            name:    "error case",
            input:   "invalid",
            want:    "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            
            got, err := DoSomething(tt.input)
            
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Test Categories

### Unit Tests
- Test single function/method
- Mock external dependencies
- Fast execution

### Integration Tests
- Test component interactions
- Use real dependencies where possible
- May require setup/teardown

### Benchmarks
```go
func BenchmarkService_DoSomething(b *testing.B) {
    for i := 0; i < b.N; i++ {
        DoSomething("input")
    }
}
```

## Table-Driven Tests

Always prefer table-driven tests:

```go
tests := []struct {
    name string
    // fields
}{
    // cases
}
```

## Assertions

- Use `require` for assertions that should stop the test
- Use `assert` for assertions that should continue

## Constraints
- ALWAYS use table-driven tests for multiple cases
- ALWAYS use `t.Parallel()` where possible
- DO NOT use global state in tests
- DO NOT skip tests without good reason
- Aim for >80% coverage on critical paths
