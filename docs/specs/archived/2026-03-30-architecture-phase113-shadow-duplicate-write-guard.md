# Phase 113 Shadow Duplicate Write Guard

## Title
Phase 113 - Add duplicate write guard between shadow runtime sink and legacy indexing path

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
- `pkg/services/indexing/shadow_write_tracker.go`
- `pkg/services/indexing/legacy_runtime_sink.go`
- `pkg/services/indexing/chain_indexer.go`
- `pkg/services/indexing/chain_indexer_test.go`
- `docs/specs/2026-03-30-architecture-phase112-monolithic-runtime-legacy-sink-wiring.md`

## Context
Phase 112 switched monolithic shared runtime wiring to the legacy-backed sink,
so shared runtime shadow batches and legacy indexing now persist the same event
objects through the same in-memory storage semantics.

## Problem Statement
Without a duplicate-write guard, the current shadow arrangement can write the
same event twice: once through shared runtime sink and once again through the
legacy chain indexer path.

## Scope
- Add a process-local shadow write tracker for event pointers persisted by the
  shared runtime sink.
- Mark events after successful shared runtime sink persistence.
- Consume the tracker in `DefaultChainIndexer` to skip duplicate legacy writes
  for the same in-flight event object.
- Add focused tests for guarded and non-guarded paths.

## Non-Goals
- No change to external storage contracts.
- No ownership handoff from legacy path to shared runtime.
- No replay, reorg, or microservice changes in this phase.

## Selected Approach
- Use an internal in-process tracker keyed by `*core.BlockchainEvent` identity.
- Keep the guard narrow to current shadow-mode duplication only.
- Consume marks once to avoid tracker growth across batches.

## Data / Contract Impact
- No external API or persistence contract changes.
- Internal indexing behavior now suppresses duplicate legacy writes after a
  successful shadow runtime persistence of the same event object.

## Observability
- Duplicate suppression remains silent for now because it is an expected shadow
- mode behavior correction rather than an operator-facing incident.
- Existing error counters/logs remain unchanged for failure paths.

## Risks
- Low risk because the guard is process-local and only affects the same event
  object flowing through shadow runtime and legacy path in one batch.

## Rollback Plan
- Remove tracker and revert legacy chain indexer to unconditional writes.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase113-shadow-duplicate-write-guard.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase113-shadow-duplicate-write-guard.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the smallest guardrail needed after switching runtime sink to the
  legacy-backed adapter in monolithic shadow mode.

## Implementation Summary
- Added a process-local shadow write tracker keyed by `*core.BlockchainEvent`.
- Marked events after successful legacy-backed shared runtime sink persistence.
- Consumed tracker state in `DefaultChainIndexer` to skip duplicate legacy
  writes for the same in-flight event object.
- Added a focused regression test that verifies duplicate legacy writes are
  suppressed while indexer counters still advance.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase113-shadow-duplicate-write-guard.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
