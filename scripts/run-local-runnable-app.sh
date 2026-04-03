#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROFILE="minimal"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-45}"
LOG_DIR="${LOG_DIR:-/tmp/chainpulse-local-runnable-app}"
KEEP_LOGS="${KEEP_LOGS:-1}"

API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-8081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}"
PULLER_PORT="${PULLER_PORT:-8083}"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-chainpulse}"
DB_PASSWORD="${DB_PASSWORD:-password}"
DB_NAME="${DB_NAME:-chainpulse}"
REDIS_CLUSTER="${REDIS_CLUSTER:-localhost:6379}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
BLOCKCHAIN_RPCS="${BLOCKCHAIN_RPCS:-http://localhost:8545}"
LOG_LEVEL="${LOG_LEVEL:-info}"

PIDS=()
STARTED_SERVICES=()
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
RUN_LOG_DIR="${LOG_DIR}/${TIMESTAMP}"

usage() {
  cat <<'EOF'
Usage: scripts/run-local-runnable-app.sh [--profile minimal|full] [--wait-seconds N]

Profiles:
  minimal  Starts api-service + api-gateway
  full     Starts api-service + api-gateway + event-processor + puller

Environment overrides:
  API_GATEWAY_PORT
  API_SERVICE_PORT
  EVENT_PROCESSOR_PORT
  PULLER_PORT
  DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME
  REDIS_CLUSTER
  KAFKA_BROKERS
  BLOCKCHAIN_RPCS
  LOG_LEVEL
  LOG_DIR
  KEEP_LOGS=0 to remove logs on clean exit
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="${2:-}"
      shift 2
      ;;
    --wait-seconds)
      WAIT_TIMEOUT_SECONDS="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ "${PROFILE}" != "minimal" && "${PROFILE}" != "full" ]]; then
  echo "Unsupported profile: ${PROFILE}" >&2
  usage
  exit 1
fi

mkdir -p "${RUN_LOG_DIR}"

log() {
  printf '[local-app] %s\n' "$*"
}

service_log_path() {
  local service="$1"
  printf '%s/%s.log' "${RUN_LOG_DIR}" "${service}"
}

cleanup() {
  local code=$?
  if [[ ${#PIDS[@]} -gt 0 ]]; then
    log "Stopping services..."
    for pid in "${PIDS[@]}"; do
      if kill -0 "${pid}" >/dev/null 2>&1; then
        kill "${pid}" >/dev/null 2>&1 || true
      fi
    done
    wait >/dev/null 2>&1 || true
  fi

  if [[ "${KEEP_LOGS}" == "0" && -d "${RUN_LOG_DIR}" ]]; then
    rm -rf "${RUN_LOG_DIR}"
  else
    log "Logs kept in ${RUN_LOG_DIR}"
  fi

  exit "${code}"
}

trap cleanup EXIT INT TERM

start_service() {
  local service="$1"
  shift

  local log_file
  log_file="$(service_log_path "${service}")"
  log "Starting ${service}..."

  (
    cd "${ROOT_DIR}"
    env "$@" >"${log_file}" 2>&1
  ) &

  local pid=$!
  PIDS+=("${pid}")
  STARTED_SERVICES+=("${service}")
  log "  ${service} pid=${pid} log=${log_file}"
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      log "Ready: ${name} -> ${url}"
      return 0
    fi
    sleep 1
  done

  log "Timeout waiting for ${name}: ${url}"
  return 1
}

run_smoke() {
  local gateway_base="http://localhost:${API_GATEWAY_PORT}"

  log "Running focused smoke checks..."
  curl -fsS "${gateway_base}/runtime/summary" >/dev/null
  curl -fsS "${gateway_base}/events?limit=5" >/dev/null
  log "Smoke checks passed"
}

print_dependency_hints() {
  log "Using local-first dependency defaults:"
  log "  PostgreSQL: ${DB_HOST}:${DB_PORT}/${DB_NAME}"
  log "  Redis:      ${REDIS_CLUSTER}"
  log "  Kafka:      ${KAFKA_BROKERS}"
  if [[ "${PROFILE}" == "full" ]]; then
    log "  RPC:        ${BLOCKCHAIN_RPCS}"
  fi
}

print_next_steps() {
  local gateway_base="http://localhost:${API_GATEWAY_PORT}"
  local service_base="http://localhost:${API_SERVICE_PORT}"

  printf '\n'
  log "Local runnable app is up"
  log "  api-service:     ${service_base}"
  log "  api-gateway:     ${gateway_base}"
  if [[ "${PROFILE}" == "full" ]]; then
    log "  event-processor: http://localhost:${EVENT_PROCESSOR_PORT}"
    log "  puller:          http://localhost:${PULLER_PORT}"
  fi
  printf '\n'
  log "Try:"
  log "  curl ${service_base}/runtime/summary"
  log "  curl ${gateway_base}/runtime/summary"
  log "  curl ${gateway_base}/events?limit=5"
  log "  bash scripts/verify-local-runnable-app.sh --profile ${PROFILE}"
  if [[ "${PROFILE}" == "full" ]]; then
    log "  curl http://localhost:${EVENT_PROCESSOR_PORT}/runtime/control"
    log "  curl http://localhost:${PULLER_PORT}/runtime/control"
  fi
  printf '\n'
  log "Press Ctrl+C to stop all started services"
}

print_dependency_hints

start_service "api-service" \
  SERVICE_PORT="${API_SERVICE_PORT}" \
  INSTANCE_ID="api-service-1" \
  DB_HOST="${DB_HOST}" \
  DB_PORT="${DB_PORT}" \
  DB_USER="${DB_USER}" \
  DB_PASSWORD="${DB_PASSWORD}" \
  DB_NAME="${DB_NAME}" \
  REDIS_CLUSTER="${REDIS_CLUSTER}" \
  KAFKA_BROKERS="${KAFKA_BROKERS}" \
  LOG_LEVEL="${LOG_LEVEL}" \
  go run ./cmd/microservices/api-service

start_service "api-gateway" \
  GATEWAY_PORT="${API_GATEWAY_PORT}" \
  INSTANCE_ID="api-gateway-1" \
  GATEWAY_UPSTREAM_SERVICES="http://localhost:${API_SERVICE_PORT}" \
  LOG_LEVEL="${LOG_LEVEL}" \
  go run ./cmd/microservices/api-gateway

if [[ "${PROFILE}" == "full" ]]; then
  start_service "event-processor" \
    PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
    INSTANCE_ID="event-processor-1" \
    KAFKA_BROKERS="${KAFKA_BROKERS}" \
    KAFKA_CONSUMER_GROUP="event-processor-consumers" \
    KAFKA_INPUT_TOPICS="raw-events,blockchain-events" \
    KAFKA_OUTPUT_TOPICS="processed-events,indexed-events" \
    DB_HOST="${DB_HOST}" \
    DB_PORT="${DB_PORT}" \
    DB_USER="${DB_USER}" \
    DB_PASSWORD="${DB_PASSWORD}" \
    DB_NAME="${DB_NAME}" \
    REDIS_CLUSTER="${REDIS_CLUSTER}" \
    LOG_LEVEL="${LOG_LEVEL}" \
    go run ./cmd/microservices/event-processor

  start_service "puller" \
    PULLER_PORT="${PULLER_PORT}" \
    INSTANCE_ID="puller-1" \
    KAFKA_BROKERS="${KAFKA_BROKERS}" \
    KAFKA_PRODUCER_GROUP="data-puller-producers" \
    KAFKA_OUTPUT_TOPICS="raw-events,blockchain-events" \
    BLOCKCHAIN_RPCS="${BLOCKCHAIN_RPCS}" \
    LOG_LEVEL="${LOG_LEVEL}" \
    go run ./cmd/microservices/puller
fi

wait_for_http "api-service health" "http://localhost:${API_SERVICE_PORT}/health"
wait_for_http "api-service runtime summary" "http://localhost:${API_SERVICE_PORT}/runtime/summary"
wait_for_http "api-gateway health" "http://localhost:${API_GATEWAY_PORT}/health"
wait_for_http "api-gateway runtime summary" "http://localhost:${API_GATEWAY_PORT}/runtime/summary"

if [[ "${PROFILE}" == "full" ]]; then
  wait_for_http "event-processor health" "http://localhost:${EVENT_PROCESSOR_PORT}/health"
  wait_for_http "puller health" "http://localhost:${PULLER_PORT}/health"
fi

run_smoke
print_next_steps

while true; do
  sleep 5
done
