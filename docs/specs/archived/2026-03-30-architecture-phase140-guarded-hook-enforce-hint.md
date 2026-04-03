# Phase 140 Guarded Hook Enforce Hint

## Title
Phase 140 - Add a service-level enforce hint above guarded cutover would-enforce summary

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
- `docs/specs/2026-03-30-architecture-phase139-guarded-hook-would-enforce-summary.md`

## Context
Phase 139 made future enforcement posture directly readable with a
`would-enforce` signal. Operators still need one even more compact hint that
can be scanned quickly during rollout review and service health inspection.

## Problem Statement
Without a service-level enforce hint, operators must still interpret
`would-allow`, `would-hold`, and `would-investigate` themselves instead of
receiving a concise operational recommendation.

## Scope
- Add an enforce hint derived from guarded cutover would-enforce summary.
- Normalize the hint into:
  - `safe-to-observe`
  - `hold-before-enforce`
  - `investigate-before-enforce`
- Expose it through:
  - rollout summary snapshot
  - readiness details
  - runtime metric code
  - structured startup/shutdown logs
  - console summary lines
- Keep runtime behavior non-blocking.

## Non-Goals
- No execution gating.
- No automatic rollout progression.
- No microservice integration.

## Selected Approach
- Add a thin interpretation layer above `would-enforce`.
- Prefer wording that is short, operationally clear, and suitable for dashboards
  and human review.

## Data / Contract Impact
- Readiness details expand with enforce hint state and reason.
- Runtime metrics expand with an enforce hint code gauge.
- Monolithic logs and console summary gain additive enforce hint lines.

## Observability
- Operators can see a single service-level recommendation for how to treat the
  current future-enforcement posture.

## Risks
- Low risk; additive observability layer only.

## Rollback Plan
- Remove the enforce hint classifier and its related readiness, metric, log,
  and summary outputs.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase140-guarded-hook-enforce-hint.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase140-guarded-hook-enforce-hint.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a compact service-level interpretation layer above
  `would-enforce`, preserving non-blocking rollout behavior.

## Implementation Notes
- Added a guarded cutover enforce hint derived from the `would-enforce`
  summary.
- Extended rollout summary, readiness details, runtime metrics, structured
  logs, and console summary with enforce hint state and reason.
- Kept the enforce hint advisory-only and non-blocking.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase140-guarded-hook-enforce-hint.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
