#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

API_GATEWAY_URL="${API_GATEWAY_URL:-http://localhost:8080}"
API_SERVICE_URL="${API_SERVICE_URL:-http://localhost:8081}"
EVENT_PROCESSOR_URL="${EVENT_PROCESSOR_URL:-http://localhost:8082}"
PULLER_URL="${PULLER_URL:-http://localhost:8083}"

DURATION_SECONDS="${DURATION_SECONDS:-1800}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-60}"
RUN_PRECHECK="${RUN_PRECHECK:-0}"
RUN_POSTCHECK="${RUN_POSTCHECK:-0}"
SOAK_LABEL="${SOAK_LABEL:-production}"

usage() {
  cat <<'EOF'
Usage: scripts/soak-check.sh

Runs the repository soak-check gate for a deployed ChainPulse environment.

Environment variables:
  DURATION_SECONDS      total soak window in seconds (default: 1800)
  INTERVAL_SECONDS      sample interval in seconds (default: 60)
  RUN_PRECHECK          set to 1 to run scripts/verify-production.sh before soak
  RUN_POSTCHECK         set to 1 to run scripts/verify-production.sh after soak
  SOAK_LABEL            environment label for reporting (default: production)
  API_GATEWAY_URL       default: http://localhost:8080
  API_SERVICE_URL       default: http://localhost:8081
  EVENT_PROCESSOR_URL   default: http://localhost:8082
  PULLER_URL            default: http://localhost:8083

This script does not generate synthetic load. It repeatedly samples health and
runtime contract endpoints across the soak window and fails if stability
regresses.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[soak-check] %s\n' "$*"
}

fail() {
  printf 'soak check failed: %s\n' "$*" >&2
  exit 1
}

assert_positive_integer() {
  local name="$1"
  local value="$2"
  if ! [[ "${value}" =~ ^[0-9]+$ ]] || [[ "${value}" -le 0 ]]; then
    fail "${name} must be a positive integer"
  fi
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

sample_service() {
  local base_url="$1"
  local health_path="$2"
  local runtime_path="$3"
  local service="$4"

  local health
  local runtime
  health="$(fetch "${base_url}${health_path}")"
  runtime="$(fetch "${base_url}${runtime_path}")"

  assert_contains "${health}" '"status":"healthy"' "${service} ${health_path}"
  assert_contains "${runtime}" "\"service\":\"${service}\"" "${service} ${runtime_path}"
  assert_contains "${runtime}" '"deployment_mode":"microservice"' "${service} ${runtime_path}"
}

sample_window() {
  sample_service "${API_GATEWAY_URL}" "/health" "/runtime/summary" "api-gateway"
  gateway_runtime="$(fetch "${API_GATEWAY_URL}/runtime/summary")"
  assert_contains "${gateway_runtime}" '"runtime_mode":"runtime-wired"' "api-gateway /runtime/summary"
  assert_contains "${gateway_runtime}" '"security_posture":"gateway-security-ready"' "api-gateway /runtime/summary"

  sample_service "${API_SERVICE_URL}" "/health" "/runtime/summary" "api-service"
  api_runtime="$(fetch "${API_SERVICE_URL}/runtime/summary")"
  assert_contains "${api_runtime}" '"security_posture":"api-service-security-ready"' "api-service /runtime/summary"

  sample_service "${EVENT_PROCESSOR_URL}" "/health" "/runtime/summary" "event-processor"
  processor_runtime="$(fetch "${EVENT_PROCESSOR_URL}/runtime/summary")"
  assert_contains "${processor_runtime}" '"runtime_mode":"runtime-wired"' "event-processor /runtime/summary"

  sample_service "${PULLER_URL}" "/health" "/runtime/summary" "puller"
  puller_runtime="$(fetch "${PULLER_URL}/runtime/summary")"
  assert_contains "${puller_runtime}" '"runtime_mode":"runtime-wired"' "puller /runtime/summary"
}

assert_positive_integer "DURATION_SECONDS" "${DURATION_SECONDS}"
assert_positive_integer "INTERVAL_SECONDS" "${INTERVAL_SECONDS}"
assert_toggle "RUN_PRECHECK" "${RUN_PRECHECK}"
assert_toggle "RUN_POSTCHECK" "${RUN_POSTCHECK}"

if [[ "${INTERVAL_SECONDS}" -gt "${DURATION_SECONDS}" ]]; then
  fail "INTERVAL_SECONDS must be less than or equal to DURATION_SECONDS"
fi

samples=$(( (DURATION_SECONDS + INTERVAL_SECONDS - 1) / INTERVAL_SECONDS ))

log "Starting soak check"
log "Environment: ${SOAK_LABEL}"
log "Duration seconds: ${DURATION_SECONDS}"
log "Interval seconds: ${INTERVAL_SECONDS}"
log "Planned samples: ${samples}"

if [[ "${RUN_PRECHECK}" == "1" ]]; then
  log "Running pre-soak production verification"
  run_verify_production
fi

for ((sample=1; sample<=samples; sample++)); do
  log "Running sample ${sample}/${samples}"
  sample_window
  if [[ "${sample}" -lt "${samples}" ]]; then
    sleep "${INTERVAL_SECONDS}"
  fi
done

if [[ "${RUN_POSTCHECK}" == "1" ]]; then
  log "Running post-soak production verification"
  run_verify_production
fi

log "Soak check passed"
