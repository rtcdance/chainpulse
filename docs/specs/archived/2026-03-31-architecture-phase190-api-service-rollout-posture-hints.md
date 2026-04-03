# Phase 190 - API Service Rollout Posture Hints

## Status
Status: Approved

## Why
- Phase 189 added query-health-specific hints to `api-service` rollout reasons.
- The next low-risk improvement is to compress runtime wiring plus query health
  into a more direct rollout posture hint so operators can scan the report
  faster without changing the report contract.

## Scope
- Keep the rollout report contract unchanged.
- Add a compact rollout posture hint for runtime-derived `api-service` rollout
  reasons.

## Implementation
- Add a small helper that maps runtime-derived rollout posture into a concise
  operator hint.
- Append the posture hint to the existing advisory reason for runtime-derived
  `api-service` rollout reports.
- Keep rollout fields, enums, and body structure unchanged.

## Validation
- Add focused helper coverage for posture hint mapping.
- Update producer coverage for healthy, degraded, and partial wiring cases.
- Update `/health/rollout` route integration coverage.
- Run Go tests and the fast micro-loop gate.

## Exit Criteria
- Runtime-derived `api-service` rollout reasons include a compact rollout
  posture hint.
- External rollout payload shape and status enums remain unchanged.
