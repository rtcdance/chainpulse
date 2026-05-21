#!/usr/bin/env bash
# ChainPulse Docker Acceptance - Verification Script
# Usage: bash docker/verify-acceptance.sh
# Prerequisites: Stack must be running (bash docker/acceptance.sh up)
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0
WARN=0

pass() { PASS=$((PASS+1)); echo -e "  ${GREEN}PASS${NC} $*"; }
fail() { FAIL=$((FAIL+1)); echo -e "  ${RED}FAIL${NC} $*"; }
warn() { WARN=$((WARN+1)); echo -e "  ${YELLOW}WARN${NC} $*"; }

echo "============================================"
echo "  ChainPulse Docker Acceptance Verification "
echo "============================================"
echo ""

# ============================================
# 1. Infrastructure Health
# ============================================
echo "--- Infrastructure ---"

if docker exec chainpulse-postgres pg_isready -U chainpulse -d chainpulse >/dev/null 2>&1; then
    pass "PostgreSQL: accepting connections"
else
    fail "PostgreSQL: not ready"
fi

if docker exec chainpulse-redis redis-cli ping 2>/dev/null | grep -q PONG; then
    pass "Redis: PONG"
else
    fail "Redis: not responding"
fi

MONGO_CONTAINER=$(docker ps --format "{{.Names}}" | grep chainpulse-mongodb | head -1)
if [ -n "$MONGO_CONTAINER" ] && docker exec "$MONGO_CONTAINER" mongosh --eval "db.adminCommand('ping')" --quiet 2>/dev/null | grep -q "ok"; then
    pass "MongoDB: ping OK"
else
    fail "MongoDB: not responding"
fi

if docker exec chainpulse-kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null | grep -q "blockchain-events"; then
    pass "Kafka: topics available"
else
    warn "Kafka: blockchain-events topic not found (may auto-create)"
fi

# ============================================
# 2. Anvil Chain Nodes
# ============================================
echo ""
echo "--- Blockchain Nodes ---"

CHAINS="ethereum:8545 polygon:8546 bsc:8547 arbitrum:8548 optimism:8549 base:8550 avalanche:8551"
for chain_port in $CHAINS; do
    chain="${chain_port%%:*}"
    container="chainpulse-anvil-${chain}"
    if docker exec "$container" cast block-number --rpc-url http://localhost:8545 2>/dev/null | grep -qE "^[0-9]+$"; then
        pass "$chain: Anvil responding"
    else
        fail "$chain: Anvil not responding"
    fi
done

# ============================================
# 3. Application Health Endpoints
# ============================================
echo ""
echo "--- Application Health ---"

check_endpoint() {
    local method="${1:-GET}"
    local path="$2"
    local expect_status="${3:-200}"
    local expect_contains="${4:-}"

    local url="http://localhost:8080${path}"
    local response
    if [ "$method" = "POST" ]; then
        response=$(curl -sf -w "\n%{http_code}" -X POST "$url" -H 'Content-Type: application/json' -d "${5:-{}}" 2>&1) || true
    else
        response=$(curl -sf -w "\n%{http_code}" "$url" 2>&1) || true
    fi

    local http_code
    http_code=$(echo "$response" | tail -1)
    local body
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "$expect_status" ]; then
        if [ -n "$expect_contains" ]; then
            if echo "$body" | grep -q "$expect_contains"; then
                pass "$method $path -> $http_code (contains: $expect_contains)"
            else
                fail "$method $path -> $http_code (missing: $expect_contains)"
            fi
        else
            pass "$method $path -> $http_code"
        fi
    else
        fail "$method $path -> $http_code (expected $expect_status)"
    fi
}

check_endpoint GET /health 200 "healthy"
check_endpoint GET /health/ready 200 "ready"
check_endpoint GET /health/live 200 "alive"
check_endpoint GET /health/components 200 "healthy"
check_endpoint GET /health/rollout 200 "rollout"

# ============================================
# 4. Runtime & Metrics Endpoints
# ============================================
echo ""
echo "--- Runtime & Metrics ---"

check_endpoint GET /runtime/summary 200 "monolithic"
check_endpoint GET /metrics 200 "chainpulse_"
check_endpoint GET /runtime/control 200 "polling-loop"

# ============================================
# 5. Query Endpoints
# ============================================
echo ""
echo "--- Query API ---"

check_endpoint GET "/events?limit=5" 200 "events"
check_endpoint GET /models 200 "models"
check_endpoint POST /graphql 200 "data" '{"query":"{ __schema { queryType { fields { name } } } }"}'

# ============================================
# 6. DLQ Replay
# ============================================
echo ""
echo "--- DLQ & Replay ---"

check_endpoint POST /runtime/indexing/dlq/replay 200 "dlq-replay" '{"chain_id":"ethereum"}'

# ============================================
# 7. WebSocket
# ============================================
echo ""
echo "--- WebSocket ---"

ws_code=$(curl -s -o /tmp/cp-ws-body -w "%{http_code}" \
    -H "Upgrade: websocket" -H "Connection: Upgrade" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    -H "Sec-WebSocket-Version: 13" \
    http://localhost:8080/ws 2>/dev/null || true)

if [ "$ws_code" = "101" ]; then
    pass "WebSocket /ws -> 101 Switching Protocols"
else
    fail "WebSocket /ws -> $ws_code (expected 101)"
fi

# ============================================
# 8. Frontend & Proxy
# ============================================
echo ""
echo "--- Frontend ---"

frontend_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:3000 2>/dev/null || echo "000")
if [ "$frontend_code" = "200" ]; then
    pass "Frontend http://localhost:3000 -> 200"
else
    fail "Frontend http://localhost:3000 -> $frontend_code"
fi

proxy_health=$(curl -sf http://localhost:3000/health 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
if [ "$proxy_health" = "healthy" ]; then
    pass "Frontend API proxy -> healthy"
else
    fail "Frontend API proxy -> status='$proxy_health'"
fi

# Check monolithic service proxies
for svc in api-gateway api-service event-processor puller; do
    proxy_status=$(curl -sf "http://localhost:3000/__proxy/${svc}/health" 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
    if [ "$proxy_status" = "healthy" ]; then
        pass "Frontend __proxy/$svc -> healthy"
    else
        fail "Frontend __proxy/$svc -> status='$proxy_status'"
    fi
done

# ============================================
# 9. Observability
# ============================================
echo ""
echo "--- Observability ---"

prom_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:9090 2>/dev/null || echo "000")
if [ "$prom_code" = "302" ] || [ "$prom_code" = "200" ]; then
    pass "Prometheus http://localhost:9090 -> $prom_code"
else
    fail "Prometheus http://localhost:9090 -> $prom_code"
fi

prom_targets=$(curl -sf "http://localhost:9090/api/v1/targets" 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); targets=d.get('data',{}).get('activeTargets',[]); t=[x for x in targets if x['health']=='up']; print(f'{len(t)}/{len(targets)}')" 2>/dev/null || echo "0/0")
if echo "$prom_targets" | grep -qE "^[1-9]"; then
    pass "Prometheus targets up: $prom_targets"
else
    fail "Prometheus targets up: $prom_targets"
fi

grafana_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:3001 2>/dev/null || echo "000")
if [ "$grafana_code" = "302" ] || [ "$grafana_code" = "200" ]; then
    pass "Grafana http://localhost:3001 -> $grafana_code"
else
    fail "Grafana http://localhost:3001 -> $grafana_code"
fi

jaeger_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:16686 2>/dev/null || echo "000")
if [ "$jaeger_code" = "200" ]; then
    pass "Jaeger http://localhost:16686 -> 200"
else
    fail "Jaeger http://localhost:16686 -> $jaeger_code"
fi

# ============================================
# 10. Multi-Chain Indexing
# ============================================
echo ""
echo "--- Multi-Chain Indexing ---"

chain_count=$(curl -sf http://localhost:8080/runtime/summary 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); ro=d.get('rollout',{}); print(ro.get('ownership_chains',0))" 2>/dev/null || echo "0")
if [ "$chain_count" = "7" ]; then
    pass "Chains registered: $chain_count"
else
    warn "Chains registered: $chain_count (expected 7)"
fi

puller_starts=$(curl -sf http://localhost:8080/metrics 2>/dev/null \
    | grep "chainpulse_plugin_starts" | grep -v "#" \
    | awk '{print $2}' | head -1 || echo "0")
if [ "$puller_starts" = "7" ]; then
    pass "Pullers started: $puller_starts"
else
    warn "Pullers started: $puller_starts (expected 7)"
fi

# ============================================
# Summary
# ============================================
echo ""
echo "============================================"
TOTAL=$((PASS + FAIL + WARN))
echo -e "  Results: ${GREEN}$PASS pass${NC}, ${RED}$FAIL fail${NC}, ${YELLOW}$WARN warn${NC} / $TOTAL total"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}ALL CRITICAL CHECKS PASSED${NC}"
else
    echo -e "  ${RED}$FAIL CHECKS FAILED${NC}"
fi
echo "============================================"

exit $FAIL
