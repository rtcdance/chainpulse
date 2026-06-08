#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/baseline_update_resolver.sh"

output_file="${1:-build/migration-governance/baseline-update-preflight.md}"
ticket="${CHAINPULSE_BASELINE_UPDATE_TICKET:-MANUAL-BASELINE}"
owner="${CHAINPULSE_BASELINE_UPDATE_OWNER:-platform-team}"
rationale="${CHAINPULSE_BASELINE_UPDATE_RATIONALE:-manual baseline refresh}"
baseline_scope="${CHAINPULSE_BASELINE_UPDATE_SCOPE:-}"
baseline_changed_baselines="${CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES:-}"
refresh_health_baseline="${CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE:-true}"
refresh_smoke_baseline="${CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE:-true}"
refresh_resolver_baseline="${CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE:-false}"
baseline_file="${CHAINPULSE_MIGRATION_BASELINE_FILE:-docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom}"
health_baseline_file="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE_FILE:-docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom}"
smoke_baseline_file="${CHAINPULSE_MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE_FILE:-docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom}"
resolver_baseline_file="${CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE:-docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom}"

case "$refresh_health_baseline" in
  true|false) ;;
  *)
    echo "[migration-kpi-preflight] invalid CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE: $refresh_health_baseline (expected true|false)"
    exit 1
    ;;
esac

case "$refresh_smoke_baseline" in
  true|false) ;;
  *)
    echo "[migration-kpi-preflight] invalid CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE: $refresh_smoke_baseline (expected true|false)"
    exit 1
    ;;
esac

case "$refresh_resolver_baseline" in
  true|false) ;;
  *)
    echo "[migration-kpi-preflight] invalid CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE: $refresh_resolver_baseline (expected true|false)"
    exit 1
    ;;
esac

if [[ -n "$baseline_scope" ]]; then
  if ! baseline_resolver_validate_scope "$baseline_scope"; then
    echo "[migration-kpi-preflight] invalid CHAINPULSE_BASELINE_UPDATE_SCOPE: $baseline_scope (expected kpi-only|health-only|dual)"
    exit 1
  fi
fi

if [[ -n "$baseline_changed_baselines" ]]; then
  if ! baseline_changed_baselines="$(baseline_resolver_normalize_changed_baselines "$baseline_changed_baselines")"; then
    echo "[migration-kpi-preflight] invalid CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES: $baseline_changed_baselines (expected subset of kpi,health,smoke,resolver)"
    exit 1
  fi
fi

resolved_scope="$(baseline_resolver_resolve_scope "$baseline_scope" "$refresh_health_baseline" "$refresh_smoke_baseline")"
resolved_changed_baselines="$(baseline_resolver_resolve_changed_baselines "$baseline_changed_baselines" "$refresh_health_baseline" "$refresh_smoke_baseline" "$refresh_resolver_baseline")"

mkdir -p "$(dirname "$output_file")"
generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
{
  echo "# Baseline Update Preflight"
  echo
  echo "- Generated At (UTC): $generated_at"
  echo
  echo "| Field | Value |"
  echo "|---|---|"
  echo "| ticket | ${ticket} |"
  echo "| owner | ${owner} |"
  echo "| scope | ${resolved_scope} |"
  echo "| changed_baselines | ${resolved_changed_baselines} |"
  echo "| rationale | ${rationale} |"
  echo "| refresh_health_baseline | ${refresh_health_baseline} |"
  echo "| refresh_smoke_baseline | ${refresh_smoke_baseline} |"
  echo "| refresh_resolver_baseline | ${refresh_resolver_baseline} |"
  echo
  echo "## Target Files"
  echo
  echo "- KPI baseline: \`${baseline_file}\`"
  if [[ "$refresh_health_baseline" == "true" ]]; then
    echo "- Health baseline: \`${health_baseline_file}\`"
  fi
  if [[ "$refresh_smoke_baseline" == "true" ]]; then
    echo "- Smoke baseline: \`${smoke_baseline_file}\`"
  fi
  if [[ "$refresh_resolver_baseline" == "true" ]]; then
    echo "- Resolver baseline: \`${resolver_baseline_file}\`"
  fi
  echo
  echo "## Suggested Changelog Entry"
  echo
  echo "- <UTC-ISO8601> | ticket=${ticket} | owner=${owner} | scope=${resolved_scope} | changed_baselines=${resolved_changed_baselines} | rationale=${rationale}"
} > "$output_file"

echo "[migration-kpi-preflight] generated:"
echo "  - $output_file"
echo "[migration-kpi-preflight] resolved scope=$resolved_scope changed_baselines=$resolved_changed_baselines"
