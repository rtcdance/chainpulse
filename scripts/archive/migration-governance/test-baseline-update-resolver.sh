#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/baseline_update_resolver.sh"

log_prefix="[baseline-resolver-test]"
output_dir="${CHAINPULSE_BASELINE_RESOLVER_TEST_OUTPUT_DIR:-build/migration-governance}"
json_output="${CHAINPULSE_BASELINE_RESOLVER_TEST_JSON_OUTPUT:-$output_dir/baseline-resolver-test.json}"
prom_output="${CHAINPULSE_BASELINE_RESOLVER_TEST_PROM_OUTPUT:-$output_dir/baseline-resolver-test.prom}"
md_output="${CHAINPULSE_BASELINE_RESOLVER_TEST_MD_OUTPUT:-$output_dir/baseline-resolver-test.md}"
generated_at_utc="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
total=0
passed=0
failed=0
results_json_items=()

record_result() {
  local name="$1"
  local result="$2"
  results_json_items+=("{\"name\":\"${name}\",\"result\":\"${result}\"}")
}

assert_eq() {
  local name="$1"
  local got="$2"
  local expected="$3"
  total=$((total + 1))
  if [[ "$got" != "$expected" ]]; then
    echo "$log_prefix fail: $name"
    echo "$log_prefix   got:      $got"
    echo "$log_prefix   expected: $expected"
    failed=$((failed + 1))
    record_result "$name" "fail"
    return 0
  fi
  passed=$((passed + 1))
  record_result "$name" "pass"
  echo "$log_prefix pass: $name"
}

assert_ok() {
  local name="$1"
  shift
  total=$((total + 1))
  if "$@" >/dev/null 2>&1; then
    passed=$((passed + 1))
    record_result "$name" "pass"
    echo "$log_prefix pass: $name"
  else
    echo "$log_prefix fail: $name"
    failed=$((failed + 1))
    record_result "$name" "fail"
  fi
}

assert_fail() {
  local name="$1"
  shift
  total=$((total + 1))
  if "$@" >/dev/null 2>&1; then
    echo "$log_prefix fail: $name (expected failure)"
    failed=$((failed + 1))
    record_result "$name" "fail"
    return 0
  fi
  passed=$((passed + 1))
  record_result "$name" "pass"
  echo "$log_prefix pass: $name"
}

assert_eq "normalize-kpi" \
  "$(baseline_resolver_normalize_changed_baselines "kpi")" \
  "kpi"

assert_eq "normalize-health-smoke-order" \
  "$(baseline_resolver_normalize_changed_baselines "smoke,health")" \
  "health,smoke"

assert_eq "normalize-dedup" \
  "$(baseline_resolver_normalize_changed_baselines "kpi,health,kpi,smoke,resolver")" \
  "kpi,health,smoke,resolver"

assert_fail "normalize-invalid-token" \
  baseline_resolver_normalize_changed_baselines "kpi,db"

assert_ok "validate-scope-dual" \
  baseline_resolver_validate_scope "dual"

assert_fail "validate-scope-invalid" \
  baseline_resolver_validate_scope "all"

assert_eq "resolve-scope-auto-dual" \
  "$(baseline_resolver_resolve_scope "" "true" "false")" \
  "dual"

assert_eq "resolve-scope-auto-kpi-only" \
  "$(baseline_resolver_resolve_scope "" "false" "false")" \
  "kpi-only"

assert_eq "resolve-scope-override" \
  "$(baseline_resolver_resolve_scope "health-only" "false" "false")" \
  "health-only"

assert_eq "resolve-changed-auto-all" \
  "$(baseline_resolver_resolve_changed_baselines "" "true" "true" "true")" \
  "kpi,health,smoke,resolver"

assert_eq "resolve-changed-auto-kpi" \
  "$(baseline_resolver_resolve_changed_baselines "" "false" "false")" \
  "kpi"

assert_eq "resolve-changed-override" \
  "$(baseline_resolver_resolve_changed_baselines "kpi,smoke,resolver" "true" "true" "true")" \
  "kpi,smoke,resolver"

mkdir -p "$output_dir"
status="pass"
if (( failed > 0 )); then
  status="fail"
fi

results_joined=""
if (( ${#results_json_items[@]} > 0 )); then
  results_joined="$(IFS=,; echo "${results_json_items[*]}")"
fi

{
  echo "{"
  echo "  \"generated_at_utc\": \"${generated_at_utc}\","
  echo "  \"status\": \"${status}\","
  echo "  \"total\": ${total},"
  echo "  \"passed\": ${passed},"
  echo "  \"failed\": ${failed},"
  echo "  \"results\": [${results_joined}]"
  echo "}"
} > "$json_output"

{
  echo "# HELP chainpulse_baseline_resolver_test_total Total resolver test cases."
  echo "# TYPE chainpulse_baseline_resolver_test_total gauge"
  echo "chainpulse_baseline_resolver_test_total ${total}"
  echo
  echo "# HELP chainpulse_baseline_resolver_test_passed Total passed resolver test cases."
  echo "# TYPE chainpulse_baseline_resolver_test_passed gauge"
  echo "chainpulse_baseline_resolver_test_passed ${passed}"
  echo
  echo "# HELP chainpulse_baseline_resolver_test_failed Total failed resolver test cases."
  echo "# TYPE chainpulse_baseline_resolver_test_failed gauge"
  echo "chainpulse_baseline_resolver_test_failed ${failed}"
  echo
  echo "# HELP chainpulse_baseline_resolver_test_status Resolver test status (1=pass,0=fail)."
  echo "# TYPE chainpulse_baseline_resolver_test_status gauge"
  if [[ "$status" == "pass" ]]; then
    echo "chainpulse_baseline_resolver_test_status 1"
  else
    echo "chainpulse_baseline_resolver_test_status 0"
  fi
} > "$prom_output"

{
  echo "# Baseline Resolver Tests"
  echo
  echo "- Generated At (UTC): $generated_at_utc"
  echo "- Status: \`$status\`"
  echo
  echo "| Field | Value |"
  echo "|---|---:|"
  echo "| total | $total |"
  echo "| passed | $passed |"
  echo "| failed | $failed |"
  echo
  if (( failed > 0 )); then
    echo "## Failure Summary"
    echo
    echo "| Case | Result |"
    echo "|---|---|"
    for item in "${results_json_items[@]}"; do
      case_name="$(echo "$item" | sed -E 's/.*"name":"([^"]+)".*/\1/')"
      result="$(echo "$item" | sed -E 's/.*"result":"([^"]+)".*/\1/')"
      if [[ "$result" == "fail" ]]; then
        echo "| $case_name | $result |"
      fi
    done
    echo
  fi
  echo "## Case Results"
  echo
  echo "| Case | Result |"
  echo "|---|---|"
  for item in "${results_json_items[@]}"; do
    case_name="$(echo "$item" | sed -E 's/.*"name":"([^"]+)".*/\1/')"
    result="$(echo "$item" | sed -E 's/.*"result":"([^"]+)".*/\1/')"
    echo "| $case_name | $result |"
  done
} > "$md_output"

echo "$log_prefix outputs:"
echo "$log_prefix   - $json_output"
echo "$log_prefix   - $prom_output"
echo "$log_prefix   - $md_output"
echo "$log_prefix summary: passed=$passed failed=$failed total=$total"

if (( failed > 0 )); then
  exit 1
fi
