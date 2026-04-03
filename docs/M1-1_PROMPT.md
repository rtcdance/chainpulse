# M1-1: 修复单体端到端链路（多链 + Reorg + 生产可用）

> 这是可以直接发给 GPT 的完整 prompt。

---

## 任务: M1-1 - 修复单体端到端链路

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`

### 架构要求（来自 ARCHITECTURE_v1.md）

M1 必须支持以下能力，缺一不可：

| 能力 | 状态 | 说明 |
|---|---|---|
| **真实链上数据** | ❌ 断裂 | Puller 从未实例化，必须连接真实以太坊 RPC |
| **多链索引** | ❌ 断裂 | 每条链需要独立 Puller + ChainIndexer + ReorgHandler |
| **Reorg 处理** | ❌ 断裂 | ReorgHandler 已实现但从未被调用 |
| **生产可用** | ❌ 断裂 | 缺少 checkpoint 持久化、RPC 故障切换、背压控制 |

### 当前状态：10 个断裂点

#### 断裂 1: EventBus 从未被创建
- 文件: `cmd/monolithic/chainpulse/main.go:196`
- 代码: `nil, // eventBus`
- 修复: 创建 `core.NewEventBus(logger)`，传给所有 Puller 和 ChainIndexer

#### 断裂 2: QueryService 和 IndexingStorage 用不同的 DB
- IndexingStorage 用 `MonolithicMemoryDatabase`，QueryService 用 MongoDB/PostgreSQL
- 修复: 让 QueryService 也使用同一个 `indexingDatabase` 实例

#### 断裂 3: Puller 从未被实例化（多链）
- `cmd/monolithic/chainpulse/main.go` 中没有 Puller 代码
- **`pkg/plugins/pullers/https_jsonrpc_puller.go` 已有完整实现**，支持 Start/PullEvents/PublishEvents
- 修复: 为每条链实例化一个 HTTPSJSONRPCPuller

#### 断裂 4: Puller 的 EventBus 是 nil
- `BaseDataPullerPlugin` 有 `eventBus` 和 `PublishEvents()`，但传的是 nil
- 修复: 把断裂 1 的 EventBus 传给每个 Puller

#### 断裂 5: 没有 Puller → Indexer 的调用链
- `HTTPSJSONRPCPuller.Start()` 只设置 running 状态，**没有内部拉取循环**
- `SharedRuntime.ProcessBatch()` 存在但没人调用
- 修复: 在 main.go 中为每条链启动 goroutine 循环

#### 断裂 6: ReorgHandler 从未被调用
- `pkg/services/reorg/reorg_handler.go` 有完整的 DetectReorg/HandleReorg/RollbackEvents
- 但 ChainIndexer 中没有任何 reorg 检查逻辑
- 修复: 在拉取循环中，每次拉取前检查 block hash 是否变化，检测到 reorg 时调用 ReorgHandler

#### 断裂 7: logToEvent 没有填充 BlockHash
- `https_jsonrpc_puller.go` 的 `logToEvent()` 方法中，`BlockHash` 字段未被赋值
- 但 `Log` 结构体有 `BlockHash` 字段（来自 `eth_getLogs` 返回）
- `core.BlockchainEvent` 也有 `BlockHash` 字段
- 修复: 在 `logToEvent()` 中添加 `BlockHash: common.HexToHash(log.BlockHash)`

#### 断裂 8: 多链配置未解析为独立 Puller
- `main.go` 有 `parseChains(config.Chains)` 解析链列表
- 有 `config.BlockchainNodeURLs` 解析节点 URL 列表
- 但没有为每条链创建独立的 Puller 实例
- 修复: 遍历 chains，为每条链创建独立的 HTTPSJSONRPCPuller

#### 断裂 9: SharedRuntime 的 Sink wiring 需要确认
- 需要确认 `BuildMonolithicIndexingRuntime` 的 EventSink 指向 `MonolithicMemoryDatabase`

#### 断裂 10: Checkpoint 未持久化
- `MonolithicMemoryDatabase` 的 lastIndexedBlock 在重启后丢失
- M1 阶段可以接受内存 checkpoint，但需要有 checkpoint 机制
- 修复: 在拉取循环中维护每条链的 lastIndexedBlock，启动时从 SharedRuntime 加载

### 完整数据流（修复后）

```
for each chain (ethereum, polygon, ...):
  HTTPSJSONRPCPuller(chainID, nodeURL, eventBus)
    → Start()
    → goroutine 循环:
      1. 检查 reorg: 对比当前 block hash 与上次记录的 hash
         - 如果不同 → ReorgHandler.DetectReorg() → HandleReorg() → RollbackEvents()
      2. fromBlock = checkpoint[chainID] + 1
      3. toBlock = min(fromBlock + 10, chainHead)
      4. events := puller.PullEvents(ctx, fromBlock, toBlock)
      5. eventBus.Publish("blockchain-events", events)
      6. envelopes := toEventEnvelopes(events)
      7. chainIndexer.ProcessBatch(ctx, chainID, envelopes)
         → SharedRuntime.ProcessBatch(ctx, chainID, envelopes)
         → EventSink.Persist(ctx, envelopes)
         → LegacyRuntimeSink → MonolithicMemoryDatabase
      8. checkpoint[chainID] = toBlock
      9. sleep 10 秒
```

### 目标
修复上述 10 个断裂，使 `make run-monolithic` 能：
1. 从**真实以太坊链**拉取事件
2. 支持**多链并行索引**（默认 ethereum + polygon）
3. **检测并处理链重组**（reorg）
4. 具备**生产级可靠性**（checkpoint、RPC 错误处理、背压）

### 成功标准

#### 基础（必须全部通过）
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 多链索引
- [ ] 启动日志显示每条链的 Puller 和 Indexer 都已注册
- [ ] `CHAINS=ethereum,polygon` 时，两条链同时拉取
- [ ] 每条链有独立的 checkpoint 追踪

#### 真实数据
- [ ] 启动后 60 秒内，GraphQL 查询返回**真实链上事件**
- [ ] 返回的数据有真实的 tx hash（0x 开头，64 字符）和 block hash
- [ ] 返回的数据有真实的 contract address

#### Reorg 处理
- [ ] ReorgHandler 被正确初始化并接入索引流
- [ ] 拉取循环中包含 block hash 对比逻辑
- [ ] 检测到 reorg 时调用 RollbackEvents 回滚数据

#### 生产可用性
- [ ] 有 checkpoint 机制（内存即可，M1 不要求持久化到磁盘）
- [ ] RPC 错误时有重试和日志（HTTPSJSONRPCPuller 已有）
- [ ] 拉取循环有背压控制（eventBus 满时暂停）

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
- `cmd/monolithic/chainpulse/main.go` — 单体入口，需要修复 wiring
- `pkg/core/eventbus.go` — EventBus 实现
- `pkg/core/config.go` — Config 结构体
- `pkg/core/blockchain_models.go` — BlockchainEvent 结构体（含 BlockHash 字段）
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — HTTPSJSONRPCPuller（需修复 logToEvent 的 BlockHash）
- `pkg/plugins/pullers/data_puller.go` — BaseDataPullerPlugin（含 eventBus 和 PublishEvents）
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler（已有完整实现）
- `pkg/adapters/indexing/monolithic_memory_storage.go` — MonolithicMemoryDatabase
- `pkg/application/indexing/runtime.go` — EventEnvelope、EventSink、SharedRuntime
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer.ProcessBatch
- `pkg/services/indexing/legacy_runtime_sink.go` — LegacyRuntimeSink
- `pkg/application/bootstrap/runtime_wiring.go` — QueryService wiring
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring

### 修复步骤（按顺序）

**Step 1: 修复 logToEvent 的 BlockHash**
```
文件: pkg/plugins/pullers/https_jsonrpc_puller.go
在 logToEvent() 方法中，添加:
  BlockHash: common.HexToHash(log.BlockHash)
这样 reorg 检测才能对比 block hash。
```

**Step 2: 创建 EventBus**
```
文件: cmd/monolithic/chainpulse/main.go
1. eventBus := core.NewEventBus(logger)
2. 传给所有 Puller 和 ChainIndexer
```

**Step 3: 为每条链实例化 Puller + Indexer + ReorgHandler**
```
文件: cmd/monolithic/chainpulse/main.go
1. 解析 chains = parseChains(config.Chains)
2. 解析 nodeURLs = strings.Split(config.BlockchainNodeURLs, ",")
3. for i, chainID := range chains:
     nodeURL := nodeURLs[i] (如果 URLs 少于 chains，循环使用)
     puller := NewHTTPSJSONRPCPuller(config, logger, metrics, eventBus)
     puller.SetNodeURL(nodeURL) // 需要确认是否有这个方法，没有就通过 config 传
     reorgHandler := reorg.NewReorgHandler(...)
     chainIndexer := NewDefaultChainIndexer(chainID, indexingDatabase, indexingCache, logger, nil)
     chainIndexer.SetSharedRuntime(sharedIndexingRuntime, metrics)
```

**Step 4: 统一 DB 来源**
```
让 QueryService 使用 indexingDatabase（MonolithicMemoryDatabase）。
最简单方案: 在 main.go 中直接创建基于 indexingDatabase 的 QueryService。
```

**Step 5: 启动多链拉取循环（含 reorg 检测）**
```
文件: cmd/monolithic/chainpulse/main.go
for each chain:
  go func(chainID string, puller *HTTPSJSONRPCPuller, chainIndexer *DefaultChainIndexer, reorgHandler *ReorgHandler) {
    checkpoint := loadCheckpoint(chainID) // 从 SharedRuntime 或内存
    for {
      // 1. 获取链头
      chainHead, err := puller.GetLatestBlock(ctx)

      // 2. Reorg 检测: 对比当前 block hash 与上次记录
      currentBlock := checkpoint
      if currentBlock > 0 {
        events, _ := puller.PullEvents(ctx, currentBlock, currentBlock)
        if len(events) > 0 && events[0].BlockHash != lastBlockHash[chainID] {
          // Reorg detected
          reorgHandler.DetectReorg(ctx, chainID, currentBlock, ...)
          reorgHandler.HandleReorg(ctx, ...)
          reorgHandler.RollbackEvents(ctx, ...)
        }
      }

      // 3. 拉取事件
      fromBlock := checkpoint + 1
      toBlock := min(fromBlock + 10, chainHead)
      events, err := puller.PullEvents(ctx, fromBlock, toBlock)

      // 4. 发布并索引
      if len(events) > 0 {
        eventBus.Publish("blockchain-events", events)
        envelopes := toEventEnvelopes(events)
        chainIndexer.ProcessBatch(ctx, chainID, envelopes)
      }

      // 5. 更新 checkpoint
      checkpoint = toBlock
      lastBlockHash[chainID] = events[len(events)-1].BlockHash

      // 6. 背压控制: 检查 eventBus 队列深度
      time.Sleep(10 * time.Second)
    }
  }(chainID, puller, chainIndexer, reorgHandler)
```

**Step 6: 确认 Sink wiring**
```
确认 BuildMonolithicIndexingRuntime 的 EventSink 指向 MonolithicMemoryDatabase。
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
- **必须使用真实 RPC 节点，不要用 MockPuller 或模拟数据**

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 手动验证多链
CHAINS=ethereum,polygon \
BLOCKCHAIN_NODE_URLS=https://eth.llamarpc.com,https://polygon-rpc.com \
make run-monolithic &
sleep 60
# 验证真实数据
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(limit: 5) { id chainId blockNumber transactionHash blockHash } }"}'
# 验证多链数据
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(chainId: \"ethereum\", limit: 2) { id blockNumber } }"}'
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(chainId: \"polygon\", limit: 2) { id blockNumber } }"}'
# 验证健康检查
curl -s http://localhost:8080/health
```
