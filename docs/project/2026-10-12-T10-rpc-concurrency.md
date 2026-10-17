# T10 交付：provider 能力适配（F25，range 切分）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T10 / P1 / 12h**。
> 状态：**In Progress** —— F25 的 **eth_getLogs 范围切分/二分退避**已完成并用单测验证；
> goroutine 盘点、有界 worker/queue、429/5xx 分类与 Retry-After、WS 断连补采、provider chain/genesis 校验、以及**把切分器接进 puller** 未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 缺陷（读码确认）

`pkg/plugins/pullers/https_jsonrpc_puller.go` 的 chunk 循环（第 266–310 行）在检测到 provider 截断时：

```go
if len(logs) >= maxEventsPerChunk {
    // 把 chunkSize 减半
    // 然后「advance chunkFrom based on the last block in the results」
}
```

**问题**：截断后**按"返回结果里的最后一个块"前进**。provider 截断的那一段（`lastBlockInResults+1 .. chunkTo`）**从未被查询**就被跳过了 —— 违反验收里的"**失败范围不跳过**"。而 F25 指出的正是："现只覆盖 429/5xx/断连"，**provider 能力差异（range 上限 / 结果条数上限）未适配**。

## 2. 实现

`pkg/plugins/pullers/range_splitter.go`（新）：`RangeSplitter[T]`。

**核心不变量：不跳过任何块。** 递归访问的子区间是 `[from, to]` 的**精确划分**。

```
fetchRange(from, to):
    items, err := fetch(from, to)
    if err == nil: 记录并返回
    if !TooWideError(err): 直接上抛（传输错误不该靠切分重试）
    if width <= MinRange: 上抛「最小范围也拒绝」错误（不再无意义地切）
    mid = from + width/2 - 1
    fetchRange(from, mid); fetchRange(mid+1, to)     # 两半都查
```

**两类 provider 拒绝统一为一种信号**（F25 说的 geth/erigon/reth 响应差异）：

| 类型 | 构造 |
| --- | --- |
| JSON-RPC 报错（range 上限） | `&ErrRangeTooWide{...}` |
| **静默截断**（结果条数上限） | `TruncationError(from, to, maxResults)` |

**两道防失控**：`MinRange`（最小可服务范围，到达即上抛）+ `MaxCalls`（单次 `Fetch` 的调用上限）。`Stats()` 暴露 `Calls`/`Splits`/`LargestRange`（学到的 provider 上限，供观测与后续自适应）。

## 3. 验收（单元 10 个，全 PASS）

| 测试 | 场景 | 断言 |
| --- | --- | --- |
| `TestSplitterCoversEveryBlockWithLowRangeLimit` | provider 只接受 ≤10 块，查 `[1,100]` | **100 个块全部覆盖、升序、无缺口无重复**；确有切分；学到上限 ≤10 |
| `TestSplitterHandlesResultCountLimit` | provider 结果上限 5 条 | 同样 20 块全覆盖 |
| `TestSplitterSurfacesErrorAtMinimumRange` | 只接受 1 块但 `MinRange=4` | 报错含 "minimum range" |
| `TestSplitterBoundsCallsWhenProviderAlwaysRefuses` | 一律拒绝，`MaxCalls=5` | **调用数 ≤6** 且报错含 "exceeded"（**无失控重试**） |
| `TestSplitterFailsFastAtMinimumRangeWithoutCallGuard` | 一律拒绝，无调用上限 | 报错含 "minimum range"，**调用数 ≤32**（到宽度 1 约 20 次，而不是一百万次） |
| `TestSplitterDoesNotSplitOnOtherErrors` | 传输错误 | **只调用 1 次**（不靠切分重试） |
| `TestSplitterStopsOnContextCancel` | 已取消的 ctx | 返回 `context.Canceled` |
| `TestSplitterRejectsInvertedRange` / `TestSplitterRecordsLargestAcceptedRange` / `TestSplitterStatsResetPerFetch` | — | 输入校验、上限记录、统计只描述最近一次 |

## 4. 未完成（T10 剩余）

- [ ] **接进 puller**：`RangeSplitter` 已就绪，但 `https_jsonrpc_puller.go` 的 chunk 循环**仍在用"按结果末尾前进"的旧逻辑**；需改为调用切分器，并把截断转成 `TruncationError`。
- [ ] **goroutine 盘点**：拉取/消费/缓存/订阅的所有权与取消路径。
- [ ] **有界 worker / queue / batch**，timeout 与 retry budget。
- [ ] **429/5xx/无效请求分类**，jitter 与 `Retry-After`。
- [ ] **WS 断开后 HTTP 按连续范围补采**。
- [ ] **provider chain/genesis 校验**（防串链）。
- [ ] **race 无告警 + profile**（CPU/heap/goroutine）。


---

## 附：F25 provider 能力适配——**接线完成**（2026-10-13）

此前 `RangeSplitter` 已实现且有单测，但**没有接进 puller 的 chunk 循环** —— 生产路径仍走旧的"按结果末尾前进"。§10 清单里的"受限 provider（getLogs 范围/结果上限）场景验收通过（F25）"因此不成立。

### 病灶（读码确认，`https_jsonrpc_puller.go:339-345`）

```go
if lastBlockInResults > 0 {
    chunkFrom = lastBlockInResults + 1   // ← 丢块在这里
} else {
    chunkFrom = chunkTo + 1
}
```

结果按块升序排列，所以**只要页上限不能整除每块日志数，截断就落在某个块的中间**。旧逻辑把"页里最后一个块"当作"已完整处理"，下一个请求从 `lastBlockInResults+1` 开始 ⇒ **该块剩余的那些日志再也不会被请求**。

### 修复

- `PullEvents` 的每个 chunk 改走 `fetchLogsWithoutGaps` → `RangeSplitter`：provider 拒绝范围或返回整页时**拆分并重试**，而不是前进。
- 新增 `classifyGetLogsError`：把 geth/erigon/reth 各家"范围/结果过大"的措辞统一成 `ErrRangeTooWide`（11 条 hint）；**不匹配的错误保持致命** —— 拆分一个非法请求只会把失败放大。
- 整页（`len(logs) >= maxEventsPerChunk`）由 `getLogs` 的包装层转成 `TruncationError`；**绝不能把截断页带 nil error 返回**，否则拆分器会认为该范围已覆盖。
- chunk size 改为按**provider 实测接受过的最大范围**自适应（`SplitStats.LargestRange`），不再只靠"减半"猜测。

### 验收（真实 HTTP JSON-RPC 桩，`httptest.NewServer` + 真 `ethclient`）

| 测试 | provider 限制 | 断言 | 结果 |
| --- | --- | --- | --- |
| `TestPullEventsCoversEveryBlockWhenProviderRefusesWideRanges` | 范围 >20 块即报 JSON-RPC error（Alchemy/Infura 形态） | 200 块 1 日志 ⇒ **200 条全回**、每块恰好 1 条；并且**provider 确实拒绝过**（否则测试无意义） | PASS |
| `TestPullEventsCoversEveryBlockWhenProviderTruncatesSilently` | 结果上限 = 9000，静默整页截断 | 3000 块 × 7 日志 ⇒ **21000 条全回**、无块被跳过 | PASS |
| `TestTruncationLossIsRealNotHypothetical`（**反向对照**） | 同上 | 用**旧的前进逻辑**跑同一 provider 与同一载荷 | **20996 / 21000 —— 丢 4 条** |

**载荷为什么要 7 日志/块**：9000 ÷ 7 = 1285 整块 + 5 条，截断点落在块 1286 中间。若每块日志数整除 9000（如 3/块），截断正好落在块边界，**旧逻辑也能通过** —— 那样的测试证明不了任何事。对照实测丢 4 条（两处页边界各丢 2 条），与新逻辑的 0 条形成对照。

### 仍属 T10 剩余（未做）

- goroutine 盘点 / 有界 worker、queue、batch；
- 429/5xx 分类与 `Retry-After`（客户端侧已实现，见 T18 对 E08 的更正）；WS 断连后按连续范围 HTTP 补采；
- provider chain/genesis 校验的接线；
- **`maxEventsPerChunk` 阈值的固有盲区**：provider 若在**低于 9000** 时静默截断，响应里没有任何信号可据以判定 ⇒ 目前无法检测。这是已知限制，**不冒充已覆盖**。
