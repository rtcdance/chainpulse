## Status
Status: Approved

## Summary

The production checklist still references several script entrypoints that do not
exist in the repository: `deploy-staging.sh`, `smoke-tests.sh`,
`verify-staging.sh`, and `verify-production.sh`.

## Decision

- Add repo-root implementations for the missing production-checklist scripts.
- Keep the scripts compositional and explicit about scope:
  - `smoke-tests.sh` reuses deployment smoke.
  - `verify-staging.sh` runs repository-local readiness and verification checks.
  - `verify-production.sh` performs live HTTP contract checks against configured service URLs.
  - `deploy-staging.sh` fails fast with a clear message until environment-specific deployment automation exists.

## Acceptance

- The previously missing scripts exist and support `--help`.
- The scripts provide truthful scope boundaries rather than pretending to deploy or certify more than they actually can.
- Static shell syntax checks pass.
