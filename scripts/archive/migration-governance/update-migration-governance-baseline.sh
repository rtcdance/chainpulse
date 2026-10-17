#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/baseline_update_resolver.sh"

allow_update="${CHAINPULSE_ALLOW_BASELINE_UPDATE:-false}"
baseline_file="${1:-docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom}"
changelog_file="${2:-docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md}"
health_baseline_file="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE_FILE:-docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom}"
refresh_health_baseline="${CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE:-true}"
smoke_baseline_file="${CHAINPULSE_MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE_FILE:-docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom}"
refresh_smoke_baseline="${CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE:-true}"
resolver_baseline_file="${CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE:-docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom}"
refresh_resolver_baseline="${CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE:-false}"
ticket="${CHAINPULSE_BASELINE_UPDATE_TICKET:-MANUAL-BASELINE}"
owner="${CHAINPULSE_BASELINE_UPDATE_OWNER:-platform-team}"
baseline_scope="${CHAINPULSE_BASELINE_UPDATE_SCOPE:-}"
baseline_changed_baselines="${CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES:-}"
rationale="${CHAINPULSE_BASELINE_UPDATE_RATIONALE:-manual baseline refresh}"
emit_template_preview="${CHAINPULSE_EMIT_BASELINE_UPDATE_TEMPLATE_PREVIEW:-true}"
template_output="${CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT:-build/migration-governance/baseline-update-template.md}"

if [[ "$allow_update" != "true" ]]; then
  echo "[migration-kpi-baseline] update blocked: set CHAINPULSE_ALLOW_BASELINE_UPDATE=true to proceed"
  exit 1
fi

case "$refresh_health_baseline" in
  true|false) ;;
  *)
    echo "[migration-kpi-baseline] invalid CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE: $refresh_health_baseline (expected true|false)"
    exit 1
    ;;
esac

case "$refresh_smoke_baseline" in
  true|false) ;;
  *)
    echo "[migration-kpi-baseline] invalid CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE: $refresh_smoke_baseline (expected true|false)"
    exit 1
    ;;
esac

case "$refresh_resolver_baseline" in
  true|false) ;;
  *)
    echo "[migration-kpi-baseline] invalid CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE: $refresh_resolver_baseline (expected true|false)"
    exit 1
    ;;
esac

if [[ -n "$baseline_scope" ]]; then
  if ! baseline_resolver_validate_scope "$baseline_scope"; then
    echo "[migration-kpi-baseline] invalid CHAINPULSE_BASELINE_UPDATE_SCOPE: $baseline_scope (expected kpi-only|health-only|dual)"
    exit 1
  fi
fi

if [[ -n "$baseline_changed_baselines" ]]; then
  if ! baseline_changed_baselines="$(baseline_resolver_normalize_changed_baselines "$baseline_changed_baselines")"; then
    echo "[migration-kpi-baseline] invalid CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES: $baseline_changed_baselines (expected subset of kpi,health,smoke,resolver)"
    exit 1
  fi
fi

case "$emit_template_preview" in
  true|false)
    ;;
  *)
    echo "[migration-kpi-baseline] invalid CHAINPULSE_EMIT_BASELINE_UPDATE_TEMPLATE_PREVIEW: $emit_template_preview (expected true|false)"
    exit 1
    ;;
esac

./scripts/export-migration-governance-kpi.sh
current_file="build/migration-governance/migration-governance-kpi.prom"

if [[ ! -f "$current_file" ]]; then
  echo "[migration-kpi-baseline] missing current KPI file: $current_file"
  exit 1
fi

mkdir -p "$(dirname "$baseline_file")"
cp "$current_file" "$baseline_file"

if [[ "$refresh_health_baseline" == "true" ]]; then
  ./scripts/check-migration-changelog-quality.sh >/dev/null
  health_current_file="build/migration-governance/ticket-registry-health.prom"
  if [[ ! -f "$health_current_file" ]]; then
    echo "[migration-kpi-baseline] missing ticket registry health file: $health_current_file"
    exit 1
  fi

  mkdir -p "$(dirname "$health_baseline_file")"
  cp "$health_current_file" "$health_baseline_file"
fi

if [[ "$refresh_smoke_baseline" == "true" ]]; then
  ./scripts/smoke-baseline-governance-scope.sh >/dev/null
  smoke_current_file="build/migration-governance/baseline-scope-smoke.prom"
  if [[ ! -f "$smoke_current_file" ]]; then
    echo "[migration-kpi-baseline] missing baseline scope smoke file: $smoke_current_file"
    exit 1
  fi

  mkdir -p "$(dirname "$smoke_baseline_file")"
  cp "$smoke_current_file" "$smoke_baseline_file"
fi

if [[ "$refresh_resolver_baseline" == "true" ]]; then
  ./scripts/test-baseline-update-resolver.sh >/dev/null
  resolver_current_file="build/migration-governance/baseline-resolver-test.prom"
  if [[ ! -f "$resolver_current_file" ]]; then
    echo "[migration-kpi-baseline] missing baseline resolver test file: $resolver_current_file"
    exit 1
  fi

  mkdir -p "$(dirname "$resolver_baseline_file")"
  cp "$resolver_current_file" "$resolver_baseline_file"
fi

resolved_scope="$(baseline_resolver_resolve_scope "$baseline_scope" "$refresh_health_baseline" "$refresh_smoke_baseline")"
resolved_changed_baselines="$(baseline_resolver_resolve_changed_baselines "$baseline_changed_baselines" "$refresh_health_baseline" "$refresh_smoke_baseline" "$refresh_resolver_baseline")"

if [[ "$emit_template_preview" == "true" ]]; then
  mkdir -p "$(dirname "$template_output")"
  {
    echo "# Baseline Update Changelog Template"
    echo
    echo "| Field | Value |"
    echo "|---|---|"
    echo "| ticket | ${ticket} |"
    echo "| owner | ${owner} |"
    echo "| scope | ${resolved_scope} |"
    echo "| changed_baselines | ${resolved_changed_baselines} |"
    echo "| rationale | ${rationale} |"
    echo
    echo "## Suggested Entry"
    echo
    echo "- <UTC-ISO8601> | ticket=${ticket} | owner=${owner} | scope=${resolved_scope} | changed_baselines=${resolved_changed_baselines} | rationale=${rationale}"
  } > "$template_output"
fi

mkdir -p "$(dirname "$changelog_file")"
if [[ ! -f "$changelog_file" ]]; then
  {
    echo "# Migration Governance Baseline Changelog"
    echo
  } > "$changelog_file"
fi

timestamp="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
tmp_file="$(mktemp)"
{
  echo "- ${timestamp} | ticket=${ticket} | owner=${owner} | scope=${resolved_scope} | changed_baselines=${resolved_changed_baselines} | rationale=${rationale}"
  cat "$changelog_file"
} > "$tmp_file"
mv "$tmp_file" "$changelog_file"

echo "[migration-kpi-baseline] updated:"
echo "  - $baseline_file"
if [[ "$refresh_health_baseline" == "true" ]]; then
  echo "  - $health_baseline_file"
fi
if [[ "$refresh_smoke_baseline" == "true" ]]; then
  echo "  - $smoke_baseline_file"
fi
if [[ "$refresh_resolver_baseline" == "true" ]]; then
  echo "  - $resolver_baseline_file"
fi
echo "  - $changelog_file"
if [[ "$emit_template_preview" == "true" ]]; then
  echo "  - $template_output"
fi
