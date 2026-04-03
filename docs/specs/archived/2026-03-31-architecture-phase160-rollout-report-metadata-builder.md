# Phase 160 Rollout Report Metadata Builder

## Title
Phase 160 - Extract rollout report metadata builder for shared report identity

## Type
- refactor
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
- `pkg/plugins/api/rollout_report_contract.go`
- `pkg/plugins/api/rollout_report_contract_test.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `docs/specs/2026-03-31-architecture-phase159-rollout-report-contract-file.md`

## Context
Phase 159 moved rollout report contract types into a dedicated file. The
monolithic rollout summary still hand-populates the metadata identity envelope
inline, which makes future reuse across deployment modes more repetitive than
necessary.

## Problem Statement
If rollout report metadata continues to be assembled inline per producer,
identity fields such as `report_id`, `schema_family`, and deployment metadata
remain more likely to drift across monolith and future microservice producers.

## Scope
- Add a shared rollout report metadata builder in the API contract layer.
- Update the monolithic rollout report builder to consume the shared metadata
  builder.
- Keep the external `/health/rollout` JSON contract unchanged.

## Non-Goals
- No rollout decision changes.
- No health or readiness semantic changes.
- No schema field additions or removals.

## Selected Approach
- Introduce a `RolloutReportMetadata` envelope and a constructor that produces a
  typed `RolloutReportDetails` pre-populated with stable report identity.
- Let monolithic report assembly fill only the rollout-specific body fields.

## Data / Contract Impact
- No external JSON changes.
- Internal producer contract becomes easier to reuse across deployment modes.

## Observability
- Reduces contract drift risk for future rollout report producers.

## Risks
- Low: primarily structural refactoring risk, mitigated by compile/test gates.

## Rollback Plan
- Revert to inline metadata assembly in monolithic rollout summary while keeping
  the typed contract itself.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase160-rollout-report-metadata-builder.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase160-rollout-report-metadata-builder.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk contract-hardening step that prepares rollout report
  identity for reuse across future deployment modes.

## Implementation Notes
- Added `RolloutReportMetadata` and `NewRolloutReportDetailsFromMetadata(...)`.
- Added contract-level tests for metadata construction.
- Updated monolithic rollout report assembly to consume the shared metadata
  builder.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase160-rollout-report-metadata-builder.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
