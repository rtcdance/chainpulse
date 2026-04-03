# M1-1: 修复单体端到端链路（严格按 ARCHITECTURE_v1.md 蓝图执行）

> 这是可以直接发给 GPT 的完整 prompt。
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M1-1 - 按蓝图修复单体端到端链路

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`

### 蓝图定义的单体模式架构

```
┌─────────────────────────────────────────────────────────────────┐
│ cmd/monolithic/chainpulse · 单进程                              │
├─────────────────────────────────────────────────────────────────┤
│   Platform Layer (pkg/observability, core/logger)               │
│     ↓ injects                                                   │
│   Shared Core (pkg/core) · PURE DOMAIN                          │
│     · interfaces: DataPullerPlugin, DatabasePlugin, etc         │
│     · models: BlockchainEvent, Block, ReorgStats                │
│     · in-process EventBus                                       │
│     ↑ implements                                                │
│   In-Process Adapters (pkg/plugins)                             │
│     · MemoryMQ / MockMQ                                         │
│     · InMemoryCache                                             │
│     · SQLiteDB / MockDB                                         │
│     · GraphQL API Server                                        │
│     ↑ uses                                                      │
│   Application Services (pkg/services)                           │
│     · indexing.MultiChainIndexer                                │
│     · reorg.ReorgHandler                                        │
│     · query.QueryService                                        │
│     · decoder.EventDecoder                                      │
│     ↑ wires                                                     │
│   Composition Root (main.go) · NO business logic                │
└─────────────────────────────────────────────────────────────────┘
```

### 蓝图对单体模式的具体要求（必须严格遵守）

#### Puller（§3.1）
| 维度 | 蓝图要求 |
|---|---|
| 职责 | 从 EVM/非-EVM 节点拉取原始区块与事件，发布到 MQ（单体为 EventBus） |
| 核心代码 | `pkg/infrastructure/data/data_puller.go`、`block_height_tracker.go` |
| MQ | `core.EventBus`（内存 chan） |
| RPC | MockPuller / 本地节点 |
| 扩缩容 | 单 goroutine per chain |
| 容错 | RPC 故障切换 + 指数退避重试 + checkpoint 落盘 + 背压控制 |

#### Indexer（§3.2）
| 维度 | 蓝图要求 |
|---|---|
| 职责 | **消费原始事件，ABI 解码，幂等持久化到数据库，维护缓存** |
| 核心代码 | `pkg/services/indexing/`、`pkg/services/decoder/`、`pkg/integrations/` |
| DB | MockDB / SQLite |
| Cache | InMemoryCache |
| 事件来源 | EventBus |
| 容错 | **幂等写入** + 批量写入 + **ABI 平滑升级** |

#### Query（§3.3）
| 维度 | 蓝图要求 |
|---|---|
| 职责 | 对外提供链上数据查询，支持**缓存、熔断、降级、一致性验证** |
| 核心代码 | `query_service.go`、`circuit_breaker.go`、`cache_service.go`、`consistency_checker.go` |
| DB | MockDB |
| Cache | InMemoryCache |
| 熔断器 | 内存实现 |
| 容错 | **熔断** + **缓存击穿防护** + **降级** + **一致性检查** |

#### Reorg Handler（§3.5）
| 维度 | 蓝图要求 |
|---|---|
| 职责 | 检测链重组，回滚受影响数据，触发下游重索引 |
| 核心代码 | `pkg/services/reorg/reorg_handler.go` |
| 容错 | **最终性确认**（reorgThreshold 按链配置）+ **原子回滚** + **最大回滚深度** |

### 当前状态：15 个断裂点

#### 断裂 1: EventBus 从未被创建
- 文件: `cmd/monolithic/chainpulse/main.go:196`
- 代码: `nil, // eventBus`
- 蓝图要求: `core.EventBus`（内存 chan），Puller 和 Indexer 都通过它通信
- 修复: 创建 `core.NewEventBus(logger)`，传给所有 Puller 和 ChainIndexer

#### 断裂 2: QueryService 和 IndexingStorage 用不同的 DB
- IndexingStorage 用 `MonolithicMemoryDatabase`，QueryService 用 MongoDB/PostgreSQL
- 蓝图要求: 单体模式统一用 **MockDB**
- 修复: QueryService 和 IndexingStorage 都使用同一个 MockDB 实例

#### 断裂 3: Puller 从未被实例化
- main.go 中没有 Puller 代码
- 蓝图要求: 单体模式用 **MockPuller / 本地节点**，单 goroutine per chain
- 修复: 为每条链实例化 Puller（使用 `pkg/plugins/pullers/https_jsonrpc_puller.go` 作为本地节点实现）

#### 断裂 4: Puller 的 EventBus 是 nil
- `BaseDataPullerPlugin` 有 `eventBus` 和 `PublishEvents()`，但传的是 nil
- 蓝图要求: Puller 通过 EventBus 发布事件
- 修复: 把 EventBus 传给每个 Puller

#### 断裂 5: 没有 Puller → Indexer 的调用链
- `HTTPSJSONRPCPuller.Start()` 只设置 running 状态，没有内部拉取循环
- `SharedRuntime.ProcessBatch()` 存在但没人调用
- 蓝图要求: Puller 拉取 → EventBus 发布 → Indexer 消费
- 修复: 在 main.go 中为每条链启动 goroutine 循环

#### 断裂 6: ReorgHandler 从未被调用
- `pkg/services/reorg/reorg_handler.go` 有完整实现但没人调用
- 蓝图要求: `reorg.ReorgHandler` 是 Application Services 的一部分
- 修复: 为每条链初始化 ReorgHandler，接入拉取循环

#### 断裂 7: logToEvent 没有填充 BlockHash 和 ChainID
- `logToEvent()` 中 `BlockHash` 未赋值，`ChainID` 硬编码为 `"1"`
- 蓝图要求: `BlockchainEvent` 包含完整的 block_hash 和 chain_id
- 修复: 添加 `BlockHash: common.HexToHash(log.BlockHash)`，ChainID 从配置读取

#### 断裂 8: 多链配置未解析为独立 Puller/Indexer/ReorgHandler
- main.go 有 `parseChains(config.Chains)` 但没有为每条链创建独立组件
- 蓝图要求: 单 goroutine per chain，每条链独立实例
- 修复: 遍历 chains，为每条链创建独立 Puller + ChainIndexer + ReorgHandler

#### 断裂 9: SharedRuntime 的 Sink wiring 需要确认
- 需要确认 EventSink 是否正确指向 MockDB/MonolithicMemoryDatabase

#### 断裂 10: Checkpoint 未持久化
- 蓝图要求: `block_height_tracker.go` checkpoint 落盘，重启后从断点续拉
- 当前: 无 checkpoint 机制
- 修复: 使用 `BlockHeightTracker`，M1 阶段至少实现内存 checkpoint

#### 断裂 11: 没有指数退避重试
- 蓝图要求: `pkg/services/resilience/retry_logic.go`，最大重试 3 次，初始间隔 1s，上限 30s
- 当前: RPC 错误直接返回，没有重试
- 修复: 在拉取循环中集成 `RetryLogic`

#### 断裂 12: ABI 解码未接入 Indexer
- 蓝图要求: Indexer 职责包含 **ABI 解码**（`decoder.EventDecoder`）
- 当前: 事件以原始 bytes 存储，没有解码
- 修复: 在 ProcessBatch 中接入 `decoder.EventDecoder`，解码后存储

#### 断裂 13: 幂等写入缺失
- 蓝图要求: Indexer 容错包含 **幂等写入**，基于 `(chain_id, tx_hash, log_index)` 唯一键去重
- 当前: 重复拉取会导致重复数据
- 修复: 在 EventSink.Persist 中接入 `processor.IdempotencyService`

#### 断裂 14: Query 容错全缺
- 蓝图要求: Query 支持 **熔断**（`circuit_breaker.go`）+ **缓存**（`cache_service.go`）+ **降级** + **一致性检查**（`consistency_checker.go`）
- 当前: QueryService 无容错能力
- 修复: 在 QueryService wiring 中接入 circuit_breaker、cache_service、consistency_checker

#### 断裂 15: 背压控制缺失
- 蓝图要求: MQ 满时 Puller 暂停拉取，防止内存溢出（§3.1）
- 当前: 无背压控制
- 修复: EventBus 添加 `IsBackpressured()` 方法，检查订阅者 chan 缓冲区使用率，满时 Puller 暂停

#### 断裂 16: RPC 故障切换缺失
- 蓝图要求: §3.1 — `pkg/infrastructure/blockchain/blockchain_cluster.go` 维护节点池，失败自动切到备用节点
- 当前: 每条链只用单个 RPC 端点，故障时 Puller 停止工作
- 修复: 为每条链配置 2+ 个 RPC 端点，失败时自动切换到备用节点

#### 断裂 17: Checkpoint 未落盘
- 蓝图要求: §3.1 — `block_height_tracker.go` checkpoint **落盘**，重启后从断点续拉
- 当前: 无 checkpoint 机制
- 修复: 使用 `BlockHeightTracker`，checkpoint 写入 JSON 文件（`.chainpulse/checkpoints/{chainID}.json`），启动时加载

#### 断裂 18: ABI 平滑升级未接入
- 蓝图要求: §3.2 — `contract_manager.go` 多版本 ABI 并存，按 block 范围路由解码逻辑
- 当前: EventDecoder 没有接入 ContractManager，不支持多版本 ABI
- 修复: 在 ProcessBatch 中使用 `ContractManager` 管理多版本 ABI，按 block 范围选择正确的 ABI 解码

#### 断裂 19: Query 降级策略缺失
- 蓝图要求: §3.3 — DB 不可用时返回缓存数据（带 `X-Cache-Stale` 头），缓存也不可用时返回预设默认值
- 当前: QueryService 无降级逻辑
- 修复: 在 QueryService 中接入 `degradation_handler.go`，DB 失败时降级到缓存，缓存失败时返回默认值

#### 断裂 20: Reorg 通知缺失
- 蓝图要求: §3.5 — `RollbackEvents()` 成功后发 `reorg_events` 通知 Indexer 重索引
- 当前: RollbackEvents 后没有通知机制
- 修复: RollbackEvents 成功后通过 EventBus 发布 `reorg_events` 事件，触发 Indexer 重索引

#### 断裂 21: API Gateway 限流/认证缺失
- 蓝图要求: §3.4 — 单体模式限流: 内存令牌桶，按 API Key 限 1000 req/min，按 IP 限 100 req/min；认证: mock
- 当前: main.go 没有接入限流和认证中间件
- 修复: 在 API Gateway wiring 中接入 `rate_limiter.go`（内存令牌桶）和 `auth_middleware.go`（mock 认证）

#### 断裂 22: WebSocket 连接池上限缺失
- 蓝图要求: §3.4 — `websocket_subscription.go` 维护 WebSocket 连接池，单 Pod 上限 10000
- 当前: 需要确认连接池有 maxConns 限制
- 修复: 确认或添加 WebSocket 连接池上限配置

#### 断裂 23: 统一标签注入缺失
- 蓝图要求: §5 — 所有指标、日志、trace 必须携带 `chain_id`、`service`、`operation`、`block_height` 四个标签
- 当前: 指标和日志没有统一携带这些标签
- 修复: 在 metrics/log 调用中统一注入这四个标签

#### 断裂 24: Puller 健康端点缺失
- 蓝图要求: §3.1 — `GET /health/puller` 返回各链拉取状态
- 当前: 没有 Puller 健康端点
- 修复: 注册 `/health/puller` 端点，返回各链的 lastIndexedBlock、blockLag、rpcErrors 等状态

#### 断裂 25: 缓存击穿防护缺失
- 蓝图要求: §3.3 — `cache_warmer.go` 预热热点数据，`cache_middleware.go` 单机锁防并发穿透
- 当前: `cache_warmer.go` 和 `cache_middleware.go` 不存在
- 修复: 创建 `cache_warmer.go`（预热热点数据）和 `cache_middleware.go`（单机锁防并发穿透）

#### 断裂 26: 分布式追踪缺失
- 蓝图要求: §5.3 — OTel Tracer，span 携带 `chain_id`、`from_block`、`to_block`
- 当前: 没有 OTel Tracing
- 修复: 在 Puller/Indexer/Query 的关键操作中注入 OTel span

### 完整数据流（严格按蓝图）

```
for each chain (ethereum, polygon, ...):
  HTTPSJSONRPCPuller(chainID, [nodeURL1, nodeURL2, ...], eventBus)  // 蓝图 §3.1: 多节点池
    → Start()
    → goroutine 循环:
      1. 指数退避重试包装 PullEvents（蓝图 §3.1: max 3 retries, 1s-30s backoff）
      2. RPC 故障切换: 当前节点失败时自动切换到备用节点（蓝图 §3.1: 节点池）
      3. 背压控制: EventBus.IsBackpressured("blockchain-events")，满时暂停（蓝图 §3.1）
      4. Checkpoint: 从 BlockHeightTracker 加载（蓝图 §3.1: 落盘到 JSON 文件）
      5. fromBlock = checkpoint + 1, toBlock = fromBlock + batchSize
      6. events := puller.PullEvents(ctx, fromBlock, toBlock)
      7. Reorg 检测: 对比 block hash（蓝图 §3.5: 最终性确认 + 原子回滚）
         - 变化时 → ReorgHandler.DetectReorg() → HandleReorg() → RollbackEvents()
         - 成功后 → eventBus.Publish("reorg_events", reorgInfo)  // 蓝图 §3.5: 通知 Indexer
      8. eventBus.Publish("blockchain-events", events)  // 蓝图: 单体用 EventBus
      9. ChainIndexer.ProcessBatch(ctx, chainID, envelopes)
         a. ABI 解码: ContractManager.GetABI(block) → EventDecoder.Decode(envelopes)  // 蓝图 §3.2: ABI 平滑升级
         b. 幂等检查: IdempotencyService.Check(envelopes)  // 蓝图 §3.2: 幂等写入
         c. SharedRuntime.ProcessBatch(ctx, chainID, envelopes)
            → EventSink.Persist(ctx, envelopes)
            → LegacyRuntimeSink → MockDB.BatchStoreEvents()  // 蓝图: MockDB + 批量写入
      10. BlockHeightTracker.UpdateBlockHeight(toBlock)  // 蓝图 §3.1: checkpoint 落盘到文件
      11. sleep pollInterval

  QueryService(MockDB, InMemoryCache, CircuitBreaker, ConsistencyChecker, DegradationHandler)  // 蓝图 §3.3
    → 熔断检查: circuit_breaker.Allow()
    → 缓存检查: cache_service.Get()
      - 缓存击穿防护: cache_middleware.GetOrLock()  // 蓝图 §3.3: 单机锁防并发穿透
      - 缓存预热: cache_warmer.Warm()  // 蓝图 §3.3: 预热热点数据
    → DB 查询: MockDB.QueryEvents()
      - DB 失败时 → 降级到缓存（X-Cache-Stale 头）  // 蓝图 §3.3: 降级
      - 缓存也不可用 → 返回预设默认值
    → 一致性检查: consistency_checker.Verify()
    → GraphQL API 返回

  API Gateway (GraphQL only, RateLimiter, AuthMiddleware, WebSocketPool)  // 蓝图 §3.4
    → 认证: auth_middleware.MockAuth()  // 蓝图 §3.4: mock 认证
    → 限流: rate_limiter.TokenBucket(API Key 1000/min, IP 100/min)  // 蓝图 §3.4: 内存令牌桶
    → WebSocket 连接池: maxConns=10000  // 蓝图 §3.4: 单 Pod 上限
    → GraphQL 路由

  Health Endpoints  // 蓝图 §3.1, §5
    → GET /health/puller → 返回各链拉取状态（lastIndexedBlock, blockLag, rpcErrors）
    → GET /health → 总体健康状态

  Observability (统一标签注入 + 分布式追踪)  // 蓝图 §5
    → 所有指标/日志/trace 携带: chain_id, service, operation, block_height
    → OTel Tracer span: pull_events(chain_id, from_block, to_block), process_batch(chain_id, count), query_events(chain_id)
```

### 目标
严格按 ARCHITECTURE_v1.md 蓝图修复单体模式，使 `make run-monolithic` 能：
1. 从真实以太坊 RPC 拉取事件（使用 `pkg/plugins/pullers/` 中的 HTTPSJSONRPCPuller）
2. 多链并行索引（单 goroutine per chain，蓝图 §3.1）
3. ABI 解码（蓝图 §3.2）
4. 幂等写入（蓝图 §3.2）
5. Reorg 检测与处理（蓝图 §3.5）
6. Query 容错: 熔断 + 缓存 + 降级 + 一致性检查（蓝图 §3.3）
7. 容错: 指数退避重试 + checkpoint + 背压控制（蓝图 §3.1）

### 成功标准

#### 基础（必须全部通过）
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性（必须全部通过）
- [ ] EventBus 作为 Puller 和 Indexer 之间的通信机制（蓝图: 单体用 `core.EventBus`）
- [ ] 每条链有独立的 Puller + ChainIndexer + ReorgHandler（蓝图: 单 goroutine per chain）
- [ ] QueryService 和 IndexingStorage 使用同一个 MockDB 实例（蓝图: 单体用 MockDB）
- [ ] InMemoryCache 用于查询缓存（蓝图: 单体用 InMemoryCache）
- [ ] Composition Root (main.go) 只负责 wiring，无业务逻辑
- [ ] **ABI 解码 + 平滑升级接入 Indexer**（蓝图 §3.2: ABI 解码 + ContractManager 多版本 ABI）
- [ ] **幂等写入接入 Indexer**（蓝图 §3.2: 基于唯一键去重）
- [ ] **批量写入接入 Indexer**（蓝图 §3.2: BatchStoreEvents）
- [ ] **Query 熔断接入**（蓝图 §3.3: circuit_breaker.go）
- [ ] **Query 缓存接入**（蓝图 §3.3: cache_service.go）
- [ ] **Query 降级接入**（蓝图 §3.3: DB 失败→缓存→默认值）
- [ ] **Query 一致性检查接入**（蓝图 §3.3: consistency_checker.go）
- [ ] **ReorgHandler 接入拉取循环**（蓝图 §3.5: 检测链重组，回滚受影响数据）
- [ ] **Reorg 通知接入**（蓝图 §3.5: RollbackEvents 成功后发 reorg_events 通知 Indexer）
- [ ] **指数退避重试接入 Puller**（蓝图 §3.1: max 3 retries, 1s-30s backoff）
- [ ] **Checkpoint 落盘接入 Puller**（蓝图 §3.1: block_height_tracker.go 落盘到文件）
- [ ] **RPC 故障切换接入 Puller**（蓝图 §3.1: 多节点池，失败自动切换）
- [ ] **背压控制接入 Puller**（蓝图 §3.1: MQ 满时暂停拉取）
- [ ] **统一标签注入**（蓝图 §5: 所有指标/日志/trace 携带 chain_id, service, operation, block_height）
- [ ] **API Gateway 限流接入**（蓝图 §3.4: 内存令牌桶，API Key 1000/min，IP 100/min）
- [ ] **API Gateway mock 认证接入**（蓝图 §3.4: mock 认证）
- [ ] **WebSocket 连接池上限**（蓝图 §3.4: 单 Pod 上限 10000）
- [ ] **Puller 健康端点**（蓝图 §3.1: GET /health/puller 返回各链拉取状态）
- [ ] **缓存击穿防护**（蓝图 §3.3: cache_warmer 预热 + cache_middleware 单机锁）
- [ ] **分布式追踪**（蓝图 §5.3: OTel Tracer span 携带 chain_id, from_block, to_block）

#### 真实数据
- [ ] 启动后 60 秒内，GraphQL 查询返回真实链上事件
- [ ] 返回的数据有真实的 tx hash、block hash、contract address
- [ ] 多链数据可同时查询

### 可用的公共以太坊 RPC 端点

```
https://eth.llamarpc.com
https://rpc.ankr.com/eth
https://ethereum-rpc.publicnode.com
https://1rpc.io/eth
```

Polygon 公共 RPC：
```
https://polygon-rpc.com
https://rpc.ankr.com/polygon
https://polygon-bor-rpc.publicnode.com
```

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`，特别是:
1. 新代码只写入正确的层
2. 不要往 `pkg/domain/`、`pkg/application/`、`pkg/adapters/` 添加新功能
3. 不要修改已有依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
4. 不要重构已工作的代码

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，所有实现必须与其一致**
- `cmd/monolithic/chainpulse/main.go` — Composition Root
- `pkg/core/eventbus.go` — EventBus（含 GetSubscriberCount 用于背压）
- `pkg/core/config.go` — Config 结构体
- `pkg/core/blockchain_models.go` — BlockchainEvent 结构体
- `pkg/core/plugin.go` — DataPullerPlugin / DatabasePlugin / CachePlugin 接口
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — HTTPSJSONRPCPuller（需修复 logToEvent）
- `pkg/plugins/pullers/data_puller.go` — BaseDataPullerPlugin（含 eventBus 和 PublishEvents）
- `pkg/plugins/database/mock_db.go` — MockDB（含 BatchStoreEvents）
- `pkg/plugins/cache/` — InMemoryCache
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler（DetectReorg/HandleReorg/RollbackEvents）
- `pkg/services/resilience/retry_logic.go` — 指数退避重试
- `pkg/services/decoder/event_decoder.go` — EventDecoder（ABI 解码）
- `pkg/services/decoder/contract_manager.go` — ContractManager（多版本 ABI）
- `pkg/services/processor/idempotency.go` — IdempotencyService（幂等写入）
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer.ProcessBatch
- `pkg/services/indexing/legacy_runtime_sink.go` — LegacyRuntimeSink
- `pkg/services/query/query_service.go` — QueryService
- `pkg/services/query/circuit_breaker.go` — 熔断器
- `pkg/services/query/cache_service.go` — 缓存服务
- `pkg/services/query/consistency_checker.go` — 一致性检查
- `pkg/services/query/degradation_handler.go` — 降级策略（DB→缓存→默认值）
- `pkg/infrastructure/blockchain/blockchain_cluster.go` — 区块链节点池（RPC 故障切换）
- `pkg/observability/metrics.go` — Prometheus 指标定义（蓝图 §5: 统一标签注入）
- `pkg/plugins/api/rate_limiter.go` — 内存令牌桶限流（蓝图 §3.4）
- `pkg/plugins/api/auth_middleware.go` — mock 认证中间件（蓝图 §3.4）
- `pkg/plugins/api/websocket_subscription.go` — WebSocket 连接池管理（蓝图 §3.4）
- `pkg/observability/tracer.go` — OTel Tracer（蓝图 §5.3: 分布式追踪）
- `pkg/application/indexing/runtime.go` — EventEnvelope、EventSink、SharedRuntime
- `pkg/infrastructure/data/block_height_tracker.go` — Checkpoint 追踪
- `pkg/application/bootstrap/runtime_wiring.go` — QueryService wiring
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring

### 修复步骤（按顺序）

**Step 1: 修复 logToEvent 的 BlockHash 和 ChainID**
```
文件: pkg/plugins/pullers/https_jsonrpc_puller.go
1. BlockHash: common.HexToHash(log.BlockHash)
2. ChainID: 从 Puller 配置读取
```

**Step 2: 创建 EventBus**
```
文件: cmd/monolithic/chainpulse/main.go
1. eventBus := core.NewEventBus(logger)
2. 传给所有 Puller 和 ChainIndexer
```

**Step 3: 为每条链实例化组件**
```
文件: cmd/monolithic/chainpulse/main.go
for i, chainID := range chains:
  // 蓝图 §3.1: 多节点池
  nodeURLs := parseNodeURLs(config.BlockchainNodeURLs, i)  // 每条链 2+ 个节点

  // Puller（蓝图 §3.1: 多节点池）
  pullerConfig := createPerChainConfig(config, nodeURLs[0], chainID)
  puller := pullers.NewHTTPSJSONRPCPuller(pullerConfig, logger, metrics, eventBus)
  puller.SetBackupNodes(nodeURLs[1:])  // 蓝图 §3.1: 节点池，失败自动切换
  puller.Start()

  // ReorgHandler（蓝图 §3.5）
  reorgHandler := reorg.NewReorgHandler(
    indexingDatabase,
    logger,
    getReorgThreshold(chainID),  // Ethereum=12, Polygon=128
    getMaxRollback(chainID),
  )

  // EventDecoder + ContractManager（蓝图 §3.2: ABI 平滑升级）
  contractManager := decoder.NewContractManager(logger)
  // 加载已知合约 ABI（从 pkg/integrations/ 或配置文件）
  contractManager.RegisterABI("ethereum", erc20.ABI, 0, math.MaxUint64)
  eventDecoder := decoder.NewEventDecoder(contractManager, logger)

  // IdempotencyService（蓝图 §3.2: 幂等写入）
  idempotencyService := processor.NewIdempotencyService(indexingDatabase, logger)

  // ChainIndexer
  chainIndexer := indexing.NewDefaultChainIndexer(
    chainID,
    indexingDatabase,
    indexingCache,
    logger,
    nil,
  )
  chainIndexer.SetSharedRuntime(sharedIndexingRuntime, metrics)
```

**Step 4: 统一 DB 来源 + Query 容错（蓝图 §3.3）**
```
文件: cmd/monolithic/chainpulse/main.go
1. QueryService 使用 indexingDatabase（MockDB）
2. 接入 circuit_breaker.go（熔断: 错误率 > 50% 且 > 10/s 时熔断 30s）
3. 接入 cache_service.go（缓存 + 缓存击穿防护）
4. 接入 consistency_checker.go（一致性检查）
5. 接入 degradation_handler.go（降级: DB 失败→缓存→默认值，带 X-Cache-Stale 头）
```

**Step 5: 启动多链拉取循环（含蓝图要求的全部容错）**
```
文件: cmd/monolithic/chainpulse/main.go
for each chain:
  go func(chainID, puller, chainIndexer, reorgHandler, eventDecoder, idempotencyService) {
    // 蓝图 §3.1: checkpoint 从文件加载
    checkpointFile := fmt.Sprintf(".chainpulse/checkpoints/%s.json", chainID)
    checkpoint := loadCheckpointFromFile(checkpointFile)

    // 蓝图 §3.1: 指数退避重试
    retry := resilience.NewRetryLogic(3, 1*time.Second, 30*time.Second, logger)

    for {
      // 蓝图 §3.1: 背压控制
      if eventBus.IsBackpressured("blockchain-events") {
        sleep(5s)
        continue
      }

      // 蓝图 §3.1: 拉取进度
      fromBlock := checkpoint + 1
      toBlock := min(fromBlock + 10, chainHead)

      // 蓝图 §3.1: 指数退避重试 + RPC 故障切换
      events, err := retry.ExecuteWithRetry(func() ([]core.BlockchainEvent, error) {
        return puller.PullEventsWithFailover(ctx, fromBlock, toBlock)  // 蓝图 §3.1: 节点池
      })

      // 蓝图 §3.5: Reorg 检测
      if len(events) > 0 && events[0].BlockHash != lastBlockHash[chainID] {
        reorgHandler.DetectReorg(ctx, chainID, fromBlock, ...)
        reorgHandler.HandleReorg(ctx, ...)
        reorgHandler.RollbackEvents(ctx, fromBlock)
        // 蓝图 §3.5: 成功后发 reorg_events 通知 Indexer
        eventBus.Publish("reorg_events", ReorgInfo{ChainID: chainID, FromBlock: fromBlock})
      }

      // 蓝图: EventBus 发布
      eventBus.Publish("blockchain-events", events)

      // 蓝图 §3.2: ABI 解码 + 平滑升级
      decodedEvents := eventDecoder.DecodeBatch(events)  // ContractManager 按 block 选 ABI

      // 蓝图 §3.2: 幂等检查
      uniqueEvents := idempotencyService.FilterDuplicates(decodedEvents)

      // 蓝图 §3.2: Indexer 消费 + 批量写入
      envelopes := toEventEnvelopes(uniqueEvents)
      chainIndexer.ProcessBatch(ctx, chainID, envelopes)

      // 蓝图 §3.1: checkpoint 落盘到文件
      checkpoint = toBlock
      saveCheckpointToFile(checkpointFile, checkpoint)
      lastBlockHash[chainID] = events[len(events)-1].BlockHash

      // 蓝图 §5: 统一标签注入
      metrics.WithLabels("chain_id", chainID, "service", "puller").Record(...)

      sleep(10s)
    }
  }
```

**Step 6: 确认 Sink wiring**
```
确认 BuildMonolithicIndexingRuntime 的 EventSink 指向 MockDB/MonolithicMemoryDatabase。
```

**Step 7: API Gateway 限流/认证 + 健康端点 + 可观测性**
```
文件: cmd/monolithic/chainpulse/main.go

// 蓝图 §3.4: API Gateway 限流 + mock 认证
rateLimiter := api.NewRateLimiter(api.RateLimitConfig{
    APIKeyRate: 1000,  // 1000 req/min per API Key
    IPRate:     100,   // 100 req/min per IP
})
authMiddleware := api.NewMockAuth()  // 蓝图 §3.4: mock 认证
gateway.SetRateLimiter(rateLimiter)
gateway.SetAuthMiddleware(authMiddleware)

// 蓝图 §3.4: WebSocket 连接池上限
websocketPool := api.NewWebSocketPool(10000)  // 单 Pod 上限 10000
gateway.SetWebSocketPool(websocketPool)

// 蓝图 §3.1: Puller 健康端点
healthHandler.RegisterPullerHealth(pullers)  // GET /health/puller 返回各链状态

// 蓝图 §5: 统一标签注入
metrics.SetDefaultLabels(map[string]string{
    "chain_id": chainID,
    "service":  "puller",
})
logger.SetDefaultFields([]string{"chain_id", "service", "operation", "block_height"})

// 蓝图 §5.3: 分布式追踪
tracer := observability.NewOTelTracer()
ctx, span := tracer.Start(ctx, "pull_events",
    trace.WithAttributes(
        attribute.String("chain_id", chainID),
        attribute.Int64("from_block", int64(fromBlock)),
        attribute.Int64("to_block", int64(toBlock)),
    ))
defer span.End()
```

**Step 8: 创建缓存击穿防护**
```
新建 pkg/services/query/cache_warmer.go:
  - 预热热点数据（最近 N 个块的事件）
  - 定时刷新热点缓存

新建 pkg/services/query/cache_middleware.go:
  - 单机锁防并发穿透（singleflight）
  - 缓存未命中时只允许一个请求查 DB，其他等待
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
- **必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因**
- **蓝图要求的以下功能全部必须实现，不可跳过**:
  - ABI 解码 + 平滑升级（§3.2: EventDecoder + ContractManager 多版本 ABI）
  - 幂等写入（§3.2: 基于唯一键去重）
  - 批量写入（§3.2: BatchStoreEvents）
  - Query 熔断（§3.3: circuit_breaker.go）
  - Query 缓存（§3.3: cache_service.go）
  - Query 降级（§3.3: DB→缓存→默认值）
  - Query 一致性检查（§3.3: consistency_checker.go）
  - Reorg 处理 + 通知（§3.5: DetectReorg + HandleReorg + RollbackEvents + reorg_events）
  - 指数退避重试（§3.1: max 3 retries, 1s-30s backoff）
  - Checkpoint 落盘（§3.1: block_height_tracker.go 写入文件）
  - RPC 故障切换（§3.1: 多节点池）
  - 背压控制（§3.1: MQ 满时暂停）
  - 统一标签注入（§5: chain_id, service, operation, block_height）
  - API Gateway 限流（§3.4: 内存令牌桶，API Key 1000/min, IP 100/min）
  - API Gateway mock 认证（§3.4）
  - WebSocket 连接池上限（§3.4: 10000）
  - Puller 健康端点（§3.1: GET /health/puller）
  - 缓存击穿防护（§3.3: cache_warmer + cache_middleware）
  - 分布式追踪（§5.3: OTel Tracer span）

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 手动验证
CHAINS=ethereum,polygon \
BLOCKCHAIN_NODE_URLS=https://eth.llamarpc.com,https://polygon-rpc.com \
make run-monolithic &
sleep 60
# 验证真实数据
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(limit: 5) { id chainId blockNumber transactionHash blockHash } }"}'
# 验证多链
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(chainId: \"ethereum\", limit: 2) { id blockNumber } }"}'
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(chainId: \"polygon\", limit: 2) { id blockNumber } }"}'
# 验证健康检查
curl -s http://localhost:8080/health
```
