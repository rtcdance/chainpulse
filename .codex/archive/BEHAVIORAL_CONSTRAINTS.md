# Codex Behavioral Constraints

**Purpose**: Define hard constraints on AI coding behavior to prevent common pitfalls.

## Code Generation Constraints

### 1. Minimal Implementation Only
```
❌ FORBIDDEN:
- Speculative features ("might need this later")
- Premature abstractions (helpers for single use)
- Over-engineering (config for hardcoded values)
- Defensive code for impossible scenarios

✅ REQUIRED:
- Solve ONLY the stated requirement
- Add abstraction when 3rd usage appears
- Hardcode until variability proven
```

### 2. No Unsolicited "Improvements"
```
❌ FORBIDDEN when fixing bug X:
- Refactoring surrounding code
- Adding logging to unrelated functions
- Updating comments in other files
- "Cleaning up" nearby code style

✅ REQUIRED:
- Change ONLY what's needed for X
- Separate PR for cleanups
```

### 3. Test Discipline
```
❌ FORBIDDEN:
- Auto-adding tests unless requested
- Modifying existing tests without reason
- Adding test utilities "for future use"

✅ REQUIRED:
- Add tests ONLY when explicitly asked
- Modify tests ONLY when behavior changes
- Keep test scope minimal
```

### 4. Dependency Hygiene
```
❌ FORBIDDEN:
- Adding libraries without approval
- Upgrading dependencies opportunistically
- Using heavy libs for simple tasks

✅ REQUIRED:
- Ask before adding dependencies
- Use stdlib when sufficient
- Justify external dependencies
```

## Code Quality Constraints

### 5. Error Handling Realism
```
❌ FORBIDDEN:
- Wrapping every error in 5 layers
- Validating internal function contracts
- Defensive checks for "impossible" states

✅ REQUIRED:
- Validate at boundaries only
- Trust internal contracts
- Return errors, don't wrap excessively
```

### 6. Comment Discipline
```
❌ FORBIDDEN:
- Adding docstrings to unchanged functions
- Explaining obvious code
- TODO comments without tickets

✅ REQUIRED:
- Comment ONLY non-obvious logic
- Update comments ONLY when changing code
- Link TODOs to issues
```

### 7. Configuration Realism
```
❌ FORBIDDEN:
- Making everything configurable
- Feature flags for single-use code
- Environment variables "for flexibility"

✅ REQUIRED:
- Hardcode until 2nd use case
- Config only for deployment variance
- No flags for unshipped features
```

## Web3-Specific Constraints

### 8. RPC Call Efficiency
```
❌ FORBIDDEN:
- Fetching full blocks when only need headers
- Per-block loops instead of filters
- Redundant calls in same function

✅ REQUIRED:
- Batch calls where possible
- Use filters over polling
- Cache immutable data
```

### 9. Gas Cost Awareness
```
❌ FORBIDDEN:
- Unbounded log queries
- Fetching ancient blocks without pagination
- No rate limiting on RPC calls

✅ REQUIRED:
- Paginate historical queries
- Respect RPC provider limits
- Track call counts in metrics
```

### 10. Reorg Safety
```
❌ FORBIDDEN:
- Assuming blocks are final
- No rollback handling
- Ignoring uncle blocks

✅ REQUIRED:
- Check finality before trusting
- Implement rollback logic
- Handle reorg scenarios
```

## Enforcement

**Pre-commit Check**:
```bash
# Reject if:
# - New files without trigger in active skill
# - Dependencies added without DEPENDENCY_APPROVAL.md entry
# - Tests modified without behavior change
# - Comments added to unchanged code
```

**Code Review Gate**:
- Reviewer asks: "What skill triggered this change?"
- If answer is vague, reject for scope creep

## Violation Examples

**❌ Bad PR**: "Fix indexer bug + refactor logger + add metrics"
- **Why**: 3 unrelated changes

**✅ Good PR**: "Fix indexer off-by-one error"
- **Why**: Single, focused change

**❌ Bad Code**:
```go
// Added "for future flexibility"
type Config struct {
    MaxRetries int
    RetryDelay time.Duration
    UseCache   bool
}
```

**✅ Good Code**:
```go
// Hardcoded until 2nd use case
const maxRetries = 3
```

## Summary

**Golden Rule**: Write the MINIMUM code to solve the STATED problem. No speculation, no "improvements", no premature abstraction.
