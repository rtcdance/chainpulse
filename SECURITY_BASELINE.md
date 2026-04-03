# Security Baseline

ChainPulse now exposes an optional, default-off security surface across the
current four-service runnable baseline:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

## Current posture

The security controls are intentionally opt-in and do not change the default
runnable path.

### Common shape

Each service exposes:

- authentication middleware
- rate limiting middleware
- runtime summary posture fields
- a clear "disabled by default" hint

### Service-specific notes

- `api-gateway`: protects the external query entrypoint
- `api-service`: protects the query service entrypoint
- `puller`: protects the runtime/control entrypoint
- `event-processor`: protects the runtime/control entrypoint

## Operational guidance

1. Keep the security surfaces off for the minimum runnable baseline.
2. Enable them explicitly when you need to harden a specific entrypoint.
3. Treat the current state as an opt-in posture baseline, not a default-hard
   deployment policy.

## Relevant docs

- [`RUNNABLE_APP.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/RUNNABLE_APP.md)
- [`docs/ARCHITECTURE.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/ARCHITECTURE.md)
- [`docs/INDEX.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/INDEX.md)
- [`docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md)
