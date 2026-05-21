#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_ACCEPTANCE_LOG_FN="log"
source "${ROOT_DIR}/scripts/lib/docker_acceptance.sh"
COMMAND="${1:-all}"
COMPOSE_FILE="${COMPOSE_FILE:-docker/docker-compose.microservices.yml}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}"
TEMP_DOCKER_CONFIG=""
ANVIL_IMAGE="${ANVIL_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"
ANVIL_IMAGE_PULL_RETRIES="${ANVIL_IMAGE_PULL_RETRIES:-3}"
export ANVIL_IMAGE

API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-8081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}"
PULLER_PORT="${PULLER_PORT:-8083}"
PROM_URL="${PROM_URL:-http://localhost:9090}"
ANVIL_RPC_URL="${ANVIL_RPC_URL:-http://127.0.0.1:8545}"

usage() {
  cat <<'EOF'
Usage: scripts/run-docker-acceptance.sh [up|accept|all|ps|down]

One-click Docker microservice stack management and acceptance entrypoint.

Commands:
  up      start the compose microservice stack and wait for runtime endpoints
  accept  run runnable verification and Prometheus live smoke against the
          currently running stack
  all     run up, then accept
  ps      show compose service status
  down    stop the compose stack and remove volumes

Environment variables:
  COMPOSE_FILE              default: docker/docker-compose.microservices.yml
  WAIT_TIMEOUT_SECONDS      default: 180
  ANVIL_IMAGE               default: ghcr.io/foundry-rs/foundry:latest
  ANVIL_IMAGE_PULL_RETRIES  default: 3
  API_GATEWAY_PORT          default: 8080
  API_SERVICE_PORT          default: 8081
  EVENT_PROCESSOR_PORT      default: 8082
  PULLER_PORT               default: 8083
  PROM_URL                  default: http://localhost:9090
  ANVIL_RPC_URL             default: http://127.0.0.1:8545
EOF
}

if [[ "${COMMAND}" == "-h" || "${COMMAND}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[run-docker-acceptance] %s\n' "$*"
}

cleanup() {
  if [[ -n "${TEMP_DOCKER_CONFIG}" && -d "${TEMP_DOCKER_CONFIG}" ]]; then
    rm -rf "${TEMP_DOCKER_CONFIG}"
  fi
}

trap cleanup EXIT INT TERM

verify_compose_contract() {
  (
    cd "${ROOT_DIR}"
    COMPOSE_FILE="${COMPOSE_FILE}" bash scripts/verify-docker-compose-stack.sh >/dev/null
  )
}

up_stack() {
  verify_compose_contract
  docker_acceptance_pull_anvil_image_with_retry
  log "Starting compose stack from ${COMPOSE_FILE}"
  (
    cd "${ROOT_DIR}"
    docker_acceptance_compose -f "${COMPOSE_FILE}" up -d --build
  )

  docker_acceptance_wait_for_evm_rpc "anvil rpc" "${ANVIL_RPC_URL}"
  docker_acceptance_wait_for_http "api-service /health" "http://localhost:${API_SERVICE_PORT}/health"
  docker_acceptance_wait_for_http "api-service /runtime/summary" "http://localhost:${API_SERVICE_PORT}/runtime/summary"
  docker_acceptance_wait_for_http "api-gateway /health" "http://localhost:${API_GATEWAY_PORT}/health"
  docker_acceptance_wait_for_http "api-gateway /runtime/summary" "http://localhost:${API_GATEWAY_PORT}/runtime/summary"
  docker_acceptance_wait_for_http "event-processor /health" "http://localhost:${EVENT_PROCESSOR_PORT}/health"
  docker_acceptance_wait_for_http "event-processor /runtime/summary" "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary"
  docker_acceptance_wait_for_http "puller /health" "http://localhost:${PULLER_PORT}/health"
  docker_acceptance_wait_for_http "puller /runtime/summary" "http://localhost:${PULLER_PORT}/runtime/summary"
  log "Compose stack is up"
}

accept_stack() {
  log "Running runnable acceptance against running compose stack"
  (
    cd "${ROOT_DIR}"
    API_GATEWAY_PORT="${API_GATEWAY_PORT}" \
    API_SERVICE_PORT="${API_SERVICE_PORT}" \
    EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
    PULLER_PORT="${PULLER_PORT}" \
    bash scripts/verify-local-runnable-app.sh --profile full
  )

  log "Running Prometheus live smoke"
  (
    cd "${ROOT_DIR}"
    PROM_URL="${PROM_URL}" \
    bash scripts/verify-prometheus-live-smoke.sh
  )

  log "Docker acceptance passed"
}

ps_stack() {
  (
    cd "${ROOT_DIR}"
    docker_acceptance_compose -f "${COMPOSE_FILE}" ps
  )
}

down_stack() {
  log "Stopping compose stack from ${COMPOSE_FILE}"
  (
    cd "${ROOT_DIR}"
    docker_acceptance_compose -f "${COMPOSE_FILE}" down -v
  )
}

docker_acceptance_require_docker
docker_acceptance_check_docker_credential_helper

case "${COMMAND}" in
  up)
    up_stack
    ;;
  accept)
    accept_stack
    ;;
  all)
    up_stack
    accept_stack
    ;;
  ps)
    ps_stack
    ;;
  down)
    down_stack
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
