# Migration Governance Dashboard Queries

**Status**: Active | **Last Updated**: 2026-03-30

## Source Metrics

Exported by:
- `scripts/export-migration-governance-kpi.sh`
- `scripts/compare-migration-governance-kpi.sh` (delta summary)

Baseline file:
- `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
- `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom`

Primary metric names:
- `chainpulse_migration_items_total`
- `chainpulse_migration_items_active_total`
- `chainpulse_migration_items_by_status{status=...}`
- `chainpulse_migration_items_by_severity{severity=...}`
- `chainpulse_migration_items_by_domain{domain=...}`
- `chainpulse_migration_ticket_registry_checks_total{mode,source,status,failure_mode}`
- `chainpulse_migration_ticket_registry_fallback_events_total{reason,failure_mode}`
- `chainpulse_migration_ticket_registry_http_latency_ms{source,status,slo_mode}`
- `chainpulse_migration_ticket_registry_http_slo_violations_total{source,slo_mode}`

## PromQL Templates

1. Total migration items
```promql
max(chainpulse_migration_items_total)
```

2. Active migration items
```promql
max(chainpulse_migration_items_active_total)
```

3. Status distribution
```promql
max by (status) (chainpulse_migration_items_by_status)
```

4. Severity distribution
```promql
max by (severity) (chainpulse_migration_items_by_severity)
```

5. Domain distribution
```promql
max by (domain) (chainpulse_migration_items_by_domain)
```

6. Ticket registry check status split
```promql
sum by (status, source) (chainpulse_migration_ticket_registry_checks_total)
```

7. Ticket registry fallback events
```promql
sum by (reason, failure_mode) (chainpulse_migration_ticket_registry_fallback_events_total)
```

8. Ticket registry HTTP latency (latest)
```promql
max by (source, status, slo_mode) (chainpulse_migration_ticket_registry_http_latency_ms)
```

9. Ticket registry HTTP SLO violations
```promql
sum by (source, slo_mode) (chainpulse_migration_ticket_registry_http_slo_violations_total)
```

## Suggested Panels

1. `Active Migration Items` (single stat)
2. `Migration Status Breakdown` (bar/pie)
3. `Migration Severity Breakdown` (bar/pie)
4. `Migration Domain Breakdown` (bar)

## Alert Seeds

1. Active migrations too high
```promql
max(chainpulse_migration_items_active_total) > 10
```

2. High-severity migration backlog
```promql
max(chainpulse_migration_items_by_severity{severity="high"}) > 3
```

## PR Delta Workflow

1. Export current snapshot:
- `./scripts/export-migration-governance-kpi.sh`

2. Compare with baseline:
- `./scripts/compare-migration-governance-kpi.sh`

3. Reuse generated draft summary:
- `build/migration-governance/migration-governance-delta.md`

4. Refresh baseline with explicit guard (when approved):
- `CHAINPULSE_ALLOW_BASELINE_UPDATE=true ./scripts/update-migration-governance-baseline.sh`
  - default behavior refreshes:
    - `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
    - `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`

Preflight before refresh (no mutations):
- `./scripts/preflight-migration-baseline-update.sh`
- output:
  - `build/migration-governance/baseline-update-preflight.md`

Resolver function tests:
- `./scripts/test-baseline-update-resolver.sh`
- output:
  - `build/migration-governance/baseline-resolver-test.json`
  - `build/migration-governance/baseline-resolver-test.prom`
  - `build/migration-governance/baseline-resolver-test.md`
  - markdown contains conditional `Failure Summary` section when failed resolver cases exist
  - CI job summary prepends `Baseline Resolver Test Highlights`

5. Validate baseline governance:
- `./scripts/check-migration-baseline-governance.sh`

6. Run baseline governance scope smoke tests:
- `./scripts/smoke-baseline-governance-scope.sh`
  - outputs:
    - `build/migration-governance/baseline-scope-smoke.json`
    - `build/migration-governance/baseline-scope-smoke.prom`
    - `build/migration-governance/baseline-scope-smoke.md`
  - report enhancement:
    - markdown contains conditional `Failure Summary` section when failed cases exist
    - markdown contains `Family Summary` section (`scope|preflight|update|custom-path|template`)
    - CI job summary prepends `Baseline Scope Smoke Highlights` before appending the full smoke markdown artifact
    - CI highlights also include smoke delta regression status and regression-signal preview when `baseline-scope-smoke-delta.md` exists
  - scenario coverage:
    - `dual_scope_alignment`
    - `kpi_scope_alignment`
    - `health_scope_alignment`
    - `resolver_changed_baselines_alignment`
    - `scope_mismatch_should_fail`
    - `changed_baselines_mismatch_should_fail`
    - `resolver_changed_baselines_mismatch_should_fail`
    - `preflight_without_resolver_refresh`
    - `preflight_with_resolver_refresh`
    - `custom_resolver_path_preflight_no_refresh_should_not_show_target`
    - `custom_resolver_path_preflight_manual_changed_baselines_override`
    - `custom_resolver_path_preflight_invalid_changed_baselines_should_fail`
    - `guarded_update_with_resolver_refresh`
    - `guarded_update_blocked_without_allow_flag`
    - `guarded_update_invalid_changed_baselines_should_fail`
    - `guarded_update_blocked_custom_template_should_not_be_created`
    - `guarded_update_invalid_changed_baselines_custom_template_should_not_be_created`
    - `custom_resolver_baseline_path_parity`
    - `custom_resolver_baseline_path_missing_should_fail`
    - `custom_resolver_baseline_path_blocked_update_should_not_create_file`
    - `custom_resolver_path_blocked_update_custom_template_should_not_be_created`
    - `custom_resolver_path_invalid_changed_baselines_custom_template_should_not_be_created`

7. Compare baseline scope smoke against baseline:
- `./scripts/compare-baseline-scope-smoke.sh`
- output:
  - `build/migration-governance/baseline-scope-smoke-delta.tsv`
  - `build/migration-governance/baseline-scope-smoke-delta.md`

8. Compare baseline resolver test against baseline:
- `./scripts/compare-baseline-resolver-test.sh`
- output:
  - `build/migration-governance/baseline-resolver-test-delta.tsv`
  - `build/migration-governance/baseline-resolver-test-delta.md`
  - CI resolver highlights include delta regression status and regression-signal preview when resolver delta markdown exists
  - smoke/resolver summary helpers reuse a shared governance summary shell library to keep rendering behavior aligned

CI overview:
- `./scripts/append-governance-overview-summary.sh`
- output:
  - appends `Governance Overview` section to CI job summary
  - summarizes smoke/resolver status, failed count, total count, and regression status in one table
  - also includes ticket-registry health and owner-drift rows for broader governance snapshot
  - overview rows include normalized `Level` values: `ok|warn|fail|info`
  - includes an inline legend for interpreting overview `Level` values
  - includes a `Source` column so each overview row points to its backing artifact file
  - includes a `Details` column for delta or supplementary artifact hints
  - includes a `Route` column to indicate the most likely report domain or operator group to engage first
  - includes a `Playbook` column to indicate the first local report or document to open for that surface
  - includes an `Action` column with short next-read guidance based on row severity
  - includes top-level `Overall Health` aggregate above the table
  - includes top-level `Overall Hint` aggregate interpretation alongside `Overall Health`
  - includes top-level `Overall Focus` to identify the first degraded surface to inspect
  - includes top-level `Overall Route` to identify the likely first escalation target for the degraded surface
  - includes top-level `Overall Playbook` to identify the first local document to open for the degraded surface
  - includes top-level `Overall Next Step` to combine degraded focus, route, and playbook into one severity-aware action hint
  - overview regression coverage now explicitly validates aggregate `fail`, `warn`, `info`, and `ok` states
  - overview regression tests now use shared fixture and assertion helpers to keep new scenarios easier to add
  - primary overview row checks are now grouped through a compact row-scenario loop to reduce test noise
  - aggregate overview checks are now grouped through compact scenario helpers to keep the state matrix easier to scan
  - overview regression script is now organized around explicit setup/run/assert scenario stages
  - non-primary overview states now also have descriptor-based row assertions for lightweight row-level expansion
  - overview setup helpers now use compact data blocks to reduce repeated fixture writes
  - setup descriptors now fail fast on unknown kinds or missing fields to prevent silent fixture drift
  - aggregate and row descriptors now also fail fast on malformed field counts
  - malformed setup, aggregate, and row descriptors now have explicit negative-path regression checks
  - descriptor parsing and validation now flow through narrower parser helpers to keep the script easier to scan
  - parser-layer aggregate and row validation also have direct micro negative checks
  - owner-drift now renders as informational when there is no owner dataset (`Distinct Owners = 0`)

9. Validate changelog entry quality:
- `./scripts/check-migration-changelog-quality.sh`
  - registry health outputs:
    - `build/migration-governance/ticket-registry-health.prom`
    - `build/migration-governance/ticket-registry-health.md`

10. Compare ticket registry health against baseline:
- `./scripts/compare-ticket-registry-health.sh`
- output:
  - `build/migration-governance/ticket-registry-health-delta.tsv`
  - `build/migration-governance/ticket-registry-health-delta.md`

Required changelog policy controls:
- `CHAINPULSE_MIGRATION_TICKET_PATTERN` (default `^[A-Z0-9]+-[A-Z0-9]+$`)
- `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE` (default `pattern`; options: `pattern|registry|both`)
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE` (default `file`; options: `file|http`)
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_FILE` (default `docs/operations/MIGRATION_TICKET_REGISTRY.txt`)
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_URL` (required when source is `http`)
- `CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE` (default `enforce`; options: `enforce|warn`)
- `CHAINPULSE_MIGRATION_OWNER_ALLOWLIST` (default `platform-team,sre-team,indexer-team`)
- `CHAINPULSE_MIGRATION_CHANGELOG_SCOPE_ALLOWLIST` (default `kpi-only,health-only,dual`)
- `CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE` (default `true`; options: `true|false`)
- `CHAINPULSE_MIGRATION_CHANGELOG_CHANGED_BASELINES_ALLOWLIST` (default `kpi,health,smoke,resolver`)
- `CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_CHANGED_BASELINES` (default `true`; options: `true|false`)
- `CHAINPULSE_FAIL_ON_UNKNOWN_MIGRATION_OWNER` (default `false`)

Changelog entry format:
- `- <UTC-ISO8601> | ticket=<ID> | owner=<team-or-user> | scope=<kpi-only|health-only|dual> | changed_baselines=<kpi[,health][,smoke][,resolver]> | rationale=<text>`
- Note: `resolver` tag is scope-orthogonal and does not change scope semantics.

Owner drift report export:
- `./scripts/export-migration-owner-drift-report.sh`
- output:
  - `build/migration-governance/migration-owner-drift-report.tsv`
  - `build/migration-governance/migration-owner-drift-report.md`

Ticket registry health output controls:
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_OUTPUT` (default `build/migration-governance/ticket-registry-health.prom`)
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_MD_OUTPUT` (default `build/migration-governance/ticket-registry-health.md`)

HTTP registry latency SLO controls:
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MS` (default `2000`)
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE` (default `warn`; options: `off|warn|enforce`)

Ticket registry delta regression policy:
- `CHAINPULSE_MIGRATION_REGISTRY_HEALTH_DELTA_FAILURE_MODE` (default `warn`; options: `warn|enforce`)

Baseline refresh controls:
- `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE_FILE` (default `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`)
- `CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE` (default `true`; options: `true|false`)
- `CHAINPULSE_MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE_FILE` (default `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`)
- `CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE` (default `true`; options: `true|false`)
- `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE` (default `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom`)
- `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE` (default `false`; options: `true|false`)
- `CHAINPULSE_BASELINE_UPDATE_SCOPE` (optional override: `kpi-only|health-only|dual`; default auto-derived)
- `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` (optional override: subset of `kpi,health,smoke,resolver`; default auto-derived)
- `CHAINPULSE_EMIT_BASELINE_UPDATE_TEMPLATE_PREVIEW` (default `true`; options: `true|false`)
- `CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT` (default `build/migration-governance/baseline-update-template.md`)
- `CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT` (default `true`; options: `true|false`)
- `CHAINPULSE_MIGRATION_ENFORCE_CHANGED_BASELINES_ALIGNMENT` (default `true`; options: `true|false`)

Baseline scope smoke artifact controls:
- `CHAINPULSE_BASELINE_SCOPE_SMOKE_OUTPUT_DIR` (default `build/migration-governance`)
- `CHAINPULSE_BASELINE_SCOPE_SMOKE_JSON_OUTPUT` (default `build/migration-governance/baseline-scope-smoke.json`)
- `CHAINPULSE_BASELINE_SCOPE_SMOKE_PROM_OUTPUT` (default `build/migration-governance/baseline-scope-smoke.prom`)
- `CHAINPULSE_BASELINE_SCOPE_SMOKE_MD_OUTPUT` (default `build/migration-governance/baseline-scope-smoke.md`)
- `CHAINPULSE_BASELINE_SCOPE_SMOKE_DELTA_FAILURE_MODE` (default `warn`; options: `warn|enforce`)

Baseline resolver test artifact controls:
- `CHAINPULSE_BASELINE_RESOLVER_TEST_OUTPUT_DIR` (default `build/migration-governance`)
- `CHAINPULSE_BASELINE_RESOLVER_TEST_JSON_OUTPUT` (default `build/migration-governance/baseline-resolver-test.json`)
- `CHAINPULSE_BASELINE_RESOLVER_TEST_PROM_OUTPUT` (default `build/migration-governance/baseline-resolver-test.prom`)
- `CHAINPULSE_BASELINE_RESOLVER_TEST_MD_OUTPUT` (default `build/migration-governance/baseline-resolver-test.md`)
- `CHAINPULSE_BASELINE_RESOLVER_TEST_DELTA_FAILURE_MODE` (default `warn`; options: `warn|enforce`)
