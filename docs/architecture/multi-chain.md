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


## PostgreSQL EVM 完整区块协议（交付方案 T05）

`DATABASE_TYPE=postgres` 的 native puller/processor 使用 `chainpulse.block.v1` 消息：canonical header、明确 chain key 和该块完整 logs；空块也有消息。默认输入 topic 为 `chainpulse.blocks.v1`，提交通知为 `chainpulse.committed-blocks.v1`。滚动迁移必须先部署支持新 schema 的 consumer，再启用新 topic 的 producer；旧 raw-event consumers 保持在原 topic，不能把两种 payload 混写到同一个 topic。其他数据库路径保留 legacy 格式，不属于本轮已验证的生产组合。

消息键就是解析后的正数 EVM chain ID。Kafka writer 使用 Hash 与 RequireAll，consumer 验证实际 record key、partition header 和 frame chain 一致。消费 callback 等待完整块事务与提交通知发布之后才返回成功，重投依赖 PostgreSQL physical event key 幂等效果。

两个模式共同调用 `application/indexing.BuildCompletedRange` 与 `SharedRuntime.ProcessCompleteBlock`。同一 fenced transaction 保存原始日志、canonical block hash/parent/time 和 checkpoint；父哈希冲突、序号空洞或 header 写入失败都会回滚，不允许用独立 metadata 写入补齐半个事务。未经 durable reorg 不能直接更换同高分支。

该接线已验证 native puller→Kafka→processor 的正常、停机积压/重启、空块和 SIGTERM。完整 API mode parity、双链和 Kafka rebalance 的 revocation 生命周期尚待交付验收，不能据此声称四服务全场景等价。


### Canonical query and cursor wiring

API-service selects the canonical PostgreSQL query runtime when `DATABASE_TYPE=postgres`. List, physical-ID detail, counts and statistics use the same declared chain scope; empty results and database failures never select legacy Mongo data. Statistics are aggregated in one repeatable-read snapshot instead of materializing all events. The frontend `chain_id` filter and camelCase aliases resolve to the existing filter, with the old parameter names retaining precedence.

REST `/events?cursor=...` and GraphQL `events(after:...)` issue `chainpulse.query-cursor.v1` tokens. Each token binds one chain, the query filter digest, canonical generation and descending `(block_number, log_index, id)` position. Generation, anchor validation and page reads share one PostgreSQL snapshot. Changed filters/chain return an invalid-cursor error; changed generation or a missing canonical anchor requires restarting pagination. Bounded offset and legacy reader cursors remain compatibility paths. This does not imply generation-safe hot caching or subscription recovery is already accepted.

### Native canonical supervision and terminal orphan frames

The PostgreSQL event processor requires `CHAINS=31337,31338` and explicit `BLOCKCHAIN_RPCS=31337=http://rpc-a:8545,31338=http://rpc-b:8545`. Startup probes each chain ID and genesis; missing, duplicate or mismatched identities stop startup. Legacy database modes keep their existing configuration. Provision `chainpulse.blocks.v1`, `chainpulse.committed-blocks.v1` and the reorg topic before enabling the producer; Kafka auto-creation is disabled by the client.

The native puller compares its durable checkpoint with RPC on every poll, including a same-height head. It uses the same bounded canonical ancestor scan as the monolith, and applies rollback with an expected checkpoint hash under the writer fence. A stale supervisor cannot revoke an already repaired branch. Restart resumes from PostgreSQL; a divergent unpublished/uncommitted intake tip replays from that durable position. `REORG_DETECTION_DEPTH` bounds the scan. Unknown ancestors, RPC errors and finalized effects stop progress and require investigation rather than guessing a rollback.

The processor validates each complete frame against RPC inside the fenced transaction, before writing effects. Expired leases cannot open a transaction. A stale branch rolls back, then the complete frame is recorded idempotently in `dlq_events` with terminal status `orphaned`; only successful audit storage permits input acknowledgement. Such frames never become committed-block notifications. RPC/database failures remain retryable processing failures and cannot turn into orphan acknowledgements. Repair jobs become done after canonical replay reaches their old high-water mark.

The puller drains at most 100 durable notices per poll, including idle polls, to `KAFKA_REORG_TOPIC` (default `chainpulse.reorg.v1`). Each chain-keyed versioned envelope includes notification ID, job ID, chain, generation, kind and original JSON payload. The outbox row is marked only after RequireAll publication; failures retain the row and increment its attempts. Duplicate delivery remains possible after a crash, so consumers must deduplicate by notification ID and use generation when invalidating derived state.

Recovery checks: compare `indexing_state` and RPC headers; inspect unfinished `reorg_jobs`, unpublished `reorg_outbox` rows and terminal orphan audit records; restore RPC/Kafka access or provision the missing topic; restart the puller to replay from the durable checkpoint. Do not delete the audit/job/outbox tables to clear an alarm. A binary rollback must preserve readers for this complete-block and reorg envelope protocol, or use the previous separate topics with an explicit reindex boundary.


Queued orphan frames may sit above the rewound checkpoint. The shared runtime
reports that ordering failure as `BlockGapError`; the native processor probes
only this rejection against RPC. A confirmed orphan still requires durable
audit before acknowledgement. A canonical gap, an unavailable RPC, or a failed
audit remains an error and cannot advance broker progress. Normal frames keep
their canonical check inside the fenced transaction. This prevents stale queued
blocks from blocking the replacement branch without accepting missing canonical
blocks.

### Broker generation ownership for PostgreSQL processing

The PostgreSQL processor accepts exactly one authoritative complete-block input
 topic. It requires generation-owned consumption; it cannot fall back to the
legacy callback that only carries a process-wide context. Each broker generation
assigns a distinct bounded owner token. Revocation cancels handler/RPC/SQL work,
waits for partition handlers to exit, then releases chain leases before the next
generation starts its effects. An expired or released lease cannot open a
transaction; released rows retain their monotonically increasing fencing history.

The owned consumer has at most 16 assigned partition readers, queue capacity 1
per reader and a maximum fetch size of 10 MB. Exceeding these bounds fails the
consumer instead of silently expanding resources. Each partition retries its
failed record without fetching or committing a later offset. Normal revocation
is cancellation rather than a persistent readiness error. Ownership cleanup has
a separate five-second deadline and never authorizes event processing.

The legacy message-consumer API remains for existing adapters. Native PostgreSQL
wiring refuses it. Changing the topic partition count still requires a controlled
producer/consumer transition: a fixed chain key alone does not guarantee the
same physical partition after repartitioning.
