# Phase 192 - Event Processor Rollout Posture Hints

## Status
Status: Approved

## Why
- Phase 191 gave `event-processor` a real rollout producer, but its
  runtime-derived reasons still required operators to parse raw enabled/missing
  dependency lists.
- The next low-risk improvement is to add a compact posture hint so the report
  communicates the intended operational stance more quickly.

## Scope
- Keep the rollout report contract unchanged.
- Add a compact rollout posture hint for runtime-derived `event-processor`
  rollout reasons.

## Implementation
- Add a small helper that maps runtime-derived `event-processor` rollout states
  into posture hints.
- Append the posture hint to the existing advisory reason for runtime-derived
  reports.
- Leave rollout fields, enums, and body structure unchanged.

## Validation
- Add focused helper coverage for posture hint mapping.
- Update producer coverage for partially wired and runtime-wired states.
- Update focused handler-level `/health/rollout` coverage.
- Run focused Go tests and `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Runtime-derived `event-processor` rollout reasons include a compact posture
  hint.
- External rollout payload shape and status enums remain unchanged.
