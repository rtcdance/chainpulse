#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${MODE:-monolithic}"
NAMESPACE="${NAMESPACE:-chainpulse}"

usage() {
  cat <<'EOF'
Usage: scripts/deploy-staging.sh [--monolithic|--microservice]

Deploys ChainPulse to the staging Kubernetes environment.

Options:
  --monolithic     deploy monolithic variant (default)
  --microservice   deploy microservice variant

Environment variables:
  NAMESPACE        Kubernetes namespace (default: chainpulse)

Prerequisites:
  - kubectl configured with a staging cluster context
  - K8s overlay files present under k8s/overlays/
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "${1:-}" == "--microservice" ]]; then
  MODE="microservice"
elif [[ "${1:-}" == "--monolithic" ]]; then
  MODE="monolithic"
elif [[ -n "${1:-}" ]]; then
  printf 'Unknown option: %s\n' "$1" >&2
  usage >&2
  exit 1
fi

log() {
  printf '[deploy-staging] %s\n' "$*"
}

fail() {
  printf '[deploy-staging] ERROR: %s\n' "$*" >&2
  exit 1
}

log "Deploying staging environment (mode=${MODE})"

if ! command -v kubectl >/dev/null 2>&1; then
  fail "kubectl not found in PATH"
fi

KUSTOMIZE_DIR="${ROOT_DIR}/k8s/overlays/${MODE}"
if [[ ! -d "${KUSTOMIZE_DIR}" ]]; then
  fail "kustomize directory not found: ${KUSTOMIZE_DIR}"
fi

log "Applying overlay: ${KUSTOMIZE_DIR}"
kubectl apply -k "${KUSTOMIZE_DIR}"

log "Waiting for deployment rollout..."
for dep in postgres redis zookeeper kafka; do
  if kubectl get deployment "${dep}" -n "${NAMESPACE}" >/dev/null 2>&1; then
    log "Waiting rollout: deployment/${dep}"
    kubectl rollout status "deployment/${dep}" -n "${NAMESPACE}" --timeout=300s
  fi
done

if kubectl get deployment chainpulse-${MODE} -n "${NAMESPACE}" >/dev/null 2>&1; then
  log "Waiting rollout: deployment/chainpulse-${MODE}"
  kubectl rollout status "deployment/chainpulse-${MODE}" -n "${NAMESPACE}" --timeout=300s
fi

log "Staging deployment completed"

log "Running staging verification..."
if [[ -x "${ROOT_DIR}/scripts/verify-staging.sh" ]]; then
  bash "${ROOT_DIR}/scripts/verify-staging.sh"
else
  log "scripts/verify-staging.sh not found, skipping"
fi