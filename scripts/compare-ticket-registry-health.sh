#!/usr/bin/env bash

set -euo pipefail

current_file="${1:-build/migration-governance/ticket-registry-health.prom}"
baseline_file="${2:-docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom}"
output_dir="${3:-build/migration-governance}"
failure_mode="${CHAINPULSE_MIGRATION_REGISTRY_HEALTH_DELTA_FAILURE_MODE:-warn}"

if [[ ! -f "$current_file" ]]; then
  echo "[ticket-registry-health-delta] missing current health file: $current_file"
  exit 1
fi

if [[ ! -f "$baseline_file" ]]; then
  echo "[ticket-registry-health-delta] baseline file not found, using current snapshot as baseline: $baseline_file"
  mkdir -p "$(dirname "$baseline_file")"
  cp "$current_file" "$baseline_file"
fi

case "$failure_mode" in
  warn|enforce)
    ;;
  *)
    echo "[ticket-registry-health-delta] invalid CHAINPULSE_MIGRATION_REGISTRY_HEALTH_DELTA_FAILURE_MODE: $failure_mode (expected warn|enforce)"
    exit 1
    ;;
esac

mkdir -p "$output_dir"
delta_md="$output_dir/ticket-registry-health-delta.md"
delta_tsv="$output_dir/ticket-registry-health-delta.tsv"

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
      key=$1
      value=$2
      print key "\t" value
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

sum_metric_values() {
  local src="$1"
  local metric_prefix="$2"
  awk -v p="$metric_prefix" '
    $1 ~ ("^" p "\\{") || $1 == p {
      sum += $2
    }
    END {
      printf "%.0f", sum
    }
  ' "$src"
}

baseline_fallback="$(sum_metric_values "$baseline_file" "chainpulse_migration_ticket_registry_fallback_events_total")"
current_fallback="$(sum_metric_values "$current_file" "chainpulse_migration_ticket_registry_fallback_events_total")"
fallback_delta=$((current_fallback - baseline_fallback))

baseline_slo_violations="$(sum_metric_values "$baseline_file" "chainpulse_migration_ticket_registry_http_slo_violations_total")"
current_slo_violations="$(sum_metric_values "$current_file" "chainpulse_migration_ticket_registry_http_slo_violations_total")"
slo_violations_delta=$((current_slo_violations - baseline_slo_violations))

regression_signals=()
if (( fallback_delta > 0 )); then
  regression_signals+=("fallback_events_increased")
fi
if (( slo_violations_delta > 0 )); then
  regression_signals+=("http_slo_violations_increased")
fi

if (( ${#regression_signals[@]} == 0 )); then
  regression_status="none"
else
  regression_status="$(IFS=','; echo "${regression_signals[*]}")"
fi

generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
{
  echo "# Ticket Registry Health Delta"
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
  echo "|fallback_events_total|$baseline_fallback|$current_fallback|$fallback_delta|"
  echo "|http_slo_violations_total|$baseline_slo_violations|$current_slo_violations|$slo_violations_delta|"
  echo
  echo "- Status: \`$regression_status\`"
  echo
  echo "## Full Delta Table"
  echo
  echo "| Metric | Baseline | Current | Delta |"
  echo "|---|---:|---:|---:|"
  awk -F'\t' '{printf "|%s|%s|%s|%s|\n", $1, $2, $3, $4}' "$delta_tsv"
} > "$delta_md"

echo "[ticket-registry-health-delta] generated:"
echo "  - $delta_tsv"
echo "  - $delta_md"

if [[ "$failure_mode" == "enforce" && "$regression_status" != "none" ]]; then
  echo "[ticket-registry-health-delta] regression detected: $regression_status"
  exit 1
fi

