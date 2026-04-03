# ChainPulse Copilot Instructions

## Project Context
ChainPulse is a Web3 blockchain event indexing system built with Go, supporting both monolithic and microservice deployment modes.

## Code Generation Guidelines

### Go Code
- Use Go 1.21+ syntax and features
- Follow standard Go naming conventions
- Prefer composition over inheritance
- Use interfaces for abstraction

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process %s: %w", item, err)
}
```

### Context Usage
```go
// Always pass context as first parameter
func Process(ctx context.Context, data *Data) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // process
}
```

### Testing
```go
// Use table-driven tests
func TestProcess(t *testing.T) {
    t.Parallel()
    tests := []struct {
        name string
        input string
        want string
        wantErr bool
    }{
        // cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // test
        })
    }
}
```

## Architecture Patterns

### DDD Layers
- **Domain**: Pure business logic, no dependencies
- **Application**: Use cases, orchestrates domain
- **Infrastructure**: External adapters, implementations

### Plugin Pattern
```go
type Plugin interface {
    Name() string
    Init(config Config) error
    Start(ctx context.Context) error
    Stop() error
}
```

## Web3 Specific

### Block Processing
- Always check for reorgs
- Use confirmation depth
- Handle missing blocks gracefully
- Implement idempotency

### Event Handling
- Validate event signature
- Decode with proper ABI
- Handle decode errors
- Track processing state

## Security

### Never
- Hardcode credentials
- Log sensitive data
- Trust user input
- Use string concatenation for SQL

### Always
- Validate inputs
- Use parameterized queries
- Mask sensitive logs
- Check permissions

## Performance

### Concurrency
- Use worker pools for parallel processing
- Implement proper shutdown
- Avoid goroutine leaks
- Use buffered channels appropriately

### Database
- Use connection pooling
- Implement batch operations
- Add proper indexes
- Use transactions for consistency

## File Headers

New Go files should follow:
```go
// Package <name> provides <brief description>
package <name>
```

No additional boilerplate or comments unless necessary.
