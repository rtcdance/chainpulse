# Phase 325 - Shared Middleware Registry Runtime Metrics

## Status
Status: Approved

## Summary

Extend the shared middleware registry from grouped middleware presence to a
compact runtime metrics surface with coverage posture, runtime posture, and a
reliability hint.

## Problem

The shared middleware registry already exposes grouped middleware composition,
but callers still have to inspect security, observability, performance, and
error-handling presence manually to decide whether the registry is unconfigured,
partial, or ready.

## Decision

Add `GetRuntimeMetrics()` to `MiddlewareRegistry` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no middleware execution redesign
- no route contract change
- only shared runtime metrics surfacing

## Scope

In scope:

- shared middleware-registry runtime metrics
- compact coverage/runtime posture
- focused middleware tests

Out of scope:

- middleware execution redesign
- per-route middleware diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestMiddlewareRegistryRuntimeMetricsUnconfigured|TestMiddlewareRegistryRuntimeMetricsPartial|TestMiddlewareRegistryRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase325-shared-middleware-registry-runtime-metrics.md`

## Exit Criteria

- `MiddlewareRegistry` exposes a compact runtime metrics surface beyond grouped middleware presence.
- Focused middleware tests confirm unconfigured, partial, and ready posture classification.
