## Status
Status: Approved

## Summary

The API gateway currently starts in production profile even when authentication and rate limiting are both disabled. This leaves the external entrypoint open by default and blocks any credible production-readiness claim.

## Decision

- Add a production-profile startup gate for `cmd/microservices/api-gateway`.
- In `production` profile, require:
  - `GATEWAY_AUTH_ENABLED=true`
  - non-empty `GATEWAY_AUTH_JWT_SECRET`
  - `GATEWAY_RATE_LIMIT_ENABLED=true`
  - `GATEWAY_RATE_LIMIT > 0`
- Keep non-production profiles unchanged so local runnable and acceptance flows continue to work.

## Acceptance

- The gateway refuses to start in production profile when required security controls are disabled or incomplete.
- Non-production profiles keep the current runnable baseline behavior.
- Targeted gateway tests pass.
