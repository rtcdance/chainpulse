#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
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

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker is not installed or not in PATH" >&2
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    local context
    context="$(docker context show 2>/dev/null || echo unknown)"
    echo "docker daemon is not reachable for context '${context}'; start Docker Desktop or the Docker daemon first" >&2
    echo "expected Docker socket: ${DOCKER_HOST:-unix:///Users/${USER}/.docker/run/docker.sock}" >&2
    exit 1
  fi
}

prepare_temp_docker_config() {
  local source_config current_context
  if [[ -n "${TEMP_DOCKER_CONFIG}" ]]; then
    return 0
  fi

  source_config="${HOME}/.docker"
  TEMP_DOCKER_CONFIG="$(mktemp -d "${TMPDIR:-/tmp}/chainpulse-docker-config.XXXXXX")"

  if [[ -d "${source_config}/contexts" ]]; then
    cp -R "${source_config}/contexts" "${TEMP_DOCKER_CONFIG}/contexts"
  fi
  if [[ -d "${source_config}/cli-plugins" ]]; then
    cp -R "${source_config}/cli-plugins" "${TEMP_DOCKER_CONFIG}/cli-plugins"
  fi

  current_context="$(sed -n 's/.*"currentContext"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${source_config}/config.json" 2>/dev/null | head -n 1)"
  if [[ -n "${current_context}" ]]; then
    cat >"${TEMP_DOCKER_CONFIG}/config.json" <<EOF
{
  "auths": {},
  "currentContext": "${current_context}"
}
EOF
  else
    cat >"${TEMP_DOCKER_CONFIG}/config.json" <<'EOF'
{
  "auths": {}
}
EOF
  fi
}

check_docker_credential_helper() {
  local docker_config helper
  docker_config="${HOME}/.docker/config.json"
  if [[ ! -f "${docker_config}" ]]; then
    return 0
  fi

  helper="$(sed -n 's/.*"credsStore"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${docker_config}" | head -n 1)"
  if [[ -z "${helper}" ]]; then
    return 0
  fi

  if ! command -v "docker-credential-${helper}" >/dev/null 2>&1; then
    log "docker credential helper missing: docker-credential-${helper}; using isolated docker config"
    prepare_temp_docker_config
    return 0
  fi

  if ! "docker-credential-${helper}" list >/dev/null 2>&1; then
    log "docker credential helper unhealthy: docker-credential-${helper}; using isolated docker config"
    prepare_temp_docker_config
  fi
}

docker_compose() {
  if [[ -n "${TEMP_DOCKER_CONFIG}" ]]; then
    env DOCKER_CONFIG="${TEMP_DOCKER_CONFIG}" docker compose "$@"
    return
  fi
  docker compose "$@"
}

pull_anvil_image_with_retry() {
  local attempt=1
  local delay=2

  if docker image inspect "${ANVIL_IMAGE}" >/dev/null 2>&1; then
    log "Found cached anvil image: ${ANVIL_IMAGE}"
    return 0
  fi

  while (( attempt <= ANVIL_IMAGE_PULL_RETRIES )); do
    log "Pre-pulling anvil image (${attempt}/${ANVIL_IMAGE_PULL_RETRIES}): ${ANVIL_IMAGE}"
    if docker pull "${ANVIL_IMAGE}"; then
      log "Pulled anvil image successfully: ${ANVIL_IMAGE}"
      return 0
    fi

    if (( attempt == ANVIL_IMAGE_PULL_RETRIES )); then
      break
    fi

    log "Anvil image pull failed; retrying in ${delay}s"
    sleep "${delay}"
    delay=$((delay * 2))
    attempt=$((attempt + 1))
  done

  echo "failed to pre-pull anvil image after ${ANVIL_IMAGE_PULL_RETRIES} attempts: ${ANVIL_IMAGE}" >&2
  exit 1
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
    sleep 2
  done

  echo "timed out waiting for ${label}: ${url}" >&2
  exit 1
}

verify_compose_contract() {
  (
    cd "${ROOT_DIR}"
    COMPOSE_FILE="${COMPOSE_FILE}" bash scripts/verify-docker-compose-stack.sh >/dev/null
  )
}

up_stack() {
  verify_compose_contract
  pull_anvil_image_with_retry
  log "Starting compose stack from ${COMPOSE_FILE}"
  (
    cd "${ROOT_DIR}"
    docker_compose -f "${COMPOSE_FILE}" up -d --build
  )

  wait_for_http "api-service /health" "http://localhost:${API_SERVICE_PORT}/health"
  wait_for_http "api-service /runtime/summary" "http://localhost:${API_SERVICE_PORT}/runtime/summary"
  wait_for_http "api-gateway /health" "http://localhost:${API_GATEWAY_PORT}/health"
  wait_for_http "api-gateway /runtime/summary" "http://localhost:${API_GATEWAY_PORT}/runtime/summary"
  wait_for_http "event-processor /health" "http://localhost:${EVENT_PROCESSOR_PORT}/health"
  wait_for_http "event-processor /runtime/summary" "http://localhost:${EVENT_PROCESSOR_PORT}/runtime/summary"
  wait_for_http "puller /health" "http://localhost:${PULLER_PORT}/health"
  wait_for_http "puller /runtime/summary" "http://localhost:${PULLER_PORT}/runtime/summary"
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
    docker_compose -f "${COMPOSE_FILE}" ps
  )
}

down_stack() {
  log "Stopping compose stack from ${COMPOSE_FILE}"
  (
    cd "${ROOT_DIR}"
    docker_compose -f "${COMPOSE_FILE}" down -v
  )
}

require_docker
check_docker_credential_helper

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
