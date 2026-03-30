# ChainPulse Architecture

> **Status**: This document is DEPRECATED. See individual focused docs below.
> **Last Updated**: 2026-03-30

## Current Architecture References

For up-to-date architecture information, see:

- **Layer Boundaries**: `docs/architecture/DIRECTORY_STRUCTURE.md`
- **Deployment Modes**: `docs/guides/DEPLOYMENT_GUIDE.md`
- **Plugin System**: `pkg/plugins/README.md`
- **Observability**: `pkg/observability/README.md`

## Quick Reference

### Directory Structure
```
pkg/
├── domain/          # Business logic (no external deps)
├── application/     # Use cases
├── adapters/        # External integrations
├── infrastructure/  # Cross-cutting concerns
└── plugins/         # Swappable implementations
```

### Key Principles
1. Domain layer has zero external dependencies
2. Adapters implement domain interfaces
3. Infrastructure provides cross-cutting concerns
4. Plugins are swappable (in-memory ↔ production)

---

## Historical Content (Archived)

<details>
<summary>Original 907-line architecture document (click to expand)</summary>

_Content moved to: `docs/archive/ARCHITECTURE_v1.md`_

</details>

## Migration Notes

This document was split into focused docs to improve:
- AI context efficiency (smaller, targeted docs)
- Maintainability (single responsibility)
- Discoverability (clear file names)

See `docs/architecture/MIGRATION_SUMMARY.md` for details.
