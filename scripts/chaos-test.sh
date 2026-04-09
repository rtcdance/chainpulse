#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker/docker-compose.microservices.yml}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}"
RPC_SETTLE_SECONDS="${RPC_SETTLE_SECONDS:-30}"
KAFKA_SETTLE_SECONDS="${KAFKA_SETTLE_SECONDS:-20}"
DB_SETTLE_SECONDS="${DB_SETTLE_SECONDS:-20}"

API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-8081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}"
PULLER_PORT="${PULLER_PORT:-8083}"
API_GATEWAY_AUTH_HEADER="${API_GATEWAY_AUTH_HEADER:-}"
API_SERVICE_AUTH_HEADER="${API_SERVICE_AUTH_HEADER:-}"
EVENT_PROCESSOR_AUTH_HEADER="${EVENT_PROCESSOR_AUTH_HEADER:-}"
PULLER_AUTH_HEADER="${PULLER_AUTH_HEADER:-}"

usage() {
  cat <<'EOF'
Usage: scripts/chaos-test.sh

Runs the current repository-local chaos baseline against the docker-compose
microservice stack by simulating:
  1. RPC node failure (anvil stop/start)
  2. Kafka failure (kafka stop/start)
  3. PostgreSQL failure (postgres stop/start)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[chaos-test] %s\n' "$*"
}

docker_compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

wait_for_http() {
  local label="$1"
  local url="$2"
  local auth_header="${3:-}"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    if [[ -n "${auth_header}" ]]; then
      if curl -fsS -H "${auth_header}" "${url}" >/dev/null 2>&1; then
        log "Ready: ${label}"
        return 0
      fi
    elif curl -fsS "${url}" >/dev/null 2>&1; then
      log "Ready: ${label}"
      return 0
    fi
    sleep 2
  done

  echo "timed out waiting for ${label}: ${url}" >&2
  exit 1
}

wait_for_json_condition() {
  local label="$1"
  local url="$2"
  local expression="$3"
  local auth_header="${4:-}"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    local payload=""
    if [[ -n "${auth_header}" ]]; then
      payload="$(curl -fsS -H "${auth_header}" "${url}" 2>/dev/null)" || payload=""
    else
      payload="$(curl -fsS "${url}" 2>/dev/null)" || payload=""
    fi
    if [[ -n "${payload}" ]]; then
      if python3 - "$payload" "$expression" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
expression = sys.argv[2]
if eval(expression, {"__builtins__": {}}, {"payload": payload}):
    raise SystemExit(0)
raise SystemExit(1)
PY
      then
        log "Observed: ${label}"
        return 0
      fi
    fi
    sleep 2
  done

  echo "timed out waiting for condition: ${label}" >&2
  exit 1
}

assert_metric_present() {
  local label="$1"
  local url="$2"
  local needle="$3"
  local auth_header="${4:-}"
  local payload
  if [[ -n "${auth_header}" ]]; then
    payload="$(curl -fsS -H "${auth_header}" "${url}")"
  else
    payload="$(curl -fsS "${url}")"
  fi
  if [[ "${payload}" != *"${needle}"* ]]; then
    printf 'chaos test failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

cleanup() {
  local code=$?
  log "Stopping compose stack"
  (
    cd "${ROOT_DIR}"
    docker_compose down -v >/dev/null 2>&1 || true
  )
  exit "${code}"
}

trap cleanup EXIT INT TERM

log "Starting compose stack and baseline readiness"
(
  cd "${ROOT_DIR}"
  KEEP_STACK_UP=1 \
  WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS}" \
  API_GATEWAY_PORT="${API_GATEWAY_PORT}" \
  API_SERVICE_PORT="${API_SERVICE_PORT}" \
  EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
  PULLER_PORT="${PULLER_PORT}" \
  COMPOSE_FILE="${COMPOSE_FILE}" \
  bash scripts/verify-docker-compose-microservices-readiness.sh
)

log "Experiment 1/3: RPC failure via anvil stop"
(
  cd "${ROOT_DIR}"
  docker_compose stop anvil >/dev/null
)
sleep "${RPC_SETTLE_SECONDS}"
assert_metric_present "puller metrics after rpc failure" "http://localhost:${PULLER_PORT}/metrics" "puller_poll_errors" "${PULLER_AUTH_HEADER}"
(
  cd "${ROOT_DIR}"
  docker_compose start anvil >/dev/null
)
wait_for_http "anvil health" "http://localhost:8545"
sleep 5

log "Experiment 2/3: Kafka failure via kafka stop"
(
  cd "${ROOT_DIR}"
  docker_compose stop kafka >/dev/null
)
sleep "${KAFKA_SETTLE_SECONDS}"
wait_for_json_condition \
  "event-processor degraded after kafka failure" \
  "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary" \
  'payload.get("runtime_posture") in {"runtime-wired-degraded", "runtime-wired-unhealthy"}' \
  "${EVENT_PROCESSOR_AUTH_HEADER}"
(
  cd "${ROOT_DIR}"
  docker_compose start kafka >/dev/null
)
wait_for_json_condition \
  "event-processor recovered after kafka restart" \
  "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary" \
  'payload.get("runtime_posture") == "runtime-wired"' \
  "${EVENT_PROCESSOR_AUTH_HEADER}"

log "Experiment 3/3: PostgreSQL failure via postgres stop"
(
  cd "${ROOT_DIR}"
  docker_compose stop postgres >/dev/null
)
sleep "${DB_SETTLE_SECONDS}"
wait_for_json_condition \
  "api-service degraded after postgres failure" \
  "http://localhost:${API_SERVICE_PORT}/runtime/summary" \
  'payload.get("query", {}).get("status") in {"degraded", "unhealthy"}' \
  "${API_SERVICE_AUTH_HEADER}"
(
  cd "${ROOT_DIR}"
  docker_compose start postgres >/dev/null
)
wait_for_json_condition \
  "api-service recovered after postgres restart" \
  "http://localhost:${API_SERVICE_PORT}/runtime/summary" \
  'payload.get("query", {}).get("status") == "healthy"' \
  "${API_SERVICE_AUTH_HEADER}"

log "Chaos baseline passed"
