#!/usr/bin/env bash

set -euo pipefail

log_prefix="[governance-overview-summary-test]"
script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/append-governance-overview-summary.sh"
total=0
passed=0
failed=0

run_overview() {
  local smoke_report="$1"
  local smoke_delta="$2"
  local resolver_report="$3"
  local resolver_delta="$4"
  local registry_report="$5"
  local registry_delta="$6"
  local owner_drift_report="$7"
  local summary_file="$8"

  CHAINPULSE_GOVERNANCE_OVERVIEW_JOB_SUMMARY_OUTPUT="$summary_file" \
    "$script_path" "$smoke_report" "$smoke_delta" "$resolver_report" "$resolver_delta" "$registry_report" "$registry_delta" "$owner_drift_report" >/dev/null
}

write_smoke_fixture() {
  local file="$1"
  local status="$2"
  local total="$3"
  local failed_cases="$4"
  printf '%s\n' \
    '# Baseline Scope Smoke' \
    '' \
    "- Status: \`$status\`" \
    '' \
    '| Field | Value |' \
    '|---|---:|' \
    "| total_cases | $total |" \
    "| failed_cases | $failed_cases |" \
    > "$file"
}

write_delta_status_fixture() {
  local title="$1"
  local file="$2"
  local status="$3"
  printf '%s\n' \
    "# $title" \
    '' \
    "- Status: \`$status\`" \
    > "$file"
}

write_resolver_fixture() {
  local file="$1"
  local status="$2"
  local total="$3"
  local failed_cases="$4"
  printf '%s\n' \
    '# Baseline Resolver Tests' \
    '' \
    "- Status: \`$status\`" \
    '' \
    '| Field | Value |' \
    '|---|---:|' \
    "| total | $total |" \
    "| failed | $failed_cases |" \
    > "$file"
}

write_registry_fixture() {
  local file="$1"
  local status="$2"
  local checks_total="$3"
  local fallback_events_total="$4"
  printf '%s\n' \
    '# Ticket Registry Health' \
    '' \
    '| Field | Value |' \
    '|---|---|' \
    "| registry_status | $status |" \
    "| checks_total | $checks_total |" \
    "| fallback_events_total | $fallback_events_total |" \
    > "$file"
}

write_owner_fixture() {
  local file="$1"
  local distinct_owners="$2"
  local unknown_owners="$3"
  printf '%s\n' \
    '# Migration Owner Drift Report' \
    '' \
    '## Summary' \
    '' \
    '| Metric | Value |' \
    '|---|---:|' \
    "| Distinct Owners | $distinct_owners |" \
    "| Unknown Owners | $unknown_owners |" \
    > "$file"
}

fail_setup_descriptor() {
  local message="$1"
  echo "$log_prefix invalid setup descriptor: $message" >&2
  exit 1
}

fail_aggregate_descriptor() {
  local message="$1"
  echo "$log_prefix invalid aggregate descriptor: $message" >&2
  exit 1
}

fail_row_descriptor() {
  local message="$1"
  echo "$log_prefix invalid row descriptor: $message" >&2
  exit 1
}

assert_setup_descriptor_fields() {
  local kind="$1"
  local expected_count="$2"
  shift 2
  local actual_count="$#"

  if (( actual_count != expected_count )); then
    fail_setup_descriptor "$kind expected $expected_count fields but got $actual_count"
  fi
}

assert_aggregate_descriptor_fields() {
  local expected_count="$1"
  shift
  local actual_count="$#"

  if (( actual_count != expected_count )); then
    fail_aggregate_descriptor "expected $expected_count fields but got $actual_count"
  fi
}

assert_row_descriptor_fields() {
  local expected_count="$1"
  shift
  local actual_count="$#"

  if (( actual_count != expected_count )); then
    fail_row_descriptor "expected $expected_count fields but got $actual_count"
  fi
}

split_descriptor_fields() {
  local raw="$1"
  SPLIT_DESCRIPTOR_FIELDS_RESULT=()

  while [[ "$raw" == *"||"* ]]; do
    SPLIT_DESCRIPTOR_FIELDS_RESULT+=("${raw%%||*}")
    raw="${raw#*||}"
  done
  SPLIT_DESCRIPTOR_FIELDS_RESULT+=("$raw")
}

parse_setup_descriptor() {
  local descriptor="$1"
  local kind="${descriptor%%||*}"
  local rest="${descriptor#*||}"

  split_descriptor_fields "$rest"
  PARSED_DESCRIPTOR_KIND="$kind"
  PARSED_DESCRIPTOR_FIELDS=("${SPLIT_DESCRIPTOR_FIELDS_RESULT[@]}")
}

parse_aggregate_descriptor() {
  local descriptor="$1"

  split_descriptor_fields "$descriptor"
  PARSED_DESCRIPTOR_FIELDS=("${SPLIT_DESCRIPTOR_FIELDS_RESULT[@]}")
  assert_aggregate_descriptor_fields 7 "${PARSED_DESCRIPTOR_FIELDS[@]}"
}

parse_row_descriptor() {
  local descriptor="$1"

  split_descriptor_fields "$descriptor"
  PARSED_DESCRIPTOR_FIELDS=("${SPLIT_DESCRIPTOR_FIELDS_RESULT[@]}")
  assert_row_descriptor_fields 2 "${PARSED_DESCRIPTOR_FIELDS[@]}"
}

apply_setup_descriptors() {
  local descriptor
  local fields=()

  for descriptor in "$@"; do
    parse_setup_descriptor "$descriptor"
    fields=("${PARSED_DESCRIPTOR_FIELDS[@]}")
    case "$PARSED_DESCRIPTOR_KIND" in
      smoke)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 3 "${fields[@]}"
        write_smoke_fixture "$smoke_report" "${fields[0]}" "${fields[1]}" "${fields[2]}"
        ;;
      smoke-delta)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 1 "${fields[@]}"
        write_delta_status_fixture "Baseline Scope Smoke Delta" "$smoke_delta" "${fields[0]}"
        ;;
      resolver)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 3 "${fields[@]}"
        write_resolver_fixture "$resolver_report" "${fields[0]}" "${fields[1]}" "${fields[2]}"
        ;;
      resolver-delta)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 1 "${fields[@]}"
        write_delta_status_fixture "Baseline Resolver Test Delta" "$resolver_delta" "${fields[0]}"
        ;;
      registry)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 3 "${fields[@]}"
        write_registry_fixture "$registry_report" "${fields[0]}" "${fields[1]}" "${fields[2]}"
        ;;
      registry-delta)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 1 "${fields[@]}"
        write_delta_status_fixture "Ticket Registry Health Delta" "$registry_delta" "${fields[0]}"
        ;;
      owner)
        assert_setup_descriptor_fields "$PARSED_DESCRIPTOR_KIND" 2 "${fields[@]}"
        write_owner_fixture "$owner_drift_report" "${fields[0]}" "${fields[1]}"
        ;;
      *)
        fail_setup_descriptor "unknown kind '$PARSED_DESCRIPTOR_KIND'"
        ;;
    esac
  done
}

assert_contains() {
  local name="$1"
  local file="$2"
  local pattern="$3"
  total=$((total + 1))
  if grep -Fq -- "$pattern" "$file"; then
    passed=$((passed + 1))
    echo "$log_prefix pass: $name"
  else
    failed=$((failed + 1))
    echo "$log_prefix fail: $name"
    echo "$log_prefix   missing: $pattern"
    return 0
  fi
}

assert_command_fails_with() {
  local name="$1"
  local pattern="$2"
  shift 2
  local output

  total=$((total + 1))
  if output="$("$@" 2>&1)"; then
    failed=$((failed + 1))
    echo "$log_prefix fail: $name"
    echo "$log_prefix   command unexpectedly succeeded"
    return 0
  fi

  if grep -Fq -- "$pattern" <<<"$output"; then
    passed=$((passed + 1))
    echo "$log_prefix pass: $name"
  else
    failed=$((failed + 1))
    echo "$log_prefix fail: $name"
    echo "$log_prefix   missing failure pattern: $pattern"
    echo "$log_prefix   output: $output"
  fi
}

assert_aggregate_block() {
  local prefix="$1"
  local file="$2"
  local health="$3"
  local hint="$4"
  local focus="$5"
  local route="$6"
  local playbook="$7"
  local next_step="$8"

  assert_contains "${prefix}-overall-health" "$file" "- Overall Health: \`$health\`"
  assert_contains "${prefix}-overall-hint" "$file" "- Overall Hint: $hint"
  assert_contains "${prefix}-overall-focus" "$file" "- Overall Focus: \`$focus\`"
  assert_contains "${prefix}-overall-route" "$file" "- Overall Route: $route"
  assert_contains "${prefix}-overall-playbook" "$file" "- Overall Playbook: $playbook"
  assert_contains "${prefix}-overall-next-step" "$file" "- Overall Next Step: $next_step"
}

assert_aggregate_scenario() {
  local file="$1"
  local scenario="$2"
  parse_aggregate_descriptor "$scenario"

  assert_aggregate_block \
    "${PARSED_DESCRIPTOR_FIELDS[0]}" \
    "$file" \
    "${PARSED_DESCRIPTOR_FIELDS[1]}" \
    "${PARSED_DESCRIPTOR_FIELDS[2]}" \
    "${PARSED_DESCRIPTOR_FIELDS[3]}" \
    "${PARSED_DESCRIPTOR_FIELDS[4]}" \
    "${PARSED_DESCRIPTOR_FIELDS[5]}" \
    "${PARSED_DESCRIPTOR_FIELDS[6]}"
}

assert_overview_row() {
  local name="$1"
  local file="$2"
  local row="$3"

  assert_contains "$name" "$file" "$row"
}

assert_overview_rows_from_descriptors() {
  local file="$1"
  shift
  local rows=("$@")
  local row

  for row in "${rows[@]}"; do
    parse_row_descriptor "$row"
    assert_overview_row "${PARSED_DESCRIPTOR_FIELDS[0]}" "$file" "${PARSED_DESCRIPTOR_FIELDS[1]}"
  done
}

assert_primary_rows() {
  local file="$1"
  assert_overview_rows_from_descriptors \
    "$file" \
    "smoke-row|| smoke | \`ok\` | \`pass\` | 0 | 22 | \`none\` | \`smoke.md\` | \`smoke-delta.md\` | platform-team / policy-contract | \`baseline-scope-smoke.md\` | verify \`smoke.md\` if needed |" \
    "resolver-row|| resolver | \`fail\` | \`fail\` | 2 | 12 | \`failed_cases_increased\` | \`resolver.md\` | \`resolver-delta.md\` | platform-team / resolver-governance | \`baseline-resolver-test.md\` | review \`resolver-delta.md\` |" \
    "registry-row|| registry-health | \`warn\` | \`healthy\` | 1 | 5 | \`fallback_events_increased\` | \`registry.md\` | \`registry-delta.md\` | sre-team / registry-health | \`ticket-registry-health.md\` | inspect \`registry-delta.md\` |" \
    "owner-row|| owner-drift | \`fail\` | \`fail\` | 1 | 3 | \`n/a\` | \`owner-drift.md\` | \`migration-owner-drift-report.tsv\` | platform-team / migration-owners | \`migration-owner-drift-report.md\` | review \`migration-owner-drift-report.tsv\` |"
}

assert_warn_rows() {
  local file="$1"
  assert_overview_rows_from_descriptors \
    "$file" \
    "warn-resolver-row|| resolver | \`warn\` | \`pass\` | 0 | 12 | \`failed_cases_increased\` | \`resolver.md\` | \`resolver-delta.md\` | platform-team / resolver-governance | \`baseline-resolver-test.md\` | inspect \`resolver-delta.md\` |"
}

assert_info_rows() {
  local file="$1"
  assert_overview_rows_from_descriptors \
    "$file" \
    "info-owner-row|| owner-drift | \`info\` | \`not_applicable\` | 0 | 0 | \`n/a\` | \`owner-drift.md\` | \`migration-owner-drift-report.tsv\` | platform-team / migration-owners | \`migration-owner-drift-report.md\` | monitor only |"
}

assert_ok_rows() {
  local file="$1"
  assert_overview_rows_from_descriptors \
    "$file" \
    "ok-registry-row|| registry-health | \`ok\` | \`healthy\` | 0 | 5 | \`none\` | \`registry.md\` | \`registry-delta.md\` | sre-team / registry-health | \`ticket-registry-health.md\` | verify \`registry.md\` if needed |"
}

assert_legend_block() {
  local file="$1"

  assert_contains "legend-header" "$file" "Legend:"
  assert_contains "legend-ok" "$file" "- \`ok\`: healthy/expected with no active regression signal"
  assert_contains "legend-warn" "$file" "- \`warn\`: usable but attention needed due to drift or soft degradation"
  assert_contains "legend-route" "$file" "- Route: most likely report domain or operator group to engage first"
  assert_contains "legend-playbook" "$file" "- Playbook: first local report or document to open for that surface"
  assert_contains "legend-next-step" "$file" "- Overall Next Step: severity-aware aggregate remediation hint"
}

assert_aggregate_scenarios() {
  local file="$1"
  shift
  local scenario

  for scenario in "$@"; do
    assert_aggregate_scenario "$file" "$scenario"
  done
}

setup_primary_overview_state() {
  apply_setup_descriptors \
    "smoke||pass||22||0" \
    "smoke-delta||none" \
    "resolver||fail||12||2" \
    "resolver-delta||failed_cases_increased" \
    "registry||healthy||5||1" \
    "registry-delta||fallback_events_increased" \
    "owner||3||1"
}

setup_warn_overview_state() {
  apply_setup_descriptors \
    "resolver||pass||12||0" \
    "owner||3||0"
}

setup_info_overview_state() {
  apply_setup_descriptors \
    "resolver||not_applicable||0||0" \
    "smoke||not_applicable||0||0" \
    "smoke-delta||n/a" \
    "resolver-delta||n/a" \
    "registry||not_applicable||0||0" \
    "registry-delta||n/a" \
    "owner||0||0"
}

setup_ok_overview_state() {
  apply_setup_descriptors \
    "smoke||pass||22||0" \
    "smoke-delta||none" \
    "resolver||pass||12||0" \
    "resolver-delta||none" \
    "registry||healthy||5||0" \
    "registry-delta||none" \
    "owner||3||0"
}

run_primary_overview_state() {
  run_overview "$smoke_report" "$smoke_delta" "$resolver_report" "$resolver_delta" "$registry_report" "$registry_delta" "$owner_drift_report" "$summary_file"
}

run_warn_overview_state() {
  run_overview "$smoke_report" "$smoke_delta" "$resolver_report" "$resolver_delta" "$registry_report" "$registry_delta" "$owner_drift_report" "$warn_summary_file"
}

run_info_overview_state() {
  run_overview "$smoke_report" "$smoke_delta" "$resolver_report" "$resolver_delta" "$registry_report" "$registry_delta" "$owner_drift_report" "$info_summary_file"
}

run_ok_overview_state() {
  run_overview "$smoke_report" "$smoke_delta" "$resolver_report" "$resolver_delta" "$registry_report" "$registry_delta" "$owner_drift_report" "$ok_summary_file"
}

assert_primary_overview_state() {
  assert_contains "header" "$summary_file" "## Governance Overview"
  assert_aggregate_scenarios \
    "$summary_file" \
    "overall||fail||address failures first||resolver||platform-team / resolver-governance||\`baseline-resolver-test.md\`||address \`resolver\` now: engage platform-team / resolver-governance and open \`baseline-resolver-test.md\` first"
  assert_contains "surface-count" "$summary_file" "- Surfaces: 4"
  assert_primary_rows "$summary_file"
  assert_legend_block "$summary_file"
}

assert_warn_overview_state() {
  assert_aggregate_scenarios \
    "$warn_summary_file" \
    "warn||warn||investigate warnings||resolver||platform-team / resolver-governance||\`baseline-resolver-test.md\`||review drift on \`resolver\`: inspect \`baseline-resolver-test.md\` with platform-team / resolver-governance"
  assert_warn_rows "$warn_summary_file"
}

assert_info_overview_state() {
  assert_aggregate_scenarios \
    "$info_summary_file" \
    "info||info||informational only||none||none||none||monitor only"
  assert_info_rows "$info_summary_file"
}

assert_ok_overview_state() {
  assert_aggregate_scenarios \
    "$ok_summary_file" \
    "ok||ok||stable||none||none||none||no action needed"
  assert_ok_rows "$ok_summary_file"
}

run_invalid_setup_descriptor_check() {
  apply_setup_descriptors "unknown||value"
}

run_invalid_aggregate_descriptor_check() {
  assert_aggregate_scenario "$summary_file" "broken||warn||only-two-fields"
}

run_invalid_row_descriptor_check() {
  assert_overview_rows_from_descriptors "$summary_file" "broken-row-only"
}

run_invalid_aggregate_parser_check() {
  parse_aggregate_descriptor "broken||warn||only-two-fields"
}

run_invalid_row_parser_check() {
  parse_row_descriptor "broken-row-only"
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

smoke_report="$tmp_dir/smoke.md"
smoke_delta="$tmp_dir/smoke-delta.md"
resolver_report="$tmp_dir/resolver.md"
resolver_delta="$tmp_dir/resolver-delta.md"
registry_report="$tmp_dir/registry.md"
registry_delta="$tmp_dir/registry-delta.md"
owner_drift_report="$tmp_dir/owner-drift.md"
summary_file="$tmp_dir/summary.md"
warn_summary_file="$tmp_dir/warn-summary.md"
info_summary_file="$tmp_dir/info-summary.md"
ok_summary_file="$tmp_dir/ok-summary.md"

setup_primary_overview_state
run_primary_overview_state
assert_primary_overview_state

setup_warn_overview_state
run_warn_overview_state
assert_warn_overview_state

setup_info_overview_state
run_info_overview_state
assert_info_overview_state

setup_ok_overview_state
run_ok_overview_state
assert_ok_overview_state

assert_command_fails_with \
  "invalid-setup-descriptor" \
  "invalid setup descriptor: unknown kind 'unknown'" \
  run_invalid_setup_descriptor_check

assert_command_fails_with \
  "invalid-aggregate-descriptor" \
  "invalid aggregate descriptor: expected 7 fields but got 3" \
  run_invalid_aggregate_descriptor_check

assert_command_fails_with \
  "invalid-row-descriptor" \
  "invalid row descriptor: expected 2 fields but got 1" \
  run_invalid_row_descriptor_check

assert_command_fails_with \
  "invalid-aggregate-parser" \
  "invalid aggregate descriptor: expected 7 fields but got 3" \
  run_invalid_aggregate_parser_check

assert_command_fails_with \
  "invalid-row-parser" \
  "invalid row descriptor: expected 2 fields but got 1" \
  run_invalid_row_parser_check

echo "$log_prefix summary: passed=$passed failed=$failed total=$total"

if (( failed > 0 )); then
  exit 1
fi
