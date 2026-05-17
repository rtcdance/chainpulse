# Go Error Handling in ChainPulse

ChainPulse uses a centralized error handling strategy built on Go 1.13+ conventions. This guide explains the patterns and how to apply them.

## Core Error Types

All project errors are defined in [pkg/core/errors.go](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/pkg/core/errors.go):

```go
// Sentinel errors — constant values used with errors.Is
var (
    ErrInvalidBlock      = errors.New("invalid block")
    ErrChainIDMismatch   = errors.New("chain ID mismatch")
    ErrGasLimitExceeded  = errors.New("gas limit exceeded")
)
```

**Sentinel errors** are package-level `errors.New` values. Callers use `errors.Is` to check:

```go
if errors.Is(err, core.ErrInvalidBlock) {
    // skip this block
}
```

## Wrapping with %w

Always use `%w` when wrapping errors from subordinate calls. This preserves the error chain:

```go
// ✅ CORRECT: preserves error chain for errors.Is / errors.As
nonce, err := GenerateNonce()
if err != nil {
    return nil, fmt.Errorf("generate challenge nonce: %w", err)
}

// ❌ WRONG: loses the cause
return nil, fmt.Errorf("generate challenge nonce: %v", err)
```

With `%w`, callers can use:
- `errors.Is(err, ErrSomeSentinel)` — check against sentinel value
- `errors.As(err, &typedErr)` — extract a custom error type

## ChainPulse Error Classification

The project has a [resilience package](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/pkg/services/resilience/) that classifies errors:

| Category | Meaning | Retry? |
|----------|---------|--------|
| **Transient** | Network timeout, RPC 429 | Yes (with backoff) |
| **Permanent** | Validation failure, bad input | No (return immediately) |
| **Critical** | Database connection lost, corruption | No (open circuit breaker) |

```go
cls := errorHandler.ClassifyError(err)
switch cls {
case resilience.ErrorTransient:
    // retry with backoff
case resilience.ErrorPermanent:
    // log and skip
case resilience.ErrorCritical:
    // open circuit breaker
}
```

## Custom Error Types

For structured errors, define a type that implements the `error` interface:

```go
type BlockReorgError struct {
    BlockNumber uint64
    OldHash     common.Hash
    NewHash     common.Hash
}

func (e *BlockReorgError) Error() string {
    return fmt.Sprintf("block %d reorg: %s → %s", e.BlockNumber, e.OldHash, e.NewHash)
}
```

Callers extract with `errors.As`:

```go
var reorgErr *BlockReorgError
if errors.As(err, &reorgErr) {
    log.Printf("reorg at block %d", reorgErr.BlockNumber)
}
```

## Error Handling in ChainPulse Plugins

Every plugin (`Initialize → Start → Stop`) follows this pattern:

```go
func (p *Plugin) Initialize(config *core.Config) error {
    if config == nil {
        return fmt.Errorf("plugin initialize: %w", core.ErrNilConfig)
    }
    // ...
    if err := p.validateConfig(config); err != nil {
        return fmt.Errorf("plugin validate: %w", err)
    }
    return nil
}
```

## Best Practices

1. **Use sentinel errors** for well-known failure modes (`errors.Is`)
2. **Use custom types** for errors with structured context (`errors.As`)
3. **Always %w** when wrapping subordinate errors
4. **Use %s/%v** only for formatting string parameters (not errors)
5. **Never inspect err.Error() string** — use `errors.Is` / `errors.As`
6. **Classify errors early** — determine transient vs permanent at the boundary
7. **Log with key-value pairs**: `logger.Error("storage failed", "error", err, "block", blockNum)`

## Common Patterns

### Pattern 1: Collecting Errors

```go
var errs []error
for _, event := range events {
    if err := process(event); err != nil {
        errs = append(errs, fmt.Errorf("process event %s: %w", event.ID, err))
    }
}
if len(errs) > 0 {
    return fmt.Errorf("batch had %d failures: %v", len(errs), errors.Join(errs...))
}
```

### Pattern 2: Return Early

```go
func validate(event *core.BlockchainEvent) error {
    if event.Network == "" {
        return fmt.Errorf("%w: network is required", core.ErrInvalidEvent)
    }
    if event.BlockNumber == 0 {
        return fmt.Errorf("%w: block number is zero", core.ErrInvalidEvent)
    }
    return nil
}
```

### Pattern 3: Context Cancellation

```go
if err := ctx.Err(); err != nil {
    return fmt.Errorf("operation cancelled: %w", err)
}
```

## Reference

- Go blog: [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [pkg/core/errors.go](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/pkg/core/errors.go) — project sentinel errors
- [pkg/services/resilience/error_handler.go](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/pkg/services/resilience/error_handler.go) — error classification