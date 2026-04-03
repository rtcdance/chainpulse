# Phase 246 - Puller Runtime HTTP Surface

## Status
Status: Approved

## Why
- Phase 245 promoted `event-processor` from handler-only rollout exposure to a
  minimal real HTTP runtime health surface.
- `puller` still had the same gap:
  meaningful rollout/health logic existed, but mostly behind internal handler
  paths and focused handler tests.
- The next useful symmetrical slice was to give `puller` the same minimal HTTP
  runtime health surface.

## Scope
- Add a minimal runtime HTTP handler surface for `puller`.
- Expose:
  - `/health`
  - `/health/ready`
  - `/health/live`
  - `/health/components`
  - `/health/rollout`
- Keep the slice focused on health/rollout exposure rather than broader
  service-plane routing.

## Implementation
- Add a `puller` runtime HTTP mux builder and server helper.
- Wire the runtime HTTP server into `main.go`.
- Add focused HTTP route coverage for `/health/rollout`.
- Update architecture coverage notes accordingly.

## Validation
- Run focused `puller` rollout/runtime HTTP tests.
- Run `pkg/plugins/api` tests.

## Exit Criteria
- `puller` no longer relies only on direct handler invocation for its rollout
  health surface.
- A minimal real HTTP runtime health surface now exists and is covered with
  focused tests.
