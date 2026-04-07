Title: H5 Acceptance Demo Console
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: frontend, scripts/mock-server, test/acceptance, docs/specs

## Status

Implemented. Header remains `Status: Approved` for repository checker
compatibility.

## Problem Statement

The repository already includes a `frontend` H5 shell intended for ChainPulse
acceptance, but it is not yet reliable enough to validate the current runnable
surface end to end.

Current gaps:

1. hard-coded endpoints and field names do not consistently match the current
   gateway/runtime contracts
2. the UI modules are fragmented and do not provide a clear acceptance flow
3. degraded, empty, and mock-data states are not handled well enough for demos
4. the demo does not currently prove all major acceptance capabilities from one
   place

## Scope

This slice will:

1. turn `frontend` into a single H5 acceptance console for ChainPulse
2. cover these verification surfaces in the UI:
   - health and component posture
   - current-slice service coverage for `api-gateway`, `api-service`,
     `event-processor`, and `puller`
   - rollout, readiness, liveness, and runtime-control surfaces where exposed
   - event querying and filtering
   - single event detail inspection
   - GraphQL querying
   - WebSocket connection and live message stream
   - Prometheus metrics inspection
   - runtime summary inspection
3. add a small front-end data access layer that can tolerate current route and
   payload differences between mock/demo and runnable backends
4. make all user-facing copy in the H5 console English
5. support a clear operator flow for demo acceptance on desktop and mobile
5. verify the H5 with a local mock server plus front-end build checks

## Non-Goals

This slice will not:

1. change ChainPulse backend API contracts
2. introduce authentication, RBAC, or release-only deployment logic
3. build a production marketing site
4. replace the existing Playwright acceptance suite with browser-driven H5 tests

## Selected Approach

Build the H5 as an acceptance control console instead of a generic dashboard.

1. add a central API/config adapter in `frontend/src` to normalize:
   - base URLs
   - `/events` vs `/api/v1/events`
   - `event_name` vs `eventName`
   - pagination/result envelopes
2. refactor the UI into clear acceptance sections with explicit request status,
   endpoint visibility, and evidence panels
3. add a current-slice service matrix that probes key endpoints on the default
   local ports:
   - `api-gateway` `:8080`
   - `api-service` `:8081`
   - `event-processor` `:8082`
   - `puller` `:8083`
4. make WebSocket, GraphQL, metrics, and runtime views executable from the UI
   with useful defaults and copyable evidence
5. keep the interface responsive and mobile-friendly so it can be used during
   demos on a phone browser
6. use `scripts/mock-server/main.go` as the deterministic verification backend
   for front-end build and interaction validation

## Risks

1. backend route drift between monolithic, gateway, and mock modes may still
   expose unmodeled payload differences
2. WebSocket behavior may differ between the mock server and the real gateway
3. the existing `frontend` workspace may already contain unreviewed local edits

## Rollback Plan

1. revert `frontend/src` back to the current dashboard implementation
2. remove any new front-end-only helper files added for the acceptance console
3. leave backend services and existing acceptance tests unchanged

## Test Strategy

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-07-h5-acceptance-demo.md`
2. `cd frontend && npm run build`
3. run `scripts/mock-server/main.go` locally and validate the H5 against it

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-07-h5-acceptance-demo.md`
2. `cd frontend && npm run build`
3. run a focused front-end verification loop against the mock server

## Review Notes

Approved as the smallest useful slice that turns the existing H5 shell into a
real acceptance console for the current ChainPulse runnable surface.

## Implementation Summary

Delivered:

1. a refactored H5 acceptance console in `frontend`
2. a shared front-end adapter for route and payload normalization
3. responsive acceptance views for health, events, GraphQL, WebSocket,
   metrics, and runtime evidence
4. an English-only operator flow plus current-slice service matrix coverage
5. default support for `api-gateway`, `api-service`, `event-processor`, and
   `puller` local acceptance surfaces

## Verification Summary

Executed:

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-07-h5-acceptance-demo.md`
   - passed
2. `cd frontend && npm run build`
   - passed
   - passed again after the English-only/current-slice expansion
3. `scripts/dev-micro-loop.sh --mode fast --base HEAD`
   - failed in existing repo/tooling state:
     - `gofumpt: command not found`
     - `golangci-lint` config/version mismatch
4. live local endpoint checks against the current `localhost:8080` service
   - `GET /health` returned mock health payload
   - `GET /events?limit=3&offset=0&event_name=Transfer` returned event list
   - `GET /runtime/summary` returned runtime payload
   - `GET /metrics` returned `chainpulse_*` samples
   - `POST /graphql` introspection returned schema data
