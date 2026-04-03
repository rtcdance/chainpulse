# Phase 106 Shared Indexing Runtime Process Batch

## Title
Phase 106 - Add first shared indexing runtime orchestration path

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
- `pkg/application/indexing/runtime.go`
- `pkg/application/indexing/runtime_test.go`
- `docs/specs/2026-03-30-architecture-phase104-indexer-runtime-closure-slice.md`
- `docs/specs/2026-03-30-architecture-phase105-shared-indexing-runtime-skeleton.md`

## Context
Phase 105 added the shared indexing runtime skeleton and ports, but it still did
not orchestrate any actual indexing work.

## Problem Statement
Without a first shared processing path, later monolith/microservice wiring would
still have no concrete runtime behavior to reuse.

## Scope
- Add a first `ProcessBatch` orchestration path to `SharedRuntime`.
- Handle:
  - duplicate filtering via idempotency store
  - batch persistence
  - processed marking
  - checkpoint advancement
  - failure routing on persist/mark failures
- Add focused unit tests for success, duplicate skip, and failure routing.

## Non-Goals
- No service entrypoint wiring in this phase.
- No replay execution flow in this phase.
- No MQ integration in this phase.

## Selected Approach
- Keep orchestration minimal and explicit.
- Persist accepted envelopes in one batch.
- Advance checkpoint from the last accepted envelope.
- Route failures through `FailureRouter` when persistence or processed marking
  fails.

## Data / Contract Impact
- Internal application-layer runtime contract expands with batch processing
  capability.
- No external deployment or API contract changes.

## Risks
- The first orchestration path may encode assumptions that later need widening.
- Mitigation: keep behavior narrow, additive, and covered by tests.

## Rollback Plan
- Remove `ProcessBatch` and revert to lifecycle-only runtime skeleton.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase106-shared-indexing-runtime-process-batch.md`
- `go test ./pkg/application/indexing/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase106-shared-indexing-runtime-process-batch.md`
- `go test ./pkg/application/indexing/...`

## Review Notes
- Approved as the first behavior-bearing step of the shared indexing runtime.

## Implementation Summary
- Added `ProcessBatch` to shared indexing runtime.
- Added tests for successful processing, duplicate skipping, and routed failure handling.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase106-shared-indexing-runtime-process-batch.md`
- `go test ./pkg/application/indexing/...`
