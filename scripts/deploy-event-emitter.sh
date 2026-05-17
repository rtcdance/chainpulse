#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────
# Deploy EventEmitter to Anvil + emit test events
# ──────────────────────────────────────────────────────────
# Prerequisites: Docker (Anvil container from docker-compose)
#
# Usage:
#   bash scripts/deploy-event-emitter.sh
#
# Then check events in ChainPulse via:
#   curl http://localhost:8080/api/v1/events
# ──────────────────────────────────────────────────────────
set -euo pipefail

ANVIL_RPC="${ANVIL_RPC:-http://localhost:8545}"
PRIVATE_KEY="${PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"  # Anvil account #0
FROM="${FROM:-0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266}"

echo "═══ EventEmitter Deploy & Emit ═══"
echo "RPC:        $ANVIL_RPC"
echo "Deployer:   $FROM"
echo ""

# Check if Anvil is reachable
if ! cast chain --rpc-url "$ANVIL_RPC" &>/dev/null; then
  echo "ERROR: Cannot reach Anvil at $ANVIL_RPC"
  echo "Start it: docker compose -f docker/docker-compose.yml up -d anvil-ethereum --wait"
  exit 1
fi

CHAIN_ID=$(cast chain-id --rpc-url "$ANVIL_RPC")
echo "Chain ID:   $CHAIN_ID"
echo ""

# ── Step 1: Compile (via Docker) ──
echo "[1/4] Compiling EventEmitter.sol..."
COMPILE_OUT=$(docker run --rm -v "$(pwd)/contracts:/contracts" \
  ghcr.io/foundry-rs/foundry:latest \
  forge build --root /contracts --contracts /contracts --out /contracts/out --evm-version cancun 2>&1)
echo "       Compilation done."

# Extract deployed bytecode
BYTECODE=$(cat contracts/out/EventEmitter.sol/EventEmitter.json 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['bytecode']['object'])" 2>/dev/null || echo "")
if [ -z "$BYTECODE" ]; then
  echo "ERROR: Failed to extract bytecode. Compilation output:"
  echo "$COMPILE_OUT"
  exit 1
fi

# ── Step 2: Deploy ──
echo "[2/4] Deploying EventEmitter..."
TX_HASH=$(cast send --rpc-url "$ANVIL_RPC" --private-key "$PRIVATE_KEY" \
  --legacy \
  --create "$BYTECODE" 2>&1 | grep -E "^contractAddress|^deployedTo|transactionHash" | head -1 | awk '{print $2}')
CONTRACT_ADDR=$(cast receipt --rpc-url "$ANVIL_RPC" "$TX_HASH" contractAddress 2>/dev/null || echo "")

if [ -z "$CONTRACT_ADDR" ]; then
  sleep 2
  CONTRACT_ADDR=$(cast receipt --rpc-url "$ANVIL_RPC" "$TX_HASH" contractAddress 2>/dev/null || echo "")
fi

if [ -z "$CONTRACT_ADDR" ]; then
  echo "ERROR: Could not get contract address from tx $TX_HASH"
  exit 1
fi

echo "       Contract: $CONTRACT_ADDR"
echo "       TX:       $TX_HASH"
echo ""

# ── Step 3: Emit events ──
echo "[3/4] Emitting events..."
for i in 1 2 3; do
  TX=$(cast send --rpc-url "$ANVIL_RPC" --private-key "$PRIVATE_KEY" \
    --legacy \
    "$CONTRACT_ADDR" "emitTransfer(address,uint256)" \
    "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" "$((i * 100))" 2>&1 | grep "^transactionHash" | awk '{print $2}')
  echo "       Transfer #$i tx: $TX"
done

TX_CUSTOM=$(cast send --rpc-url "$ANVIL_RPC" --private-key "$PRIVATE_KEY" \
  --legacy \
  "$CONTRACT_ADDR" 'emitCustom(string)' \
  '"hello from debugger"' 2>&1 | grep "^transactionHash" | awk '{print $2}')
echo "       CustomEvent tx: $TX_CUSTOM"

TX_BATCH=$(cast send --rpc-url "$ANVIL_RPC" --private-key "$PRIVATE_KEY" \
  --legacy \
  "$CONTRACT_ADDR" "emitBatch(uint256)" 5 2>&1 | grep "^transactionHash" | awk '{print $2}')
echo "       Batch (5 events) tx: $TX_BATCH"
echo ""

# ── Step 4: Summary ──
echo "[4/4] Done!"
echo ""
echo "═══ Summary ═══"
echo "Contract:   $CONTRACT_ADDR"
echo "Chain:      $ANVIL_RPC"
echo "Events:     3 Transfer + 1 CustomEvent + 5 Batch = 9 total"
echo ""
echo "Now debug ChainPulse pointing to:"
echo "  CHAINPULSE_BLOCKCHAIN_NODE_URL=$ANVIL_RPC"
echo "  CHAINPULSE_CHAINS=ethereum"
echo "  CHAINPULSE_START_BLOCK=0"
echo ""
echo "Set breakpoints in the debugger at:"
echo "  - pkg/plugins/pullers/https_jsonrpc_puller.go  (raw RPC calls)"
echo "  - pkg/core/event_decoder.go                    (ABI decoding)"
echo "  - pkg/adapters/indexing/monolithic_memory_storage.go  (storage)"
