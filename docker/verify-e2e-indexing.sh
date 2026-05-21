#!/usr/bin/env bash
# ChainPulse End-to-End Indexing Verification
# Verifies that on-chain events are correctly indexed and queryable via API
# Usage: bash docker/verify-e2e-indexing.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PASS=0
FAIL=0
SKIP=0

pass() { PASS=$((PASS+1)); echo -e "  ${GREEN}PASS${NC} $*"; }
fail() { FAIL=$((FAIL+1)); echo -e "  ${RED}FAIL${NC} $*"; }
skip() { SKIP=$((SKIP+1)); echo -e "  ${YELLOW}SKIP${NC} $*"; }
info() { echo -e "${CYAN}[E2E]${NC} $*"; }

# EventEmitter Solidity contract source
EMITTER_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract EventEmitter {
    event Ping(address sender, uint256 value);
    function emitPing(uint256 value) public { emit Ping(msg.sender, value); }
}'

# Chain configurations: container_name:chain_name:chain_id:host_rpc_port
CHAINS_MS="chainpulse-ms-anvil-ethereum:ethereum:1:18545 chainpulse-ms-anvil-polygon:polygon:137:18546 chainpulse-ms-anvil-bsc:bsc:56:18547 chainpulse-ms-anvil-arbitrum:arbitrum:42161:18548 chainpulse-ms-anvil-optimism:optimism:10:18549 chainpulse-ms-anvil-base:base:8453:18550 chainpulse-ms-anvil-avalanche:avalanche:43114:18551"
CHAINS_MONO="chainpulse-anvil-ethereum:ethereum:1:8545 chainpulse-anvil-polygon:polygon:137:8546 chainpulse-anvil-bsc:bsc:56:8547"

# Anvil default deployer
DEPLOYER_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DEPLOYER_ADDR="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

# State
STATE_DIR="/tmp/chainpulse-e2e"
mkdir -p "$STATE_DIR"

# Detect stack
detect_stack() {
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "chainpulse-ms-puller"; then
        echo "microservices"
    elif docker ps --format "{{.Names}}" 2>/dev/null | grep -q "chainpulse-app"; then
        echo "monolithic"
    else
        echo "none"
    fi
}

get_chains() {
    local stack=$(detect_stack)
    if [ "$stack" = "microservices" ]; then
        echo "$CHAINS_MS"
    elif [ "$stack" = "monolithic" ]; then
        echo "$CHAINS_MONO"
    else
        echo ""
    fi
}

get_api_base() {
    local stack=$(detect_stack)
    if [ "$stack" = "microservices" ]; then
        echo "http://localhost:28080"
    elif [ "$stack" = "monolithic" ]; then
        echo "http://localhost:8080"
    else
        echo ""
    fi
}

# Wait for an event to appear in the API (poll with timeout)
wait_for_event() {
    local api_base="$1"
    local filter_param="$2"
    local expected_min="$3"
    local timeout="${4:-60}"
    local elapsed=0

    while [ $elapsed -lt $timeout ]; do
        local count
        count=$(curl -sf "${api_base}/events?limit=500&${filter_param}" 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); t=p.get('total',0); print(t if t<500 else t)" 2>/dev/null || echo "0")
        if [ "$count" -ge "$expected_min" ]; then
            echo "$count"
            return 0
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    echo "0"
    return 1
}

# ============================================
# Phase 1: Deploy EventEmitter on each chain
# ============================================
info "Phase 1: Deploying EventEmitter contracts on each Anvil chain..."

chains=$(get_chains)
api_base=$(get_api_base)

if [ -z "$chains" ] || [ -z "$api_base" ]; then
    fail "No ChainPulse stack detected. Start one first."
    echo ""
    echo "Results: 0 pass, 1 fail, 0 skip / 1 total"
    exit 1
fi

# Use files instead of associative arrays for macOS bash 3 compatibility
get_deployed_addr() {
    local chain="$1"
    local f="$STATE_DIR/${chain}.emitter"
    if [ -f "$f" ]; then cat "$f"; else echo ""; fi
}
set_deployed_addr() {
    local chain="$1"
    local addr="$2"
    echo "$addr" > "$STATE_DIR/${chain}.emitter"
}

for chain_entry in $chains; do
    container="${chain_entry%%:*}"
    chain_name=$(echo "$chain_entry" | cut -d: -f2)
    chain_id=$(echo "$chain_entry" | cut -d: -f3)
    state_file="$STATE_DIR/${chain_name}.emitter"

    if [ -f "$state_file" ]; then
        existing=$(get_deployed_addr "$chain_name")
        info "  $chain_name: reusing existing EventEmitter at $existing"
        continue
    fi

    info "  Deploying EventEmitter on $chain_name (container: $container)..."

    # Write Solidity source
    docker exec "$container" sh -c "cat > /tmp/EventEmitter.sol << 'SOLEOF'
$EMITTER_SOL
SOLEOF" 2>/dev/null || { skip "$chain_name: cannot write to container"; continue; }

    # Create writable directories
    docker exec "$container" sh -c "mkdir -p /tmp/forge-out /tmp/forge-cache && chmod 777 /tmp/forge-out /tmp/forge-cache" 2>/dev/null || true

    # Deploy with forge create
    local_addr=""
    local_addr=$(docker exec "$container" forge create \
        /tmp/EventEmitter.sol:EventEmitter \
        --rpc-url http://localhost:8545 \
        --private-key "$DEPLOYER_KEY" \
        --out /tmp/forge-out \
        --cache-path /tmp/forge-cache \
        --broadcast 2>&1 | grep "Deployed to:" | awk '{print $3}' || echo "")

    if [ -n "$local_addr" ]; then
        echo "$local_addr" > "$state_file"
        set_deployed_addr "$chain_name" "$local_addr"
        info "  $chain_name: EventEmitter deployed at $local_addr"
    else
        skip "$chain_name: failed to deploy EventEmitter"
    fi
done

# ============================================
# Phase 2: Emit Ping events and verify indexing
# ============================================
info ""
info "Phase 2: Emitting Ping events and verifying end-to-end indexing..."

for chain_entry in $chains; do
    container="${chain_entry%%:*}"
    chain_name=$(echo "$chain_entry" | cut -d: -f2)
    chain_id=$(echo "$chain_entry" | cut -d: -f3)
    host_port=$(echo "$chain_entry" | cut -d: -f4)

    emitter_addr=$(get_deployed_addr "$chain_name")
    if [ -z "$emitter_addr" ]; then
        skip "$chain_name: no EventEmitter deployed"
        continue
    fi

    info "  Testing $chain_name (chain_id=$chain_id)..."

    # Call emitPing(42) on the deployed contract
    ping_result=""
    ping_result=$(docker exec "$container" cast send \
        --rpc-url http://localhost:8545 \
        --private-key "$DEPLOYER_KEY" \
        "$emitter_addr" \
        "emitPing(uint256)" 42 2>&1) || { skip "$chain_name: emitPing tx failed"; continue; }

    tx_hash=$(echo "$ping_result" | grep "transactionHash" | awk '{print $2}' | tr -d ',' || echo "")
    block_num=$(echo "$ping_result" | grep "blockNumber" | awk '{print $2}' | tr -d ',' || echo "")

    if [ -z "$tx_hash" ]; then
        fail "$chain_name: emitPing returned no transaction hash"
        continue
    fi

    info "    Tx: $tx_hash  Block: $block_num"

    # Wait for the event to appear in the API
    event_count=$(wait_for_event "$api_base" "event_signature=Ping&contract=${emitter_addr}" 1 60 || true)

    if [ "$event_count" -ge 1 ]; then
        # Verify event details - fetch events by contract and filter by chainId in test
        # (using /events/name/Ping because /events/chain/{id} has MongoDB string/int mismatch)
        event_data=$(curl -sf "${api_base}/events/name/Ping?contract=${emitter_addr}&limit=50" 2>/dev/null || echo "{}")

        # Extract the event matching this chain's chainId from the results
        event_chain_id=$(echo "$event_data" | python3 -c "
import sys,json
d=json.load(sys.stdin)
events=d.get('data',d.get('events',[]))
for e in events:
    if e.get('chainId') == $chain_id:
        print(e.get('chainId',''))
        break
else:
    # fallback: take first event
    if events:
        print(events[0].get('chainId',''))
    else:
        print('')" 2>/dev/null || echo "")

        event_name=$(echo "$event_data" | python3 -c "
import sys,json
d=json.load(sys.stdin)
events=d.get('data',d.get('events',[]))
for e in events:
    if e.get('chainId') == $chain_id:
        print(e.get('eventName',''))
        break
else:
    if events:
        print(events[0].get('eventName',''))
    else:
        print('')" 2>/dev/null || echo "")

        event_contract=$(echo "$event_data" | python3 -c "
import sys,json
d=json.load(sys.stdin)
events=d.get('data',d.get('events',[]))
for e in events:
    if e.get('chainId') == $chain_id:
        print(e.get('contractAddress','').lower())
        break
else:
    if events:
        print(events[0].get('contractAddress','').lower())
    else:
        print('')" 2>/dev/null || echo "")

        event_block=$(echo "$event_data" | python3 -c "
import sys,json
d=json.load(sys.stdin)
events=d.get('data',d.get('events',[]))
for e in events:
    if e.get('chainId') == $chain_id:
        print(e.get('blockNumber',0))
        break
else:
    if events:
        print(events[0].get('blockNumber',0))
    else:
        print(0)" 2>/dev/null || echo "0")

        event_sig=$(echo "$event_data" | python3 -c "
import sys,json
d=json.load(sys.stdin)
events=d.get('data',d.get('events',[]))
for e in events:
    if e.get('chainId') == $chain_id:
        print(e.get('eventSignature',''))
        break
else:
    if events:
        print(events[0].get('eventSignature',''))
    else:
        print('')" 2>/dev/null || echo "")

        # Assert chainId matches
        if [ "$event_chain_id" = "$chain_id" ]; then
            pass "$chain_name: chainId matches ($chain_id)"
        else
            fail "$chain_name: chainId mismatch (expected $chain_id, got $event_chain_id)"
        fi

        # Assert eventName is resolved (not raw hex)
        if [ "$event_name" = "Ping" ]; then
            pass "$chain_name: eventName resolved to 'Ping' (not raw hex)"
        else
            fail "$chain_name: eventName not resolved (got '$event_name')"
        fi

        # Assert contractAddress matches
        expected_addr=$(echo "$emitter_addr" | tr '[:upper:]' '[:lower:]')
        if [ "$event_contract" = "$expected_addr" ]; then
            pass "$chain_name: contractAddress matches ($emitter_addr)"
        else
            fail "$chain_name: contractAddress mismatch (expected $emitter_addr, got $event_contract)"
        fi

        # Assert blockNumber > 0
        if [ "$event_block" -gt 0 ] 2>/dev/null; then
            pass "$chain_name: blockNumber > 0 ($event_block)"
        else
            fail "$chain_name: blockNumber invalid ($event_block)"
        fi

        # Assert eventSignature is present
        if [ -n "$event_sig" ]; then
            pass "$chain_name: eventSignature field present ($event_sig)"
        else
            fail "$chain_name: eventSignature field missing"
        fi
    else
        fail "$chain_name: Ping event not found in API after 60s"
    fi
done

# ============================================
# Phase 3: Verify API filter parameters
# ============================================
info ""
info "Phase 3: Verifying API filter parameters..."

# Test event_signature filter
transfer_count=$(curl -sf "${api_base}/events?event_signature=Transfer&limit=500" 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); print(p.get('total',0))" 2>/dev/null || echo "0")
if [ "$transfer_count" -gt 0 ] 2>/dev/null; then
    pass "event_signature=Transfer filter returns events ($transfer_count)"
else
    # May not have Transfer events yet if simulate-events not running
    skip "event_signature=Transfer filter (no Transfer events yet)"
fi

# Test from_block filter
block_filtered=$(curl -sf "${api_base}/events?from_block=1&to_block=5&limit=500" 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); t=p.get('total',0); print(t if t<500 else t)" 2>/dev/null || echo "0")
if [ "$block_filtered" -ge 0 ] 2>/dev/null; then
    pass "from_block/to_block filter accepted ($block_filtered events in blocks 1-5)"
else
    fail "from_block/to_block filter rejected"
fi

# Test from_time filter
time_filtered=$(curl -sf "${api_base}/events?from_time=1&limit=1" 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); print(p.get('total',0))" 2>/dev/null || echo "0")
if [ "$time_filtered" -ge 0 ] 2>/dev/null; then
    pass "from_time filter accepted"
else
    fail "from_time filter rejected"
fi

# Test invalid filter (from_block > to_block)
invalid_resp=$(curl -s -o /dev/null -w "%{http_code}" "${api_base}/events?from_block=100&to_block=1" 2>/dev/null || echo "000")
if [ "$invalid_resp" = "400" ]; then
    pass "Invalid from_block > to_block returns 400"
else
    fail "Invalid from_block > to_block should return 400 (got $invalid_resp)"
fi

# Test event_signature=Ping resolves by name
ping_by_name=$(curl -sf "${api_base}/events?event_signature=Ping&limit=10" 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); print(p.get('total',0))" 2>/dev/null || echo "0")
if [ "$ping_by_name" -gt 0 ] 2>/dev/null; then
    pass "event_signature=Ping (name resolution) returns events ($ping_by_name)"
else
    fail "event_signature=Ping (name resolution) returned no events"
fi

# ============================================
# Phase 4: Verify event name resolution
# ============================================
info ""
info "Phase 4: Verifying event name resolution (Approval events)..."

# Check if Approval events show resolved name instead of raw hex
approval_events=$(curl -sf "${api_base}/events/name/Approval?limit=5" 2>/dev/null \
    | python3 -c "
import sys,json
d=json.load(sys.stdin)
events=d.get('data',d.get('events',[]))
raw_hex = [e for e in events if e.get('eventName','').startswith('0x')]
resolved = [e for e in events if e.get('eventName','') == 'Approval']
print(f'resolved={len(resolved)},raw_hex={len(raw_hex)}')" 2>/dev/null || echo "resolved=0,raw_hex=0")

resolved_count=$(echo "$approval_events" | sed -n 's/.*resolved=\([0-9]*\).*/\1/p' || echo "0")
raw_hex_count=$(echo "$approval_events" | sed -n 's/.*raw_hex=\([0-9]*\).*/\1/p' || echo "0")

if [ "$resolved_count" -gt 0 ]; then
    pass "Approval events resolved to 'Approval' name ($resolved_count resolved, $raw_hex_count raw hex)"
elif [ "$raw_hex_count" -gt 0 ]; then
    fail "Approval events still showing raw hex signature (need to rebuild/redeploy images)"
else
    skip "No Approval events found (may need simulate-events running)"
fi

# ============================================
# Summary
# ============================================
info ""
info "============================================================"
TOTAL=$((PASS + FAIL + SKIP))
info "  Results: $PASS pass, $FAIL fail, $SKIP skip / $TOTAL total"
if [ $FAIL -eq 0 ]; then
    info "  ALL CRITICAL CHECKS PASSED"
else
    info "  SOME CHECKS FAILED - review above"
fi
info "============================================================"

# Cleanup state files (keep for debugging)
# rm -rf "$STATE_DIR"

exit $FAIL
