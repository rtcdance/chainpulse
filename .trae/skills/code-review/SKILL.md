---
name: "code-review"
description: "Reviews code for quality, bugs, and improvements. Invoke after completing significant code changes or before merging."
---

# Code Review Guidelines

## Purpose
Ensure code quality through systematic review process.

## When to Invoke
- After completing significant code changes
- Before merging pull requests
- When user requests code review
- After implementing new features

## Review Checklist

### Correctness
- [ ] Logic is correct and handles edge cases
- [ ] Error handling is comprehensive
- [ ] No nil pointer dereferences
- [ ] No race conditions

### Code Quality
- [ ] Follows project code style
- [ ] No code duplication
- [ ] Functions are focused and small
- [ ] Clear naming conventions

### Performance
- [ ] No unnecessary allocations
- [ ] Efficient data structures
- [ ] Proper use of goroutines
- [ ] Connection pooling where appropriate

### Security
- [ ] No hardcoded secrets
- [ ] Input validation
- [ ] Proper authentication/authorization
- [ ] No SQL injection vulnerabilities

### Testing
- [ ] Unit tests for new functionality
- [ ] Tests cover edge cases
- [ ] Integration tests where needed

## Review Output Format

```markdown
## Code Review Summary

### ✅ Strengths
- <list positive aspects>

### ⚠️ Issues Found
- **[Severity]**: <description>
  - Location: <file:line>
  - Suggestion: <fix>

### 💡 Suggestions
- <optional improvements>
```

## Constraints
- ALWAYS provide actionable feedback
- ALWAYS explain the reasoning
- DO NOT approve code with critical issues
- PRIORITIZE security and correctness over style
