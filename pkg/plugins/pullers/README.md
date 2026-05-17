# Pullers — 区块链数据拉取器

## Web3 概念

区块链节点提供 JSON-RPC 接口查询链上数据。ChainPulse 的 Pullers 负责：

1. **轮询（Polling）**：定期调用 `eth_blockNumber` → `eth_getLogs` 获取新区块的事件日志
2. **订阅（Subscription）**：通过 WebSocket `eth_subscribe` 实时接收新区块头
3. **重组回退**：当检测到链重组时，从 reorg 点重新拉取事件

## 架构

```
DataPullerPlugin（接口）
  ├── HTTPSJSONRPCPuller  — HTTP/HTTPS 轮询（生产主力）
  ├── WebSocketJSONRPCPuller — WebSocket 推送 + 轮询回退
  ├── GRPCPuller          — gRPC 协议（预留）
  └── SolanaPuller        — Solana 链（预留）
```

## Go 要点

| 模式 | 文件 | 说明 |
|------|------|------|
| `ethclient` 集成 | `https_jsonrpc_puller.go` | 使用 go-ethereum 官方库替代手写 RPC |
| 生命周期 Context | `https_jsonrpc_puller.go:39-41` | `lifecycleCtx` 管理 goroutine 生命周期 |
| errgroup 并发 | `multi_chain_puller.go` | `golang.org/x/sync/errgroup` 并发拉取多链 |
| 指数退避 | 各 puller 循环 | 失败后递增等待，上限 5s |
| 插件接口 | `data_puller.go` | `BaseDataPullerPlugin` 提供通用字段和方法 |

## 学习路径

1. 从 `https_jsonrpc_puller.go:Start()` 开始，看如何连接节点
2. 跟踪 `Poll()` → `GetLatestBlock()` → `FilterLogs()` 的数据流
3. 看 `ethLogToEvent()` 如何将 `types.Log` 转换为 `BlockchainEvent`
4. 看 `PublishEvent()` 如何通过 EventBus 分发事件

## 关键文件

| 文件 | 功能 |
|------|------|
| `https_jsonrpc_puller.go` | HTTP ethclient 拉取器（670 行） |
| `websocket_jsonrpc_puller.go` | WebSocket 拉取器（850 行） |
| `multi_chain_puller.go` | 多链并发拉取编排 |
| `data_puller.go` | 基础插件 + 事件发布 + 检查点持久化 |
| `rpc_block_hash_provider.go` | 为重组检测提供链上区块哈希 |
| `jsonrpc_helpers.go` | WebSocket 拉取器保留的共享 RPC 类型 |