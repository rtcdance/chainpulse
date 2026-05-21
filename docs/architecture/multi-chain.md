# 多链架构

## 概念区分

ChainPulse 当前的"多链"支持属于 **并行单链** 模式，而非 **原子跨链**。

```
并行单链                         原子跨链
┌────────┐  ┌────────┐           ┌────────────────┐
│ ChainA │  │ ChainB │           │  ChainA → ChainB │
│ Indexer│  │ Indexer│           │  消息中继 + 证明 │
└────────┘  └────────┘           └────────────────┘
各自独立运行                     需要共识层交互
```

## 并行单链（当前实现）

### 架构

```
MultiChainIndexer
  ├── ChainIndexer("ethereum")  → HTTPSJSONRPCPuller + EventProcessor
  ├── ChainIndexer("polygon")   → HTTPSJSONRPCPuller + EventProcessor
  └── ChainIndexer("solana")    → SolanaPuller + EventProcessor
```

每个链：
- 有独立的 `DataPullerPlugin` 实例（通过 `ChainID()` 标识）
- 有独立的 checkpoint 队列
- 事件互不相干，不会跨链合并

### 链配置

通过 `BlockchainConfig` 加载：

```go
config.Blockchains["ethereum"] = BlockchainConfig{
    ChainID:    "1",
    NodeURL:    "https://eth-mainnet.g.alchemy.com/...",
    StartBlock: 18_000_000,
}
```

## 原子跨链（不在当前范围）

真正的跨链操作需要：

1. **轻客户端验证**: 在 ChainA 上验证 ChainB 的区块头（如无需信任的桥）
2. **Merkle 证明验证**: 验证 ChainB 的交易的 inclusion proof
3. **最终性差异处理**: 不同链的最终性时间不同（如以太坊 2 epochs vs Solana 32 slots）
4. **消息路由**: 将 ChainA 的事件中继到 ChainB 的合约

如果需要实现跨链功能，建议路径：

```
Phase 1: 添加 BlockHasProvider 的多链路由（支持每条链独立 RPC）
Phase 2: 实现 SPV (Simplified Payment Verification) 验证器
Phase 3: 构建跨链消息路由器 (IBC 或类似模式)
```

## 当前边界

| 能力 | 状态 | 说明 |
|---|---|---|
| 多链并行索引 | ✅ | 通过 MultiChainIndexer 实现 |
| 链独立 RPC | ✅ | 每条链配置独立的 NodeURL |
| 跨链查询 | ❌ | 不支持跨链事件关联查询 |
| 原子跨链 | ❌ | 不在当前路线图上 |
| 跨链重组一致 | ⚠️ | 各链独立处理重组，互不影响 |

## 设计原则

1. **链隔离**: 每条链的数据在存储层面隔离（chain_id 字段分区）
2. **统一接口**: 所有链通过 `DataPullerPlugin` 接口暴露数据
3. **独立 checkpoint**: 每条链维护自己的索引进度
4. **差异文档化**: 链间的架构差异在 puller 各自的文档中说明
