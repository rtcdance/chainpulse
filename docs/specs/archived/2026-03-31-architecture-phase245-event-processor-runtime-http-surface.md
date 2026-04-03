# Phase 245 - Event Processor Runtime HTTP Surface

## Status
Status: Approved

## Why
- `event-processor` already had a real rollout health handler and meaningful
  runtime-derived rollout state.
- But that surface still mostly existed as an internal handler path plus
  focused handler tests.
- The next higher-value slice was to expose a minimal real HTTP runtime health
  surface instead of continuing to treat rollout only as an internal wiring
  concern.

## Scope
- Add a minimal runtime HTTP handler surface for `event-processor`.
- Expose:
  - `/health`
  - `/health/ready`
  - `/health/live`
  - `/health/components`
  - `/health/rollout`
- Keep the slice focused on health/rollout exposure rather than broader
  service-plane routing.

## Implementation
- Add an `event-processor` runtime HTTP mux builder and server helper.
- Wire the runtime HTTP server into `main.go`.
- Add focused HTTP route coverage for `/health/rollout`.
- Update architecture coverage notes accordingly.

## Validation
- Run focused `event-processor` rollout/runtime HTTP tests.
- Run `pkg/plugins/api` tests.

## Exit Criteria
- `event-processor` no longer relies only on direct handler invocation for its
  rollout health surface.
- A minimal real HTTP runtime health surface now exists and is covered with
  focused tests.
