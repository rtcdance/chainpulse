# Phase 165 API Service Rollout Producer Skeleton

## Title
Phase 165 - Add api-service rollout report producer skeleton

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
- `cmd/microservices/api-service/rollout_report_producer.go`
- `cmd/microservices/api-service/rollout_report_producer_test.go`
- `cmd/microservices/api-service/main.go`
- `pkg/plugins/api/rollout_report_contract.go`
- `docs/specs/2026-03-31-architecture-phase164-rollout-report-producer-interface.md`

## Context
Phase 164 introduced a shared rollout report producer interface so `/health/rollout`
 no longer depends on handler-local callback registration. At that point only
the monolith actually implemented the producer boundary.

## Problem Statement
Without a second producer, the rollout report contract is still effectively a
monolith-only surface and cannot yet demonstrate cross-deployment reuse.

## Scope
- Add a minimal rollout report producer skeleton for `cmd/microservices/api-service`.
- Wire the skeleton into the existing health handler.
- Keep the payload explicit about being a skeleton with no real ownership
  runtime state yet.

## Non-Goals
- No real microservice ownership rollout logic yet.
- No API gateway or puller/event-processor rollout producers yet.
- No rollout enforcement behavior changes.

## Selected Approach
- Implement a minimal `api-service` producer returning a typed rollout report
  with stable metadata and clearly marked `unknown/investigate` rollout states.
- Register it on the shared health handler during `api-service` startup.

## Data / Contract Impact
- `/health/rollout` becomes available from `api-service` with a skeleton report.
- Existing monolithic payload contract remains unchanged.

## Observability
- Establishes the second concrete rollout report producer and makes cross-mode
  report reuse visible without pretending the microservice has rollout state it
  does not yet own.

## Risks
- Low: skeleton states must remain clearly labeled so consumers do not confuse
  them with real ownership rollout signals.

## Rollback Plan
- Remove the api-service skeleton producer wiring while keeping the shared
  producer interface intact.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase165-api-service-rollout-producer-skeleton.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase165-api-service-rollout-producer-skeleton.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first non-monolithic rollout report producer, with explicit
  skeleton semantics and no claim of real ownership rollout state.

## Implementation Notes
- Added an api-service rollout producer skeleton.
- Wired the skeleton producer into the existing health handler.
- Added a focused producer test covering metadata and skeleton state posture.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase165-api-service-rollout-producer-skeleton.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-service/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
