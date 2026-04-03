# Phase 172 - API Service Runtime Wiring Completeness Helper

## Status
Status: Approved

## Why
- Phase 169 and Phase 171 made `api-service` rollout state more truthful and
  more readable.
- The producer now owns repeated logic for:
  - `partially-wired` vs `runtime-wired`
  - enabled/missing runtime signal lists
  - shared runtime-derived reason strings
- That logic should live behind a small helper before we keep adding more
  microservice rollout signals.

## Scope
- Keep `/health/rollout` payload shape and field values unchanged.
- Extract a small completeness helper for runtime wiring classification.

## Implementation
- Add a helper that returns:
  - rollout `mode`
  - advisory status
  - enabled signal list
  - missing signal list
  - shared reason string
- Make the producer consume that helper instead of re-deriving the same pieces
  inline.

## Validation
- Add helper-focused unit coverage.
- Run api-service tests, shared tests, and the fast micro-loop gate.

## Exit Criteria
- Producer no longer owns inline runtime wiring completeness assembly.
- Existing runtime-derived rollout states and reasons remain unchanged.
