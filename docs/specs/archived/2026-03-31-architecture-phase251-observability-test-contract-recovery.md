# Phase 251 - Observability Test Contract Recovery

## Status
Status: Approved

## Why
- After cleaning the database and MQ environment-gated test debt, the next
  broader validation blocker surfaced in `pkg/observability`.
- `go test ./pkg/...` still stopped on a compile-time drift between the
  distributed tracing tests and the current tracer contract.
- This was a small issue, but it blocked the larger package graph again, so it
  was the highest-value next cleanup step.

## Scope
- Recover the tracing test in `pkg/observability` so it matches the current
  `InjectContext` contract.
- Re-validate the observability package and then re-run the broader `./pkg/...`
  graph.

## Implementation
- Update `pkg/observability/distributed_tracing_test.go` so the inject-context
  test passes a `*TraceContext`, matching the current tracer interface and
  implementation.

## Validation
- Run `go test ./pkg/observability/...`
- Run `go test ./pkg/...`

## Exit Criteria
- `pkg/observability` no longer fails at compile time because of tracing test
  drift.
- `go test ./pkg/...` completes successfully after the fix.
