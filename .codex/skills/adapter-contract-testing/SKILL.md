---
name: "adapter-contract-testing"
description: "Add or update adapter contract tests for DB/MQ/Cache/RPC/API implementations. Ensure behavior parity across in-memory and production adapters. Invoke when adding/changing any adapter implementation (DB/MQ/Cache/RPC/API)."
---

# Skill: adapter-contract-testing

## Trigger

Use this skill when adding/changing any adapter implementation (DB/MQ/Cache/RPC/API).

## Must Do

1. Keep a stable interface contract at domain/application boundaries.
2. Add or update contract tests for each adapter behavior.
3. Validate parity between in-memory/mock and production adapters.
4. Cover success, timeout, retryable failure, and permanent failure cases.
5. Keep error semantics consistent across adapter implementations.

## ChainPulse Pointers

- Adapter-like modules:
  - `pkg/plugins/database/*`
  - `pkg/plugins/mq/*`
  - `pkg/plugins/cache/*`
  - `pkg/plugins/pullers/*`
  - `pkg/plugins/api/*`
- Test suites:
  - `pkg/**/*_test.go`
  - `pkg/**/*_property_test.go`
  - `test/integration/*`

## Must Not

- No adapter-specific behavior leaks into business use cases.
- No incompatible method semantics between different adapter implementations.

## Exit Criteria

- Contract tests pass for changed adapters.
- Compatibility behavior is verified and documented in PR notes.
