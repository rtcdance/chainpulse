# golangci-lint Fixes Specification

## Overview

This specification documents the work to resolve all golangci-lint compilation and linting errors in the ChainPulse blockchain data pullers package (`pkg/plugins/pullers`). The goal is to achieve a clean build with zero linting warnings.

## Current Status

**Phase**: Completion & Remaining Issues Resolution

**Completion**: ~90% - Major type errors fixed, minor linting issues remain

## Key Achievements

### Phase 1: Type System Fixes (COMPLETED)
- Fixed `TransactionHash` and `ContractAddress` type mismatches
- Updated all comparisons to use proper zero-value checks
- Added ethereum/common imports to all puller files
- Updated BlockchainEvent field types to match core model

### Phase 2: Interface Updates (COMPLETED)
- Updated DataPullerPlugin interface signatures
- Fixed logger method names (LogInfo → Info, etc.)
- Fixed metrics method signatures
- Updated EventBus.Publish() to include context.Context

### Phase 3: Remaining Issues (IN PROGRESS)
- 7 unchecked error returns (errcheck)
- 1 printf format type mismatch (govet)
- 1 deprecated gRPC call (staticcheck)
- 1 unused field (unused)

## Files Involved

### Core Files
- `pkg/plugins/pullers/data_puller.go` - Base puller implementation
- `pkg/plugins/pullers/grpc_puller.go` - gRPC data puller
- `pkg/plugins/pullers/https_jsonrpc_puller.go` - HTTPS JSON-RPC puller
- `pkg/plugins/pullers/websocket_jsonrpc_puller.go` - WebSocket JSON-RPC puller
- `pkg/plugins/pullers/reorg_handler.go` - Reorganization handler
- `pkg/core/plugin.go` - Plugin interface definitions
- `pkg/core/blockchain_models.go` - Blockchain data models

## Remaining Issues to Fix

### 1. Unchecked Error Returns (errcheck) - 7 issues
- `grpc_puller.go:301` - `p.conn.Close()` error not checked
- `https_jsonrpc_puller.go:358` - `httpResp.Body.Close()` error not checked
- `https_jsonrpc_puller.go:415` - `fmt.Sscanf()` error not checked
- `reorg_handler.go:155` - `h.eventBus.Publish()` error not checked
- `websocket_jsonrpc_puller.go:330` - `p.conn.Close()` error not checked
- `websocket_jsonrpc_puller.go:350` - `p.conn.SetWriteDeadline()` error not checked
- `websocket_jsonrpc_puller.go:364` - `p.conn.SetReadDeadline()` error not checked

### 2. Printf Format Type Mismatch (govet) - 1 issue
- `data_puller.go:294` - `fmt.Sprintf` format %s has arg event.LogIndex of wrong type uint

### 3. Deprecated gRPC Call (staticcheck) - 1 issue
- `grpc_puller.go:284` - `grpc.Dial` is deprecated, should use `NewClient` instead

### 4. Unused Field (unused) - 1 issue
- `data_puller.go:24` - field `retryCount` is unused

## Implementation Plan

### Task 1: Fix Unchecked Error Returns
- Add error handling for all Close() calls
- Add error handling for SetDeadline() calls
- Add error handling for Publish() call
- Add error handling for Sscanf() call

### Task 2: Fix Printf Format Type Mismatch
- Change format specifier from %s to %d for uint LogIndex

### Task 3: Fix Deprecated gRPC Call
- Replace `grpc.Dial` with `grpc.NewClient`

### Task 4: Remove Unused Field
- Remove `retryCount` field from BaseDataPullerPlugin

### Task 5: Verification
- Run golangci-lint to verify all issues are resolved
- Ensure no new issues are introduced
- Verify compilation succeeds

## Success Criteria

- [ ] All 7 errcheck issues resolved
- [ ] Printf format type mismatch fixed
- [ ] Deprecated gRPC call updated
- [ ] Unused field removed
- [ ] `golangci-lint run ./pkg/plugins/pullers/...` returns 0 issues
- [ ] All tests pass
- [ ] No new linting warnings introduced

## Related Documentation

- Previous fixes: `GOLANGCI_LINT_FIXES_COMPLETE.md`
- Build compilation: `BUILD_COMPILATION_FIXES_COMPLETE.md`
- Type system fixes: `COMPILATION_FIXES_SESSION_COMPLETE.md`
