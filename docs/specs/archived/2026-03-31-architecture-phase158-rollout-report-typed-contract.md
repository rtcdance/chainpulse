# Phase 158 Rollout Report Typed Contract

## Title
Phase 158 - Convert `/health/rollout` to a typed rollout report contract

## Type
- feature
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
- `pkg/plugins/api/health_check_handler.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `cmd/monolithic/chainpulse/main.go`
- `pkg/plugins/api/health_check_handler_test.go`
- `pkg/plugins/api/gateway_runtime_integration_test.go`
- `docs/specs/2026-03-31-architecture-phase157-rollout-report-identity.md`

## Context
Phases 153-157 established a growing `/health/rollout` contract backed by
`map[string]interface{}` payload assembly. The report now has enough structure
that continuing to extend it as a loose map increases drift risk for future
monolith and microservice producers.

## Problem Statement
Without a typed contract, rollout report producers and consumers remain more
fragile than necessary, especially as the same report shape is expected to be
reused across deployment modes.

## Scope
- Introduce a typed rollout report details contract in the API layer.
- Change the rollout report provider to return the typed contract.
- Convert monolithic rollout report assembly to build the typed payload.
- Keep the `/health/rollout` JSON shape stable.

## Non-Goals
- No rollout decision logic changes.
- No route changes.
- No health/readiness semantic changes.

## Selected Approach
- Add typed report structs to the health API package.
- Replace the rollout report provider signature from a map-based payload to a
  typed payload.
- Preserve existing field names and nested JSON structure.

## Data / Contract Impact
- Internal provider contract becomes typed.
- External `/health/rollout` JSON contract remains backward-compatible.

## Observability
- Reduces payload drift risk and makes future rollout report producers easier
  to align and test.

## Risks
- Moderate-low: any remaining map-based assumptions in tests or providers need
  to be updated to the typed shape.

## Rollback Plan
- Revert the provider signature and monolithic report assembly back to
  `map[string]interface{}` while preserving current report fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase158-rollout-report-typed-contract.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase158-rollout-report-typed-contract.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first real contract-hardening step for `/health/rollout`
  after stabilizing the report metadata envelope.

## Implementation Notes
- Added typed rollout report structs in the API layer.
- Updated monolithic rollout report assembly to build the typed payload.
- Updated rollout report provider wiring and tests to use the typed contract.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase158-rollout-report-typed-contract.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
