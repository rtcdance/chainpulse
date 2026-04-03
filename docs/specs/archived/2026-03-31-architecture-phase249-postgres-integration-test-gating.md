# Phase 249 - Postgres Integration Test Gating

## Status
Status: Approved

## Why
- After clearing the import and adapter compile debt, the next recurring
  repository-health blocker was environment-gated database testing.
- `pkg/plugins/database` still assumed a local PostgreSQL instance for several
  integration and performance tests during a normal `go test` run.
- That made the package noisy in ordinary local validation and hid the real
  distinction between unit coverage and explicit PostgreSQL integration
  coverage.

## Scope
- Add an explicit opt-in gate for PostgreSQL integration and benchmark tests in
  `pkg/plugins/database`.
- Make normal `go test ./pkg/plugins/database/...` skip PostgreSQL-dependent
  cases unless the environment explicitly enables them.

## Implementation
- Add a shared PostgreSQL integration-test helper in:
  - `pkg/plugins/database/postgres_test_helpers_test.go`
- Require `CHAINPULSE_RUN_POSTGRES_INTEGRATION=1` for PostgreSQL-dependent
  tests, while continuing to skip in `testing.Short()` mode.
- Replace direct `testing.Short()`-only guards in:
  - `pkg/plugins/database/postgres_database_integration_test.go`
  - `pkg/plugins/database/postgres_database_integration_suite_test.go`
  - `pkg/plugins/database/postgres_database_benchmark_test.go`
  - `pkg/plugins/database/postgres_database_transaction_test.go`

## Validation
- Run `go test ./pkg/plugins/database/...`

## Exit Criteria
- Normal `go test ./pkg/plugins/database/...` no longer assumes a local
  PostgreSQL service.
- PostgreSQL integration and benchmark coverage remains available behind an
  explicit environment gate.
