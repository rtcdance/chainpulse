#!/usr/bin/env bash

set -euo pipefail

changelog_file="${1:-docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md}"
enforce="${CHAINPULSE_ENFORCE_BASELINE_CHANGELOG_QUALITY:-true}"
ticket_pattern="${CHAINPULSE_MIGRATION_TICKET_PATTERN:-^[A-Z0-9]+-[A-Z0-9]+$}"
owner_allowlist="${CHAINPULSE_MIGRATION_OWNER_ALLOWLIST:-platform-team,sre-team,indexer-team}"
scope_allowlist="${CHAINPULSE_MIGRATION_CHANGELOG_SCOPE_ALLOWLIST:-kpi-only,health-only,dual}"
require_scope="${CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE:-true}"
changed_baselines_allowlist="${CHAINPULSE_MIGRATION_CHANGELOG_CHANGED_BASELINES_ALLOWLIST:-kpi,health,smoke,resolver}"
require_changed_baselines="${CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_CHANGED_BASELINES:-true}"
ticket_verify_mode="${CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE:-pattern}"
ticket_registry_source="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE:-file}"
ticket_registry_file="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_FILE:-docs/operations/MIGRATION_TICKET_REGISTRY.txt}"
ticket_registry_url="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_URL:-}"
ticket_verify_failure_mode="${CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE:-enforce}"
ticket_registry_timeout_seconds="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_TIMEOUT_SECONDS:-5}"
ticket_registry_http_slo_ms="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MS:-2000}"
ticket_registry_http_slo_mode="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE:-warn}"
registry_health_output="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_OUTPUT:-build/migration-governance/ticket-registry-health.prom}"
registry_health_md_output="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_MD_OUTPUT:-build/migration-governance/ticket-registry-health.md}"

if [[ "$enforce" != "true" ]]; then
  echo "[migration-kpi-changelog] quality check disabled"
  exit 0
fi

if [[ ! -f "$changelog_file" ]]; then
  echo "[migration-kpi-changelog] missing changelog file: $changelog_file"
  exit 1
fi

line_no=0
entry_count=0
registry_tmp="$(mktemp)"
registry_http_raw_tmp="$(mktemp)"
registry_initialized="false"
registry_check_status="not_applicable"
registry_failure_reason="none"
registry_checks_total=0
registry_fallback_events_total=0
registry_http_latency_ms=0
registry_http_latency_status="not_applicable"
registry_http_slo_violations_total=0

cleanup() {
  rm -f "$registry_tmp"
  rm -f "$registry_http_raw_tmp"
}
trap cleanup EXIT

fail_or_warn() {
  local message="$1"
  if [[ "$ticket_verify_failure_mode" == "warn" ]]; then
    echo "[migration-kpi-changelog] warning: $message"
    return 0
  fi
  echo "[migration-kpi-changelog] $message"
  return 1
}

owner_allowed() {
  local owner="$1"
  IFS=',' read -r -a owners <<< "$owner_allowlist"
  for allowed in "${owners[@]}"; do
    if [[ "$(echo "$allowed" | xargs)" == "$owner" ]]; then
      return 0
    fi
  done
  return 1
}

scope_allowed() {
  local scope="$1"
  IFS=',' read -r -a scopes <<< "$scope_allowlist"
  for allowed in "${scopes[@]}"; do
    if [[ "$(echo "$allowed" | xargs)" == "$scope" ]]; then
      return 0
    fi
  done
  return 1
}

changed_baselines_has() {
  local changed_baselines="$1"
  local expected="$2"
  IFS=',' read -r -a values <<< "$changed_baselines"
  for value in "${values[@]}"; do
    if [[ "$value" == "$expected" ]]; then
      return 0
    fi
  done
  return 1
}

normalize_changed_baselines() {
  local raw="$1"
  local has_kpi="false"
  local has_health="false"
  local has_smoke="false"
  local has_resolver="false"

  if [[ -z "$raw" ]]; then
    return 1
  fi

  IFS=',' read -r -a values <<< "$raw"
  for value in "${values[@]}"; do
    value="$(echo "$value" | xargs)"
    if [[ -z "$value" ]]; then
      continue
    fi

    case "$value" in
      kpi)
        has_kpi="true"
        ;;
      health)
        has_health="true"
        ;;
      smoke)
        has_smoke="true"
        ;;
      resolver)
        has_resolver="true"
        ;;
      *)
        return 1
        ;;
    esac
  done

  normalized=""
  if [[ "$has_kpi" == "true" ]]; then
    normalized="kpi"
  fi
  if [[ "$has_health" == "true" ]]; then
    if [[ -n "$normalized" ]]; then
      normalized="${normalized},health"
    else
      normalized="health"
    fi
  fi
  if [[ "$has_smoke" == "true" ]]; then
    if [[ -n "$normalized" ]]; then
      normalized="${normalized},smoke"
    else
      normalized="smoke"
    fi
  fi
  if [[ "$has_resolver" == "true" ]]; then
    if [[ -n "$normalized" ]]; then
      normalized="${normalized},resolver"
    else
      normalized="resolver"
    fi
  fi

  if [[ -z "$normalized" ]]; then
    return 1
  fi

  echo "$normalized"
}

changed_baselines_scope_compatible() {
  local scope="$1"
  local changed_baselines="$2"

  local has_kpi="false"
  local has_health="false"
  local has_smoke="false"
  local has_resolver="false"
  if changed_baselines_has "$changed_baselines" "kpi"; then
    has_kpi="true"
  fi
  if changed_baselines_has "$changed_baselines" "health"; then
    has_health="true"
  fi
  if changed_baselines_has "$changed_baselines" "smoke"; then
    has_smoke="true"
  fi
  if changed_baselines_has "$changed_baselines" "resolver"; then
    has_resolver="true"
  fi

  # Resolver baseline tags are orthogonal to scope semantics.
  if [[ "$has_kpi" != "true" && "$has_health" != "true" && "$has_smoke" != "true" && "$has_resolver" == "true" ]]; then
    return 0
  fi

  case "$scope" in
    kpi-only)
      [[ "$has_kpi" == "true" && "$has_health" != "true" && "$has_smoke" != "true" ]]
      ;;
    health-only)
      [[ "$has_kpi" != "true" && ("$has_health" == "true" || "$has_smoke" == "true") ]]
      ;;
    dual)
      [[ "$has_kpi" == "true" && ("$has_health" == "true" || "$has_smoke" == "true") ]]
      ;;
    *)
      return 1
      ;;
  esac
}

load_ticket_registry() {
  if [[ "$registry_initialized" == "true" ]]; then
    return 0
  fi
  registry_checks_total=$((registry_checks_total + 1))

  case "$ticket_registry_source" in
    file)
      if [[ ! -f "$ticket_registry_file" ]]; then
        registry_check_status="failure"
        registry_failure_reason="missing_registry_file"
        if fail_or_warn "ticket registry file not found: $ticket_registry_file"; then
          registry_fallback_events_total=$((registry_fallback_events_total + 1))
        else
          return 1
        fi
        : > "$registry_tmp"
      else
        registry_check_status="success"
        grep -Ev '^[[:space:]]*#|^[[:space:]]*$' "$ticket_registry_file" | sed 's/[[:space:]]*$//' | sort -u > "$registry_tmp"
      fi
      ;;
    http)
      if [[ -z "$ticket_registry_url" ]]; then
        registry_check_status="failure"
        registry_failure_reason="missing_registry_url"
        if fail_or_warn "ticket registry source=http requires CHAINPULSE_MIGRATION_TICKET_REGISTRY_URL"; then
          registry_fallback_events_total=$((registry_fallback_events_total + 1))
        else
          return 1
        fi
        : > "$registry_tmp"
      elif ! latency_seconds="$(curl -fsSL --max-time "$ticket_registry_timeout_seconds" --output "$registry_http_raw_tmp" --write-out '%{time_total}' "$ticket_registry_url")"; then
        registry_check_status="failure"
        registry_failure_reason="registry_fetch_failed"
        registry_http_latency_status="fetch_failed"
        if fail_or_warn "failed to fetch ticket registry from URL: $ticket_registry_url"; then
          registry_fallback_events_total=$((registry_fallback_events_total + 1))
        else
          return 1
        fi
        : > "$registry_tmp"
      else
        registry_check_status="success"
        grep -Ev '^[[:space:]]*#|^[[:space:]]*$' "$registry_http_raw_tmp" | sed 's/[[:space:]]*$//' | sort -u > "$registry_tmp"
        registry_http_latency_ms="$(awk -v s="$latency_seconds" 'BEGIN { printf "%.0f", s * 1000 }')"
        registry_http_latency_status="within_slo"

        if [[ "$ticket_registry_http_slo_mode" != "off" && "$registry_http_latency_ms" -gt "$ticket_registry_http_slo_ms" ]]; then
          registry_http_latency_status="above_slo"
          registry_http_slo_violations_total=$((registry_http_slo_violations_total + 1))

          if [[ "$ticket_registry_http_slo_mode" == "warn" ]]; then
            echo "[migration-kpi-changelog] warning: ticket registry http latency above slo (${registry_http_latency_ms}ms > ${ticket_registry_http_slo_ms}ms)"
          else
            echo "[migration-kpi-changelog] ticket registry http latency above slo (${registry_http_latency_ms}ms > ${ticket_registry_http_slo_ms}ms)"
            return 1
          fi
        fi
      fi
      ;;
    *)
      echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE: $ticket_registry_source (expected file|http)"
      return 1
      ;;
  esac

  registry_initialized="true"
  return 0
}

verify_ticket() {
  local ticket="$1"
  local line="$2"

  case "$ticket_verify_mode" in
    pattern|both)
      if [[ ! "$ticket" =~ $ticket_pattern ]]; then
        fail_or_warn "invalid ticket at line $line: '$ticket' (required pattern: $ticket_pattern)" || return 1
      fi
      ;;
  esac

  case "$ticket_verify_mode" in
    registry|both)
      load_ticket_registry || return 1
      if [[ -s "$registry_tmp" ]] && ! grep -Fxq "$ticket" "$registry_tmp"; then
        fail_or_warn "ticket not found in registry at line $line: '$ticket'" || return 1
      fi
      ;;
  esac

  return 0
}

case "$ticket_verify_mode" in
  pattern|registry|both)
    ;;
  *)
    echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE: $ticket_verify_mode (expected pattern|registry|both)"
    exit 1
    ;;
esac

case "$ticket_verify_failure_mode" in
  enforce|warn)
    ;;
  *)
    echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE: $ticket_verify_failure_mode (expected enforce|warn)"
    exit 1
    ;;
esac

case "$ticket_registry_http_slo_mode" in
  off|warn|enforce)
    ;;
  *)
    echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE: $ticket_registry_http_slo_mode (expected off|warn|enforce)"
    exit 1
    ;;
esac

if [[ ! "$ticket_registry_http_slo_ms" =~ ^[0-9]+$ ]]; then
  echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MS: $ticket_registry_http_slo_ms (expected non-negative integer milliseconds)"
  exit 1
fi

case "$require_scope" in
  true|false)
    ;;
  *)
    echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE: $require_scope (expected true|false)"
    exit 1
    ;;
esac

case "$require_changed_baselines" in
  true|false)
    ;;
  *)
    echo "[migration-kpi-changelog] invalid CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_CHANGED_BASELINES: $require_changed_baselines (expected true|false)"
    exit 1
    ;;
esac

while IFS= read -r line || [[ -n "$line" ]]; do
  line_no=$((line_no + 1))

  if [[ -z "$line" ]]; then
    continue
  fi
  if [[ "$line" == \#* ]]; then
    continue
  fi

  entry_count=$((entry_count + 1))
  if [[ "$require_scope" == "true" && "$require_changed_baselines" == "true" ]]; then
    if [[ ! "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ scope=([^[:space:]]+)\ \|\ changed_baselines=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      echo "[migration-kpi-changelog] invalid changelog entry format at line $line_no:"
      echo "  $line"
      echo "Expected format:"
      echo "  - <UTC-ISO8601> | ticket=<ID> | owner=<team-or-user> | scope=<kpi-only|health-only|dual> | changed_baselines=<kpi[,health][,smoke][,resolver]> | rationale=<text>"
      exit 1
    fi
    ticket="${BASH_REMATCH[2]}"
    owner="${BASH_REMATCH[3]}"
    scope="${BASH_REMATCH[4]}"
    changed_baselines_raw="${BASH_REMATCH[5]}"
    rationale="${BASH_REMATCH[6]}"
  elif [[ "$require_scope" == "true" && "$require_changed_baselines" != "true" ]]; then
    if [[ "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ scope=([^[:space:]]+)\ \|\ changed_baselines=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      ticket="${BASH_REMATCH[2]}"
      owner="${BASH_REMATCH[3]}"
      scope="${BASH_REMATCH[4]}"
      changed_baselines_raw="${BASH_REMATCH[5]}"
      rationale="${BASH_REMATCH[6]}"
    elif [[ "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ scope=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      ticket="${BASH_REMATCH[2]}"
      owner="${BASH_REMATCH[3]}"
      scope="${BASH_REMATCH[4]}"
      rationale="${BASH_REMATCH[5]}"
      case "$scope" in
        kpi-only) changed_baselines_raw="kpi" ;;
        health-only) changed_baselines_raw="health" ;;
        dual) changed_baselines_raw="kpi,health" ;;
      esac
    else
      echo "[migration-kpi-changelog] invalid changelog entry format at line $line_no:"
      echo "  $line"
      echo "Expected format:"
      echo "  - <UTC-ISO8601> | ticket=<ID> | owner=<team-or-user> | scope=<kpi-only|health-only|dual> | changed_baselines=<kpi[,health][,smoke][,resolver]> | rationale=<text>"
      exit 1
    fi
  else
    if [[ "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ scope=([^[:space:]]+)\ \|\ changed_baselines=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      ticket="${BASH_REMATCH[2]}"
      owner="${BASH_REMATCH[3]}"
      scope="${BASH_REMATCH[4]}"
      changed_baselines_raw="${BASH_REMATCH[5]}"
      rationale="${BASH_REMATCH[6]}"
    elif [[ "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ changed_baselines=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      ticket="${BASH_REMATCH[2]}"
      owner="${BASH_REMATCH[3]}"
      scope="kpi-only"
      changed_baselines_raw="${BASH_REMATCH[4]}"
      rationale="${BASH_REMATCH[5]}"
    elif [[ "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ scope=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      ticket="${BASH_REMATCH[2]}"
      owner="${BASH_REMATCH[3]}"
      scope="${BASH_REMATCH[4]}"
      rationale="${BASH_REMATCH[5]}"
      case "$scope" in
        kpi-only) changed_baselines_raw="kpi" ;;
        health-only) changed_baselines_raw="health" ;;
        dual) changed_baselines_raw="kpi,health" ;;
      esac
    elif [[ "$line" =~ ^-\ ([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\ \|\ ticket=([^[:space:]]+)\ \|\ owner=([^[:space:]]+)\ \|\ rationale=(.+)$ ]]; then
      ticket="${BASH_REMATCH[2]}"
      owner="${BASH_REMATCH[3]}"
      scope="kpi-only"
      changed_baselines_raw="kpi"
      rationale="${BASH_REMATCH[4]}"
    else
      echo "[migration-kpi-changelog] invalid changelog entry format at line $line_no:"
      echo "  $line"
      echo "Expected format:"
      echo "  - <UTC-ISO8601> | ticket=<ID> | owner=<team-or-user> | scope=<kpi-only|health-only|dual> | changed_baselines=<kpi[,health][,smoke][,resolver]> | rationale=<text>"
      echo "Legacy accepted only when both CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=false and CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_CHANGED_BASELINES=false"
      exit 1
    fi
  fi

  verify_ticket "$ticket" "$line_no" || exit 1

  if ! owner_allowed "$owner"; then
    echo "[migration-kpi-changelog] owner not in allowlist at line $line_no: '$owner'"
    echo "  allowlist: $owner_allowlist"
    exit 1
  fi

  if ! scope_allowed "$scope"; then
    echo "[migration-kpi-changelog] scope not in allowlist at line $line_no: '$scope'"
    echo "  allowlist: $scope_allowlist"
    exit 1
  fi

  if ! changed_baselines="$(normalize_changed_baselines "$changed_baselines_raw")"; then
    echo "[migration-kpi-changelog] invalid changed_baselines at line $line_no: '$changed_baselines_raw'"
    echo "  expected comma-separated subset of: $changed_baselines_allowlist"
    exit 1
  fi

  if ! changed_baselines_scope_compatible "$scope" "$changed_baselines"; then
    echo "[migration-kpi-changelog] scope and changed_baselines mismatch at line $line_no"
    echo "  scope: $scope"
    echo "  changed_baselines: $changed_baselines"
    exit 1
  fi

  if [[ -z "$rationale" || "$rationale" == "n/a" || "$rationale" == "N/A" ]]; then
    echo "[migration-kpi-changelog] rationale too weak at line $line_no: '$rationale'"
    exit 1
  fi
done < "$changelog_file"

if (( entry_count == 0 )); then
  echo "[migration-kpi-changelog] changelog has no entries: $changelog_file"
  exit 1
fi

mkdir -p "$(dirname "$registry_health_output")"
mkdir -p "$(dirname "$registry_health_md_output")"

{
  echo "# HELP chainpulse_migration_ticket_registry_checks_total Ticket registry checks by mode/source/status."
  echo "# TYPE chainpulse_migration_ticket_registry_checks_total counter"
  echo "chainpulse_migration_ticket_registry_checks_total{mode=\"${ticket_verify_mode}\",source=\"${ticket_registry_source}\",status=\"${registry_check_status}\",failure_mode=\"${ticket_verify_failure_mode}\"} ${registry_checks_total}"
  echo
  echo "# HELP chainpulse_migration_ticket_registry_fallback_events_total Ticket verification fallback events."
  echo "# TYPE chainpulse_migration_ticket_registry_fallback_events_total counter"
  echo "chainpulse_migration_ticket_registry_fallback_events_total{reason=\"${registry_failure_reason}\",failure_mode=\"${ticket_verify_failure_mode}\"} ${registry_fallback_events_total}"
  echo
  echo "# HELP chainpulse_migration_ticket_registry_http_latency_ms Ticket registry HTTP latency in milliseconds."
  echo "# TYPE chainpulse_migration_ticket_registry_http_latency_ms gauge"
  echo "chainpulse_migration_ticket_registry_http_latency_ms{source=\"${ticket_registry_source}\",status=\"${registry_http_latency_status}\",slo_mode=\"${ticket_registry_http_slo_mode}\"} ${registry_http_latency_ms}"
  echo
  echo "# HELP chainpulse_migration_ticket_registry_http_slo_violations_total Ticket registry HTTP SLO violations."
  echo "# TYPE chainpulse_migration_ticket_registry_http_slo_violations_total counter"
  echo "chainpulse_migration_ticket_registry_http_slo_violations_total{source=\"${ticket_registry_source}\",slo_mode=\"${ticket_registry_http_slo_mode}\"} ${registry_http_slo_violations_total}"
} > "$registry_health_output"

{
  echo "# Ticket Registry Health"
  echo
  echo "| Field | Value |"
  echo "|---|---|"
  echo "| verify_mode | ${ticket_verify_mode} |"
  echo "| registry_source | ${ticket_registry_source} |"
  echo "| registry_status | ${registry_check_status} |"
  echo "| failure_reason | ${registry_failure_reason} |"
  echo "| checks_total | ${registry_checks_total} |"
  echo "| fallback_events_total | ${registry_fallback_events_total} |"
  echo "| failure_mode | ${ticket_verify_failure_mode} |"
  echo "| http_latency_ms | ${registry_http_latency_ms} |"
  echo "| http_latency_status | ${registry_http_latency_status} |"
  echo "| http_slo_ms | ${ticket_registry_http_slo_ms} |"
  echo "| http_slo_mode | ${ticket_registry_http_slo_mode} |"
  echo "| http_slo_violations_total | ${registry_http_slo_violations_total} |"
} > "$registry_health_md_output"

echo "[migration-kpi-changelog] quality check passed (entries=$entry_count)"
echo "[migration-kpi-changelog] registry health output: $registry_health_output"
echo "[migration-kpi-changelog] registry health report: $registry_health_md_output"
