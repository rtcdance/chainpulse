# 写路径勘查：端口栈 vs 实际部署栈（2026-10-13）

> 状态：**Finding，待作者决策**。这不是一个任务，而是"本轮哪一个栈是生产栈"的架构问题 —— 按计划 §12，范围/架构变更需人工决定，**本文件不擅自改动装配**。
> 所有结论都附 `文件:行` 证据；**未验证的部分显式标注**，不作推断。

---

## 0. 结论一句话

计划 §5 的 T02/T03/T05 都是围绕 **`core.DatabasePlugin` 端口**做的（PG 适配层、链作用域、单事务 `BatchCommitter`、fenced lease）。
但**端口不是部署中的存储**：单体路径的端口实现是**内存**（microservice 模式下甚至是 **MockDB**）；四服务路径用的是**另一套 service 级 store（Mongo + PG）**，它**不经过** `BatchCommitter`，也**没有**在本包内看到任何 checkpoint 或 offset 提交。

⇒ T05 关掉的那个半提交窗口，**在真正写 PG 的那条路径上仍然开着**。

---

## 1. 证据：生产装配里没有 Postgres 端口实现

### 1.1 唯一的数据库插件选择器返回内存 / MockDB

`pkg/application/bootstrap/indexing_storage.go:76-81`

```go
func newMonolithicIndexingDatabaseForMode(logger core.Logger, config core.Config) core.DatabasePlugin {
	if config.DeploymentMode == "microservice" {
		return newSnapshotCompatibleDatabase(plugindatabase.NewMockDB())
	}
	return NewMonolithicMemoryDatabase(logger)
}
```

缓存同理：`indexing_storage.go:22-24` → `NewMonolithicMemoryCache()`（恒定，不看 `CacheConnectionURL`）。

### 1.2 适配层与 Redis 适配层**在装配里零调用**

| 构造器 | 生产调用点 | 证据 |
| --- | --- | --- |
| `NewPostgreSQLPluginAdapter` | **无** | 全仓 10 处命中（含定义处 `postgres_plugin_adapter.go:42`），其余全在 `test/`（contracts 3、acceptance 3、acceptance/tools 2）；`cmd/` 与 `pkg/application` 零命中 |
| `NewPostgreSQLDatabase` | **无** | 命中全部在 `pkg/plugins/database/*_test.go` 与 `test/contracts/` |
| `NewRedisCachePlugin` | **无** | 命中全部在插件自身测试与 `test/contracts/` |
| `PostgreSQLPluginAdapter.WithChainScope` | **无** | 仅 `test/contracts/postgres_chain_isolation_test.go:202,208,236` |

生产侧调用链已确认：`pkg/application/bootstrap/monolithic_starter.go:64`
→ `BuildMonolithicIndexingStorage(ctx, logger, cfg)`
→ `indexing_storage.go:90` 用**默认 deps** → `newMonolithicIndexingDatabaseForMode`（§1.1）。

⇒ **T03 的链作用域机制无法接线** —— 因为装配根本不构造适配器；它不是"忘了声明链"，而是**没有可达路径**。

### 1.3 `AdapterFactory` 在生产中是死代码

`pkg/infrastructure/deployment/adapter_factory.go` 的默认类型是 `postgres`（`:76-79`），但

- `default_plugins.go:14-24` **只注册** `mq:memory`、`cache:inmemory`、`database:mock` —— **没有 `database:postgres`、没有 `cache:redis`**；
- `NewAdapterFactory` / `CreateDatabasePlugin` / `CreateCachePlugin` / `RegisterDefaultFactories` 的调用点**全部在 `pkg/infrastructure/deployment` 自身与其测试内**，`cmd/` 与 `pkg/application` **零调用**。

> 两点**未验证**：① 是否通过 `CHAINPULSE_DATABASE_TYPE=mock` 之类的环境覆盖规避了缺注册；② 是否存在本仓之外的注册路径。此处只陈述"仓内没有调用点"。

### 1.4 `DATABASE_TYPE` 被读出并打印，但不决定索引存储

`cmd/monolithic/chainpulse/main.go:625` 读 `DATABASE_TYPE`（默认 `postgres`），`main.go:67` 把它**打印**出来；而真实索引存储来自 §1.1 的 `BuildMonolithicIndexingStorage`。⇒ **配置项与生效行为不一致**，看启动日志会以为在跑 Postgres。

---

## 2. 证据：四服务走的是另一套 store，且没有本包内的 checkpoint / offset

### 2.1 四服务写路径 = service 级 store，不是端口

`cmd/microservices/event-processor/main.go:113-175`

```go
eventStore       := query.NewMongoDBEventStore(dbManager, logger, metrics, eventStoreConfig)
pgMetadataStore  := query.NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)
...
processorRuntime, err := newEventProcessorProcessingRuntimeWithStorage(
	config, logger, metrics,
	newPersistentEventProcessorStorage(eventStore, storageMetadataStore),   // ← 写路径在这里
)
```

`processorRuntime` **不接收任何 `core.DatabasePlugin`**。⇒ 端口栈的 `BatchCommitter` / `Transactioner` / `FencedTx` 在这条路径上**不可达**。

### 2.2 写路径只有三个方法，没有事务边界

`cmd/microservices/event-processor/persistent_storage.go`：

```
:31  func (s *persistentEventProcessorStorage) WriteEvent(...)
:81  func (s *persistentEventProcessorStorage) WriteBatch(...)
:140 func (s *persistentEventProcessorStorage) DeleteEvent(...)
```

方法集合里**没有** `BeginTx`、没有 `SaveCheckpoint`。`WriteBatch` 把事件写进 **Mongo 事件存储 + PG 元数据存储**（两个不同数据库）。

> **未验证**：`MongoDBEventStore.WriteBatch` 自身是否在单个 store 内原子（与本结论无关——**跨 Mongo/PG 两个 store 的原子性无法用单库事务表达**）。

### 2.3 本包内没有 checkpoint 与 offset 提交

对 `cmd/microservices/event-processor` 全包检索：

- `SaveCheckpoint` / `indexing_state` / `last_indexed_block` → **零命中**（唯一的 `LastCheckpoint*` 是 `shared_runtime_shadow.go:24-26` 里 rollout 摘要的状态字段，不是持久化 checkpoint）；
- `CommitOffset` / `Commit(` / `Ack(` → **零命中**。

而消费缝的构造签名（`consumer_runtime.go:71-79`）是
`consumer, processor, publisher, topics, outputTopics, dlqDB`
—— 第二个 `kafkaMQ` 是 **publisher**（输出 topic），**没有 committer 位**。

> **未验证**：Kafka 插件内部是否 auto-commit、以及 offset 语义是否由插件隐式承担。本文件不断言"完全没有 ack 协调"，只断言"**本包内没有**提交调用点，消费缝也没有 committer 参数"。

---

## 3. 对本轮与计划结论的影响（重要）

| 已"完成"项 | 实际覆盖范围 | 影响 |
| --- | --- | --- |
| T05 `BatchCommitter` / fenced lease / 原子提交 | **端口栈**（装配中 = 内存/MockDB） | 半提交窗口在四服务真实写路径上**仍开着** |
| T03 `WithChainScope` | 端口栈，且**无装配调用点** | 机制不可达；"多链下拒绝未限定方法"在生产不生效 |
| T02 PG/Redis 契约 runner | 只证明**适配器**合规 | §7.3"每个必需生产 factory executed>0"若指装配路径，则必需 factory 是 `NewMonolithicMemoryDatabase` / `NewMonolithicMemoryCache`，**不是** PG/Redis |
| §10 "契约/E2E 不空跑" | contracts lane 打端口；`evm-e2e` 打四服务 | **两条 lane 测的不是同一套实现** |
| §7.5 CD「应用 expand 迁移，启动单链写入者」 | 依赖端口栈接 PG | 按当前装配**不成立** |
| §8「PostgreSQL 是事实来源」 | 计划核心决策 I01–I07 | 与装配现状**不一致** |

计划 Risks 段已写"schema/处理路径分裂导致估时偏低"。本文件是**该分裂的具体形态**：不是 schema 分裂，而是**存储栈分裂**（端口栈 vs service-store 栈）。

---

## 4. 建议的决策点（不擅自执行）

三选一，**都需要作者确认**：

| 选项 | 含义 | 代价 |
| --- | --- | --- |
| **A. 端口栈为生产栈** | 让 `newMonolithicIndexingDatabaseForMode` 在配了 `DatabaseURL` 时返回 `PostgreSQLPluginAdapter`（并 `WithChainScope`），四服务改为构造端口实现 | 需要重写四服务的 processor 装配；`persistentEventProcessorStorage`（Mongo 事件 + PG 元数据）需要取舍或改为端口底座 |
| **B. service-store 栈为生产栈** | 承认端口栈是历史遗留；把 T05 的原子提交/checkpoint/fencing 语义**在 service-store 上重做**（Mongo 事件 + PG 元数据 ⇒ 需要 outbox/saga，因为跨库无法单事务） | 需要新的跨库一致性设计；§4.2 的"PG 事务包住事件与 checkpoint"前提改写 |
| **C. 冻结范围** | 把端口栈标为"单体验证路径"，四服务标为 experimental，DoD 只声明端口栈 | §8/§10 的多项承诺需要下修；简历不得声称 PostgreSQL 生产链路 |

**我倾向先做 A/B 之间的一个小切口而不是全量选边**：把"四服务 event-processor 的 checkpoint 与 offset 归属"measure 清楚（本包内没有 ⇒ 到底谁负责？），因为**任何选项都必须先回答这个问题**，而且它可能暴露出比栈选择更基础的缺口。这一步只读码/只测量，不改装配。

---

## 5. 本文件**没有**做什么

- 没有改动任何装配代码（`indexing_storage.go`、`main.go`、`adapter_factory.go` 未动）。
- 没有把 T02/T03/T05 的结论标为"生产可用"——它们仍然是**端口级**已验证。
- 没有断言"Mongo/Kafka 内部行为"——那些标为未验证。
- 没有删除或降级任何既有测试。
