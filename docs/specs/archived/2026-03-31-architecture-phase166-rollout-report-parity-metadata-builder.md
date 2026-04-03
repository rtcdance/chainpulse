# Phase 166 Rollout Report Parity Metadata Builder

## Title
Phase 166 - Share rollout report parity metadata builder across producers

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
- `cmd/microservices/api-service/rollout_report_producer.go`
- `docs/specs/2026-03-31-architecture-phase165-api-service-rollout-producer-skeleton.md`

## Context
Phase 165 introduced a second rollout report producer in `api-service`, but
monolith and microservice producers still repeated the shared rollout identity
metadata inline.

## Problem Statement
Without a shared ownership-rollout metadata builder, parity between monolith and
microservice report identity relies on duplicated literals rather than code-level
reuse.

## Scope
- Add shared constants for ownership-rollout report identity metadata.
- Add a shared `NewOwnershipRolloutReportMetadata(...)` builder.
- Update monolith and api-service producers to consume the shared builder.

## Non-Goals
- No rollout logic changes.
- No payload schema changes.
- No producer behavior changes beyond metadata reuse.

## Selected Approach
- Define shared identity constants in the rollout report contract layer.
- Provide a dedicated builder for ownership-rollout report metadata so both
  monolith and microservice producers share the same identity defaults.

## Data / Contract Impact
- No external JSON changes.
- Internal producer parity now uses shared metadata construction.

## Observability
- Reduces rollout report identity drift risk across deployment modes.

## Risks
- Low: refactor only, mitigated by contract-level tests and existing producer
  tests.

## Rollback Plan
- Revert producers to inline metadata construction while preserving the producer
  interface and typed report contract.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase166-rollout-report-parity-metadata-builder.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase166-rollout-report-parity-metadata-builder.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first real parity hardening step between monolith and
  microservice rollout report producers.

## Implementation Notes
- Added shared ownership-rollout metadata constants and builder.
- Updated monolith and api-service producers to consume the shared builder.
- Added contract-level tests for the shared metadata builder.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase166-rollout-report-parity-metadata-builder.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
