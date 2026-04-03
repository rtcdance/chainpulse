# Phase 367 - Shared Connection Pool Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `ConnectionPool.GetMetrics()` with the compact posture-oriented runtime
surface while preserving existing pool counter compatibility.

## Problem

Shared connection pool already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetMetrics()` still receive only
raw pool counters. Those callers do not get the same compact posture and
reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `capacity_posture` (compatibility)
- `runtime_posture`
- `reliability_hint`

Keep existing pool metrics fields unchanged:

- `pool_name`
- `created`
- `reused`
- `closed`
- `errors`
- `current_size`
- `max_size`
- `available`

Keep the change intentionally small:

- no pool lifecycle redesign
- no acquire/release contract change
- only legacy pool metrics alignment

## Scope

In scope:

- shared connection-pool legacy metrics posture alignment
- compatibility preservation for existing pool counter consumers
- focused pool metrics tests

Out of scope:

- connection eviction redesign
- factory behavior changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestConnectionPoolMetricsIncludesPostureFields|TestConnectionPoolRuntimeMetricsHealthy|TestConnectionPoolRuntimeMetricsUnobserved|TestConnectionPoolRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase367-shared-connection-pool-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestConnectionPoolMetricsIncludesPostureFields|TestConnectionPoolRuntimeMetricsHealthy|TestConnectionPoolRuntimeMetricsUnobserved|TestConnectionPoolRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase367-shared-connection-pool-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing pool counter fields remain compatible.
- Focused tests confirm aligned posture values for healthy/unobserved/degraded states.
