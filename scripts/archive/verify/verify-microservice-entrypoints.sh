#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-30}"
LOG_DIR="${LOG_DIR:-/tmp/chainpulse-microservice-entrypoints}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
RUN_LOG_DIR="${LOG_DIR}/${TIMESTAMP}"

API_GATEWAY_PORT="${API_GATEWAY_PORT:-18080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-18081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-18082}"
PULLER_PORT="${PULLER_PORT:-18083}"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-chainpulse}"
DB_PASSWORD="${DB_PASSWORD:-password}"
DB_NAME="${DB_NAME:-chainpulse}"
REDIS_CLUSTER="${REDIS_CLUSTER:-localhost:6379}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
BLOCKCHAIN_RPCS="${BLOCKCHAIN_RPCS:-http://localhost:8545}"
LOG_LEVEL="${LOG_LEVEL:-info}"

SERVICE_FILTER="all"
PIDS=()

usage() {
  cat <<'EOF'
Usage: scripts/verify-microservice-entrypoints.sh [--service name|all] [--wait-seconds N]

Services:
  api-service
  api-gateway
  event-processor
  puller
  all

This script starts each selected microservice entrypoint independently,
waits for /health and /runtime/summary (or /runtime/control where relevant),
then stops it again.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service)
      SERVICE_FILTER="${2:-}"
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

case "${SERVICE_FILTER}" in
  all|api-service|api-gateway|event-processor|puller) ;;
  *)
    echo "Unsupported service filter: ${SERVICE_FILTER}" >&2
    usage
    exit 1
    ;;
esac

mkdir -p "${RUN_LOG_DIR}"

log() {
  printf '[verify-entrypoints] %s\n' "$*"
}

service_log_path() {
  local service="$1"
  printf '%s/%s.log' "${RUN_LOG_DIR}" "${service}"
}

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    if kill -0 "${pid}" >/dev/null 2>&1; then
      kill "${pid}" >/dev/null 2>&1 || true
    fi
  done
  wait >/dev/null 2>&1 || true
}

trap cleanup EXIT INT TERM

start_service() {
  local service="$1"
  shift

  local log_file
  log_file="$(service_log_path "${service}")"
  log "Starting ${service}"

  (
    cd "${ROOT_DIR}"
    env "$@" >"${log_file}" 2>&1
  ) &

  local pid=$!
  PIDS+=("${pid}")
  log "  pid=${pid} log=${log_file}"
}

wait_for_http() {
  local label="$1"
  local url="$2"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      log "Ready: ${label}"
      return 0
    fi
    sleep 1
  done

  log "Timeout waiting for ${label}: ${url}"
  return 1
}

stop_current_services() {
  cleanup
  PIDS=()
}

should_run() {
  local service="$1"
  [[ "${SERVICE_FILTER}" == "all" || "${SERVICE_FILTER}" == "${service}" ]]
}

verify_api_service() {
  start_service "api-service" \
    SERVICE_PORT="${API_SERVICE_PORT}" \
    INSTANCE_ID="api-service-entrypoint-check" \
    DB_HOST="${DB_HOST}" \
    DB_PORT="${DB_PORT}" \
    DB_USER="${DB_USER}" \
    DB_PASSWORD="${DB_PASSWORD}" \
    DB_NAME="${DB_NAME}" \
    REDIS_CLUSTER="${REDIS_CLUSTER}" \
    KAFKA_BROKERS="${KAFKA_BROKERS}" \
    LOG_LEVEL="${LOG_LEVEL}" \
    go run ./cmd/microservices/api-service

  wait_for_http "api-service /health" "http://localhost:${API_SERVICE_PORT}/health"
  wait_for_http "api-service /runtime/summary" "http://localhost:${API_SERVICE_PORT}/runtime/summary"
  stop_current_services
}

verify_api_gateway() {
  start_service "api-gateway" \
    GATEWAY_PORT="${API_GATEWAY_PORT}" \
    INSTANCE_ID="api-gateway-entrypoint-check" \
    GATEWAY_UPSTREAM_SERVICES="http://localhost:${API_SERVICE_PORT}" \
    LOG_LEVEL="${LOG_LEVEL}" \
    go run ./cmd/microservices/api-gateway

  wait_for_http "api-gateway /health" "http://localhost:${API_GATEWAY_PORT}/health"
  wait_for_http "api-gateway /runtime/summary" "http://localhost:${API_GATEWAY_PORT}/runtime/summary"
  stop_current_services
}

verify_event_processor() {
  start_service "event-processor" \
    PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
    INSTANCE_ID="event-processor-entrypoint-check" \
    DB_HOST="${DB_HOST}" \
    DB_PORT="${DB_PORT}" \
    DB_USER="${DB_USER}" \
    DB_PASSWORD="${DB_PASSWORD}" \
    DB_NAME="${DB_NAME}" \
    REDIS_CLUSTER="${REDIS_CLUSTER}" \
    LOG_LEVEL="${LOG_LEVEL}" \
    go run ./cmd/microservices/event-processor

  wait_for_http "event-processor /health" "http://localhost:${EVENT_PROCESSOR_PORT}/health"
  wait_for_http "event-processor /runtime/summary" "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary"
  wait_for_http "event-processor /runtime/control" "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/control"
  stop_current_services
}

verify_puller() {
  start_service "puller" \
    PULLER_PORT="${PULLER_PORT}" \
    INSTANCE_ID="puller-entrypoint-check" \
    KAFKA_BROKERS="${KAFKA_BROKERS}" \
    BLOCKCHAIN_RPCS="${BLOCKCHAIN_RPCS}" \
    LOG_LEVEL="${LOG_LEVEL}" \
    go run ./cmd/microservices/puller

  wait_for_http "puller /health" "http://localhost:${PULLER_PORT}/health"
  wait_for_http "puller /runtime/summary" "http://localhost:${PULLER_PORT}/runtime/summary"
  wait_for_http "puller /runtime/control" "http://localhost:${PULLER_PORT}/runtime/control"
  stop_current_services
}

log "Logs will be written to ${RUN_LOG_DIR}"

if should_run "api-service"; then
  verify_api_service
fi

if should_run "api-gateway"; then
  verify_api_gateway
fi

if should_run "event-processor"; then
  verify_event_processor
fi

if should_run "puller"; then
  verify_puller
fi

log "Microservice entrypoint verification passed for service=${SERVICE_FILTER}"
