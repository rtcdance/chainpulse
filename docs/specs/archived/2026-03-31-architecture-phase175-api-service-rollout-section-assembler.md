# Phase 175 - API Service Rollout Section Assembler

## Status
Status: Approved

## Why
- Phase 174 gave `api-service` a sectioned rollout producer shape, but the
  producer still assembled those sections inline.
- Monolith already uses a clearer `section builder -> assembler -> apply`
  structure, and `api-service` should converge on that model before deeper
  cross-mode reuse work.

## Scope
- Keep `/health/rollout` values unchanged.
- Add a dedicated assembler layer for skeleton and runtime-derived api-service
  rollout sections.

## Implementation
- Add:
  - `buildAPIServiceSkeletonSections()`
  - `buildAPIServiceRuntimeDerivedSections(...)`
- Keep `applyAPIServiceRolloutReportSections(...)` as the final applier.
- Retain existing section builders for `surface`, `approval`, and `guarded`.

## Validation
- Add focused assembler coverage for both skeleton and runtime-derived paths.
- Run api-service tests, shared tests, and the fast micro-loop gate.

## Exit Criteria
- `api-service` rollout production follows:
  - section builder
  - section assembler
  - apply
- External rollout payload values remain unchanged.
