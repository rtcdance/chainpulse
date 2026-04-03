#!/usr/bin/env bash

set -euo pipefail

current_file="${1:-build/migration-governance/baseline-scope-smoke.prom}"
baseline_file="${2:-docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom}"
output_dir="${3:-build/migration-governance}"
failure_mode="${CHAINPULSE_BASELINE_SCOPE_SMOKE_DELTA_FAILURE_MODE:-warn}"

if [[ ! -f "$current_file" ]]; then
  echo "[baseline-scope-smoke-delta] missing current smoke metrics file: $current_file"
  exit 1
fi

if [[ ! -f "$baseline_file" ]]; then
  echo "[baseline-scope-smoke-delta] baseline file not found, using current snapshot as baseline: $baseline_file"
  mkdir -p "$(dirname "$baseline_file")"
  cp "$current_file" "$baseline_file"
fi

case "$failure_mode" in
  warn|enforce)
    ;;
  *)
    echo "[baseline-scope-smoke-delta] invalid CHAINPULSE_BASELINE_SCOPE_SMOKE_DELTA_FAILURE_MODE: $failure_mode (expected warn|enforce)"
    exit 1
    ;;
esac

mkdir -p "$output_dir"
delta_tsv="$output_dir/baseline-scope-smoke-delta.tsv"
delta_md="$output_dir/baseline-scope-smoke-delta.md"

tmp_union="$(mktemp)"
tmp_baseline="$(mktemp)"
tmp_current="$(mktemp)"

cleanup() {
  rm -f "$tmp_union" "$tmp_baseline" "$tmp_current"
}
trap cleanup EXIT

extract_pairs() {
  local src="$1"
  awk '
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*$/ { next }
    {
      print $1 "\t" $2
    }
  ' "$src" | sort
}

extract_pairs "$baseline_file" > "$tmp_baseline"
extract_pairs "$current_file" > "$tmp_current"

awk -F'\t' '{print $1}' "$tmp_baseline" > "$tmp_union"
awk -F'\t' '{print $1}' "$tmp_current" >> "$tmp_union"
sort -u "$tmp_union" -o "$tmp_union"

{
  while IFS= read -r key; do
    baseline_val="$(awk -F'\t' -v k="$key" '$1==k {print $2}' "$tmp_baseline" | tail -n1)"
    current_val="$(awk -F'\t' -v k="$key" '$1==k {print $2}' "$tmp_current" | tail -n1)"
    baseline_val="${baseline_val:-0}"
    current_val="${current_val:-0}"
    delta_val="$(awk -v b="$baseline_val" -v c="$current_val" 'BEGIN { printf "%.0f", c-b }')"
    printf "%s\t%s\t%s\t%s\n" "$key" "$baseline_val" "$current_val" "$delta_val"
  done < "$tmp_union"
} | sort > "$delta_tsv"

metric_value() {
  local src="$1"
  local metric="$2"
  awk -v m="$metric" '$1==m {print $2; found=1} END { if (!found) print "0" }' "$src"
}

baseline_failed="$(metric_value "$baseline_file" "chainpulse_baseline_scope_smoke_failed_total")"
current_failed="$(metric_value "$current_file" "chainpulse_baseline_scope_smoke_failed_total")"
failed_delta=$((current_failed - baseline_failed))

baseline_status="$(metric_value "$baseline_file" "chainpulse_baseline_scope_smoke_status")"
current_status="$(metric_value "$current_file" "chainpulse_baseline_scope_smoke_status")"
status_delta=$((current_status - baseline_status))

regression_signals=()
if (( failed_delta > 0 )); then
  regression_signals+=("failed_cases_increased")
fi
if (( baseline_status == 1 && current_status == 0 )); then
  regression_signals+=("status_regressed_to_fail")
fi

if (( ${#regression_signals[@]} == 0 )); then
  regression_status="none"
else
  regression_status="$(IFS=','; echo "${regression_signals[*]}")"
fi

generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
{
  echo "# Baseline Scope Smoke Delta"
  echo
  echo "- Generated At (UTC): $generated_at"
  echo "- Baseline: $baseline_file"
  echo "- Current: $current_file"
  echo "- Failure Mode: $failure_mode"
  echo
  echo "## Regression Signals"
  echo
  echo "| Signal | Baseline | Current | Delta |"
  echo "|---|---:|---:|---:|"
  echo "|failed_total|$baseline_failed|$current_failed|$failed_delta|"
  echo "|status|$baseline_status|$current_status|$status_delta|"
  echo
  echo "- Status: \`$regression_status\`"
  echo
  echo "## Full Delta Table"
  echo
  echo "| Metric | Baseline | Current | Delta |"
  echo "|---|---:|---:|---:|"
  awk -F'\t' '{printf "|%s|%s|%s|%s|\n", $1, $2, $3, $4}' "$delta_tsv"
} > "$delta_md"

echo "[baseline-scope-smoke-delta] generated:"
echo "  - $delta_tsv"
echo "  - $delta_md"

if [[ "$failure_mode" == "enforce" && "$regression_status" != "none" ]]; then
  echo "[baseline-scope-smoke-delta] regression detected: $regression_status"
  exit 1
fi

