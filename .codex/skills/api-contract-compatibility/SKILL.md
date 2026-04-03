---
name: "api-contract-compatibility"
description: "Preserve API backward compatibility by default. Require versioning/migration plan for breaking changes. Invoke when changing REST/gRPC/WebSocket request/response fields, status codes, or semantics."
---

# Skill: api-contract-compatibility

## Trigger

Use this skill when changing REST/gRPC/WebSocket request/response fields, status codes, or semantics.

## Must Do

1. Keep backward-compatible API evolution by default.
2. If breaking change is required, define versioning and migration path.
3. Update API docs and examples.
4. Add compatibility tests for changed contracts.
5. Define deprecation timeline and consumer impact.

## Must Not

- No silent breaking API changes.
- No undocumented payload/semantic drift.

## Exit Criteria

- Contract updates are tested and documented.
- Consumer migration path is explicit.
