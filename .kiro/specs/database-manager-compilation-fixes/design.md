# Design: Database Manager Compilation Fixes

## Overview

The database manager tests have compilation errors due to method signature changes. The `GetMongoClient()` and `GetPostgresDB()` methods now require `context.Context` parameters and return both a value and an error. The property-based tests also need to use proper gopter reporter patterns instead of passing `*testing.T` directly.

## Architecture

### Current Issues

1. **Method Signature Mismatch**: Tests call `GetMongoClient()` without context, but the method signature is `GetMongoClient(ctx context.Context) (interface{}, error)`
2. **Missing Error Handling**: Tests don't handle the error return values from these methods
3. **gopter Reporter Issue**: Property tests pass `*testing.T` to `properties.Run()`, but gopter expects a `gopter.Reporter` implementation
4. **Incomplete Assertions**: Tests need to properly validate both success and error cases

### Solution Approach

1. **Update manager_test.go**:
   - Add context creation for each test that calls GetMongoClient or GetPostgresDB
   - Handle both return values (interface{}, error)
   - Add proper error assertions

2. **Update manager_property_test.go**:
   - Create a proper gopter.Reporter wrapper for *testing.T
   - Update all properties.Run() calls to use the wrapper
   - Fix method calls to include context.Context parameter
   - Handle error returns properly

3. **Error Handling Patterns**:
   - For uninitialized manager: expect error
   - For initialized manager: expect success (or connection error if services unavailable)
   - For closed manager: expect error

## Components and Interfaces

### DatabaseManager Interface

```go
type DatabaseManager interface {
    GetMongoClient(ctx context.Context) (interface{}, error)
    GetPostgresDB(ctx context.Context) (interface{}, error)
    // ... other methods
}
```

### gopter.Reporter Implementation

```go
type testReporter struct {
    t *testing.T
}

func (r *testReporter) ReportTestResult(name string, passed bool, result *gopter.TestResult) {
    if !passed {
        r.t.Errorf("Property test failed: %s", name)
    }
}
```

## Data Models

No new data models needed. The existing DatabaseManager interface and DefaultDatabaseManager implementation remain unchanged.

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Context Propagation

**For any** database manager and any context, calling GetMongoClient or GetPostgresDB with that context SHALL properly propagate the context to the underlying database operations.

**Validates: Requirements 1.1, 2.1**

### Property 2: Error Handling Consistency

**For any** uninitialized database manager, calling GetMongoClient or GetPostgresDB SHALL return an error indicating the manager is not initialized.

**Validates: Requirements 1.3, 2.3**

### Property 3: Return Value Correctness

**For any** database manager method call, the return values SHALL match the declared method signature (interface{}, error).

**Validates: Requirements 1.2, 2.2**

### Property 4: Reporter Integration

**For any** property test, the gopter.Reporter SHALL properly report test results without panicking or failing to report.

**Validates: Requirement 3.2**

## Error Handling

1. **Uninitialized Manager**: Return error "database manager not initialized"
2. **Nil Client/DB**: Return error "MongoDB client not available" or "PostgreSQL connection not available"
3. **Context Timeout**: Propagate context timeout errors from underlying operations
4. **Connection Errors**: Return connection errors from MongoDB or PostgreSQL

## Testing Strategy

### Unit Tests

- Test GetMongoClient with uninitialized manager (expect error)
- Test GetPostgresDB with uninitialized manager (expect error)
- Test method calls with valid context
- Test method calls with cancelled context
- Test concurrent access patterns

### Property-Based Tests

- Property 1: Context propagation across all pool sizes
- Property 2: Error handling for uninitialized state
- Property 3: Return value correctness for all configurations
- Property 4: Reporter integration with various test scenarios

### Test Configuration

- Minimum 10 successful property test iterations
- Use gopter.DefaultTestParameters() with custom reporter
- Test with pool sizes from 1 to 100
- Test with timeouts from 1 to 60 seconds

