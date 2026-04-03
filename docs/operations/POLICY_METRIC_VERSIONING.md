# Policy Metric Versioning and Deprecation

**Status**: Active | **Last Updated**: 2026-03-30

## Goal

Enable zero-downtime dashboard/alert migration when policy metric schema evolves.

## Schema Modes

Configured by env:
- `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE=v1|dual_write|v2`
- `CHAINPULSE_POLICY_V1_DEPRECATION_DATE=YYYY-MM-DD` (optional CI cutoff)
- `CHAINPULSE_POLICY_V1_DEPRECATION_WARN_DAYS=<int>` (optional warning window, default `14`)

Default:
- `v1`

## Metric Names by Schema

1. `v1` (legacy)
- `core_config_overrides_applied_total`
- `core_config_overrides_policy_evaluation_total`

2. `v2` (new)
- `chainpulse_policy_overrides_applied_total`
- `chainpulse_policy_overrides_evaluation_total`

3. `dual_write` (migration window)
- emit both `v1` and `v2`

## Tag Compatibility

All emitted policy metrics include:
- `metric_schema_version`
- `metric_schema_deprecated`

Guidance:
- During `dual_write`, build dashboards against `v2`, keep `v1` as fallback.
- Use `metric_schema_deprecated=true` to find remaining legacy dependency.

## Recommended Migration Workflow

1. Prepare
- Deploy with `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE=dual_write`.
- Ensure both v1/v2 metrics are visible.

2. Migrate dashboards/alerts
- Switch queries from v1 names to v2 names.
- Verify alert parity for at least 24h.

3. Cutover
- Set `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE=v2`.
- Keep rollback plan to `dual_write`.

4. Cleanup
- Remove legacy v1 queries after one full release cycle.

## Rollback

If v2 pipeline regression occurs:
- set `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE=dual_write` (preferred)
- or temporary fallback `v1`

## Guardrail

CI enforces contract tests via:
- `scripts/check-policy-metric-contract.sh`
- `scripts/check-migration-manifest.sh`

When `CHAINPULSE_POLICY_V1_DEPRECATION_DATE` is set:
- before cutoff: non-`v2` modes produce warning near deadline
- on/after cutoff: non-`v2` mode fails CI

Migration ownership/deadline tracking is managed in:
- `docs/operations/MIGRATION_MANIFEST.csv`
