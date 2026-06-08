#!/usr/bin/env bash

set -euo pipefail

current_file="${1:-build/migration-governance/migration-governance-kpi.prom}"
baseline_file="${2:-docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom}"
output_dir="${3:-build/migration-governance}"

if [[ ! -f "$current_file" ]]; then
  echo "[migration-kpi-delta] missing current KPI file: $current_file"
  exit 1
fi

if [[ ! -f "$baseline_file" ]]; then
  echo "[migration-kpi-delta] baseline file not found, using current snapshot as baseline: $baseline_file"
  mkdir -p "$(dirname "$baseline_file")"
  cp "$current_file" "$baseline_file"
fi

mkdir -p "$output_dir"
delta_md="$output_dir/migration-governance-delta.md"
delta_tsv="$output_dir/migration-governance-delta.tsv"

tmp_union="$(mktemp)"
tmp_baseline="$(mktemp)"
tmp_current="$(mktemp)"

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

generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
{
  echo "# Migration Governance KPI Delta"
  echo
  echo "- Generated At (UTC): $generated_at"
  echo "- Baseline: $baseline_file"
  echo "- Current: $current_file"
  echo
  echo "## Delta Table"
  echo
  echo "| Metric | Baseline | Current | Delta |"
  echo "|---|---:|---:|---:|"
  awk -F'\t' '{printf "|%s|%s|%s|%s|\n", $1, $2, $3, $4}' "$delta_tsv"
  echo
  echo "## PR Comment Draft"
  echo
  echo "### Migration Governance KPI Delta"
  echo
  echo "- Baseline: \`$baseline_file\`"
  echo "- Current: \`$current_file\`"
  echo
  echo "| Metric | Delta |"
  echo "|---|---:|"
  awk -F'\t' '{printf "|%s|%s|\n", $1, $4}' "$delta_tsv"
} > "$delta_md"

rm -f "$tmp_union" "$tmp_baseline" "$tmp_current"

echo "[migration-kpi-delta] generated:"
echo "  - $delta_tsv"
echo "  - $delta_md"
