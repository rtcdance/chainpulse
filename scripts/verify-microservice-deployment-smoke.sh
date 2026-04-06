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
Usage: scripts/verify-microservice-deployment-smoke.sh

This script starts the full local runnable microservice profile, verifies the
cross-entrypoint deployment smoke, and then stops the stack.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-deployment-smoke] %s\n' "$*"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'deployment smoke failed: %s missing "%s"\n' "${label}" "${needle}" >&2
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

log "Starting full local runnable stack"
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

log "Checking gateway deployment posture"
gateway_summary="$(curl -fsS "http://localhost:${API_GATEWAY_PORT}/runtime/summary")"
assert_contains "${gateway_summary}" '"query_bridge_posture":"query-bridge-ready"' "api-gateway runtime summary"
assert_contains "${gateway_summary}" '"upstream_query_health_state":"query-upstream-healthy"' "api-gateway runtime summary"

log "Checking api-service deployment posture"
service_summary="$(curl -fsS "http://localhost:${API_SERVICE_PORT}/runtime/summary")"
assert_contains "${service_summary}" '"service":"api-service"' "api-service runtime summary"
assert_contains "${service_summary}" '"query"' "api-service runtime summary"

log "Checking event-processor deployment posture"
processor_summary="$(curl -fsS "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary")"
assert_contains "${processor_summary}" '"service":"event-processor"' "event-processor runtime summary"
assert_contains "${processor_summary}" '"processor"' "event-processor runtime summary"

log "Checking puller deployment posture"
puller_summary="$(curl -fsS "http://localhost:${PULLER_PORT}/runtime/summary")"
assert_contains "${puller_summary}" '"service":"puller"' "puller runtime summary"
assert_contains "${puller_summary}" '"metrics"' "puller runtime summary"

log "Microservice deployment smoke passed"
