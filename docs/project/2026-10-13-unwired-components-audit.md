# 未接线组件审计（2026-10-13）

> 目的：把"实现了、有测试、但生产从不构造"这一类缺陷**一次性枚举**，而不是一个个手工撞见。
> 工具：`scripts/audit/unwired-components.py`（可重复运行）。
> 结论：**348 个导出构造器中有 163 个在生产代码里没有调用方**（已过滤测试替身与测试支持目录）。这不是 163 个缺陷，但它解释了为什么多个 T 任务的"已完成"在生产里不生效。

---

## 1. 为什么会有这份审计

本仓库的多个发现是**同一形状**：组件被实现、被单测、然后**从未在测试之外被构造**，于是测试全绿而功能在部署的二进制里什么也不做。手工撞到的例子：

| 组件 | 位置 | 发现于 |
| --- | --- | --- |
| `MetricLabelGuard` / `LagRecorder` | `pkg/observability` | T13（已接线 guard） |
| `IndexerMetrics` / `IndexerHealth` | `pkg/observability` | T13（lag 信号在生产中不存在） |
| `VersionedCache` | `pkg/plugins/cache` | T11 |
| `OutboxDrainerLoop` | `pkg/services/reorg` | T06 |
| `BatchCommitter` | `pkg/services/processor` | F05 |

一个个撞见太慢，而且容易在"找到几个就以为找完了"处停下。故写成脚本。

## 2. 工具做什么

对 `pkg/`、`cmd/` 下每个非测试文件里 `func New*` / `func Build*` 形状的导出构造器，统计它在

- **生产**（非 `_test.go`，不含定义行本身）与
- **测试**（`_test.go`）

中的文本引用数。**生产 0 且测试 > 0** ⇒ 列为候选。

```bash
python3 scripts/audit/unwired-components.py            # 人读
python3 scripts/audit/unwired-components.py --json out.json
python3 scripts/audit/unwired-components.py --include-test-doubles --include-test-helpers
```

## 3. 结果

```
scanned 781 Go files under pkg cmd
exported New*/Build* constructors: 348
test-only, after filtering 10 test double(s) and 3 test-support file(s): 163
```

按目录：

| 区域 | 数量 |
| --- | --- |
| `pkg/plugins` | 53 |
| `pkg/infrastructure` | 53 |
| `pkg/services` | 20 |
| `pkg/core` | 13 |
| `pkg/observability` | 6 |
| 其他（consensus / integrations / defi / gas / application / mempool / domain / siwe） | 18 |

测试引用最多（即"测得最狠、生产零调用"）的 12 个：

| 测试引用 | 构造器 | 位置 |
| --- | --- | --- |
| 123 | `NewBaseMQPlugin` | `pkg/core/mq_plugin.go:89` |
| 107 | `NewContractManager` | `pkg/services/decoder/contract_manager.go:31` |
| 100 | `NewBaseRequest` | `pkg/plugins/api/core/request.go:47` |
| 85 | `NewProtocolDetector` | `pkg/plugins/api/core/detector.go:16` |
| 79 | `NewEventDecoder` | `pkg/services/decoder/event_decoder.go:36` |
| 59 | `NewIndexerMetrics` | `pkg/observability/indexer_metrics.go:69` |
| 58 | `NewGenericContractIndexer` | `pkg/integrations/generic/generic_indexer.go:63` |
| 50 | `NewCacheMiddleware` | `pkg/plugins/api/cache_middleware.go:105` |
| 38 | `NewServiceRegistry` | `pkg/infrastructure/discovery/service_registry.go:40` |
| 36 | `NewGracefulShutdownManager` | `pkg/infrastructure/reliability/graceful_shutdown.go:48` |
| 31 | `NewBlockchainLogic` | `pkg/infrastructure/blockchain/blockchain_logic.go:58` |
| 30 | `NewSchemaBuilder` | `pkg/plugins/api/graphql/schema.go:52` |

## 4. 工具本身被检查过（两个方向）

**零命中型结论必须双向验证**——本仓库已被"BSD grep 不支持 `\|`"坑过多次，故本工具用 Python 实现，并做了对照：

**方向一：已知已接线的必须不出现** ✅

| 符号 | 结果 |
| --- | --- |
| `NewKafkaMQPlugin` / `NewHTTPSJSONRPCPuller` / `NewDefaultLogger` / `NewSharedRuntime` / `BuildMonolithicStarter` / `NewMultiChainIndexer` / `NewDefaultChainIndexer` / `NewMetricLabelGuard` | **均不在列表** |

**方向二：已知缺口必须出现** ✅

| 符号 | 结果 | 对应 |
| --- | --- | --- |
| `NewBatchCommitter` | 在列表 | F05（`options.Committer` 从未设置） |
| `NewVersionedCache` | 在列表 | T11 |
| `NewOutboxDrainerLoop` | 在列表 | T06 |
| `NewIndexerMetrics` / `NewIndexerHealth` | 在列表 | T13（lag 信号不存在） |

**过程中修掉的一个假阴性**：第一版把**文档注释**里出现的构造器名也算作生产引用（`// NewLagRecorder creates a recorder.` 这类），于是恰好把 `NewLagRecorder`、`NewOutboxDrainerLoop`、`NewVersionedKeyGenerator` 这些**我自己手工找到的**组件漏掉了——数量从 21 跳到 163。教训与仓库既有纪律一致：**检查器本身也要被检查**。

## 5. 这些不是 163 个缺陷（重要）

`prod == 0` 有多种合法解释，**逐条读代码之前不要当成缺陷**：

1. **对外 API**：本仓库之外可能有人调用（尤其 `pkg/infrastructure/*` 的部署/网关/发现类）。
2. **被更具体的变体取代**：如 `BuildMonolithicIndexingRuntime` 与 `BuildMonolithicIndexingRuntimeWithOptions` 并存，生产只用后者。
3. **泛型工厂/反射构造**：不经由构造器名。
4. **测试专用但命名不含 Mock/Test**：过滤只按名字与目录，覆盖不全。

反过来也有**假阴性**：注释里的名字已排除，但**行尾注释**与**字符串字面量**仍会计入（只会高估生产引用，即只会漏报，不会误报）。

## 6. 建议的三步用法

1. 先看**测试引用数最高**的那些——"测得越狠、生产零调用"越可疑（上表 12 个）。
2. 对每个候选，先判断它属于 §5 的哪一类；只有确属"应当被构造"的，才进入接线。
3. 接线时必须同时回答：**谁构造它、在哪启动、失败如何可见**。本仓库两次"假过"（F26 熔断器、F27 装饰器）都是"启动日志报成功、运行时才炸"——所以**新接线的生命周期路径必须在真实部署里跑一次真实流量**再报完成。

## 7. 与计划剩余项的关系

本审计把计划里散落的"未接线"项归到同一个根因下。计划 §6 的 T06/T10/T11/T13 剩余项**基本都能在本列表里找到对应的构造器**，说明它们的共同瓶颈不是"缺实现"，而是"缺装配与所有权"——这与 `2026-10-13-write-path-findings.md` §4 的栈选择（A/B/C）是同一个决策面。
