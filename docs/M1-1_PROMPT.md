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
`make run-monolithic` 能启动，但**数据链路是断的**。以下是已确认的 4 个断裂点：

#### 断裂 1: EventBus 从未被创建
- 文件: `cmd/monolithic/chainpulse/main.go:196`
- 代码: `nil, // eventBus`
- 影响: ChainIndexer 的 EventBus 是 nil，无法接收 Puller 拉取的事件
- 修复方向: 在 main.go 中创建 `core.NewEventBus(logger)`，传给每个 ChainIndexer

#### 断裂 2: QueryService 和 IndexingStorage 用不同的 DB
- IndexingStorage (`pkg/application/bootstrap/indexing_storage.go`) 创建的是 `MonolithicMemoryDatabase`
- QueryService (`pkg/application/bootstrap/runtime_wiring.go` → `BuildRuntimeWiring`) 创建的是 MongoDB/PostgreSQL adapter
- 影响: Indexer 写入内存 DB，QueryService 从 MongoDB/PostgreSQL 读 — **永远查不到数据**
- 修复方向: 让 QueryService 也使用 `indexingDatabase`（MonolithicMemoryDatabase），或者让 QueryService 的 DB adapter 指向同一个内存实例

#### 断裂 3: Puller 从未被实例化或启动
- `cmd/monolithic/chainpulse/main.go` 中没有 Puller 的实例化代码
- `pkg/plugins/pullers/multi_chain_puller.go` 有 `MultiChainDataPuller`，但没有被使用
- 影响: 没有数据源，系统空转
- 修复方向: 在 main.go 中实例化 MockPuller 或 MemoryPuller，注册到 MultiChainDataPuller，启动拉取循环

#### 断裂 4: 没有 Pull → Indexer 的循环驱动
- `SharedRuntime.Start()` 存在但不包含拉取循环
- `MultiChainDataPuller` 有 `PullEventsFromAllChains()` 但需要外部调用
- 影响: 即使 Puller 和 Indexer 都初始化了，也没有循环驱动它们工作
- 修复方向: 在 main.go 中启动一个 goroutine，循环调用 `PullEventsFromAllChains()` → 通过 EventBus 发布 → ChainIndexer 消费

### 目标
修复上述 4 个断裂，使 `make run-monolithic` 能完成 Puller → EventBus → Indexer → DB → Query API 的完整数据流。

### 成功标准
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic
- [ ] 启动后 30 秒内，`curl http://localhost:8080/graphql` 能执行查询并返回数据
- [ ] 日志中能看到 Puller 拉取区块和 Indexer 处理事件的输出

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`，特别是:
1. 新代码只写入正确的层
2. 不要往 `pkg/domain/`、`pkg/application/`、`pkg/adapters/` 添加新功能
3. 不要修改已有依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
4. 不要重构已工作的代码

### 参考文件
- `cmd/monolithic/chainpulse/main.go` — 单体入口，需要修复 wiring
- `pkg/core/eventbus.go` — EventBus 实现（DefaultEventBus，可直接用）
- `pkg/adapters/indexing/monolithic_memory_storage.go` — MonolithicMemoryDatabase（Indexer 写入的 DB）
- `pkg/application/bootstrap/runtime_wiring.go` — QueryService wiring（需要修改 DB 来源）
- `pkg/application/bootstrap/indexing_storage.go` — IndexingStorage wiring
- `pkg/plugins/pullers/multi_chain_puller.go` — MultiChainDataPuller
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — HTTP JSON-RPC Puller 实现
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer（有 ProcessBatch 方法）

### 修复步骤（按顺序）

**Step 1: 创建 EventBus 并传给 ChainIndexer**
```
在 main.go 中:
1. 创建 eventBus := core.NewEventBus(logger)
2. 把 eventBus 传给每个 ChainIndexer（替换 nil）
```

**Step 2: 统一 DB 来源**
```
让 QueryService 使用 indexingDatabase（MonolithicMemoryDatabase）而不是 MongoDB/PostgreSQL adapter。
方案 A: 修改 BuildRuntimeWiring，让它接受一个可选的 DatabasePlugin 参数
方案 B: 在 main.go 中直接创建一个基于 indexingDatabase 的 QueryService
选择更简单的方案，不要过度设计。
```

**Step 3: 实例化并启动 Puller**
```
在 main.go 中:
1. 创建 puller := pullers.NewMultiChainDataPuller(logger)
2. 为每个 chain 注册一个 MockPuller 或 MemoryPuller
3. 启动一个 goroutine，循环调用 PullEventsFromAllChains()
4. 拉取的事件通过 eventBus.Publish() 发布
```

**Step 4: 连接 Puller → EventBus → Indexer**
```
1. ChainIndexer 订阅 EventBus 的 "new-events" topic
2. Puller 拉取事件后通过 eventBus.Publish("new-events", events) 发布
3. ChainIndexer 收到事件后调用 ProcessBatch 处理
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
# 验证 API 返回数据
curl -s http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events(limit: 5) { id chainId blockNumber eventName } }"}'
# 验证健康检查
curl -s http://localhost:8080/health
```
