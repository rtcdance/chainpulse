# Phase 131 Manual Approval Checkpoint Signal

## Title
Phase 131 - Emit manual approval checkpoint signal for monolithic cutover candidates

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
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase130-cutover-candidate-signal.md`

## Context
Phase 130 introduced an explicit cutover candidate signal, but operators still
need one more stable surface that says whether the instance is now waiting for
manual approval rather than merely being candidate-eligible.

## Problem Statement
Without a dedicated manual approval checkpoint signal, operators and future
automation must infer "candidate but not yet cut over" from multiple rollout
fields and logs.

## Scope
- Add a non-blocking manual approval checkpoint classifier.
- Treat an eligible cutover candidate as `awaiting-approval`.
- Treat non-candidate states as `inactive`.
- Treat unknown rollout progression as `unknown`.
- Emit the signal through:
  - readiness details
  - runtime metric gauge
  - structured startup/shutdown logs
  - console summary lines
- Add focused tests for classification and metric/readiness exposure.

## Non-Goals
- No cutover enforcement.
- No microservice integration.
- No new rollout flags or policy modes.

## Selected Approach
- Reuse the existing cutover candidate and progression signals.
- Keep checkpoint classification local to monolithic rollout aggregation.
- Export the checkpoint as an additive audit-style state for future approval
  workflows.

## Data / Contract Impact
- Readiness details expand with checkpoint state and reason.
- Runtime metrics expand with a checkpoint code gauge.
- Monolithic logs and console summary gain additive checkpoint fields.

## Observability
- Operators can distinguish:
  - not yet a candidate
  - candidate awaiting manual approval
  - unknown rollout posture
- This becomes the audit-oriented bridge between dry-run recommendation and
  future operator-approved cutover.

## Risks
- Low risk; additive signaling only.

## Rollback Plan
- Remove the checkpoint classifier and its associated metric/log/summary
  outputs.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase131-manual-approval-checkpoint-signal.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase131-manual-approval-checkpoint-signal.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as an audit-only checkpoint layer above cutover candidate detection,
  while preserving non-blocking rollout behavior.
- Implemented with readiness, metrics, structured logs, console summary, and
  focused monolithic tests.
