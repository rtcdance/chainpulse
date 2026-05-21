#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROVIDER="${PROVIDER:-docker}"
COMMAND="${1:-all}"
EXPECT_API="${EXPECT_API:-1}"
RUN_H5_ACCEPTANCE="${RUN_H5_ACCEPTANCE:-1}"
PLAYWRIGHT_CMD="${PLAYWRIGHT_CMD:-npm test}"

usage() {
  cat <<'EOF'
Usage: scripts/run-deploy-event-acceptance.sh [deploy|event|api|h5|all]

One-click entrypoint for deploy -> real Event injection -> API/H5 acceptance.

Commands:
  deploy  run only the deployment step for the selected provider
  event   inject a real on-chain event against the deployed stack
  api     run API/runtime acceptance checks against the deployed stack
  h5      run Playwright H5 acceptance against the deployed stack
  all     run deploy, event, api, then h5

Environment variables:
  PROVIDER           default: docker (docker|k8s)
  EXPECT_API         default: 1
  RUN_H5_ACCEPTANCE  default: 1 (set 0 to skip Playwright)
  PLAYWRIGHT_CMD     default: npm test
  RPC_URL            forwarded to real-event acceptance
  API_URL            forwarded to real-event acceptance / Playwright
  API_GATEWAY_URL    default: API_URL or http://localhost:8080
  API_SERVICE_URL    default: http://localhost:8081
  PROMETHEUS_URL     default: PROM_URL or http://localhost:9090
  OVERLAY            forwarded to k8s deploy flow
  NAMESPACE          forwarded to k8s deploy flow
EOF
}

log() {
  printf '[run-deploy-event-acceptance] %s\n' "$*"
}

fail() {
  printf '[run-deploy-event-acceptance] ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "${COMMAND}" == "-h" || "${COMMAND}" == "--help" ]]; then
  usage
  exit 0
fi

require_provider() {
  case "${PROVIDER}" in
    docker|k8s)
      ;;
    *)
      fail "unsupported PROVIDER=${PROVIDER}, expected docker or k8s"
      ;;
  esac
}

deploy_stack() {
  case "${PROVIDER}" in
    docker)
      log "Deploying docker stack"
      (
        cd "${ROOT_DIR}"
        bash scripts/run-docker-acceptance.sh up
      )
      ;;
    k8s)
      log "Deploying k8s stack"
      (
        cd "${ROOT_DIR}"
        OVERLAY="${OVERLAY:-microservice}" \
        NAMESPACE="${NAMESPACE:-chainpulse}" \
        WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}" \
        bash scripts/run-k8s-deploy.sh up
      )
      ;;
  esac
}

inject_event() {
  log "Injecting deployed real event"
  (
    cd "${ROOT_DIR}"
    EXPECT_API="${EXPECT_API}" \
    RPC_URL="${RPC_URL:-http://127.0.0.1:8545}" \
    API_URL="${API_URL:-http://127.0.0.1:8080}" \
    bash scripts/run-deployed-real-event-acceptance.sh
  )
}

run_api_acceptance() {
  case "${PROVIDER}" in
    docker)
      log "Running docker API acceptance"
      (
        cd "${ROOT_DIR}"
        API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}" \
        API_SERVICE_PORT="${API_SERVICE_PORT:-8081}" \
        EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}" \
        PULLER_PORT="${PULLER_PORT:-8083}" \
        PROM_URL="${PROM_URL:-http://localhost:9090}" \
        bash scripts/run-docker-acceptance.sh accept
      )
      ;;
    k8s)
      log "Running k8s API acceptance"
      (
        cd "${ROOT_DIR}"
        OVERLAY="${OVERLAY:-microservice}" \
        NAMESPACE="${NAMESPACE:-chainpulse}" \
        bash scripts/run-k8s-deploy.sh accept
      )
      ;;
  esac
}

run_h5_acceptance() {
  if [[ "${RUN_H5_ACCEPTANCE}" != "1" ]]; then
    log "Skipping H5 acceptance because RUN_H5_ACCEPTANCE=${RUN_H5_ACCEPTANCE}"
    return
  fi

  log "Running H5 Playwright acceptance"
  (
    cd "${ROOT_DIR}"
    API_URL="${API_URL:-http://localhost:8080}" \
    API_GATEWAY_URL="${API_GATEWAY_URL:-${API_URL:-http://localhost:8080}}" \
    API_SERVICE_URL="${API_SERVICE_URL:-http://localhost:8081}" \
    PROMETHEUS_URL="${PROMETHEUS_URL:-${PROM_URL:-http://localhost:9090}}" \
    sh -c "${PLAYWRIGHT_CMD}"
  )
}

require_provider

case "${COMMAND}" in
  deploy)
    deploy_stack
    ;;
  event)
    inject_event
    ;;
  api)
    run_api_acceptance
    ;;
  h5)
    run_h5_acceptance
    ;;
  all)
    deploy_stack
    inject_event
    run_api_acceptance
    run_h5_acceptance
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
