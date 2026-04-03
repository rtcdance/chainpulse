#!/usr/bin/env bash

set -euo pipefail

enforce="${CHAINPULSE_ENFORCE_BASELINE_CHANGELOG:-true}"
diff_ref="${CHAINPULSE_MIGRATION_BASELINE_DIFF_REF:-HEAD~1}"
baseline_file="docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom"
health_baseline_file="${CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE_FILE:-docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom}"
smoke_baseline_file="${CHAINPULSE_MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE_FILE:-docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom}"
resolver_baseline_file="${CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE:-docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom}"
changelog_file="docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md"
enforce_scope_alignment="${CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT:-true}"
enforce_changed_baselines_alignment="${CHAINPULSE_MIGRATION_ENFORCE_CHANGED_BASELINES_ALIGNMENT:-true}"

if [[ "$enforce" != "true" ]]; then
  echo "[migration-kpi-baseline] governance check disabled"
  exit 0
fi

case "$enforce_scope_alignment" in
  true|false)
    ;;
  *)
    echo "[migration-kpi-baseline] invalid CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT: $enforce_scope_alignment (expected true|false)"
    exit 1
    ;;
esac

case "$enforce_changed_baselines_alignment" in
  true|false)
    ;;
  *)
    echo "[migration-kpi-baseline] invalid CHAINPULSE_MIGRATION_ENFORCE_CHANGED_BASELINES_ALIGNMENT: $enforce_changed_baselines_alignment (expected true|false)"
    exit 1
    ;;
esac

normalize_changed_baselines() {
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

if [[ ! -f "$baseline_file" ]]; then
  echo "[migration-kpi-baseline] missing baseline file: $baseline_file"
  exit 1
fi

if [[ ! -f "$health_baseline_file" ]]; then
  echo "[migration-kpi-baseline] missing ticket registry health baseline file: $health_baseline_file"
  exit 1
fi

if [[ ! -f "$smoke_baseline_file" ]]; then
  echo "[migration-kpi-baseline] missing baseline scope smoke baseline file: $smoke_baseline_file"
  exit 1
fi

if [[ ! -f "$resolver_baseline_file" ]]; then
  echo "[migration-kpi-baseline] missing baseline resolver test baseline file: $resolver_baseline_file"
  exit 1
fi

if [[ ! -f "$changelog_file" ]]; then
  echo "[migration-kpi-baseline] missing changelog file: $changelog_file"
  exit 1
fi

./scripts/check-migration-changelog-quality.sh "$changelog_file" >/dev/null

if ! git rev-parse --verify "$diff_ref" >/dev/null 2>&1; then
  echo "[migration-kpi-baseline] diff ref '$diff_ref' not available; skipping changed-file comparison"
  exit 0
fi

changed_files="$(git diff --name-only "$diff_ref"...HEAD || true)"
baseline_changed="false"
health_baseline_changed="false"
smoke_baseline_changed="false"
resolver_baseline_changed="false"
changelog_changed="false"

if echo "$changed_files" | grep -Fxq "$baseline_file"; then
  baseline_changed="true"
fi
if echo "$changed_files" | grep -Fxq "$health_baseline_file"; then
  health_baseline_changed="true"
fi
if echo "$changed_files" | grep -Fxq "$smoke_baseline_file"; then
  smoke_baseline_changed="true"
fi
if echo "$changed_files" | grep -Fxq "$resolver_baseline_file"; then
  resolver_baseline_changed="true"
fi
if echo "$changed_files" | grep -Fxq "$changelog_file"; then
  changelog_changed="true"
fi

if [[ ("$baseline_changed" == "true" || "$health_baseline_changed" == "true" || "$smoke_baseline_changed" == "true" || "$resolver_baseline_changed" == "true") && "$changelog_changed" != "true" ]]; then
  echo "[migration-kpi-baseline] baseline changed without changelog update"
  echo "  kpi baseline: $baseline_file"
  echo "  health baseline: $health_baseline_file"
  echo "  smoke baseline: $smoke_baseline_file"
  echo "  resolver baseline: $resolver_baseline_file"
  echo "  required changelog: $changelog_file"
  exit 1
fi

if [[ "$baseline_changed" == "true" || "$health_baseline_changed" == "true" || "$smoke_baseline_changed" == "true" || "$resolver_baseline_changed" == "true" ]]; then
  echo "[migration-kpi-baseline] baseline change is accompanied by changelog update"
  echo "  kpi baseline changed: $baseline_changed"
  echo "  health baseline changed: $health_baseline_changed"
  echo "  smoke baseline changed: $smoke_baseline_changed"
  echo "  resolver baseline changed: $resolver_baseline_changed"

  if [[ "$enforce_scope_alignment" == "true" ]]; then
    latest_scope="missing"
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ -z "$line" || "$line" == \#* ]]; then
        continue
      fi
      if [[ "$line" =~ scope=([^[:space:]]+) ]]; then
        latest_scope="${BASH_REMATCH[1]}"
      fi
      break
    done < "$changelog_file"

    if [[ "$latest_scope" == "missing" || -z "$latest_scope" ]]; then
      echo "[migration-kpi-baseline] latest changelog entry missing scope tag"
      echo "  required scope values: kpi-only|health-only|dual"
      exit 1
    fi

    expected_scope=""
    any_health_related_changed="false"
    if [[ "$health_baseline_changed" == "true" || "$smoke_baseline_changed" == "true" ]]; then
      any_health_related_changed="true"
    fi

    if [[ "$baseline_changed" == "true" && "$any_health_related_changed" == "true" ]]; then
      expected_scope="dual"
    elif [[ "$baseline_changed" == "true" && "$any_health_related_changed" != "true" ]]; then
      expected_scope="kpi-only"
    elif [[ "$baseline_changed" != "true" && "$any_health_related_changed" == "true" ]]; then
      expected_scope="health-only"
    fi

    if [[ -n "$expected_scope" && "$latest_scope" != "$expected_scope" ]]; then
      echo "[migration-kpi-baseline] changelog scope does not match baseline diff"
      echo "  expected scope: $expected_scope"
      echo "  latest scope: $latest_scope"
      exit 1
    fi
  fi

  if [[ "$enforce_changed_baselines_alignment" == "true" ]]; then
    latest_changed_baselines="missing"
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ -z "$line" || "$line" == \#* ]]; then
        continue
      fi
      if [[ "$line" =~ changed_baselines=([^[:space:]]+) ]]; then
        latest_changed_baselines="${BASH_REMATCH[1]}"
      fi
      break
    done < "$changelog_file"

    if [[ "$latest_changed_baselines" == "missing" || -z "$latest_changed_baselines" ]]; then
      echo "[migration-kpi-baseline] latest changelog entry missing changed_baselines tag"
      echo "  required values: subset of kpi,health,smoke,resolver"
      exit 1
    fi

    if ! latest_changed_baselines="$(normalize_changed_baselines "$latest_changed_baselines")"; then
      echo "[migration-kpi-baseline] invalid changed_baselines tag in latest changelog entry: $latest_changed_baselines"
      exit 1
    fi

    expected_changed_baselines=""
    if [[ "$baseline_changed" == "true" ]]; then
      expected_changed_baselines="kpi"
    fi
    if [[ "$health_baseline_changed" == "true" ]]; then
      if [[ -n "$expected_changed_baselines" ]]; then
        expected_changed_baselines="${expected_changed_baselines},health"
      else
        expected_changed_baselines="health"
      fi
    fi
    if [[ "$smoke_baseline_changed" == "true" ]]; then
      if [[ -n "$expected_changed_baselines" ]]; then
        expected_changed_baselines="${expected_changed_baselines},smoke"
      else
        expected_changed_baselines="smoke"
      fi
    fi
    if [[ "$resolver_baseline_changed" == "true" ]]; then
      if [[ -n "$expected_changed_baselines" ]]; then
        expected_changed_baselines="${expected_changed_baselines},resolver"
      else
        expected_changed_baselines="resolver"
      fi
    fi

    if [[ -n "$expected_changed_baselines" && "$latest_changed_baselines" != "$expected_changed_baselines" ]]; then
      echo "[migration-kpi-baseline] changelog changed_baselines does not match baseline diff"
      echo "  expected changed_baselines: $expected_changed_baselines"
      echo "  latest changed_baselines: $latest_changed_baselines"
      exit 1
    fi
  fi
else
  echo "[migration-kpi-baseline] no baseline change detected"
fi
