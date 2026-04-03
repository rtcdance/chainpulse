# Phase 157 Rollout Report Identity Metadata

## Title
Phase 157 - Add report identity metadata to `/health/rollout`

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
- `docs/specs/2026-03-31-architecture-phase156-rollout-report-deployment-mode.md`

## Context
Phase 156 extended `GET /health/rollout` with scope, source, report mode, and
deployment mode metadata. The payload still lacks a fixed identity token that
consumers can use as a stable schema anchor across deployment variants.

## Problem Statement
As rollout reporting grows beyond the monolith, consumers need a stable report
identity that can be matched across producers without reconstructing identity
from several metadata keys.

## Scope
- Extend `/health/rollout` metadata with:
  - `report_id`
  - `schema_family`
- Keep existing rollout report content and metadata intact.

## Non-Goals
- No rollout decision changes.
- No route or handler behavior changes.
- No readiness semantic changes.

## Selected Approach
- Add a stable report identity envelope:
  - `report_id=monolithic-ownership-rollout-runtime`
  - `schema_family=ownership-rollout-report`
- Preserve current metadata and rollout payload shape.

## Data / Contract Impact
- Extends `GET /health/rollout` details with two identity metadata keys.
- Existing metadata and business fields remain present.

## Observability
- Makes rollout report consumers easier to version, classify, and align across
  monolithic and future microservice producers.

## Risks
- Low: downstream consumers need to tolerate extra metadata keys.

## Rollback Plan
- Remove `report_id` and `schema_family` from the rollout report metadata while
  keeping the rest of the report unchanged.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase157-rollout-report-identity.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase157-rollout-report-identity.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk contract-hardening step for future rollout report
  consumers and deployment-parity tooling.

## Implementation Notes
- Added `report_id` and `schema_family` to rollout report details.
- Extended health handler and gateway rollout tests to assert the new identity
  metadata keys.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase157-rollout-report-identity.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
