# Phase 164 Rollout Report Producer Interface

## Title
Phase 164 - Add shared rollout report producer interface

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
- `pkg/plugins/api/rollout_report_contract.go`
- `pkg/plugins/api/health_check_handler.go`
- `pkg/plugins/api/health_check_handler_test.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/specs/2026-03-31-architecture-phase163-rollout-report-section-assembler.md`

## Context
Phase 163 aligned monolithic rollout report assembly with the same sectioned
pattern used by the rest of the rollout control stack. The rollout report
surface is still wired primarily through a raw function provider rather than a
shared producer abstraction.

## Problem Statement
Without a shared producer interface, future rollout report producers in other
deployment modes still have to fit themselves into handler-local function
callbacks instead of implementing a reusable report contract boundary.

## Scope
- Add a shared `RolloutReportProducer` interface.
- Add a function adapter for the existing callback-based usage.
- Let the health handler accept a producer directly while preserving the
  existing provider setter for backward compatibility.
- Update monolithic wiring to use the producer abstraction.

## Non-Goals
- No rollout logic changes.
- No report schema changes.
- No route or readiness changes.

## Selected Approach
- Introduce `RolloutReportProducer` and `RolloutReportProducerFunc` in the API
  rollout contract layer.
- Keep `SetRolloutReportProvider(...)` as a compatibility wrapper around the new
  producer setter.
- Migrate monolithic wiring to `SetRolloutReportProducer(...)`.

## Data / Contract Impact
- No external JSON changes.
- Internal rollout report production now has a reusable producer boundary.

## Observability
- Preserves existing rollout report behavior while making future producer reuse
  more explicit and testable.

## Risks
- Low: mostly wiring refactor risk, mitigated by compatibility wrapper and
  tests.

## Rollback Plan
- Revert the producer interface and return to callback-only rollout report
  registration without changing the external `/health/rollout` surface.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase164-rollout-report-producer-interface.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase164-rollout-report-producer-interface.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first shared producer boundary for rollout report generation
  across deployment modes.

## Implementation Notes
- Added `RolloutReportProducer` and `RolloutReportProducerFunc`.
- Added direct producer registration to the health handler while keeping the old
  provider setter as a compatibility layer.
- Updated monolithic wiring to use the producer interface.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase164-rollout-report-producer-interface.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
