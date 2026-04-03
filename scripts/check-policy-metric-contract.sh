#!/usr/bin/env bash

set -euo pipefail

schema_mode="${CHAINPULSE_POLICY_METRIC_SCHEMA_MODE:-v1}"
deprecation_date="${CHAINPULSE_POLICY_V1_DEPRECATION_DATE:-}"
warn_days="${CHAINPULSE_POLICY_V1_DEPRECATION_WARN_DAYS:-14}"

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

enforce_deprecation_cutoff() {
  if [[ -z "$deprecation_date" ]]; then
    return 0
  fi

  local cutoff_epoch now_epoch days_left
  cutoff_epoch="$(to_epoch_utc "$deprecation_date")" || {
    echo "[policy-contract] invalid CHAINPULSE_POLICY_V1_DEPRECATION_DATE: $deprecation_date (expected YYYY-MM-DD)"
    exit 1
  }
  now_epoch="$(date -u +%s)"
  days_left="$(( (cutoff_epoch - now_epoch) / 86400 ))"

  if (( now_epoch >= cutoff_epoch )); then
    if [[ "$schema_mode" != "v2" ]]; then
      echo "[policy-contract] v1 deprecation deadline reached ($deprecation_date); CHAINPULSE_POLICY_METRIC_SCHEMA_MODE must be v2, got: $schema_mode"
      exit 1
    fi
    return 0
  fi

  if (( days_left <= warn_days )) && [[ "$schema_mode" != "v2" ]]; then
    echo "[policy-contract] warning: v1 deprecation deadline in ${days_left}d ($deprecation_date); current schema mode: $schema_mode"
  fi
}

echo "[policy-contract] schema_mode=$schema_mode"
enforce_deprecation_cutoff
echo "[policy-contract] validating policy metric/tag schema contracts"
go test -short ./pkg/application/bootstrap -run 'TestPolicyMetricTagContractStability|TestEmitPolicyOverrideMetricsBySchemaMode|TestPolicyMetricSchemaPlanForMode' -count=1
echo "[policy-contract] contract check passed"
