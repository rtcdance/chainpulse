# Phase 349 - Shared Monitoring Runtime Surface Alignment

## Status
Status: Implemented

## Summary

Align shared monitoring runtime surfacing with the common no-arg
`GetRuntimeMetrics()` contract while retaining protocol-scoped metrics access.

## Problem

Shared monitoring currently exposes runtime metrics only via
`GetRuntimeMetrics(protocol string)`, which diverges from the common
component-level runtime metrics shape used across shared/core modules.

## Decision

Refactor monitoring runtime APIs to:

- expose aggregate no-arg `GetRuntimeMetrics()`
- preserve protocol-scoped view through `GetProtocolRuntimeMetrics(protocol)`

Keep the change intentionally small:

- no monitoring data model redesign
- no request recording contract change
- only runtime metrics surface alignment

## Scope

In scope:

- monitoring runtime metrics API alignment
- aggregate runtime posture/coverage surfacing
- focused monitoring runtime tests

Out of scope:

- metrics storage redesign
- alerting policy redesign
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestMonitoringRuntimeMetricsUnobserved|TestMonitoringRuntimeMetricsHealthy|TestMonitoringRuntimeMetricsDegraded|TestMonitoringAggregateRuntimeMetricsUnobserved|TestMonitoringAggregateRuntimeMetricsHealthy|TestMonitoringAggregateRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase349-shared-monitoring-runtime-surface-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestMonitoringRuntimeMetricsUnobserved|TestMonitoringRuntimeMetricsHealthy|TestMonitoringRuntimeMetricsDegraded|TestMonitoringAggregateRuntimeMetricsUnobserved|TestMonitoringAggregateRuntimeMetricsHealthy|TestMonitoringAggregateRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase349-shared-monitoring-runtime-surface-alignment.md` passed.

## Exit Criteria

- `Monitoring` exposes no-arg aggregate `GetRuntimeMetrics()` with compact runtime semantics.
- Protocol-scoped runtime metrics remain available via explicit API.
- Focused tests confirm both protocol-scoped and aggregate runtime posture coverage.
