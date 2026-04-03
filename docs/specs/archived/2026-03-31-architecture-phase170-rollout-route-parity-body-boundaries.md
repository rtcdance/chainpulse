# Phase 170 - Rollout Route Parity Body Boundaries

## Status
Status: Approved

## Why
- Phase 168 locked shared rollout metadata parity at the HTTP route layer.
- Phase 169 made the `api-service` report more truthful with runtime-derived
  local wiring state.
- We now need one more guardrail: a small set of body-level semantics that stay
  aligned across monolith and api-service without pretending the full payloads
  are identical.

## Scope
- Keep service-specific rollout body differences intact.
- Add HTTP-level parity checks for a minimal shared semantic boundary:
  - `progression.state`
  - `cutover_dry_run.action`
  - `guarded_cutover.overview.state`

## Implementation
- Extend the monolith `/health/rollout` route integration test to assert the
  shared body boundary fields.
- Extend the api-service `/health/rollout` route integration test to assert the
  same body boundary fields.
- Keep the parity layer intentionally small and conservative.

## Validation
- Run shared api/monolith/api-service tests.
- Run the fast micro-loop gate.

## Exit Criteria
- Monolith and api-service both prove, at the HTTP route layer, that they
  currently share:
  - `progression.state=observe`
  - `cutover_dry_run.action=would-hold`
  - `guarded_cutover.overview.state=hold`
- Shared metadata parity from Phase 168 remains intact.
