---
name: "code-organization-placement"
description: "Enforce clean file organization and directory structure. Prevent random file placement and maintain navigability. Invoke when creating new files or directories, moving/refactoring code, adding generated code or artifacts, or implementing new features."
---

# Code Organization & File Placement

## Purpose
Enforce clean file organization and prevent random file placement that degrades codebase navigability.

## Trigger
- Creating new files or directories
- Moving/refactoring code
- Adding generated code or artifacts
- Implementing new features

## Must Do

### 1. Follow Directory Structure
```
pkg/
├── domain/          # Business logic, entities (no external deps)
├── application/     # Use cases, orchestration
├── adapters/        # External integrations (DB, RPC, API)
├── infrastructure/  # Cross-cutting (logging, metrics, config)
└── plugins/         # Swappable implementations

services/            # Microservice entry points
├── indexer/
├── api/
└── worker/

test/
├── contracts/       # Contract tests
├── helpers/         # Test utilities
└── performance/     # Benchmarks

scripts/             # Automation, not application code
docs/                # Documentation only
```

### 2. File Naming Rules
```go
// ✅ Good: Clear, specific names
pkg/adapters/rpc/ethereum_client.go
pkg/domain/block/repository.go
pkg/application/indexing/service.go

// ❌ Bad: Vague or misplaced
pkg/utils/stuff.go
helpers.go (at root)
temp_fix.go
```

### 3. Generated Code Placement
```
# Build artifacts
/bin/              # Compiled binaries
/dist/             # Release packages

# Generated code
/pkg/generated/    # Code generation output
  ├── mocks/       # Mock implementations
  ├── proto/       # Protobuf generated
  └── contracts/   # Smart contract bindings

# Never commit to root or random locations
❌ /output/
❌ /generated_*.go (at root)
❌ /pkg/domain/mock_*.go (mocks in domain)
```

### 4. Temporary Files
```bash
# Use proper temp directories
/tmp/chainpulse/   # Runtime temp files
/.cache/           # Build cache

# Add to .gitignore
*.tmp
*.bak
*_backup.*
```

## Exit Criteria
- [ ] New files follow pkg/domain/application/adapters structure
- [ ] No files at repository root (except standard: README, Makefile, go.mod)
- [ ] Generated code in `/pkg/generated/` or `/bin/`
- [ ] Test files colocated with code (`*_test.go`) or in `/test/`
- [ ] No `utils/`, `helpers/`, `common/` dumping grounds
- [ ] `.gitignore` updated for new generated artifacts

## Anti-Patterns
- ❌ Creating `utils/` or `helpers/` packages
- ❌ Placing files at root because "it's temporary"
- ❌ Generated code mixed with hand-written code
- ❌ Test utilities scattered across packages
- ❌ Vague names like `manager.go`, `handler.go` without context

## Enforcement
```bash
# Pre-commit hook check
scripts/check-file-organization.sh

# Fails if:
# - Files at root (except whitelist)
# - utils/ or helpers/ directories in pkg/
# - Generated code outside /pkg/generated/
```

## References
- `docs/ARCHITECTURE.md` - Layer boundaries
- `.gitignore` - Artifact exclusions
- `scripts/check-file-organization.sh` - Validation script
