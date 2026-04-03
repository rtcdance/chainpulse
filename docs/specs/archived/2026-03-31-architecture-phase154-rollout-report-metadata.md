# Phase 154 Rollout Report Metadata

## Title
Phase 154 - Add stable metadata envelope to `/health/rollout`

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
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `pkg/plugins/api/health_check_handler_test.go`
- `pkg/plugins/api/gateway_runtime_integration_test.go`
- `docs/specs/2026-03-31-architecture-phase153-ownership-rollout-health-report-surface.md`

## Context
Phase 153 introduced `GET /health/rollout` as a dedicated ownership rollout
report surface. The payload currently contains rollout content but no explicit
metadata envelope for schema versioning or producer identity.

## Problem Statement
Without a stable metadata layer, downstream tooling has no clear way to detect
report schema version, producing service, or report generation time as the
surface evolves.

## Scope
- Add stable metadata fields to the rollout report payload:
  - `report_version`
  - `service`
  - `generated_at`
- Keep the existing rollout report content intact.

## Non-Goals
- No rollout logic changes.
- No readiness or health semantics changes.
- No route changes.

## Selected Approach
- Extend the existing rollout report details map with a small metadata envelope.
- Use a stable version token (`v1`) and current monolithic service identity.

## Data / Contract Impact
- Extends `GET /health/rollout` details with metadata fields.
- Existing detail keys remain present.

## Observability
- Improves report consumers' ability to reason about report freshness and
  schema compatibility.

## Risks
- Low: downstream tests or ad-hoc parsing that assumed only business fields may
  need to tolerate metadata additions.

## Rollback Plan
- Remove the metadata fields from the rollout report payload while keeping the
  endpoint itself.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase154-rollout-report-metadata.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase154-rollout-report-metadata.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a lightweight metadata hardening step for the new rollout report
  surface.

## Implementation Notes
- Added `report_version`, `service`, and `generated_at` to rollout report
  details.
- Extended health handler and gateway runtime integration tests to assert the
  metadata envelope.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase154-rollout-report-metadata.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
