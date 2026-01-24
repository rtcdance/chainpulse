#!/bin/bash

# Local Unit Tests Script
# Runs unit tests locally with proper timeouts to avoid CI delays

set -e

echo "=== Running Local Unit Tests ==="
echo "Starting at $(date)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
UNIT_TIMEOUT="5m"
INTEGRATION_TIMEOUT="10m"
E2E_TIMEOUT="15m"
RACE_DETECTOR="-race"

# Counters
PASSED=0
FAILED=0

# Function to run tests
run_tests() {
    local package=$1
    local timeout=$2
    local name=$3
    
    echo ""
    echo -e "${YELLOW}Testing ${name}...${NC}"
    
    # Create coverage directory if it doesn't exist
    mkdir -p coverage
    
    # Sanitize name for filename
    local safe_name=$(echo ${name} | sed 's/[^a-zA-Z0-9]/_/g')
    
    if go test -v ${RACE_DETECTOR} -coverprofile=coverage/coverage-${safe_name}.out -timeout ${timeout} ${package} 2>&1; then
        echo -e "${GREEN}✓ ${name} passed${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗ ${name} failed${NC}"
        ((FAILED++))
        return 1
    fi
}

# Run unit tests
echo ""
echo -e "${YELLOW}=== UNIT TESTS ===${NC}"
run_tests "./pkg/core" "${UNIT_TIMEOUT}" "pkg/core" || true
run_tests "./pkg/infrastructure/..." "${UNIT_TIMEOUT}" "pkg/infrastructure" || true
run_tests "./pkg/plugins/cache" "${UNIT_TIMEOUT}" "pkg/plugins/cache" || true
run_tests "./pkg/plugins/database" "${UNIT_TIMEOUT}" "pkg/plugins/database" || true
run_tests "./pkg/plugins/mq" "${UNIT_TIMEOUT}" "pkg/plugins/mq" || true
run_tests "./pkg/plugins/api/core" "${UNIT_TIMEOUT}" "pkg/plugins/api/core" || true
run_tests "./pkg/services/query" "${UNIT_TIMEOUT}" "pkg/services/query" || true

# Run integration tests (optional, can be skipped for faster feedback)
echo ""
echo -e "${YELLOW}=== INTEGRATION TESTS (Optional) ===${NC}"
if [ "${SKIP_INTEGRATION}" != "true" ]; then
    run_tests "./test/integration" "${INTEGRATION_TIMEOUT}" "integration" || true
else
    echo "Skipping integration tests (SKIP_INTEGRATION=true)"
fi

# Summary
echo ""
echo -e "${YELLOW}=== TEST SUMMARY ===${NC}"
echo -e "Passed: ${GREEN}${PASSED}${NC}"
echo -e "Failed: ${RED}${FAILED}${NC}"
echo "Completed at $(date)"

if [ ${FAILED} -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
