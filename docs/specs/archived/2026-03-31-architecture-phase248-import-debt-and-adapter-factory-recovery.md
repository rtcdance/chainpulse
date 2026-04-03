# Phase 248 - Import Debt And Adapter Factory Recovery

## Status
Status: Approved

## Why
- After switching away from the rollout/control line, the highest-value new
  work was repository health:
  remove the recurring test and import debt that kept blocking broader package
  validation.
- The first visible layer was a small cluster of stale module-path imports.
- After removing those stale imports, the next blocking layer became
  `adapter_factory`, which was still written against older plugin constructor
  names and interface assumptions.

## Scope
- Fix the stale module-path imports that no longer match `module chainpulse`.
- Recover `pkg/infrastructure/deployment/adapter_factory.go` into a minimal
  truthful state that compiles against the current core plugin contracts.

## Implementation
- Update stale imports from `github.com/chainpulse/chainpulse/...` to
  `chainpulse/...` in:
  - `pkg/plugins/mq/memory_mq.go`
  - `pkg/plugins/cache/inmemory_cache.go`
  - `pkg/plugins/database/mock_db.go`
  - `pkg/infrastructure/deployment/adapter_factory.go`
- Recover `adapter_factory` by:
  - keeping currently compatible factories available
  - returning explicit errors for plugin choices that no longer satisfy the
    current core plugin interfaces
- Validate the recovered deployment package directly.

## Validation
- Run `pkg/infrastructure/deployment` tests.
- Run `pkg/plugins/cache` tests.

## Exit Criteria
- The stale module-path import cluster is removed.
- `pkg/infrastructure/deployment` no longer fails at compile time because of
  outdated constructor names or incompatible direct returns.
