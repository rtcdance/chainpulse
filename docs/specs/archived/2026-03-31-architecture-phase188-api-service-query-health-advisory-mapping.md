# Phase 188 - API Service Query Health Advisory Mapping

## Status
Status: Approved

## Why
- Phase 187 folded query runtime health into the `api-service` rollout report,
  but the health signal only surfaced through `advisory.ready` and reason text.
- The next useful refinement is to make fully wired-but-degraded query runtime
  states explicit in `advisory.status`, without disturbing the current partial
  wiring semantics.

## Scope
- Keep the rollout report contract unchanged.
- Refine `api-service` advisory status mapping for fully wired runtime states.

## Implementation
- Add a small helper that maps query runtime health to advisory status for the
  fully wired case:
  - `healthy -> runtime-wired`
  - `degraded -> runtime-wired-degraded`
  - `unhealthy -> runtime-wired-unhealthy`
  - fallback -> `runtime-wired-query-unknown`
- Preserve `partial-runtime-wiring` behavior for partially wired cases.

## Validation
- Add producer coverage for fully wired + degraded query runtime.
- Add completeness helper coverage for fully wired degraded mapping.
- Run Go tests and the fast micro-loop gate.

## Exit Criteria
- `api-service` rollout report distinguishes fully wired healthy versus fully
  wired degraded query-runtime states via `advisory.status`.
- Partial wiring semantics remain unchanged.
