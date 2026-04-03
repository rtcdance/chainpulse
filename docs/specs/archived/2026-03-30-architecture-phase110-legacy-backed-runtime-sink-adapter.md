# Phase 110 Legacy-Backed Runtime Sink Adapter

## Title
Phase 110 - Add legacy-backed shared runtime sink adapter for indexing storage semantics

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
- `pkg/services/indexing/legacy_runtime_sink.go`
- `pkg/services/indexing/legacy_runtime_sink_test.go`
- `pkg/services/indexing/chain_indexer.go`
- `docs/specs/2026-03-30-architecture-phase109-chain-indexer-shared-runtime-shadow-batch.md`

## Context
Phase 109 sends real monolithic event batches into the shared runtime in shadow
mode, but the shared runtime still uses a no-op sink in the monolithic bootstrap.

## Problem Statement
Without a sink adapter that matches existing indexing storage semantics, the
shared runtime can observe real batches but still cannot exercise realistic
persistence behavior.

## Scope
- Add a legacy-backed `appindexing.EventSink` adapter in the indexing layer.
- Persist payload-backed `core.BlockchainEvent` instances through the existing
  `core.DatabasePlugin`.
- Mirror existing event cache key behavior through the existing
  `core.CachePlugin`.
- Add focused unit tests for persistence, caching, and invalid payload guardrails.

## Non-Goals
- No monolithic entrypoint switch to this sink yet.
- No rewrite of `DefaultChainIndexer.indexEvent`.
- No replay, reorg, or DLQ behavior changes in this phase.

## Selected Approach
- Keep the adapter in `pkg/services/indexing` because it bridges shared runtime
  envelopes to legacy indexing storage semantics.
- Require database plugin; allow cache plugin to be optional.
- Reuse the same cache key shape already used by legacy indexing writes.

## Data / Contract Impact
- Internal shared runtime sink catalog expands with a legacy-backed adapter.
- No external API or deployment contract changes in this phase.

## Observability
- Preserve warning-only cache failure behavior.
- Keep future switch-over simple by matching current storage side effects.
- SLI intent: sink adapter persistence should remain behaviorally equivalent to
  legacy `indexEvent` writes for accepted events.

## Risks
- Low risk because adapter is introduced behind tests only in this phase.
- Main risk is storage-semantic drift between sink adapter and legacy path.

## Rollback Plan
- Remove legacy sink adapter and its tests.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase110-legacy-backed-runtime-sink-adapter.md`
- `go test ./pkg/services/indexing/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase110-legacy-backed-runtime-sink-adapter.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the storage-semantics preparation step before switching shared
  runtime away from no-op sink wiring.

## Implementation Summary
- Added `LegacyRuntimeSink` as an `appindexing.EventSink` backed by the current
  indexing `DatabasePlugin` and optional `CachePlugin`.
- Reused legacy cache key semantics from the existing chain indexer path.
- Added tests for persistence, cache mirroring, optional cache behavior, and
  invalid payload rejection.
- Kept monolithic entrypoint wiring unchanged because real db/cache plugins are
  not yet connected there.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase110-legacy-backed-runtime-sink-adapter.md`
- `go test ./pkg/services/indexing/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
