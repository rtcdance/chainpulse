# Checklist: golangci-lint Fixes

## Pre-Implementation

- [ ] Read README.md for overview
- [ ] Read requirements.md for acceptance criteria
- [ ] Read design.md for technical approach
- [ ] Read tasks.md for detailed instructions
- [ ] Verify current state: `golangci-lint run ./pkg/plugins/pullers/...`
- [ ] Backup current code (git commit)

## Task 1: Fix grpc_puller.go Close()

- [ ] Open `pkg/plugins/pullers/grpc_puller.go`
- [ ] Locate line 301: `p.conn.Close()`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/grpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/grpc_puller.go`
- [ ] Confirm issue resolved

## Task 2: Fix grpc_puller.go Deprecated Call

- [ ] Open `pkg/plugins/pullers/grpc_puller.go`
- [ ] Locate line 284: `grpc.Dial`
- [ ] Replace with `grpc.NewClient`
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/grpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/grpc_puller.go`
- [ ] Confirm issue resolved

## Task 3: Fix https_jsonrpc_puller.go Body.Close()

- [ ] Open `pkg/plugins/pullers/https_jsonrpc_puller.go`
- [ ] Locate line 358: `httpResp.Body.Close()`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/https_jsonrpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/https_jsonrpc_puller.go`
- [ ] Confirm issue resolved

## Task 4: Fix https_jsonrpc_puller.go Sscanf()

- [ ] Open `pkg/plugins/pullers/https_jsonrpc_puller.go`
- [ ] Locate line 415: `fmt.Sscanf`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/https_jsonrpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/https_jsonrpc_puller.go`
- [ ] Confirm issue resolved

## Task 5: Fix data_puller.go Format String

- [ ] Open `pkg/plugins/pullers/data_puller.go`
- [ ] Locate line 294: `fmt.Sprintf` in `GenerateEventHash`
- [ ] Change format specifier from %s to %d for LogIndex
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/data_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/data_puller.go`
- [ ] Confirm issue resolved

## Task 6: Remove Unused Field

- [ ] Open `pkg/plugins/pullers/data_puller.go`
- [ ] Locate `retryCount` field in `BaseDataPullerPlugin` struct
- [ ] Search for all references: `grep -n "retryCount" pkg/plugins/pullers/data_puller.go`
- [ ] Verify no references exist
- [ ] Remove the field
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/data_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/data_puller.go`
- [ ] Confirm issue resolved

## Task 7: Fix websocket_jsonrpc_puller.go Close()

- [ ] Open `pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Locate line 330: `p.conn.Close()`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Confirm issue resolved

## Task 8: Fix websocket_jsonrpc_puller.go SetWriteDeadline()

- [ ] Open `pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Locate line 350: `p.conn.SetWriteDeadline`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Confirm issue resolved

## Task 9: Fix websocket_jsonrpc_puller.go SetReadDeadline()

- [ ] Open `pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Locate line 364: `p.conn.SetReadDeadline`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/websocket_jsonrpc_puller.go`
- [ ] Confirm issue resolved

## Task 10: Fix reorg_handler.go Publish()

- [ ] Open `pkg/plugins/pullers/reorg_handler.go`
- [ ] Locate line 155: `h.eventBus.Publish`
- [ ] Add error handling
- [ ] Verify syntax: `go build ./pkg/plugins/pullers/reorg_handler.go`
- [ ] Run linter: `golangci-lint run ./pkg/plugins/pullers/reorg_handler.go`
- [ ] Confirm issue resolved

## Final Verification

- [ ] Run full linter check: `golangci-lint run ./pkg/plugins/pullers/... --timeout=5m`
- [ ] Verify 0 issues reported
- [ ] Run tests: `go test ./pkg/plugins/pullers/... -v`
- [ ] Verify all tests pass
- [ ] Check for any new warnings
- [ ] Review all changes: `git diff`

## Code Quality

- [ ] All error handling follows consistent patterns
- [ ] Logging is appropriate for each error
- [ ] No breaking changes to public APIs
- [ ] Code is readable and maintainable
- [ ] Comments are clear and helpful

## Documentation

- [ ] Update progress file: `docs/progress/GOLANGCI_LINT_FIXES_FINAL_COMPLETION.md`
- [ ] Create quick reference: `docs/guides/GOLANGCI_LINT_FIXES_QUICK_REFERENCE.md`
- [ ] Document all changes made
- [ ] Update README if needed

## Sign-Off

- [ ] All tasks completed
- [ ] All tests passing
- [ ] Code review approved
- [ ] Ready for merge
- [ ] Progress documented

## Notes

Use this space to track any issues or notes:

```
[Add notes here]
```

## Completion Date

**Started**: _______________  
**Completed**: _______________  
**Total Time**: _______________  

## Sign-Off

**Completed By**: _______________  
**Reviewed By**: _______________  
**Date**: _______________
