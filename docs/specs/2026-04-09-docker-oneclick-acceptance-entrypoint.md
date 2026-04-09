## Status
Status: Approved

## Summary

The repository already has a compose-based readiness smoke, but operators still
need to remember multiple commands for startup, acceptance, status inspection,
and teardown. There is no single user-facing entrypoint that turns the Docker
microservice stack into a repeatable one-click capability.

## Decision

- Add `scripts/run-docker-acceptance.sh` as the user-facing Docker acceptance
  entrypoint.
- Support explicit subcommands:
  - `up` to start and wait for the compose microservice stack
  - `accept` to run acceptance against an already running stack
  - `all` to perform `up` and then `accept`
  - `ps` to inspect service status
  - `down` to tear the stack down
- Reuse the existing acceptance scripts for runnable verification and
  Prometheus live smoke instead of introducing a second acceptance flow.

## Acceptance

- `scripts/run-docker-acceptance.sh` exists and supports `--help`.
- `all` performs startup plus the existing acceptance checks.
- `accept` does not force a stack restart when the services are already up.
- `down` removes the compose stack and volumes.
- README references the new one-click Docker acceptance entrypoint.
