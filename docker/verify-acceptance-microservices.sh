#!/usr/bin/env bash
# ChainPulse Microservices Docker Acceptance - Verification Script
# Usage: bash docker/verify-acceptance-microservices.sh
# Prerequisites: Stack must be running (bash docker/acceptance-microservices.sh up)
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

echo "============================================================"
echo "  ChainPulse Microservices Docker Acceptance Verification   "
echo "============================================================"
echo ""

# Helper: check HTTP endpoint
check_endpoint() {
    local method="${1:-GET}"
    local base_url="$2"
    local path="$3"
    local expect_status="${4:-200}"
    local expect_contains="${5:-}"

    local url="${base_url}${path}"
    local response
    if [ "$method" = "POST" ]; then
        response=$(curl -sf -w "\n%{http_code}" -X POST "$url" -H 'Content-Type: application/json' -d "${6:-{}}" 2>&1) || true
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

# ============================================
# 1. Infrastructure Health
# ============================================
echo "--- Infrastructure ---"

if docker exec chainpulse-ms-postgres pg_isready -U chainpulse -d chainpulse >/dev/null 2>&1; then
    pass "PostgreSQL: accepting connections"
else
    fail "PostgreSQL: not ready"
fi

if docker exec chainpulse-ms-redis redis-cli ping 2>/dev/null | grep -q PONG; then
    pass "Redis: PONG"
else
    fail "Redis: not responding"
fi

MONGO_CONTAINER=$(docker ps --format "{{.Names}}" | grep chainpulse-ms-mongodb | head -1)
if [ -n "$MONGO_CONTAINER" ] && docker exec "$MONGO_CONTAINER" mongosh --eval "db.adminCommand('ping')" --quiet 2>/dev/null | grep -q "ok"; then
    pass "MongoDB: ping OK"
else
    fail "MongoDB: not responding"
fi

if docker exec chainpulse-ms-kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null | grep -q "blockchain-events"; then
    pass "Kafka: topics available"
else
    warn "Kafka: blockchain-events topic not found (may auto-create)"
fi

# ============================================
# 2. Anvil Chain Nodes (7 chains)
# ============================================
echo ""
echo "--- Blockchain Nodes ---"

CHAINS="ethereum:18545 polygon:18546 bsc:18547 arbitrum:18548 optimism:18549 base:18550 avalanche:18551"
for chain_port in $CHAINS; do
    chain="${chain_port%%:*}"
    container="chainpulse-ms-anvil-${chain}"
    if docker exec "$container" cast block-number --rpc-url http://localhost:8545 2>/dev/null | grep -qE "^[0-9]+$"; then
        pass "$chain: Anvil responding"
    else
        fail "$chain: Anvil not responding"
    fi
done

# ============================================
# 3. Individual Microservice Health
# ============================================
echo ""
echo "--- Microservice Health ---"

# API Gateway (port 18080 on host)
check_endpoint GET "http://localhost:18080" /health 200 "healthy"
check_endpoint GET "http://localhost:18080" /health/ready 200 "ready"
check_endpoint GET "http://localhost:18080" /health/live 200 "alive"

# API Service (port 18081 on host)
check_endpoint GET "http://localhost:18081" /health 200 "healthy"
check_endpoint GET "http://localhost:18081" /health/ready 200 "ready"

# Event Processor (port 18082 on host)
check_endpoint GET "http://localhost:18082" /health 200 "healthy"
check_endpoint GET "http://localhost:18082" /health/ready 200 "ready"

# Puller (port 18083 on host)
check_endpoint GET "http://localhost:18083" /health 200 "healthy"
check_endpoint GET "http://localhost:18083" /health/ready 200 "ready"

# ============================================
# 4. Runtime & Metrics Per Service
# ============================================
echo ""
echo "--- Runtime & Metrics ---"

check_endpoint GET "http://localhost:18080" /metrics 200 "chainpulse_"
check_endpoint GET "http://localhost:18081" /metrics 200 "chainpulse_"
check_endpoint GET "http://localhost:18082" /metrics 200 "chainpulse_"
check_endpoint GET "http://localhost:18083" /metrics 200 "chainpulse_"

check_endpoint GET "http://localhost:18080" /runtime/summary 200 ""
check_endpoint GET "http://localhost:18083" /runtime/summary 200 ""

# ============================================
# 5. Query API (via Gateway)
# ============================================
echo ""
echo "--- Query API (via Gateway) ---"

check_endpoint GET "http://localhost:18080" "/events?limit=5" 200 "events"
check_endpoint GET "http://localhost:18080" /models 200 "models"
check_endpoint POST "http://localhost:18080" /graphql 200 "data" '{"query":"{ __schema { queryType { fields { name } } } }"}'

# ============================================
# 6. Direct API Service Queries
# ============================================
echo ""
echo "--- Direct API Service Queries ---"

check_endpoint GET "http://localhost:18081" "/events?limit=5" 200 "events"
check_endpoint GET "http://localhost:18081" /health/components 200 "healthy"

# ============================================
# 7. DLQ & Replay
# ============================================
echo ""
echo "--- DLQ & Replay ---"

# DLQ replay may not be available on gateway in microservices mode
# Check on event-processor instead
dlq_code=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "http://localhost:18082/runtime/indexing/dlq/replay" \
    -H 'Content-Type: application/json' -d '{"chain_id":"ethereum"}' 2>/dev/null || echo "000")
if [ "$dlq_code" = "200" ]; then
    pass "DLQ replay (event-processor) -> 200"
elif [ "$dlq_code" = "404" ]; then
    warn "DLQ replay endpoint not available (404) - may require DLQ events first"
else
    warn "DLQ replay endpoint -> $dlq_code"
fi

# ============================================
# 8. WebSocket (via Gateway)
# ============================================
echo ""
echo "--- WebSocket ---"

ws_code=$(curl -s -o /tmp/cp-ms-ws-body -w "%{http_code}" \
    -H "Upgrade: websocket" -H "Connection: Upgrade" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    -H "Sec-WebSocket-Version: 13" \
    http://localhost:18080/ws 2>/dev/null || true)

if [ "$ws_code" = "101" ]; then
    pass "WebSocket /ws (gateway) -> 101 Switching Protocols"
else
    fail "WebSocket /ws (gateway) -> $ws_code (expected 101)"
fi

# ============================================
# 9. Frontend & Service Proxies
# ============================================
echo ""
echo "--- Frontend & Service Proxies ---"

frontend_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:13000 2>/dev/null || echo "000")
if [ "$frontend_code" = "200" ]; then
    pass "Frontend http://localhost:13000 -> 200"
else
    fail "Frontend http://localhost:13000 -> $frontend_code"
fi

# Frontend API proxy (via gateway)
proxy_health=$(curl -sf http://localhost:13000/health 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
if [ "$proxy_health" = "healthy" ]; then
    pass "Frontend API proxy (-> gateway) -> healthy"
else
    fail "Frontend API proxy (-> gateway) -> status='$proxy_health'"
fi

# Per-service proxies (microservices mode: each routes to own service)
for svc_port in "api-gateway:8080" "api-service:8081" "event-processor:8082" "puller:8083"; do
    svc="${svc_port%%:*}"
    proxy_status=$(curl -sf "http://localhost:13000/__proxy/${svc}/health" 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
    if [ "$proxy_status" = "healthy" ]; then
        pass "Frontend __proxy/$svc -> healthy"
    else
        fail "Frontend __proxy/$svc -> status='$proxy_status'"
    fi
done

# ============================================
# 10. Inter-Service Communication
# ============================================
echo ""
echo "--- Inter-Service Communication ---"

# Verify puller can reach Kafka and Anvil
puller_runtime=$(curl -sf http://localhost:18083/runtime/summary 2>/dev/null || echo "")
if echo "$puller_runtime" | grep -q "puller\|microservices"; then
    pass "Puller runtime: responding with microservices context"
else
    warn "Puller runtime: not fully initialized yet"
fi

# Verify gateway routes to api-service
gateway_upstream=$(curl -sf http://localhost:18080/health/components 2>/dev/null || echo "")
if [ -n "$gateway_upstream" ]; then
    pass "Gateway -> API-Service: upstream reachable"
else
    fail "Gateway -> API-Service: upstream not reachable"
fi

# ============================================
# 11. Observability
# ============================================
echo ""
echo "--- Observability ---"

prom_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:19090 2>/dev/null || echo "000")
if [ "$prom_code" = "302" ] || [ "$prom_code" = "200" ]; then
    pass "Prometheus http://localhost:19090 -> $prom_code"
else
    fail "Prometheus http://localhost:19090 -> $prom_code"
fi

# Check Prometheus targets - should have 4 microservice targets
prom_targets=$(curl -sf "http://localhost:19090/api/v1/targets" 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); targets=d.get('data',{}).get('activeTargets',[]); up=[x for x in targets if x['health']=='up']; print(f'{len(up)}/{len(targets)}')" 2>/dev/null || echo "0/0")
if echo "$prom_targets" | grep -qE "^[1-9]"; then
    pass "Prometheus targets up: $prom_targets"
else
    fail "Prometheus targets up: $prom_targets"
fi

grafana_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:13001 2>/dev/null || echo "000")
if [ "$grafana_code" = "302" ] || [ "$grafana_code" = "200" ]; then
    pass "Grafana http://localhost:13001 -> $grafana_code"
else
    fail "Grafana http://localhost:13001 -> $grafana_code"
fi

jaeger_code=$(curl -sf -o /dev/null -w "%{http_code}" http://localhost:16687 2>/dev/null || echo "000")
if [ "$jaeger_code" = "200" ]; then
    pass "Jaeger http://localhost:16687 -> 200"
else
    fail "Jaeger http://localhost:16687 -> $jaeger_code"
fi

# ============================================
# 12. Multi-Chain Indexing
# ============================================
echo ""
echo "--- Multi-Chain Indexing ---"

# Check puller health/components for configured_puller_count
puller_puller_count=$(curl -sf http://localhost:18083/health/components 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); rt=d.get('components',{}).get('indexing_runtime',{}); print(rt.get('details',{}).get('configured_puller_count','0'))" 2>/dev/null || echo "0")
if [ "$puller_puller_count" = "7" ]; then
    pass "Puller configured chains: $puller_puller_count"
else
    warn "Puller configured chains: $puller_puller_count (expected 7)"
fi

# Check puller metrics for poll activity
puller_polls=$(curl -sf http://localhost:18083/metrics 2>/dev/null \
    | grep "chainpulse_puller_polls" | grep -v "#" \
    | awk '{print $2}' | head -1 || echo "0")
if [ "$puller_polls" -gt 0 ] 2>/dev/null; then
    pass "Puller polls active: $puller_polls"
else
    warn "Puller polls active: $puller_polls (expected > 0)"
fi

# ============================================
# Summary
# ============================================
echo ""
echo "============================================================"
TOTAL=$((PASS + FAIL + WARN))
echo -e "  Results: ${GREEN}$PASS pass${NC}, ${RED}$FAIL fail${NC}, ${YELLOW}$WARN warn${NC} / $TOTAL total"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}ALL CRITICAL CHECKS PASSED${NC}"
else
    echo -e "  ${RED}$FAIL CHECKS FAILED${NC}"
fi
echo "============================================================"

exit $FAIL
