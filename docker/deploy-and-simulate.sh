#!/usr/bin/env bash
# ChainPulse Deploy & Simulate
# Dispatches to mode-specific scripts.
# Usage: bash docker/deploy-and-simulate.sh [microservices|monolith|stop|status]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-microservices}"

case "$MODE" in
    monolith|monolithic)
        info "[deploy-and-simulate] Delegating to deploy-monolith.sh..."
        exec bash "$SCRIPT_DIR/deploy-monolith.sh" "${@:2}"
        ;;
    microservices|microservice|ms)
        exec bash "$SCRIPT_DIR/deploy-microservices.sh" "${@:2}"
        ;;
    stop)
        warn "Stopping both stacks..."
        bash "$SCRIPT_DIR/deploy-microservices.sh" stop 2>/dev/null || true
        bash "$SCRIPT_DIR/deploy-monolith.sh" stop 2>/dev/null || true
        ;;
    status)
        echo "=== Microservices stack ==="
        bash "$SCRIPT_DIR/deploy-microservices.sh" status 2>/dev/null || echo "(not running)"
        echo ""
        echo "=== Monolith stack ==="
        bash "$SCRIPT_DIR/deploy-monolith.sh" status 2>/dev/null || echo "(not running)"
        ;;
    *)
        echo "Usage: bash docker/deploy-and-simulate.sh [microservices|monolith|stop|status]"
        echo ""
        echo "  microservices (default)  Deploy 4-service microservices stack + simulate"
        echo "  monolith                 Deploy single-binary monolithic stack + simulate"
        echo "  stop                     Stop all stacks"
        echo "  status                   Show status of all stacks"
        exit 1
        ;;
esac
