# Phase 25 Policy Metric Versioning and Deprecation

## Title
Phase 25 - Add policy metric schema versioning and deprecation workflow for zero-downtime observability migrations

## Type
- architecture

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

## Target Deadline
2026-07-01

## Related Modules
- `pkg/application/bootstrap/core_config_override_policy.go`
- `pkg/application/bootstrap/core_config_override_policy_test.go`
- `cmd/microservices/api-service/main.go`
- `cmd/monolithic/chainpulse/main.go`
- `scripts/check-policy-metric-contract.sh`
- `docs/operations/POLICY_METRIC_VERSIONING.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 24 added policy metric contract CI gates, but metric-name evolution still lacked first-class dual-write and deprecation workflow support.

## Problem Statement
Changing metric names without migration controls can break dashboards/alerts during policy platform upgrades.

## Scope
- Add policy metric schema mode env:
  - `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE`
  - values: `v1`, `dual_write`, `v2`
- Add shared metric emission helper with schema plan:
  - v1 only
  - dual-write (v1+v2)
  - v2 only
- Add schema tags:
  - `metric_schema_version`
  - `metric_schema_deprecated`
- Route startup metric emission through shared helper in both startup paths.
- Add tests for schema mode resolution and emission behavior.
- Extend contract-check script with schema mode regression tests.
- Add operations doc for metric migration/deprecation workflow.

## Non-Goals
- No policy logic behavior changes.
- No hard removal of v1 metrics in this phase.

## Options Considered
- Option A: one-time metric rename.
- Option B: dual-write migration workflow with explicit schema mode control.

## Selected Approach
Choose Option B for enterprise-safe dashboard migration.

## Data / Contract Impact
Adds new optional v2 metric names and schema tags; v1 remains available unless switched to `v2`.

## Risks
- Duplicate metric streams in dual-write may increase cardinality/cost.
- Mitigation: migration window only; default remains single-schema.

## Rollback Plan
Set `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE=v1` to return to legacy-only emission.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase25-policy-metric-versioning-and-deprecation.md`
- `./scripts/check-policy-metric-contract.sh`
- `go test -short ./pkg/application/bootstrap/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase25-policy-metric-versioning-and-deprecation.md`
- `./scripts/check-policy-metric-contract.sh`

## Review Notes
- Approved for zero-downtime metric migration support.

## Implementation Summary
- Added schema mode resolver, dual-write emission helper, v2 metric names, and migration runbook doc.

## Final Verification
- Contract checks and bootstrap package tests pass with schema mode coverage.
