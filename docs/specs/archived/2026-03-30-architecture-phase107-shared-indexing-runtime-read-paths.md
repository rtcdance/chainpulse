# Phase 107 Shared Indexing Runtime Read Paths

## Title
Phase 107 - Add checkpoint and replay read helpers to shared indexing runtime

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
- `docs/specs/2026-03-30-architecture-phase106-shared-indexing-runtime-process-batch.md`

## Context
Phase 106 added the first write-path orchestration for the shared indexing
runtime, but there is still no shared read path for checkpoint loading or replay
batch loading.

## Problem Statement
Without explicit read helpers, later monolith and microservice recovery wiring
would still need to reach directly into checkpoint and replay adapters.

## Scope
- Add `LoadCheckpoint` helper to `SharedRuntime`.
- Add `LoadReplayBatch` helper to `SharedRuntime`.
- Add tests for checkpoint loading, replay loading, and lifecycle guardrails.

## Non-Goals
- No replay execution loop in this phase.
- No runtime entrypoint wiring in this phase.
- No mutation of existing write-path behavior.

## Selected Approach
- Keep read helpers narrow and explicit.
- Require runtime initialization before checkpoint/replay reads.
- Allow replay source to be optional and fail explicitly when absent.

## Data / Contract Impact
- Internal shared runtime contract expands with read helpers.
- No external deployment or API contract changes.

## Risks
- Minimal; later phases may widen replay semantics.

## Rollback Plan
- Remove read helpers and keep checkpoint/replay access behind adapters only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase107-shared-indexing-runtime-read-paths.md`
- `go test ./pkg/application/indexing/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase107-shared-indexing-runtime-read-paths.md`
- `go test ./pkg/application/indexing/...`

## Review Notes
- Approved as the next additive step before monolith parity wiring.

## Implementation Summary
- Added shared checkpoint and replay read helpers to `SharedRuntime`.
- Added tests for success and lifecycle/availability error paths.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase107-shared-indexing-runtime-read-paths.md`
- `go test ./pkg/application/indexing/...`
