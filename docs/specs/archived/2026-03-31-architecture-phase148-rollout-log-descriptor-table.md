# Phase 148 Rollout Log Descriptor Table

## Title
Phase 148 - Convert rollout lifecycle logs into a descriptor table

## Type
- refactor
- architecture
- observability

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
2026-03-31

## Related Modules
- `cmd/monolithic/chainpulse/ownership_rollout_presenter.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-31-architecture-phase147-rollout-presenter-descriptor-table.md`

## Context
Phase 147 converted rollout console presentation into a descriptor table. The
structured lifecycle logs still rely on a long sequence of manual `logger.Info`
calls.

## Problem Statement
Keeping rollout logs hand-written while console output is descriptor-driven
creates asymmetry in the presenter layer and makes future rollout signal
changes easier to miss in one surface.

## Scope
- Convert rollout lifecycle logs into a descriptor table.
- Preserve existing log event names and field selection.
- Add focused tests for log descriptor boundaries.

## Non-Goals
- No rollout behavior changes.
- No readiness or metric contract changes.
- No log schema expansion.

## Selected Approach
- Add a `ownershipRolloutLogDescriptor` table containing the log message name
  and a field renderer.
- Keep `logOwnershipRolloutSummary(...)` as the single presenter entry and make
  it iterate descriptors.

## Data / Contract Impact
- No external contract change intended.
- Existing log messages remain stable.

## Observability
- Refactor only; structured rollout logging remains semantically unchanged.

## Risks
- Medium-low: a descriptor could omit a field or map the wrong rollout value.

## Rollback Plan
- Inline the descriptor table back into `logOwnershipRolloutSummary(...)` and
  remove the helper type.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase148-rollout-log-descriptor-table.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase148-rollout-log-descriptor-table.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a presenter symmetry cleanup that keeps rollout logs and console
  rendering on parallel descriptor-driven structures.

## Implementation Notes
- Added a lifecycle log descriptor table for rollout messages and field
  rendering.
- Added focused tests for descriptor count and first/last log message
  stability.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase148-rollout-log-descriptor-table.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
