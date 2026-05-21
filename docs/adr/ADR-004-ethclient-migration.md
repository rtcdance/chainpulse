# ADR-004: ethclient 迁移 — 从手写 JSON-RPC 到 go-ethereum 官方库

**Date**: 2026-05-13

### Status

Accepted

### Context

ChainPulse 最初使用手写 JSON-RPC 客户端与以太坊节点通信。`HTTPSJSONRPCPuller` 中包含了 `sendRequest()`、`sendBatchRequest()`、`hexToUint64()`、`uint64ToHex()` 等约 350 行手写 RPC 代码，覆盖 `eth_blockNumber`、`eth_getLogs`、`eth_getBlockByNumber` 等调用。

这种方案的问题：

1. **类型安全缺失**：所有 RPC 响应以 `map[string]interface{}` 或 `[]byte` 返回，字段访问依赖运行时类型断言和十六进制解析
2. **维护成本**：每个新 RPC 方法都需要手写请求构造和响应解析
3. **订阅支持**：WebSocket 订阅（`eth_subscribe`）需要额外的连接管理和心跳逻辑
4. **版本兼容**：以太坊节点 RPC 接口演进（如 Paris/Shanghai 升级后的新字段）需要手动跟踪

`go-ethereum` 的 `ethclient` 库是官方维护的以太坊 RPC 客户端，封装了所有标准 JSON-RPC 方法，提供类型安全的 Go 接口。

### Decision

将所有以太坊 RPC 调用从手写 JSON-RPC 迁移到 `go-ethereum/ethclient`：

| 功能 | 手写实现 | ethclient 替代 |
|------|---------|---------------|
| 获取最新区块号 | `eth_blockNumber` → `hexToUint64` | `ethClient.BlockNumber(ctx)` |
| 查询事件日志 | `eth_getLogs` → 手写 FilterLogs 参数 | `ethClient.FilterLogs(ctx, ethereum.FilterQuery)` |
| 获取区块头 | `eth_getBlockByNumber` → 手写 BlockHeader 解析 | `ethClient.HeaderByNumber(ctx, big.NewInt(n))` |
| 链 ID 验证 | `eth_chainId` → 手写 hex 解析 | `ethClient.ChainID(ctx)` |
| 新区块订阅 | 手写 WebSocket 管理 | `ethClient.SubscribeNewHead(ctx, headCh)` |
| 连接管理 | `*http.Client` + 手动重连 | `ethclient.DialContext(ctx, url)` + `ethClient.Close()` |

关键设计决策：

- **连接延迟**：`ethclient.DialContext()` 在 `Start()` 中调用，而非构造函数中，避免启动时阻塞
- **WebSocket 回退**：如果节点 URL 以 `ws://` 或 `wss://` 开头，`Poll()` 自动使用 `SubscribeNewHead` 推送模式，回退到轮询
- **共享类型**：WebSocket 拉取器仍使用手写 JSON-RPC 类型（`JSONRPCError`、`BlockHeader`、`Log`），这些类型保留在 `jsonrpc_helpers.go` 中
- **Parent hash 链验证**：`verifyParentHashChain()` 使用 `ethClient.HeaderByNumber` 替代手写区块头请求

### 代码结构变化

```
移除（~350 行）：
  https_jsonrpc_puller.go:
    - sendRequest(), sendBatchRequest()
    - JSONRPCRequest, JSONRPCResponse, JSONRPCError 类型
    - BlockHeader, Log 类型
    - hexToUint64(), uint64ToHex()
    - 所有手写 RPC 调用方法

保留（WebSocket 拉取器仍需要）：
  jsonrpc_helpers.go（新增）:
    - JSONRPCError, BlockHeader, Log
    - hexToUint64(), uint64ToHex()
    - JSONRPCRequest, JSONRPCResponse

新增（~80 行）：
  https_jsonrpc_puller.go:
    - *ethclient.Client 字段
    - ethLogToEvent() 替代 logToEvent()
    - pollWithSubscription() WebSocket 推送模式
    - GetBlockHeader() 导出方法返回 *types.Header
```

### Consequences

- **Positive**：类型安全 — `types.Log`、`types.Header` 等强类型替代 `map[string]interface{}`
- **Positive**：减少 ~270 行手写 RPC 代码
- **Positive**：官方维护 — 自动兼容以太坊节点升级
- **Positive**：原生 WebSocket 订阅支持 — `SubscribeNewHead` 替代手写 WebSocket 管理
- **Negative**：增加对 `go-ethereum` 的依赖版本敏感度（v1.16.8）
- **Negative**：`ethclient.FilterLogs` 的 `ethereum.FilterQuery` 参数构造不如手写灵活（如不支持某些非标准过滤器）
- **Neutral**：测试需要模拟 `ethclient` 接口而非简单的 HTTP mock

### 相关 ADR

- ADR-001：插件架构 — `HTTPSJSONRPCPuller` 作为 `DataPullerPlugin` 实现
- ADR-002：EventBus — 拉取器通过 EventBus 发布 `blockchain-events` 和 `reorg-rollback` 事件
- ADR-003：Reorg 处理 — `GetBlockHeader()` 为重组检测提供区块哈希查询