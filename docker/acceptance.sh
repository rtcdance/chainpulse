#!/usr/bin/env bash
# ChainPulse Docker Acceptance - One-click Management Script
# Usage: bash docker/acceptance.sh [build|up|down|status|verify|logs|inject]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.acceptance.yml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ============================================
# Build images (local binary + cached base)
# ============================================
build_images() {
    info "Building ChainPulse Linux binary..."
    mkdir -p "$PROJECT_ROOT/build/bin/linux"
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo \
        -o "$PROJECT_ROOT/build/bin/linux/chainpulse" ./cmd/monolithic/chainpulse
    # Dockerfile.prebuilt expects the binary at project root as chainpulse-linux
    cp "$PROJECT_ROOT/build/bin/linux/chainpulse" "$PROJECT_ROOT/chainpulse-linux"
    info "Binary built: $(ls -lh "$PROJECT_ROOT/build/bin/linux/chainpulse" | awk '{print $5}')"

    info "Building ChainPulse Docker image..."
    DOCKER_BUILDKIT=0 docker build -f "$SCRIPT_DIR/Dockerfile.prebuilt" \
        -t chainpulse-acceptance:latest "$PROJECT_ROOT"
    rm -f "$PROJECT_ROOT/chainpulse-linux"

    info "Building frontend static files..."
    cd "$PROJECT_ROOT/frontend"
    npm ci --quiet 2>/dev/null
    npm run build

    info "Building frontend Docker image..."
    DOCKER_BUILDKIT=0 docker build -f Dockerfile.prebuilt \
        -t chainpulse-frontend:latest .

    cd "$PROJECT_ROOT"
    info "All images built successfully!"
    docker images | grep -E "chainpulse-acceptance|chainpulse-frontend"
}

# ============================================
# Start full stack
# ============================================
stack_up() {
    info "Starting ChainPulse acceptance stack..."
    # Remove orphan containers from previous microservice runs
    docker compose -f "$COMPOSE_FILE" up -d --remove-orphans 2>/dev/null || \
        docker compose -f "$COMPOSE_FILE" up -d

    info "Waiting for services to become healthy..."
    local max_wait=180
    local elapsed=0
    while [ $elapsed -lt $max_wait ]; do
        local unhealthy
        unhealthy=$(docker compose -f "$COMPOSE_FILE" ps --format "{{.Name}} {{.Status}}" 2>/dev/null \
            | grep -v "healthy" | grep -v "Up " | grep -c "." || true)
        if [ "$unhealthy" -eq 0 ]; then
            local total
            total=$(docker compose -f "$COMPOSE_FILE" ps --format "{{.Name}}" 2>/dev/null | wc -l | tr -d ' ')
            if [ "$total" -gt 5 ]; then
                info "All services are healthy!"
                stack_status
                return 0
            fi
        fi
        sleep 5
        elapsed=$((elapsed + 5))
        echo -n "."
    done
    echo ""
    warn "Some services may still be starting. Check status with: bash docker/acceptance.sh status"
    stack_status
}

# ============================================
# Stop full stack
# ============================================
stack_down() {
    info "Stopping ChainPulse acceptance stack..."
    docker compose -f "$COMPOSE_FILE" down --remove-orphans -v
    info "Stack stopped and volumes removed."
}

# ============================================
# Show stack status
# ============================================
stack_status() {
    echo ""
    echo "=== ChainPulse Acceptance Stack Status ==="
    docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || \
        echo "Stack is not running."
    echo ""

    # Quick health check
    if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
        local status
        status=$(curl -sf http://localhost:8080/health | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])" 2>/dev/null || echo "unknown")
        info "API Health: $status"
    else
        warn "API not reachable on port 8080"
    fi

    if curl -sf http://localhost:3000 >/dev/null 2>&1; then
        info "Frontend: accessible on http://localhost:3000"
    else
        warn "Frontend not reachable on port 3000"
    fi
}

# ============================================
# Inject test transactions
# ============================================
inject_events() {
    info "Injecting test transactions into Anvil nodes..."

    local chains="ethereum:8545 polygon:8546 bsc:8547"
    local key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    local to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

    for chain_port in $chains; do
        local chain="${chain_port%%:*}"
        local port="${chain_port##*:}"
        local container="chainpulse-anvil-${chain}"

        info "  Sending transactions to $chain (port $port)..."
        for i in 1 2 3; do
            docker exec "$container" cast send \
                --rpc-url http://localhost:8545 \
                --private-key "$key" \
                --value 0ether \
                "$to" 2>&1 | grep -E "blockNumber|transactionHash" || true
        done
    done

    info "Transactions injected. Events will be indexed shortly."
    info "Query events: curl http://localhost:8080/events?limit=10"
}

# ============================================
# Show logs
# ============================================
stack_logs() {
    local service="${1:-}"
    if [ -n "$service" ]; then
        docker compose -f "$COMPOSE_FILE" logs -f "$service"
    else
        docker compose -f "$COMPOSE_FILE" logs -f chainpulse
    fi
}

# ============================================
# Main
# ============================================
case "${1:-help}" in
    build)
        build_images
        ;;
    up)
        stack_up
        ;;
    down)
        stack_down
        ;;
    status)
        stack_status
        ;;
    verify)
        bash "$SCRIPT_DIR/verify-acceptance.sh"
        ;;
    logs)
        stack_logs "${2:-}"
        ;;
    inject)
        inject_events
        ;;
    simulate)
        bash "$SCRIPT_DIR/simulate-events.sh" "${2:-start}"
        ;;
    help|*)
        echo "ChainPulse Docker Acceptance - One-click Management"
        echo ""
        echo "Usage: bash docker/acceptance.sh <command>"
        echo ""
        echo "Commands:"
        echo "  build    Build Docker images (Go binary + frontend)"
        echo "  up       Start full acceptance stack"
        echo "  down     Stop stack and remove volumes"
        echo "  status   Show stack status and health"
        echo "  verify   Run acceptance verification tests"
        echo "  logs     Show logs (optional: service name)"
        echo "  inject   Inject test transactions into Anvil nodes"
        echo "  simulate Start/stop continuous event simulation (start|stop|status)"
        echo ""
        echo "Quick start:"
        echo "  bash docker/acceptance.sh build"
        echo "  bash docker/acceptance.sh up"
        echo "  bash docker/acceptance.sh verify"
        ;;
esac
