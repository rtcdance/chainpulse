# Observability Indexer Health SLI Harmonization Plan

## Title
Align indexer health SLI naming and threshold semantics across monolithic and microservice deployment modes

## Type
- architecture
- observability

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Planned

## Owner
indexer-team

## Reviewers
- Platform SRE
- Indexer Domain Owner

## Date
2026-03-30

## Target Deadline
2026-08-01

## Related Modules
- `pkg/observability/indexer_health.go`
- `pkg/observability/indexer_metrics.go`
- `docs/operations/MIGRATION_MANIFEST.csv`

## Context
Current SLI naming and threshold interpretations differ between monolithic and microservice operational dashboards, complicating incident triage and SLO reporting.

## Problem Statement
Without harmonized SLI contracts, enterprise operations cannot compare health posture consistently across deployment modes.

## Scope
- Define canonical SLI names for:
  - indexing lag
  - processing latency
  - cache hit ratio
  - error rate
- Standardize threshold semantics and alert labels across modes.
- Update dashboards/runbooks after implementation phase.

## Non-Goals
- No immediate runtime behavior changes in this planning spec.
- No storage backend migration.

## Risks
- Naming changes can temporarily duplicate dashboard panels.

## Rollback Plan
Retain legacy labels until dashboard migration completion.

## Test and Verification Plan
- Spec approval gate only in planning phase.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-observability-indexer-health-sli-harmonization-plan.md`
