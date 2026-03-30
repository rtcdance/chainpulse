---
name: "git-commit"
description: "Enforces conventional commit message format. Invoke when creating git commits or user asks about commit conventions."
---

# Git Commit Guidelines

## Purpose
Maintain clean, meaningful git history through conventional commits.

## When to Invoke
- Creating git commits
- Writing commit messages
- User asks about commit conventions

## Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

## Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `style` | Code style changes (formatting) |
| `refactor` | Code refactoring |
| `test` | Adding/modifying tests |
| `chore` | Maintenance tasks |
| `perf` | Performance improvements |

## Scopes

| Scope | Description |
|-------|-------------|
| `puller` | Data puller service |
| `indexer` | Indexer service |
| `api` | API service |
| `mq` | Message queue |
| `db` | Database |
| `cache` | Caching layer |
| `reorg` | Reorg handling |
| `config` | Configuration |

## Examples

### Feature
```
feat(puller): add WebSocket support for real-time data

- Implement WebSocket connection manager
- Add automatic reconnection logic
- Support multiple chain endpoints
```

### Bug Fix
```
fix(indexer): handle reorg correctly during high load

The indexer was not properly handling reorg events when
processing multiple blocks concurrently. This fix adds
proper locking and state management.

Fixes #123
```

### Breaking Change
```
feat(api)!: change API response format

BREAKING CHANGE: The API response format has changed from
v1 to v2. See migration guide in docs.
```

## Constraints
- ALWAYS use conventional commit format
- Keep subject line under 72 characters
- Use imperative mood in subject
- DO NOT commit directly to main branch
- DO NOT include secrets in commits
