# M1-1: 修复单体端到端链路（使用真实链上数据）

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
- 影响: ChainIndexer 和 Puller 的 EventBus 都是 nil
- 修复: 在 main.go 中创建 `core.NewEventBus(logger)`，同时传给 Puller 和 ChainIndexer

#### 断裂 2: QueryService 和 IndexingStorage 用不同的 DB
- IndexingStorage 用的是 `MonolithicMemoryDatabase`（内存）
- QueryService 用的是 MongoDB/PostgreSQL adapter（外部数据库）
- 影响: Indexer 写入内存，QueryService 读外部库 — **永远查不到数据**
- 修复: 让 QueryService 也使用同一个 `indexingDatabase` 实例

#### 断裂 3: Puller 从未被实例化
- `cmd/monolithic/chainpulse/main.go` 中没有 Puller 代码
- **但 `pkg/plugins/pullers/https_jsonrpc_puller.go` 已有完整的 HTTPS-JSONRPC Puller 实现**
- 它支持 `Start()`、`PullEvents()`、`PublishEvents()`（通过 EventBus 发布）
- 修复: 在 main.go 中实例化 `HTTPSJSONRPCPuller`，配置公共以太坊 RPC 端点

#### 断裂 4: Puller 的 EventBus 是 nil
- `BaseDataPullerPlugin` 有 `eventBus` 字段和 `PublishEvents()` 方法
- 但创建 Puller 时 eventBus 传的是 nil
- 修复: 把断裂 1 创建的 EventBus 传给 Puller

#### 断裂 5: 没有 Puller → Indexer 的调用链
- `SharedRuntime` 有 `ProcessBatch()` 方法，但没人调用它
- `HTTPSJSONRPCPuller` 的 `Start()` 只设置 running 状态，**没有内部拉取循环**
- 修复: 在 main.go 中启动 goroutine 循环: Puller 拉取 → EventBus 发布 → ChainIndexer.ProcessBatch

#### 断裂 6: SharedRuntime 的 Sink wiring 需要确认
- `SharedRuntime` 需要 `EventSink` 接口来持久化数据
- `LegacyRuntimeSink` 实现了 `EventSink`，把 `EventEnvelope` 转回 `BlockchainEvent` 存入 DB
- 需要确认 Sink 是否正确指向 `MonolithicMemoryDatabase`

### 完整数据流（修复后应该是这样）

```
HTTPSJSONRPCPuller (连接真实以太坊 RPC 节点)
  → PullEvents(fromBlock, toBlock) 拉取真实链上事件
  → eventBus.Publish("blockchain-events", events) 发布
  → ChainIndexer 订阅 EventBus，收到事件
  → ChainIndexer.ProcessBatch(ctx, chainID, envelopes)
  → SharedRuntime.ProcessBatch(ctx, chainID, envelopes)
  → EventSink.Persist(ctx, envelopes)
  → LegacyRuntimeSink 转回 BlockchainEvent 存入 MonolithicMemoryDatabase
  → QueryService 从 MonolithicMemoryDatabase 查询
  → GraphQL API 返回真实链上数据
```

### 目标
修复上述 6 个断裂，使 `make run-monolithic` 能从**真实以太坊链**拉取事件，完成完整数据流。

### 成功标准
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic
- [ ] 启动后 60 秒内，`curl http://localhost:8080/graphql` 能查询到**真实链上事件**（有真实的 tx hash、block hash）
- [ ] 日志中能看到 HTTPSJSONRPCPuller 连接 RPC 节点、拉取区块、发布事件的输出

### 可用的公共以太坊 RPC 端点

以下端点免费、无需 API key、支持 `eth_getLogs`：

```
https://eth.llamarpc.com
https://rpc.ankr.com/eth
https://ethereum-rpc.publicnode.com
https://1rpc.io/eth
```

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`，特别是:
1. 新代码只写入正确的层
2. 不要往 `pkg/domain/`、`pkg/application/`、`pkg/adapters/` 添加新功能
3. 不要修改已有依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
4. 不要重构已工作的代码

### 参考文件
- `cmd/monolithic/chainpulse/main.go` — 单体入口，需要修复 wiring
- `pkg/core/eventbus.go` — EventBus 实现（DefaultEventBus）
- `pkg/core/config.go` — Config 结构体，含 BlockchainNodeURL、StartBlock
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — HTTPSJSONRPCPuller（已有完整实现，含 Start/Stop/PullEvents/PublishEvents）
- `pkg/plugins/pullers/data_puller.go` — BaseDataPullerPlugin（含 eventBus 和 PublishEvents）
- `pkg/adapters/indexing/monolithic_memory_storage.go` — MonolithicMemoryDatabase
- `pkg/application/indexing/runtime.go` — EventEnvelope、EventSink 接口、SharedRuntime.ProcessBatch
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer.ProcessBatch
- `pkg/services/indexing/legacy_runtime_sink.go` — LegacyRuntimeSink 实现
- `pkg/application/bootstrap/runtime_wiring.go` — QueryService wiring
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring

### 修复步骤（按顺序）

**Step 1: 创建 EventBus**
```
在 main.go 中:
1. eventBus := core.NewEventBus(logger)
2. 把 eventBus 同时传给 HTTPSJSONRPCPuller 和 ChainIndexer
```

**Step 2: 实例化 HTTPSJSONRPCPuller**
```
在 main.go 中:
1. 创建 config，设置 BlockchainNodeURL 为公共 RPC（如 https://eth.llamarpc.com）
2. 设置 StartBlock 为最近 100 个块前，这样能快速拉到数据
3. puller := pullers.NewHTTPSJSONRPCPuller(config, logger, metrics, eventBus)
4. puller.Start()
```

**Step 3: 统一 DB 来源**
```
让 QueryService 使用 indexingDatabase（MonolithicMemoryDatabase）而不是 MongoDB/PostgreSQL adapter。
最简单方案: 在 main.go 中直接创建基于 indexingDatabase 的 QueryService，
绕过 BuildRuntimeWiring 的 MongoDB/PG 初始化逻辑。
```

**Step 4: 建立 Puller → EventBus → Indexer 调用链**
```
在 main.go 中启动 goroutine:
1. 获取当前链头 block number（通过 puller.GetLatestBlock）
2. 计算 fromBlock = chainHead - 100（拉取最近 100 个块）
3. toBlock = chainHead
4. events, err := puller.PullEvents(ctx, fromBlock, toBlock)
5. if len(events) > 0:
     eventBus.Publish("blockchain-events", events)
     envelopes := toEventEnvelopes(events)
     chainIndexer.ProcessBatch(ctx, chainID, envelopes)
6. sleep 10 秒（避免频繁 RPC 调用）
7. 更新 fromBlock = toBlock + 1，继续循环
```

**Step 5: 确认 Sink wiring**
```
确认 BuildMonolithicIndexingRuntime 的 EventSink 指向 MonolithicMemoryDatabase。
如果不是，修复它。
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
- **必须使用真实以太坊 RPC 节点，不要用 MockPuller 或模拟数据**

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 手动验证
make run-monolithic &
sleep 60
# 验证 GraphQL 返回真实链上数据
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(limit: 5) { id chainId blockNumber transactionHash blockHash } }"}'
# 验证返回的数据有真实的 tx hash（0x 开头，64 字符）和 block hash
# 验证健康检查
curl -s http://localhost:8080/health
```
