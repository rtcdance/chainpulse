# Phase 265 - Monolith Parity Action Guidance

## Status
Status: Approved

## Why
- Route-oriented services already consume shared monolith parity posture, hint,
  and target decision.
- The next low-risk improvement is a shared action-guidance layer so those
  services can emit the same next-step recommendation instead of each service
  interpreting the target decision independently later.

## Scope
- Add a shared monolith parity action-guidance helper in `pkg/plugins/api`.
- Extend shared monolith parity reason assembly and validation to include the
  compact action guidance.
- Update `api-service` and `api-gateway` focused rollout tests to lock the new
  route-oriented reason surface.

## Implementation
- Extended `RouteOwnershipParitySourceSnapshot` with:
  - `MonolithActionGuidance`
- Added:
  - `BuildMonolithOwnershipParityActionGuidance(...)`
- Populated readiness-backed source snapshots with the shared guidance.
- Extended `AppendMonolithOwnershipParityReason(...)` and
  `ValidateRouteMonolithOwnershipParityReason(...)` to carry:
  - `monolith_parity_action_guidance`

## Validation
- `go test ./pkg/plugins/api/...`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_report_integration_test.go`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_runtime_support.go rollout_runtime_support_test.go main.go`

## Exit Criteria
- Route-oriented monolith-backed parity source now exposes a shared action
  guidance layer in addition to posture, hint, and target decision.
- `api-service` and `api-gateway` emit the same shared action guidance through
  the shared monolith parity reason path.
