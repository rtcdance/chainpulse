#!/usr/bin/env bash
# ChainPulse Microservices Docker Acceptance - One-click Management Script
# Usage: bash docker/acceptance-microservices.sh [build|up|down|status|verify|logs|inject]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.acceptance-microservices.yml"

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
    info "Building ChainPulse microservice Linux binaries..."
    mkdir -p "$PROJECT_ROOT/build/bin/linux/microservices"

    for svc in puller event-processor api-service api-gateway; do
        info "  Compiling $svc..."
        CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo \
            -o "$PROJECT_ROOT/build/bin/linux/microservices/$svc" \
            "./cmd/microservices/$svc"
        local size
        size=$(ls -lh "$PROJECT_ROOT/build/bin/linux/microservices/$svc" | awk '{print $5}')
        info "  $svc: $size"
    done

    info "Building 4 microservice Docker images..."
    for svc in puller event-processor api-service api-gateway; do
        info "  Building chainpulse-$svc:latest..."
        DOCKER_BUILDKIT=0 docker build --build-arg "SERVICE=$svc" \
            -f "$SCRIPT_DIR/Dockerfile.microservices.prebuilt" \
            -t "chainpulse-$svc:latest" "$PROJECT_ROOT" 2>&1 | tail -1
    done

    info "Building frontend static files..."
    cd "$PROJECT_ROOT/frontend"
    npm ci --quiet 2>/dev/null
    npm run build

    info "Building frontend Docker image (microservices mode)..."
    cp nginx.microservices.conf nginx.conf
    DOCKER_BUILDKIT=0 docker build -f Dockerfile.prebuilt \
        -t chainpulse-frontend-microservices:latest .

    cd "$PROJECT_ROOT"
    info "All images built successfully!"
    docker images | grep -E "chainpulse-(puller|event-processor|api-service|api-gateway|frontend-microservices)"
}

# ============================================
# Start full stack
# ============================================
stack_up() {
    info "Starting ChainPulse microservices acceptance stack..."
    docker compose -f "$COMPOSE_FILE" up -d --remove-orphans 2>/dev/null || \
        docker compose -f "$COMPOSE_FILE" up -d

    info "Waiting for services to become healthy (this may take a few minutes)..."
    local max_wait=300
    local elapsed=0
    while [ $elapsed -lt $max_wait ]; do
        local total
        total=$(docker compose -f "$COMPOSE_FILE" ps --format "{{.Name}}" 2>/dev/null | wc -l | tr -d ' ')
        if [ "$total" -gt 5 ]; then
            local unhealthy
            unhealthy=$(docker compose -f "$COMPOSE_FILE" ps --format "{{.Name}} {{.Status}}" 2>/dev/null \
                | grep -v "healthy" | grep -v "Up " | grep -c "." || true)
            if [ "$unhealthy" -eq 0 ]; then
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
    warn "Some services may still be starting. Check status with: bash docker/acceptance-microservices.sh status"
    stack_status
}

# ============================================
# Stop full stack
# ============================================
stack_down() {
    info "Stopping ChainPulse microservices acceptance stack..."
    docker compose -f "$COMPOSE_FILE" down --remove-orphans -v
    info "Stack stopped and volumes removed."
}

# ============================================
# Show stack status
# ============================================
stack_status() {
    echo ""
    echo "=== ChainPulse Microservices Acceptance Stack Status ==="
    docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || \
        echo "Stack is not running."
    echo ""

    # Quick health checks per service
    for svc_port in "api-gateway:18080" "api-service:18081" "event-processor:18082" "puller:18083"; do
        local svc="${svc_port%%:*}"
        local port="${svc_port##*:}"
        if curl -sf "http://localhost:$port/health" >/dev/null 2>&1; then
            local status
            status=$(curl -sf "http://localhost:$port/health" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','unknown'))" 2>/dev/null || echo "unknown")
            info "$svc (:$port): $status"
        else
            warn "$svc (:$port): not reachable"
        fi
    done

    if curl -sf http://localhost:13000 >/dev/null 2>&1; then
        info "Frontend: accessible on http://localhost:13000"
    else
        warn "Frontend not reachable on port 13000"
    fi
}

# ============================================
# Inject test transactions
# ============================================
inject_events() {
    info "Injecting test transactions into Anvil nodes..."

    local chains="ethereum:18545 polygon:18546 bsc:18547"
    local key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    local to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

    for chain_port in $chains; do
        local chain="${chain_port%%:*}"
        local port="${chain_port##*:}"
        local container="chainpulse-ms-anvil-${chain}"

        info "  Sending transactions to $chain (host port $port)..."
        for i in 1 2 3; do
            docker exec "$container" cast send \
                --rpc-url http://localhost:8545 \
                --private-key "$key" \
                --value 0ether \
                "$to" 2>&1 | grep -E "blockNumber|transactionHash" || true
        done
    done

    info "Transactions injected. Events will flow: Puller -> Kafka -> Event-Processor -> DB"
    info "Query events via gateway: curl http://localhost:18080/events?limit=10"
}

# ============================================
# Show logs
# ============================================
stack_logs() {
    local service="${1:-}"
    if [ -n "$service" ]; then
        docker compose -f "$COMPOSE_FILE" logs -f "$service"
    else
        docker compose -f "$COMPOSE_FILE" logs -f api-gateway api-service event-processor puller
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
        bash "$SCRIPT_DIR/verify-acceptance-microservices.sh"
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
        echo "ChainPulse Microservices Docker Acceptance - One-click Management"
        echo ""
        echo "Usage: bash docker/acceptance-microservices.sh <command>"
        echo ""
        echo "Commands:"
        echo "  build    Build Docker images (4 microservice binaries + frontend)"
        echo "  up       Start full microservices acceptance stack"
        echo "  down     Stop stack and remove volumes"
        echo "  status   Show stack status and health per service"
        echo "  verify   Run acceptance verification tests"
        echo "  logs     Show logs (optional: service name)"
        echo "  inject   Inject test transactions into Anvil nodes"
        echo "  simulate Start/stop continuous event simulation (start|stop|status)"
        echo ""
        echo "Quick start:"
        echo "  bash docker/acceptance-microservices.sh build"
        echo "  bash docker/acceptance-microservices.sh up"
        echo "  bash docker/acceptance-microservices.sh verify"
        echo "  bash docker/acceptance-microservices.sh simulate start"
        echo ""
        echo "Ports (microservices mode):"
        echo "  Frontend:       http://localhost:13000"
        echo "  API Gateway:    http://localhost:18080"
        echo "  API Service:    http://localhost:18081"
        echo "  Event Processor:http://localhost:18082"
        echo "  Puller:         http://localhost:18083"
        echo "  PostgreSQL:     localhost:15432"
        echo "  Redis:          localhost:16379"
        echo "  MongoDB:        localhost:27018"
        echo "  Kafka:          localhost:19092"
        echo "  Prometheus:     http://localhost:19090"
        echo "  Grafana:        http://localhost:13001"
        echo "  Jaeger:         http://localhost:16687"
        ;;
esac
