## Status
Status: Approved

## Summary

The initial `scripts/verify-production.sh` only checks basic health and a small
subset of runtime fields. For a production gate, it should verify the stronger
contracts that reflect the recent production-hardening work.

## Decision

- Strengthen `scripts/verify-production.sh` to require:
  - microservice deployment mode across the foreground services
  - gateway `runtime_mode=runtime-wired`
  - gateway rollout `ready=true`
  - gateway `query_bridge_posture=query-bridge-ready`
  - gateway and api-service `security_posture=...-security-ready`
- Keep the checks string-based so the script remains lightweight and dependency-free.

## Acceptance

- `verify-production.sh` enforces stronger production runtime and security contracts.
- The script remains self-documented via `--help`.
- Shell syntax checks pass.
