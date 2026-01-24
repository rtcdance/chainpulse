# Design: Final golangci-lint Fixes (January 18, 2026)

## Overview

This design addresses the final 10 golangci-lint violations blocking CI/CD integration. The fixes involve:
1. Creating custom types for context keys (SA1029)
2. Removing or properly using ineffectual assignments (ineffassign)
3. Removing unused struct fields (unused)
4. Enabling golangci-lint in GitHub Actions workflow

## Architecture

### SA1029 Fix Pattern

For each file with SA1029 violations, we'll:
1. Define a custom type at the package level: `type contextKey string`
2. Create a constant for each key: `const keyName contextKey = "keyName"`
3. Replace all `context.WithValue(ctx, "string", value)` with `context.WithValue(ctx, keyName, value)`
4. Replace all `ctx.Value("string")` with `ctx.Value(keyName)`

### ineffassign Fix Pattern

For each ineffectual assignment:
1. Analyze the code to determine if the variable is actually needed
2. If needed: use the variable in a meaningful way (e.g., assertion, validation)
3. If not needed: remove the assignment entirely
4. If it's a loop variable: consider if it should be used in the loop body

### unused Field Fix Pattern

For each unused field:
1. Check if the field is used anywhere in the codebase
2. If truly unused: remove it
3. If used via reflection or in tests: add a comment explaining the usage
4. If it's a placeholder for future use: remove it (add TODO if needed)

## Components and Interfaces

### Context Key Definitions

**File: pkg/observability/distributed_tracing.go**
```go
type contextKey string
const (
    traceIDKey contextKey = "traceID"
)
```

**File: pkg/plugins/api/rate_limiter.go**
```go
type contextKey string
const (
    rateLimitKey contextKey = "rateLimit"
)
```

**File: pkg/services/resilience/error_handler_property_test.go**
```go
type contextKey string
const (
    errorHandlerKey contextKey = "errorHandler"
)
```

## Data Models

No new data models required. We're using existing context.Context with custom key types.

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Context Keys Are Type-Safe

*For any* context key used in the system, it SHALL be defined as a custom type (not a built-in string type) to prevent collisions and ensure type safety.

**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: No Ineffectual Assignments

*For any* variable assignment in the codebase, if the variable is assigned, it SHALL be used in a meaningful way within the same scope or the assignment SHALL be removed.

**Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5**

### Property 3: No Unused Struct Fields

*For any* struct field declaration, it SHALL either be used within the struct's methods or be removed from the struct definition.

**Validates: Requirements 3.1, 3.2**

## Error Handling

- If a context key is used in multiple places, ensure all usages are updated consistently
- If removing a field breaks tests, update the tests accordingly
- If an assignment is part of error handling, ensure the error is still properly handled

## Testing Strategy

### Unit Tests

- Verify context values can be retrieved using custom key types
- Verify that removing unused fields doesn't break functionality
- Verify that assignments are used correctly

### Property-Based Tests

- Property 1: For all context operations, verify keys are custom types
- Property 2: For all variable assignments, verify they're used or removed
- Property 3: For all struct fields, verify they're used or removed

### Integration Tests

- Run golangci-lint on the entire codebase and verify no violations remain
- Run the full test suite to ensure no regressions
- Verify CI/CD pipeline executes golangci-lint successfully

## CI/CD Integration

Update `.github/workflows/test.yml` to include golangci-lint:

```yaml
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v3
  with:
    version: latest
    args: --timeout=5m
```

This ensures code quality is enforced on every commit.
