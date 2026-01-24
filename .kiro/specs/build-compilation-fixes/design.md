# Design: Build Compilation Fixes - Test Integration Issues

## Overview

This design addresses 27 typecheck errors in test files that prevent the project from compiling. The errors fall into several categories:
1. Interface implementation gaps (MockCachePlugin missing Initialize)
2. Type mismatches in struct fields
3. Incorrect function signatures
4. Missing imports
5. Undefined methods

The solution involves updating test mocks to properly implement interfaces, fixing method signatures to match actual implementations, and correcting type mismatches.

## Architecture

### Test Mock Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Test Mocks                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  MockCachePlugin                                        │
│  ├── Initialize(ctx context.Context) error             │
│  ├── Get(ctx context.Context, key string) ([]byte, err)│
│  ├── Set(ctx context.Context, key string, val []byte)  │
│  └── Delete(ctx context.Context, key string) error     │
│                                                         │
│  MockDatabasePlugin                                     │
│  ├── events: map[string]*core.BlockchainEvent          │
│  ├── QueryEvents(ctx context.Context, filter interface{})
│  └── StoreEvent(ctx context.Context, event *Event)     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Error Categories and Solutions

**Category 1: MockCachePlugin Missing Initialize**
- Issue: MockCachePlugin doesn't implement core.CachePlugin interface
- Solution: Add Initialize method to MockCachePlugin
- Files: test/integration/query_examples_test.go

**Category 2: Event Storage Type Mismatch**
- Issue: events field is []*core.BlockchainEvent but interface expects map[string]*core.BlockchainEvent
- Solution: Change events field type to map[string]*core.BlockchainEvent
- Files: test/integration/query_examples_test.go, multi_chain_indexing_test.go

**Category 3: QueryEvents Signature Mismatch**
- Issue: Tests call QueryEvents(filter, offset, pageSize) but signature is QueryEvents(ctx, interface{})
- Solution: Update test calls to use correct signature with context
- Files: test/integration/query_examples_test.go

**Category 4: Cache Method Signatures**
- Issue: Tests call cache.Get(key) and cache.Set(key, value, ttl) but signatures require context
- Solution: Update test calls to include context parameter
- Files: test/integration/query_examples_test.go

**Category 5: Missing Imports**
- Issue: fixtures package import fails
- Solution: Verify fixtures package exists or remove unused imports
- Files: test/integration/multi_chain_integration_test.go

**Category 6: Undefined Methods**
- Issue: SetPoolMetadata called on UniswapIndexer but method doesn't exist
- Solution: Remove calls to undefined methods or add them to implementation
- Files: test/integration/query_examples_test.go

## Components and Interfaces

### MockCachePlugin Implementation

```go
type MockCachePlugin struct {
    data map[string][]byte
    mu   sync.RWMutex
}

func (m *MockCachePlugin) Initialize(ctx context.Context) error {
    return nil
}

func (m *MockCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    val, ok := m.data[key]
    if !ok {
        return nil, errors.New("key not found")
    }
    return val, nil
}

func (m *MockCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[key] = value
    return nil
}

func (m *MockCachePlugin) Delete(ctx context.Context, key string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    delete(m.data, key)
    return nil
}
```

### MockDatabasePlugin Update

```go
type MockDatabasePlugin struct {
    events map[string]*core.BlockchainEvent
    mu     sync.RWMutex
}

func (m *MockDatabasePlugin) QueryEvents(ctx context.Context, filter interface{}) (interface{}, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // Convert filter to EventFilter and query
    // Return results in expected format
}
```

## Data Models

### Event Storage
- Change from: `events []*core.BlockchainEvent`
- Change to: `events map[string]*core.BlockchainEvent`
- Key format: `{chainID}:{blockNumber}:{txHash}:{logIndex}`

### Cache Storage
- Key: string (cache key)
- Value: []byte (serialized data)
- TTL: int (seconds)

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Mock Cache Implements Interface
*For any* MockCachePlugin instance, calling Initialize, Get, Set, and Delete methods SHALL succeed without panicking and SHALL return appropriate values or errors.
**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: Event Storage Type Consistency
*For any* MockDatabasePlugin instance, the events field SHALL maintain type consistency as map[string]*core.BlockchainEvent, and QueryEvents SHALL return results in the correct format.
**Validates: Requirements 2.1, 2.2, 2.3, 2.4**

### Property 3: QueryEvents Signature Correctness
*For any* call to QueryEvents with context and filter parameters, the method SHALL accept these parameters and return results without type errors.
**Validates: Requirements 3.1, 3.2, 3.3, 3.4**

### Property 4: Cache Method Signatures
*For any* cache operation with context parameter, the Get and Set methods SHALL accept context.Context as the first parameter and execute successfully.
**Validates: Requirements 4.1, 4.2, 4.3, 4.4**

### Property 5: Import Resolution
*For any* test file that imports fixtures package, the import SHALL resolve successfully without errors.
**Validates: Requirements 5.1, 5.2, 5.3**

### Property 6: Method Existence
*For any* method call in test code, the method SHALL exist on the target type or the call SHALL be removed.
**Validates: Requirements 6.1, 6.2, 6.3**

### Property 7: Clean Compilation
*For any* execution of golangci-lint, the typecheck linter SHALL report 0 errors for test files.
**Validates: Requirements 7.1, 7.2, 7.3**

## Error Handling

- All context operations should handle context cancellation
- Cache operations should return appropriate errors for missing keys
- Database queries should return empty results for no matches
- Type mismatches should be caught at compile time

## Testing Strategy

### Unit Tests
- Test MockCachePlugin methods individually
- Test MockDatabasePlugin event storage and retrieval
- Test method signatures with correct parameters

### Integration Tests
- Test mocks with actual indexer implementations
- Test cache operations in context of queries
- Test database operations with event filters

### Property-Based Tests
- Generate random cache keys and values
- Verify cache operations maintain consistency
- Verify event storage maintains type safety

</content>
</invoke>