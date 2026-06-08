#!/usr/bin/env bash

set -euo pipefail

manifest_file="${1:-docs/operations/MIGRATION_MANIFEST.csv}"
output_dir="${2:-build/migration-governance}"
owner_allowlist="${CHAINPULSE_MIGRATION_OWNER_ALLOWLIST:-platform-team,sre-team,indexer-team}"
fail_on_unknown="${CHAINPULSE_FAIL_ON_UNKNOWN_MIGRATION_OWNER:-false}"

if [[ ! -f "$manifest_file" ]]; then
  echo "[migration-owner-drift] missing manifest file: $manifest_file"
  exit 1
fi

mkdir -p "$output_dir"
report_md="$output_dir/migration-owner-drift-report.md"
report_tsv="$output_dir/migration-owner-drift-report.tsv"

allow_tmp="$(mktemp)"
owners_tmp="$(mktemp)"

echo "$owner_allowlist" | tr ',' '\n' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed '/^$/d' | sort -u > "$allow_tmp"

awk -F',' '
  /^[[:space:]]*#/ { next }
  /^[[:space:]]*$/ { next }
  NF >= 3 { print $3 }
' "$manifest_file" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed '/^$/d' | sort -u > "$owners_tmp"

unknown_tmp="$(mktemp)"
comm -23 "$owners_tmp" "$allow_tmp" > "$unknown_tmp"

{
  while IFS= read -r owner; do
    [[ -z "$owner" ]] && continue
    status="allowed"
    if grep -Fxq "$owner" "$unknown_tmp"; then
      status="unknown"
    fi
    printf "%s\t%s\n" "$owner" "$status"
  done < "$owners_tmp"
} > "$report_tsv"

generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
unknown_count="$(awk -F'\t' '$2=="unknown"{c++} END{print c+0}' "$report_tsv")"
total_count="$(wc -l < "$report_tsv" | tr -d ' ')"

{
  echo "# Migration Owner Drift Report"
  echo
  echo "- Generated At (UTC): $generated_at"
  echo "- Manifest: $manifest_file"
  echo "- Allowlist: $owner_allowlist"
  echo
  echo "## Summary"
  echo
  echo "| Metric | Value |"
  echo "|---|---:|"
  echo "| Distinct Owners | $total_count |"
  echo "| Unknown Owners | $unknown_count |"
  echo
  echo "## Owners"
  echo
  echo "| Owner | Status |"
  echo "|---|---|"
  awk -F'\t' '{printf "|%s|%s|\n",$1,$2}' "$report_tsv"
} > "$report_md"

rm -f "$allow_tmp" "$owners_tmp" "$unknown_tmp"

echo "[migration-owner-drift] generated:"
echo "  - $report_tsv"
echo "  - $report_md"

if [[ "$fail_on_unknown" == "true" && "$unknown_count" -gt 0 ]]; then
  echo "[migration-owner-drift] failed: unknown owners detected ($unknown_count)"
  exit 1
fi
