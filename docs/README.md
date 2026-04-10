# ChainPulse Documentation

## Quick Navigation

### Getting Started
- **[Developer Guide](guides/DEVELOPER_GUIDE.md)** - Development setup and guidelines
- **[API Documentation](guides/API_DOCUMENTATION.md)** - REST, gRPC, and WebSocket APIs

### Deployment & Operations
- **[Deployment Guide](guides/DEPLOYMENT_GUIDE.md)** - Deployment procedures
- **[Operations Guide](guides/OPERATIONS_GUIDE.md)** - Monitoring and maintenance
- **[Production Checklist](deployment/production-checklist.md)** - Pre-production verification

### Architecture
- **[Directory Structure](architecture/DIRECTORY_STRUCTURE.md)** - Project organization
- **[Migration Summary](architecture/MIGRATION_SUMMARY.md)** - Architecture improvements

### Testing
- **[E2E Testing Guide](e2e-testing/README.md)** - End-to-end testing framework
- **[Unit Test Standards](guides/UNIT_TEST_STANDARDS.md)** - Testing best practices
- **[Engineering Constraint Framework](guides/ENGINEERING_CONSTRAINT_FRAMEWORK.md)** - Spec + skills + workflow + micro-loop gates
- **Repository Hygiene** - `make repo-hygiene` (防止产物/二进制/过时目录进入仓库)

### Guides
- **[MQ Error Handling Recovery](guides/MQ_ERROR_HANDLING_RECOVERY.md)** - MQ recovery and operator guidance

### Archive
- **[Architecture v1](archive/ARCHITECTURE_v1.md)** - Historical blueprint reference

## Directory Structure

```
docs/
├── README.md                    # Documentation entrypoint
├── INDEX.md                     # Broad inventory index
├── architecture/                # Architecture and structure notes
├── archive/                     # Historical references and retired plans
├── deployment/                  # Deployment and go-live runbooks
├── e2e-testing/                 # E2E testing documentation
├── guides/                      # Active engineering guides
├── operations/                  # Governance and rollout operations docs
└── specs/                       # Design-review specs and decision records
```

## Documentation Standards

- Keep documentation concise and actionable
- Include code examples where applicable
- Update docs when making architectural changes
- Link related documents for better navigation
- Prefer `archive/` for historical references instead of leaving retired docs in the `docs/` root
