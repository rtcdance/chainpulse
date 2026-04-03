# Phase 311 - Shared Health Runtime Summary

## Status
Status: Approved

## Summary

Extend the shared health helper from raw summary counts to a compact runtime
summary with posture and a reliability hint.

## Problem

The shared health helper already exposes raw summary counts, but callers still
have to interpret those counts manually to determine whether the health runtime
is healthy, degraded, unhealthy, or still unobserved.

## Decision

Add `GetRuntimeSummary()` to `HealthCheck` and expose:

- `overall_status`
- `total_components`
- `healthy_count`
- `degraded_count`
- `unhealthy_count`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no component-level contract change
- no scheduler redesign
- only shared runtime summary surfacing

## Scope

In scope:

- shared health runtime summary
- compact health posture and hint
- focused shared health tests

Out of scope:

- health scheduling redesign
- per-component hint generation
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestHealthCheckRuntimeSummaryHealthy|TestHealthCheckRuntimeSummaryDegraded|TestHealthCheckRuntimeSummaryUnhealthy|TestHealthCheckRuntimeSummaryUnobserved'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase311-shared-health-runtime-summary.md`

## Exit Criteria

- `HealthCheck` exposes a compact runtime summary on top of the existing raw
  counts.
- Focused health tests confirm healthy, degraded, unhealthy, and unobserved posture classification.
