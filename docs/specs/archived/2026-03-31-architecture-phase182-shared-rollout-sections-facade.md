# Phase 182 - Shared Rollout Sections Facade

## Status
Status: Approved

## Why
- By Phase 181, monolith and `api-service` already shared typed builders for
  `surface`, `approval`, and `guarded-cutover`, but each deployment mode still
  had to manually stitch those three builders together.
- The next safe cleanup step is a shared facade that assembles all three
  sections in one place without changing report values.

## Scope
- Keep rollout report values unchanged.
- Add a shared `sections input -> sections output` facade.

## Implementation
- Add:
  - `RolloutReportSections`
  - `RolloutReportSectionsInput`
  - `BuildRolloutReportSections(...)`
- Update monolith and `api-service` section assemblers to feed the shared
  facade instead of directly calling the three section builders inline.

## Validation
- Add contract-level coverage for the shared sections facade.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both assemble rollout report sections through the
  same shared facade.
- External rollout payload values remain unchanged.
