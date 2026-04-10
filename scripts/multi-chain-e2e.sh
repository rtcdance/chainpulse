#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${MODE:-auto}"
START_EVM_LOCAL="${START_EVM_LOCAL:-1}"
START_SOLANA_LOCAL="${START_SOLANA_LOCAL:-0}"
EVM_FORK_MODE="${EVM_FORK_MODE:-0}"
EVM_FORK_URL="${EVM_FORK_URL:-}"
EVM_FORK_URLS="${EVM_FORK_URLS:-}"
EVM_FORK_BLOCK_NUMBER="${EVM_FORK_BLOCK_NUMBER:-}"
SOLANA_RPC_ENDPOINT="${SOLANA_RPC_ENDPOINT:-http://localhost:8899}"
GO_TEST_PATTERN="${GO_TEST_PATTERN:-TestMultiChainProtocolAcceptance}"

usage() {
  cat <<'EOF'
Usage: scripts/multi-chain-e2e.sh [--mode auto|strict]

One-click multi-chain E2E acceptance:
- multi-EVM protocol probes
- optional Solana protocol probe

Environment variables:
  MODE                 default: auto (auto|strict)
  START_EVM_LOCAL      default: 1 (start local Anvil set when available)
  START_SOLANA_LOCAL   default: 0 (start solana-test-validator when available)
  EVM_FORK_MODE        default: 0 (1 to start Anvil with --fork-url)
  EVM_FORK_URL         optional default fork URL for all EVM chains
  EVM_FORK_URLS        optional per-chain fork URLs: chain=url,chain=url
  EVM_FORK_BLOCK_NUMBER optional fork block number for all started EVM forks
  SOLANA_RPC_ENDPOINT  default: http://localhost:8899
  GO_TEST_PATTERN      default: TestMultiChainProtocolAcceptance
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ "${1:-}" == "--mode" ]]; then
  MODE="${2:-}"
fi

if [[ "${MODE}" != "auto" && "${MODE}" != "strict" ]]; then
  echo "[multi-chain-e2e] ERROR: unsupported mode ${MODE}" >&2
  exit 1
fi

log() {
  printf '[multi-chain-e2e] %s\n' "$*"
}

PIDS=()
cleanup() {
  local code=$?
  for pid in "${PIDS[@]:-}"; do
    kill "${pid}" >/dev/null 2>&1 || true
  done
  exit "${code}"
}
trap cleanup EXIT INT TERM

is_port_open() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  return 1
}

fork_url_for_chain() {
  local chain="$1"
  local item
  for item in ${EVM_FORK_URLS//,/ }; do
    local key="${item%%=*}"
    local value="${item#*=}"
    if [[ "${key}" == "${chain}" && -n "${value}" && "${key}" != "${value}" ]]; then
      printf '%s' "${value}"
      return 0
    fi
  done
  case "${chain}" in
    ethereum) if [[ -n "${CHAINPULSE_ETHEREUM_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_ETHEREUM_NODE_URL}"; return 0; fi ;;
    polygon) if [[ -n "${CHAINPULSE_POLYGON_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_POLYGON_NODE_URL}"; return 0; fi ;;
    bsc) if [[ -n "${CHAINPULSE_BSC_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_BSC_NODE_URL}"; return 0; fi ;;
    arbitrum) if [[ -n "${CHAINPULSE_ARBITRUM_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_ARBITRUM_NODE_URL}"; return 0; fi ;;
    optimism) if [[ -n "${CHAINPULSE_OPTIMISM_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_OPTIMISM_NODE_URL}"; return 0; fi ;;
    base) if [[ -n "${CHAINPULSE_BASE_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_BASE_NODE_URL}"; return 0; fi ;;
    avalanche) if [[ -n "${CHAINPULSE_AVALANCHE_NODE_URL:-}" ]]; then printf '%s' "${CHAINPULSE_AVALANCHE_NODE_URL}"; return 0; fi ;;
  esac
  printf '%s' "${EVM_FORK_URL}"
}

start_evm_locals() {
  local chains=(
    "ethereum:8545:1"
    "polygon:8546:137"
    "bsc:8547:97"
    "arbitrum:8548:421614"
    "optimism:8549:11155420"
    "base:8550:84532"
    "avalanche:8551:43113"
  )

  local anvil_bin=""
  if command -v anvil >/dev/null 2>&1; then
    anvil_bin="anvil"
  elif [[ -x "${HOME}/.foundry/bin/anvil" ]]; then
    anvil_bin="${HOME}/.foundry/bin/anvil"
  fi

  if [[ -z "${anvil_bin}" ]]; then
    log "Anvil not found; skip local EVM startup"
    return 0
  fi

  local endpoint_pairs=()
  local started_count=0
  local chain_info
  for chain_info in "${chains[@]}"; do
    IFS=':' read -r name port chain_id <<< "${chain_info}"
    local fork_url=""
    if [[ "${EVM_FORK_MODE}" == "1" ]]; then
      fork_url="$(fork_url_for_chain "${name}")"
      if [[ -z "${fork_url}" ]]; then
        log "skip ${name}: fork mode enabled but no fork URL configured"
        continue
      fi
    fi

    if is_port_open "${port}"; then
      log "reuse ${name} on :${port}"
      endpoint_pairs+=("${name}=http://localhost:${port}")
      started_count=$((started_count + 1))
      continue
    fi
    if [[ "${EVM_FORK_MODE}" == "1" ]]; then
      log "start ${name} anvil fork on :${port} (chain-id=${chain_id})"
      if [[ -n "${EVM_FORK_BLOCK_NUMBER}" ]]; then
        "${anvil_bin}" --host 127.0.0.1 --port "${port}" --chain-id "${chain_id}" --timestamp 0 --fork-url "${fork_url}" --fork-block-number "${EVM_FORK_BLOCK_NUMBER}" >/tmp/chainpulse-anvil-"${name}".log 2>&1 &
      else
        "${anvil_bin}" --host 127.0.0.1 --port "${port}" --chain-id "${chain_id}" --timestamp 0 --fork-url "${fork_url}" >/tmp/chainpulse-anvil-"${name}".log 2>&1 &
      fi
    else
      log "start ${name} anvil on :${port} (chain-id=${chain_id})"
      "${anvil_bin}" --host 127.0.0.1 --port "${port}" --chain-id "${chain_id}" --timestamp 0 >/tmp/chainpulse-anvil-"${name}".log 2>&1 &
    fi
    local pid=$!
    sleep 1
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      log "start failed for ${name} on :${port}, see /tmp/chainpulse-anvil-${name}.log"
      continue
    fi

    PIDS+=("${pid}")
    endpoint_pairs+=("${name}=http://localhost:${port}")
    started_count=$((started_count + 1))
  done

  if [[ "${EVM_FORK_MODE}" == "1" && "${started_count}" -eq 0 ]]; then
    log "ERROR: fork mode enabled but no EVM chain was configured with a fork URL"
    exit 1
  fi

  EVM_RPC_ENDPOINTS="$(IFS=,; echo "${endpoint_pairs[*]}")"
  export EVM_RPC_ENDPOINTS
  log "EVM_RPC_ENDPOINTS=${EVM_RPC_ENDPOINTS}"
}

start_solana_local() {
  local solana_bin=""
  if command -v solana-test-validator >/dev/null 2>&1; then
    solana_bin="solana-test-validator"
  fi
  if [[ -z "${solana_bin}" ]]; then
    log "solana-test-validator not found; skip local Solana startup"
    return 0
  fi

  if is_port_open 8899; then
    log "reuse solana rpc on :8899"
    return 0
  fi

  log "start solana-test-validator on :8899"
  "${solana_bin}" --rpc-port 8899 --reset >/tmp/chainpulse-solana-validator.log 2>&1 &
  PIDS+=("$!")
  sleep 3
}

if [[ "${START_EVM_LOCAL}" == "1" ]]; then
  start_evm_locals
fi

if [[ "${START_SOLANA_LOCAL}" == "1" ]]; then
  start_solana_local
fi

cd "${ROOT_DIR}"

if [[ "${MODE}" == "strict" ]]; then
  export MULTICHAIN_STRICT=1
  export MULTICHAIN_REQUIRE_SOLANA=1
else
  export MULTICHAIN_STRICT=0
  export MULTICHAIN_REQUIRE_SOLANA=0
fi

export SOLANA_RPC_ENDPOINT
log "run go test pattern=${GO_TEST_PATTERN}, mode=${MODE}"
go test ./test/e2e -run "${GO_TEST_PATTERN}" -v -count=1
log "multi-chain e2e acceptance finished"
