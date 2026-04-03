# Phase 167 API Service Rollout Route Integration

## Title
Phase 167 - Add api-service `/health/rollout` integration test

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
- `cmd/microservices/api-service/rollout_report_integration_test.go`
- `cmd/microservices/api-service/rollout_report_producer.go`
- `pkg/plugins/api/health_check_handler.go`
- `docs/specs/2026-03-31-architecture-phase165-api-service-rollout-producer-skeleton.md`

## Context
Phase 165 added an api-service rollout producer skeleton, but coverage still
only proved the producer in isolation. The HTTP route path was not yet
validated from the api-service side.

## Problem Statement
Without an integration test for `/health/rollout`, the api-service skeleton
producer could regress at route wiring time even if its unit test keeps
passing.

## Scope
- Add a focused api-service integration test covering `GET /health/rollout`.
- Verify that the skeleton producer is visible through the shared runtime route
  integration layer.

## Non-Goals
- No rollout logic changes.
- No new microservice ownership state.
- No route behavior changes.

## Selected Approach
- Compose a lightweight gateway runtime integration in the api-service package.
- Register the api-service rollout producer skeleton on the shared health
  handler.
- Assert the HTTP response exposes the expected skeleton rollout posture.

## Data / Contract Impact
- No external JSON changes.
- Adds regression coverage for the api-service route path.

## Observability
- Improves confidence that the second rollout report producer is not only
  constructed but actually reachable over HTTP.

## Risks
- Low: test-only change plus a tiny test helper on the health handler.

## Rollback Plan
- Remove the integration test and test helper if the approach proves too
  coupled, without changing runtime behavior.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase167-api-service-rollout-route-integration.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase167-api-service-rollout-route-integration.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first HTTP-path validation for the non-monolithic rollout
  report producer.

## Implementation Notes
- Added `cmd/microservices/api-service/rollout_report_integration_test.go`.
- Added a small health-handler test helper for focused route tests.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase167-api-service-rollout-route-integration.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
