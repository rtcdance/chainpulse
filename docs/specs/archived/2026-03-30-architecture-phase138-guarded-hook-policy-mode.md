# Phase 138 Guarded Hook Policy Mode

## Title
Phase 138 - Add guarded cutover hook policy mode for monolithic rollout control

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
- `docs/specs/2026-03-30-architecture-phase137-guarded-cutover-hook-dry-run.md`

## Context
Phase 137 introduced a no-op guarded cutover hook signal, but the system still
does not distinguish between the current safe default posture and a future
policy posture that could eventually become enforcement-aware.

## Problem Statement
Without a guarded hook policy mode, operators cannot tell whether the hook is
running in pure no-op observation mode or in a future-facing mode that is being
prepared for stronger control semantics.

## Scope
- Add a guarded hook policy mode with explicit normalization and advisory-only
  action mapping.
- Support:
  - `noop-only`
  - `enforce-ready`
- Keep all modes non-blocking for now.
- Expose the new policy layer through:
  - rollout summary snapshot
  - readiness details
  - runtime metric code
  - structured startup/shutdown logs
  - console summary lines

## Non-Goals
- No cutover enforcement.
- No execution-path mutation.
- No microservice integration.

## Selected Approach
- Default to `noop-only`.
- Add an env-driven policy resolver for guarded hook mode.
- Let `enforce-ready` produce a future-facing advisory action while explicitly
  remaining non-blocking.

## Data / Contract Impact
- Readiness details expand with guarded hook policy mode/action/reason.
- Runtime metrics expand with guarded hook policy mode code.
- Monolithic logs and console summary gain additive guarded hook policy fields.

## Observability
- Operators can distinguish:
  - the hook outcome itself
  - the policy mode interpreting that outcome

## Risks
- Low risk; additive observability and policy semantics only.

## Rollback Plan
- Remove guarded hook policy mode resolution and revert to the direct guarded
  hook signal only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase138-guarded-hook-policy-mode.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase138-guarded-hook-policy-mode.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a non-blocking policy layer above the guarded cutover hook,
  preserving runtime behavior while making future control intent explicit.

## Implementation Notes
- Added a guarded cutover hook policy layer with env normalization for
  `noop-only` and `enforce-ready`.
- Extended rollout summary, readiness details, runtime metrics, structured
  logs, and console summary with guarded hook policy mode/action/reason.
- Kept `enforce-ready` advisory-only; it expresses future control intent
  without changing current execution behavior.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase138-guarded-hook-policy-mode.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
