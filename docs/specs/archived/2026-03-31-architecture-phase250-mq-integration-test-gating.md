# Phase 250 - MQ Integration Test Gating

## Status
Status: Approved

## Why
- After separating PostgreSQL integration coverage from normal database package
  validation, the next repository-health blocker was `pkg/plugins/mq`.
- Normal `go test ./pkg/plugins/mq/...` still mixed in explicit Kafka and
  ZeroMQ integration slices.
- In practice, the ZeroMQ publish path could hang without an external peer,
  which made the package a poor default validation target.

## Scope
- Add an explicit opt-in gate for MQ integration tests.
- Keep ordinary `pkg/plugins/mq` validation focused on default-safe tests while
  preserving external MQ coverage behind an environment switch.

## Implementation
- Add a shared MQ integration-test helper in:
  - `pkg/plugins/mq/mq_test_helpers_test.go`
- Require `CHAINPULSE_RUN_MQ_INTEGRATION=1` for:
  - `pkg/plugins/mq/kafka_mq_integration_test.go`
  - the external-peer ZeroMQ integration slice in
    `pkg/plugins/mq/zeromq_mq_test.go`
- Preserve `testing.Short()` skipping as an additional guard.

## Validation
- Run `go test ./pkg/plugins/mq/...`

## Exit Criteria
- Normal `go test ./pkg/plugins/mq/...` no longer hangs on external MQ
  integration behavior.
- Kafka and ZeroMQ integration coverage remains available behind an explicit
  environment gate.
