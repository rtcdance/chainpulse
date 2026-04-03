# Phase 219 - Execution Progress Parity Helper

## Status
Status: Approved

## Why
- Phase 218 aligned `puller` and `event-processor` on one shared
  execution-progress input+builder path.
- The remaining duplication lived in tests, where each service still asserted
  its execution-progress reason details separately.
- The next useful slice is to add one shared parity/helper layer for checking
  that rollout reasons actually cover the execution-progress details implied by
  the shared facade.

## Scope
- Keep rollout semantics unchanged.
- Add one shared validator for execution-progress reason coverage.
- Repoint `puller` and `event-processor` tests to the shared validator for
  stable execution-progress reason assertions.

## Implementation
- Add `ValidateRolloutExecutionProgressReasonCoverage(...)`.
- Build expected progress parts through the shared execution-progress facade and
  shared reason appenders, then validate those parts against the rollout
  reason text.
- Add contract-level tests for:
  - successful reason coverage
  - missing execution-progress detail failure
- Update:
  - `puller` runtime-support rollout test
  - `event-processor` producer/runtime-support rollout tests
  to consume the shared validator instead of duplicating progress-specific
  string coverage checks.

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Execution-progress reason coverage has one shared validator.
- `puller` and `event-processor` tests use the shared validator for stable
  progress detail assertions without changing rollout payload semantics.
