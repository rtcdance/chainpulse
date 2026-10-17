#!/usr/bin/env bash

set -euo pipefail

manifest_file="${1:-docs/operations/MIGRATION_MANIFEST.csv}"
warn_days="${CHAINPULSE_MIGRATION_WARN_DAYS:-14}"
require_spec_sync="${CHAINPULSE_MIGRATION_REQUIRE_SPEC_SYNC:-true}"
require_spec_metadata_sync="${CHAINPULSE_MIGRATION_REQUIRE_SPEC_METADATA_SYNC:-true}"
owner_allowlist="${CHAINPULSE_MIGRATION_OWNER_ALLOWLIST:-platform-team,sre-team,indexer-team}"

if [[ ! -f "$manifest_file" ]]; then
  echo "[migration-manifest] missing manifest file: $manifest_file"
  exit 1
fi

to_epoch_utc() {
  local date_input="$1"
  if date -u -d "$date_input" +%s >/dev/null 2>&1; then
    date -u -d "$date_input" +%s
    return 0
  fi
  if date -u -j -f "%Y-%m-%d" "$date_input" +%s >/dev/null 2>&1; then
    date -u -j -f "%Y-%m-%d" "$date_input" +%s
    return 0
  fi
  return 1
}

valid_status() {
  case "$1" in
    planned|in_progress|blocked|completed|waived)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

owner_allowed() {
  local owner="$1"
  IFS=',' read -r -a owners <<< "$owner_allowlist"
  for allowed in "${owners[@]}"; do
    if [[ "$(echo "$allowed" | xargs)" == "$owner" ]]; then
      return 0
    fi
  done
  return 1
}

normalize_status() {
  case "$1" in
    planned)
      echo "Planned"
      ;;
    in_progress|blocked)
      echo "Implemented"
      ;;
    completed)
      echo "Implemented"
      ;;
    waived)
      echo "Planned"
      ;;
    *)
      echo ""
      ;;
  esac
}

check_spec_metadata_sync() {
  local spec_file="$1"
  local expected_owner="$2"
  local expected_deadline="$3"
  local expected_status="$4"

  if [[ ! -f "$spec_file" ]]; then
    echo "[migration-manifest] missing spec_ref file: $spec_file"
    return 1
  fi

  if ! grep -Eq '^## Owner' "$spec_file"; then
    echo "[migration-manifest] spec missing ## Owner section: $spec_file"
    return 1
  fi

  local owner_line
  owner_line="$(awk '
    /^## Owner/ {in_owner=1; next}
    /^## / {if (in_owner) exit}
    in_owner && NF>0 {print; exit}
  ' "$spec_file")"

  if [[ -z "$owner_line" ]]; then
    echo "[migration-manifest] spec owner value not found: $spec_file"
    return 1
  fi

  local owner_lc expected_owner_lc
  owner_lc="$(echo "$owner_line" | tr '[:upper:]' '[:lower:]')"
  expected_owner_lc="$(echo "$expected_owner" | tr '[:upper:]' '[:lower:]')"
  if [[ "$owner_lc" != *"$expected_owner_lc"* ]]; then
    echo "[migration-manifest] owner mismatch: manifest owner='$expected_owner' spec owner line='$owner_line' spec_ref=$spec_file"
    return 1
  fi

  local normalized_expected_status
  normalized_expected_status="$(normalize_status "$expected_status")"
  if [[ -n "$normalized_expected_status" ]]; then
    if ! grep -Eq "^## Delivery Status" "$spec_file"; then
      echo "[migration-manifest] spec missing ## Delivery Status section: $spec_file"
      return 1
    fi

    local delivery_line
    delivery_line="$(awk '
      /^## Delivery Status/ {in_delivery=1; next}
      /^## / {if (in_delivery) exit}
      in_delivery && NF>0 {print; exit}
    ' "$spec_file")"
    if [[ -z "$delivery_line" ]]; then
      echo "[migration-manifest] spec delivery status value not found: $spec_file"
      return 1
    fi
    if [[ "$delivery_line" != *"$normalized_expected_status"* ]]; then
      echo "[migration-manifest] delivery status mismatch: manifest status='$expected_status' expected delivery='$normalized_expected_status' spec delivery line='$delivery_line' spec_ref=$spec_file"
      return 1
    fi
  fi

  if ! grep -Fq "$expected_deadline" "$spec_file"; then
    echo "[migration-manifest] deadline not found in spec: manifest deadline='$expected_deadline' spec_ref=$spec_file"
    return 1
  fi

  return 0
}

today_epoch="$(date -u +%s)"
overdue_count=0
warn_count=0
line_no=0

echo "[migration-manifest] validating $manifest_file"
while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  line_no=$((line_no + 1))
  line="$(echo "$raw_line" | sed 's/[[:space:]]*$//')"

  if [[ -z "$line" || "$line" == \#* ]]; then
    continue
  fi

  IFS=',' read -r id domain owner status deadline severity spec_ref notes <<< "$line"
  if [[ -z "${notes:-}" ]]; then
    echo "[migration-manifest] line $line_no invalid format (expected 8 comma-separated fields): $line"
    exit 1
  fi

  if [[ -z "$id" || -z "$domain" || -z "$owner" || -z "$status" || -z "$deadline" || -z "$severity" || -z "$spec_ref" ]]; then
    echo "[migration-manifest] line $line_no has empty required field: $line"
    exit 1
  fi

  if ! owner_allowed "$owner"; then
    echo "[migration-manifest] line $line_no owner '$owner' is not in allowlist: $owner_allowlist"
    exit 1
  fi

  if ! valid_status "$status"; then
    echo "[migration-manifest] line $line_no has invalid status '$status' (allowed: planned,in_progress,blocked,completed,waived)"
    exit 1
  fi

  deadline_epoch="$(to_epoch_utc "$deadline")" || {
    echo "[migration-manifest] line $line_no has invalid deadline '$deadline' (expected YYYY-MM-DD)"
    exit 1
  }

  if [[ "$status" != "completed" && "$status" != "waived" ]]; then
    if [[ "$require_spec_sync" == "true" ]]; then
      if [[ ! -f "$spec_ref" ]]; then
        echo "[migration-manifest] line $line_no missing spec_ref file: $spec_ref"
        exit 1
      fi
      ./scripts/spec-approval-check.sh "$spec_ref" >/dev/null

      if [[ "$require_spec_metadata_sync" == "true" ]]; then
        check_spec_metadata_sync "$spec_ref" "$owner" "$deadline" "$status" || exit 1
      fi
    fi

    if (( today_epoch > deadline_epoch )); then
      overdue_count=$((overdue_count + 1))
      echo "[migration-manifest] overdue: id=$id owner=$owner status=$status deadline=$deadline severity=$severity spec_ref=$spec_ref"
    else
      days_left=$(( (deadline_epoch - today_epoch) / 86400 ))
      if (( days_left <= warn_days )); then
        warn_count=$((warn_count + 1))
        echo "[migration-manifest] warning: id=$id owner=$owner status=$status deadline=$deadline in ${days_left}d spec_ref=$spec_ref"
      fi
    fi
  fi
done < "$manifest_file"

if (( overdue_count > 0 )); then
  echo "[migration-manifest] failed: $overdue_count overdue migration item(s)"
  exit 1
fi

echo "[migration-manifest] passed: no overdue items (warnings=$warn_count)"
