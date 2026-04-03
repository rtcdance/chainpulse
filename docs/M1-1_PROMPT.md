# M1-1: 修复单体端到端链路

> 这是可以直接发给 GPT 的完整 prompt。

---

## 任务: M1-1 - 修复单体端到端链路

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`

### 当前状态
`make run-monolithic` 能启动，但**数据链路是断的**。以下是已确认的 6 个断裂点：

#### 断裂 1: EventBus 从未被创建
- 文件: `cmd/monolithic/chainpulse/main.go:196`
- 代码: `nil, // eventBus`
- 影响: ChainIndexer 的 EventBus 是 nil
- 修复: 在 main.go 中创建 `core.NewEventBus(logger)`

#### 断裂 2: QueryService 和 IndexingStorage 用不同的 DB
- IndexingStorage 用的是 `MonolithicMemoryDatabase`（内存）
- QueryService 用的是 MongoDB/PostgreSQL adapter（外部数据库）
- 影响: Indexer 写入内存，QueryService 读外部库 — **永远查不到数据**
- 修复: 让 QueryService 也使用同一个 `indexingDatabase` 实例

#### 断裂 3: 没有 MockPuller
- `pkg/plugins/pullers/` 里所有 Puller（HTTPSJSONRPCPuller、GRPCPuller、WebSocketJSONRPCPuller）都需要真实 RPC 节点
- **没有 MockPuller 或 MemoryPuller**
- 修复: 新建 `pkg/plugins/pullers/mock_puller.go`，产生合理的 `core.BlockchainEvent`

#### 断裂 4: Puller 从未被实例化
- `cmd/monolithic/chainpulse/main.go` 中没有 Puller 代码
- 修复: 在 main.go 中实例化 MockPuller，注册到 MultiChainDataPuller

#### 断裂 5: 没有 Puller → Indexer 的调用链
- 当前没有任何代码把 Puller 拉取的事件传给 Indexer
- 修复: 在 main.go 中启动 goroutine 循环: Puller 拉取 → 调用 ChainIndexer.ProcessBatch

#### 断裂 6: SharedRuntime 的 Sink 没有指向 MonolithicMemoryDatabase
- `SharedRuntime` 需要 `EventSink` 接口来持久化数据
- `LegacyRuntimeSink` 实现了 `EventSink`，它把 `EventEnvelope` 转回 `BlockchainEvent` 存入 DB
- 但 monolithic 中 `SharedRuntime` 的 Sink 是否指向 `MonolithicMemoryDatabase` 需要确认
- 修复: 确认或修复 Sink 的 wiring

### 完整数据流（修复后应该是这样）

```
MockPuller.PullEvents() 
  → 产生 []core.BlockchainEvent
  → 调用 ChainIndexer.ProcessBatch(ctx, chainID, envelopes)
  → ChainIndexer 转换为 []EventEnvelope
  → 调用 SharedRuntime.ProcessBatch(ctx, chainID, envelopes)
  → SharedRuntime 验证、去重
  → 调用 EventSink.Persist(ctx, envelopes)
  → LegacyRuntimeSink 转回 BlockchainEvent 存入 MonolithicMemoryDatabase
  → QueryService 从 MonolithicMemoryDatabase 查询
  → GraphQL API 返回数据
```

### 目标
修复上述 6 个断裂，使 `make run-monolithic` 能完成完整数据流。

### 成功标准
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic
- [ ] 启动后 30 秒内，`curl http://localhost:8080/graphql` 能执行查询并返回非空数据
- [ ] 日志中能看到 Puller 拉取区块和 Indexer 处理事件的输出

### MockPuller 规格

新建 `pkg/plugins/pullers/mock_puller.go`：

```go
// MockPuller 产生模拟的 BlockchainEvent，不需要真实 RPC 节点
type MockPuller struct {
    chainID     string
    currentBlock uint64
    logger      core.Logger
    mu          sync.Mutex
}

// 要求:
// 1. 实现 core.DataPullerPlugin 接口
// 2. PullEvents 返回 1-5 个模拟事件，block number 递增
// 3. 事件必须有正确的 ID 格式: "evt-{chainID}-{block}-{logIndex}"
// 4. 事件必须有正确的 ChainID、BlockNumber、BlockHash、LogIndex
// 5. 每次调用 PullEvents 后 currentBlock 递增
// 6. 支持配置起始 block 和每次拉取的 block 数量
```

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`，特别是:
1. MockPuller 写在 `pkg/plugins/pullers/mock_puller.go`
2. 新代码只写入正确的层
3. 不要往 `pkg/domain/`、`pkg/application/`、`pkg/adapters/` 添加新功能
4. 不要修改已有依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
5. 不要重构已工作的代码

### 参考文件
- `cmd/monolithic/chainpulse/main.go` — 单体入口，需要修复 wiring
- `pkg/core/eventbus.go` — EventBus 实现（DefaultEventBus）
- `pkg/core/plugin.go` — DataPullerPlugin 接口定义
- `pkg/core/blockchain_models.go` — BlockchainEvent 结构体
- `pkg/adapters/indexing/monolithic_memory_storage.go` — MonolithicMemoryDatabase
- `pkg/application/indexing/runtime.go` — EventEnvelope、EventSink 接口、SharedRuntime
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer.ProcessBatch 方法
- `pkg/services/indexing/legacy_runtime_sink.go` — LegacyRuntimeSink 实现
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring

### 修复步骤（按顺序）

**Step 1: 创建 MockPuller**
```
新建 pkg/plugins/pullers/mock_puller.go:
1. 实现 core.DataPullerPlugin 接口
2. PullEvents 返回模拟的 BlockchainEvent，block number 递增
3. 事件 ID 格式: "evt-{chainID}-{block}-{logIndex}"
4. 写单元测试
```

**Step 2: 创建 EventBus 并统一 DB**
```
在 main.go 中:
1. 创建 eventBus := core.NewEventBus(logger)
2. 确认 BuildMonolithicIndexingRuntime 的 Sink 指向 MonolithicMemoryDatabase
3. 让 QueryService 也使用 indexingDatabase（而不是 MongoDB/PG adapter）
   - 最简单方案: 在 BuildRuntimeWiring 中传入 indexingDatabase 作为 DB 来源
```

**Step 3: 实例化 Puller 并注册**
```
在 main.go 中:
1. 创建 mockPuller := pullers.NewMockPuller(chainID, logger)
2. 创建 multiPuller := pullers.NewMultiChainDataPuller(logger)
3. multiPuller.RegisterPuller(chainID, mockPuller)
```

**Step 4: 建立 Puller → Indexer 调用链**
```
在 main.go 中启动 goroutine:
1. 循环调用 multiPuller.PullEventsFromAllChains()
2. 对每个 chain 的事件，转换为 EventEnvelope
3. 调用 chainIndexer.ProcessBatch(ctx, chainID, envelopes)
4. sleep pollInterval（如 5 秒）
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 手动验证
make run-monolithic &
sleep 30
# 验证 GraphQL 返回数据
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(limit: 5) { id chainId blockNumber eventName } }"}'
# 验证健康检查
curl -s http://localhost:8080/health
# 验证日志中有 Puller 和 Indexer 活动
```
