# Phase 120 Ownership Rollout Readiness Surface

## Title
Phase 120 - Add ownership rollout readiness details to monolithic `/health/ready`

## Type
- architecture
- indexing
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
2026-03-30

## Related Modules
- `pkg/plugins/api/health_check_handler.go`
- `pkg/plugins/api/health_check_handler_test.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase119-ownership-health-surface.md`

## Context
Phase 119 exposed ownership rollout state in `/health`, but operators still
need to infer whether that state is only informative (`shadow`) or actually
ready for a deliberate move toward `runtime-owned`.

## Problem Statement
Without a rollout-readiness detail surface, `/health/ready` only answers
infrastructure readiness and cannot distinguish "service is alive and shadowing"
from "service is prepared for the next ownership-shift step."

## Scope
- Extend readiness responses with optional rollout details.
- Add an additive readiness details provider hook to the health handler.
- Expose ownership rollout readiness details in monolithic mode, including:
  - `ownership_mode`
  - `rollout_ready_for_runtime_owned`
  - `rollout_status`
  - `rollout_reason`
- Keep existing readiness HTTP status semantics unchanged.
- Add focused tests for:
  - readiness details provider wiring
  - monolithic ownership rollout readiness classification

## Non-Goals
- No change to Kubernetes readiness status-code behavior.
- No automatic ownership cutover.
- No microservice rollout wiring.

## Selected Approach
- Keep infrastructure readiness and rollout readiness separate.
- Continue returning `ready=true|false` based on critical dependency health.
- Attach rollout-readiness metadata as an additive details map in the readiness
  payload so operators and future automation can inspect it without changing
  existing probe expectations.

## Data / Contract Impact
- `/health/ready` may now include a `details` object.
- Existing readiness fields remain unchanged.

## Observability
- Ownership rollout state is now visible in:
  - terminal summary
  - runtime metrics
  - `/health`
  - `/health/ready` rollout details

## Risks
- Low risk; additive payload only.

## Rollback Plan
- Remove readiness detail provider and monolithic ownership rollout readiness
  wiring.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase120-ownership-rollout-readiness-surface.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase120-ownership-rollout-readiness-surface.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as an additive rollout-readiness layer that does not disturb current
  readiness probe semantics.

## Implementation Summary
- Extended `ReadinessResponse` with optional `details`.
- Added an optional readiness-details provider to `HealthCheckHandler`.
- Monolithic mode now injects ownership rollout readiness details into
  `/health/ready`, including:
  - `ownership_mode`
  - `rollout_ready_for_runtime_owned`
  - `rollout_status`
  - `rollout_reason`
- Added focused tests for readiness detail injection and monolithic rollout
  readiness classification.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase120-ownership-rollout-readiness-surface.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
