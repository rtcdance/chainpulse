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

### Guides
- **[MQ Error Handling Recovery](guides/MQ_ERROR_HANDLING_RECOVERY.md)** - MQ recovery and operator guidance

### Planning
- **[Planning Index](planning/README.md)** - Active planning and AI workflow artifacts
- **[Execution Plan](planning/EXECUTION_PLAN.md)** - Historical restructuring plan and rationale

### Archive
- **[Architecture v1](archive/ARCHITECTURE_v1.md)** - Historical blueprint reference
- **[Error Handling Guide (Archived)](archive/ERROR_HANDLING_GUIDE.md)** - Historical error handling reference
- **[Resilience Patterns (Archived)](archive/RESILIENCE_PATTERNS_GUIDE.md)** - Historical resilience reference
- **[Microservices Implementation (Archived)](archive/MICROSERVICES_IMPLEMENTATION_GUIDE.md)** - Historical implementation guide

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
├── planning/                    # Active planning / AI workflow artifacts
└── specs/                       # Design-review specs and decision records
```

## Documentation Standards

- Keep documentation concise and actionable
- Include code examples where applicable
- Update docs when making architectural changes
- Link related documents for better navigation
- Prefer `archive/` for historical references instead of leaving retired docs in the `docs/` root
