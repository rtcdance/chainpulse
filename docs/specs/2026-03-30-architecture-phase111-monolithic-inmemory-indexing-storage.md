# Phase 111 Monolithic In-Memory Indexing Storage

## Title
Phase 111 - Add real in-memory storage wiring for monolithic indexing path

## Type
- architecture
- indexing

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
platform-team

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `pkg/adapters/indexing/monolithic_memory_storage.go`
- `pkg/adapters/indexing/monolithic_memory_storage_test.go`
- `pkg/application/bootstrap/indexing_storage.go`
- `pkg/application/bootstrap/indexing_storage_test.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/specs/2026-03-30-architecture-phase110-legacy-backed-runtime-sink-adapter.md`

## Context
The monolithic entrypoint now hosts a shared runtime and forwards real event
batches into it, but the legacy indexing path still builds `DefaultChainIndexer`
with `nil` database and cache dependencies.

## Problem Statement
Without real storage plugins on the monolithic indexing path, legacy indexing
cannot persist events safely and the new legacy-backed runtime sink still
cannot be adopted by the monolithic runtime wiring.

## Scope
- Add in-memory database and cache adapters that satisfy the current
  `core.DatabasePlugin` and `core.CachePlugin` contracts used by the indexing
  layer.
- Add a bootstrap helper that initializes and starts those adapters for
  monolithic mode.
- Wire monolithic chain indexers to the new in-memory storage adapters.
- Stop those adapters during monolithic graceful shutdown.
- Add focused unit tests for adapters and bootstrap helper behavior.

## Non-Goals
- No switch to legacy-backed shared runtime sink in this phase.
- No external database or redis wiring in this phase.
- No microservice entrypoint changes in this phase.

## Selected Approach
- Use in-memory adapters as the first real storage path for monolithic debug
  mode.
- Keep adapters small and faithful to the current indexing contracts.
- Keep bootstrap responsible for lifecycle wiring only.

## Data / Contract Impact
- Monolithic indexing path now has real storage side effects in memory.
- No external API or deployment contract changes in this phase.

## Observability
- Log monolithic indexing storage mode during startup.
- Keep lifecycle metrics/logs explicit so operators can see that indexing
  storage is active.
- SLI intent: monolithic in-memory indexing storage should initialize and stop
  cleanly with the process lifecycle.

## Risks
- Low risk because storage remains in-memory and debug-oriented.
- Main risk is accidental drift from `core.DatabasePlugin`/`core.CachePlugin`
  semantics; keep tests focused on those contracts.

## Rollback Plan
- Remove in-memory storage adapters and bootstrap helper.
- Revert monolithic indexer construction back to `nil` storage dependencies.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase111-monolithic-inmemory-indexing-storage.md`
- `go test ./pkg/adapters/indexing/... ./pkg/application/bootstrap/... ./pkg/services/indexing/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase111-monolithic-inmemory-indexing-storage.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the storage-activation step required before monolithic shared
  runtime can adopt the legacy-backed sink.

## Implementation Summary
- Added in-memory database and cache adapters for the current monolithic
  indexing contracts.
- Added bootstrap helper to initialize and start monolithic indexing storage.
- Wired monolithic chain indexers to those storage adapters instead of `nil`
  dependencies.
- Added adapter and bootstrap tests covering lifecycle and basic storage
  behavior.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase111-monolithic-inmemory-indexing-storage.md`
- `go test ./pkg/adapters/indexing/... ./pkg/application/bootstrap/... ./pkg/services/indexing/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
