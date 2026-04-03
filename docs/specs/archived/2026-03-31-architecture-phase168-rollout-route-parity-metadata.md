# Phase 168 Rollout Route Parity Metadata

## Title
Phase 168 - Add monolith and api-service rollout route parity metadata coverage

## Type
- test
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
- `cmd/monolithic/chainpulse/rollout_report_integration_test.go`
- `cmd/microservices/api-service/rollout_report_integration_test.go`
- `pkg/plugins/api/rollout_report_contract.go`
- `docs/specs/2026-03-31-architecture-phase167-api-service-rollout-route-integration.md`

## Context
Phase 167 proved that the api-service rollout producer skeleton is visible over
`GET /health/rollout`. We still lacked route-level parity coverage ensuring
that both monolith and api-service expose the shared rollout identity metadata
contract over HTTP.

## Problem Statement
Without parity tests at the route level, monolith and api-service could drift in
their shared rollout metadata exposure even if they continue to compile and
produce payloads independently.

## Scope
- Add monolith HTTP integration coverage for `/health/rollout`.
- Extend api-service HTTP integration coverage to assert shared rollout metadata
  fields.
- Verify parity on shared metadata/identity fields while allowing service-local
  rollout body differences.

## Non-Goals
- No rollout logic changes.
- No body semantic alignment beyond shared metadata fields.
- No route or handler behavior changes.

## Selected Approach
- Assert route-level parity for:
  - `schema_family`
  - `report_version`
  - `report_scope`
  - `report_mode`
- Keep service-specific fields like `service`, `report_source`, and
  `deployment_mode` distinct where appropriate.

## Data / Contract Impact
- No external JSON changes.
- Adds route-level regression coverage for shared metadata parity.

## Observability
- Increases confidence that shared rollout identity metadata is consistent
  across deployment modes at the actual HTTP surface.

## Risks
- Low: test-only change.

## Rollback Plan
- Remove the parity tests if they prove too brittle without changing the
  rollout report contract.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase168-rollout-route-parity-metadata.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase168-rollout-route-parity-metadata.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first HTTP-level parity guard for shared rollout identity
  metadata across deployment modes.

## Implementation Notes
- Added monolithic `/health/rollout` route integration coverage.
- Extended api-service route integration to assert shared metadata parity
  against shared rollout identity constants.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase168-rollout-route-parity-metadata.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
