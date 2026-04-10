#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMMAND="${1:-all}"
OVERLAY="${OVERLAY:-microservice}"
NAMESPACE="${NAMESPACE:-chainpulse}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}"

usage() {
  cat <<'EOF'
Usage: scripts/run-k8s-deploy.sh [up|down|status|accept|all]

One-click Kubernetes deployment entrypoint.

Commands:
  up      apply k8s overlay and wait for deployment rollout
  down    delete k8s overlay resources
  status  show pods/services/deployments in namespace
  accept  run k8s capability acceptance checks
  all     run up, accept, then status

Environment variables:
  OVERLAY               default: microservice (microservice|monolithic)
  NAMESPACE             default: chainpulse
  WAIT_TIMEOUT_SECONDS  default: 180
EOF
}

if [[ "${COMMAND}" == "-h" || "${COMMAND}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[run-k8s-deploy] %s\n' "$*"
}

fail() {
  printf '[run-k8s-deploy] ERROR: %s\n' "$*" >&2
  exit 1
}

require_kubectl() {
  if ! command -v kubectl >/dev/null 2>&1; then
    fail "kubectl not found"
  fi
}

require_context() {
  if ! kubectl config current-context >/dev/null 2>&1; then
    fail "kube context is not set"
  fi
}

require_overlay() {
  if [[ "${OVERLAY}" != "microservice" && "${OVERLAY}" != "monolithic" ]]; then
    fail "unsupported OVERLAY=${OVERLAY}, expected microservice or monolithic"
  fi
}

kustomize_dir() {
  printf '%s/k8s/overlays/%s' "${ROOT_DIR}" "${OVERLAY}"
}

wait_rollout() {
  local deps=(
    "postgres"
    "redis"
    "zookeeper"
    "kafka"
  )

  local app_dep=""
  if [[ "${OVERLAY}" == "microservice" ]]; then
    app_dep="chainpulse-microservice"
  else
    app_dep="chainpulse-monolithic"
  fi

  local dep
  for dep in "${deps[@]}"; do
    log "Waiting rollout: deployment/${dep}"
    kubectl rollout status "deployment/${dep}" -n "${NAMESPACE}" --timeout="${WAIT_TIMEOUT_SECONDS}s" >/dev/null
  done

  log "Waiting rollout: deployment/${app_dep}"
  kubectl rollout status "deployment/${app_dep}" -n "${NAMESPACE}" --timeout="${WAIT_TIMEOUT_SECONDS}s" >/dev/null
}

up() {
  require_kubectl
  require_context
  require_overlay
  log "Applying overlay: $(kustomize_dir)"
  kubectl apply -k "$(kustomize_dir)"
  wait_rollout
  log "K8s deploy up finished"
}

down() {
  require_kubectl
  require_context
  require_overlay
  log "Deleting overlay: $(kustomize_dir)"
  kubectl delete -k "$(kustomize_dir)" --ignore-not-found=true
  log "K8s deploy down finished"
}

status() {
  require_kubectl
  require_context
  log "Namespace: ${NAMESPACE}"
  kubectl get pods -n "${NAMESPACE}" -o wide
  kubectl get svc -n "${NAMESPACE}"
  kubectl get deployments -n "${NAMESPACE}"
}

accept() {
  log "Running k8s acceptance entrypoint"
  (
    cd "${ROOT_DIR}"
    make k8s-acceptance
  )
}

all() {
  up
  accept
  status
}

case "${COMMAND}" in
  up)
    up
    ;;
  down)
    down
    ;;
  status)
    status
    ;;
  accept)
    accept
    ;;
  all)
    all
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
