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
