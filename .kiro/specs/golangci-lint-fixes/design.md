# Design: golangci-lint Fixes

## Architecture Overview

The fixes are organized into four main categories:

1. **Error Handling** - Proper error checking for I/O operations
2. **Type Safety** - Correct printf format specifiers
3. **API Updates** - Replace deprecated gRPC calls
4. **Code Cleanup** - Remove unused fields

## Detailed Design

### 1. Error Handling Strategy

#### Close() Operations
For resource cleanup operations like `Close()`, we have two options:

**Option A: Log and Ignore** (for cleanup operations)
```go
defer func() {
    if err := conn.Close(); err != nil {
        logger.Warn("failed to close connection", "error", err)
    }
}()
```

**Option B: Return Error** (for critical operations)
```go
if err := conn.Close(); err != nil {
    logger.Error("failed to close connection", "error", err)
    return err
}
```

**Decision**: Use Option A for cleanup operations (defer), Option B for explicit close calls.

#### SetDeadline() Operations
```go
if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
    logger.Error("failed to set write deadline", "error", err)
    return err
}
```

#### Publish() Operations
```go
if err := h.eventBus.Publish(ctx, topic, event); err != nil {
    logger.Error("failed to publish event", "error", err)
    // Decide: return error or continue
}
```

#### Sscanf() Operations
```go
if _, err := fmt.Sscanf(hexStr, "%x", &result); err != nil {
    logger.Error("failed to parse hex string", "error", err)
    return err
}
```

### 2. Type Safety Fix

**Current Issue:**
```go
return fmt.Sprintf("%s:%d:%s:%d:%s",
    event.TransactionHash,      // common.Hash
    event.BlockNumber,           // uint64
    event.LogIndex,              // uint - WRONG FORMAT
    event.ContractAddress,       // common.Address
    event.EventName,             // string
)
```

**Fix:**
```go
return fmt.Sprintf("%s:%d:%d:%s:%s",
    event.TransactionHash,      // %s
    event.BlockNumber,           // %d
    event.LogIndex,              // %d (changed from %s)
    event.ContractAddress,       // %s
    event.EventName,             // %s
)
```

### 3. gRPC API Update

**Current (Deprecated):**
```go
conn, err := grpc.Dial(p.nodeURL, opts...)
```

**Updated:**
```go
conn, err := grpc.NewClient(p.nodeURL, opts...)
```

**Considerations:**
- `grpc.NewClient` returns `*grpc.ClientConn` (same as Dial)
- Error handling remains the same
- Connection lifecycle management unchanged
- Backward compatible

### 4. Unused Field Removal

**Current:**
```go
type BaseDataPullerPlugin struct {
    // ... other fields
    retryCount        int  // UNUSED
    maxRetries        int
    // ... other fields
}
```

**Fix:**
- Remove `retryCount` field
- Verify no code references it
- Update any related initialization code

## Implementation Approach

### Phase 1: Error Handling
1. Fix Close() calls in grpc_puller.go
2. Fix Close() calls in websocket_jsonrpc_puller.go
3. Fix SetDeadline() calls in websocket_jsonrpc_puller.go
4. Fix Body.Close() in https_jsonrpc_puller.go
5. Fix Sscanf() in https_jsonrpc_puller.go
6. Fix Publish() in reorg_handler.go

### Phase 2: Type Safety
1. Update fmt.Sprintf format specifier in data_puller.go

### Phase 3: API Updates
1. Replace grpc.Dial with grpc.NewClient in grpc_puller.go

### Phase 4: Code Cleanup
1. Remove retryCount field from data_puller.go

### Phase 5: Verification
1. Run golangci-lint
2. Verify 0 issues
3. Run tests
4. Verify no regressions

## Error Handling Patterns

### Pattern 1: Deferred Cleanup
```go
defer func() {
    if err := resource.Close(); err != nil {
        logger.Warn("cleanup error", "error", err)
    }
}()
```

### Pattern 2: Explicit Error Return
```go
if err := operation(); err != nil {
    logger.Error("operation failed", "error", err)
    return err
}
```

### Pattern 3: Error Logging Only
```go
if err := operation(); err != nil {
    logger.Error("operation failed", "error", err)
    // Continue execution
}
```

## Testing Strategy

1. **Unit Tests**: Verify error handling doesn't break functionality
2. **Integration Tests**: Verify pullers still work correctly
3. **Linting Tests**: Verify golangci-lint passes
4. **Regression Tests**: Verify no new issues introduced

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| Breaking API changes | Low | High | Use backward-compatible updates |
| Error handling changes behavior | Low | Medium | Thorough testing |
| Unused field removal breaks code | Low | High | Verify no references |
| gRPC update incompatibility | Low | Medium | Test with current gRPC version |

## Success Metrics

- golangci-lint reports 0 issues
- All tests pass
- No performance degradation
- No new warnings introduced
- Code review approval
