#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/governance_summary.sh"

log_prefix="[governance-overview-summary]"
smoke_report="${1:-build/migration-governance/baseline-scope-smoke.md}"
smoke_delta="${2:-build/migration-governance/baseline-scope-smoke-delta.md}"
resolver_report="${3:-build/migration-governance/baseline-resolver-test.md}"
resolver_delta="${4:-build/migration-governance/baseline-resolver-test-delta.md}"
registry_report="${5:-build/migration-governance/ticket-registry-health.md}"
registry_delta="${6:-build/migration-governance/ticket-registry-health-delta.md}"
owner_drift_report="${7:-build/migration-governance/migration-owner-drift-report.md}"
summary_output="${CHAINPULSE_GOVERNANCE_OVERVIEW_JOB_SUMMARY_OUTPUT:-${GITHUB_STEP_SUMMARY:-}}"

normalize_overview_level() {
  local raw_status="$1"
  local regression_status="$2"

  case "$raw_status" in
    fail|failed|unknown)
      echo "fail"
      return
      ;;
    not_applicable|n/a)
      echo "info"
      return
      ;;
  esac

  if [[ "$regression_status" != "none" && "$regression_status" != "n/a" ]]; then
    echo "warn"
    return
  fi

  case "$raw_status" in
    pass|healthy|allowed|ok|none)
      echo "ok"
      ;;
    *)
      echo "warn"
      ;;
  esac
}

select_overview_action() {
  local level="$1"
  local source_name="$2"
  local detail_name="$3"

  case "$level" in
    fail)
      if [[ "$detail_name" != "n/a" ]]; then
        echo "review \`$detail_name\`"
      else
        echo "check \`$source_name\`"
      fi
      ;;
    warn)
      if [[ "$detail_name" != "n/a" ]]; then
        echo "inspect \`$detail_name\`"
      else
        echo "review \`$source_name\`"
      fi
      ;;
    info)
      echo "monitor only"
      ;;
    *)
      echo "verify \`$source_name\` if needed"
      ;;
  esac
}

compute_overall_health() {
  local levels=("$@")
  local level

  for level in "${levels[@]}"; do
    if [[ "$level" == "fail" ]]; then
      echo "fail"
      return
    fi
  done

  for level in "${levels[@]}"; do
    if [[ "$level" == "warn" ]]; then
      echo "warn"
      return
    fi
  done

  for level in "${levels[@]}"; do
    if [[ "$level" == "ok" ]]; then
      echo "ok"
      return
    fi
  done

  echo "info"
}

compute_overall_hint() {
  local overall_health="$1"

  case "$overall_health" in
    fail)
      echo "address failures first"
      ;;
    warn)
      echo "investigate warnings"
      ;;
    ok)
      echo "stable"
      ;;
    *)
      echo "informational only"
      ;;
  esac
}

compute_overall_focus() {
  local smoke_level="$1"
  local resolver_level="$2"
  local registry_level="$3"
  local owner_level="$4"

  if [[ "$smoke_level" == "fail" ]]; then
    echo "smoke"
    return
  fi
  if [[ "$resolver_level" == "fail" ]]; then
    echo "resolver"
    return
  fi
  if [[ "$registry_level" == "fail" ]]; then
    echo "registry-health"
    return
  fi
  if [[ "$owner_level" == "fail" ]]; then
    echo "owner-drift"
    return
  fi

  if [[ "$smoke_level" == "warn" ]]; then
    echo "smoke"
    return
  fi
  if [[ "$resolver_level" == "warn" ]]; then
    echo "resolver"
    return
  fi
  if [[ "$registry_level" == "warn" ]]; then
    echo "registry-health"
    return
  fi
  if [[ "$owner_level" == "warn" ]]; then
    echo "owner-drift"
    return
  fi

  echo "none"
}

compute_overall_next_step() {
  local overall_health="$1"
  local overall_focus="$2"
  local overall_route="$3"
  local overall_playbook="$4"

  case "$overall_health" in
    fail)
      echo "address \`$overall_focus\` now: engage $overall_route and open $overall_playbook first"
      ;;
    warn)
      echo "review drift on \`$overall_focus\`: inspect $overall_playbook with $overall_route"
      ;;
    ok)
      echo "no action needed"
      ;;
    *)
      echo "monitor only"
      ;;
  esac
}

select_overview_route() {
  local surface="$1"

  case "$surface" in
    smoke)
      echo "platform-team / policy-contract"
      ;;
    resolver)
      echo "platform-team / resolver-governance"
      ;;
    registry-health)
      echo "sre-team / registry-health"
      ;;
    owner-drift)
      echo "platform-team / migration-owners"
      ;;
    *)
      echo "none"
      ;;
  esac
}

select_overview_playbook() {
  local surface="$1"

  case "$surface" in
    smoke)
      echo "\`baseline-scope-smoke.md\`"
      ;;
    resolver)
      echo "\`baseline-resolver-test.md\`"
      ;;
    registry-health)
      echo "\`ticket-registry-health.md\`"
      ;;
    owner-drift)
      echo "\`migration-owner-drift-report.md\`"
      ;;
    *)
      echo "none"
      ;;
  esac
}

governance_summary_assert_inputs "$log_prefix" "$smoke_report" "$summary_output"
governance_summary_assert_inputs "$log_prefix" "$resolver_report" "$summary_output"
governance_summary_assert_inputs "$log_prefix" "$smoke_delta" "$summary_output"
governance_summary_assert_inputs "$log_prefix" "$resolver_delta" "$summary_output"
governance_summary_assert_inputs "$log_prefix" "$registry_report" "$summary_output"
governance_summary_assert_inputs "$log_prefix" "$registry_delta" "$summary_output"
governance_summary_assert_inputs "$log_prefix" "$owner_drift_report" "$summary_output"

smoke_status="$(governance_summary_extract_status "$smoke_report")"
smoke_total="$(governance_summary_trim "$(governance_summary_extract_table_value "$smoke_report" "total_cases")")"
smoke_failed="$(governance_summary_trim "$(governance_summary_extract_table_value "$smoke_report" "failed_cases")")"
smoke_delta_status="$(governance_summary_extract_status "$smoke_delta")"

resolver_status="$(governance_summary_extract_status "$resolver_report")"
resolver_total="$(governance_summary_trim "$(governance_summary_extract_table_value "$resolver_report" "total")")"
resolver_failed="$(governance_summary_trim "$(governance_summary_extract_table_value "$resolver_report" "failed")")"
resolver_delta_status="$(governance_summary_extract_status "$resolver_delta")"

registry_status="$(governance_summary_trim "$(governance_summary_extract_table_value "$registry_report" "registry_status")")"
registry_fallbacks="$(governance_summary_trim "$(governance_summary_extract_table_value "$registry_report" "fallback_events_total")")"
registry_checks="$(governance_summary_trim "$(governance_summary_extract_table_value "$registry_report" "checks_total")")"
registry_delta_status="$(governance_summary_extract_status "$registry_delta")"

owner_unknown="$(governance_summary_trim "$(governance_summary_extract_table_value "$owner_drift_report" "Unknown Owners")")"
owner_total="$(governance_summary_trim "$(governance_summary_extract_table_value "$owner_drift_report" "Distinct Owners")")"
owner_status="pass"
if [[ "$owner_total" == "0" ]]; then
  owner_status="not_applicable"
elif [[ "$owner_unknown" != "0" ]]; then
  owner_status="fail"
fi

smoke_level="$(normalize_overview_level "$smoke_status" "$smoke_delta_status")"
resolver_level="$(normalize_overview_level "$resolver_status" "$resolver_delta_status")"
registry_level="$(normalize_overview_level "$registry_status" "$registry_delta_status")"
owner_level="$(normalize_overview_level "$owner_status" "n/a")"
smoke_source_name="$(basename "$smoke_report")"
smoke_detail_name="$(basename "$smoke_delta")"
resolver_source_name="$(basename "$resolver_report")"
resolver_detail_name="$(basename "$resolver_delta")"
registry_source_name="$(basename "$registry_report")"
registry_detail_name="$(basename "$registry_delta")"
owner_source_name="$(basename "$owner_drift_report")"
owner_detail_name="migration-owner-drift-report.tsv"
smoke_action="$(select_overview_action "$smoke_level" "$smoke_source_name" "$smoke_detail_name")"
resolver_action="$(select_overview_action "$resolver_level" "$resolver_source_name" "$resolver_detail_name")"
registry_action="$(select_overview_action "$registry_level" "$registry_source_name" "$registry_detail_name")"
owner_action="$(select_overview_action "$owner_level" "$owner_source_name" "$owner_detail_name")"
overall_health="$(compute_overall_health "$smoke_level" "$resolver_level" "$registry_level" "$owner_level")"
overall_hint="$(compute_overall_hint "$overall_health")"
overall_focus="$(compute_overall_focus "$smoke_level" "$resolver_level" "$registry_level" "$owner_level")"
smoke_route="$(select_overview_route "smoke")"
resolver_route="$(select_overview_route "resolver")"
registry_route="$(select_overview_route "registry-health")"
owner_route="$(select_overview_route "owner-drift")"
overall_route="$(select_overview_route "$overall_focus")"
smoke_playbook="$(select_overview_playbook "smoke")"
resolver_playbook="$(select_overview_playbook "resolver")"
registry_playbook="$(select_overview_playbook "registry-health")"
owner_playbook="$(select_overview_playbook "owner-drift")"
overall_playbook="$(select_overview_playbook "$overall_focus")"
overall_next_step="$(compute_overall_next_step "$overall_health" "$overall_focus" "$overall_route" "$overall_playbook")"

{
  echo "## Governance Overview"
  echo
  echo "- Overall Health: \`$overall_health\`"
  echo "- Overall Hint: $overall_hint"
  echo "- Overall Focus: \`$overall_focus\`"
  echo "- Overall Route: $overall_route"
  echo "- Overall Playbook: $overall_playbook"
  echo "- Overall Next Step: $overall_next_step"
  echo "- Surfaces: 4"
  echo
  echo "| Surface | Level | Status | Failed | Total | Regression | Source | Details | Route | Playbook | Action |"
  echo "|---|---|---|---:|---:|---|---|---|---|---|---|"
  echo "| smoke | \`$smoke_level\` | \`$smoke_status\` | $smoke_failed | $smoke_total | \`$smoke_delta_status\` | \`$smoke_source_name\` | \`$smoke_detail_name\` | $smoke_route | $smoke_playbook | $smoke_action |"
  echo "| resolver | \`$resolver_level\` | \`$resolver_status\` | $resolver_failed | $resolver_total | \`$resolver_delta_status\` | \`$resolver_source_name\` | \`$resolver_detail_name\` | $resolver_route | $resolver_playbook | $resolver_action |"
  echo "| registry-health | \`$registry_level\` | \`$registry_status\` | $registry_fallbacks | $registry_checks | \`$registry_delta_status\` | \`$registry_source_name\` | \`$registry_detail_name\` | $registry_route | $registry_playbook | $registry_action |"
  echo "| owner-drift | \`$owner_level\` | \`$owner_status\` | $owner_unknown | $owner_total | \`n/a\` | \`$owner_source_name\` | \`$owner_detail_name\` | $owner_route | $owner_playbook | $owner_action |"
  echo
  echo "Legend:"
  echo "- \`ok\`: healthy/expected with no active regression signal"
  echo "- \`warn\`: usable but attention needed due to drift or soft degradation"
  echo "- \`fail\`: active governance risk or policy violation"
  echo "- \`info\`: informational/non-applicable state"
  echo "- Route: most likely report domain or operator group to engage first"
  echo "- Playbook: first local report or document to open for that surface"
  echo "- Overall Next Step: severity-aware aggregate remediation hint"
  echo
} >> "$summary_output"

echo "$log_prefix appended overview to: $summary_output"
