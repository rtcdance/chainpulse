#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/governance_summary.sh"

log_prefix="[baseline-resolver-test-summary]"
report_file="${1:-build/migration-governance/baseline-resolver-test.md}"
delta_file="${2:-build/migration-governance/baseline-resolver-test-delta.md}"
summary_output="${CHAINPULSE_BASELINE_RESOLVER_TEST_JOB_SUMMARY_OUTPUT:-${GITHUB_STEP_SUMMARY:-}}"
governance_summary_assert_inputs "$log_prefix" "$report_file" "$summary_output"

status="$(governance_summary_extract_status "$report_file")"
total="$(governance_summary_trim "$(governance_summary_extract_table_value "$report_file" "total")")"
passed="$(governance_summary_trim "$(governance_summary_extract_table_value "$report_file" "passed")")"
failed="$(governance_summary_trim "$(governance_summary_extract_table_value "$report_file" "failed")")"

{
  echo "## Baseline Resolver Test Highlights"
  echo
  echo "- Status: \`$status\`"
  echo "- Total Cases: $total"
  echo "- Passed Cases: $passed"
  echo "- Failed Cases: $failed"
  echo
} >> "$summary_output"

governance_summary_append_failure_section \
  "$report_file" \
  "$summary_output" \
  "### Failure Summary" \
  "- No failed resolver scenarios."

{
  echo
} >> "$summary_output"

governance_summary_append_delta_highlights "$delta_file" "$summary_output"

{
  echo
} >> "$summary_output"

echo "$log_prefix appended highlights to: $summary_output"
