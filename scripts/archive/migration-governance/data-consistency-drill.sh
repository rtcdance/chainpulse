#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PULLER_URL="${PULLER_URL:-http://localhost:8083}"
TARGET_CHAIN="${TARGET_CHAIN:-}"
RUN_PRECHECK="${RUN_PRECHECK:-0}"
RUN_POSTCHECK="${RUN_POSTCHECK:-0}"
EXPECTED_COVERAGE_POSTURE="${EXPECTED_COVERAGE_POSTURE:-}"
EXPECTED_REORG_STATE="${EXPECTED_REORG_STATE:-}"

usage() {
  cat <<'EOF'
Usage: TARGET_CHAIN=<chain> scripts/data-consistency-drill.sh

Runs the repository data-consistency / replay / reorg recovery drill gate for a
deployed ChainPulse environment.

Required environment variables:
  TARGET_CHAIN               chain identifier expected in puller checkpoint summary

Optional environment variables:
  PULLER_URL                 default: http://localhost:8083
  RUN_PRECHECK               set to 1 to run scripts/verify-production.sh before drill
  RUN_POSTCHECK              set to 1 to run scripts/verify-production.sh after drill
  EXPECTED_COVERAGE_POSTURE  optional expected checkpoint coverage posture
  EXPECTED_REORG_STATE       optional expected reorg checkpoint state
  API_GATEWAY_URL            forwarded to scripts/verify-production.sh
  API_SERVICE_URL            forwarded to scripts/verify-production.sh
  EVENT_PROCESSOR_URL        forwarded to scripts/verify-production.sh

This script does not synthesize a blockchain reorg or replay by itself. It
validates the puller runtime checkpoint/reorg contract before and after an
operator-driven drill window so the release team can capture consistency
evidence.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[data-consistency-drill] %s\n' "$*"
}

fail() {
  printf 'data consistency drill failed: %s\n' "$*" >&2
  exit 1
}

assert_toggle() {
  local name="$1"
  local value="$2"
  if [[ "${value}" != "0" && "${value}" != "1" ]]; then
    fail "${name} must be 0 or 1"
  fi
}

fetch() {
  local url="$1"
  curl -fsS "${url}"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    fail "${label} missing \"${needle}\""
  fi
}

run_verify_production() {
  (
    cd "${ROOT_DIR}"
    bash scripts/verify-production.sh
  )
}

validate_puller_runtime() {
  local payload="$1"
  local stage="$2"

  assert_contains "${payload}" '"service":"puller"' "${stage} puller runtime"
  assert_contains "${payload}" '"deployment_mode":"microservice"' "${stage} puller runtime"
  assert_contains "${payload}" '"runtime_mode":"runtime-wired"' "${stage} puller runtime"
  assert_contains "${payload}" '"checkpoint_coverage_posture":"' "${stage} puller runtime"
  assert_contains "${payload}" '"checkpoint_chain_summary":"' "${stage} puller runtime"
  assert_contains "${payload}" "\"${TARGET_CHAIN}=" "${stage} puller runtime"

  if [[ -n "${EXPECTED_COVERAGE_POSTURE}" ]]; then
    assert_contains "${payload}" "\"checkpoint_coverage_posture\":\"${EXPECTED_COVERAGE_POSTURE}\"" "${stage} puller runtime"
  fi

  if [[ -n "${EXPECTED_REORG_STATE}" ]]; then
    assert_contains "${payload}" "\"reorg_checkpoint_state\":\"${EXPECTED_REORG_STATE}\"" "${stage} puller runtime"
  fi
}

if [[ -z "${TARGET_CHAIN}" ]]; then
  fail "missing required environment variable TARGET_CHAIN"
fi

assert_toggle "RUN_PRECHECK" "${RUN_PRECHECK}"
assert_toggle "RUN_POSTCHECK" "${RUN_POSTCHECK}"

if [[ "${RUN_PRECHECK}" == "1" ]]; then
  log "Running production precheck"
  run_verify_production
fi

log "Capturing pre-drill puller runtime summary"
pre_payload="$(fetch "${PULLER_URL}/runtime/summary")"
validate_puller_runtime "${pre_payload}" "pre-drill"

cat <<EOF

Consistency drill execution sequence
1. Record the current puller/runtime evidence for ${TARGET_CHAIN}.
2. Execute the operator-driven replay, checkpoint recovery, or reorg handling
   drill in the target environment.
3. Capture:
   - drill start/end timestamps
   - chain / block range under evaluation
   - replay or recovery command / pipeline identifier
   - any reorg or duplicate-event evidence observed
4. Re-run this script or continue to the post-drill validation below.

Expected post-drill signals:
  - puller runtime stays deployment_mode=microservice
  - puller runtime stays runtime_mode=runtime-wired
  - checkpoint coverage remains visible for ${TARGET_CHAIN}
  - reorg/checkpoint posture matches the intended drill outcome

EOF

log "Capturing post-drill puller runtime summary"
post_payload="$(fetch "${PULLER_URL}/runtime/summary")"
validate_puller_runtime "${post_payload}" "post-drill"

if [[ "${RUN_POSTCHECK}" == "1" ]]; then
  log "Running production postcheck"
  run_verify_production
fi

log "Data consistency drill gate completed"
