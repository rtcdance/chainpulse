# Phase 264 - Monolith Parity Target Decision

## Status
Status: Approved

## Why
- Route-oriented services already consume shared monolith parity posture and
  hint, but they still need to infer whether the monolith currently represents
  a usable parity target.
- A compact shared target decision keeps that interpretation stable across
  `api-service` and `api-gateway`.

## Scope
- Add a shared monolith parity target decision helper in `pkg/plugins/api`.
- Extend the shared monolith parity reason appender and validator to include
  the compact target decision.
- Move `api-service` and `api-gateway` onto the shared decision output.

## Implementation
- Extended `RouteOwnershipParitySourceSnapshot` with:
  - `MonolithTargetReady`
  - `MonolithTargetDecision`
- Added:
  - `BuildMonolithOwnershipParityTargetDecision(...)`
  - `AppendMonolithOwnershipParityReason(...)`
- Updated readiness-backed source snapshots to populate the compact target
  decision.
- Updated route-oriented services to append the shared monolith parity reason
  bundle instead of manually appending posture and hint fields.

## Validation
- `go test ./pkg/plugins/api/...`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_report_integration_test.go`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_runtime_support.go rollout_runtime_support_test.go main.go`

## Exit Criteria
- Shared monolith-backed ownership parity source now exposes a compact target
  decision in addition to posture and hint.
- `api-service` and `api-gateway` consume the shared target decision through a
  shared reason appender and shared validator.
