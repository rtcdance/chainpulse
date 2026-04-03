# M1-1a: 修复单体基础数据链路（严格按 ARCHITECTURE_v1.md 蓝图）

> 这是 M1-1 的第一阶段。完成后：Puller 能拉到真实数据 → EventBus → Indexer → DB → Query → API。
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M1-1a - 修复单体基础数据链路

### 背景
ChainPulse 是一个区块链事件索引系统。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`

### 蓝图定义的单体模式数据流

```
HTTPSJSONRPCPuller → EventBus → ChainIndexer → SharedRuntime → EventSink → MockDB
                                                          ↓
                                                    QueryService → GraphQL API
```

### 当前状态：9 个断裂点

#### 断裂 1: EventBus 从未被创建
- 文件: `cmd/monolithic/chainpulse/main.go:196`
- 代码: `nil, // eventBus`
- 蓝图要求: `core.EventBus`（内存 chan）
- 修复: 创建 `core.NewEventBus(logger)`，传给所有 Puller 和 ChainIndexer

#### 断裂 2: QueryService 和 IndexingStorage 用不同的 DB
- IndexingStorage 用 `MonolithicMemoryDatabase`，QueryService 用 MongoDB/PostgreSQL
- 蓝图要求: 单体模式统一用 **MockDB**
- 修复: QueryService 和 IndexingStorage 都使用同一个 MockDB 实例

#### 断裂 3: Puller 从未被实例化
- main.go 中没有 Puller 代码
- 蓝图要求: 单体模式用 **本地节点**（HTTPSJSONRPCPuller 连接公共 RPC），单 goroutine per chain
- 修复: 为每条链实例化 `pkg/plugins/pullers/https_jsonrpc_puller.go`

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

### 完整数据流（修复后）

```
for each chain (ethereum, polygon, ...):
  HTTPSJSONRPCPuller(chainID, nodeURL, eventBus)
    → Start()
    → goroutine 循环:
      1. fromBlock = lastProcessedBlock + 1, toBlock = fromBlock + 10
      2. events := puller.PullEvents(ctx, fromBlock, toBlock)
      3. eventBus.Publish("blockchain-events", events)
       4. envelopes := toEventEnvelope(event)  // 使用 chain_indexer.go 已有的 toEventEnvelope 函数
      5. chainIndexer.ProcessBatch(ctx, chainID, envelopes)
         → SharedRuntime.ProcessBatch(ctx, chainID, envelopes)
         → EventSink.Persist(ctx, envelopes)
         → LegacyRuntimeSink → MockDB
      6. lastProcessedBlock = toBlock
      7. sleep 10s

  QueryService(MockDB)
    → MockDB.QueryEvents()
    → GraphQL API 返回
```

### 目标
修复上述 9 个断裂，使 `make run-monolithic` 能从真实以太坊 RPC 拉取事件，写入 MockDB，通过 GraphQL API 查询到。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic
- [ ] 启动后 60 秒内，`curl http://localhost:8080/graphql` 能查询到真实链上事件

#### 蓝图一致性
- [ ] EventBus 作为 Puller 和 Indexer 之间的通信机制
- [ ] 每条链有独立的 Puller + ChainIndexer + ReorgHandler
- [ ] QueryService 和 IndexingStorage 使用同一个 MockDB 实例
- [ ] Composition Root (main.go) 只负责 wiring，无业务逻辑
- [ ] ReorgHandler 初始化并接入拉取循环
- [ ] logToEvent 填充 BlockHash 和 ChainID

### 可用的公共以太坊 RPC 端点

```
https://eth.llamarpc.com
https://rpc.ankr.com/eth
https://ethereum-rpc.publicnode.com
https://1rpc.io/eth
```

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图**
- `cmd/monolithic/chainpulse/main.go` — Composition Root
- `pkg/core/eventbus.go` — EventBus
- `pkg/core/config.go` — Config
- `pkg/core/blockchain_models.go` — BlockchainEvent
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — HTTPSJSONRPCPuller（需修复 logToEvent）
- `pkg/plugins/pullers/data_puller.go` — BaseDataPullerPlugin
- `pkg/plugins/database/mock_db.go` — MockDB
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer
- `pkg/services/indexing/legacy_runtime_sink.go` — LegacyRuntimeSink
- `pkg/application/indexing/runtime.go` — EventEnvelope、EventSink、SharedRuntime
- `pkg/application/bootstrap/runtime_wiring.go` — QueryService wiring
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring

### 修复步骤

**Step 1: 修复 logToEvent 的 BlockHash 和 ChainID**
```
文件: pkg/plugins/pullers/https_jsonrpc_puller.go
1. BlockHash: common.HexToHash(log.BlockHash)
2. ChainID: 从 Puller 配置读取，不是硬编码 "1"
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
  nodeURL := nodeURLs[i % len(nodeURLs)]

  // 创建 per-chain config: 复制全局 config，替换 BlockchainNodeURL 和 StartBlock
  pullerConfig := config  // 复制
  pullerConfig.BlockchainNodeURL = nodeURL
  pullerConfig.StartBlock = 0  // 从最新块开始

  puller := pullers.NewHTTPSJSONRPCPuller(pullerConfig, logger, metrics, eventBus)
  puller.Start()

  // ReorgHandler: Ethereum=12, Polygon=128, BSC=15
  var reorgThreshold uint64
  switch chainID {
  case "ethereum": reorgThreshold = 12
  case "polygon":  reorgThreshold = 128
  case "bsc":      reorgThreshold = 15
  default:         reorgThreshold = 12
  }
  maxRollback := reorgThreshold * 10  // 最大回滚深度 = reorgThreshold 的 10 倍

  reorgHandler := reorg.NewReorgHandler(
    indexingDatabase, logger,
    reorgThreshold,
    maxRollback,
  )

  chainIndexer := indexing.NewDefaultChainIndexer(
    chainID, indexingDatabase, indexingCache, logger, nil,
  )
  chainIndexer.SetSharedRuntime(sharedIndexingRuntime, metrics)
```

**Step 4: 统一 DB 来源**
```
让 QueryService 使用 indexingDatabase（MockDB）。
最简单方案: 在 main.go 中直接创建基于 indexingDatabase 的 QueryService，
绕过 BuildRuntimeWiring 的 MongoDB/PG 初始化逻辑。
```

**Step 5: 启动多链拉取循环**
```
文件: cmd/monolithic/chainpulse/main.go
for each chain:
  go func(chainID, puller, chainIndexer, reorgHandler) {
    lastBlock := uint64(0)
    lastBlockHash := ""
    for {
      chainHead, _ := puller.GetLatestBlock(ctx)
      fromBlock := lastBlock + 1
      toBlock := min(fromBlock + 10, chainHead)

      events, err := puller.PullEvents(ctx, fromBlock, toBlock)
      if err != nil || len(events) == 0 {
        time.Sleep(10 * time.Second)
        continue
      }

      // Reorg 检测: 对比 block hash
      // DetectReorg 签名: (ctx, currentBlock uint64, newBlockHash common.Hash) (bool, uint64, error)
      if lastBlockHash != "" && events[0].BlockHash.Hex() != lastBlockHash {
        detected, depth, err := reorgHandler.DetectReorg(ctx, fromBlock, events[0].BlockHash)
        if err == nil && detected {
          reorgHandler.HandleReorg(ctx, fromBlock)  // 参数: (ctx, reorgBlock uint64)
          reorgHandler.RollbackEvents(ctx, fromBlock)  // 参数: (ctx, fromBlock uint64) (int64, error)
        }
      }

      eventBus.Publish("blockchain-events", events)

      // 转换为 EventEnvelope: chain_indexer.go 已有 toEventEnvelope 函数
      // 签名: func toEventEnvelope(event *core.BlockchainEvent) appindexing.EventEnvelope
      envelopes := make([]appindexing.EventEnvelope, 0, len(events))
      for i := range events {
        envelopes = append(envelopes, toEventEnvelope(&events[i]))
      }
      chainIndexer.ProcessBatch(ctx, chainID, envelopes)

      lastBlock = toBlock
      lastBlockHash = events[len(events)-1].BlockHash.Hex()
      time.Sleep(10 * time.Second)
    }
  }(chainID, puller, chainIndexer, reorgHandler)
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
- **必须与 ARCHITECTURE_v1.md 蓝图一致**
- **本阶段只做基础数据链路，不做容错（retry/backpressure/checkpoint/ABI/idempotency）、可观测性、API 限流**

### 验证步骤
```bash
make build
make test-unit
make vet
make run-monolithic &
sleep 60
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(limit: 5) { id chainId blockNumber transactionHash blockHash } }"}'
curl -s http://localhost:8080/health
```
