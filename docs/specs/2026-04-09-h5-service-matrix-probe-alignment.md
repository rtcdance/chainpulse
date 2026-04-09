Title: H5 Service Matrix Probe Alignment
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: frontend/src/lib/chainpulse.ts, frontend/src/components/Dashboard.tsx, frontend/src/components/Runtime.tsx

## Status

Approved for implementation.

## Problem Statement

The H5 service matrix still reports two misleading failures:

1. `api-service /ws` returns `Bad Request` because the matrix probes it with a
   plain HTTP GET instead of a WebSocket upgrade
2. `puller /status` returns `404` because the runtime no longer exposes that
   route

These failures make the H5 acceptance surface look unhealthy even when the
actual live services are behaving as expected.

## Scope

This bugfix will:

1. stop treating `api-service /ws` as an HTTP readiness probe
2. replace the stale `puller /status` probe with a live runtime endpoint
3. keep the dedicated WebSocket acceptance page responsible for WebSocket
   verification

## Non-Goals

This bugfix will not:

1. add a new HTTP shim for `/ws`
2. add a new `/status` endpoint to puller
3. redesign the H5 service matrix UI

## Selected Approach

1. remove `api-service /ws` from the service-matrix endpoint map
2. replace `puller /status` with `puller /runtime/control`
3. keep WebSocket validation on the dedicated H5 WebSocket page

## Risks

1. low risk because the change only aligns the matrix with current contracts

## Rollback Plan

1. revert the endpoint map update in `frontend/src/lib/chainpulse.ts`

## Test Strategy

1. `bash scripts/spec-approval-check.sh docs/specs/2026-04-09-h5-service-matrix-probe-alignment.md`
2. `cd frontend && npm run build`
3. verify the H5 service matrix no longer shows the stale `Bad Request` and
   `404` probe entries

