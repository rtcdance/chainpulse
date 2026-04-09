#!/bin/bash
set -e

echo "=== ChainPulse 多链 E2E 环境启动 ==="

CHAINS=(
  "ethereum:8545:1"
  "polygon:8546:137"
  "bsc:8547:97"
  "arbitrum:8548:421614"
  "optimism:8549:11155420"
  "base:8550:84532"
  "avalanche:8551:43113"
)

PIDS=()
cleanup() {
    echo "=== 清理进程 ==="
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    echo "已清理所有 Anvil 进程"
}
trap cleanup EXIT

for chain_info in "${CHAINS[@]}"; do
    IFS=':' read -r name port chain_id <<< "$chain_info"
    echo "启动 $name (port=$port, chain-id=$chain_id)..."
    anvil --port "$port" --chain-id "$chain_id" --timestamp 0 --host 0.0.0.0 &
    PIDS+=($!)
    sleep 1
done

sleep 3

echo ""
echo "=== 多链环境已启动 ==="
echo "Ethereum:      http://localhost:8545 (chain-id: 1)"
echo "Polygon:       http://localhost:8546 (chain-id: 137)"
echo "BSC:           http://localhost:8547 (chain-id: 97)"
echo "Arbitrum:      http://localhost:8548 (chain-id: 421614)"
echo "Optimism:      http://localhost:8549 (chain-id: 11155420)"
echo "Base:          http://localhost:8550 (chain-id: 84532)"
echo "Avalanche:     http://localhost:8551 (chain-id: 43113)"
echo ""

# Run tests
echo "=== 运行 E2E 多链测试 ==="
cd /Users/mingo/Applications/workspace/web3/project/chainpulse
go test ./test/e2e/... -run "TestMultiChain|TestChainConfig|TestAnvil" -v -count=1

echo ""
echo "=== 测试完成 ==="
