#!/usr/bin/env bash
# ChainPulse ingestion benchmark.
#
# Measures, against a running indexer + Anvil:
#   1. block catch-up rate (blocks/s)
#   2. event ingestion rate (events/s)
#   3. query latency P50/P99 (via k6, see query_events.js)
#
# Requirements: the indexer is up on $API, Anvil is up on $RPC, and
# foundry (cast) is on PATH.
#
# Usage:
#   RPC=http://127.0.0.1:18546 API=http://127.0.0.1:8080 ./scripts/bench/ingest.sh
set -euo pipefail

RPC="${RPC:-http://127.0.0.1:18546}"
API="${API:-http://127.0.0.1:8080}"
DEPLOYER_KEY="${DEPLOYER_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"
CONTRACT="${CONTRACT:-}"
BLOCKS="${BLOCKS:-500}"
EVENTS="${EVENTS:-50}"

# A system HTTP proxy is configured on some machines; never route loopback
# traffic through it. Using a function (not a string) avoids glob/word-splitting
# surprises with --noproxy.
export NO_PROXY="127.0.0.1,localhost"
export no_proxy="127.0.0.1,localhost"
curl_get() { curl -s -m 10 --noproxy '*' "$@"; }

rpc() { curl_get -X POST -H 'Content-Type: application/json' --data "$1" "$RPC"; }

anvil_height() { rpc '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | python3 -c 'import json,sys;print(int(json.load(sys.stdin)["result"],16))'; }

checkpoint() { curl_get "$API/runtime/summary" | python3 -c 'import json,sys;print(json.load(sys.stdin)["indexing"]["last_checkpoint_block"])'; }

total_events() { curl_get "$API/events/stats" | python3 -c 'import json,sys;print(json.load(sys.stdin)["total"])'; }

echo "=== ChainPulse ingestion benchmark ==="
echo "RPC=$RPC  API=$API"
echo "start: anvil_height=$(anvil_height) indexed_checkpoint=$(checkpoint) total_events=$(total_events)"
echo

# ---------------------------------------------------------------- blocks/s
echo "--- 1. block catch-up rate ---"
c0=$(checkpoint)
echo "mining $BLOCKS blocks..."
rpc "{\"jsonrpc\":\"2.0\",\"method\":\"anvil_mine\",\"params\":[\"$(printf '0x%x' "$BLOCKS")\"],\"id\":1}" > /dev/null

t0=$(date +%s)
target=$((c0 + BLOCKS))
for _ in $(seq 1 120); do
  c=$(checkpoint)
  if [ "$c" -ge "$target" ]; then break; fi
  sleep 1
done
t1=$(date +%s)
elapsed=$((t1 - t0))
[ "$elapsed" -lt 1 ] && elapsed=1
final=$(checkpoint)
indexed=$((final - c0))
echo "indexed $indexed blocks in ${elapsed}s -> $(python3 -c "print(round($indexed/$elapsed,1))") blocks/s"
echo

# ---------------------------------------------------------------- events/s
if [ -n "$CONTRACT" ]; then
  echo "--- 2. event ingestion rate ---"
  e0=$(total_events)
  echo "sending $EVENTS registerContent txs to $CONTRACT..."
  t0=$(date +%s)
  for i in $(seq 1 "$EVENTS"); do
    cast send --rpc-url "$RPC" --private-key "$DEPLOYER_KEY" "$CONTRACT" \
      "registerContent(string,string,string)" "bench-$(date +%s)-$i" "0xdeadbeef" "ipfs://bench" \
      > /dev/null 2>&1 || true
  done
  sent_at=$(date +%s)
  for _ in $(seq 1 120); do
    e=$(total_events)
    if [ "$e" -ge $((e0 + EVENTS)) ]; then break; fi
    sleep 1
  done
  t1=$(date +%s)
  ingested=$(( $(total_events) - e0 ))
  send_secs=$((sent_at - t0)); [ "$send_secs" -lt 1 ] && send_secs=1
  index_secs=$((t1 - t0)); [ "$index_secs" -lt 1 ] && index_secs=1
  echo "tx send rate : $(python3 -c "print(round($EVENTS/$send_secs,1))") tx/s"
  echo "events indexed: $ingested in ${index_secs}s -> $(python3 -c "print(round($ingested/$index_secs,1))") events/s"
  echo
else
  echo "--- 2. event ingestion rate: SKIPPED (set CONTRACT=<address> to enable) ---"
  echo
fi

# ---------------------------------------------------------------- queries
echo "--- 3. query latency (k6) ---"
if command -v k6 > /dev/null 2>&1; then
  BASE_URL="$API" DURATION="${DURATION:-60s}" RATE="${RATE:-100}" \
    k6 run "$(dirname "$0")/query_events.js" 2>&1 | grep -E "http_req_duration|http_reqs|http_req_failed" | head -10
else
  echo "k6 not found — install with: go install go.k6.io/k6@latest"
fi
