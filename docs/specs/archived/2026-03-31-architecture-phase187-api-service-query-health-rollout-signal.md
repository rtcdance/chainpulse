# Phase 187 - API Service Query Health Rollout Signal

## Status
Status: Approved

## Why
- Phase 186 expanded `api-service` rollout completeness using additional real
  runtime route signals.
- The next higher-value step is to include a true runtime health signal so the
  rollout report can say not only whether the local route graph is wired, but
  whether the query runtime is actually healthy enough to support it.

## Scope
- Keep the rollout report contract unchanged.
- Add query runtime health signal consumption to the `api-service` rollout
  producer.

## Implementation
- Feed `QueryService.Health(ctx)` into the `api-service` rollout state provider.
- Reflect query health in:
  - `advisory.ready`
  - rollout explanatory reason text
- Keep progression/cutover semantics unchanged for now.

## Validation
- Update producer tests for healthy and degraded query-runtime cases.
- Update route integration tests.
- Run Go tests and the fast micro-loop gate.

## Exit Criteria
- `api-service` rollout report includes query runtime health context.
- `advisory.ready` becomes meaningful for the fully wired + healthy case.
- External rollout payload shape remains unchanged.
