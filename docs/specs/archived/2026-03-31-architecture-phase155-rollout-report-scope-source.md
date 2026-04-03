# Phase 155 Rollout Report Scope And Source

## Title
Phase 155 - Add report scope and source metadata to `/health/rollout`

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
- `docs/specs/2026-03-31-architecture-phase154-rollout-report-metadata.md`

## Context
Phase 154 added a stable metadata envelope to `GET /health/rollout` with
version, service identity, and generation time. The report still does not
explicitly state which rollout domain it describes or which producer family the
payload belongs to.

## Problem Statement
As rollout reporting expands to microservice surfaces, report consumers need a
stable way to distinguish report domain and producer source without inferring
them from URL shape or surrounding deployment context.

## Scope
- Extend `/health/rollout` report metadata with:
  - `report_scope`
  - `report_source`
- Use stable monolithic ownership-rollout values.

## Non-Goals
- No rollout decision logic changes.
- No handler routing changes.
- No readiness or health semantic changes.

## Selected Approach
- Keep the existing metadata envelope and add two explicit identity fields:
  - `report_scope=ownership-rollout`
  - `report_source=monolithic`
- Preserve the rest of the rollout payload unchanged.

## Data / Contract Impact
- Extends the `GET /health/rollout` details payload with two new metadata keys.
- Existing detail keys remain intact.

## Observability
- Makes rollout reports easier to classify in dashboards, scripts, and future
  monolith/microservice parity tooling.

## Risks
- Low: consumers that hard-coded a smaller metadata set may need to tolerate
  the extra fields.

## Rollback Plan
- Remove `report_scope` and `report_source` from the rollout report while
  keeping the rest of the metadata envelope.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase155-rollout-report-scope-source.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase155-rollout-report-scope-source.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk metadata-hardening step that prepares `/health/rollout`
  for future multi-surface rollout reporting.

## Implementation Notes
- Added `report_scope` and `report_source` to the rollout report details.
- Extended health handler and gateway rollout report tests to assert the new
  metadata keys.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase155-rollout-report-scope-source.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
