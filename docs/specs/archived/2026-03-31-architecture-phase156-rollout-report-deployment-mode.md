# Phase 156 Rollout Report Deployment Mode Metadata

## Title
Phase 156 - Add report mode and deployment mode metadata to `/health/rollout`

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
- `docs/specs/2026-03-31-architecture-phase155-rollout-report-scope-source.md`

## Context
Phase 155 added explicit report scope and source metadata to `GET /health/rollout`.
The payload still does not explicitly describe whether the report comes from a
runtime-oriented surface or which deployment mode produced it.

## Problem Statement
As the same rollout report schema expands toward microservice deployments,
consumers need a stable way to distinguish runtime-vs-other report modes and
monolithic-vs-microservice deployment context without inferring it from service
names alone.

## Scope
- Extend `/health/rollout` metadata with:
  - `report_mode`
  - `deployment_mode`
- Use stable monolithic runtime values.

## Non-Goals
- No rollout decision changes.
- No route changes.
- No handler semantic changes.

## Selected Approach
- Extend the existing rollout metadata envelope with:
  - `report_mode=runtime`
  - `deployment_mode=monolithic`
- Preserve the rest of the rollout report payload unchanged.

## Data / Contract Impact
- Extends `GET /health/rollout` details with two metadata keys.
- Existing payload structure remains intact.

## Observability
- Improves downstream classification for dashboards, automation, and future
  monolith/microservice parity checks.

## Risks
- Low: consumers that assume a smaller metadata set need to tolerate additional
  keys.

## Rollback Plan
- Remove `report_mode` and `deployment_mode` from the rollout report metadata
  while preserving the rest of the report.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase156-rollout-report-deployment-mode.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase156-rollout-report-deployment-mode.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk metadata extension to prepare `/health/rollout` for
  shared report contracts across deployment modes.

## Implementation Notes
- Added `report_mode` and `deployment_mode` to rollout report details.
- Extended health handler and gateway rollout tests to assert the new fields.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase156-rollout-report-deployment-mode.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
