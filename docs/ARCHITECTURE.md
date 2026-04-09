# ChainPulse Architecture

**Status**: Active | **Last Updated**: 2026-03-30

## Milestone Rollout (2026-04-04)

- `M1a`: completed
- `M1b`: completed
- `M1c`: completed
- `M2`: completed
- Latest `M2` slice:
  - transport boundary posture is now derived from the selected adapter boundary
    and real gateway bridge facts
  - `transport_boundary_posture` and `transport_boundary_hint` now surface
    through the monolithic deployment summary
  - startup output now shows the selected transport boundary before runtime
    summary inspection
  - `M2` has been recorded as complete and the remaining focus has moved to `M3a`

## Quick Reference

### Layer Structure
```
pkg/
├── domain/          # Business logic (zero external deps)
├── application/     # Use cases, orchestration
├── adapters/        # External integrations (DB, RPC, API)
├── infrastructure/  # Cross-cutting (logging, metrics, config)
└── plugins/         # Swappable implementations
```

### Key Principles
1. **Domain Purity**: No external dependencies in `pkg/domain/`
2. **Dependency Inversion**: Adapters implement domain interfaces
3. **Plugin Swapping**: In-memory ↔ Production (same interface)
4. **Observability First**: Metrics, tracing, health checks built-in

## Implementation Rollout

### Phase 1 (Completed)
- Added foundational packages:
  - `pkg/domain`
  - `pkg/application`
  - `pkg/adapters`
- Established package-level boundary contracts.
- No runtime behavior changes in this phase.

### Phase 2 (Completed)
- Added query vertical compatibility slice:
  - `pkg/domain/query`
  - `pkg/application/query`
  - `pkg/adapters/query`
- Legacy `pkg/services/query` behavior kept unchanged via wrapper approach.

### Phase 3 (Completed)
- Wired query domain facade bridge in API service bootstrap:
  - `cmd/microservices/api-service/main.go`
- Legacy query service remains the active runtime path in this phase.

### Phase 4 (Completed)
- Aligned plugin dependency boundary:
  - `pkg/plugins/api/graphql_handler.go` now depends on `pkg/domain/query.Service`.
- Runtime behavior remains unchanged in this phase.

### Phase 5 (Completed)
- Migrated GraphQL module `EventStore` dependency to domain query contract:
  - `pkg/plugins/api/graphql/resolvers.go`
  - `pkg/plugins/api/graphql/schema.go`
  - `pkg/plugins/api/graphql/mutations.go`
- Added domain EventStore contract at:
  - `pkg/domain/query/event_store.go`

### Phase 6 (Completed)
- Added optional domain query bridge in event query handler:
  - `pkg/plugins/api/event_query_handler.go`
- Default retrieval-service behavior remains primary path.

### Phase 7 (Completed)
- Added runtime bootstrap wiring surface for domain query bridge:
  - `pkg/plugins/api/gateway.go`
  - `cmd/microservices/api-service/main.go`
- Added bridge readiness logging/observability fields in API gateway plugin.

### Phase 8 (Completed)
- Instantiated event query runtime stack in API service bootstrap:
  - `EventStore` + `EventMetadataStore` + `EventRetrievalService`
  - `EventQueryHandler` with domain bridge injection
- Wired runtime handler into API gateway plugin as additive migration surface.

### Phase 9 (Completed)
- Migrated `GET /events/{id}` handler to domain-first execution for hash-like IDs.
- Preserved safe fallback to retrieval service when domain query misses/fails.
- Added focused regression tests for domain-first and fallback behaviors.

### Phase 10 (Completed)
- Added runtime route composition hook in API gateway:
  - `pkg/plugins/api/http/plugin.go` native handler override
  - `pkg/plugins/api/gateway.go` integration composition and status fields
- Wired subscription and health handlers in API service bootstrap:
  - `cmd/microservices/api-service/main.go`
- Enabled runtime request routing through `GatewayRouterIntegration` when all
  required handlers are configured.

### Phase 11 (Completed)
- Added runtime composition integration test proving `/events/{id}` request path
  reaches domain-first event query logic via gateway-composed routing.

### Phase 12 (Completed)
- Aligned monolithic bootstrap wiring with microservice path:
  - database manager + query service + domain bridge
  - event query/subscription/health handlers
  - runtime route composition wiring in API gateway plugin
- Added graceful shutdown coverage for newly introduced runtime components.

### Phase 13 (Completed)
- Added shared runtime bootstrap constructor:
  - `pkg/application/bootstrap/runtime_wiring.go`
- Migrated both `api-service` and `monolithic` startup paths to shared wiring.
- Reduced duplicated initialization and shutdown orchestration logic.

### Phase 14 (Completed)
- Added dependency-injection seams in shared bootstrap constructor for testing.
- Added failure-injection unit tests for config/db/query/event bootstrap stages.
- Added utility guards tests (timeout conversion and nil-safe close).

### Phase 15 (Completed)
- Added shared deployment-mode core config builders and override policy:
  - `pkg/application/bootstrap/core_config.go`
- Added table-driven tests for override precedence and feature flag merge:
  - `pkg/application/bootstrap/core_config_test.go`
- Applied shared config helper in:
  - `cmd/microservices/api-service/main.go`
  - `cmd/monolithic/chainpulse/main.go`

### Phase 16 (Completed)
- Added shared environment override ingestion and validation:
  - `pkg/application/bootstrap/core_config_overrides_env.go`
- Added table-driven tests for env parsing and summary behavior:
  - `pkg/application/bootstrap/core_config_overrides_env_test.go`
- Wired parsed overrides and audit summary logging into both startup paths.

### Phase 17 (Completed)
- Added shared CLI override parsing:
  - `pkg/application/bootstrap/core_config_overrides_cli.go`
- Added precedence merge helper:
  - `MergeCoreConfigOverrides` with policy `CLI > env > mode defaults`
- Added table-driven tests for CLI parsing and precedence merge:
  - `pkg/application/bootstrap/core_config_overrides_cli_test.go`
- Integrated CLI+env merged overrides into both startup paths.

### Phase 18 (Completed)
- Added runtime profile-aware override policy validation:
  - `pkg/application/bootstrap/core_config_override_policy.go`
- Enforced production denylist checks for risky override keys/values.
- Added structured override audit tags and emitted startup metric:
  - `core_config_overrides_applied_total`
- Added policy and audit tag unit tests.

### Phase 19 (Completed)
- Added optional non-production override profile allowlist:
  - `CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES`
  - `pkg/application/bootstrap/core_config_override_policy.go`
- Added allowlist-aware validation path for runtime profile policy:
  - `ValidateCoreConfigOverridesForProfileWithAllowlist`
- Expanded override audit tags with allowlist and profile-state dimensions.
- Integrated allowlist policy parsing/wiring in:
  - `cmd/microservices/api-service/main.go`
  - `cmd/monolithic/chainpulse/main.go`
- Added profile matrix tests covering production/staging/canary/qa behavior:
  - `pkg/application/bootstrap/core_config_override_policy_test.go`

### Phase 20 (Completed)
- Added layered policy presets for override governance:
  - `CHAINPULSE_OVERRIDE_POLICY_PRESET`
  - presets: `strict`, `balanced`, `open`
- Added startup-safe environment-tier defaulting:
  - production => `strict`
  - staging/canary/preprod/qa/test => `balanced`
  - development/local => `open`
- Added explicit allowlist highest-precedence resolution:
  - `CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES` overrides preset behavior
- Added resolved policy runtime helper and integrated into startup paths:
  - `pkg/application/bootstrap/core_config_override_policy.go`
  - `cmd/microservices/api-service/main.go`
  - `cmd/monolithic/chainpulse/main.go`
- Extended override audit tags with policy dimensions:
  - `policy_preset`
  - `policy_preset_source`
- Added policy preset/default/precedence table-driven tests.

### Phase 21 (Completed)
- Introduced per-key override policy bundles for:
  - API type allowlist
  - API port allowed range
  - denied feature flags when enabled
- Added structured policy validation errors with machine-readable codes:
  - `POLICY_PROFILE_NOT_ALLOWLISTED`
  - `POLICY_API_TYPE_DENIED`
  - `POLICY_API_PORT_OUT_OF_RANGE`
  - `POLICY_FEATURE_FLAG_DENIED`
- Added unified policy validation API and startup logging of `policy_code`:
  - `ValidateCoreConfigOverridesWithPolicy`
  - `cmd/microservices/api-service/main.go`
  - `cmd/monolithic/chainpulse/main.go`
- Extended override audit tags with key-policy dimensions:
  - `policy_api_port_min`
  - `policy_api_port_max`
  - `policy_api_type_count`
  - `policy_ff_deny_count`
- Added table-driven policy bundle and error-code tests.

### Phase 22 (Completed)
- Added policy enforcement mode selector:
  - `CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT`
  - values: `enforce` (default) and `audit`
- Added rollout-safe policy evaluation API:
  - `ValidateCoreConfigOverridesWithMode`
  - returns structured enforcement decision (`violation`, `blocked`, `violation_code`)
- Integrated audit-mode startup behavior in both entrypoints:
  - policy violation in `audit` mode emits warning and continues startup
  - `enforce` mode preserves startup block behavior
- Added policy evaluation metric in both startup paths:
  - `core_config_overrides_policy_evaluation_total`
- Extended override audit tags with enforcement dimensions:
  - `policy_enforcement`
  - `policy_violation`
  - `policy_blocked`
  - `policy_violation_code`
- Added enforce/audit parsing and decision matrix tests.

### Phase 23 (Completed)
- Added policy rollout SLO definitions with explicit indicators, targets, and
  error budgets:
  - `docs/operations/POLICY_ROLLOUT_SLO.md`
- Added dashboard and alert query templates for policy metrics:
  - `docs/operations/POLICY_DASHBOARD_QUERIES.md`
- Added enterprise rollout runbook for `audit -> enforce` progression and
  rollback triggers:
  - `docs/operations/POLICY_ROLLOUT_RUNBOOK.md`
- Updated operations guide links to point to in-repo policy operations docs:
  - `docs/guides/OPERATIONS_GUIDE.md`
- Updated documentation index with operations assets:
  - `docs/INDEX.md`

### Phase 24 (Completed)
- Added shared policy evaluation tag builder to avoid startup-path tag drift:
  - `BuildPolicyEvaluationTags`
  - `pkg/application/bootstrap/core_config_override_policy.go`
- Added explicit policy metric/tag contract stability test:
  - `TestPolicyMetricTagContractStability`
  - `pkg/application/bootstrap/core_config_override_policy_test.go`
- Added dedicated contract-check script:
  - `scripts/check-policy-metric-contract.sh`
- Added CI workflow gate job:
  - `.github/workflows/ci.yml` job `policy-contract`
- Added full local micro-loop gate step:
  - `scripts/dev-micro-loop.sh` full mode

### Phase 25 (Completed)
- Added policy metric schema mode control:
  - `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE`
  - modes: `v1`, `dual_write`, `v2`
- Added unified metric emission helper with schema plan:
  - `EmitPolicyOverrideMetrics`
  - `PolicyMetricSchemaPlanForMode`
- Added v2 policy metric names and dual-write migration path:
  - `chainpulse_policy_overrides_applied_total`
  - `chainpulse_policy_overrides_evaluation_total`
- Added schema metadata tags for migration observability:
  - `metric_schema_version`
  - `metric_schema_deprecated`
- Migrated both startup paths to shared schema-aware metric emitter.
- Extended policy contract script/test coverage for schema mode behavior.
- Added metric versioning operations guide:
  - `docs/operations/POLICY_METRIC_VERSIONING.md`

### Phase 26 (Completed)
- Added automated `v1` deprecation cutoff checks in policy contract script:
  - `scripts/check-policy-metric-contract.sh`
  - supports:
    - `CHAINPULSE_POLICY_V1_DEPRECATION_DATE`
    - `CHAINPULSE_POLICY_V1_DEPRECATION_WARN_DAYS`
- Added warning window behavior before cutoff and hard CI fail on/after cutoff
  for non-`v2` schema mode.
- Wired CI policy contract job with explicit schema migration policy env:
  - `.github/workflows/ci.yml`
- Updated metric versioning operations guide with deadline gate semantics:
  - `docs/operations/POLICY_METRIC_VERSIONING.md`

### Phase 27 (Completed)
- Added repo-level migration manifest source of truth:
  - `docs/operations/MIGRATION_MANIFEST.csv`
- Added migration governance guide and schema/process documentation:
  - `docs/operations/MIGRATION_MANIFEST.md`
- Added manifest validation and overdue gate script:
  - `scripts/check-migration-manifest.sh`
  - validates CSV format/status/deadline and fails on overdue items
- Integrated migration manifest check into:
  - `.github/workflows/ci.yml` (`policy-contract` job)
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`check-migration-manifest`)
- Updated documentation indexes/operations links for manifest governance.

### Phase 28 (Completed)
- Extended migration manifest schema with `spec_ref`:
  - `docs/operations/MIGRATION_MANIFEST.csv`
- Added automated spec sync validation in manifest checker:
  - verifies `spec_ref` file exists
  - verifies referenced spec passes `scripts/spec-approval-check.sh`
- Added migration checker escape hatch env (default enforce):
  - `CHAINPULSE_MIGRATION_REQUIRE_SPEC_SYNC=true|false`
- Added missing observability planning spec referenced by manifest:
  - `docs/specs/2026-03-30-observability-indexer-health-sli-harmonization-plan.md`
- Updated migration governance doc for new schema and validation behavior:
  - `docs/operations/MIGRATION_MANIFEST.md`

### Phase 29 (Completed)
- Added migration manifest metadata consistency checks for active items:
  - owner alignment (`manifest owner` vs spec `## Owner`)
  - delivery-status alignment (`manifest status` vs spec `## Delivery Status`)
  - deadline presence check (`manifest deadline` must exist in spec content)
- Added metadata-sync enforcement toggle (default enabled):
  - `CHAINPULSE_MIGRATION_REQUIRE_SPEC_METADATA_SYNC=true|false`
- Updated CI policy-contract job env to explicitly enforce metadata sync.
- Backfilled referenced specs with aligned owner/deadline metadata.

### Phase 30 (Completed)
- Added migration governance KPI exporter:
  - `scripts/export-migration-governance-kpi.sh`
  - outputs:
    - `build/migration-governance/migration-governance-kpi.prom`
    - `build/migration-governance/migration-governance-kpi.md`
- Added CI generation and artifact upload for governance KPI snapshot:
  - `.github/workflows/ci.yml`
- Added local full micro-loop KPI export step:
  - `scripts/dev-micro-loop.sh`
- Added Makefile target:
  - `export-migration-kpi`
- Added governance dashboard query templates:
  - `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`

### Phase 31 (Completed)
- Added migration governance KPI delta compare script:
  - `scripts/compare-migration-governance-kpi.sh`
  - compares current snapshot vs baseline and outputs:
    - `migration-governance-delta.tsv`
    - `migration-governance-delta.md`
- Added baseline snapshot file:
  - `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
- Integrated CI delta workflow:
  - run compare step after KPI export
  - append delta markdown to `GITHUB_STEP_SUMMARY`
  - include delta output in uploaded artifact bundle
- Added local full micro-loop delta generation step.
- Updated governance dashboard doc with PR delta workflow guidance.

### Phase 32 (Completed)
- Added guarded baseline refresh workflow:
  - `scripts/update-migration-governance-baseline.sh`
  - requires `CHAINPULSE_ALLOW_BASELINE_UPDATE=true`
- Added baseline governance gate:
  - `scripts/check-migration-baseline-governance.sh`
  - enforces baseline change must include changelog update
- Added baseline changelog artifact:
  - `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- Integrated baseline governance check in:
  - `.github/workflows/ci.yml` (`policy-contract` job)
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` targets (`check-migration-baseline`, `update-migration-baseline`)

### Phase 33 (Completed)
- Added migration changelog quality checker:
  - `scripts/check-migration-changelog-quality.sh`
  - enforces structured entry format:
    - timestamp
    - `ticket=...`
    - `owner=...`
    - `rationale=...`
- Integrated changelog quality into baseline governance checks.
- Updated baseline refresh script to emit structured changelog entries.
- Integrated changelog quality checks into:
  - `.github/workflows/ci.yml` (`policy-contract` job)
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`check-migration-changelog-quality`)
- Backfilled existing changelog entry to compliant structured format.

### Phase 34 (Completed)
- Added changelog ticket pattern validation:
  - `CHAINPULSE_MIGRATION_TICKET_PATTERN`
  - enforced by `scripts/check-migration-changelog-quality.sh`
- Added shared owner allowlist validation:
  - `CHAINPULSE_MIGRATION_OWNER_ALLOWLIST`
  - enforced in:
    - `scripts/check-migration-changelog-quality.sh`
    - `scripts/check-migration-manifest.sh`
- Wired CI policy-contract job with explicit ticket-pattern/owner-allowlist env.
- Updated governance docs with new policy controls.

### Phase 35 (Completed)
- Consolidated ticket/owner policy-as-code enforcement across governance checks:
  - `scripts/check-migration-changelog-quality.sh`
  - `scripts/check-migration-manifest.sh`
- Standardized CI policy variables for verification mode:
  - `CHAINPULSE_MIGRATION_TICKET_PATTERN`
  - `CHAINPULSE_MIGRATION_OWNER_ALLOWLIST`
- Ensured baseline governance workflow inherits changelog quality checks and
  owner policy constraints.
- Updated governance docs to reflect configurable verification policy controls.
- Added owner drift report export and CI summary integration:
  - `scripts/export-migration-owner-drift-report.sh`
  - uploaded under `build/migration-governance/`

### Phase 36 (Completed)
- Added registry-aware ticket verification adapter in changelog quality checks:
  - `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern|registry|both`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE=file|http`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_FILE`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_URL`
- Added ticket verification failure degrade mode:
  - `CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE=enforce|warn`
- Added default local ticket registry:
  - `docs/operations/MIGRATION_TICKET_REGISTRY.txt`
- Updated CI policy-contract job to enforce `both + file + enforce`.
- Updated governance docs with ticket registry/failure-mode controls.

### Phase 37 (Completed)
- Added ticket registry health telemetry outputs in changelog quality gate:
  - `build/migration-governance/ticket-registry-health.prom`
  - `build/migration-governance/ticket-registry-health.md`
- Added new controls for registry health output paths:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_OUTPUT`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_MD_OUTPUT`
- Added explicit registry check/fallback counters:
  - `chainpulse_migration_ticket_registry_checks_total`
  - `chainpulse_migration_ticket_registry_fallback_events_total`
- Updated CI policy-contract summary to append ticket registry health report.

### Phase 38 (Completed)
- Added HTTP ticket-registry latency SLO controls in changelog quality checks:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MS`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE=off|warn|enforce`
- Added HTTP latency and violation telemetry:
  - `chainpulse_migration_ticket_registry_http_latency_ms`
  - `chainpulse_migration_ticket_registry_http_slo_violations_total`
- Extended ticket registry health markdown with SLO/latency fields for CI artifacts.
- Updated governance query documentation with HTTP latency and SLO violation queries.

### Phase 39 (Completed)
- Added ticket registry health baseline delta script:
  - `scripts/compare-ticket-registry-health.sh`
  - outputs:
    - `build/migration-governance/ticket-registry-health-delta.tsv`
    - `build/migration-governance/ticket-registry-health-delta.md`
- Added regression baseline snapshot:
  - `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- Added regression policy control:
  - `CHAINPULSE_MIGRATION_REGISTRY_HEALTH_DELTA_FAILURE_MODE=warn|enforce`
- Integrated health-delta generation into:
  - `.github/workflows/ci.yml` (`policy-contract` summary)
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`compare-ticket-registry-health`)

### Phase 40 (Completed)
- Extended governed baseline refresh to cover ticket-registry health baseline:
  - default refresh target:
    - `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- Added refresh controls:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE_FILE`
  - `CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=true|false`
- Extended baseline governance check so changelog updates are mandatory when
  either baseline changes:
  - `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
  - `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- Updated operations governance docs with dual-baseline refresh controls.

### Phase 41 (Completed)
- Extended changelog schema with explicit baseline intent tag:
  - `scope=kpi-only|health-only|dual`
- Added scope policy controls in changelog quality gate:
  - `CHAINPULSE_MIGRATION_CHANGELOG_SCOPE_ALLOWLIST`
  - `CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=true|false`
- Updated baseline refresh script to auto-write scope tags:
  - derived from refresh mode (`dual` or `kpi-only`)
  - optional override via `CHAINPULSE_BASELINE_UPDATE_SCOPE`
- Added baseline-diff and changelog-scope alignment gate:
  - `CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true|false`
- Updated CI policy-contract env with explicit scope policy toggles.

### Phase 42 (Completed)
- Added baseline governance scope smoke test script:
  - `scripts/smoke-baseline-governance-scope.sh`
- Added isolated governance smoke scenarios:
  - `dual` scope success
  - `kpi-only` scope success
  - `health-only` scope success
  - mismatch expected failure
- Integrated smoke tests into:
  - `.github/workflows/ci.yml` (`policy-contract` job)
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`smoke-baseline-governance-scope`)
- Updated operations governance workflow documentation with smoke-step guidance.

### Phase 43 (Completed)
- Extended baseline scope smoke script with machine-readable artifact outputs:
  - `build/migration-governance/baseline-scope-smoke.json`
  - `build/migration-governance/baseline-scope-smoke.prom`
  - `build/migration-governance/baseline-scope-smoke.md`
- Added configurable smoke artifact output controls:
  - `CHAINPULSE_BASELINE_SCOPE_SMOKE_OUTPUT_DIR`
  - `CHAINPULSE_BASELINE_SCOPE_SMOKE_JSON_OUTPUT`
  - `CHAINPULSE_BASELINE_SCOPE_SMOKE_PROM_OUTPUT`
  - `CHAINPULSE_BASELINE_SCOPE_SMOKE_MD_OUTPUT`
- Updated CI policy-contract summary to append baseline-scope smoke markdown.
- Updated operations and index docs with smoke artifact references.

### Phase 44 (Completed)
- Added baseline-scope smoke delta compare script:
  - `scripts/compare-baseline-scope-smoke.sh`
  - outputs:
    - `build/migration-governance/baseline-scope-smoke-delta.tsv`
    - `build/migration-governance/baseline-scope-smoke-delta.md`
- Added smoke delta baseline snapshot:
  - `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- Added smoke delta failure policy:
  - `CHAINPULSE_BASELINE_SCOPE_SMOKE_DELTA_FAILURE_MODE=warn|enforce`
- Integrated smoke delta compare into:
  - `.github/workflows/ci.yml` (`policy-contract`)
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`compare-baseline-scope-smoke`)
- Updated CI summary to append baseline-scope smoke delta report.

### Phase 45 (Completed)
- Extended governed baseline refresh workflow to support smoke baseline refresh:
  - default smoke baseline target:
    - `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- Added smoke baseline refresh controls:
  - `CHAINPULSE_MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE_FILE`
  - `CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=true|false`
- Extended baseline governance check to require changelog updates when smoke
  baseline changes.
- Extended scope-alignment logic so smoke baseline changes are treated as
  health-related changes for `health-only|dual` validation.
- Updated CI policy env and operations docs with explicit smoke baseline
  governance controls.

### Phase 46 (Completed)
- Extended changelog schema with explicit changed-set tag:
  - `changed_baselines=<kpi[,health][,smoke][,resolver]>`
- Extended changelog quality gate with:
  - changed-set parsing/normalization
  - scope vs changed-set compatibility enforcement
- Extended baseline refresh workflow to auto-write changed baseline sets:
  - `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` override support
- Extended baseline governance check with changed-set alignment enforcement:
  - `CHAINPULSE_MIGRATION_ENFORCE_CHANGED_BASELINES_ALIGNMENT=true|false`
- Updated smoke fixture changelog entries and CI policy env for changed-set
  enforcement.

### Phase 47 (Completed)
- Extended baseline governance smoke suite with explicit changed-set mismatch
  negative case:
  - `changed_baselines_mismatch_should_fail`
- Smoke suite now validates both:
  - `scope` mismatch failure
  - `changed_baselines` mismatch failure
- Smoke artifact outputs now reflect updated scenario count and per-case result.

### Phase 48 (Completed)
- Extended baseline refresh workflow with generated changelog template preview:
  - default output:
    - `build/migration-governance/baseline-update-template.md`
- Added preview controls:
  - `CHAINPULSE_EMIT_BASELINE_UPDATE_TEMPLATE_PREVIEW=true|false`
  - `CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT`
- Template preview includes resolved fields used by governance checks:
  - `scope`
  - `changed_baselines`
  - suggested changelog entry format
- Updated operations/index docs with template preview artifact references.

### Phase 49 (Completed)
- Added no-mutation baseline update preflight script:
  - `scripts/preflight-migration-baseline-update.sh`
- Preflight outputs resolved governance preview:
  - `scope`
  - `changed_baselines`
  - target baseline files
  - suggested changelog entry
- Added preflight output artifact:
  - `build/migration-governance/baseline-update-preflight.md`
- Added Makefile target:
  - `preflight-migration-baseline-update`
- Updated operations/index docs with preflight workflow and output references.

### Phase 50 (Completed)
- Integrated baseline update preflight into CI `policy-contract` workflow.
- Added CI step summary append for:
  - `build/migration-governance/baseline-update-preflight.md`
- Integrated preflight execution into local full micro-loop before baseline
  governance checks.

### Phase 51 (Completed)
- Added shared resolver helper:
  - `scripts/lib/baseline_update_resolver.sh`
- Consolidated shared logic:
  - scope validation
  - changed baseline normalization
  - resolved scope computation
  - resolved changed baseline computation
- Refactored scripts to source shared helper:
  - `scripts/update-migration-governance-baseline.sh`
  - `scripts/preflight-migration-baseline-update.sh`
- Reduced duplicated governance resolution logic to prevent drift.

### Phase 52 (Completed)
- Added lightweight resolver shell test suite:
  - `scripts/test-baseline-update-resolver.sh`
- Added function-level test coverage for:
  - changed baseline normalization
  - scope validation
  - scope resolution
  - changed-set resolution
- Integrated resolver tests into:
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`test-baseline-update-resolver`)

### Phase 53 (Completed)
- Integrated resolver shell tests into CI `policy-contract` workflow:
  - `./scripts/test-baseline-update-resolver.sh`
- Achieved CI/local parity for baseline resolver governance verification.

### Phase 54 (Completed)
- Extended resolver shell tests with machine-readable outputs:
  - `build/migration-governance/baseline-resolver-test.json`
  - `build/migration-governance/baseline-resolver-test.prom`
  - `build/migration-governance/baseline-resolver-test.md`
- Added configurable resolver test output controls:
  - `CHAINPULSE_BASELINE_RESOLVER_TEST_OUTPUT_DIR`
  - `CHAINPULSE_BASELINE_RESOLVER_TEST_JSON_OUTPUT`
  - `CHAINPULSE_BASELINE_RESOLVER_TEST_PROM_OUTPUT`
  - `CHAINPULSE_BASELINE_RESOLVER_TEST_MD_OUTPUT`
- Updated CI policy-contract summary to append resolver test markdown report.
- Updated operations/index docs with resolver test artifact references.

### Phase 55 (Completed)
- Added resolver test baseline/delta comparator:
  - `scripts/compare-baseline-resolver-test.sh`
- Added resolver test regression baseline:
  - `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom`
- Added resolver delta artifacts:
  - `build/migration-governance/baseline-resolver-test-delta.tsv`
  - `build/migration-governance/baseline-resolver-test-delta.md`
- Wired resolver delta compare into:
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` (`compare-baseline-resolver-test`)
  - CI `policy-contract` workflow
- Updated CI summary and docs with resolver delta report references.

### Phase 56 (Completed)
- Added governed resolver baseline refresh controls:
  - `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE`
  - `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true|false` (default false)
- Extended baseline update preflight/update scripts to support optional resolver
  baseline refresh and preview output.
- Extended changelog changed-set semantics to include `resolver` token while
  preserving existing scope semantics.
- Extended baseline governance diff alignment to validate resolver baseline file
  changes against changelog changed-set tags.
- Updated CI allowlist default and operations docs for resolver changed-set
  policy.

### Phase 57 (Completed)
- Extended baseline governance smoke fixture to include resolver baseline file:
  - `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom` (fixture)
- Added resolver changed-baselines smoke scenarios:
  - `resolver_changed_baselines_alignment`
  - `resolver_changed_baselines_mismatch_should_fail`
- Closed resolver changed-baselines policy regression coverage gap without
  runtime behavior changes.

### Phase 58 (Completed)
- Extended smoke fixture with preflight dependencies:
  - `scripts/preflight-migration-baseline-update.sh`
  - `scripts/lib/baseline_update_resolver.sh`
- Added preflight consistency smoke scenarios:
  - `preflight_without_resolver_refresh`
  - `preflight_with_resolver_refresh`
- Smoke now verifies resolver refresh flag effect on:
  - preflight `changed_baselines` preview
  - resolver baseline target file line presence/absence

### Phase 59 (Completed)
- Extended smoke fixture with guarded update-flow dependencies:
  - `scripts/update-migration-governance-baseline.sh`
  - `scripts/export-migration-governance-kpi.sh`
  - `scripts/test-baseline-update-resolver.sh`
  - `docs/operations/MIGRATION_MANIFEST.csv` (fixture)
- Added guarded update-flow smoke scenario:
  - `guarded_update_with_resolver_refresh`
- Smoke now validates under `CHAINPULSE_ALLOW_BASELINE_UPDATE=true`:
  - resolver baseline file mutation
  - changelog insertion with `changed_baselines=kpi,resolver`
  - post-update governance check success with explicit diff ref

### Phase 60 (Completed)
- Added negative guarded update-flow smoke scenario:
  - `guarded_update_blocked_without_allow_flag`
- Smoke now validates blocked-path governance behavior:
  - update command fails when `CHAINPULSE_ALLOW_BASELINE_UPDATE=false`
  - resolver baseline file remains unchanged
  - changelog is not mutated by blocked updates

### Phase 61 (Completed)
- Added custom resolver baseline path parity smoke scenario:
  - `custom_resolver_baseline_path_parity`
- Smoke now validates custom path override parity across:
  - preflight target file preview
  - guarded update baseline mutation
  - post-update governance diff/alignment check
- Covered non-default resolver baseline path:
  - `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE`

### Phase 62 (Completed)
- Added missing custom resolver baseline path negative smoke scenario:
  - `custom_resolver_baseline_path_missing_should_fail`
- Smoke now validates explicit governance failure when
  `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE` points to a
  missing file.

### Phase 63 (Completed)
- Added blocked update negative smoke scenario for custom resolver path:
  - `custom_resolver_baseline_path_blocked_update_should_not_create_file`
- Smoke now validates custom-path blocked update safety:
  - update command fails when allow flag is false
  - custom resolver baseline file is not created
  - changelog remains unchanged

### Phase 64 (Completed)
- Added custom path preflight no-refresh smoke scenario:
  - `custom_resolver_path_preflight_no_refresh_should_not_show_target`
- Smoke now validates that preflight output does not include resolver target
  preview line when:
  - `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=false`
  - custom resolver baseline path override is configured

### Phase 65 (Completed)
- Added custom path preflight manual changed-baselines override scenario:
  - `custom_resolver_path_preflight_manual_changed_baselines_override`
- Smoke now validates under custom path + refresh disabled:
  - manual `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` value is preserved in
    preflight output
  - resolver target preview line is still absent

### Phase 66 (Completed)
- Added invalid manual changed-baselines override negative scenario:
  - `custom_resolver_path_preflight_invalid_changed_baselines_should_fail`
- Smoke now validates explicit preflight failure when
  `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` contains invalid tokens under
  custom path override flows.

### Phase 67 (Completed)
- Added guarded update invalid changed-baselines negative scenario:
  - `guarded_update_invalid_changed_baselines_should_fail`
- Smoke now validates mutation-path parity for invalid manual overrides:
  - guarded update fails with invalid changed-baselines override
  - resolver baseline file remains unchanged
  - changelog remains unchanged

### Phase 68 (Completed)
- Added template output side-effect safety negative scenarios:
  - `guarded_update_blocked_custom_template_should_not_be_created`
  - `guarded_update_invalid_changed_baselines_custom_template_should_not_be_created`
- Smoke now validates blocked/invalid update failures do not create custom
  template preview output files.

### Phase 69 (Completed)
- Added custom resolver-path template side-effect safety scenarios:
  - `custom_resolver_path_blocked_update_custom_template_should_not_be_created`
  - `custom_resolver_path_invalid_changed_baselines_custom_template_should_not_be_created`
- Smoke now validates template no-side-effects parity across default/custom
  resolver path failure flows.

### Phase 70 (Completed)
- Added smoke markdown family summary aggregation:
  - `scope`
  - `preflight`
  - `update`
  - `custom-path`
  - `template`
- Smoke report now includes per-family total/pass/fail counters to improve
  troubleshooting speed.

### Phase 71 (Completed)
- Added smoke markdown failure-first summary block.
- Smoke report now renders conditional `Failure Summary` section before the full
  case table when failed scenarios exist, while keeping JSON/Prom outputs
  unchanged.

### Phase 72 (Completed)
- Added CI smoke highlight helper:
  - `scripts/append-baseline-scope-smoke-summary.sh`
- Added helper regression test:
  - `scripts/test-append-baseline-scope-smoke-summary.sh`
- Updated CI `policy-contract` summary flow to prepend compact smoke highlights
  before appending the full smoke markdown artifact.

### Phase 73 (Completed)
- Extended smoke CI highlight helper to consume
  `baseline-scope-smoke-delta.md` when available.
- Smoke job summary highlights now include:
  - delta regression status
  - regression-signal preview table
- Kept full smoke and delta markdown artifacts appended unchanged after the
  compact highlight section.

### Phase 74 (Completed)
- Added resolver CI highlight helper:
  - `scripts/append-baseline-resolver-test-summary.sh`
- Added resolver helper regression test:
  - `scripts/test-append-baseline-resolver-test-summary.sh`
- Updated CI `policy-contract` summary flow to prepend compact resolver
  highlights before appending the full resolver test markdown artifact.
- Resolver highlights now include resolver delta regression status and
  regression-signal preview when resolver delta markdown exists.

### Phase 75 (Completed)
- Extracted shared governance summary rendering helpers:
  - `scripts/lib/governance_summary.sh`
- Refactored smoke/resolver summary entrypoints to reuse the shared shell
  library while preserving existing interfaces and rendered output.
- Kept CI workflow call sites unchanged.

### Phase 76 (Completed)
- Added conditional resolver `Failure Summary` block to
  `baseline-resolver-test.md`.
- Extended shared governance summary library with reusable failure-section
  rendering.
- Updated resolver CI highlights to surface failed resolver cases before delta
  details when failures exist.

### Phase 77 (Completed)
- Added governance overview CI summary helper:
  - `scripts/append-governance-overview-summary.sh`
- Added overview helper regression test:
  - `scripts/test-append-governance-overview-summary.sh`
- Updated CI `policy-contract` summary flow to prepend a compact
  `Governance Overview` block before smoke/resolver detail sections.

### Phase 78 (Completed)
- Extended `Governance Overview` to include:
  - ticket-registry health
  - owner-drift
- Governance overview now provides a broader top-level governance operating
  snapshot before detailed report sections.

### Phase 79 (Completed)
- Added normalized overview severity mapping with compact `Level` values:
  - `ok`
  - `warn`
  - `fail`
  - `info`
- Governance overview now preserves raw status while also exposing a normalized
  severity column for faster triage and future presentation styling.

### Phase 80 (Completed)
- Added inline legend to `Governance Overview` for interpreting overview
  `Level` values.
- Overview now explains `ok|warn|fail|info` semantics directly in the CI
  summary block.

### Phase 81 (Completed)
- Added `Source` metadata to `Governance Overview` rows.
- Overview now indicates the primary backing artifact file for each top-level
  governance surface to reduce investigation friction.

### Phase 82 (Completed)
- Added `Details` metadata to `Governance Overview` rows.
- Overview now indicates delta or supplementary artifacts alongside the primary
  source artifact for each governance surface.

### Phase 83 (Completed)
- Added short per-row `Action` hints to `Governance Overview`.
- Overview now suggests the next most useful report or delta artifact to inspect
  based on row severity and available metadata.

### Phase 84 (Completed)
- Added top-level `Overall Health` aggregation above `Governance Overview`.
- Overview now exposes one immediate aggregate governance health signal before
  the detailed per-surface table.

### Phase 85 (Completed)
- Added top-level `Overall Hint` alongside `Overall Health`.
- Overview now translates aggregate governance health into a short operational
  interpretation such as `stable`, `investigate warnings`, or
  `address failures first`.

### Phase 86 (Completed)
- Added top-level `Overall Focus` alongside aggregate health/hint.
- Overview now identifies the first degraded governance surface to inspect when
  overall health is not fully healthy.

### Phase 87 (Completed)
- Added lightweight escalation routing metadata to `Governance Overview`.
- Overview now exposes a per-row `Route` hint and a top-level `Overall Route`
  so degraded governance health can point to the most likely operator group or
  report domain to engage first.

### Phase 88 (Completed)
- Added lightweight playbook routing metadata to `Governance Overview`.
- Overview now exposes a per-row `Playbook` hint and a top-level
  `Overall Playbook` so degraded governance health can point directly to the
  first local document or generated report to open.

### Phase 89 (Completed)
- Added synthesized aggregate remediation guidance to `Governance Overview`.
- Overview now exposes `Overall Next Step` so degraded governance health can
  summarize the first concrete action using focus, route, and playbook hints.

### Phase 90 (Completed)
- Refined `Overall Next Step` wording based on aggregate severity.
- Overview now uses more urgent wording for hard failures and more review-
  oriented wording for softer drift states.

### Phase 91 (Completed)
- Added dedicated aggregate-state regression coverage for `warn` and `info`.
- Overview tests now validate top-level severity-aware guidance beyond the
  failure path.

### Phase 92 (Completed)
- Added dedicated aggregate-state regression coverage for `ok`.
- Extracted lightweight fixture helpers in overview tests so future state
  scenarios can be added with less repeated setup.

### Phase 93 (Completed)
- Extracted aggregate and row assertion helpers in overview regression tests.
- Overview test scenarios now read more like scenario intent and less like
  repeated string-matching boilerplate.

### Phase 94 (Completed)
- Converted primary overview row checks to a compact scenario loop.
- Overview regression tests now keep exact row matching while reducing long
  inline assertion noise in the main scenario section.

### Phase 95 (Completed)
- Converted aggregate overview checks to compact scenario helpers.
- Overview regression tests now represent both row coverage and aggregate-state
  coverage in a more table-driven form.

### Phase 96 (Completed)
- Restructured overview regression scenarios into explicit setup/run/assert
  stages.
- Overview test flow is now easier to scan because fixture mutation,
  execution, and assertions are grouped by responsibility.

### Phase 97 (Completed)
- Added descriptor-based row assertions for non-primary overview states.
- Overview regression tests can now expand `warn`, `info`, and `ok` row
  coverage without returning to long inline markdown strings.

### Phase 98 (Completed)
- Converted overview setup helpers to compact data blocks.
- Overview regression tests now use descriptor-style setup, execution, and
  assertion layers consistently.

### Phase 99 (Completed)
- Added validation for overview setup descriptors.
- Overview regression tests now fail fast on malformed setup kinds or missing
  descriptor fields instead of silently drifting.

### Phase 100 (Completed)
- Added validation for aggregate and row descriptors in overview tests.
- Overview regression harness now fails fast across setup, aggregate, and row
  descriptor layers instead of relying on permissive parsing.

### Phase 101 (Completed)
- Added negative tests for malformed setup, aggregate, and row descriptors.
- Overview regression harness now verifies that descriptor validation failures
  stay explicit and stable, not just that happy paths continue to pass.

### Phase 102 (Completed)
- Extracted narrower parser helpers for setup, aggregate, and row descriptors.
- Overview regression harness now keeps parser details more isolated from the
  scenario flow while preserving the same validation behavior.

### Phase 103 (Completed)
- Added parser-focused micro negative checks for aggregate and row descriptors.
- Overview regression harness now verifies malformed parser inputs both through
  wrapper flows and direct parser entry points.

### Phase 108 (Completed)
- Added additive monolithic wiring for the shared indexing runtime through
  bootstrap-managed no-op and in-memory ports.
- Monolith startup now initializes and starts the shared runtime, exposes basic
  runtime state through logs and metrics, and stops it during graceful
  shutdown without replacing the legacy indexing path.

### Phase 109 (Completed)
- Added optional shared runtime shadow batch forwarding to the default chain
  indexer used by monolith mode.
- Real same-chain event batches now flow into shared runtime envelopes before
  legacy indexing continues, allowing checkpoint and idempotency scaffolding to
  observe production-adjacent input without taking ownership of persistence.

### Phase 110 (Completed)
- Added a legacy-backed shared runtime sink adapter in the indexing layer so
  shared runtime persistence can follow current database and cache semantics.
- Kept monolithic runtime wiring on the no-op sink for now because the
  entrypoint still does not supply real legacy database and cache plugins to
  the indexing path.

### Phase 111 (Completed)
- Added in-memory database and cache adapters for the current monolithic
  indexing contracts and wired them into the monolithic chain indexer path.
- Monolithic indexing now runs with real in-process storage dependencies
  instead of `nil` database and cache placeholders.

### Phase 112 (Completed)
- Switched monolithic shared runtime wiring from the no-op sink to the
  legacy-backed sink adapter.
- Shared runtime shadow batches now exercise the same in-memory storage
  semantics already active on the monolithic legacy indexing path.

### Phase 113 (Completed)
- Added a process-local duplicate-write guard between shared runtime shadow
  persistence and the legacy chain indexer write path.
- Monolithic shadow mode now avoids writing the same in-flight event object
  twice while preserving legacy-path counters and failure behavior.

### Phase 114 (Completed)
- Added explicit metrics for writes now effectively owned by shared runtime in
  monolithic shadow mode.
- Operators can now distinguish shadow-owned suppressions from legacy fallback
  writes when evaluating the next ownership-shift slices.

### Phase 115 (Completed)
- Added ownership-aware status counters to the default chain indexer so local
  debug and status inspection can distinguish shadow-owned and legacy-owned
  indexed events.
- Ownership accounting now exists in both metrics and status surfaces, not just
  aggregate indexed-event totals.

### Phase 116 (Completed)
- Added service-level ownership aggregation in the monolithic entrypoint.
- Monolithic running and shutdown output now summarize shadow-owned versus
  legacy-owned events across all registered chains.

### Phase 117 (Completed)
- Added explicit ownership mode classification to the monolithic service
  surface.
- Monolithic output now states whether the service is currently `idle`,
  `legacy-only`, `shadow`, or `runtime-owned` based on aggregated ownership
  totals.

### Phase 118 (Completed)
- Added runtime metric emission for service-level ownership totals and
  ownership mode code.
- Ownership rollout state is now visible through metrics as well as terminal
  output and per-chain status surfaces.

### Phase 119 (Completed)
- Added ownership summary and mode to the monolithic HTTP health surface.
- `/health` and `/health/components` can now expose an `indexing_runtime`
  component with rollout details such as `ownership_mode`,
  `shadow_owned_events`, and `legacy_owned_events`.

### Phase 120 (Completed)
- Added ownership rollout readiness details to the monolithic readiness
  surface.
- `/health/ready` can now distinguish basic infrastructure readiness from
  rollout readiness by exposing fields such as
  `rollout_ready_for_runtime_owned` and `rollout_status`.

### Phase 121 (Completed)
- Added a shared advisory helper for monolithic ownership rollout decisions.
- Rollout readiness details now expose normalized advisory semantics such as
  `allow`, `hold`, and `unknown`, preparing the path for future guarded
  cutover policy.

### Phase 122 (Completed)
- Added a safe default `report-only` policy mode for ownership rollout
  advisory.
- Rollout readiness details now expose how advisory decisions are currently
  treated by the running service, without enforcing any blocking behavior.

### Phase 123 (Completed)
- Added `manual-gate` as a stronger but still non-blocking policy mode for
  ownership rollout advisory.
- Rollout readiness details can now explicitly signal that operator review is
  expected before progression, without yet enforcing a hard block.

### Phase 124 (Completed)
- Added an explicit acknowledgment input for `manual-gate` ownership rollout
  policy.
- Rollout readiness details can now distinguish pending manual review from
  acknowledged manual review without yet changing runtime behavior.

### Phase 125 (Completed)
- Added a shared effective progression state for ownership rollout control.
- Readiness details now expose a single derived state such as `observe`,
  `review-required`, `acknowledged`, or `ready-for-cutover`, making later
  dashboarding and cutover controls easier to build on.

### Phase 126 (Completed)
- Added runtime metrics for ownership rollout progression state.
- Effective progression, policy mode, and acknowledgment state are now visible
  not only in readiness details but also in metrics suitable for dashboards and
  alerts.

### Phase 127 (Completed)
- Added effective progression state and reason to monolithic console summary
  output.
- Console, readiness, and metrics surfaces now expose the same rollout control
  posture with consistent terminology.

### Phase 128 (Completed)
- Added a dry-run cutover decision hook based on effective progression state.
- Readiness details now expose a non-binding cutover recommendation such as
  `would-allow`, `would-hold`, or `would-unknown`.

### Phase 129 (Completed)
- Added metrics and console summary output for the dry-run cutover decision.
- Cutover recommendation is now visible consistently across readiness, metrics,
  and local terminal output.

### Phase 130 (Completed)
- Added a non-blocking cutover candidate classifier for monolithic rollout
  state.
- Runtime metrics and structured startup/shutdown logs now expose one explicit
  signal for instances that satisfy the current cutover candidate posture.

### Phase 131 (Completed)
- Added an audit-oriented manual approval checkpoint signal above cutover
  candidate detection.
- Readiness details, runtime metrics, structured logs, and console summary now
  expose whether an instance is awaiting manual approval before any future
  cutover action.

### Phase 132 (Completed)
- Added an operator handoff summary above the manual approval checkpoint.
- Readiness details, runtime metrics, structured logs, and console summary now
  expose whether rollout requires no action, operator review, or investigation.

### Phase 133 (Completed)
- Added an approval work item summary above operator handoff.
- Readiness details, runtime metrics, structured logs, and console summary now
  expose the current work item status, owner, and first review fields for
  approval-oriented rollout follow-up.

### Phase 134 (Completed)
- Added an approval checklist summary above the approval work item.
- Readiness details, runtime metrics, structured logs, and console summary now
  expose whether approval prerequisites are ready, incomplete, or need
  investigation.

### Phase 135 (Completed)
- Extracted monolithic ownership rollout summary aggregation into a dedicated
  helper file to keep rollout derivation out of the main entrypoint.
- Centralized readiness detail assembly and ownership rollout metrics emission
  behind the shared summary snapshot without changing current rollout behavior.

### Phase 136 (Completed)
- Extracted ownership rollout control primitives into a dedicated helper file
  so the monolithic entrypoint no longer carries the full approval/cutover
  classifier graph inline.
- Preserved current rollout semantics while separating orchestration
  presentation from rollout control decision helpers.

### Phase 137 (Completed)
- Added a no-op guarded cutover hook signal as the first explicit consumer of
  approval and cutover rollout summaries.
- Exposed guarded hook action and reason through readiness, runtime metrics,
  structured logs, and console summary without changing execution behavior.

### Phase 138 (Completed)
- Added a guarded cutover hook policy layer so operators can distinguish the
  current safe `noop-only` posture from the future-facing `enforce-ready`
  posture.
- Exposed guarded hook policy mode, action, and reason through readiness,
  runtime metrics, structured logs, and console summary while keeping runtime
  behavior non-blocking.

### Phase 139 (Completed)
- Added a unified `would-enforce` interpretation layer above guarded hook
  outcome and policy intent so operators can directly see future enforcement
  posture.
- Exposed would-enforce action and reason through readiness, runtime metrics,
  structured logs, and console summary without changing execution behavior.

### Phase 140 (Completed)
- Added a service-level enforce hint above `would-enforce` so operators can
  quickly read whether the current posture is safe to observe, should hold, or
  should be investigated before any future enforce decision.
- Exposed enforce hint state and reason through readiness, runtime metrics,
  structured logs, and console summary without changing execution behavior.

### Phase 141 (Completed)
- Added a compact guarded cutover overview above the guarded hook leaf signals
  so operators can read the current guarded-cutover posture in one place.
- Exposed guarded cutover overview state and reason through readiness, runtime
  metrics, structured logs, and console summary without changing execution
  behavior.

### Phase 142 (Completed)
- Extracted guarded-cutover summary assembly into a dedicated helper file so
  rollout summary construction no longer has to inline the whole guarded branch.
- Centralized guarded-cutover readiness and metric emission behind the helper
  while preserving existing snapshot fields and current behavior.

### Phase 143 (Completed)
- Extracted approval summary assembly into a dedicated helper file so rollout
  summary construction no longer has to inline the whole approval branch.
- Centralized approval readiness and metric emission behind the helper while
  preserving existing snapshot fields and current behavior.

### Phase 144 (Completed)
- Extracted the common rollout surface skeleton into a dedicated helper file so
  rollout summary assembly no longer has to inline shared readiness and metric
  population logic.
- Preserved current snapshot fields and helper composition while making
  `ownership_rollout_summary.go` closer to a pure orchestration layer.

### Phase 145 (Completed)
- Added a rollout section assembler so ownership rollout summary construction no
  longer has to wire common rollout, approval, and guarded-cutover sections
  inline.
- Preserved existing snapshot fields and behavior while making rollout summary
  composition more obviously sectional.

### Phase 146 (Completed)
- Extracted rollout lifecycle logging and console summary formatting into a
  dedicated presenter helper so `main.go` no longer has to duplicate startup
  and shutdown output blocks inline.
- Preserved current rollout wording and behavior while making the monolithic
  entrypoint closer to a pure orchestration layer.

### Phase 147 (Completed)
- Converted rollout presenter console formatting into a descriptor table so
  startup and shutdown label differences live in one structured place instead
  of a long sequence of manual print statements.
- Preserved current rollout wording and value formatting while making future
  presenter changes less error-prone.

### Phase 148 (Completed)
- Converted rollout lifecycle logs into a descriptor table so the structured log
  surface now evolves in parallel with the descriptor-driven console presenter.
- Preserved current log messages and rollout field selection while making the
  presenter layer more symmetric and easier to extend.

### Phase 149 (Completed)
- Extracted shared rollout presenter accessors so descriptor-driven console
  rendering no longer repeats the same snapshot field paths inline.
- Preserved current rollout wording and formatting while reducing the internal
  change surface for future presenter updates.

### Phase 150 (Completed)
- Split rollout presenter descriptors into ownership, approval, and
  guarded-cutover section builders so the presentation layer mirrors the same
  structural decomposition used by rollout summary assembly.
- Preserved descriptor ordering and current output wording while making the
  presenter layer easier to scan and evolve.

### Phase 151 (Completed)
- Added explicit section assemblers for rollout presenter lines and log
  descriptors so the presenter layer now follows the same
  `section builder -> assembler -> entrypoint` pattern as the rollout summary
  layer.
- Preserved flattened output order and wording while making presenter
  composition more obviously structured.

### Phase 152 (Completed)
- Grouped rollout presenter accessors into ownership, approval, and
  guarded-cutover sections so the internal accessor layer now mirrors the same
  section boundaries used by presenter builders and assemblers.
- Preserved current output wording and ordering while reducing the remaining
  structural mismatch inside the presenter layer.

### Phase 153 (Completed)
- Added a dedicated `/health/rollout` report surface so operators and tooling
  can fetch the current monolithic ownership rollout snapshot from one HTTP
  endpoint instead of stitching it together from multiple surfaces.
- Wired the monolith health handler to expose the existing rollout summary
  snapshot as a read-only report payload without changing readiness semantics.

### Phase 154 (Completed)
- Added a stable metadata envelope to `/health/rollout` so downstream tooling
  can read `report_version`, producer `service`, and `generated_at` without
  inferring them from context.
- Preserved the existing rollout report body while making schema evolution and
  report freshness easier to reason about.

### Phase 155 (Completed)
- Added explicit `report_scope` and `report_source` metadata to
  `/health/rollout` so consumers can distinguish ownership-rollout reports from
  future rollout surfaces without inferring producer context from the route.
- Preserved the existing report structure while making monolith and future
  microservice rollout report alignment easier.

### Phase 156 (Completed)
- Added `report_mode` and `deployment_mode` metadata to `/health/rollout` so
  consumers can distinguish runtime-oriented monolithic rollout reports from
  future surfaces that may share the same schema.
- Preserved existing rollout content while making deployment-aware report
  consumers easier to build.

### Phase 157 (Completed)
- Added `report_id` and `schema_family` metadata to `/health/rollout` so
  consumers can anchor to one stable rollout report identity instead of
  reconstructing it from several metadata fields.
- Preserved the current rollout report contract while making future
  monolith/microservice schema alignment easier.

### Phase 158 (Completed)
- Converted `/health/rollout` from a loose map-backed provider contract to a
  typed rollout report payload while preserving the external JSON shape.
- Hardened monolithic rollout report assembly so future monolith and
  microservice producers can align against a shared typed contract.

### Phase 159 (Completed)
- Moved rollout report types into a dedicated contract file so `/health/rollout`
  producers no longer depend on handler-local type definitions.
- Preserved the typed rollout report contract while making future monolithic
  and microservice reuse cleaner.

### Phase 160 (Completed)
- Added a shared rollout report metadata builder so monolithic report assembly
  no longer hand-populates the stable identity envelope inline.
- Preserved the external `/health/rollout` contract while making future
  monolithic and microservice report identity alignment easier.

### Phase 161 (Completed)
- Extracted a dedicated monolithic rollout report body builder so
  `reportDetails()` now follows a cleaner `metadata builder + body builder`
  assembly flow.
- Preserved the existing `/health/rollout` contract while making future report
  producer reuse easier.

### Phase 162 (Completed)
- Split the monolithic rollout report body builder into `surface`, `approval`,
  and `guarded-cutover` section helpers so report assembly now mirrors the same
  decomposition used by the broader rollout summary model.
- Preserved the existing `/health/rollout` contract while making future
  producer reuse more granular.

### Phase 163 (Completed)
- Added a rollout report section assembler so monolithic report body building
  now follows the same `section helper -> assembler -> entrypoint` structure as
  the existing summary and presenter layers.
- Preserved the existing `/health/rollout` contract while making future report
  producer reuse more structurally consistent.

### Phase 164 (Completed)
- Added a shared rollout report producer interface and function adapter so
  rollout report generation is no longer limited to handler-local callback
  registration.
- Preserved the existing `/health/rollout` contract while giving future
  monolithic and microservice producers a cleaner integration boundary.

### Phase 165 (Completed)
- Added an `api-service` rollout report producer skeleton so the shared
  `/health/rollout` contract now has a second deployment-mode producer beyond
  the monolith.
- Kept the microservice payload explicitly marked as a skeleton with
  `unknown/investigate` rollout posture until real ownership state is wired in.

### Phase 166 (Completed)
- Added a shared ownership-rollout metadata builder so monolith and
  microservice rollout report producers no longer duplicate the common report
  identity envelope inline.
- Preserved the existing `/health/rollout` contract while making cross-mode
  report identity parity code-driven instead of convention-driven.

### Phase 167 (Completed)
- Added an api-service HTTP integration test for `GET /health/rollout` so the
  microservice rollout producer is now validated through the shared route path,
  not just by producer unit tests.
- Preserved the existing `/health/rollout` contract while increasing confidence
  that the second deployment-mode producer is actually reachable.

### Phase 168 (Completed)
- Added HTTP-level parity coverage for shared rollout metadata across monolith
  and api-service so both deployment modes are now checked against the same
  identity contract at the route layer.
- Preserved service-specific rollout body differences while tightening
  cross-mode metadata consistency.

### Phase 169 (Completed)
- Upgraded the `api-service` rollout producer from a pure skeleton to a
  runtime-derived posture when local API runtime wiring is present, while still
  leaving ownership-specific state explicitly non-authoritative.
- Preserved the shared `/health/rollout` contract and cross-mode metadata
  parity while making the microservice payload more truthful about its current
  runtime composition state.

### Phase 170 (Completed)
- Added HTTP-level parity coverage for a minimal shared rollout body boundary
  across monolith and api-service, locking `progression`, `cutover_dry_run`,
  and `guarded_cutover.overview` to a common conservative posture.
- Preserved service-specific rollout body differences while making shared
  cross-mode semantics explicit at the route layer.

### Phase 171 (Completed)
- Made `api-service` runtime-derived rollout reasons enumerate enabled and
  missing local wiring signals so route-level rollout posture is now easier to
  interpret during debugging and operator review.
- Preserved rollout states and shared contract fields while improving the
  actionability of microservice rollout explanations.

### Phase 172 (Completed)
- Extracted a small `api-service` runtime wiring completeness helper so rollout
  mode, advisory status, enabled/missing signal lists, and shared reason
  assembly no longer live inline inside the producer.
- Preserved the existing microservice rollout contract while making the next
  runtime-signal additions easier to land safely.

### Phase 173 (Completed)
- Made `api-service` rollout reports populate ownership summary counts
  explicitly with zero values so microservice reports no longer depend on
  default-zero behavior for unwired ownership counters.
- Preserved the shared rollout schema while keeping ownership-named summary
  fields honest until real microservice ownership counters exist.

### Phase 174 (Completed)
- Restructured the `api-service` rollout producer into `surface`, `approval`,
  and `guarded-cutover` sections so its internal shape now more closely matches
  the monolith rollout report builder.
- Preserved the existing rollout payload while creating a cleaner bridge toward
  future cross-mode body builder reuse.

### Phase 175 (Completed)
- Added a dedicated section assembler for `api-service` rollout production so
  the microservice path now follows the same `section builder -> assembler ->
  apply` model as the monolith rollout report body.
- Preserved the existing rollout payload while making monolith/microservice
  report construction patterns more structurally aligned.

### Phase 176 (Completed)
- Extracted a shared rollout surface apply helper in the common API contract
  package so monolith and `api-service` no longer duplicate the same
  `surface -> details` write plumbing.
- Preserved rollout values while beginning the next layer of cross-mode report
  body reuse from the safest shared surface boundary.

### Phase 177 (Completed)
- Extracted a shared rollout approval apply helper in the common API contract
  package so monolith and `api-service` no longer duplicate `approval ->
  details` write plumbing either.
- Preserved rollout values while extending cross-mode report body reuse one
  layer deeper than the shared surface section.

### Phase 178 (Completed)
- Extracted a shared rollout guarded-cutover apply helper in the common API
  contract package so monolith and `api-service` now reuse all three rollout
  body apply layers: `surface`, `approval`, and `guarded`.
- Preserved rollout values while completing apply-level structural parity for
  the monolith and microservice rollout report bodies.

### Phase 179 (Completed)
- Added a shared typed input model and builder for rollout `surface` sections
  so monolith and `api-service` now share not only surface apply plumbing but
  also the first layer of surface section construction shape.
- Preserved rollout values while moving cross-mode reuse one step beyond apply
  helpers into shared builder inputs.

### Phase 180 (Completed)
- Added a shared typed input model and builder for rollout `approval`
  sections so monolith and `api-service` now share approval construction
  shape in addition to shared approval apply plumbing.
- Preserved rollout values while extending cross-mode builder reuse from
  `surface` into the next stable section boundary.

### Phase 181 (Completed)
- Added a shared typed input model and builder for rollout `guarded-cutover`
  sections so monolith and `api-service` now share guarded section
  construction shape in addition to shared guarded apply plumbing.
- Preserved rollout values while completing shared typed builder coverage for
  `surface`, `approval`, and `guarded-cutover`.

### Phase 182 (Completed)
- Added a shared facade for rollout section assembly so monolith and
  `api-service` now build `surface`, `approval`, and `guarded-cutover`
  together through one shared `sections input -> sections output` path.
- Preserved rollout values while reducing deployment-specific section
  stitching logic at the producer/body-assembler layer.

### Phase 183 (Completed)
- Split rollout `surface` input assembly into shared `core` and `cutover`
  input layers so monolith and `api-service` now compose stable posture fields
  and cutover-specific fields through the same shared merge path.
- Preserved rollout values while tightening cross-mode reuse one level deeper
  inside the `surface` input model.

### Phase 184 (Completed)
- Split rollout `approval` input assembly into shared `flow` and `work item`
  input layers so monolith and `api-service` now compose approval state flow
  and work-item payloads through the same shared merge path.
- Preserved rollout values while extending deeper cross-mode input reuse from
  `surface` into the `approval` model.

### Phase 185 (Completed)
- Split rollout `guarded-cutover` input assembly into shared `hook` and
  `enforcement` input layers so monolith and `api-service` now compose current
  hook posture and future enforcement posture through the same shared merge path.
- Preserved rollout values while completing the same deeper input stratification
  pattern across `surface`, `approval`, and `guarded-cutover`.

### Phase 186 (Completed)
- Added more real `api-service` runtime route signals to the rollout producer
  by exposing gateway plugin wiring state for subscription and health handlers
  alongside the previously tracked runtime routes, event query, and domain bridge.
- Preserved the rollout contract while making `api-service` completeness and
  explanatory reasons more truthful about local runtime route composition.

### Phase 187 (Completed)
- Added query runtime health as a real `api-service` rollout signal by feeding
  `QueryService.Health(ctx)` into the rollout producer alongside local runtime
  route wiring state.
- Preserved rollout contract shape while making `advisory.ready` and rollout
  reasons more meaningful for the fully wired versus degraded query-runtime cases.

### Phase 188 (Completed)
- Refined `api-service` advisory status mapping so fully wired runtime states
  now distinguish healthy, degraded, unhealthy, and unknown query-runtime
  postures without changing the rollout report contract.
- Preserved existing `partial-runtime-wiring` behavior while making fully
  wired runtime states more operationally readable.

### Phase 189 (Completed)
- Added query-health-specific reason hints to runtime-derived `api-service`
  rollout reports so operators now get an immediate next-step hint alongside
  advisory status and raw query health state.
- Preserved rollout contract shape and status enums while making degraded and
  unhealthy query-runtime states more actionable to read.

### Phase 190 (Completed)
- Added compact rollout-posture hints to runtime-derived `api-service`
  rollout reports so operators can scan partial, healthy, degraded, and
  unhealthy runtime-derived states more quickly.
- Preserved rollout contract shape and status enums while making rollout
  reasons easier to interpret at a glance.

### Phase 191 (Completed)
- Added an `event-processor` rollout report producer built on the shared
  rollout report contract, covering skeleton and runtime-derived dependency
  wiring states.
- Added focused producer and handler-level rollout tests while documenting a
  pre-existing package-level import-path blocker that still prevents full
  `go test ./cmd/microservices/event-processor/...` coverage.

### Phase 192 (Completed)
- Added compact rollout-posture hints to runtime-derived `event-processor`
  rollout reports so operators can scan partial versus fully wired dependency
  states more quickly.
- Preserved rollout contract shape and status enums while making
  `event-processor` rollout reasons easier to interpret at a glance.

### Phase 193 (Completed)
- Added a `puller` rollout report producer built on the shared rollout report
  contract, covering skeleton and runtime-derived dependency wiring states.
- Added focused producer and handler-level rollout tests while documenting a
  likely pre-existing package-level blocker pattern that still needs separate
  cleanup before full `go test ./cmd/microservices/puller/...` coverage.

### Phase 194 (Completed)
- Added a focused microservice rollout producer coverage summary documenting
  current producer presence, runtime-derived signals, verification depth, and
  remaining gaps across `api-service`, `event-processor`, and `puller`.
- Captured the current recommendation to add `api-gateway` next before going
  deeper on service-entrypoint health route exposure for the other services.

### Phase 195 (Completed)
- Added an `api-gateway` rollout report producer skeleton built on the shared
  rollout report contract, covering skeleton and runtime-derived gateway wiring
  states.
- Added focused producer and handler-level rollout tests so the fourth
  microservice producer entry now exists without changing the external rollout
  contract.

### Phase 196 (Completed)
- Wired `api-gateway` runtime rollout components through a focused helper that
  now provisions the health handler, runtime route handlers, and rollout report
  producer together at the service entrypoint.
- Added focused gateway-integration coverage proving that `/health/rollout` is
  reachable through the shared gateway route path without changing the rollout
  contract shape.

### Phase 197 (Completed)
- Added shared microservice rollout parity guardrails that now validate common
  metadata identity and runtime-derived body posture boundaries from one place.
- Applied the same parity checks across `api-service`, `event-processor`,
  `puller`, and `api-gateway` focused producer tests to reduce cross-service
  contract drift risk.

### Phase 198 (Completed)
- Refined fully wired `event-processor` rollout advisories so local processing
  dependency health now distinguishes healthy, degraded, unhealthy, and unknown
  runtime-wired states.
- Kept the external rollout contract unchanged while appending component-level
  health status/message details and health-aware posture hints to advisory
  reasons.

### Phase 199 (Completed)
- Refined fully wired `puller` rollout advisories so database and Kafka health
  now distinguish healthy, degraded, unhealthy, and unknown runtime-wired
  states.
- Kept the external rollout contract unchanged while preserving the existing
  partially wired posture and adding health-aware reason details only to the
  fully wired path.

### Phase 200 (Completed)
- Added a dedicated `puller` runtime support helper that now constructs a real
  rollout health handler from the service entrypoint and derives rollout state
  from PostgreSQL health, Kafka health, and local runtime configuration.
- Added focused runtime support coverage so `/health/rollout` is now exercised
  through the wired handler path instead of producer-only tests.

### Phase 201 (Completed)
- Added a dedicated `event-processor` runtime support helper that now
  constructs a real rollout health handler from the service entrypoint and
  derives rollout state from database readiness, event store health, metadata
  store health, and Kafka health.
- Added focused runtime support coverage so `/health/rollout` is now exercised
  through the wired handler path instead of producer-only tests.

### Phase 202 (Completed)
- Applied the shared microservice rollout parity validators to the wired
  `/health/rollout` handler paths for both `puller` and `event-processor`.
- Locked the real entrypoint-level rollout exposure to the same metadata and
  runtime-derived posture boundary already enforced at the producer layer,
  without forcing service-specific reason text to be identical.

### Phase 203 (Completed)
- Applied the same shared microservice rollout parity validators to the wired
  `api-gateway` `/health/rollout` route path.
- Entry-point-level rollout parity now covers the microservices that already
  expose real runtime support wiring, closing the gap between producer-level
  and wired-handler-level rollout guardrails.

### Phase 204 (Completed)
- Refreshed the microservice rollout coverage matrix so it now distinguishes
  producer-level coverage from wired entrypoint-level rollout protection.
- Rebased the “remaining gaps” list onto current reality, with the highest-value
  next work now centered on fuller runtime route exposure and deeper execution
  health signals rather than missing producers.

### Phase 205 (Completed)
- Applied the shared microservice rollout parity validators directly to the
  real `api-service` `/health/rollout` route integration test.
- Route-level rollout parity for `api-service` now uses the same shared
  guardrails as the producer and wired-handler paths already use elsewhere.

### Phase 206 (Completed)
- Added a lightweight `puller` execution-progress tracker that now records poll
  count and last poll time from the real polling loop.
- Folded that execution-progress snapshot into runtime-derived rollout reasons
  so `puller` rollout state no longer depends only on wiring and dependency
  health.

### Phase 207 (Completed)
- Added lightweight Kafka activity extraction for `event-processor` rollout
  state using the existing Kafka health details payload.
- Folded Kafka message/error counts into runtime-derived rollout reasons so
  `event-processor` rollout state now carries one layer of real runtime
  activity beyond pure health classification.

### Phase 208 (Completed)
- Refreshed the rollout coverage summary so it now distinguishes wiring
  signals, dependency-health signals, and lightweight runtime-activity signals.
- Reframed the next gap more precisely: the remaining work is not “any progress
  signal,” but stronger execution-progress semantics beyond the new lightweight
  activity indicators.

### Phase 209 (Completed)
- Upgraded `puller` lightweight poll-loop progress into a stronger derived
  `poll_activity_state` signal with `no-polls-yet`, `active`, and `stale`
  states.
- Folded that derived activity state into runtime-derived rollout reasons so
  the `puller` rollout surface now says not only that polls happened, but also
  whether the loop still appears active.

### Phase 210 (Completed)
- Upgraded `event-processor` lightweight Kafka activity counts into a stronger
  derived `kafka_activity_state` signal with `active` and `stalled` states.
- Folded that derived activity state into runtime-derived rollout reasons so
  the `event-processor` rollout surface now says not only that Kafka counters
  exist, but also whether the processor appears to be seeing live activity.

### Phase 211 (Completed)
- Reused Kafka consumer-group status and lag hooks to derive an
  `event-processor` `consumer_progress_state` with `idle`, `lagging`,
  `active`, and `monitoring` states.
- Folded those consumer-side progress details into runtime-derived rollout
  reasons so the `event-processor` rollout surface now says not only that Kafka
  activity exists, but also whether consumer progress appears idle or backed
  up.

### Phase 212 (Completed)
- Extracted `event-processor` consumer progress assembly into a dedicated
  snapshot helper so status extraction, lag extraction, and progress
  classification no longer live inline inside runtime support.
- Preserved the Phase 211 rollout semantics while giving future offset/lag
  sources a cleaner place to plug into the same rollout surface.

### Phase 213 (Completed)
- Extended Kafka health details with a minimal consumer progress export:
  `active_consumers`, `consumer_group_lag`, and `consumer_group_metrics`.
- Updated the `event-processor` consumer progress helper to prefer those health
  details first, while preserving the older interface-based fallbacks.

### Phase 214 (Completed)
- Extracted `puller` poll progress assembly into a dedicated snapshot helper so
  poll count extraction, last-poll extraction, and activity classification no
  longer live inline inside runtime support.
- Preserved the existing `puller` rollout semantics while bringing its progress
  path closer to the same helper-based shape already used by
  `event-processor`.

### Phase 215 (Completed)
- Added a shared typed execution-progress contract so `puller` poll progress
  and `event-processor` consumer progress now share one snapshot-contract
  layer.
- Preserved rollout semantics while making the two execution-service progress
  helpers easier to compare, test, and evolve together.

### Phase 216 (Completed)
- Added a shared execution-progress reason helper so `puller` and
  `event-processor` now append their progress snapshots to rollout reasons
  through one common helper layer.
- Preserved rollout semantics while reducing the last obvious duplication in
  execution-service progress reason formatting.

### Phase 217 (Completed)
- Added a shared execution-progress facade so `puller` and
  `event-processor` now pass one common execution-progress structure into the
  rollout reason layer instead of choosing individual appenders themselves.
- Preserved rollout semantics while creating a cleaner shared entrypoint above
  the poll-progress and consumer-progress leaf appenders.

### Phase 218 (Completed)
- Added a shared execution-progress input+builder layer so `puller` and
  `event-processor` no longer assemble the execution-progress facade inline
  inside their rollout producers.
- Preserved rollout semantics while moving one more layer of execution-service
  progress assembly into shared API-side helpers.

### Phase 219 (Completed)
- Added a shared execution-progress parity/helper layer so rollout reason
  coverage for `puller` and `event-processor` progress details is now
  validated through one common API-side helper instead of duplicated test
  string checks.
- Preserved rollout semantics while centralizing stable execution-progress
  reason assertions around the shared facade.

### Phase 220 (Completed)
- Added a minimal Kafka-tracked offset export so `event-processor` consumer
  progress now carries not only lag and active-consumer posture, but also a
  shared `consumer_offset` detail derived from Kafka offset tracking.
- Preserved higher-level rollout semantics while making execution progress more
  concrete through the existing health-export and shared execution-progress
  helper path.

### Phase 221 (Completed)
- Extended the shared poll-progress carrier so `puller` rollout progress can
  now represent observed block height, processed block height, and derived
  block gap alongside basic poll-loop activity.
- Preserved higher-level rollout semantics while giving the `puller` progress
  path a more realistic block-progress shape for later runtime wiring.

### Phase 222 (Completed)
- Wired the `puller` block-progress carrier to the real multi-chain runtime
  abstraction so the polling loop can now capture highest observed and highest
  processed block facts from registered pullers.
- Preserved higher-level rollout semantics while moving `puller` block progress
  from test-only injection toward actual runtime sourcing.

### Phase 223 (Completed)
- Added a lightweight checkpoint cadence layer on top of `puller` block
  progress so rollout details can now express whether checkpoint progress is
  uninitialized, pending, or due, plus how many blocks remain until the next
  checkpoint boundary.
- Preserved higher-level rollout semantics while turning raw processed-block
  facts into a more operator-friendly checkpoint-aware progress signal.

### Phase 224 (Completed)
- Added a minimal runtime-backed checkpoint source for `puller` so rollout
  progress can now distinguish checkpoint cadence from the latest recorded
  checkpoint state.
- Preserved higher-level rollout semantics while exposing whether persisted
  checkpoint state is missing, current, or behind processed runtime progress.

### Phase 225 (Completed)
- Added a minimal reorg-aware checkpoint risk layer for `puller` so rollout
  progress can now indicate when a recorded checkpoint may have been invalidated
  by regressed chain progress.
- Preserved higher-level rollout semantics while keeping the implementation
  lightweight and additive, without claiming that full reorg recovery wiring is
  already active in the main loop.

### Phase 226 (Completed)
- Added a lightweight reconciled checkpoint posture for `puller` so rollout
  progress can now distinguish active reorg risk from a later checkpoint that
  supersedes the risky state.
- Preserved higher-level rollout semantics while making recovery posture more
  informative without wiring the full reorg-recovery pipeline into the main
  loop.

### Phase 227 (Completed)
- Added a lightweight per-chain checkpoint summary for `puller` so rollout
  reasons can now identify which chain currently has a recorded, risky, or
  reconciled checkpoint posture.
- Preserved higher-level rollout semantics while making checkpoint/recovery
  posture easier to scan without introducing a larger structured per-chain
  payload contract in this phase.

### Phase 228 (Completed)
- Added a lightweight freshness layer on top of `puller` per-chain checkpoint
  summaries so rollout reasons can now show whether each chain's checkpoint
  posture looks fresh or stale.
- Preserved higher-level rollout semantics while improving operator scanability
  without expanding the rollout contract into a larger structured per-chain
  payload.

### Phase 229 (Completed)
- Added a compact checkpoint coverage hint for `puller` so rollout reasons can
  now summarize how many tracked chains currently have recorded, risky, or
  reconciled checkpoint posture.
- Preserved higher-level rollout semantics while giving operators a faster
  top-level scan over checkpoint coverage without introducing a new structured
  coverage payload.

### Phase 230 (Completed)
- Added a compact checkpoint coverage posture for `puller` so rollout reasons
  can now compress per-chain checkpoint coverage counts into a smaller
  operator-ready conclusion such as healthy, partial, risky, or reconciled.
- Preserved the existing compact coverage-count hint while making checkpoint
  recovery posture easier to scan without adding a larger structured payload.

### Phase 231 (Completed)
- Added a compact per-chain checkpoint posture summary for `puller` so rollout
  reasons can now compress each tracked chain into a smaller operator-readable
  checkpoint conclusion without removing the existing detailed chain summary.
- Preserved the existing detailed per-chain checkpoint facts while making the
  rollout reason easier to skim when operators only need a top-level chain
  posture view.

### Phase 232 (Completed)
- Added a compact consumer progress posture for `event-processor` so rollout
  reasons can now compress existing consumer lag, offset, activity, and
  progress-state facts into a smaller operator-readable execution posture.
- Preserved the existing detailed consumer progress facts while making
  execution-service rollout reasoning easier to skim and more symmetric with
  the recent `puller` checkpoint posture work.

### Phase 233 (Completed)
- Added a shared execution-progress posture helper so compact poll and consumer
  progress posture derivation now lives in the shared rollout execution
  progress layer instead of continuing to drift apart inside individual
  services.
- Updated both `puller` and `event-processor` to consume the shared posture
  helper while preserving their existing service-specific higher-level rollout
  posture signals.

### Phase 234 (Completed)
- Added compact consumer lag severity for `event-processor` so rollout reasons
  can now classify backlog size as low, medium, or high without replacing the
  existing raw lag fact.
- Preserved the existing consumer lag and progress posture details while making
  backlog triage more operator-friendly.

### Phase 235 (Completed)
- Added a compact consumer backlog hint for `event-processor` so rollout
  reasons can now turn backlog posture plus lag severity into a more
  operator-facing next-step hint.
- Preserved the existing lag facts, lag severity, and compact consumer posture
  while making backlog response guidance easier to scan.

### Phase 236 (Completed)
- Added a compact checkpoint recovery hint for `puller` so rollout reasons can
  now turn checkpoint posture, coverage posture, and poll catch-up posture into
  a more operator-facing recovery or observation hint.
- Preserved the existing checkpoint facts and checkpoint posture summaries
  while making checkpoint recovery guidance easier to skim.

### Phase 237 (Completed)
- Added a shared execution operator-hint helper so execution-service rollout
  reasons now publish operator-facing poll and consumer hints through one
  shared helper instead of continuing to drift across service-local reason
  keys.
- Updated both `puller` and `event-processor` to keep their current hint
  semantics while sharing the operator-hint mounting path.

### Phase 238 (Completed)
- Refreshed the rollout coverage summary into a stage-assessment matrix that
  now includes monolith and all currently implemented microservice producers.
- Reframed the current refactor state in terms of:
  - facts
  - posture
  - operator hints
  - exposure depth
  - parity depth
- Added a short stage assessment so the next step can be chosen from a clearer
  “what is already stage-complete versus what is still endgame work” view.

### Phase 239 (Completed)
- Added explicit stage-complete criteria to the rollout coverage summary so the
  repository now documents what must be true before this rollout/control line
  should honestly be called stage-complete.
- Separated “ready to stop” conditions from “still missing baseline rollout
  surfaces” conditions, so future continue/stop decisions can be made against
  written criteria instead of local implementation momentum.

### Phase 240 (Completed)
- Recorded an explicit stop/go decision against the new stage-complete
  criteria, instead of leaving the repository at “criteria exist but current
  answer is implied”.
- Chose the honest current answer:
  this rollout/control line is strong enough to stop treating as open-ended
  baseline work, but it still should not be labeled stage-complete until one
  execution-oriented service is promoted beyond wired-handler exposure or the
  team intentionally reopens the line for deeper parity.

### Phase 241 (Completed)
- Switched from rollout/control stop-go assessment into the first small
  ownership/runtime parity slice by updating `api-service`.
- Added an explicit ownership parity marker so `api-service` rollout now says
  something stable and honest that zeroed ownership counters alone could not:
  runtime wiring is present, but ownership-runtime parity with monolith is
  still pending.

### Phase 242 (Completed)
- Extended the same ownership/runtime parity pattern into `api-gateway` so the
  second most mature route-oriented microservice producer now also makes the
  ownership gap explicit instead of leaving it implicit.
- Kept the shared contract unchanged while making `api-gateway` advisory/work
  item semantics align more closely with the new parity baseline started by
  `api-service`.

### Phase 243 (Completed)
- Turned the new `api-service` + `api-gateway` ownership parity markers into a
  shared baseline instead of leaving them as two parallel service-local
  patterns.
- Added a focused shared validator for the ownership parity marker boundary so
  the route-oriented microservice parity layer is now enforced as one explicit
  shared semantic checkpoint.

### Phase 244 (Completed)
- Recorded the next ownership/runtime parity decision explicitly:
  execution-oriented services do not inherit the new route-oriented ownership
  parity marker baseline by default right now.
- Chose to preserve the current separation between:
  - route/control parity semantics for `api-service` and `api-gateway`
  - execution/runtime posture semantics for `event-processor` and `puller`

### Phase 245 (Completed)
- Reopened the execution-service line with a concrete route/runtime exposure
  slice instead of more document-only parity work.
- Added a minimal runtime HTTP health surface for `event-processor`, so its
  rollout/health endpoints are no longer limited to direct handler invocation
  in focused tests.

### Phase 246 (Completed)
- Matched the same minimal runtime HTTP health-surface promotion in `puller`,
  so both execution-oriented services now expose rollout/health through a real
  runtime HTTP path instead of only through direct handler invocation.
- Preserved the current lightweight execution semantics while moving the
  exposure boundary forward in a symmetrical way.

### Phase 247 (Completed)
- Refreshed the rollout coverage summary again so the repository now explicitly
  records the two strongest current microservice baselines:
  - route-oriented ownership parity
  - execution-oriented minimal HTTP runtime exposure
- Reframed the current stop/go posture around those baselines instead of only
  around individual service slices.

### Phase 248 (Completed)
- Switched to the repository-health line and removed a small cluster of stale
  module-path imports that no longer matched `module chainpulse`.
- Recovered `pkg/infrastructure/deployment/adapter_factory.go` into a minimal,
  honest state that compiles against the current core plugin contracts instead
  of pretending older constructor/interface combinations still exist.

### Phase 249 (Completed)
- Continued the repository-health line by separating PostgreSQL integration
  coverage from ordinary package validation in `pkg/plugins/database`.
- Added an explicit opt-in gate so normal `go test` runs no longer assume a
  local PostgreSQL service just to validate the database package.

### Phase 250 (Completed)
- Continued the repository-health line by separating Kafka and ZeroMQ
  integration slices from ordinary `pkg/plugins/mq` validation.
- Added an explicit opt-in gate so normal `go test ./pkg/plugins/mq/...` no
  longer depends on external MQ peers or hangs on ZeroMQ publish behavior.

### Phase 251 (Completed)
- Continued the repository-health line by recovering a small but blocking test
  contract drift in `pkg/observability`.
- Realigned the distributed tracing test with the current `InjectContext`
  pointer contract, which restored `go test ./pkg/...` to a green state.

### Phase 252 (Completed)
- Pushed the repository-health checkpoint one layer higher and verified that
  the full `./cmd/...` test graph is also green.
- Turned that result into an explicit milestone so the command-layer validation
  state is now recorded alongside the recovered `./pkg/...` graph.

### Phase 253 (Completed)
- Recorded a repository-health stage assessment after confirming both
  `./pkg/...` and `./cmd/...` are green.
- Captured the current recommendation to pause repository health as the
  foreground line unless a broader graph exposes a new concrete blocker.

### Phase 254 (Completed)
- Switched back to the ownership/runtime parity line and moved the
  route-oriented ownership parity marker into shared API helpers.
- Kept `api-service` and `api-gateway` semantics stable while turning that
  baseline from duplicated service-local logic into a shared contract.

### Phase 255 (Completed)
- Continued the same route-oriented ownership/runtime parity cleanup by moving
  ownership parity approval work-item assembly into shared API helpers.
- Kept `api-service` and `api-gateway` work-item semantics stable while
  removing another layer of service-local duplication from the baseline.

### Phase 256 (Completed)
- Continued the route-oriented ownership/runtime parity cleanup by adding a
  shared ownership parity state/input model in the API layer.
- Routed both advisory reason assembly and approval work-item assembly through
  that shared state so `api-service` and `api-gateway` no longer rebuild the
  same parity inputs independently.

### Phase 257 (Completed)
- Continued the same ownership/runtime parity line by introducing a minimal
  shared route ownership parity source abstraction.
- Routed the shared parity state assembly through that provider boundary so the
  next deeper ownership/runtime signal can plug into a stable source layer.

### Phase 258 (Completed)
- Landed the first real shared ownership/runtime source path by adapting
  monolith readiness rollout details into the shared route ownership parity
  source snapshot.
- Kept current route-oriented microservice semantics unchanged while giving the
  shared source layer a concrete monolith-backed input path.

### Phase 259 (Completed)
- Added a compact monolith ownership parity posture on top of the shared
  monolith-readiness-backed source snapshot.
- Kept the route-oriented microservice behavior unchanged while giving the
  shared ownership/runtime source layer a more easily consumable posture.

### Phase 260 (Completed)
- Added a shared monolith ownership parity hint layer on top of the compact
  monolith parity posture.
- Kept the route-oriented microservice behavior unchanged while giving the
  shared ownership/runtime source layer a stable next-step recommendation.

### Phase 261 (Completed)
- Added a small `api-service` adapter that builds a shared route ownership
  parity source from readiness rollout details.
- Exposed shared monolith parity posture and hint in the `api-service`
  rollout advisory reason through a focused producer and route-tested path.

### Phase 262 (Completed)
- Added a matching `api-gateway` adapter that builds a shared route ownership
  parity source from readiness rollout details.
- Exposed shared monolith parity posture and hint in the `api-gateway`
  rollout advisory reason through focused producer and runtime route tests.

### Phase 263 (Completed)
- Added a shared route-oriented monolith parity reason validator in
  `pkg/plugins/api`.
- Moved `api-service` and `api-gateway` onto the shared validator instead of
  repeating monolith parity reason assertions in service-local tests.

### Phase 264 (Completed)
- Added a shared compact monolith parity target decision in
  `pkg/plugins/api` on top of the existing posture and hint layers.
- Moved `api-service` and `api-gateway` onto the shared monolith parity reason
  appender so both route-oriented services emit the same posture, hint, and
  target-decision bundle.

### Phase 265 (Completed)
- Added a shared compact monolith parity action-guidance layer in
  `pkg/plugins/api` on top of posture, hint, and target decision.
- Moved `api-service` and `api-gateway` onto the shared action guidance so
  both route-oriented services emit the same next-step recommendation.

### Phase 266 (Completed)
- Added a shared monolith parity recommendation bundle in `pkg/plugins/api`
  so posture, hint, target decision, and action guidance can be consumed as a
  single stable shape.
- Moved route-oriented focused tests onto the shared bundle validator to make
  endgame assessment and future parity consumers simpler.

### Phase 267 (Completed)
- Added an explicit stage assessment and stop-line proposal for the
  route-oriented deeper parity subline.
- Documented that the current route-oriented line is stage-complete for parity
  target surfacing, but not for true ownership/runtime parity closure.

### Phase 268 (Completed)
- Added an overall endgame assessment across repository health,
  route-oriented deeper parity, and execution-oriented rollout exposure.
- Documented that the rollout/control refactor line is now at a strong pause
  boundary rather than an open-ended implementation track.

### Phase 269 (Completed)
- Added a final pause record for the rollout/control refactor line.
- Documented that future work on this track must be an explicit reopen rather
  than a default continuation.

### Phase 270 (Completed)
- Strengthened the `event-processor` minimal runtime HTTP health surface by
  wiring rollout-aware readiness details and runtime component details into
  `/health/ready` and `/health/components`.
- Refreshed the execution-service coverage summary so `event-processor` and
  `puller` are both described as having minimal runtime HTTP health routes
  rather than only wired-handler exposure.

### Phase 271 (Completed)
- Strengthened the `puller` minimal runtime HTTP health surface by wiring
  rollout-aware readiness details and runtime component details into
  `/health/ready` and `/health/components`.
- Brought `puller` back to the same execution-service-plane level as
  `event-processor` by exposing poll/checkpoint rollout facts through runtime
  readiness and component details.

### Phase 272 (Completed)
- Added an explicit stage assessment and stop-line for the execution-oriented
  service-plane line after `event-processor` and `puller` reached the same
  minimal runtime HTTP/readiness/component baseline.
- Documented that the current execution-oriented line is stage-complete for the
  minimal symmetric health/runtime baseline, but not for a broader service
  plane.

### Phase 273 (Completed)
- Added a stable `meta` block to event query responses so callers can see the
  query source and metadata completeness for both domain-first and
  retrieval-backed event reads.
- Locked the new response meta through handler tests and gateway runtime route
  coverage, giving the event/query data plane a clearer externally visible
  shape without rewriting query execution internals.

### Phase 274 (Completed)
- Extended event query response `meta` so callers can now see the concrete
  query path and whether domain-first resolution fell back to retrieval.
- Tightened handler and gateway runtime-route coverage so the data plane now
  exposes a clearer query-path contract instead of only a coarse source label.

### Phase 275 (Completed)
- Extended event query response `meta` with concrete metadata coverage counts
  so callers can distinguish complete, partial, and missing metadata with more
  precision.
- Kept the change additive and externally visible by locking it through handler
  tests and gateway runtime route coverage rather than changing query execution
  internals.

### Phase 276 (Completed)
- Added a compact metadata coverage posture to event query response `meta` so
  callers can scan response quality without reading raw counts first.
- Kept the change additive and externally visible by validating it through
  handler tests and gateway runtime route coverage rather than changing query
  execution internals.

### Phase 277 (Completed)
- Added a compact execution summary to event query response `meta` so callers
  can scan query path, source, fallback, and coverage posture without reading
  every field individually.
- Kept the change additive and externally visible by validating it through
  handler tests and gateway runtime route coverage rather than changing query
  execution internals.

### Phase 278 (Completed)
- Added a compact consistency posture to event query response `meta` so callers
  can scan query trust posture without interpreting path, fallback, and
  coverage fields individually.
- Kept the change additive and externally visible by validating it through
  handler tests and gateway runtime route coverage rather than changing query
  execution internals.

### Phase 279 (Completed)
- Added a compact query source posture to event query response `meta` so
  callers can distinguish cache, MongoDB, PostgreSQL, retrieval fallback, and
  direct domain/query-service paths.
- Made the `/events` list path explicitly surface its query-service-backed
  posture instead of leaving that execution source implicit.

### Phase 280 (Completed)
- Added a compact reliability hint to event query response `meta` so callers
  can quickly understand whether the response came from a direct path, a
  cache-backed query-service path, a fallback path, or a metadata-degraded
  retrieval path.
- Kept the change additive and externally visible by validating it through
  handler tests and gateway runtime route coverage instead of changing query
  execution internals.

### Phase 281 (Completed)
- Extracted shared builders for single-event and paginated event query
  responses so the event/query data plane no longer assembles its response
  envelope ad hoc at each route.
- Tightened the contract surface by validating that response `data`,
  `pagination`, `meta`, and `timestamp` now flow through one stable assembly
  path.

### Phase 282 (Completed)
- Extracted a shared event query meta input/builder layer so single-event,
  retrieval-list, and domain-list paths now derive common query `meta`
  semantics through one stable assembly path.
- Preserved domain-list-specific source posture and total-count adjustments
  while reducing the remaining contract-derivation duplication inside the
  event/query data plane.

### Phase 283 (Completed)
- Extended `GET /events/chain/{chainId}` to use the domain query path before
  falling back to retrieval so a real non-root event read path now shares the
  same query-service-backed contract evolution as `GET /events`.
- Added `domain-chain` query-path semantics so chain-filtered reads can expose
  query-service source, consistency posture, and reliability hints through the
  existing event query `meta` surface.

### Phase 284 (Completed)
- Extended `GET /events/name/{eventName}` to use the domain query path before
  falling back to retrieval so query-service-backed contract coverage now spans
  another real non-root list path beyond the chain-filtered route.
- Added `domain-name` query-path semantics so event-name reads expose the same
  query-service source, consistency posture, and reliability hints as the other
  evolved event query paths.

### Phase 285 (Completed)
- Extended `GET /events/contract/{address}` to use the domain query path before
  falling back to retrieval so the three filter-list event query routes now
  share a symmetric domain-query-first baseline.
- Added `domain-contract` query-path semantics so contract-filtered reads
  expose the same query-service source, consistency posture, and reliability
  hints as the other evolved event query paths.

### Phase 286 (Completed)
- Added an explicit stage assessment for the event/query data plane after the
  query-service-backed contract expanded across the root list path, the
  single-event path, and the three filter-list event read routes.
- Marked the current state as stage-complete for the query-service-backed event
  query baseline, so further work can be treated as an explicit reopen for a
  new objective rather than automatic baseline continuation.

### Phase 287 (Completed)
- Extended the source-surfacing line into the GraphQL protocol surface by
  adding a compact `querySourcePosture` field on GraphQL event results for the
  `eventsByName` read path.
- Kept the slice intentionally small by distinguishing only cache-hit and live
  event-store reads without reshaping the broader GraphQL response envelope.

### Phase 288 (Completed)
- Added a small cross-protocol assessment after the first GraphQL
  query-source-surfacing slice so the repository now clearly distinguishes
  between the strong HTTP baseline and the narrower GraphQL pilot.
- Marked the current state as cross-protocol source surfacing proven at pilot
  scope rather than full protocol parity.

### Phase 289 (Completed)
- Extended the GraphQL query-source-surfacing pilot from the `eventsByName`
  list path to the single-event `event` path so the pilot now covers both a
  list and a single-item event query shape.
- Preserved the pilot's intentionally small scope by reusing the same compact
  cache-hit versus live event-store source semantics without adding a broader
  GraphQL query meta envelope.

### Phase 290 (Completed)
- Added an explicit stage assessment for the GraphQL query-source-surfacing
  pilot after it expanded across both `event` and `eventsByName`.
- Marked the current GraphQL state as a real pilot boundary rather than full
  GraphQL parity, so any further resolver expansion becomes an explicit reopen.

### Phase 291 (Completed)
- Extended the GraphQL query-source-surfacing pilot to `eventsByAddress` so the
  pilot now covers another list-style event read path in addition to `event`
  and `eventsByName`.
- Preserved the pilot's intentionally small contract by reusing the same
  compact cache-hit versus live event-store source semantics without adding a
  broader GraphQL query meta envelope.

### Phase 292 (Completed)
- Refreshed the GraphQL query-source-surfacing assessment after the pilot
  expanded across `event`, `eventsByName`, and `eventsByAddress`.
- Reclassified the current GraphQL state as a mini-baseline rather than just an
  initial pilot, while still keeping full GraphQL parity explicitly out of
  scope.

### Phase 293 (Completed)
- Extended the GraphQL query-source-surfacing mini-baseline to `eventsByBlock`
  so another event-list read path now exposes the same compact source signal as
  the existing GraphQL mini-baseline.
- Kept the slice intentionally small by surfacing only the live event-store
  path and avoiding any broader GraphQL envelope changes.

### Phase 294 (Completed)
- Refreshed the GraphQL query-source-surfacing assessment after coverage
  expanded across `event`, `eventsByName`, `eventsByAddress`, and
  `eventsByBlock`.
- Reclassified the current state as a strong GraphQL source-surfacing
  mini-baseline with a clearer stop-line, while still keeping full GraphQL
  parity explicitly out of scope.

### Phase 295 (Completed)
- Extended GraphQL query-source surfacing to the root `events` connection so
  the generic paginated list path now exposes the same compact source signal as
  the rest of the GraphQL mini-baseline.
- Kept the slice intentionally small by attaching the existing
  `graphql-event-store` posture to returned event nodes rather than introducing
  a broader GraphQL response-meta envelope.

### Phase 296 (Completed)
- Refreshed the GraphQL query-source-surfacing assessment after coverage
  expanded across `event`, `events`, `eventsByName`, `eventsByAddress`, and
  `eventsByBlock`.
- Reclassified the current state as a stronger GraphQL event-query source
  baseline with an explicit stop-line, while still keeping full GraphQL parity
  out of scope.

### Phase 297 (Completed)
- Restored GraphQL source-surfacing parity for `eventsByName` across the
  schema-builder and resolver implementations so both paths now expose the same
  compact `graphql-event-store` source signal.
- Added focused schema-builder coverage to catch future drift inside the same
  GraphQL event family.

### Phase 298 (Completed)
- Extended GraphQL root `events` source surfacing to cover cache-hit responses
  in addition to live event-store responses, so the generic paginated list path
  now carries both sides of the current compact source contract.
- Kept the change intentionally small by updating connection node payloads in
  place rather than redesigning the GraphQL connection envelope.

### Phase 299 (Completed)
- Extended the GraphQL schema-builder path with cache-aware source surfacing so
  it can now expose the same cache-hit versus live event-store semantics as the
  resolver path across the core cached event-family reads.
- Kept `eventsByBlock` live-only, matching the current resolver behavior, while
  closing the structural cache/source parity gap for the rest of the event
  family.

### Phase 300 (Completed)
- Assessed the GraphQL event-family source work after resolver/schema-builder
  cache-aware parity was established across the core event reads.
- Classified the current state as stage-complete for the GraphQL event-query
  source plane baseline, with future GraphQL expansion treated as an explicit
  reopen rather than default continuation.

### Phase 301 (Completed)
- Extended the gRPC streaming metrics surface with compact source posture
  signals so event stream backends can describe where server-stream and
  client-stream flows are currently being served from.
- Kept the slice intentionally small by exposing posture only through metrics,
  without introducing new protobuf fields or rewriting stream payloads.

### Phase 302 (Completed)
- Extended the gRPC streaming metrics surface from raw source hints to compact
  delivery posture so server-stream and client-stream flows can be read as
  higher-level runtime conclusions rather than just counters.
- Kept the slice intentionally small by classifying posture only from existing
  metrics and source hints, without changing protobuf contracts or stream
  payloads.

### Phase 303 (Completed)
- Extended the gRPC streaming metrics surface from source and delivery posture
  to compact reliability hints so server-stream and client-stream flows now
  expose a more directly consumable operating recommendation.
- Kept the slice intentionally small by deriving hints only from existing
  metrics-level posture, without changing protobuf contracts or stream
  payloads.

### Phase 304 (Completed)
- Assessed the gRPC streaming data-plane work after the metrics surface reached
  compact source posture, delivery posture, and reliability hint.
- Classified the current state as stage-complete for the gRPC streaming
  data-plane baseline, with future expansion treated as an explicit reopen
  rather than default continuation.

### Phase 305 (Completed)
- Added a compact websocket connection metrics surface so the runtime can
  expose transport posture, connection posture, and a reliability hint on top
  of the existing client-count and TLS facts.
- Kept the slice intentionally small by surfacing only runtime metrics-level
  posture rather than changing websocket message contracts.

### Phase 306 (Completed)
- Assessed the websocket runtime connection work after it reached compact
  facts, posture, and reliability hint through the new connection metrics
  surface.
- Classified the current state as stage-complete for the websocket connection
  baseline, with future expansion treated as an explicit reopen rather than
  default continuation.

### Phase 307 (Completed)
- Added a compact HTTP runtime metrics surface so the plugin can expose route
  count, transport posture, runtime posture, and a reliability hint on top of
  the existing running/TLS facts.
- Added a stable router route-count helper so the HTTP transport can report
  registered route coverage without depending on router internals.

### Phase 308 (Completed)
- Assessed the HTTP runtime work after it reached compact facts, posture, and
  reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the HTTP runtime
  baseline, with future expansion treated as an explicit reopen rather than
  default continuation.

### Phase 309 (Completed)
- Extended the shared TLS manager from raw reload/error counters to a compact
  runtime metrics surface with certificate posture, reload posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning reload orchestration or plugin wiring.

### Phase 310 (Completed)
- Assessed the shared TLS runtime work after it reached compact facts,
  posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared TLS runtime
  baseline, with future expansion treated as an explicit reopen rather than
  default continuation.

### Phase 311 (Completed)
- Extended the shared health helper from raw summary counts to a compact
  runtime summary with posture and a reliability hint.
- Kept the slice intentionally small by improving only the shared summary layer
  without redesigning health scheduling or component contracts.

### Phase 312 (Completed)
- Assessed the shared health runtime work after it reached compact facts,
  posture, and reliability hint through the new runtime summary surface.
- Classified the current state as stage-complete for the shared health runtime
  baseline, with future expansion treated as an explicit reopen rather than
  default continuation.

### Phase 313 (Completed)
- Extended the shared connection pool from raw counters to a compact runtime
  metrics surface with capacity posture, runtime posture, and a reliability
  hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning acquire/release or cleanup behavior.

### Phase 314 (Completed)
- Assessed the shared connection-pool runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared
  connection-pool runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 315 (Completed)
- Extended the shared request batcher from raw counters to a compact runtime
  metrics surface with capacity posture, runtime posture, and a reliability
  hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning the batching loop or processor contract.

### Phase 316 (Completed)
- Assessed the shared request-batcher runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared
  request-batcher runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 317 (Completed)
- Extended the shared response compressor from raw counters to a compact
  runtime metrics surface with coverage posture, efficiency posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning compression thresholds or transport
  contracts.

### Phase 318 (Completed)
- Assessed the shared response-compressor runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared
  response-compressor runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 319 (Completed)
- Extended shared monitoring from raw request counters to a compact runtime
  metrics surface with coverage posture, runtime posture, and a reliability
  hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning protocol instrumentation or request
  pipelines.

### Phase 320 (Completed)
- Assessed the shared monitoring runtime work after it reached compact facts,
  posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared monitoring
  runtime baseline, with future expansion treated as an explicit reopen rather
  than default continuation.

### Phase 321 (Completed)
- Extended the shared error handler from raw circuit-breaker state to a compact
  runtime metrics surface with circuit posture, retry posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning the circuit breaker or retry policy.

### Phase 322 (Completed)
- Assessed the shared error-handler runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared error-handler
  runtime baseline, with future expansion treated as an explicit reopen rather
  than default continuation.

### Phase 323 (Completed)
- Extended shared authentication from raw token counts to a compact runtime
  metrics surface with coverage posture, runtime posture, and a reliability
  hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning token validation or permission contracts.

### Phase 324 (Completed)
- Assessed the shared authentication runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the shared
  authentication runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 325 (Completed)
- Extended the shared middleware registry from grouped middleware presence to a
  compact runtime metrics surface with coverage posture, runtime posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the shared runtime
  metrics layer without redesigning middleware execution or route contracts.

### Phase 326 (Completed)
- Assessed the shared middleware-registry runtime work after it reached
  compact facts, posture, and reliability hint through the new runtime metrics
  surface.
- Classified the current state as stage-complete for the shared
  middleware-registry runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 327 (Completed)
- Extended the core plugin registry from raw registry counters to a compact
  runtime metrics surface with coverage posture, runtime posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the core runtime
  metrics layer without redesigning plugin lifecycle orchestration or
  interface contracts.

### Phase 328 (Completed)
- Assessed the core plugin-registry runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the core
  plugin-registry runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 329 (Completed)
- Extended the core API router from raw route counting to a compact runtime
  metrics surface with coverage posture, runtime posture, and a reliability
  hint.
- Kept the slice intentionally small by improving only the core runtime
  metrics layer without redesigning routing or handler contracts.

### Phase 330 (Completed)
- Assessed the core API-router runtime work after it reached compact facts,
  posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the core API-router
  runtime baseline, with future expansion treated as an explicit reopen rather
  than default continuation.

### Phase 331 (Completed)
- Extended the core API layer from raw router wiring to a compact runtime
  metrics surface with coverage posture, runtime posture, and a reliability
  hint.
- Kept the slice intentionally small by improving only the core runtime
  metrics layer without redesigning routing or error-mapper contracts.

### Phase 332 (Completed)
- Assessed the core API-layer runtime work after it reached compact facts,
  posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the core API-layer
  runtime baseline, with future expansion treated as an explicit reopen rather
  than default continuation.

### Phase 333 (Completed)
- Extended the core protocol detector from raw supported-protocol metrics to a
  compact runtime metrics surface with coverage posture, runtime posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the core runtime
  metrics layer without redesigning protocol detection or routing contracts.

### Phase 334 (Completed)
- Assessed the core protocol-detector runtime work after it reached compact
  facts, posture, and reliability hint through the new runtime metrics surface.
- Classified the current state as stage-complete for the core
  protocol-detector runtime baseline, with future expansion treated as an
  explicit reopen rather than default continuation.

### Phase 335 (Completed)
- Extended the core base protocol handler from raw running/router wiring to a
  compact runtime metrics surface with coverage posture, runtime posture, and a
  reliability hint.
- Kept the slice intentionally small by improving only the core runtime
  metrics layer without redesigning handler lifecycle or processor contracts.

### Phase 355 (Completed)
- Extended shared TLS runtime metrics to expose aligned `coverage_posture`
  while preserving existing `certificate_posture` compatibility for current
  consumers.
- Kept the slice intentionally small by aligning runtime posture semantics
  only, without redesigning TLS lifecycle or certificate reload behavior.

### Phase 356 (Completed)
- Assessed shared TLS coverage posture alignment after compatibility
  preservation and classified the current runtime surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future TLS posture field expansion is
  treated as an explicit reopen rather than default continuation.

### Phase 357 (Completed)
- Extended shared error-handler runtime metrics to expose aligned
  `coverage_posture` while preserving existing `circuit_posture`
  compatibility for current consumers.
- Kept the slice intentionally small by aligning runtime posture semantics
  only, without redesigning circuit-breaker or retry behavior.

### Phase 358 (Completed)
- Assessed shared error-handler coverage posture alignment after compatibility
  preservation and classified the current runtime surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future error-handler posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 359 (Completed)
- Extended shared health summary surfacing to include aligned
  `coverage_posture`, `runtime_posture`, and `reliability_hint`
  in addition to existing summary count fields.
- Kept the slice intentionally small by aligning summary posture semantics
  only, without redesigning health checks or component contracts.

### Phase 360 (Completed)
- Assessed shared health summary posture alignment after compatibility
  preservation and classified the current summary surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future summary posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 361 (Completed)
- Extended core plugin-registry legacy metrics to include aligned
  `coverage_posture`, `runtime_posture`, and `reliability_hint`
  in addition to the existing registry counters.
- Kept the slice intentionally small by aligning legacy metrics semantics
  only, without redesigning plugin lifecycle or registry contracts.

### Phase 362 (Completed)
- Assessed core plugin-registry legacy metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy registry posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 363 (Completed)
- Extended shared monitoring legacy protocol metrics to include aligned
  `coverage_posture`, `runtime_posture`, and `reliability_hint`
  in addition to the existing protocol counters.
- Kept the slice intentionally small by aligning legacy protocol metrics
  semantics only, without redesigning monitoring storage or request recording.

### Phase 364 (Completed)
- Assessed shared monitoring legacy protocol metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy monitoring posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 365 (Completed)
- Extended shared request-batcher legacy metrics to include aligned
  `coverage_posture`, `capacity_posture`, `runtime_posture`, and
  `reliability_hint` in addition to the existing batch counters.
- Kept the slice intentionally small by aligning legacy batch metrics
  semantics only, without redesigning batching flow or processor contracts.

### Phase 366 (Completed)
- Assessed shared request-batcher legacy metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy batcher posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 367 (Completed)
- Extended shared connection-pool legacy metrics to include aligned
  `coverage_posture`, `capacity_posture`, `runtime_posture`, and
  `reliability_hint` in addition to the existing pool counters.
- Kept the slice intentionally small by aligning legacy pool metrics
  semantics only, without redesigning pool lifecycle or factory contracts.

### Phase 368 (Completed)
- Assessed shared connection-pool legacy metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy pool posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 369 (Completed)
- Extended shared auth legacy metrics to include aligned `coverage_posture`,
  `runtime_posture`, and `reliability_hint` in addition to the existing token
  counts.
- Kept the slice intentionally small by aligning legacy auth metrics semantics
  only, without redesigning token validation or permission contracts.

### Phase 370 (Completed)
- Assessed shared auth legacy metrics posture alignment after compatibility
  preservation and classified the current surface as stage-complete for this
  baseline.
- Captured an explicit stop-line so future legacy auth posture field expansion
  is treated as an explicit reopen rather than default continuation.

### Phase 371 (Completed)
- Extended shared TLS legacy metrics to include aligned `coverage_posture`,
  `certificate_posture`, `reload_posture`, and `reliability_hint`
  in addition to the existing TLS counters.
- Kept the slice intentionally small by aligning legacy TLS metrics semantics
  only, without redesigning TLS lifecycle or certificate reload behavior.

### Phase 372 (Completed)
- Assessed shared TLS legacy metrics posture alignment after compatibility
  preservation and classified the current surface as stage-complete for this
  baseline.
- Captured an explicit stop-line so future legacy TLS posture field expansion
  is treated as an explicit reopen rather than default continuation.

### Phase 373 (Completed)
- Extended shared error-handler legacy metrics to include aligned
  `coverage_posture`, `circuit_posture`, `retry_posture`, and
  `reliability_hint` in addition to the existing error-handler metrics.
- Kept the slice intentionally small by aligning legacy error-handler metrics
  semantics only, without redesigning circuit-breaker or retry behavior.

### Phase 374 (Completed)
- Assessed shared error-handler legacy metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy error-handler posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 375 (Completed)
- Extended shared response-compressor legacy metrics to include aligned
  `coverage_posture`, `efficiency_posture`, and `reliability_hint`
  in addition to the existing compression metrics.
- Kept the slice intentionally small by aligning legacy compression metrics
  semantics only, without redesigning compression policy or transport flow.

### Phase 376 (Completed)
- Assessed shared response-compressor legacy metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy compression posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 377 (Completed)
- Extended core protocol-detector legacy metrics to include aligned
  `coverage_posture`, `runtime_posture`, and `reliability_hint`
  in addition to the existing protocol metrics.
- Kept the slice intentionally small by aligning legacy detector metrics
  semantics only, without redesigning protocol detection or routing policy.

### Phase 378 (Completed)
- Assessed core protocol-detector legacy metrics posture alignment after
  compatibility preservation and classified the current surface as
  stage-complete for this baseline.
- Captured an explicit stop-line so future legacy detector posture field
  expansion is treated as an explicit reopen rather than default continuation.

### Phase 379 (Completed)
- Assessed the overall endgame state of the current core/shared runtime-surface
  refactor track after runtime baselines, coverage alignments, and legacy
  metrics compatibility passes all reached explicit stop-lines.
- Classified the current track as architecture-complete for this refactor
  sequence, with any follow-up work treated as an explicit reopen tied to a new
  objective.

### Phase 380 (Completed)
- Recorded the final completion state of the current core/shared runtime-surface
  architecture refactor track and marked it complete rather than merely paused.
- Established that future work in this area belongs to a new architecture
  objective instead of default continuation.

### Completion State
- The current core/shared runtime-surface architecture refactor sequence is
  complete.
- Future work should reopen a new foreground architecture objective instead of
  extending this finished track.

### Phase 381 (Completed)
- Reopened a new architecture objective for `event-processor` service-plane
  completeness by wiring the runtime `/metrics` route that startup already
  advertised.
- Kept the slice intentionally small by exporting the existing in-process
  metrics collector as JSON without redesigning the wider metrics contract.

### Phase 382 (Completed)
- Extended the same service-plane completion slice to `puller` by wiring the
  runtime `/metrics` route that startup already advertised.
- Kept the slice intentionally small by exporting the existing in-process
  metrics collector as JSON without redesigning the wider metrics contract.

### Phase 383 (Completed)
- Refreshed the execution-service stage assessment after `event-processor` and
  `puller` both reached a stronger symmetric runtime surface with real
  `/health*` and `/metrics` exposure.
- Recorded an updated stop-line so future execution-service work is reopened
  from this stronger runtime baseline instead of the earlier health-only one.

### Phase 384 (Completed)
- Reopened execution-service service-plane work with a read-only
  `event-processor` `/runtime/summary` route that combines rollout posture,
  component state, and compact metrics summary in one operator-facing surface.
- Kept the slice intentionally small by avoiding mutable control actions and
  wider metrics contract redesign.

### Phase 385 (Completed)
- Extended the same read-only operator-facing runtime-summary slice to `puller`
  so checkpoint/polling posture and compact metrics summary can be read from
  one route.
- Kept the slice intentionally small by avoiding mutable control actions and
  wider metrics contract redesign.

### Phase 386 (Completed)
- Refreshed the execution-service stage assessment after `event-processor` and
  `puller` both reached a stronger read-only operator surface with real
  `/health*`, `/metrics`, and `/runtime/summary` exposure.
- Recorded an updated stop-line so future execution-service work is reopened
  from this stronger operator baseline instead of the earlier metrics-only
  stage.

### Phase 387 (Completed)
- Reopened execution-service control-plane work with a minimal writable
  `puller` runtime control surface for pausing and resuming the polling loop.
- Kept the slice intentionally small by using an in-process, concurrency-safe
  loop controller without introducing distributed control coordination.

### Phase 388 (Completed)
- Assessed the execution control-plane state after `puller` gained the first
  minimal writable runtime control slice for pause/resume operations.
- Recorded an explicit stop-line so wider execution control-plane expansion is
  treated as a deliberate reopen from this pilot instead of default
  continuation.

### Phase 389 (Completed)
- Recorded a no-go feasibility decision for copying the writable pause/resume
  control slice into `event-processor` at the current architecture state.
- Kept the line honest by requiring stronger execution-loop ownership before
  any future `event-processor` writable control implementation.

### Phase 390 (Completed)
- Assessed the current execution control-plane state after the `puller`
  writable control pilot and the explicit `event-processor` no-go decision.
- Recorded a stop-line that classifies the current writable control line as a
  real but intentionally asymmetric pilot rather than a shared baseline.

### Phase 391 (Completed)
- Wired a real processor runtime lifecycle into the `event-processor`
  microservice entrypoint without claiming full consume/process loop ownership.
- Extended the runtime summary and readiness surfaces so operators can see
  processor lifecycle health and processing counters alongside the existing
  Kafka/store runtime state.

### Phase 392 (Completed)
- Refreshed the `event-processor` execution-ownership decision after phase 391.
- Recorded the more honest current state as stronger than the original no-go
  boundary, but still not ready for writable control semantics.

### Phase 393 (Completed)
- Added a minimal consume/process seam to `event-processor` so the microservice
  entrypoint now owns a real bridge from Kafka message consumption into the
  local processor runtime.
- Extended runtime summary and readiness details with consume-loop ownership
  facts while keeping writable control explicitly out of scope.

### Phase 394 (Completed)
- Refreshed `event-processor` writable-control readiness after the new
  consume/process seam was added in phase 393.
- Recorded the more honest current state as closer to a targeted control reopen
  while still not ready for default pause/resume-style control semantics.

### Phase 395 (Completed)
- Recorded the preferred future writable-control target for `event-processor`
  as consume-loop gating rather than processor lifecycle stop/start.
- Kept the line honest by narrowing future reopen work to a safer intake-side
  control candidate instead of broad whole-service pause/resume semantics.

### Phase 396 (Completed)
- Implemented the first narrow writable control slice for `event-processor`
  around consume-loop intake gating.
- Kept the control semantics explicitly narrower than whole-service pause/resume
  by targeting intake-side consume loops instead of processor lifecycle state.

### Phase 397 (Completed)
- Refreshed the execution control-plane state after both `puller` and
  `event-processor` reached real writable control slices.
- Recorded the current maturity as a stronger dual-pilot baseline that is still
  intentionally service-shaped rather than a shared control abstraction.

### Phase 398 (Completed)
- Added an explicit compatibility matrix for the current execution-control
  pilots.
- Recorded both the already-aligned control shape and the intentional
  service-specific differences so future shared-control work has a cleaner
  starting point.

### Phase 399 (Completed)
- Extracted a shared helper for the already-aligned execution-control envelope
  and core control fields.
- Reused the shared helper in both `puller` and `event-processor` while
  preserving intentional route and target differences.

### Phase 400 (Completed)
- Assessed the execution-control line after it reached a dual-pilot writable
  baseline, an explicit compatibility matrix, and a shared envelope/helper
  layer.
- Recorded the current maturity as a strong service-shaped control baseline
  that is now suitable for a default pause boundary.

### Phase 401 (Completed)
- Added a shared validator for the already-aligned execution-control envelope
  and control-core fields.
- Reused the validator in both `puller` and `event-processor` runtime-control
  tests while preserving intentional service-specific route and target
  semantics.

### Phase 402 (Completed)
- Recorded the final assessment for the current execution-control line after it
  reached dual pilots, a compatibility matrix, a shared envelope/helper layer,
  and a shared validator.
- Marked the line as stage-complete for the current service-shaped
  execution-control baseline while explicitly stopping short of claiming a full
  shared control contract.

### Phase 403 (Completed)
- Promoted execution-control target metadata into the aligned contract by
  switching `puller` onto the shared target-aware control envelope.
- Added shared target constants so both writable control pilots now expose
  explicit control targets while preserving service-specific route naming and
  action semantics.

### Phase 404 (Completed)
- Refreshed the execution-control line after target alignment to record the
  stronger aligned-layer maturity more accurately.
- Marked the line as stage-complete for the aligned execution-control baseline
  while still explicitly stopping short of a fully normalized shared control
  contract.

### Phase 405 (Completed)
- Refreshed the overall architecture endgame judgment after execution runtime,
  operator, and control lines all reached explicit baseline stop-lines.
- Recorded the current state as a stronger endgame pause boundary that should
  leave the foreground backlog by default unless a new reopen goal is chosen.

### Phase 406 (Completed)
- Recorded the final completion state of the current architecture optimization
  sequence after the overall endgame refresh.
- Marked the sequence as completed so future work is framed as a new reopen
  objective rather than default continuation.

### Phase 407 (Completed)
- Added a read-only `/runtime/summary` route to the `api-service` runtime route
  composition.
- Exposed compact query runtime posture, rollout posture, and metrics summary
  through the existing query-service-backed microservice entrypoint.

### Phase 408 (Completed)
- Added a compact query runtime summary contract in `pkg/services/query`.
- Upgraded `api-service` `/runtime/summary` to expose query posture fields for
  cache, circuit-breaker, consistency, and reliability while keeping unwired
  concerns explicitly marked as `not-wired`.

### Phase 409 (Completed)
- Added a read-only `/runtime/summary` route to `api-gateway` through the
  existing runtime route composition path.
- Exposed compact gateway runtime posture, rollout posture, and metrics summary
  so the external microservice entrypoint now matches the current runnable-app
  operator baseline more closely.

### Phase 410 (Completed)
- Added a minimal upstream query bridge from `api-gateway` to configured
  `api-service` endpoints for read-only `/events*` routes.
- Reused the existing request-router/load-balancer primitives so the external
  gateway entrypoint now begins to satisfy the runnable-app query path expected
  by the archived architecture blueprint.

### Phase 411 (Completed)
- Upgraded `api-gateway` `/runtime/summary` to expose upstream query bridge
  posture, including configured, attached, and available upstream counts.
- Made the external gateway entrypoint more operator-friendly by surfacing a
  compact query-bridge posture and reliability hint alongside existing runtime
  summary fields.

### Phase 412 (Completed)
- Added a structured JSON error surface for `api-gateway` upstream query bridge
  failures instead of falling back to a plain text `502 Bad Gateway`.
- Exposed compact bridge posture and reliability metadata so clients and
  operators can distinguish upstream query outages from generic gateway errors.

### Phase 413 (Completed)
- Added active upstream query health aggregation for `api-gateway` runtime
  summary using upstream `/health` probes.
- Exposed compact upstream query health state so the gateway entrypoint now
  more honestly reports whether configured `api-service` backends are healthy
  enough to serve query traffic.

### Phase 414 (Completed)
- Changed `api-gateway` upstream defaults to a local-runnable configuration
  centered on `http://localhost:8081`.
- Added `GATEWAY_UPSTREAM_SERVICES` parsing so local runs use a sensible
  default while clustered deployments can still override upstream endpoints
  explicitly.

### Phase 415 (Completed)
- Added a dedicated local runnable quickstart for `api-gateway` covering the
  minimal `api-gateway + api-service` query app path.
- Updated quickstart cross-links so the new local two-service runnable path is
  discoverable from existing microservice docs.

### Phase 416 (Completed)
- Added a focused automated smoke slice for the minimal local runnable
  `api-gateway + api-service` query path.
- Locked both gateway runtime-summary bridge posture and `/events` forwarding in
  one deterministic in-process test.

### Phase 417 (Completed)
- Recorded the current `api-gateway + api-service + puller + event-processor`
  state as stage-complete for the minimal runnable-app baseline.
- Explicitly marked the next highest-value reopen target as orchestration of
  the four-service slice rather than more gateway-only detail work.

### Phase 418 (Completed)
- Added a shared local/dev orchestration entry via
  [`scripts/run-local-runnable-app.sh`](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/run-local-runnable-app.sh)
  so the current runnable baseline can be started from one repository-root
  command.
- Added `minimal` and `full` profiles and aligned the most relevant quickstart
  docs to point back to the shared entry.

### Phase 419 (Completed)
- Added a shared local/dev verification entry via
  [`scripts/verify-local-runnable-app.sh`](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/verify-local-runnable-app.sh)
  so the current runnable baseline can be checked with repeatable `minimal` and
  `full` acceptance profiles.
- Aligned the shared startup entry and quickstarts to point to the new focused
  verification path.

### Phase 420 (Completed)
- Compared the current runnable baseline directly against
  [`ARCHITECTURE_v1.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/archive/ARCHITECTURE_v1.md)
  and recorded the repo as near the lowest acceptable blueprint-aligned
  runnable-app state.
- Identified the remaining highest-value gap as a repository-root runbook and
  boundary statement rather than another new runtime capability line.

### Phase 421 (Completed)
- Added a repository-root runnable-app runbook via
  [`RUNNABLE_APP.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/project/RUNNABLE_APP.md)
  so the current blueprint-aligned runnable slice now has one primary entry for
  startup, verification, dependency assumptions, and service boundaries.
- Updated the root README to point its quick-start flow at the current
  runnable-app path instead of leaving it as a generic older bootstrap path.

### Phase 422 (Completed)
- Recorded the minimum viable blueprint-aligned runnable-app state as completed
  and closed the runnable-app baseline as a finished sequence.
- Marked the repo-root runnable-app runbook as the final entry for this scope
  rather than a continuing default backlog item.

### Phase 423 (Completed)
- Added an optional API gateway security surface via auth and rate-limit
  middleware that stays disabled by default for the runnable baseline.
- Extended the gateway runtime summary and root runbook to expose the security
  posture and the opt-in configuration knobs.

### Phase 424 (Completed)
- Added an optional API service security surface via auth and rate-limit
  middleware that stays disabled by default for the runnable baseline.
- Extended the API service runtime summary and local quickstart docs to expose
  the security posture and the opt-in configuration knobs.

### Phase 425 (Completed)
- Added optional security surfaces to the puller and event-processor runtime
  control/read entrypoints via auth and rate-limit middleware that stay
  disabled by default for the runnable baseline.
- Extended both services' runtime summaries and local docs to expose the new
  security posture and opt-in configuration knobs.

### Phase 426 (Completed)
- Added a repo-root security posture baseline document summarizing the current
  opt-in security surfaces across the four-service runnable baseline.
- Linked the new overview into the root README and docs index so the posture is
  discoverable from the top of the repository.

### Phase 427 (Completed)
- Added a repo-root security rollout/rollback guide for incrementally enabling
  the four-service opt-in security surfaces.
- Linked the guide into the root README and docs index so the enablement order
  and rollback path are discoverable from the top of the repository.

### Phase 428 (Completed)
- Extended the shared local runnable-app verification script so it asserts the
  default-off security posture for the four services as part of the same
  runnable verification flow.
- Kept the verification flow shell-based and aligned to the repo-root security
  baseline and rollout guidance.

### Phase 429 (Completed)
- Added a CI-level runnable-app security check that runs the four command
  package test suites so the default-off security posture remains covered in
  automated workflow execution.
- Kept the CI addition lightweight and aligned to the existing runnable-app and
  security baseline documentation.

### Phase 430 (Completed)
- Tightened the lint scope used by the full quality gates so it targets the
  repository's real source directories instead of test-only or empty parent
  paths.
- Aligned both the local micro-loop and CI lint entrypoints to the same source
  roots so full-gate lint no longer fails on empty package contexts.

### Phase 431 (Completed)
- Normalized the lint execution cache to a workspace-safe location so the full
  gate no longer depends on the host's default Go build cache path.
- Kept the full-gate lint scope aligned to real source files while preserving
  the existing lint rules.

### Phase 432 (Completed)
- Narrowed the fast micro-loop lint step to the changed packages instead of the
  whole repository, keeping the fast gate focused on the current diff.
- Left the full lint gate responsible for broader source-root coverage and the
  cache-normalized execution path.

### Phase 433 (Completed)
- Added a monolithic runtime summary route that exposes the shared indexing
  runtime contract, ownership rollout posture, gateway wiring posture, and a
  compact metrics snapshot through the existing gateway surface.
- Kept the change additive so the monolithic runnable baseline remains intact
  while becoming easier to inspect from a single-process debug perspective.

### Phase 434 (Completed)
- Replaced the monolithic shared runtime's no-op failure router with a real
  in-memory failure journal and reused the same journal as an additive replay
  source.
- Extended shared runtime status and the monolithic runtime summary so
  checkpoint, idempotency, failure-routing, replay, duplicate-skip, and latest
  checkpoint facts are now visible through the runnable/debug baseline.

### Phase 435 (Completed)
- Wrapped the `event-processor` primary processor runtime with an additive
  shared-runtime shadow adapter that lazily provisions per-chain shared
  indexing runtimes after successful event processing.
- Upgraded the local processor runtime so `ProcessEvent` uses a started
  in-memory database plugin, and extended `/runtime/summary` to expose the
  shared-runtime shadow status as part of the microservice indexing seam.

### M1a Slice 4 (Completed)
- Wired the application-level `reorg.ReorgHandler` into the monolithic puller
  runtime as a per-chain rollback seam instead of leaving reorg handling as a
  disconnected service.
- Added minimal in-memory block snapshot persistence plus monolithic
  `/runtime/summary` reorg posture surfacing so the single-process indexing
  path can now truthfully express best-effort rollback ownership.

### M1a Slice 5 (Completed)
- Added a compact monolithic puller runtime surface that reports aggregated
  puller health posture and reliability hints instead of leaving puller state
  buried inside the monolithic process.
- Extended gateway runtime-route composition so monolithic mode can expose a
  read-only `/runtime/control` endpoint using the shared runtime-control
  envelope without overclaiming writable pause/resume semantics.

### M1a Assessment (Completed)
- Recorded that the monolithic foundational runtime baseline is now stage-complete:
  the monolith has ingest closure, indexing-backed query closure, chain-route
  alignment, minimal reorg rollback ownership, and a compact puller runtime surface.
- Shifted active milestone focus from `M1a` to `M1b`, so future work can target
  resilience hardening instead of reopening the foundational runtime slices.

### M1b Slice 1 (Completed)
- Added bounded restart/backoff supervision to the monolithic per-chain pull
  loops so unexpected `Poll(...)` exits no longer permanently kill the runtime
  ingestion loop for that chain.
- Extended the monolithic runtime summary puller surface with restart/failure/
  backoff facts and a recovering posture, making the first resilience slice in
  `M1b` visible through the existing operator-facing runtime summary.

### M1b Slice 2 (Completed)
- Added a real shared-runtime `RecoverChain(...)` seam so checkpoint loading and
  replay are no longer just static capability flags in monolithic mode.
- Wired monolithic startup to run per-chain recovery probes and extended the
  runtime summary with concrete recovery facts, posture, and reliability hints.

### M1b Slice 3 (Completed)
- Tightened the top-level monolithic runtime summary so it no longer reports a
  fully healthy lifecycle while puller, recovery, or reorg resilience seams are
  in watch/degraded states.
- Added additive top-level `fault_posture` and `reliability_hint` fields and
  derived runtime lifecycle from the real resilience surfaces instead of only
  from shared-runtime start state plus gateway wiring.

### M1b Assessment (Completed)
- Recorded that the monolithic resilience baseline is now stage-complete:
  the monolith has pull-loop restart/backoff ownership, checkpoint/replay
  recovery closure, and truthful degraded/fault runtime semantics.
- Shifted active milestone focus from `M1b` to `M1c`, so future work can target
  observability and gateway hardening instead of reopening resilience slices.

### M1c Slice 1 (Completed)
- Added an explicit monolithic gateway runtime-route `/metrics` contract instead
  of only mentioning metrics in startup output and docs.
- Reused the existing metrics collector export through gateway runtime-route
  composition and surfaced `metrics_route_enabled` in the monolithic runtime
  summary gateway section.

### M1c Slice 2 (Completed)
- Added a real gateway runtime-route inventory derived from the initialized
  router state instead of relying only on boolean feature flags.
- Extended the monolithic runtime summary gateway section with registered/runtime
  route counts plus a compact runtime-surface posture, making the current
  operator-facing route footprint explicit for `M1c`.

### M1c Slice 3 (Completed)
- Hardened the gateway route method contract so wrong-method requests no longer
  fall through to read-only runtime routes just because the path matches.
- Added explicit `405 Method Not Allowed` behavior with `Allow` headers and
  surfaced a compact method-contract posture through the monolithic runtime
  summary gateway section.

### M1c Assessment (Completed)
- Recorded that the monolithic observability + gateway baseline is now
  stage-complete: the monolith has explicit health/summary/control/metrics
  runtime routes, route inventory surfacing, and truthful method-boundary
  hardening for the current blueprint-aligned baseline.
- Shifted active milestone focus from `M1c` to `M2`, so future work can target
  dual-mode switching instead of continuing to add small monolithic gateway
  slices by default.

### M2 Slice 1 (Completed)
- Turned `DEPLOYMENT_MODE` from a README-only environment variable into a real
  monolithic cmd-layer contract with normalization and safe fallback behavior.
- Extended monolithic startup/runtime summary so deployment-mode posture is now
  visible to operators before the later adapter-factory slices land.

### M2 Slice 2 (Completed)
- Added a cmd-layer monolithic adapter profile resolver so deployment mode now
  produces an explicit adapter/profile decision instead of remaining only a mode
  label.
- Extended monolithic startup/runtime summary with adapter-profile facts, making
  the current `M2` boundary explicit: monolithic profile is concrete, while the
  microservice profile is still an intent/seam pending later adapter cutover.

### M2 Slice 3 (Completed)
- Made monolithic indexing storage selection deployment-mode-aware instead of
  always constructing the same in-memory storage path regardless of adapter profile.
- Kept the reorg block-snapshot seam alive under microservice intent by wrapping
  the compatibility database path with a minimal snapshot-capable layer.

### M2 Slice 4 (Completed)
- Made monolithic query surface selection deployment-mode-aware instead of
  always forcing the indexing-backed query cutover.
- Kept `monolithic` on the indexing-backed query surface and kept
  `microservice` intent on the managed-db/shared runtime query path, while
  aligning runtime/deployment summary fields with the real selected query adapter.

### M2 Slice 5 (Completed)
- Made monolithic gateway exposure deployment-mode-aware instead of always
  wiring the full in-process business API surface.
- Kept `monolithic` on the full in-process gateway surface, while `microservice`
  intent now exposes runtime/operator routes only and no longer overclaims
  monolithic query/subscription ownership.
- Tightened shared gateway route registration so business routes are no longer
  registered when the corresponding runtime handlers are absent.

### M2 Assessment (Current)
- Recorded that `M2` has crossed into a minimum truthful dual-mode baseline:
  deployment mode now affects real monolithic cmd-layer choices for storage,
  query surface, and gateway surface, instead of remaining only startup metadata.
- Kept `M2` in progress because the strongest remaining gap is still shared
  wiring and transport-boundary alignment across dual-mode execution, not more
  summary-only posture work.

### M2 Slice 6 (Completed)
- Added a narrow monolithic transport-boundary classifier so deployment summary
  now reports:
  - `transport_boundary_posture`
  - `transport_boundary_hint`
- The transport boundary is now derived from both the selected adapter boundary
  and real gateway bridge facts:
  - configured upstreams
  - attached upstream handlers
  - available upstream handlers
- Extended startup output so the selected transport boundary is visible before
  inspecting runtime summary.

### M2 Completion (Completed)
- Marked `M2` complete after confirming the repository now has a minimum
  truthful dual-mode baseline rather than just deployment-mode metadata.
- Recorded the handoff boundary from `M2` to `M3a`: remaining work is now about
  microservice deployment verification and broader runnable validation, not more
  monolithic dual-mode seam shaping.

### M3a Slice 1 (Completed)
- Added a repo-root independent microservice entrypoint verification script so
  `api-service`, `api-gateway`, `event-processor`, and `puller` can each be
  started and checked in isolation.
- The first `M3a` slice focuses on operator-surface reachability:
  - `/health`
  - `/runtime/summary`
  - `/runtime/control` for execution services

### M3a Slice 2 (Completed)
- Added a focused four-service deployment smoke script that starts the current
  full local runnable profile and verifies the shared deployment baseline.
- Reused the existing full-profile startup and verification entries instead of
  creating a second orchestration path, then added narrow cross-entrypoint
  assertions for gateway bridge readiness and service runtime-summary reachability.

### M3a Completion (Completed)
- Marked `M3a` complete after confirming the repository now has a minimum
  microservice deployment-verification baseline instead of only runnable-app
  scripts and ad hoc service docs.
- Recorded the handoff boundary from `M3a` to `M3b`: the next work is about
  observability and alerting completeness, not more baseline deployment smoke.

### M3b Slice 1 (Completed)
- Added a repo-root microservice observability baseline verification script so
  the current four-service slice can be checked for shared metrics, runtime
  summary, and rollout-advisory reachability.
- Reused the existing full runnable profile and focused the first `M3b` slice on
  proving operator-visible signals are consistently reachable before introducing
  heavier observability tooling.

### M3b Slice 2 (Completed)
- Added a repo-root microservice alert-readiness verification script so the
  current four-service slice can be checked for shared rollout advisory output.
- Kept the implementation narrow by validating the existing `/health/rollout`
  contracts instead of introducing a real external alerting platform at this
  stage.

### M3b Completion (Completed)
- Marked `M3b` complete after confirming the repository now has a minimum
  observability and alert-readiness baseline instead of only service-local
  runtime routes.
- Recorded the handoff boundary from `M3b` to `M3c`: the next work is about
  production-readiness rehearsal and final readiness closure, not more baseline
  observability verification.

### M3c Slice 1 (Completed)
- Added a repo-root production-readiness rehearsal script so the current
  deployment, observability, and alert-readiness baselines can be exercised as
  one ordered drill.
- Kept the implementation intentionally thin by sequencing the existing
  baseline verification entries instead of duplicating their logic.

### M3c Completion (Completed)
- Marked `M3c` complete after confirming the repository now has a minimum
  production-readiness rehearsal baseline rather than only separate verification
  entries.
- Recorded that the current milestone sequence is now complete:
  - `M1a`
  - `M1b`
  - `M1c`
  - `M2`
  - `M3a`
  - `M3b`
  - `M3c`

### Final Sequence Completion (Completed)
- Recorded a final sequence-level assessment and completion record for the full
  `M1a → M1b → M1c → M2 → M3a → M3b → M3c` execution path.
- From this point, further work should reopen as a new objective instead of
  being described as unfinished milestone-sequence work.

### Reopen: Compose Stack Verification Baseline
- Opened a new post-sequence objective around docker-compose / platform
  orchestration verification.
- Added a lightweight repo-root verification script that checks the current
  compose file still declares the expected infrastructure and observability
  services, while keeping the first reopen slice intentionally non-destructive.
- Added a dedicated `docker-compose.microservices.yml` profile that resolves the
  four foreground microservices together with shared infrastructure and
  observability services through the existing generic microservice Dockerfile.
- Added a compose-based microservice readiness smoke entry that can bring the
  new microservice compose profile up, wait for the four foreground services,
  and reuse the existing full runnable verification baseline.

## Detailed Documentation

- **Directory Structure**: `docs/architecture/DIRECTORY_STRUCTURE.md`
- **Deployment Modes**: `docs/guides/DEPLOYMENT_GUIDE.md`
- **Plugin System**: `pkg/plugins/README.md`
- **Observability**: `pkg/observability/README.md`
- **Testing Strategy**: `docs/TESTING.md`

## Architecture Decisions

See `.codex/skills/` for enforced patterns:
- `web3-go-architecture-guardrails` - Layer boundaries
- `adapter-contract-testing` - Plugin contracts
- `observability-slo-gates` - Telemetry requirements

---
**Note**: Original 907-line document archived to `docs/archive/ARCHITECTURE_v1.md`
