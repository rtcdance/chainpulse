# Summary: golangci-lint Fixes

## Overview

This spec addresses the final 10 linting issues remaining in the `pkg/plugins/pullers` package after major type system fixes were completed.

## Issues to Fix

| # | File | Line | Issue | Type | Priority |
|---|------|------|-------|------|----------|
| 1 | grpc_puller.go | 301 | `p.conn.Close()` error not checked | errcheck | High |
| 2 | grpc_puller.go | 284 | `grpc.Dial` is deprecated | staticcheck | High |
| 3 | https_jsonrpc_puller.go | 358 | `httpResp.Body.Close()` error not checked | errcheck | High |
| 4 | https_jsonrpc_puller.go | 415 | `fmt.Sscanf()` error not checked | errcheck | High |
| 5 | data_puller.go | 294 | Printf format %s has arg of wrong type uint | govet | High |
| 6 | data_puller.go | 24 | Field `retryCount` is unused | unused | Medium |
| 7 | websocket_jsonrpc_puller.go | 330 | `p.conn.Close()` error not checked | errcheck | High |
| 8 | websocket_jsonrpc_puller.go | 350 | `p.conn.SetWriteDeadline()` error not checked | errcheck | High |
| 9 | websocket_jsonrpc_puller.go | 364 | `p.conn.SetReadDeadline()` error not checked | errcheck | High |
| 10 | reorg_handler.go | 155 | `h.eventBus.Publish()` error not checked | errcheck | High |

## Solution Summary

### Error Handling (7 issues)
- Add proper error checking for all I/O operations
- Use appropriate logging for cleanup operations
- Return errors for critical operations

### Type Safety (1 issue)
- Fix printf format specifier from %s to %d for uint LogIndex

### API Updates (1 issue)
- Replace deprecated `grpc.Dial` with `grpc.NewClient`

### Code Cleanup (1 issue)
- Remove unused `retryCount` field

## Implementation Strategy

1. **Systematic Approach**: Fix issues by file and type
2. **Error Handling**: Use consistent patterns for error checking
3. **Testing**: Verify each fix with golangci-lint
4. **Verification**: Final comprehensive check

## Expected Outcome

- ✅ 0 linting issues in `pkg/plugins/pullers`
- ✅ All tests passing
- ✅ Clean, maintainable code
- ✅ Follows Go best practices

## Files Modified

- `pkg/plugins/pullers/data_puller.go`
- `pkg/plugins/pullers/grpc_puller.go`
- `pkg/plugins/pullers/https_jsonrpc_puller.go`
- `pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- `pkg/plugins/pullers/reorg_handler.go`

## Related Specs

- Previous: `build-compilation-fixes` - Major type system fixes
- Related: `web3-indexer-enhancements` - Overall indexer improvements
- Related: `chainpulse-enterprise-refactor` - Enterprise architecture

## Success Criteria

- [ ] All 10 issues resolved
- [ ] golangci-lint reports 0 issues
- [ ] All tests pass
- [ ] No new warnings introduced
- [ ] Code review approved
