#!/usr/bin/env bash

set -euo pipefail

log_prefix="[baseline-resolver-test-summary-test]"
script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/append-baseline-resolver-test-summary.sh"
total=0
passed=0
failed=0

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

assert_not_contains() {
  local name="$1"
  local file="$2"
  local pattern="$3"
  total=$((total + 1))
  if grep -Fq -- "$pattern" "$file"; then
    failed=$((failed + 1))
    echo "$log_prefix fail: $name"
    echo "$log_prefix   unexpected: $pattern"
    return 0
  fi
  passed=$((passed + 1))
  echo "$log_prefix pass: $name"
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

pass_report="$tmp_dir/pass.md"
pass_delta="$tmp_dir/pass-delta.md"
pass_summary="$tmp_dir/pass-summary.md"
cat > "$pass_report" <<'EOF'
# Baseline Resolver Tests

- Generated At (UTC): 2026-03-30T00:00:00Z
- Status: `pass`

| Field | Value |
|---|---:|
| total | 12 |
| passed | 12 |
| failed | 0 |

## Case Results

| Case | Result |
|---|---|
| normalize-kpi | pass |
EOF

cat > "$pass_delta" <<'EOF'
# Baseline Resolver Test Delta

- Generated At (UTC): 2026-03-30T00:00:00Z
- Baseline: docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom
- Current: build/migration-governance/baseline-resolver-test.prom
- Failure Mode: warn

## Regression Signals

| Signal | Baseline | Current | Delta |
|---|---:|---:|---:|
|failed_total|0|0|0|
|status|1|1|0|

- Status: `none`
EOF

CHAINPULSE_BASELINE_RESOLVER_TEST_JOB_SUMMARY_OUTPUT="$pass_summary" \
  "$script_path" "$pass_report" "$pass_delta" >/dev/null

assert_contains "pass-header" "$pass_summary" "## Baseline Resolver Test Highlights"
assert_contains "pass-status" "$pass_summary" "- Status: \`pass\`"
assert_contains "pass-failed-count" "$pass_summary" "- Failed Cases: 0"
assert_contains "pass-failure-heading" "$pass_summary" "### Failure Summary"
assert_contains "pass-no-failure-message" "$pass_summary" "- No failed resolver scenarios."
assert_contains "pass-delta-heading" "$pass_summary" "### Delta Highlights"
assert_contains "pass-delta-status" "$pass_summary" "- Regression Status: \`none\`"
assert_contains "pass-delta-row" "$pass_summary" "|failed_total|0|0|0|"
assert_not_contains "pass-no-case-results-heading" "$pass_summary" "## Case Results"

fail_report="$tmp_dir/fail.md"
fail_delta="$tmp_dir/fail-delta.md"
fail_summary="$tmp_dir/fail-summary.md"
cat > "$fail_report" <<'EOF'
# Baseline Resolver Tests

- Generated At (UTC): 2026-03-30T00:00:00Z
- Status: `fail`

| Field | Value |
|---|---:|
| total | 12 |
| passed | 10 |
| failed | 2 |

## Failure Summary

| Case | Result |
|---|---|
| normalize-kpi | fail |
| validate-scope-dual | fail |

## Case Results

| Case | Result |
|---|---|
| normalize-kpi | fail |
EOF

cat > "$fail_delta" <<'EOF'
# Baseline Resolver Test Delta

- Generated At (UTC): 2026-03-30T00:00:00Z
- Baseline: docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom
- Current: build/migration-governance/baseline-resolver-test.prom
- Failure Mode: enforce

## Regression Signals

| Signal | Baseline | Current | Delta |
|---|---:|---:|---:|
|failed_total|0|2|2|
|status|1|0|-1|

- Status: `failed_cases_increased,status_regressed_to_fail`
EOF

CHAINPULSE_BASELINE_RESOLVER_TEST_JOB_SUMMARY_OUTPUT="$fail_summary" \
  "$script_path" "$fail_report" "$fail_delta" >/dev/null

assert_contains "fail-status" "$fail_summary" "- Status: \`fail\`"
assert_contains "fail-failed-count" "$fail_summary" "- Failed Cases: 2"
assert_contains "fail-failure-heading" "$fail_summary" "### Failure Summary"
assert_contains "fail-failure-row" "$fail_summary" "| normalize-kpi | fail |"
assert_contains "fail-delta-status" "$fail_summary" "- Regression Status: \`failed_cases_increased,status_regressed_to_fail\`"
assert_contains "fail-delta-row" "$fail_summary" "|status|1|0|-1|"
assert_not_contains "fail-no-case-results-heading" "$fail_summary" "## Case Results"

echo "$log_prefix summary: passed=$passed failed=$failed total=$total"

if (( failed > 0 )); then
  exit 1
fi
