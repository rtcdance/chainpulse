# Phase 137 Guarded Cutover Hook Dry-Run

## Title
Phase 137 - Add a no-op guarded cutover hook signal for monolithic rollout control

## Type
- architecture
- indexing
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
2026-03-30

## Related Modules
- `cmd/monolithic/chainpulse/ownership_rollout_control.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase136-ownership-rollout-control-helper.md`

## Context
The rollout control plane already exposes dry-run cutover, candidate, manual
approval, handoff, and checklist signals. The next step is to let one signal
actually consume that upstream graph, while still avoiding any execution-path
change.

## Problem Statement
Without a guarded hook signal, we still do not have a single summary decision
that answers: "if we attached a guarded cutover consumer here, would it hold,
allow, or require investigation?"

## Scope
- Add a no-op guarded cutover hook classifier derived from existing rollout
  signals.
- Expose the hook through:
  - rollout summary snapshot
  - readiness details
  - runtime metric code
  - structured startup/shutdown logs
  - console summary lines
- Keep runtime behavior unchanged.

## Non-Goals
- No cutover enforcement.
- No mutation of indexing execution paths.
- No microservice integration.

## Selected Approach
- Classify a guarded hook action into:
  - `noop-allow`
  - `noop-hold`
  - `noop-investigate`
- Base the decision on the existing dry-run cutover action, cutover candidate,
  and approval checklist posture.
- Treat the hook as advisory-only output for now.

## Data / Contract Impact
- Readiness details expand with guarded hook action and reason.
- Runtime metrics expand with a guarded hook code gauge.
- Monolithic logs and console summary gain additive guarded hook lines.

## Observability
- Operators can now see the first consumer-style decision that would be used by
  a future guarded cutover controller, without changing service behavior.

## Risks
- Low risk; additive decision surface only.

## Rollback Plan
- Remove the guarded hook classifier and its related readiness, metric, log,
  and summary outputs.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase137-guarded-cutover-hook-dry-run.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase137-guarded-cutover-hook-dry-run.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first no-op consumer of approval and cutover summary signals,
  preserving current execution behavior while preparing for future gating.

## Implementation Notes
- Added a guarded cutover hook classifier that derives a no-op consumer action
  from the existing dry-run cutover, candidate, and approval checklist signals.
- Extended the ownership rollout summary snapshot, readiness details, metrics,
  structured logs, and console summary with guarded hook action and reason.
- Kept execution behavior unchanged; the hook is advisory-only and does not
  mutate indexing flow.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase137-guarded-cutover-hook-dry-run.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
