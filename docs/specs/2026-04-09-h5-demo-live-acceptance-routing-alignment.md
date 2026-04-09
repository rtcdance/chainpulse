Title: H5 Demo Live Acceptance Routing Alignment
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: frontend, docs/specs

## Status

Implemented on 2026-04-09. Header remains `Status: Approved` for repository
checker compatibility.

## Problem Statement

The H5 acceptance demo renders in Vite, but its live browser acceptance is not
reliable enough for local review against the current Docker stack.

Observed issues:

1. the demo probes `localhost:8080/8081/8082/8083` directly from the browser,
   which triggers CORS failures instead of acceptance evidence
2. the current-slice service matrix treats all probe targets as direct remote
   origins even during local Vite development, so same-origin proxy support is
   bypassed
3. the WebSocket URL builder assumes `/ws` should be resolved against the live
   service host directly, which prevents a same-origin dev proxy path from being
   used during H5 acceptance

## Scope

This bugfix will:

1. add explicit Vite proxy prefixes for the gateway, api-service,
   event-processor, and puller
2. make the H5 data access layer route service probes through same-origin proxy
   prefixes when running under the Vite dev server
3. make WebSocket URLs use same-origin proxy paths during local H5 development
4. keep non-dev/live-origin behavior unchanged outside the Vite dev server

## Non-Goals

This bugfix will not:

1. change backend API or WebSocket contracts
2. make unavailable live WebSocket routes appear healthy
3. redesign the H5 acceptance UI flow

## Options Considered

1. enable CORS on every backend service
   - rejected because it broadens runtime behavior and is larger than needed for
     local H5 acceptance
2. move H5 acceptance traffic through Vite same-origin proxies
   - selected because it is frontend-local, reversible, and matches the dev
     demo workflow

## Selected Approach

1. add `/__proxy/<service>` HTTP proxy prefixes in `frontend/vite.config.ts`
2. add `/__ws/<service>` WebSocket proxy prefixes in `frontend/vite.config.ts`
3. teach `frontend/src/lib/chainpulse.ts` to emit proxy-based base URLs and
   WebSocket URLs when it detects the Vite dev server origin
4. preserve the existing labels and evidence paths so the operator still sees
   the underlying target endpoint names

## Risks

1. the live stack may still expose missing WebSocket handlers after proxying
2. proxy path mistakes could break the H5 in dev mode if prefixes drift

## Rollback Plan

1. revert the Vite proxy additions
2. revert the H5 URL builder changes in `frontend/src/lib/chainpulse.ts`

## Test Strategy

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-09-h5-demo-live-acceptance-routing-alignment.md`
2. `cd frontend && npm run build`
3. run focused browser acceptance against the local H5 demo and Docker stack

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-09-h5-demo-live-acceptance-routing-alignment.md`
2. `cd frontend && npm run build`
3. focused local H5 acceptance rerun with the updated routing behavior

## Review Notes

Approved as the smallest safe fix that restores actionable H5 acceptance
evidence in local development without changing backend contracts.

## Implementation Summary

Delivered:

1. per-service Vite HTTP proxy prefixes for local H5 acceptance
2. per-service Vite WebSocket proxy prefixes for local H5 acceptance
3. H5 routing logic that automatically uses same-origin proxy paths under the
   Vite dev server

## Verification Summary

Executed:

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-09-h5-demo-live-acceptance-routing-alignment.md`
   - passed before implementation
2. `cd frontend && npm run build`
   - passed after the routing changes
3. focused browser validation against `http://127.0.0.1:3000`
   - `Dashboard`, `Events`, `GraphQL`, and `Metrics` now render live data
     without browser CORS failures
   - the service matrix recovered from cross-origin `Network Error` state to
     live probe evidence, reaching `35/37` ready in the observed run
   - `Runtime` page renders live runtime evidence when exercised directly
   - `WebSocket` acceptance remains blocked by live backend route availability
     rather than browser CORS
4. `scripts/dev-micro-loop.sh --mode fast --base HEAD`
   - blocked by existing repository-wide format drift in unrelated Go files at
     `fmt-check`; not caused by this frontend-only change
