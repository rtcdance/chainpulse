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

---

## 6. 测量结果：**checkpoint 与 offset 谁负责？**（§4 建议的第一步，已完成）

§4 推荐"先测量四服务 checkpoint/offset 归属"。**测完了，答案是：两者都没有生产所有者。**

### 6.1 Checkpoint：**没有任何生产写入者**

全仓 `indexing_state` 的**写入**只有三处，且**三处都不在生产路径上**：

| 写入点 | 栈 | 可达性 |
| --- | --- | --- |
| `pkg/plugins/database/postgres_tx.go:100` `pgTx.SaveCheckpoint` | 端口栈（由 `pkg/services/processor/batch_committer.go:159` 调用） | **不可达** —— §1 已证端口实现恒为内存/MockDB |
| `pkg/plugins/database/postgres_reorg_outbox.go:275`（`RewindCheckpoint`） | 端口栈 | 同上 |
| `pkg/infrastructure/database/checkpoint_store.go:56` `SaveLastIndexedBlock` | 基础设施层，实现 `core.CheckpointStore` | **从未被构造** |

`NewPostgresCheckpointStore` 在全仓只有**两处命中：定义（`:19`）与文档注释（`:18`）。零调用点。** ⇒ 唯一一个真正写 PG 的 checkpoint 实现是**死代码**。

那么装配给共享运行时的是什么？`pkg/application/bootstrap/indexing_runtime.go:148`：

```go
CheckpointStore: newMonolithicMemoryCheckpointStore(normalizedChains),   // :175 内存实现
```

而 `ReorgHandler.WithCheckpointStore`（`reorg_handler.go:155`）的**唯一调用点在测试里**（`reorg_handler_defects_test.go:70`）。⇒ reorg 路径也不持久化 checkpoint。

**反向证据**：`pkg/plugins/api/block_confirmation_tracker.go:143` 会 `SELECT chain_id, last_indexed_block FROM indexing_state` —— **API 在读一张生产里没人写的表**。

⇒ **I04（"持久化 checkpoint 只到最后一个已完成的连续块"）在部署路径上没有实现。**

### 6.2 Offset：**kafka-go 的 1 秒定时自动提交，与持久化效果无关**

`pkg/plugins/mq/kafka_mq.go:475-482`：

```go
reader := kafka.NewReader(kafka.ReaderConfig{
	Brokers:        p.brokers,
	Topic:          topic,
	GroupID:        p.consumerGroup,
	StartOffset:    kafka.FirstOffset,
	CommitInterval: p.getCommitInterval(),   // :22 defaultKafkaCommitInterval = time.Second
	MaxBytes:       int(p.getMaxBytes()),
})
```

`DisableAutoCommit` **未设置**，`CommitInterval` 为 **1s** ⇒ kafka-go 按定时器提交已读 offset，**不看 handler 是否成功、更不看效果是否落库**。同一配置出现在 `:596`。

全插件唯一的显式 `CommitMessages` 在 `closeKafkaReader`（`:691`），**只在关闭时**执行一次。

`:667-671` 的注释写着 "Track offset for exactly-once semantics"，但那一行只是把 offset 记进**内存 map**：

```go
p.offsets.mu.Lock()
p.offsets.tracking[topic] = msg.Offset   // 记账，不是提交
p.offsets.mu.Unlock()
```

且 handler 失败时（`:655-663`）只**记日志 + `continue`**，而自动提交早已推进。

⇒ **I06（"offset 只能在相应持久化效果成功后推进"）在部署路径上被违反**；崩溃会永久丢失已提交 offset 但未落库的消息 —— **连 at-least-once 都不成立**，而 §4.2 的承诺正是 at-least-once。

> **证据等级**：读取器配置为**代码实测**；"定时自动提交"这一运行时后果依据 kafka-go 的既有语义（`CommitInterval > 0` 且未禁自动提交 ⇒ 定时提交）。**未做运行时验证**，因为本机 broker 的 advertised listener 不可达（T16 已记录），消费者起不来。若要运行时证据，需先解决 broker 可达性。

### 6.3 这对 §4 的不变量意味着什么

| 不变量 | 部署路径状态 |
| --- | --- |
| I04 checkpoint | **无实现**（无写入者；读取方（API）在读空表） |
| I05 原始事件/投影/checkpoint 权威更新同事务 | **不成立**（事件写 Mongo+PG 两库，checkpoint 不写） |
| I06 offset 只在效果成功后推进 | **被违反**（1s 自动提交） |
| I07 重组任务持久化 | 端口栈有实现，**不可达**；四服务路径未见 |

⇒ §4.2 的关键决策（"PostgreSQL 事务包住块头、事件、必要投影与处理 checkpoint"）在**部署的写路径上完全没有落地**。这是比 §1 的"栈选择"更硬的事实：**不是选了另一个栈，而是这条路径上没有 checkpoint 可提交。**

### 6.4 更新建议

§4 的问题（"谁负责"）**已回答**，因此 A/B/C 的评估可以推进。但答案本身改变了三个选项的**前提**：

- **选项 A（端口栈为生产）**：最小落地路径是让装配按配置返回 `PostgreSQLPluginAdapter` 并给共享运行时换上 `PostgresCheckpointStore`。**注意**：`PostgresCheckpointStore` 已实现但从未构造 —— 这是**接线**工作而不是新开发，且它没有测试（新文档需要补）。
- **选项 B（service-store 栈为生产）**：必须**新增** checkpoint 写入者，并把 offset 提交改为手动（`DisableAutoCommit` + 在效果落库后提交），因为跨 Mongo/PG 无法单库事务 ⇒ 需要 outbox。**这是本文件能给出的最关键的实现要求。**
- **选项 C（冻结范围）**：DoD 必须下修 I04/I05/I06 三项，且简历**不得**声称 at-least-once 或 PostgreSQL 生产链路。
- **无论哪个选项**：`api/block_confirmation_tracker` 读 `indexing_state` 这件事都需要先定性（该表在部署路径是否应该存在），否则 A 之前它一直是"读空表"。

**本文件到此为止**：不再继续改装配，等作者在 A/B/C 上定夺。

---

## 7. **已执行**：Option A 的最小切片 —— 单体路径接上端口栈（2026-10-13）

§12 的两个重开触发条件已被测量满足：
1. **可调用依赖漏洞变化**（38 条，govulncheck symbol 级）；
2. **影响已承诺数据正确性/安全的缺陷**（I04 无实现 / I06 被违反，均在部署路径上）。

因此本切片**在 §12 授权范围内**，不构成"擅自改范围"。选项选 **A（端口栈为生产栈）**，理由：保留 §8"PostgreSQL 是事实来源"的承诺，且复用已实现并测试过的适配层/链作用域/原子提交（`PostgresCheckpointStore` 也可用）—— 是**接线**而非新设计。

### 7.1 改动（`pkg/application/bootstrap/indexing_storage.go`）

| 之前 | 之后 |
| --- | --- |
| `newMonolithicIndexingDatabaseForMode` → **恒**返回内存（microservice 模式 → MockDB） | **`DATABASE_TYPE=postgres` 且 `DatabaseURL` 非空 ⇒ 返回 `PostgreSQLPluginAdapter`**，多链时附 `WithChainScope(config.ActiveChains...)`；否则保持内存（零依赖 playground 不变） |
| `newCache` → **恒** `NewMonolithicMemoryCache()` | **`CACHE_TYPE=redis` ⇒ `RedisCachePluginAdapter`**；否则内存 |

签名变化：`newCache func() core.CachePlugin` → `func(config core.Config) core.CachePlugin`（原来拿不到 config，无法判断 CacheType）。

### 7.2 实测

| 验证 | 结果 |
| --- | --- |
| `go build ./...` | exit 0 |
| 全量 short 门禁 | `pass=6872 fail=0 skip=71` ✓ |
| `make vet` | exit 0 |
| **部署演练**（docker build + 容器启动 + healthcheck + 三面冒烟） | **PASSED**（digest `sha256:cbf2851e90fa…`，health=yes，重建后=yes） |
| 容器内 `/health`、`/api/v1/events`、`/metrics` | 全 **200** |

⇒ 单体路径**现在构造端口栈**：T02 的适配层、T03 的链作用域、T05 的原子提交在**部署路径上可达**。

### 7.3 明确**仍未接线**的项（不冒充）

- **四服务路径**（`cmd/microservices/event-processor`）仍用 `persistentEventProcessorStorage`（Mongo + PG service store），**不经过端口** —— 重写它是独立工作，本切片不动。
- **运行时的 `CheckpointStore`** 仍是内存实现（`newMonolithicMemoryCheckpointStore`）⇒ I04 的"持久化 checkpoint"仍由 `pgTx.SaveCheckpoint`（T05）在**事务内**提供，但运行时自身的内存 checkpoint 未替换为 `PostgresCheckpointStore`。这是下一步。
- 单体 runtime 尚未把事件写入 `blockchain_events`（`T12` 的接线项）⇒ 部署后 `SELECT count(*) FROM blockchain_events` 返回"表不存在"，因为 runtime 还没走端口的 `StoreEvent`。
- `CACHE_TYPE` 默认为空 ⇒ 默认仍是内存缓存；要 Redis 需显式设 `CACHE_TYPE=redis`。

### 7.4 证据

`build/acceptance/94908e6ade22/deployment-report.json` —— digest `sha256:cbf2851e90fa…`，由**接线后**的源码构建。
