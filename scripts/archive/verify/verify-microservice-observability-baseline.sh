#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-60}"
API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-8081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}"
PULLER_PORT="${PULLER_PORT:-8083}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-microservice-observability-baseline.sh

This script starts the full local runnable microservice profile, verifies the
shared observability baseline across the four foreground services, and then
stops the stack.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-observability] %s\n' "$*"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'observability verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

RUN_PID=""

cleanup() {
  if [[ -n "${RUN_PID}" ]] && kill -0 "${RUN_PID}" >/dev/null 2>&1; then
    kill "${RUN_PID}" >/dev/null 2>&1 || true
    wait "${RUN_PID}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

log "Starting full local runnable stack for observability verification"
(
  cd "${ROOT_DIR}"
  WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS}" \
  API_GATEWAY_PORT="${API_GATEWAY_PORT}" \
  API_SERVICE_PORT="${API_SERVICE_PORT}" \
  EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
  PULLER_PORT="${PULLER_PORT}" \
  bash scripts/run-local-runnable-app.sh --profile full
) &
RUN_PID=$!

log "Waiting for shared full-profile verification"
(
  cd "${ROOT_DIR}"
  WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS}" \
  API_GATEWAY_PORT="${API_GATEWAY_PORT}" \
  API_SERVICE_PORT="${API_SERVICE_PORT}" \
  EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
  PULLER_PORT="${PULLER_PORT}" \
  bash scripts/verify-local-runnable-app.sh --profile full
)

check_service_observability() {
  local name="$1"
  local port="$2"
  local summary_path="$3"
  local rollout_path="$4"
  local summary_expected="$5"

  log "Checking ${name} metrics route"
  local metrics_payload
  metrics_payload="$(curl -fsS "http://localhost:${port}/metrics")"
  assert_contains "${metrics_payload}" "# TYPE " "${name} metrics"

  log "Checking ${name} runtime summary"
  local summary_payload
  summary_payload="$(curl -fsS "http://localhost:${port}${summary_path}")"
  assert_contains "${summary_payload}" '"collector_state":"available"' "${name} runtime summary"
  assert_contains "${summary_payload}" "${summary_expected}" "${name} runtime summary"

  log "Checking ${name} rollout advisory"
  local rollout_payload
  rollout_payload="$(curl -fsS "http://localhost:${port}${rollout_path}")"
  assert_contains "${rollout_payload}" '"advisory"' "${name} rollout report"
}

check_service_observability "api-gateway" "${API_GATEWAY_PORT}" "/runtime/summary" "/health/rollout" '"upstream_query_health_state":"query-upstream-healthy"'
check_service_observability "api-service" "${API_SERVICE_PORT}" "/runtime/summary" "/health/rollout" '"execution_summary"'
check_service_observability "event-processor" "${EVENT_PROCESSOR_PORT}" "/runtime/summary" "/health/rollout" '"processor"'
check_service_observability "puller" "${PULLER_PORT}" "/runtime/summary" "/health/rollout" '"execution_summary"'

log "Microservice observability baseline passed"
