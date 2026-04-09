## Status
Status: Approved

## Summary

The new one-click Docker acceptance entrypoint and the existing compose
readiness smoke currently duplicate the same Docker runtime, credential helper,
image pull, and HTTP wait logic. That duplication increases maintenance cost and
creates drift risk.

## Decision

- Add a shared shell helper under `scripts/lib/docker_acceptance.sh`.
- Move the common Docker runtime and compose-acceptance helpers into that shared
  library.
- Update both `scripts/run-docker-acceptance.sh` and
  `scripts/verify-docker-compose-microservices-readiness.sh` to source the
  library instead of carrying their own copies.

## Acceptance

- The shared helper file exists.
- Both scripts keep their current user-facing behavior and `--help` output.
- Shell syntax checks pass for the helper and both callers.
- The one-click Docker acceptance flow still passes after the refactor.
