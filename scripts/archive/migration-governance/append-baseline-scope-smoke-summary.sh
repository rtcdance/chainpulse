#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/governance_summary.sh"

log_prefix="[baseline-scope-smoke-summary]"
report_file="${1:-build/migration-governance/baseline-scope-smoke.md}"
delta_file="${2:-build/migration-governance/baseline-scope-smoke-delta.md}"
summary_output="${CHAINPULSE_BASELINE_SCOPE_SMOKE_JOB_SUMMARY_OUTPUT:-${GITHUB_STEP_SUMMARY:-}}"

append_failure_summary() {
  if awk '/^## Failure Summary$/ { found=1; exit } END { exit(found ? 0 : 1) }' "$report_file"; then
    awk '
      /^## Failure Summary$/ { in_block=1; print "### Failure Summary"; next }
      /^## / && in_block { exit }
      in_block { print }
    ' "$report_file" >> "$summary_output"
  else
    {
      echo "### Failure Summary"
      echo
      echo "- No failed smoke scenarios."
    } >> "$summary_output"
  fi
}

governance_summary_assert_inputs "$log_prefix" "$report_file" "$summary_output"

status="$(governance_summary_extract_status "$report_file")"
total_cases="$(governance_summary_trim "$(governance_summary_extract_table_value "$report_file" "total_cases")")"
passed_cases="$(governance_summary_trim "$(governance_summary_extract_table_value "$report_file" "passed_cases")")"
failed_cases="$(governance_summary_trim "$(governance_summary_extract_table_value "$report_file" "failed_cases")")"

{
  echo "## Baseline Scope Smoke Highlights"
  echo
  echo "- Status: \`$status\`"
  echo "- Total Cases: $total_cases"
  echo "- Passed Cases: $passed_cases"
  echo "- Failed Cases: $failed_cases"
  echo
} >> "$summary_output"

append_failure_summary

{
  echo
} >> "$summary_output"

governance_summary_append_delta_highlights "$delta_file" "$summary_output"

{
  echo
} >> "$summary_output"

echo "$log_prefix appended highlights to: $summary_output"
