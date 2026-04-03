# Phase 169 - API Service Runtime-Derived Rollout State

## Status
Status: Approved

## Why
- `api-service` already exposes a rollout report producer, but the payload is
  still a pure skeleton with `unknown/investigate` posture.
- The service now has stable local runtime signals such as runtime route
  composition, event query wiring, and domain bridge wiring that can be safely
  reflected without pretending ownership runtime parity is already complete.

## Scope
- Keep the shared `/health/rollout` contract unchanged.
- Upgrade `api-service` rollout production from static skeleton values to a
  runtime-derived posture when local API runtime wiring is present.
- Keep ownership-specific rollout state explicitly non-authoritative until a
  true ownership runtime source is wired.

## Implementation
- Extend `newAPIServiceRolloutReportProducer(...)` with a lightweight runtime
  state provider that reports:
  - `runtime_routes_enabled`
  - `event_query_enabled`
  - `domain_bridge_enabled`
- Derive rollout posture conservatively:
  - fully wired local runtime => `mode=runtime-wired`
  - partially wired local runtime => `mode=partially-wired`
  - no local runtime signal => existing skeleton fallback
- Register the producer after `APIGatewayPlugin.Initialize(...)` so the runtime
  state provider can read real gateway wiring flags.

## Validation
- Add producer unit coverage for runtime-derived state.
- Update `api-service` route integration coverage to assert the new partial
  runtime posture over `/health/rollout`.
- Run focused api-service tests, shared api/monolith/api-service tests, and the
  fast micro-loop gate.

## Exit Criteria
- `api-service` no longer reports pure `unknown/investigate` when runtime route
  wiring is actually present.
- Ownership-specific fields remain conservative and do not claim full parity.
- Existing shared metadata parity across monolith and api-service remains intact.
