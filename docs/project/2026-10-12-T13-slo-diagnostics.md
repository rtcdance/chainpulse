# T13 交付：SLO 与诊断（标签基数 + 分相 lag）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T13 / P1 / 6h**。
> 状态：**In Progress** —— 标签基数守卫与采集/提交/finalized 三相 lag 已完成并验证；
> **`MetricLabelGuard` 已完成生产接线（2026-10-13，提交 270ff18 / ac76352）**；
> **`LagRecorder` / `IndexerMetrics` / `IndexerHealth` 仍未接线**，且经核查**在生产中根本没有调用方**（见 §4.1）；
> trace 跨 MQ 连续性、readiness/liveness 拆分、告警↔runbook 对应、故障注入下的指标变化未做。
> 日期：2026-10-12（§4.1 更新于 2026-10-13），Asia/Singapore。

---

## 1. 缺陷（读码确认）

`pkg/ports/metrics.go`：

```go
type MetricsCollector interface {
	RecordCounter(name string, value int64, tags map[string]string)
	...
}
```

`tags` 是**任意** `map[string]string` ⇒ **没有任何机制阻止调用方把 tx hash / 地址 / 块高当标签**。这类标签的取值**每个事件都不同**，等于**每个事件一条时间序列** —— 指标后端的经典宕机方式。计划把这一点写成了验收项："**标签无 tx hash/address/block height 无界基数**"。

另外 `IndexerMetrics` 只有一个 `IndexingLag` 字段：**采集 lag、提交 lag、finalized lag 混成一个数**。一条链可能"采集已追上但提交远远落后"，也可能"提交追上了但 finality 还差 12 个块"，一个数字无法区分，告警也就说不清是哪一级落后。

## 2. 实现

### 2.1 `MetricLabelGuard`（`pkg/observability/metric_labels.go`）

包装任意 `MetricsCollector`，让**每个标签集都有界**：

| 规则 | 行为 |
| --- | --- |
| 键在 `UnboundedLabelKeys` 中（`tx_hash`/`transaction_hash`/`address`/`contract_address`/`from_address`/`to_address`/`block_hash`/`block_number`/`block_height`/`event_id`/`correlation_id`/`request_id`/`session_id`/`client_id`/`search_query`/`query`） | 值的**替换**为 `__redacted__`（**保留键**，使该序列仍可与"从未带过此标签"的序列区分） |
| 其他键 | 全进程生命周期内**至多 `MaxSeriesPerKey`（默认 64）个不同取值**；超出者变为 `__overflow__` |
| 计数 | `Stats()` 暴露 `Redacted`/`Overflow`；`OnRedact` 回调让**原因可观测而非静默** |

**度量本身不受影响**：计数器照常自增，只是标签被收敛。这是刻意的取舍——**宁可丢标签，不可丢度量，更不可让后端倒下**。

### 2.2 `LagRecorder`（`pkg/observability/indexer_lag.go`）

`LagPhase` 三值，**各自独立序列**：

| 相位 | 含义 |
| --- | --- |
| `ingest` | 链头 − puller 已观测到的最高块 |
| `commit` | 已观测最高块 − **已持久提交**（checkpoint）的最高块 ⇒ **进程死掉会丢数据的那一段** |
| `finalized` | 链头 − **已 finalized** 的最高块；**恒 ≥ commit lag**，相等仅当链完全 final |

`RecordLagSnapshot` **拒绝不可能的组合**（commit > ingest、finalized < commit、空 chainID）：相位关系被违反意味着计数器彼此不一致，**静默发布会让看板说谎**。

标签只有 `chain_id`（受部署链列表约束，有界）。

## 3. 验收（单元 11 个，全 PASS）

| 测试 | 断言 |
| --- | --- |
| `TestLabelGuardRedactsUnboundedKeys` | 9 类无界键（tx hash / 地址 / 块高 / event_id / correlation_id …）**全部被 redact**，且 `Redacted == 9` |
| `TestLabelGuardKeepsBoundedLabels` | `chain_id`/`phase`/`status` **原样通过**，零 redaction（守卫不误伤） |
| `TestLabelGuardCapsSeriesPerKey` | 预算 3：第 4 个不同值**塌缩为 `__overflow__`**；**已见过的值仍放行**（预算不泄漏） |
| `TestLabelGuardReportsRedactions` | 回调报出 `键=原值->替代值` |
| `TestLabelGuardRecordsThroughToInner` | **标签被 redact 后度量仍被记录** |
| `TestLagPhasesAreDistinctSeries` | 三个相位**名字互不相同**，`commit` 的名字精确匹配 |
| `TestLagRecorderRecordsEveryPhase` | 用 `ExportPrometheus()` 断言**抓取端真能看到三条序列** |
| `TestLagRecorderRejectsImpossibleOrderings` | `commit > ingest`、`finalized < commit`、空 chainID **全部报错** |
| `TestLagRecorderAndGuardCompose` | lag recorder 接在守卫之后：`chain_id` **不被误 redact** |

> 写测试时我自己先写错了一次（`finalized=12 < commit=90`），**被不变式当场拒绝** —— 这正是该检查的价值。

## 4. 未完成（T13 剩余）

- [x] **`MetricLabelGuard` 接线（2026-10-13 完成）**：见 §4.2。
- [ ] **`LagRecorder` / `IndexerMetrics` / `IndexerHealth` 接线**：见 §4.1 —— 比"没接"更严重：**生产代码里没有任何调用方**。
- [ ] **trace 跨 MQ 连续**。
- [ ] **readiness 看依赖与处理可用性、liveness 看进程**（`indexer_health.go` 需按此拆分）。
- [ ] **每个告警对应 runbook**（`monitoring/prometheus`、`alerts.go` 需补齐映射）。
- [ ] **故障注入验收**：注入 RPC/DB/MQ 故障时指标与告警按预期出现、恢复后消退。

### 4.1 核查发现：lag / health 指标在生产中是**死代码**（2026-10-13）

上一版只写了"尚未接进启动路径"。实际核查（`grep -rn <sym> --include="*.go" pkg/ cmd/ | grep -v _test.go`）比这更糟：

| 符号 | 生产调用方 |
| --- | --- |
| `NewIndexerMetrics` | **无**（仅定义在 `indexer_metrics.go`） |
| `NewIndexerHealth` | **无**（仅定义在 `indexer_health.go`） |
| `NewLagRecorder` | **无**（仅定义在 `indexer_lag.go`） |
| `RecordIndexingProgress` | **仅测试**（`indexer_metrics_test.go`、`indexer_health_test.go`） |

**推论（可直接验证）**：

1. `IndexerMetrics` 从未被实例化 ⇒ `IndexingLag` 字段永远是零值；
2. `IndexerHealth.CheckHealth()` 读的正是该字段 ⇒ 即使被构造，也只会报 `indexing_lag = 0`；
3. `indexer_metrics.go:388` 的 `mc.RecordGauge("chainpulse_indexer_lag", …)` 因此**在生产中永不发出**；
4. 现行 `/health` 处理器（`pkg/plugins/api/health_check_handler.go`）**根本不含 lag 字段** —— 也就是说，**这条链路的"滞后"信号目前完全不存在**，不是"数值不准"。

**为什么不能顺手接上**：三相 lag 需要四个高度来源（链头、puller 已观测最高块、checkpoint、已 finalized 最高块），而运行时的 `EventSource` 接口**只有 `Fetch`**（`pkg/application/indexing/runtime.go:30`），没有任何链头来源；`CheckpointStore` 只提供 checkpoint。⇒ 接线的前提是**先引入一个链头来源**（接口 + 依赖注入），这是一个需要明确设计的改动，不是一行接线。

### 4.2 `MetricLabelGuard` 接线（已完成）

`observability.Guarded(inner)` 一行包装（nil inner 回退而非 panic），接到：

- `pkg/application/bootstrap/providers.go` 的 `provideMetrics()`（DI 路径）；
- 5 个入口点 —— `cmd/monolithic/chainpulse/main.go`、`cmd/microservices/{api-service,api-gateway,event-processor,puller}/main.go`（后 4 个新增 `pkg/observability` import）。

核对：`grep -rn "NewDefaultMetricsCollector()" --include="*.go" pkg/ cmd/ | grep -v _test.go` 现在只剩定义处与 `pkg/testhelpers/test_helpers.go`（测试助手，理应保持裸 collector）。⇒ **guard 的生产接线 100% 完成**，`tx_hash`/`address`/`block_height` 等无界标签在生产中会被 redact 而非每事件一条序列。

测试：`providers_test.go` 断言 provider 返回的确是 `*MetricLabelGuard` 且 `tx_hash` 被 redact；`metric_labels_test.go` 增 `TestGuardedWrapsAndRedacts`、`TestGuardedNilInnerFallsBack`。`go build ./...`、`go vet ./cmd/...`、`gofumpt`、`make test-short` 均通过。
