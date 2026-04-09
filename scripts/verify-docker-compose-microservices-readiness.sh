#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_ACCEPTANCE_LOG_FN="log"
source "${ROOT_DIR}/scripts/lib/docker_acceptance.sh"
COMPOSE_FILE="${COMPOSE_FILE:-docker/docker-compose.microservices.yml}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}"
KEEP_STACK_UP="${KEEP_STACK_UP:-0}"
TEMP_DOCKER_CONFIG=""
ANVIL_IMAGE="${ANVIL_IMAGE:-ghcr.io/foundry-rs/foundry:latest}"
ANVIL_IMAGE_PULL_RETRIES="${ANVIL_IMAGE_PULL_RETRIES:-3}"
export ANVIL_IMAGE

API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-8081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}"
PULLER_PORT="${PULLER_PORT:-8083}"
PROM_URL="${PROM_URL:-http://localhost:9090}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-docker-compose-microservices-readiness.sh

Starts the docker-compose microservices profile, waits for the four foreground
services to become reachable, verifies the shared runnable baseline, and then
stops the stack unless KEEP_STACK_UP=1.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-compose-readiness] %s\n' "$*"
}

pull_anvil_image_with_retry() {
  docker_acceptance_pull_anvil_image_with_retry
}

cleanup() {
  local code=$?
  if [[ "${KEEP_STACK_UP}" != "1" ]]; then
    log "Stopping compose stack"
    (
      cd "${ROOT_DIR}"
      docker_acceptance_compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
    )
  else
    log "KEEP_STACK_UP=1 so compose stack is left running"
  fi
  if [[ -n "${TEMP_DOCKER_CONFIG}" && -d "${TEMP_DOCKER_CONFIG}" ]]; then
    rm -rf "${TEMP_DOCKER_CONFIG}"
  fi
  exit "${code}"
}

trap cleanup EXIT INT TERM

docker_acceptance_require_docker
docker_acceptance_check_docker_credential_helper
pull_anvil_image_with_retry

log "Checking compose service set before startup"
(
  cd "${ROOT_DIR}"
  COMPOSE_FILE="${COMPOSE_FILE}" bash scripts/verify-docker-compose-stack.sh >/dev/null
)

log "Starting compose stack from ${COMPOSE_FILE}"
(
  cd "${ROOT_DIR}"
  docker_acceptance_compose -f "${COMPOSE_FILE}" up -d --build
)

docker_acceptance_wait_for_http "api-service /health" "http://localhost:${API_SERVICE_PORT}/health"
docker_acceptance_wait_for_http "api-service /runtime/summary" "http://localhost:${API_SERVICE_PORT}/runtime/summary"
docker_acceptance_wait_for_http "api-gateway /health" "http://localhost:${API_GATEWAY_PORT}/health"
docker_acceptance_wait_for_http "api-gateway /runtime/summary" "http://localhost:${API_GATEWAY_PORT}/runtime/summary"
docker_acceptance_wait_for_http "event-processor /health" "http://localhost:${EVENT_PROCESSOR_PORT}/health"
docker_acceptance_wait_for_http "event-processor /runtime/summary" "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary"
docker_acceptance_wait_for_http "puller /health" "http://localhost:${PULLER_PORT}/health"
docker_acceptance_wait_for_http "puller /runtime/summary" "http://localhost:${PULLER_PORT}/runtime/summary"

log "Running full runnable verification against compose stack"
(
  cd "${ROOT_DIR}"
  API_GATEWAY_PORT="${API_GATEWAY_PORT}" \
  API_SERVICE_PORT="${API_SERVICE_PORT}" \
  EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT}" \
  PULLER_PORT="${PULLER_PORT}" \
  bash scripts/verify-local-runnable-app.sh --profile full
)

log "Running Prometheus live smoke against compose stack"
(
  cd "${ROOT_DIR}"
  PROM_URL="${PROM_URL}" \
  bash scripts/verify-prometheus-live-smoke.sh
)

log "Docker-compose microservices readiness smoke passed"
