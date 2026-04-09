## Status
Status: Approved

## Summary

The microservice API gateway currently reports `domain_bridge_enabled=false` unless a local domain query service is injected. In the current architecture, gateway query routes are actually backed by the upstream api-service bridge, so rollout readiness remains stuck at `partial-runtime-wiring` even when the live query bridge is configured and healthy.

## Decision

- Treat a configured upstream query bridge as satisfying the gateway domain-bridge signal.
- Keep local domain query service support intact for monolithic or in-process bridge cases.
- Align gateway runtime and rollout tests with the updated microservice bridge semantics.

## Acceptance

- A gateway with configured upstream query endpoints reports `domain_bridge_enabled=true`.
- Gateway rollout/runtime summaries can reach `runtime-wired` when all other route signals are present.
- Targeted gateway tests pass.
