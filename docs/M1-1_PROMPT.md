# M1-1: 修复单体端到端链路（严格按 ARCHITECTURE_v1.md 蓝图执行）

> 这是可以直接发给 GPT 的完整 prompt。

---

## 任务: M1-1 - 按蓝图修复单体端到端链路

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**这是唯一权威来源，所有实现必须与其一致**）
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
                              ↓
                    ┌─────────────────┐
                    │ EVM Node / Mock │
                    └─────────────────┘
```

### 蓝图对单体模式的具体要求（必须严格遵守）

#### Puller
| 维度 | 蓝图要求 |
|---|---|
| MQ | `core.EventBus`（内存 chan） |
| RPC | MockPuller / 本地节点 |
| 扩缩容 | 单 goroutine per chain |
| 容错 | RPC 故障切换 + 指数退避重试 + checkpoint 落盘 + 背压控制 |

#### Indexer
| 维度 | 蓝图要求 |
|---|---|
| DB | MockDB / SQLite |
| Cache | InMemoryCache |
| 事件来源 | EventBus |
| 容错 | 幂等写入 + 批量写入 + ABI 平滑升级 |

#### Query
| 维度 | 蓝图要求 |
|---|---|
| DB | MockDB |
| Cache | InMemoryCache |
| 熔断器 | 内存实现 |
| 容错 | 熔断 + 缓存击穿防护 + 降级 + 一致性检查 |

### 当前状态：12 个断裂点

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
- 修复: 为每条链实例化 Puller（使用真实 RPC 节点替代 MockPuller，因为需要验证真实数据流）

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

#### 断裂 12: 没有背压控制
- 蓝图要求: MQ 满时 Puller 暂停拉取，防止内存溢出
- 当前: 无背压控制
- 修复: 检查 EventBus 订阅者队列深度，满时暂停拉取

### 完整数据流（严格按蓝图）

```
for each chain (ethereum, polygon, ...):
  HTTPSJSONRPCPuller(chainID, nodeURL, eventBus)  // 替代 MockPuller，使用真实 RPC
    → Start()
    → goroutine 循环:
      1. 指数退避重试包装 PullEvents（蓝图要求: max 3 retries, 1s-30s backoff）
      2. 背压控制: 检查 EventBus 队列深度，满时暂停（蓝图要求）
      3. Checkpoint: 从 BlockHeightTracker 加载 lastIndexedBlock（蓝图要求: 落盘）
      4. fromBlock = checkpoint + 1, toBlock = fromBlock + batchSize
      5. events := puller.PullEvents(ctx, fromBlock, toBlock)
      6. Reorg 检测: 对比 block hash，变化时调用 ReorgHandler
      7. eventBus.Publish("blockchain-events", events)  // 蓝图: 单体用 EventBus
      8. ChainIndexer.ProcessBatch(ctx, chainID, envelopes)
         → SharedRuntime.ProcessBatch(ctx, chainID, envelopes)
         → EventSink.Persist(ctx, envelopes)
         → LegacyRuntimeSink → MockDB/MonolithicMemoryDatabase  // 蓝图: MockDB/SQLite
      9. BlockHeightTracker.UpdateBlockHeight(toBlock)  // 蓝图: checkpoint 落盘
      10. sleep pollInterval

  QueryService(MockDB, InMemoryCache)  // 蓝图: MockDB + InMemoryCache
    → 从同一个 MockDB 查询数据
    → GraphQL API 返回
```

### 目标
严格按 ARCHITECTURE_v1.md 蓝图修复单体模式，使 `make run-monolithic` 能：
1. 从真实以太坊 RPC 拉取事件（MockPuller 的替代，因为需要验证真实数据流）
2. 多链并行索引（单 goroutine per chain）
3. Reorg 检测与处理
4. 容错: 指数退避重试 + checkpoint + 背压控制
5. 查询: 从同一个 DB 返回索引数据

### 成功标准

#### 基础（必须全部通过）
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性
- [ ] EventBus 作为 Puller 和 Indexer 之间的通信机制（内存 chan）
- [ ] 每条链有独立的 Puller + ChainIndexer + ReorgHandler（单 goroutine per chain）
- [ ] QueryService 和 IndexingStorage 使用同一个 DB 实例（MockDB）
- [ ] InMemoryCache 用于查询缓存
- [ ] Composition Root (main.go) 只负责 wiring，无业务逻辑

#### 容错（蓝图要求）
- [ ] 指数退避重试: RPC 错误时重试，最大 3 次，初始 1s，上限 30s
- [ ] Checkpoint: 拉取进度可追踪，重启后从断点续拉（M1 至少内存 checkpoint）
- [ ] 背压控制: EventBus 队列满时 Puller 暂停拉取
- [ ] Reorg 检测: block hash 变化时调用 ReorgHandler

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
- `cmd/monolithic/chainpulse/main.go` — Composition Root，只负责 wiring
- `pkg/core/eventbus.go` — EventBus 实现（内存 chan）
- `pkg/core/config.go` — Config 结构体
- `pkg/core/blockchain_models.go` — BlockchainEvent 结构体
- `pkg/core/plugin.go` — DataPullerPlugin / DatabasePlugin / CachePlugin 接口
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — HTTPSJSONRPCPuller（替代 MockPuller，使用真实 RPC）
- `pkg/plugins/pullers/data_puller.go` — BaseDataPullerPlugin（含 eventBus 和 PublishEvents）
- `pkg/plugins/database/mock_db.go` — MockDB（蓝图要求的单体 DB）
- `pkg/plugins/cache/` — InMemoryCache（蓝图要求的单体 Cache）
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler
- `pkg/services/resilience/retry_logic.go` — 指数退避重试
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer
- `pkg/application/indexing/runtime.go` — EventEnvelope、EventSink、SharedRuntime
- `pkg/services/indexing/legacy_runtime_sink.go` — LegacyRuntimeSink
- `pkg/infrastructure/data/block_height_tracker.go` — Checkpoint 追踪
- `pkg/application/bootstrap/runtime_wiring.go` — QueryService wiring
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring

### 修复步骤（按顺序）

**Step 1: 修复 logToEvent 的 BlockHash 和 ChainID**
```
文件: pkg/plugins/pullers/https_jsonrpc_puller.go
1. BlockHash: common.HexToHash(log.BlockHash)  // eth_getLogs 返回了但没赋值
2. ChainID: 从 Puller 配置读取，不是硬编码 "1"
```

**Step 2: 创建 EventBus**
```
文件: cmd/monolithic/chainpulse/main.go
1. eventBus := core.NewEventBus(logger)  // 蓝图: in-process EventBus
2. 传给所有 Puller 和 ChainIndexer
```

**Step 3: 为每条链实例化组件**
```
文件: cmd/monolithic/chainpulse/main.go
for i, chainID := range chains:
  nodeURL := nodeURLs[i % len(nodeURLs)]  // 循环使用

  // Puller（蓝图: 单 goroutine per chain，MockPuller / 本地节点）
  pullerConfig := createPerChainConfig(config, nodeURL, chainID)
  puller := pullers.NewHTTPSJSONRPCPuller(pullerConfig, logger, metrics, eventBus)
  puller.Start()

  // ReorgHandler（蓝图: Application Services 的一部分）
  reorgHandler := reorg.NewReorgHandler(
    indexingDatabase,  // 蓝图: MockDB / SQLite
    logger,
    getReorgThreshold(chainID),  // Ethereum=12, Polygon=128
    getMaxRollback(chainID),
  )

  // ChainIndexer
  chainIndexer := indexing.NewDefaultChainIndexer(
    chainID,
    indexingDatabase,  // 蓝图: MockDB
    indexingCache,     // 蓝图: InMemoryCache
    logger,
    nil,
  )
  chainIndexer.SetSharedRuntime(sharedIndexingRuntime, metrics)
```

**Step 4: 统一 DB 来源（蓝图: MockDB）**
```
让 QueryService 使用 indexingDatabase（MockDB/MonolithicMemoryDatabase）。
最简单方案: 在 main.go 中直接创建基于 indexingDatabase 的 QueryService，
绕过 BuildRuntimeWiring 的 MongoDB/PG 初始化逻辑。
```

**Step 5: 启动多链拉取循环（含蓝图要求的容错）**
```
文件: cmd/monolithic/chainpulse/main.go
for each chain:
  go func(chainID, puller, chainIndexer, reorgHandler) {
    // 蓝图: checkpoint 加载
    checkpoint := blockHeightTracker.GetBlockHeight(ctx, chainID)

    // 蓝图: 指数退避重试
    retry := resilience.NewRetryLogic(3, 1*time.Second, 30*time.Second, logger)

    for {
      // 蓝图: 背压控制
      if eventBus.IsBackpressured("blockchain-events") {
        sleep(5s)
        continue
      }

      // 蓝图: 拉取进度
      fromBlock := checkpoint + 1
      toBlock := min(fromBlock + 10, chainHead)

      // 蓝图: 指数退避重试包装
      events, err := retry.ExecuteWithRetry(func() ([]core.BlockchainEvent, error) {
        return puller.PullEvents(ctx, fromBlock, toBlock)
      })

      // 蓝图: Reorg 检测
      if len(events) > 0 && events[0].BlockHash != lastBlockHash[chainID] {
        reorgHandler.DetectReorg(ctx, chainID, fromBlock, ...)
        reorgHandler.HandleReorg(ctx, ...)
        reorgHandler.RollbackEvents(ctx, fromBlock)
      }

      // 蓝图: EventBus 发布
      eventBus.Publish("blockchain-events", events)

      // 蓝图: Indexer 消费
      envelopes := toEventEnvelopes(events)
      chainIndexer.ProcessBatch(ctx, chainID, envelopes)

      // 蓝图: checkpoint 更新
      checkpoint = toBlock
      blockHeightTracker.UpdateBlockHeight(ctx, chainID, toBlock)
      lastBlockHash[chainID] = events[len(events)-1].BlockHash

      sleep(10s)
    }
  }
```

**Step 6: 确认 Sink wiring**
```
确认 BuildMonolithicIndexingRuntime 的 EventSink 指向 MockDB/MonolithicMemoryDatabase。
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
- **必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因**

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
