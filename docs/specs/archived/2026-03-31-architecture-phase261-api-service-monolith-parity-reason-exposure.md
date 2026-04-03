# Phase 261 - API Service Monolith Parity Reason Exposure

## Status
Status: Approved

## Why
- The shared ownership parity source, posture, and hint layers were already in
  place, but `api-service` still mostly exercised them through local source
  injection only.
- This phase moves one route-oriented service onto a more realistic
  monolith-backed adapter path without pretending route parity is complete.

## Scope
- Add a small `api-service` adapter that builds a shared route ownership
  parity source from readiness rollout details.
- Expose shared monolith parity posture and hint in the `api-service`
  rollout advisory reason through the existing ownership source path.
- Lock the behavior with focused producer and route integration coverage.

## Implementation
- Added `newAPIServiceRolloutReportProducerWithReadinessDetails(...)`.
- Added `buildAPIServiceOwnershipParitySourceFromReadinessDetails(...)`.
- Updated focused producer coverage to assert:
  - `monolith_parity_posture: ...`
  - `monolith_parity_hint: ...`
- Updated the route integration test to exercise the readiness-backed source
  adapter path instead of only local source injection.

## Validation
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_report_integration_test.go`
- `go test ./pkg/plugins/api/...`

## Exit Criteria
- `api-service` rollout advisory reason can expose shared monolith parity
  posture and hint through a readiness-backed ownership source adapter.
- Producer and route-focused tests lock the new reason surface without
  changing the conservative ownership parity decision.
