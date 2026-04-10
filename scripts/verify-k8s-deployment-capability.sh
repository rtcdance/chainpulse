#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${MODE:-auto}"
STRICT_CLUSTER_DRY_RUN="${STRICT_CLUSTER_DRY_RUN:-0}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-k8s-deployment-capability.sh [--mode auto|static|cluster-dry-run]

Modes:
  auto            Run static checks; if kubectl exists, also run cluster dry-run checks.
  static          Run filesystem/kustomization-reference checks only.
  cluster-dry-run Run kubectl kustomize + kubectl apply --dry-run=client checks.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "${1:-}" == "--mode" ]]; then
  MODE="${2:-}"
fi

log() {
  printf '[verify-k8s] %s\n' "$*"
}

fail() {
  printf '[verify-k8s] ERROR: %s\n' "$*" >&2
  exit 1
}

assert_file() {
  local path="$1"
  if [[ ! -f "${ROOT_DIR}/${path}" ]]; then
    fail "missing required file: ${path}"
  fi
}

verify_static() {
  log "running static checks"

  local required_files=(
    "k8s/base/kustomization.yaml"
    "k8s/overlays/monolithic/kustomization.yaml"
    "k8s/overlays/microservice/kustomization.yaml"
    "k8s/namespace.yaml"
    "k8s/configmap.yaml"
    "k8s/postgres-deployment.yaml"
    "k8s/redis-deployment.yaml"
    "k8s/kafka-deployment.yaml"
    "k8s/chainpulse-monolithic-deployment.yaml"
    "k8s/chainpulse-microservice-deployment.yaml"
  )

  local f
  for f in "${required_files[@]}"; do
    assert_file "${f}"
  done

  log "validating kustomization resource references"
  local kustomization
  for kustomization in \
    "k8s/base/kustomization.yaml" \
    "k8s/overlays/monolithic/kustomization.yaml" \
    "k8s/overlays/microservice/kustomization.yaml"; do
    while IFS= read -r ref; do
      [[ -z "${ref}" ]] && continue
      local resolved
      resolved="$(cd "${ROOT_DIR}/$(dirname "${kustomization}")" && realpath "${ref}")"
      if [[ ! -e "${resolved}" ]]; then
        fail "broken reference in ${kustomization}: ${ref}"
      fi
    done < <(awk '
      /^resources:/ {in_resources=1; next}
      in_resources && /^[^[:space:]-]/ {in_resources=0}
      in_resources && /^[[:space:]]*-[[:space:]]*/ {
        sub(/^[[:space:]]*-[[:space:]]*/, "", $0)
        print $0
      }
    ' "${ROOT_DIR}/${kustomization}")
  done

  log "static checks passed"
}

verify_cluster_dry_run() {
  if ! command -v kubectl >/dev/null 2>&1; then
    if [[ "${STRICT_CLUSTER_DRY_RUN}" == "1" ]]; then
      fail "kubectl not found but STRICT_CLUSTER_DRY_RUN=1"
    fi
    log "SKIP cluster-dry-run: kubectl not found"
    return 0
  fi

  log "running kubectl kustomize checks"
  kubectl kustomize "${ROOT_DIR}/k8s/overlays/monolithic" >/dev/null
  kubectl kustomize "${ROOT_DIR}/k8s/overlays/microservice" >/dev/null

  log "running kubectl client dry-run apply checks"
  kubectl apply --dry-run=client --validate=false -k "${ROOT_DIR}/k8s/overlays/monolithic" >/dev/null
  kubectl apply --dry-run=client --validate=false -k "${ROOT_DIR}/k8s/overlays/microservice" >/dev/null

  log "cluster dry-run checks passed"
}

case "${MODE}" in
  auto)
    verify_static
    verify_cluster_dry_run
    ;;
  static)
    verify_static
    ;;
  cluster-dry-run)
    verify_cluster_dry_run
    ;;
  *)
    fail "unsupported mode: ${MODE}"
    ;;
esac

log "k8s deployment capability verification passed"
