# Tasks: golangci-lint Fixes

## Task 1: Fix Unchecked Error Returns in grpc_puller.go

**Issue**: `grpc_puller.go:301` - `p.conn.Close()` error not checked

**File**: `pkg/plugins/pullers/grpc_puller.go`

**Changes**:
1. Locate the Close() call at line 301
2. Add error handling:
   ```go
   if err := p.conn.Close(); err != nil {
       p.logger.Warn("failed to close gRPC connection", "error", err)
   }
   ```

**Verification**: Run `golangci-lint run ./pkg/plugins/pullers/grpc_puller.go`

---

## Task 2: Fix Deprecated gRPC Call in grpc_puller.go

**Issue**: `grpc_puller.go:284` - `grpc.Dial` is deprecated

**File**: `pkg/plugins/pullers/grpc_puller.go`

**Changes**:
1. Locate the `grpc.Dial` call at line 284
2. Replace with `grpc.NewClient`:
   ```go
   // OLD: conn, err := grpc.Dial(p.nodeURL, opts...)
   // NEW:
   conn, err := grpc.NewClient(p.nodeURL, opts...)
   ```

**Verification**: Run `golangci-lint run ./pkg/plugins/pullers/grpc_puller.go`

---

## Task 3: Fix Unchecked Error Returns in https_jsonrpc_puller.go

**Issues**:
- `https_jsonrpc_puller.go:358` - `httpResp.Body.Close()` error not checked
- `https_jsonrpc_puller.go:415` - `fmt.Sscanf()` error not checked

**File**: `pkg/plugins/pullers/https_jsonrpc_puller.go`

**Changes**:

### 3a: Fix Body.Close() at line 358
```go
defer func() {
    if err := httpResp.Body.Close(); err != nil {
        p.logger.Warn("failed to close response body", "error", err)
    }
}()
```

### 3b: Fix Sscanf() at line 415
```go
if _, err := fmt.Sscanf(hexStr, "%x", &result); err != nil {
    p.logger.Error("failed to parse hex string", "error", err, "hex_str", hexStr)
    return 0, err
}
```

**Verification**: Run `golangci-lint run ./pkg/plugins/pullers/https_jsonrpc_puller.go`

---

## Task 4: Fix Printf Format Type Mismatch in data_puller.go

**Issue**: `data_puller.go:294` - `fmt.Sprintf` format %s has arg event.LogIndex of wrong type uint

**File**: `pkg/plugins/pullers/data_puller.go`

**Changes**:
1. Locate the `GenerateEventHash` function
2. Update the format string from `%s` to `%d` for LogIndex:
   ```go
   // OLD: return fmt.Sprintf("%s:%d:%s:%d:%s", ...)
   // NEW:
   return fmt.Sprintf("%s:%d:%d:%s:%s",
       event.TransactionHash,
       event.BlockNumber,
       event.LogIndex,      // Changed from %s to %d
       event.ContractAddress,
       event.EventName,
   )
   ```

**Verification**: Run `golangci-lint run ./pkg/plugins/pullers/data_puller.go`

---

## Task 5: Remove Unused Field in data_puller.go

**Issue**: `data_puller.go:24` - field `retryCount` is unused

**File**: `pkg/plugins/pullers/data_puller.go`

**Changes**:
1. Locate the `BaseDataPullerPlugin` struct
2. Remove the `retryCount` field:
   ```go
   // REMOVE THIS LINE:
   // retryCount        int
   ```
3. Verify no code references `retryCount`
4. Update any initialization code if needed

**Verification**: 
- Run `golangci-lint run ./pkg/plugins/pullers/data_puller.go`
- Search for "retryCount" in the file to ensure it's not referenced

---

## Task 6: Fix Unchecked Error Returns in websocket_jsonrpc_puller.go

**Issues**:
- `websocket_jsonrpc_puller.go:330` - `p.conn.Close()` error not checked
- `websocket_jsonrpc_puller.go:350` - `p.conn.SetWriteDeadline()` error not checked
- `websocket_jsonrpc_puller.go:364` - `p.conn.SetReadDeadline()` error not checked

**File**: `pkg/plugins/pullers/websocket_jsonrpc_puller.go`

**Changes**:

### 6a: Fix Close() at line 330
```go
if err := p.conn.Close(); err != nil {
    p.logger.Warn("failed to close websocket connection", "error", err)
}
```

### 6b: Fix SetWriteDeadline() at line 350
```go
if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
    p.logger.Error("failed to set write deadline", "error", err)
    return err
}
```

### 6c: Fix SetReadDeadline() at line 364
```go
if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
    p.logger.Error("failed to set read deadline", "error", err)
    return err
}
```

**Verification**: Run `golangci-lint run ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`

---

## Task 7: Fix Unchecked Error Return in reorg_handler.go

**Issue**: `reorg_handler.go:155` - `h.eventBus.Publish()` error not checked

**File**: `pkg/plugins/pullers/reorg_handler.go`

**Changes**:
1. Locate the `Publish()` call at line 155
2. Add error handling:
   ```go
   if err := h.eventBus.Publish(ctx, "blockchain_reorg", reorgEvent); err != nil {
       h.logger.Error("failed to publish reorg event", "error", err)
       // Decide: return error or continue
       // For now, log and continue
   }
   ```

**Verification**: Run `golangci-lint run ./pkg/plugins/pullers/reorg_handler.go`

---

## Task 8: Final Verification

**Objective**: Verify all issues are resolved

**Steps**:
1. Run full linting check:
   ```bash
   golangci-lint run ./pkg/plugins/pullers/... --timeout=5m
   ```
2. Verify output shows 0 issues
3. Run tests:
   ```bash
   go test ./pkg/plugins/pullers/...
   ```
4. Verify all tests pass
5. Check for any new warnings

**Success Criteria**:
- golangci-lint reports 0 issues
- All tests pass
- No new warnings introduced

---

## Task 9: Documentation Update

**Objective**: Update progress documentation

**Steps**:
1. Create completion summary
2. Document all changes made
3. Update progress tracking
4. Create quick reference guide

**Files to Update**:
- `docs/progress/GOLANGCI_LINT_FIXES_FINAL_COMPLETION.md`
- `docs/guides/GOLANGCI_LINT_FIXES_QUICK_REFERENCE.md`

---

## Execution Order

1. Task 1: Fix grpc_puller.go Close()
2. Task 2: Fix grpc_puller.go deprecated call
3. Task 3: Fix https_jsonrpc_puller.go errors
4. Task 4: Fix data_puller.go format string
5. Task 5: Remove unused field
6. Task 6: Fix websocket_jsonrpc_puller.go errors
7. Task 7: Fix reorg_handler.go Publish()
8. Task 8: Final verification
9. Task 9: Documentation update

## Estimated Effort

- Task 1: 5 minutes
- Task 2: 5 minutes
- Task 3: 10 minutes
- Task 4: 5 minutes
- Task 5: 5 minutes
- Task 6: 15 minutes
- Task 7: 5 minutes
- Task 8: 10 minutes
- Task 9: 15 minutes

**Total**: ~75 minutes
