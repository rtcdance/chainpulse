---
name: "project-structure"
description: "Enforces proper project directory structure and file placement. Invoke when creating new files, directories, or organizing code."
---

# Project Structure Guidelines

## Purpose
Ensure all files are placed in appropriate directories following project conventions.

## When to Invoke
- Creating new files
- Creating new directories
- Organizing code
- Refactoring structure

## Directory Structure

```
chainpulse/
├── cmd/                        # Application entry points
│   ├── monolithic/             # Single binary entry
│   └── microservices/          # Microservice entries
│
├── pkg/                        # Public packages (importable)
│   ├── core/                   # Domain types & interfaces
│   ├── plugins/                # Plugin implementations
│   ├── services/               # Business services
│   └── infrastructure/         # Infrastructure utilities
│
├── docs/                       # Documentation
├── scripts/                    # Utility scripts
├── test/                       # Test suites
├── proto/                      # Protocol definitions
├── docker/                     # Docker configuration
└── .trae/                      # AI configuration
```

## File Placement Rules

### Go Source Files

| File Type | Location | Example |
|-----------|----------|---------|
| Entry point | `cmd/*/main.go` | `cmd/monolithic/chainpulse/main.go` |
| Domain types | `pkg/core/*/` | `pkg/core/block/block.go` |
| Interfaces | `pkg/core/*/` | `pkg/core/indexer/interface.go` |
| Implementations | `pkg/plugins/*/` | `pkg/plugins/cache/redis/cache.go` |
| Business logic | `pkg/services/*/` | `pkg/services/indexer/service.go` |
| Utilities | `pkg/infrastructure/*/` | `pkg/infrastructure/logger/logger.go` |
| Tests | Same directory | `pkg/core/block/block_test.go` |

### Documentation Files

| File Type | Location | Example |
|-----------|----------|---------|
| README | Root or subdirectory | `README.md`, `docs/README.md` |
| Guides | `docs/guides/` | `docs/guides/DEVELOPER_GUIDE.md` |
| Architecture | `docs/architecture/` | `docs/architecture/ARCHITECTURE.md` |
| API docs | `docs/api/` | `docs/api/REST_API.md` |

### Configuration Files

| File Type | Location | Example |
|-----------|----------|---------|
| Project config | Root | `go.mod`, `Makefile` |
| Docker | `docker/` | `docker/Dockerfile` |
| CI/CD | `.github/` | `.github/workflows/ci.yml` |
| AI config | `.trae/` | `.trae/ai-config.json` |

### Script Files

| File Type | Location | Example |
|-----------|----------|---------|
| Build scripts | `scripts/build/` | `scripts/build/build.sh` |
| Deploy scripts | `scripts/deploy/` | `scripts/deploy/k8s.sh` |
| Dev scripts | `scripts/dev/` | `scripts/dev/setup.sh` |

## Naming Conventions

### Directories
- Use lowercase
- Use hyphens for multi-word names
- Be descriptive but concise

### Files
- Use snake_case for Go files
- Match package name for main file
- Use `_test.go` suffix for tests

## Forbidden Placements

### Never Place Files Here

| Location | Reason |
|----------|--------|
| Root directory (except configs) | Clutters project root |
| `pkg/` root | Should be in subpackages |
| `cmd/` subdirectories (except main.go) | Entry points only |
| Random locations | Breaks conventions |

### Examples of Forbidden Placements

```
❌ /chainpulse/utils.go           # Should be in pkg/infrastructure/
❌ /chainpulse/service.go         # Should be in pkg/services/
❌ /chainpulse/handler.go         # Should be in pkg/plugins/api/
❌ /chainpulse/docs.md            # Should be in docs/
❌ /chainpulse/test.go            # Should be in test/
```

## File Creation Checklist

Before creating a new file, ask:

1. **Is this an entry point?** → `cmd/`
2. **Is this a domain type?** → `pkg/core/`
3. **Is this an implementation?** → `pkg/plugins/`
4. **Is this business logic?** → `pkg/services/`
5. **Is this infrastructure?** → `pkg/infrastructure/`
6. **Is this documentation?** → `docs/`
7. **Is this a script?** → `scripts/`
8. **Is this a test?** → Same directory or `test/`

## Constraints

- ALWAYS follow the directory structure
- ALWAYS place files in appropriate locations
- NEVER create files in root directory (except configs)
- NEVER create random directories
- NEVER mix concerns in same directory
- ALWAYS use consistent naming
