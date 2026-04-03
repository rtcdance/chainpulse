# Phase 353 - Shared Connection Pool Coverage Posture Alignment

## Status
Status: Implemented

## Summary

Align shared connection-pool runtime metrics to expose explicit
`coverage_posture` while preserving existing `capacity_posture` compatibility.

## Problem

`ConnectionPool.GetRuntimeMetrics()` currently exposes `capacity_posture` as
its primary coverage-like signal, which diverges from the broader runtime
metrics contract that uses `coverage_posture` across components.

## Decision

Extend `GetRuntimeMetrics()` to include:

- `coverage_posture` (aligned with existing pool capacity posture semantics)
- existing `capacity_posture` retained for backward compatibility

Keep the change intentionally small:

- no pool lifecycle redesign
- no acquire/release contract change
- only runtime metrics field alignment

## Scope

In scope:

- connection-pool runtime coverage field alignment
- compatibility preservation for existing capacity posture consumers
- focused runtime metrics tests

Out of scope:

- connection eviction or cleanup redesign
- factory behavior changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestConnectionPoolRuntimeMetricsHealthy|TestConnectionPoolRuntimeMetricsUnobserved|TestConnectionPoolRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase353-shared-connection-pool-coverage-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestConnectionPoolRuntimeMetricsHealthy|TestConnectionPoolRuntimeMetricsUnobserved|TestConnectionPoolRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase353-shared-connection-pool-coverage-posture-alignment.md` passed.

## Exit Criteria

- `ConnectionPool` runtime metrics expose `coverage_posture` and retain
  `capacity_posture` compatibility.
- Focused tests confirm posture values for healthy/unobserved/degraded states.
