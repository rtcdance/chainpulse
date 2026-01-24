# Getting Started: golangci-lint Fixes

## Quick Overview

This spec guides you through fixing the remaining 10 linting issues in the `pkg/plugins/pullers` package. The work is straightforward and involves:

1. Adding error handling for I/O operations
2. Fixing a printf format specifier
3. Updating a deprecated gRPC call
4. Removing an unused field

## Current Status

```
golangci-lint run ./pkg/plugins/pullers/...

10 issues:
* errcheck: 7
* govet: 1
* staticcheck: 1
* unused: 1
```

## Quick Start

### Step 1: Understand the Issues

Run golangci-lint to see the current state:

```bash
golangci-lint run ./pkg/plugins/pullers/... --timeout=5m
```

### Step 2: Review the Tasks

Read `tasks.md` for detailed instructions on each fix.

### Step 3: Execute Fixes

Follow the tasks in order:

1. **grpc_puller.go** - Fix Close() and deprecated call
2. **https_jsonrpc_puller.go** - Fix Body.Close() and Sscanf()
3. **data_puller.go** - Fix format string and remove unused field
4. **websocket_jsonrpc_puller.go** - Fix Close() and SetDeadline() calls
5. **reorg_handler.go** - Fix Publish() error handling

### Step 4: Verify

After each fix, run:

```bash
golangci-lint run ./pkg/plugins/pullers/... --timeout=5m
```

### Step 5: Final Check

When all issues are fixed:

```bash
golangci-lint run ./pkg/plugins/pullers/... --timeout=5m
# Should show: 0 issues
```

## Key Files

| File | Issues | Priority |
|------|--------|----------|
| `pkg/plugins/pullers/grpc_puller.go` | 2 | High |
| `pkg/plugins/pullers/https_jsonrpc_puller.go` | 2 | High |
| `pkg/plugins/pullers/data_puller.go` | 2 | High |
| `pkg/plugins/pullers/websocket_jsonrpc_puller.go` | 3 | High |
| `pkg/plugins/pullers/reorg_handler.go` | 1 | High |

## Common Patterns

### Pattern 1: Error Handling for Close()

```go
// For deferred cleanup
defer func() {
    if err := resource.Close(); err != nil {
        logger.Warn("cleanup error", "error", err)
    }
}()

// For explicit close
if err := resource.Close(); err != nil {
    logger.Error("close error", "error", err)
    return err
}
```

### Pattern 2: Error Handling for SetDeadline()

```go
if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
    logger.Error("failed to set deadline", "error", err)
    return err
}
```

### Pattern 3: Error Handling for Sscanf()

```go
if _, err := fmt.Sscanf(hexStr, "%x", &result); err != nil {
    logger.Error("failed to parse hex", "error", err)
    return 0, err
}
```

## Troubleshooting

### Issue: golangci-lint not found

```bash
# Install golangci-lint
brew install golangci-lint

# Or use
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Issue: Still seeing errors after fixes

1. Verify the line numbers match
2. Check for multiple occurrences
3. Run with verbose output: `golangci-lint run -v ./pkg/plugins/pullers/...`

### Issue: New errors after fixes

1. Check for syntax errors
2. Verify imports are correct
3. Run `go build ./pkg/plugins/pullers/...`

## Testing

After fixes, run tests:

```bash
go test ./pkg/plugins/pullers/... -v
```

## Documentation

- **README.md** - Overview and status
- **requirements.md** - User stories and requirements
- **design.md** - Technical design and patterns
- **tasks.md** - Detailed task instructions
- **SUMMARY.md** - Issue summary table
- **GETTING_STARTED.md** - This file

## Next Steps

1. Read `tasks.md` for detailed instructions
2. Execute fixes in order
3. Verify with golangci-lint after each fix
4. Run tests to ensure no regressions
5. Update progress documentation

## Estimated Time

- Reading: 10 minutes
- Execution: 60 minutes
- Verification: 10 minutes
- **Total: ~80 minutes**

## Success Indicators

✅ golangci-lint reports 0 issues  
✅ All tests pass  
✅ No new warnings introduced  
✅ Code review approved  

## Questions?

Refer to:
- `design.md` for technical details
- `tasks.md` for step-by-step instructions
- `requirements.md` for acceptance criteria
