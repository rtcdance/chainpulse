# Phase 186 - API Service Additional Runtime Route Signals

## Status
Status: Approved

## Why
- The recent rollout refactors brought the report contract and builders into a
  stable shape.
- The next higher-value step is to increase the amount of real runtime state
  that `api-service` contributes to `/health/rollout`, instead of continuing to
  only polish structure.

## Scope
- Keep the rollout report contract unchanged.
- Add more real `api-service` runtime wiring signals derived from the gateway
  plugin itself.

## Implementation
- Add `APIGatewayPlugin` getters for:
  - event subscription runtime handler wiring
  - health check runtime handler wiring
- Feed those signals into the `api-service` rollout producer.
- Update completeness/reason logic and route-level expectations accordingly.

## Validation
- Add gateway plugin toggle coverage for the new getters.
- Update rollout producer and route integration tests.
- Run Go tests and the fast micro-loop gate.

## Exit Criteria
- `api-service` rollout report reflects additional real runtime route signals.
- The rollout contract stays unchanged while the report becomes more truthful
  about local runtime wiring completeness.
