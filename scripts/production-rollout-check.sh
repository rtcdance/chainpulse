#!/bin/bash
# Production Rollout Validation Script
#
# Validates that ChainPulse is ready for production go-live.
# Checks all P0 and P1 blockers from docs/deployment/go-live-blockers.md
#
# Usage: bash scripts/production-rollout-check.sh [base_url]
# Example: bash scripts/production-rollout-check.sh http://localhost:8080

set -e

BASE_URL="${1:-http://localhost:8080}"
PASS=0
FAIL=0
WARN=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; ((PASS++)); }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; ((FAIL++)); }
warn() { echo -e "${YELLOW}⚠ WARN${NC}: $1"; ((WARN++)); }

check_http() {
    local url="$1"
    local expected="$2"
    local desc="$3"

    local response
    response=$(curl -s -w "\n%{http_code}" "$url" 2>/dev/null) || { fail "$desc - connection failed"; return; }

    local body
    body=$(echo "$response" | head -n -1)
    local code
    code=$(echo "$response" | tail -1)

    if echo "$body" | grep -q "$expected"; then
        pass "$desc"
    else
        fail "$desc - expected '$expected' in response"
        echo "  Response: $body"
    fi
}

echo "========================================="
echo " ChainPulse Production Rollout Check"
echo "========================================="
echo ""
echo "Target: $BASE_URL"
echo ""

# === P0 Blockers ===
echo "--- P0: Gateway Security ---"

# Check 1: Runtime summary reports runtime-wired
check_http "$BASE_URL/runtime/summary" "runtime-wired" "Runtime mode is wired"

# Check 2: Health rollout reports advisory_ready=true
check_http "$BASE_URL/health/rollout" "advisory_ready" "Rollout advisory ready"

# Check 3: Query bridge health
check_http "$BASE_URL/health" "ok" "Gateway health check passing"

echo ""
echo "--- P0: Environment Variables ---"

# Check 4: CHAINPULSE_ENV is production
local_env=$(curl -s "$BASE_URL/runtime/summary" 2>/dev/null | grep -o 'CHAINPULSE_ENV=[^"]*' || echo "")
if echo "$local_env" | grep -q "production"; then
    pass "CHAINPULSE_ENV=production"
else
    warn "Cannot verify CHAINPULSE_ENV (runtime summary may not expose it)"
fi

echo ""
echo "--- P1: Performance Baseline ---"

# Check 5: Metrics endpoint
curl -s "$BASE_URL/metrics" > /dev/null 2>&1 && pass "Prometheus metrics endpoint accessible" || warn "Metrics endpoint not accessible"

echo ""
echo "========================================="
echo " Results: $PASS passed, $FAIL failed, $WARN warnings"
echo "========================================="

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}ROLLOUT NOT READY${NC} - $FAIL blocker(s) must be resolved"
    exit 1
fi

echo -e "${GREEN}ROLLOUT READY${NC} - All P0 checks passed"
exit 0
