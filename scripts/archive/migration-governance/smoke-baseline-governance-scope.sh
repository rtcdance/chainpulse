#!/usr/bin/env bash

set -euo pipefail

log_prefix="[baseline-scope-smoke]"
output_dir="${CHAINPULSE_BASELINE_SCOPE_SMOKE_OUTPUT_DIR:-build/migration-governance}"
json_output="${CHAINPULSE_BASELINE_SCOPE_SMOKE_JSON_OUTPUT:-$output_dir/baseline-scope-smoke.json}"
prom_output="${CHAINPULSE_BASELINE_SCOPE_SMOKE_PROM_OUTPUT:-$output_dir/baseline-scope-smoke.prom}"
md_output="${CHAINPULSE_BASELINE_SCOPE_SMOKE_MD_OUTPUT:-$output_dir/baseline-scope-smoke.md}"
generated_at_utc="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
total_cases=0
passed_cases=0
failed_cases=0
results_json_items=()

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$log_prefix missing required command: $1"
    exit 1
  fi
}

require_cmd git

family_for_case() {
  local case_name="$1"
  case "$case_name" in
    *template*)
      echo "template"
      ;;
    preflight_*|custom_resolver_path_preflight_*)
      echo "preflight"
      ;;
    guarded_update_*)
      echo "update"
      ;;
    custom_resolver_*)
      echo "custom-path"
      ;;
    *)
      echo "scope"
      ;;
  esac
}

prepend_changelog_entry() {
  local changelog_file="$1"
  local entry="$2"
  local tmp_file
  tmp_file="$(mktemp)"
  {
    echo "$entry"
    cat "$changelog_file"
  } > "$tmp_file"
  mv "$tmp_file" "$changelog_file"
}

setup_repo() {
  local repo_dir="$1"
  mkdir -p "$repo_dir/scripts/lib" "$repo_dir/docs/operations"

  cp scripts/check-migration-changelog-quality.sh "$repo_dir/scripts/"
  cp scripts/check-migration-baseline-governance.sh "$repo_dir/scripts/"
  cp scripts/preflight-migration-baseline-update.sh "$repo_dir/scripts/"
  cp scripts/update-migration-governance-baseline.sh "$repo_dir/scripts/"
  cp scripts/export-migration-governance-kpi.sh "$repo_dir/scripts/"
  cp scripts/test-baseline-update-resolver.sh "$repo_dir/scripts/"
  cp scripts/lib/baseline_update_resolver.sh "$repo_dir/scripts/lib/"
  chmod +x "$repo_dir/scripts/check-migration-changelog-quality.sh"
  chmod +x "$repo_dir/scripts/check-migration-baseline-governance.sh"
  chmod +x "$repo_dir/scripts/preflight-migration-baseline-update.sh"
  chmod +x "$repo_dir/scripts/update-migration-governance-baseline.sh"
  chmod +x "$repo_dir/scripts/export-migration-governance-kpi.sh"
  chmod +x "$repo_dir/scripts/test-baseline-update-resolver.sh"

  cat > "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom" <<'EOF'
# baseline
chainpulse_migration_items_total 1
EOF

  cat > "$repo_dir/docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom" <<'EOF'
# baseline
chainpulse_migration_ticket_registry_fallback_events_total{reason="none",failure_mode="enforce"} 0
EOF

  cat > "$repo_dir/docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom" <<'EOF'
# baseline
chainpulse_baseline_scope_smoke_status 1
EOF

  cat > "$repo_dir/docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom" <<'EOF'
# baseline
chainpulse_baseline_resolver_test_status 1
EOF

  cat > "$repo_dir/docs/operations/MIGRATION_TICKET_REGISTRY.txt" <<'EOF'
PHASE31-BASELINE
EOF

  cat > "$repo_dir/docs/operations/MIGRATION_MANIFEST.csv" <<'EOF'
phase,id,component,status,owner,severity,eta,notes
31,PHASE31-BASELINE,governance,active,platform-team,medium,2026-04-01,baseline smoke fixture
EOF

  cat > "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md" <<'EOF'
# Migration Governance Baseline Changelog

- 2026-03-30T00:00:00Z | ticket=PHASE31-BASELINE | owner=platform-team | scope=dual | changed_baselines=kpi,health,smoke | rationale=initial baseline
EOF

  (
    cd "$repo_dir"
    git init -q
    git config user.email "chainpulse-smoke@example.com"
    git config user.name "ChainPulse Smoke"
    git add .
    git commit -q -m "init baseline fixtures"
  )
}

run_case_expect_success() {
  local case_name="$1"
  local scope="$2"
  local mutate_mode="$3"
  local override_changed_baselines="${4:-}"
  local changed_baselines=""
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  case "$mutate_mode" in
    dual)
      echo "chainpulse_migration_items_total 2" >> "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom"
      echo "chainpulse_migration_ticket_registry_fallback_events_total{reason=\"none\",failure_mode=\"enforce\"} 1" >> "$repo_dir/docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom"
      changed_baselines="kpi,health"
      ;;
    kpi-only)
      echo "chainpulse_migration_items_total 2" >> "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom"
      changed_baselines="kpi"
      ;;
    health-only)
      echo "chainpulse_migration_ticket_registry_fallback_events_total{reason=\"none\",failure_mode=\"enforce\"} 1" >> "$repo_dir/docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom"
      changed_baselines="health"
      ;;
    resolver-only)
      echo "chainpulse_baseline_resolver_test_status 0" >> "$repo_dir/docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom"
      changed_baselines="resolver"
      ;;
    *)
      echo "$log_prefix invalid mutate mode: $mutate_mode"
      rm -rf "$repo_dir"
      exit 1
      ;;
  esac

  if [[ -n "$override_changed_baselines" ]]; then
    changed_baselines="$override_changed_baselines"
  fi

  prepend_changelog_entry \
    "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md" \
    "- 2026-03-30T00:10:00Z | ticket=PHASE31-BASELINE | owner=platform-team | scope=${scope} | changed_baselines=${changed_baselines} | rationale=${case_name}"

  (
    cd "$repo_dir"
    git add .
    git commit -q -m "case ${case_name}"
    CHAINPULSE_MIGRATION_BASELINE_DIFF_REF=HEAD~1 \
    CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern \
    CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=true \
    CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true \
    ./scripts/check-migration-baseline-governance.sh >/dev/null
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass: $case_name"
}

run_case_expect_failure() {
  local case_name="$1"
  local scope="$2"
  local mutate_mode="$3"
  local override_changed_baselines="${4:-}"
  local changed_baselines=""
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  case "$mutate_mode" in
    kpi-only)
      echo "chainpulse_migration_items_total 2" >> "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom"
      changed_baselines="kpi"
      ;;
    health-only)
      echo "chainpulse_migration_ticket_registry_fallback_events_total{reason=\"none\",failure_mode=\"enforce\"} 1" >> "$repo_dir/docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom"
      changed_baselines="health"
      ;;
    dual)
      echo "chainpulse_migration_items_total 2" >> "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom"
      echo "chainpulse_migration_ticket_registry_fallback_events_total{reason=\"none\",failure_mode=\"enforce\"} 1" >> "$repo_dir/docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom"
      changed_baselines="kpi,health"
      ;;
    resolver-only)
      echo "chainpulse_baseline_resolver_test_status 0" >> "$repo_dir/docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom"
      changed_baselines="resolver"
      ;;
    *)
      echo "$log_prefix invalid mutate mode: $mutate_mode"
      rm -rf "$repo_dir"
      exit 1
      ;;
  esac

  if [[ -n "$override_changed_baselines" ]]; then
    changed_baselines="$override_changed_baselines"
  fi

  prepend_changelog_entry \
    "$repo_dir/docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md" \
    "- 2026-03-30T00:10:00Z | ticket=PHASE31-BASELINE | owner=platform-team | scope=${scope} | changed_baselines=${changed_baselines} | rationale=${case_name}"

  (
    cd "$repo_dir"
    git add .
    git commit -q -m "case ${case_name}"
    if CHAINPULSE_MIGRATION_BASELINE_DIFF_REF=HEAD~1 \
      CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern \
      CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=true \
      CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true \
      ./scripts/check-migration-baseline-governance.sh >/dev/null 2>&1; then
      echo "$log_prefix expected failure but passed: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_preflight_case_expect_success() {
  local case_name="$1"
  local refresh_resolver_flag="$2"
  local expected_changed_baselines="$3"
  local expect_resolver_line="$4"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE="$refresh_resolver_flag" \
      ./scripts/preflight-migration-baseline-update.sh build/preflight.md >/dev/null

    if ! grep -Fq "| changed_baselines | ${expected_changed_baselines} |" build/preflight.md; then
      echo "$log_prefix expected changed_baselines not found in preflight: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    if [[ "$expect_resolver_line" == "true" ]]; then
      if ! grep -Fq "Resolver baseline: \`docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom\`" build/preflight.md; then
        echo "$log_prefix expected resolver baseline target missing: $case_name"
        failed_cases=$((failed_cases + 1))
        results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
        exit 1
      fi
    else
      if grep -Fq "Resolver baseline: \`docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom\`" build/preflight.md; then
        echo "$log_prefix unexpected resolver baseline target present: $case_name"
        failed_cases=$((failed_cases + 1))
        results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
        exit 1
      fi
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass: $case_name"
}

run_preflight_custom_path_no_refresh_case_expect_success() {
  local case_name="$1"
  local custom_resolver_baseline="docs/operations/custom/PREFLIGHT_RESOLVER_BASELINE.prom"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=false \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      ./scripts/preflight-migration-baseline-update.sh build/preflight.md >/dev/null

    if grep -Fq "Resolver baseline:" build/preflight.md; then
      echo "$log_prefix preflight unexpectedly leaked resolver target with refresh disabled: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass: $case_name"
}

run_preflight_custom_path_manual_changed_baselines_case_expect_success() {
  local case_name="$1"
  local custom_resolver_baseline="docs/operations/custom/MANUAL_OVERRIDE_RESOLVER_BASELINE.prom"
  local manual_changed_baselines="kpi,resolver"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=false \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES="$manual_changed_baselines" \
      ./scripts/preflight-migration-baseline-update.sh build/preflight.md >/dev/null

    if ! grep -Fq "| changed_baselines | ${manual_changed_baselines} |" build/preflight.md; then
      echo "$log_prefix preflight did not preserve manual changed_baselines override: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    if grep -Fq "Resolver baseline:" build/preflight.md; then
      echo "$log_prefix preflight unexpectedly leaked resolver target with refresh disabled under manual override: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass: $case_name"
}

run_preflight_custom_path_invalid_changed_baselines_case_expect_failure() {
  local case_name="$1"
  local custom_resolver_baseline="docs/operations/custom/INVALID_OVERRIDE_RESOLVER_BASELINE.prom"
  local invalid_changed_baselines="kpi,invalid"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=false \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES="$invalid_changed_baselines" \
      ./scripts/preflight-migration-baseline-update.sh build/preflight.md >/dev/null 2>&1; then
      echo "$log_prefix expected preflight failure for invalid manual changed_baselines override: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_update_case_expect_success() {
  local case_name="$1"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    CHAINPULSE_ALLOW_BASELINE_UPDATE=true \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      ./scripts/update-migration-governance-baseline.sh >/dev/null

    if ! git diff --name-only HEAD | grep -Fxq "docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom"; then
      echo "$log_prefix resolver baseline was not updated: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    if ! grep -E '^\- .*changed_baselines=kpi,resolver .*rationale='"${case_name}"'$' docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md >/dev/null; then
      echo "$log_prefix expected changelog entry not found: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    git add .
    git commit -q -m "apply ${case_name}"

    CHAINPULSE_MIGRATION_BASELINE_DIFF_REF=HEAD~1 \
      CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern \
      CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=true \
      CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true \
      ./scripts/check-migration-baseline-governance.sh >/dev/null
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass: $case_name"
}

run_update_case_expect_failure() {
  local case_name="$1"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=false \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected update blocking but command passed: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if ! git diff --quiet -- docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom; then
      echo "$log_prefix resolver baseline changed despite blocked update: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if grep -Fq "$case_name" docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md; then
      echo "$log_prefix changelog mutated despite blocked update: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_custom_resolver_path_case_expect_success() {
  local case_name="$1"
  local custom_resolver_baseline="docs/operations/custom/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  mkdir -p "$repo_dir/docs/operations/custom"
  cp "$repo_dir/docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom" \
    "$repo_dir/$custom_resolver_baseline"

  (
    cd "$repo_dir"
    CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      ./scripts/preflight-migration-baseline-update.sh build/preflight.md >/dev/null

    if ! grep -Fq "Resolver baseline: \`$custom_resolver_baseline\`" build/preflight.md; then
      echo "$log_prefix expected custom resolver baseline target missing: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    CHAINPULSE_ALLOW_BASELINE_UPDATE=true \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null

    if ! grep -Fq "chainpulse_baseline_resolver_test_total" "$custom_resolver_baseline"; then
      echo "$log_prefix custom resolver baseline content missing expected metrics: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    if grep -Fq "chainpulse_baseline_resolver_test_total" "docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom"; then
      echo "$log_prefix default resolver baseline unexpectedly mutated for custom path case: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"fail\"}")
      exit 1
    fi

    git add .
    git commit -q -m "apply ${case_name}"

    CHAINPULSE_MIGRATION_BASELINE_DIFF_REF=HEAD~1 \
      CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern \
      CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=true \
      CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      ./scripts/check-migration-baseline-governance.sh >/dev/null
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"success\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass: $case_name"
}

run_custom_resolver_path_missing_case_expect_failure() {
  local case_name="$1"
  local missing_custom_resolver_baseline="docs/operations/custom/MISSING_RESOLVER_BASELINE.prom"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    prepend_changelog_entry \
      "docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md" \
      "- 2026-03-30T00:20:00Z | ticket=PHASE31-BASELINE | owner=platform-team | scope=kpi-only | changed_baselines=resolver | rationale=${case_name}"
    git add .
    git commit -q -m "case ${case_name}"

    if CHAINPULSE_MIGRATION_BASELINE_DIFF_REF=HEAD~1 \
      CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern \
      CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=true \
      CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$missing_custom_resolver_baseline" \
      ./scripts/check-migration-baseline-governance.sh >/dev/null 2>&1; then
      echo "$log_prefix expected governance failure for missing custom resolver baseline path: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_custom_resolver_path_blocked_update_case_expect_failure() {
  local case_name="$1"
  local blocked_custom_resolver_baseline="docs/operations/custom/BLOCKED_RESOLVER_BASELINE.prom"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=false \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$blocked_custom_resolver_baseline" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected blocked update failure for custom path but command passed: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if [[ -f "$blocked_custom_resolver_baseline" ]]; then
      echo "$log_prefix blocked update unexpectedly created custom resolver baseline file: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if grep -Fq "$case_name" docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md; then
      echo "$log_prefix blocked update unexpectedly mutated changelog for custom path case: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_custom_resolver_path_blocked_update_template_side_effect_case_expect_failure() {
  local case_name="$1"
  local blocked_custom_resolver_baseline="docs/operations/custom/BLOCKED_TEMPLATE_RESOLVER_BASELINE.prom"
  local custom_template_output="build/custom/blocked-custom-path-template.md"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=false \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$blocked_custom_resolver_baseline" \
      CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT="$custom_template_output" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected blocked custom-path update failure with template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if [[ -f "$custom_template_output" ]]; then
      echo "$log_prefix blocked custom-path update unexpectedly created template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_custom_resolver_path_invalid_changed_baselines_template_side_effect_case_expect_failure() {
  local case_name="$1"
  local custom_resolver_baseline="docs/operations/custom/INVALID_TEMPLATE_RESOLVER_BASELINE.prom"
  local custom_template_output="build/custom/invalid-custom-path-template.md"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=true \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE="$custom_resolver_baseline" \
      CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES="kpi,invalid" \
      CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT="$custom_template_output" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected invalid changed_baselines custom-path update failure with template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if [[ -f "$custom_template_output" ]]; then
      echo "$log_prefix invalid changed_baselines custom-path failure unexpectedly created template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_update_invalid_changed_baselines_case_expect_failure() {
  local case_name="$1"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=true \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=false \
      CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true \
      CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES="kpi,invalid" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected guarded update failure for invalid changed_baselines override: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if ! git diff --quiet -- docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom; then
      echo "$log_prefix resolver baseline changed despite invalid changed_baselines failure: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if grep -Fq "$case_name" docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md; then
      echo "$log_prefix changelog mutated despite invalid changed_baselines failure: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_update_blocked_template_side_effect_case_expect_failure() {
  local case_name="$1"
  local custom_template_output="build/custom/blocked-template.md"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=false \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT="$custom_template_output" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected blocked update failure with custom template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if [[ -f "$custom_template_output" ]]; then
      echo "$log_prefix blocked update unexpectedly created custom template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_update_invalid_changed_baselines_template_side_effect_case_expect_failure() {
  local case_name="$1"
  local custom_template_output="build/custom/invalid-template.md"
  local repo_dir
  repo_dir="$(mktemp -d)"
  total_cases=$((total_cases + 1))

  setup_repo "$repo_dir"

  (
    cd "$repo_dir"
    if CHAINPULSE_ALLOW_BASELINE_UPDATE=true \
      CHAINPULSE_BASELINE_UPDATE_TICKET=PHASE31-BASELINE \
      CHAINPULSE_BASELINE_UPDATE_OWNER=platform-team \
      CHAINPULSE_BASELINE_UPDATE_RATIONALE="${case_name}" \
      CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES="kpi,invalid" \
      CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT="$custom_template_output" \
      ./scripts/update-migration-governance-baseline.sh >/dev/null 2>&1; then
      echo "$log_prefix expected invalid changed_baselines failure with custom template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi

    if [[ -f "$custom_template_output" ]]; then
      echo "$log_prefix invalid changed_baselines failure unexpectedly created custom template output: $case_name"
      failed_cases=$((failed_cases + 1))
      results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"fail\"}")
      exit 1
    fi
  )

  passed_cases=$((passed_cases + 1))
  results_json_items+=("{\"name\":\"${case_name}\",\"expected\":\"failure\",\"result\":\"pass\"}")
  rm -rf "$repo_dir"
  echo "$log_prefix pass (expected failure): $case_name"
}

run_case_expect_success "dual_scope_alignment" "dual" "dual"
run_case_expect_success "kpi_scope_alignment" "kpi-only" "kpi-only"
run_case_expect_success "health_scope_alignment" "health-only" "health-only"
run_case_expect_success "resolver_changed_baselines_alignment" "kpi-only" "resolver-only"
run_case_expect_failure "scope_mismatch_should_fail" "dual" "kpi-only"
run_case_expect_failure "changed_baselines_mismatch_should_fail" "kpi-only" "kpi-only" "health"
run_case_expect_failure "resolver_changed_baselines_mismatch_should_fail" "kpi-only" "resolver-only" "kpi,resolver"
run_preflight_case_expect_success "preflight_without_resolver_refresh" "false" "kpi,health,smoke" "false"
run_preflight_case_expect_success "preflight_with_resolver_refresh" "true" "kpi,health,smoke,resolver" "true"
run_preflight_custom_path_no_refresh_case_expect_success "custom_resolver_path_preflight_no_refresh_should_not_show_target"
run_preflight_custom_path_manual_changed_baselines_case_expect_success "custom_resolver_path_preflight_manual_changed_baselines_override"
run_preflight_custom_path_invalid_changed_baselines_case_expect_failure "custom_resolver_path_preflight_invalid_changed_baselines_should_fail"
run_update_case_expect_success "guarded_update_with_resolver_refresh"
run_update_case_expect_failure "guarded_update_blocked_without_allow_flag"
run_update_invalid_changed_baselines_case_expect_failure "guarded_update_invalid_changed_baselines_should_fail"
run_update_blocked_template_side_effect_case_expect_failure "guarded_update_blocked_custom_template_should_not_be_created"
run_update_invalid_changed_baselines_template_side_effect_case_expect_failure "guarded_update_invalid_changed_baselines_custom_template_should_not_be_created"
run_custom_resolver_path_case_expect_success "custom_resolver_baseline_path_parity"
run_custom_resolver_path_missing_case_expect_failure "custom_resolver_baseline_path_missing_should_fail"
run_custom_resolver_path_blocked_update_case_expect_failure "custom_resolver_baseline_path_blocked_update_should_not_create_file"
run_custom_resolver_path_blocked_update_template_side_effect_case_expect_failure "custom_resolver_path_blocked_update_custom_template_should_not_be_created"
run_custom_resolver_path_invalid_changed_baselines_template_side_effect_case_expect_failure "custom_resolver_path_invalid_changed_baselines_custom_template_should_not_be_created"

mkdir -p "$output_dir"

status="pass"
if (( failed_cases > 0 )); then
  status="fail"
fi

results_joined=""
if (( ${#results_json_items[@]} > 0 )); then
  results_joined="$(IFS=,; echo "${results_json_items[*]}")"
fi

family_scope_total=0
family_scope_pass=0
family_scope_fail=0
family_preflight_total=0
family_preflight_pass=0
family_preflight_fail=0
family_update_total=0
family_update_pass=0
family_update_fail=0
family_custom_path_total=0
family_custom_path_pass=0
family_custom_path_fail=0
family_template_total=0
family_template_pass=0
family_template_fail=0

for item in "${results_json_items[@]}"; do
  case_name="$(echo "$item" | sed -E 's/.*"name":"([^"]+)".*/\1/')"
  result="$(echo "$item" | sed -E 's/.*"result":"([^"]+)".*/\1/')"
  family="$(family_for_case "$case_name")"
  case "$family" in
    scope)
      family_scope_total=$((family_scope_total + 1))
      if [[ "$result" == "pass" ]]; then
        family_scope_pass=$((family_scope_pass + 1))
      else
        family_scope_fail=$((family_scope_fail + 1))
      fi
      ;;
    preflight)
      family_preflight_total=$((family_preflight_total + 1))
      if [[ "$result" == "pass" ]]; then
        family_preflight_pass=$((family_preflight_pass + 1))
      else
        family_preflight_fail=$((family_preflight_fail + 1))
      fi
      ;;
    update)
      family_update_total=$((family_update_total + 1))
      if [[ "$result" == "pass" ]]; then
        family_update_pass=$((family_update_pass + 1))
      else
        family_update_fail=$((family_update_fail + 1))
      fi
      ;;
    custom-path)
      family_custom_path_total=$((family_custom_path_total + 1))
      if [[ "$result" == "pass" ]]; then
        family_custom_path_pass=$((family_custom_path_pass + 1))
      else
        family_custom_path_fail=$((family_custom_path_fail + 1))
      fi
      ;;
    template)
      family_template_total=$((family_template_total + 1))
      if [[ "$result" == "pass" ]]; then
        family_template_pass=$((family_template_pass + 1))
      else
        family_template_fail=$((family_template_fail + 1))
      fi
      ;;
  esac
done

{
  echo "{"
  echo "  \"generated_at_utc\": \"${generated_at_utc}\","
  echo "  \"status\": \"${status}\","
  echo "  \"total_cases\": ${total_cases},"
  echo "  \"passed_cases\": ${passed_cases},"
  echo "  \"failed_cases\": ${failed_cases},"
  echo "  \"results\": [${results_joined}]"
  echo "}"
} > "$json_output"

{
  echo "# HELP chainpulse_baseline_scope_smoke_total Baseline scope smoke test cases."
  echo "# TYPE chainpulse_baseline_scope_smoke_total gauge"
  echo "chainpulse_baseline_scope_smoke_total ${total_cases}"
  echo
  echo "# HELP chainpulse_baseline_scope_smoke_passed_total Baseline scope smoke passed cases."
  echo "# TYPE chainpulse_baseline_scope_smoke_passed_total gauge"
  echo "chainpulse_baseline_scope_smoke_passed_total ${passed_cases}"
  echo
  echo "# HELP chainpulse_baseline_scope_smoke_failed_total Baseline scope smoke failed cases."
  echo "# TYPE chainpulse_baseline_scope_smoke_failed_total gauge"
  echo "chainpulse_baseline_scope_smoke_failed_total ${failed_cases}"
  echo
  echo "# HELP chainpulse_baseline_scope_smoke_status Baseline scope smoke status (1=pass,0=fail)."
  echo "# TYPE chainpulse_baseline_scope_smoke_status gauge"
  if [[ "$status" == "pass" ]]; then
    echo "chainpulse_baseline_scope_smoke_status 1"
  else
    echo "chainpulse_baseline_scope_smoke_status 0"
  fi
} > "$prom_output"

{
  echo "# Baseline Scope Smoke"
  echo
  echo "- Generated At (UTC): $generated_at_utc"
  echo "- Status: \`$status\`"
  echo
  echo "| Field | Value |"
  echo "|---|---:|"
  echo "| total_cases | $total_cases |"
  echo "| passed_cases | $passed_cases |"
  echo "| failed_cases | $failed_cases |"
  echo
  if (( failed_cases > 0 )); then
    echo "## Failure Summary"
    echo
    echo "| Case | Expected | Result |"
    echo "|---|---|---|"
    for item in "${results_json_items[@]}"; do
      case_name="$(echo "$item" | sed -E 's/.*"name":"([^"]+)".*/\1/')"
      expected="$(echo "$item" | sed -E 's/.*"expected":"([^"]+)".*/\1/')"
      result="$(echo "$item" | sed -E 's/.*"result":"([^"]+)".*/\1/')"
      if [[ "$result" == "fail" ]]; then
        echo "| $case_name | $expected | $result |"
      fi
    done
    echo
  fi
  echo "## Case Results"
  echo
  echo "| Case | Expected | Result |"
  echo "|---|---|---|"
  for item in "${results_json_items[@]}"; do
    case_name="$(echo "$item" | sed -E 's/.*"name":"([^"]+)".*/\1/')"
    expected="$(echo "$item" | sed -E 's/.*"expected":"([^"]+)".*/\1/')"
    result="$(echo "$item" | sed -E 's/.*"result":"([^"]+)".*/\1/')"
    echo "| $case_name | $expected | $result |"
  done
  echo
  echo "## Family Summary"
  echo
  echo "| Family | Total | Passed | Failed |"
  echo "|---|---:|---:|---:|"
  echo "| scope | $family_scope_total | $family_scope_pass | $family_scope_fail |"
  echo "| preflight | $family_preflight_total | $family_preflight_pass | $family_preflight_fail |"
  echo "| update | $family_update_total | $family_update_pass | $family_update_fail |"
  echo "| custom-path | $family_custom_path_total | $family_custom_path_pass | $family_custom_path_fail |"
  echo "| template | $family_template_total | $family_template_pass | $family_template_fail |"
} > "$md_output"

echo "$log_prefix outputs:"
echo "$log_prefix   - $json_output"
echo "$log_prefix   - $prom_output"
echo "$log_prefix   - $md_output"
echo "$log_prefix all scenarios passed"
