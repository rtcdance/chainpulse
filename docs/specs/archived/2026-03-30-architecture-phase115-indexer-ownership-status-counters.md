# Phase 115 Indexer Ownership Status Counters

## Title
Phase 115 - Add shadow-owned and legacy-owned event counters to chain indexer status

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
- `pkg/services/indexing/chain_indexer.go`
- `pkg/services/indexing/chain_indexer_test.go`
- `docs/specs/2026-03-30-architecture-phase114-shadow-write-ownership-metrics.md`

## Context
Phase 114 added ownership metrics for shared-runtime-owned writes, but the
chain indexer status surface still reports only aggregate totals.

## Problem Statement
Without ownership-aware status counters, local debugging and status-based checks
cannot distinguish events written through shared runtime from events still owned
by the legacy write path.

## Scope
- Add `shadowOwnedEvents` and `legacyOwnedEvents` counters to
  `DefaultChainIndexer`.
- Increment the appropriate counter on shadow-owned suppression and on normal
  legacy writes.
- Expose both counters through `GetStatus()`.
- Add focused tests for ownership-aware status reporting.

## Non-Goals
- No change to write ownership rules.
- No new external APIs.
- No microservice or replay changes.

## Selected Approach
- Keep ownership accounting local to `DefaultChainIndexer`.
- Continue using `total_events_indexed` as the aggregate top-line counter.
- Add ownership-specific counters as subordinate status fields.

## Data / Contract Impact
- `GetStatus()` expands with:
  - `shadow_owned_events`
  - `legacy_owned_events`
- No persistence contract changes.

## Observability
- Status output now complements the Phase 114 metrics.
- Operators and developers can compare:
  - aggregate indexed events
  - shared-runtime-owned events
  - legacy-owned events

## Risks
- Low risk; additive accounting only.

## Rollback Plan
- Remove the two counters and their status fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase115-indexer-ownership-status-counters.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase115-indexer-ownership-status-counters.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the next local-debug/status visibility slice after adding shadow
  ownership metrics.

## Implementation Summary
- Added `shadowOwnedEvents` and `legacyOwnedEvents` counters to
  `DefaultChainIndexer`.
- Exposed both counters through `GetStatus()`.
- Updated tests so status now verifies ownership split for both legacy-owned and
  shadow-owned paths.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase115-indexer-ownership-status-counters.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
