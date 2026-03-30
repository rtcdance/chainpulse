# ChainPulse Architecture

**Status**: Active | **Last Updated**: 2026-03-30

## Quick Reference

### Layer Structure
```
pkg/
├── domain/          # Business logic (zero external deps)
├── application/     # Use cases, orchestration
├── adapters/        # External integrations (DB, RPC, API)
├── infrastructure/  # Cross-cutting (logging, metrics, config)
└── plugins/         # Swappable implementations
```

### Key Principles
1. **Domain Purity**: No external dependencies in `pkg/domain/`
2. **Dependency Inversion**: Adapters implement domain interfaces
3. **Plugin Swapping**: In-memory ↔ Production (same interface)
4. **Observability First**: Metrics, tracing, health checks built-in

## Detailed Documentation

- **Directory Structure**: `docs/architecture/DIRECTORY_STRUCTURE.md`
- **Deployment Modes**: `docs/guides/DEPLOYMENT_GUIDE.md`
- **Plugin System**: `pkg/plugins/README.md`
- **Observability**: `pkg/observability/README.md`
- **Testing Strategy**: `docs/TESTING.md`

## Architecture Decisions

See `.codex/skills/` for enforced patterns:
- `web3-go-architecture-guardrails` - Layer boundaries
- `adapter-contract-testing` - Plugin contracts
- `observability-slo-gates` - Telemetry requirements

---

**Note**: Original 907-line document archived to `docs/archive/ARCHITECTURE_v1.md`
