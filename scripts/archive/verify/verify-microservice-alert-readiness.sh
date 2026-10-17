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
Usage: scripts/verify-microservice-alert-readiness.sh

This script starts the full local runnable microservice profile, verifies the
shared rollout/advisory baseline across the four foreground services, and then
stops the stack.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-alert-readiness] %s\n' "$*"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'alert-readiness verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
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

log "Starting full local runnable stack for alert-readiness verification"
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

check_rollout_advisory() {
  local name="$1"
  local port="$2"

  log "Checking ${name} rollout advisory"
  local payload
  payload="$(curl -fsS "http://localhost:${port}/health/rollout")"
  assert_contains "${payload}" '"advisory"' "${name} rollout advisory"
  assert_contains "${payload}" '"status":"' "${name} rollout advisory"
  assert_contains "${payload}" '"ready":' "${name} rollout advisory"
  assert_contains "${payload}" 'rollout_posture_hint:' "${name} rollout advisory"
}

check_rollout_advisory "api-gateway" "${API_GATEWAY_PORT}"
check_rollout_advisory "api-service" "${API_SERVICE_PORT}"
check_rollout_advisory "event-processor" "${EVENT_PROCESSOR_PORT}"
check_rollout_advisory "puller" "${PULLER_PORT}"

log "Microservice alert-readiness baseline passed"
