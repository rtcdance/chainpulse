#!/usr/bin/env bash

# Shared resolver helpers for baseline update/preflight scripts.

baseline_resolver_normalize_changed_baselines() {
  local raw="$1"
  local has_kpi="false"
  local has_health="false"
  local has_smoke="false"
  local has_resolver="false"

  if [[ -z "$raw" ]]; then
    return 1
  fi

  IFS=',' read -r -a values <<< "$raw"
  for value in "${values[@]}"; do
    value="$(echo "$value" | xargs)"
    case "$value" in
      kpi) has_kpi="true" ;;
      health) has_health="true" ;;
      smoke) has_smoke="true" ;;
      resolver) has_resolver="true" ;;
      *) return 1 ;;
    esac
  done

  local normalized=""
  if [[ "$has_kpi" == "true" ]]; then
    normalized="kpi"
  fi
  if [[ "$has_health" == "true" ]]; then
    if [[ -n "$normalized" ]]; then
      normalized="${normalized},health"
    else
      normalized="health"
    fi
  fi
  if [[ "$has_smoke" == "true" ]]; then
    if [[ -n "$normalized" ]]; then
      normalized="${normalized},smoke"
    else
      normalized="smoke"
    fi
  fi
  if [[ "$has_resolver" == "true" ]]; then
    if [[ -n "$normalized" ]]; then
      normalized="${normalized},resolver"
    else
      normalized="resolver"
    fi
  fi

  if [[ -z "$normalized" ]]; then
    return 1
  fi

  echo "$normalized"
}

baseline_resolver_validate_refresh_flag() {
  local value="$1"
  case "$value" in
    true|false) return 0 ;;
    *) return 1 ;;
  esac
}

baseline_resolver_validate_scope() {
  local value="$1"
  case "$value" in
    kpi-only|health-only|dual) return 0 ;;
    *) return 1 ;;
  esac
}

baseline_resolver_resolve_scope() {
  local provided_scope="$1"
  local refresh_health_baseline="$2"
  local refresh_smoke_baseline="$3"

  if [[ -n "$provided_scope" ]]; then
    echo "$provided_scope"
    return 0
  fi

  if [[ "$refresh_health_baseline" == "true" || "$refresh_smoke_baseline" == "true" ]]; then
    echo "dual"
  else
    echo "kpi-only"
  fi
}

baseline_resolver_resolve_changed_baselines() {
  local provided_changed_baselines="$1"
  local refresh_health_baseline="$2"
  local refresh_smoke_baseline="$3"
  local refresh_resolver_baseline="${4:-false}"

  if [[ -n "$provided_changed_baselines" ]]; then
    echo "$provided_changed_baselines"
    return 0
  fi

  local resolved="kpi"
  if [[ "$refresh_health_baseline" == "true" ]]; then
    resolved="${resolved},health"
  fi
  if [[ "$refresh_smoke_baseline" == "true" ]]; then
    resolved="${resolved},smoke"
  fi
  if [[ "$refresh_resolver_baseline" == "true" ]]; then
    resolved="${resolved},resolver"
  fi
  echo "$resolved"
}
