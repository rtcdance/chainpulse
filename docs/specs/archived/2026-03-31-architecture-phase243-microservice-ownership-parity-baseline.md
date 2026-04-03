# Phase 243 - Microservice Ownership Parity Baseline

## Status
Status: Approved

## Why
- Phase 241 added the first explicit ownership parity marker in `api-service`.
- Phase 242 extended the same pattern into `api-gateway`.
- The repository still needed one more step:
  a shared baseline that treats those markers as one parity boundary instead of
  two isolated service-local choices.

## Scope
- Keep the rollout contract unchanged.
- Add a shared validator for the route-oriented microservice ownership parity
  marker boundary.
- Use that validator from both `api-service` and `api-gateway` focused tests.

## Implementation
- Add `ValidateMicroserviceOwnershipParityMarker(...)` to shared rollout parity
  helpers.
- Keep the validator intentionally narrow:
  - advisory reason includes `ownership_parity_hint`
  - work item review fields include `ownership_runtime_parity`
  - work item/advisory text explicitly mentions ownership-runtime parity with
    monolith
- Update `api-service` and `api-gateway` tests to reuse the shared validator.
- Refresh the coverage summary so the new ownership parity baseline is visible
  in architecture documentation.

## Validation
- Run `pkg/plugins/api` tests.
- Run focused `api-service` rollout tests.
- Run focused `api-gateway` rollout tests.

## Exit Criteria
- `api-service` and `api-gateway` now share one explicit ownership parity
  baseline enforced by a shared validator.
- The repository documents that this is a baseline marker layer, not a claim
  that ownership-runtime parity has already been achieved.
