#!/usr/bin/env bash

set -euo pipefail

manifest_file="${1:-docs/operations/MIGRATION_MANIFEST.csv}"
output_dir="${2:-build/migration-governance}"

if [[ ! -f "$manifest_file" ]]; then
  echo "[migration-kpi] missing manifest file: $manifest_file"
  exit 1
fi

mkdir -p "$output_dir"

prom_file="$output_dir/migration-governance-kpi.prom"
md_file="$output_dir/migration-governance-kpi.md"
tmp_file="$(mktemp)"

awk -F',' '
  BEGIN {
    total=0;
    active=0;
  }
  /^[[:space:]]*#/ { next }
  /^[[:space:]]*$/ { next }
  {
    if (NF < 8) next;
    domain=$2;
    status=$4;
    severity=$6;

    total++;
    status_count[status]++;
    severity_count[severity]++;
    domain_count[domain]++;

    if (status != "completed" && status != "waived") {
      active++;
    }
  }
  END {
    print "SUMMARY\ttotal\t" total;
    print "SUMMARY\tactive\t" active;
    for (k in status_count) {
      print "STATUS\t" k "\t" status_count[k];
    }
    for (k in severity_count) {
      print "SEVERITY\t" k "\t" severity_count[k];
    }
    for (k in domain_count) {
      print "DOMAIN\t" k "\t" domain_count[k];
    }
  }
' "$manifest_file" | sort > "$tmp_file"

generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

{
  echo "# HELP chainpulse_migration_items_total Total migration manifest items."
  echo "# TYPE chainpulse_migration_items_total gauge"
  awk -F'\t' '$1=="SUMMARY" && $2=="total" {print "chainpulse_migration_items_total " $3}' "$tmp_file"
  echo
  echo "# HELP chainpulse_migration_items_active_total Active migration items (non-completed/non-waived)."
  echo "# TYPE chainpulse_migration_items_active_total gauge"
  awk -F'\t' '$1=="SUMMARY" && $2=="active" {print "chainpulse_migration_items_active_total " $3}' "$tmp_file"
  echo
  echo "# HELP chainpulse_migration_items_by_status Migration items grouped by status."
  echo "# TYPE chainpulse_migration_items_by_status gauge"
  awk -F'\t' '$1=="STATUS" {printf "chainpulse_migration_items_by_status{status=\"%s\"} %s\n", $2, $3}' "$tmp_file"
  echo
  echo "# HELP chainpulse_migration_items_by_severity Migration items grouped by severity."
  echo "# TYPE chainpulse_migration_items_by_severity gauge"
  awk -F'\t' '$1=="SEVERITY" {printf "chainpulse_migration_items_by_severity{severity=\"%s\"} %s\n", $2, $3}' "$tmp_file"
  echo
  echo "# HELP chainpulse_migration_items_by_domain Migration items grouped by domain."
  echo "# TYPE chainpulse_migration_items_by_domain gauge"
  awk -F'\t' '$1=="DOMAIN" {printf "chainpulse_migration_items_by_domain{domain=\"%s\"} %s\n", $2, $3}' "$tmp_file"
} > "$prom_file"

{
  echo "# Migration Governance KPI Snapshot"
  echo
  echo "- Generated At (UTC): $generated_at"
  echo "- Manifest: $manifest_file"
  echo
  echo "## Summary"
  echo
  echo "| Metric | Value |"
  echo "|---|---:|"
  awk -F'\t' '$1=="SUMMARY" && $2=="total" {print "| Total Items | " $3 " |"}' "$tmp_file"
  awk -F'\t' '$1=="SUMMARY" && $2=="active" {print "| Active Items | " $3 " |"}' "$tmp_file"
  echo
  echo "## By Status"
  echo
  echo "| Status | Count |"
  echo "|---|---:|"
  awk -F'\t' '$1=="STATUS" {print "|" $2 "|" $3 "|"}' "$tmp_file"
  echo
  echo "## By Severity"
  echo
  echo "| Severity | Count |"
  echo "|---|---:|"
  awk -F'\t' '$1=="SEVERITY" {print "|" $2 "|" $3 "|"}' "$tmp_file"
  echo
  echo "## By Domain"
  echo
  echo "| Domain | Count |"
  echo "|---|---:|"
  awk -F'\t' '$1=="DOMAIN" {print "|" $2 "|" $3 "|"}' "$tmp_file"
} > "$md_file"

rm -f "$tmp_file"

echo "[migration-kpi] generated:"
echo "  - $prom_file"
echo "  - $md_file"
