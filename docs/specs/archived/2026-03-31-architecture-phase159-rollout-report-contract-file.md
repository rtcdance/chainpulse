# Phase 159 Rollout Report Contract File Extraction

## Title
Phase 159 - Extract rollout report contract types into a dedicated file

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
- `pkg/plugins/api/health_check_handler.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `docs/specs/2026-03-31-architecture-phase158-rollout-report-typed-contract.md`

## Context
Phase 158 introduced a typed rollout report contract, but the types still live
inside `health_check_handler.go`. That keeps the contract artificially coupled
to one specific handler implementation.

## Problem Statement
If rollout report types remain embedded in the handler file, future monolithic
and microservice producers will have to depend on handler-local definitions
instead of a clean shared contract location.

## Scope
- Move rollout report types into a dedicated contract file in `pkg/plugins/api`.
- Keep handler behavior, payload shape, and producer logic unchanged.

## Non-Goals
- No rollout logic changes.
- No payload schema changes.
- No route or readiness changes.

## Selected Approach
- Create a dedicated `rollout_report_contract.go` file for rollout report
  request/response structs.
- Remove duplicated contract definitions from the handler file.

## Data / Contract Impact
- No external JSON changes.
- Internal contract ownership moves to a dedicated file.

## Observability
- Improves maintainability of the rollout report surface and makes future
  producer reuse more direct.

## Risks
- Low: mainly import/file-organization risk, mitigated by compile/test gates.

## Rollback Plan
- Move the contract definitions back into `health_check_handler.go` without
  changing the external `/health/rollout` contract.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase159-rollout-report-contract-file.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase159-rollout-report-contract-file.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk structural cleanup that separates report contract
  ownership from handler implementation details.

## Implementation Notes
- Added `pkg/plugins/api/rollout_report_contract.go`.
- Removed rollout report type definitions from `health_check_handler.go`.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase159-rollout-report-contract-file.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
