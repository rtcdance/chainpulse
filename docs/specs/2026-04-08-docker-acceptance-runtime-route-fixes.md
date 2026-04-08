Title: Docker Acceptance Runtime Route Fixes
Type: bugfix
Status: Implemented
Owner: Codex
Reviewers: ChainPulse Team
Related Modules: cmd/microservices/api-service, cmd/microservices/puller, pkg/plugins/api, pkg/services/query

## Context
Running ChainPulse with `docker/docker-compose.microservices.acceptance.local.yml` and full acceptance tests reveals runtime blockers:
- api-service listens on 8080 due plugin hardcode while compose expects 8081.
- query runtime reports degraded because cache service is initialized but never started.
- puller exits because it passes empty MongoDB URI to database manager.
- gateway does not normalize `/api/v1/*` prefixed paths used by acceptance tests.

## Scope
- Fix API plugin port binding to use core config port.
- Start/stop cache lifecycle with query service start/stop.
- Use loaded MongoDB URI when creating puller database manager.
- Normalize `/api/v1` request prefix in gateway router integration before route match/forward.
- Add or update focused tests for changed behavior.

## Non-Goals
- No architectural redesign of routing.
- No large compose refactor.
- No unrelated test-suite rewrites.

## Options Considered
1. Patch acceptance tests to match current behavior.
2. Patch runtime behavior to satisfy declared microservice contract and current tests.

Selected: Option 2, because runtime contract and compose should be authoritative for acceptance.

## Risks
- Route normalization could affect existing non-prefixed paths.
- Cache lifecycle changes could impact query service tests.

## Mitigations
- Keep normalization additive: only strip exact `/api/v1` prefix; existing paths unchanged.
- Add targeted tests around new behaviors.

## Rollback
- Revert the small set of touched files if regressions appear.

## Verification Plan
- Unit tests for modified packages.
- Rebuild/restart acceptance compose stack.
- Re-run `npm test` acceptance suite.

## Quality Gate Plan
- Fast: targeted go tests for touched packages.
- Full: `npm test` with running docker acceptance stack.

## Verification Summary
- Targeted gate: `go test ./pkg/services/query -run 'TestQueryServiceRuntimeSummaryReady|TestQueryServiceRuntimeSummaryHealthyCacheAfterStart|TestQueryServiceStartStopManagesCacheLifecycle'` passed.
- Full acceptance gate: `npm test` passed with `24 passed, 1 skipped, 0 failed`.
