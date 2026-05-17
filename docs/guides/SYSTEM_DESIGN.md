# ChainPulse System Design

**适用于**: Web3+Go 系统设计面试准备 | **最后更新**: 2026-05-13

---

## 1. 系统概述

ChainPulse 是一个**区块链事件索引系统**：从区块链节点拉取原始事件日志，解码为结构化数据，并通过多协议 API（REST/gRPC/GraphQL/WebSocket）对外提供查询。

### 核心约束

| 维度 | 要求 |
|------|------|
| 数据源 | EVM 兼容链（Ethereum, Polygon, BSC, Arbitrum 等）|
| 延迟 | 新区块 → 可查询 < 30s（P99）|
| 吞吐 | 单链 ~1000 事件/秒 |
| 一致性 | 最终一致性（reorg 后自动修正）|
| 持久性 | 事件数据不丢失（MongoDB + PostgreSQL）|

### 面试关键词

```
索引延迟、重组成分、最终性模型、幂等写入、背压控制、优雅关闭、断路器、RED 可观测性
```

---

## 2. 端到端数据流

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────────┐
│              │     │                  │     │                  │
│  区块链节点    │────▶  Puller 层        │────▶  Processor 层     │
│  (Anvil/RPC)  │     │  ethclient.Dial  │     │  ChainedDecoder  │
│              │     │  FilterLogs      │     │  Idempotency     │
└──────────────┘     └──────────────────┘     └────────┬─────────┘
                                                        │
                                                        ▼
┌──────────────┐     ┌──────────────────┐     ┌──────────────────┐
│              │     │                  │     │                  │
│  API 层      │◀────│  Query 层        │◀────│  Storage 层      │
│  REST/gRPC   │     │  EventRetrieval  │     │  MongoDB         │
│  GraphQL/WS  │     │  Cache           │     │  PostgreSQL      │
│              │     │  CircuitBreaker  │     │  In-Memory       │
└──────────────┘     └──────────────────┘     └──────────────────┘
```

### Pull 阶段详解

```
每个 Poll 周期 (5s):
  1. eth_blockNumber → 获取最新块号
  2. 计算未索引范围 (lastCheckpoint → latestBlock)
  3. 分批 eth_getLogs (每批 1000 块)
  4. log → BlockchainEvent 转换 (ABI 解码)
  5. 事件注入 Shared Runtime (shadow mode)
  6. checkpoint 更新

RED 指标:
  chainpulse_rpc_calls_total      {method, chain_id, status}
  chainpulse_rpc_errors_total     {method, chain_id, error_code}
  chainpulse_rpc_duration_seconds {method, chain_id}
```

---

## 3. 关键设计决策

### 3.1 为什么选 Rollback + Reindex 而不是 Versioned Events？

| 方案 | 优势 | 劣势 |
|------|------|------|
| Rollback + Reindex ✅ | 存储高效，实现简单（2 操作） | 旧数据丢失，重索引期间有窗口 |
| Versioned Events | 审计完整 | 双倍存储，读路径需合并 |
| Tombstone Marking | 审计完整 | 每次查询需过滤 `removed: true` |

**决策依据**: 对索引器来说，事件是"推"给下游的——如果旧数据被 reorg，下游应该收到修正后的事件。保留 tombstone 对查询层无意义。

### 3.2 为什么用 EventBus（进程内）而不是 Kafka？

```
EventBus (进程内)        Kafka (跨进程)
┌─────────────────┐    ┌─────────────────┐
│ Publish → 16     │    │ Produce →       │
│ worker pool      │    │ Partition →     │
│ sync dispatch    │    │ Consumer Group  │
│ no persistence   │    │ persistent      │
└─────────────────┘    └─────────────────┘
```

**单体模式**: EventBus 轻量、有序、测试友好。跨进程通信仍用 Kafka（微服务模式时）。

### 3.3 为什么用 Shadow Mode 而不是直接写入？

```
Shadow Mode:
  Pull → SharedRuntime(计数) → LegacyIndexer(写入)
                            ↓
                      指标暴露 (shadow_owned_events)
                      不下游消费

目的: 在不影响现有数据管道的情况下验证新 runtime 的正确性
```

---

## 4. 扩展性

### 水平扩展（单链→多链）

```
MultiChainIndexer
  ├── ChainIndexer("ethereum")   → Puller + Processor
  ├── ChainIndexer("polygon")    → Puller + Processor
  ├── ChainIndexer("bsc")        → Puller + Processor
  └── ChainIndexer("solana")     → SolanaPuller + Processor
```

每条链：
- 独立的 DataPullerPlugin 实例
- 独立的 checkpoint 队列
- 独立的最终性策略（Eth2FinalityStrategy / ProbabilisticFinalityStrategy / L2RollupFinalityStrategy）
- 可配置多个 RPC 端点自动故障转移（MultiRPCPuller）

### 吞吐瓶颈

| 瓶颈 | 当前限制 | 缓解方案 |
|------|---------|---------|
| RPC eth_getLogs | 公共 RPC 限制 1000 块/查询 | 分块查询 + FailoverRPCClient |
| ABI 解码 | 单线程 | 可改为 worker pool |
| DB 写入 | 单条 INSERT | 已改为 batch INSERT (100条/批) |
| API 查询 | 无缓存 | Redis 缓存 + Cache-Aside 模式 |

---

## 5. 故障模式与缓解

| 故障 | 影响 | 自动恢复 |
|------|------|---------|
| RPC 节点断开 | Pull 失败 | FailoverRPCClient 自动切换到备用节点 |
| RPC 限流 429 | Pull 延迟 | Token Bucket + 退避 |
| 链重组 | 已索引事件可能无效 | Rollback + Reindex (maxRollback=120) |
| 数据库断开 | 写入失败 | 断路器，暂存到 DLQ，恢复后重放 |
| 上下文取消 | 操作中断 | context.Background 改用 lifecycleCtx |

---

## 6. 与同类方案对比

| 特性 | ChainPulse | The Graph | QuickNode | 自建 (ethereum-etl) |
|------|-----------|-----------|-----------|-------------------|
| 部署模式 | 单体/微服务双模式 | 托管/自建 | 托管 | 自建 |
| 数据存储 | PG + Mongo + Redis | PostgreSQL | 专有 | PG |
| 查询接口 | REST/gRPC/GraphQL/WS | GraphQL | REST/WS | 无 |
| Reorg 处理 | 自动 Rollback+Reindex | 基于 PoI | 有 | 需自建 |
| 最终性策略 | 每条链独立配置 | 统一 | 统一 | 无 |
| 断路器 | 有 | 无（图节点） | 有 | 无 |
| RED 指标 | 11 个标准指标 | 有限 | 有 | 无 |
| 幂等写入 | 有 | 有 | 有 | 需自建 |

---

## 7. 面试话术

### "说说你设计过的 Web3 系统"

```
我参与设计了 ChainPulse——一个区块链事件索引系统，架构特点是：

1. 双模式部署：同一代码库支持单体开发调试和微服务生产部署
2. 插件化数据源：通过 DataPullerPlugin 接口支持 HTTPS/WS/gRPC 多种拉取协议
3. 自动 Reorg 处理：基于 Rollback+Reindex 策略，每条链独立配置最终性窗口
4. 企业级可观测：11 个 RED 指标覆盖所有 RPC 调用，按 error_code 分类告警
5. 迁移友好：Shadow Mode 渐进式接管，不中断现有管道

最大的技术挑战是在不中断服务的情况下引入共享运行时——我们用 Shadow Mode + 双写计数
解决了这个信任问题。现在生产流量同时走旧路径和新路径，通过 shadow_owned_events 指标
验证正确性。
```

### "如何处理区块链重组？"

```
我们的策略是 Rollback + Reindex：

1. 检测：BlockConfirmationTracker 定期比较本地哈希和链上哈希
2. 回滚：删除 reorgBlock 之后的所有事件（maxRollback=120 保护）
3. 通知：通过 EventBus 发布 reorg-detected 事件，下游清理缓存
4. 重新索引：Puller 自动从 reorgBlock 开始重新拉取

ETH PoS 的 reorg 深度很少超过 1 个 slot（12秒），但我们的配置允许每条链独立设置
（Ethereum: 32 safe, 64 finalized; BSC: 15 safe, 30 finalized）
```

### "为什么选 Go 而不是 Rust/TypeScript？"

```
Go 在 Web3 中的定位是中间层/基础设施层：

优势：
- goroutine 天然适配每条链独立轮询的并发模型
- 编译快、部署简单（单二进制）
- go-ethereum 是事实标准的 Ethereum 客户端库
- 静态类型 + 接口抽象 + 编译时安全检查

对比 Rust：Rust 性能更好但开发效率低，适合执行层/客户端
对比 TS：TS 生态丰富但运行时开销大，适合 dApp/前端层
Go 处于中间的"服务层"——索引器、查询服务、API 网关都适合 Go
```

---

## 8. 推荐的深入阅读

| 主题 | 代码位置 |
|------|---------|
| Puller 实现 | `pkg/plugins/pullers/https_jsonrpc_puller.go` |
| Reorg 处理 | `pkg/services/reorg/reorg_handler.go` |
| Event Decoder | `pkg/core/chained_decoder.go` |
| RED 指标 | `pkg/observability/red_metrics.go` |
| 错误码体系 | `pkg/core/errors.go` |
| API 错误映射 | `pkg/plugins/api/errors.go` |
| 最终性策略 | `pkg/services/indexing/finality_adapter.go` |
| 最终性检查 | `pkg/services/finality/finality_checker.go` |
| RPC 故障转移 | `pkg/infrastructure/rpc/failover_client.go` |
| 多链拉取 | `pkg/plugins/pullers/multi_chain_puller.go` |
