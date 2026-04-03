# Phase 24 Policy Metric Contract CI Gate

## Title
Phase 24 - Add CI gate for policy metric/tag contract stability

## Type
- architecture

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `pkg/application/bootstrap/core_config_override_policy.go`
- `pkg/application/bootstrap/core_config_override_policy_test.go`
- `scripts/check-policy-metric-contract.sh`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `docs/ARCHITECTURE.md`

## Context
Phase 23 completed policy rollout operations docs, but metric/tag schema stability was not yet enforced as an explicit CI gate.

## Problem Statement
Unreviewed policy metric/tag schema drift can silently break dashboards, alerts, and runbook assumptions.

## Scope
- Add explicit policy metric contract test for:
  - `BuildOverrideAuditTags` key set
  - `BuildPolicyEvaluationTags` key set
- Add dedicated contract-check script:
  - `scripts/check-policy-metric-contract.sh`
- Wire CI workflow gate:
  - `.github/workflows/ci.yml` job `policy-contract`
- Wire full local micro-loop gate:
  - `scripts/dev-micro-loop.sh` full mode
- Replace duplicated policy evaluation tag assembly in startup code with shared helper.

## Non-Goals
- No changes to runtime policy semantics.
- No external monitoring vendor integration.

## Options Considered
- Option A: rely on general unit tests only.
- Option B: add dedicated CI gate and explicit contract test.

## Selected Approach
Choose Option B to make metric schema compatibility checks visible and mandatory.

## Data / Contract Impact
No runtime API changes. Internal telemetry contract now has an explicit stability gate.

## Risks
- Legitimate metric/tag evolution now requires contract test updates.
- Mitigation: intentional review/update flow is desired for observability stability.

## Rollback Plan
Remove CI job and script wiring, keep existing runtime metrics unchanged.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase24-policy-metric-contract-ci-gate.md`
- `./scripts/check-policy-metric-contract.sh`
- `go test -short ./pkg/application/bootstrap/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase24-policy-metric-contract-ci-gate.md`
- `./scripts/check-policy-metric-contract.sh`

## Review Notes
- Approved to prevent policy observability regressions from silent schema drift.

## Implementation Summary
- Added contract test + script + CI job and shared policy evaluation tag builder.

## Final Verification
- Contract script and bootstrap package tests pass; CI workflow includes policy contract gate.
