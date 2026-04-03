# Phase 130 Cutover Candidate Signal

## Title
Phase 130 - Emit cutover candidate signal from monolithic dry-run rollout state

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
- `docs/specs/2026-03-30-architecture-phase129-cutover-dry-run-observability-alignment.md`

## Context
Phase 129 aligned dry-run cutover recommendation across readiness, metrics, and
console output, but the service still does not emit a single structured signal
that clearly marks "this instance is now a cutover candidate."

## Problem Statement
Without a structured cutover candidate signal, downstream operators and future
automation must infer readiness for cutover by combining several rollout fields.

## Scope
- Add a non-blocking cutover candidate classifier.
- Treat `manual-gate + acknowledged + ready-for-cutover + would-allow` as a
  cutover candidate.
- Emit the signal through:
  - structured startup/shutdown logs
  - runtime metric gauge
- Add focused tests for candidate classification and metric emission.

## Non-Goals
- No enforced cutover.
- No runtime behavior change.
- No microservice integration.

## Selected Approach
- Keep classification local to monolithic rollout aggregation.
- Reuse existing policy, effective progression, and dry-run cutover states.
- Export a boolean-like gauge and structured log fields for future consumers.

## Data / Contract Impact
- Runtime metrics expand with a cutover candidate gauge.
- Monolithic logs gain additive cutover candidate fields.

## Observability
- Operators can now identify cutover-ready instances through one explicit
  signal, instead of inferring from multiple rollout fields.

## Risks
- Low risk; additive signaling only.

## Rollback Plan
- Remove cutover candidate classifier, metric, and structured logs.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase130-cutover-candidate-signal.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase130-cutover-candidate-signal.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first structured consumer of dry-run cutover state while
  preserving safe, non-blocking behavior.
