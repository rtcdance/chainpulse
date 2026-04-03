# Migration Manifest Governance

**Status**: Active | **Last Updated**: 2026-03-30

## Purpose

Track cross-cutting observability migration items with owner/deadline/status and enforce overdue checks in CI.

## Source of Truth

- Manifest file: `docs/operations/MIGRATION_MANIFEST.csv`
- Validation script: `scripts/check-migration-manifest.sh`

## CSV Schema

Header:
```text
id,domain,owner,status,deadline,severity,spec_ref,notes
```

Field rules:
1. `id`: unique migration identifier (`kebab-case` recommended)
2. `domain`: migration area (for example `observability`)
3. `owner`: accountable team or role
4. `status`: one of:
   - `planned`
   - `in_progress`
   - `blocked`
   - `completed`
   - `waived`
5. `deadline`: `YYYY-MM-DD` (UTC-based check)
6. `severity`: `low|medium|high|critical` (free-form but required)
7. `spec_ref`: approved implementation/planning spec path under `docs/specs/`
8. `notes`: short operational context

## CI Behavior

`scripts/check-migration-manifest.sh`:
- fails on malformed rows
- fails on invalid status/date
- fails when active item has missing or non-approved `spec_ref`
- fails when active item metadata does not match spec (`owner`/`delivery status`/`deadline`)
- fails when non-completed items are overdue
- warns when a deadline is within warning window

Env control:
- `CHAINPULSE_MIGRATION_WARN_DAYS` (default `14`)
- `CHAINPULSE_MIGRATION_REQUIRE_SPEC_SYNC` (default `true`)
- `CHAINPULSE_MIGRATION_REQUIRE_SPEC_METADATA_SYNC` (default `true`)
- `CHAINPULSE_MIGRATION_OWNER_ALLOWLIST` (default `platform-team,sre-team,indexer-team`)
- `CHAINPULSE_FAIL_ON_UNKNOWN_MIGRATION_OWNER` (default `false`; CI can set `true`)

## Operational Workflow

1. Add migration item when introducing new schema/versioned contract.
2. Assign explicit owner and realistic deadline.
3. Update status continuously in PRs.
4. Mark `completed` immediately after rollout.
5. Use `waived` only with explicit architecture review decision.
