# Phase 189 - API Service Query Health Reason Hints

## Status
Status: Approved

## Why
- Phase 188 made fully wired healthy versus degraded query-runtime states
  explicit in `advisory.status`.
- The next low-risk improvement is to make the rollout reason itself more
  actionable so operators can see not just the status but the immediate intent
  behind it.

## Scope
- Keep the rollout report contract unchanged.
- Add query-health-specific reason hints for `api-service` rollout reports.

## Implementation
- Add a small helper that maps query runtime health to a human-readable hint.
- Append the hint to the existing advisory reason text for runtime-derived
  `api-service` rollout reports.
- Leave rollout fields and enum values unchanged.

## Validation
- Add producer coverage for healthy and degraded query health hints.
- Add focused helper coverage for hint mapping.
- Update route integration tests.
- Run Go tests and the fast micro-loop gate.

## Exit Criteria
- `api-service` rollout reasons contain query-health-specific operator hints.
- External rollout payload shape and status enums remain unchanged.
