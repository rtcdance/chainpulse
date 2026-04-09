# Security Rollout

The current four-service runnable baseline exposes optional, default-off
security surfaces across:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

This guide explains the safest way to enable them incrementally without
changing the default runnable path.

## Rollout order

Enable the services in this order:

1. `api-gateway`
2. `api-service`
3. `puller`
4. `event-processor`

This order keeps the external entrypoint and query bridge under observation
first, then hardens the query service, and only then narrows the execution
control surfaces.

## Suggested verification after each step

After enabling each service, verify:

- `/runtime/summary` reflects the expected `security_posture`
- the service still responds on `/health`
- the current runnable slice still passes its focused verification profile

For the minimal runnable app, keep the gateway query smoke path passing.

## Rollback order

If any step causes unexpected friction, roll back in reverse order:

1. `event-processor`
2. `puller`
3. `api-service`
4. `api-gateway`

## Defaults

The default runnable baseline remains open and does not require these controls
to be enabled.

## References

- [`SECURITY_BASELINE.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/project/SECURITY_BASELINE.md)
- [`RUNNABLE_APP.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/project/RUNNABLE_APP.md)
- [`docs/ARCHITECTURE.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/ARCHITECTURE.md)
- [`docs/INDEX.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/INDEX.md)
