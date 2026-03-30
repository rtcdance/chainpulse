---
name: "code-style"
description: "Enforces Go code style conventions and best practices. Invoke when writing, editing, or reviewing Go code."
---

# Code Style Guidelines

## Purpose
Ensure all Go code follows consistent style conventions and best practices.

## When to Invoke
- Writing new Go code
- Editing existing Go code
- Reviewing code changes
- Refactoring code

## Go Code Conventions

### Naming
- **Packages**: lowercase, single word preferred (e.g., `indexer`, `puller`)
- **Types**: PascalCase for exported, camelCase for internal
- **Functions**: PascalCase for exported, camelCase for internal
- **Constants**: PascalCase or UPPER_SNAKE_CASE
- **Interfaces**: typically end with `-er` (e.g., `Reader`, `Indexer`)

### Code Organization
```
pkg/
├── core/           # Domain types and interfaces
├── plugins/        # Plugin implementations
├── services/       # Business logic
└── infrastructure/ # Infrastructure utilities
```

### Import Organization
```go
// Standard library
import (
    "context"
    "fmt"
)

// Third-party
import (
    "github.com/ethereum/go-ethereum"
    "go.uber.org/zap"
)

// Local packages
import (
    "chainpulse/pkg/core"
    "chainpulse/pkg/plugins"
)
```

### Error Handling
```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process block %d: %w", blockNum, err)
}

// Bad: Ignore or silently return
if err != nil {
    return err
}
```

### Comments
- No comments unless explicitly requested
- Exported functions may have brief doc comments if necessary

## Constraints
- DO NOT add unnecessary comments
- DO NOT use global variables
- DO NOT import unused packages
- ALWAYS handle errors explicitly
- ALWAYS use context for cancellation
