# T06 交付：重组闭环（事务性 outbox + 持久修复任务）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T06 / P0 / 12h**。
> 状态：**In Progress** —— 事务性 outbox 与持久修复任务已完成并用真实 Postgres 验证；
> 与 `pkg/services/reorg` 调用方的接线、generation 失效缓存、深度 1/2/12 与"修复中崩溃"的端到端演练未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 已完成：事务性 outbox + 持久修复任务（F07）

### 1.1 缺陷（实测确认）

`pkg/services/reorg/reorg_handler.go` 的 `HandleReorg`：

1. 回滚事件（`rollbackWithSnapshots`）**提交**；
2. 再 `eventBus.Publish(...)` 发 `reorg.detected` 与 `reorg.rollback`；
3. **发布失败只 `logger.Error`**（第 276、286 行），既不重试也无持久记录。

⇒ 崩溃窗口存在：**回滚已完成、但没有任何人被告知**；且被中断的重组**没有任何记录可续跑**。这正是 F07。

### 1.2 实现

**迁移 `000012_reorg_outbox`**：

| 表 | 作用 |
| --- | --- |
| `reorg_jobs` | 修复工作本身（`chain_id`/`generation`/`from_block`/`to_block`/`status`）⇒ **中断的重组可被重新发现并续跑** |
| `reorg_outbox` | 该任务**应发**的通知（`job_id`/`topic`/`payload`/`attempts`/`published_at`），**与任务同一事务写入** |

`pkg/plugins/database/postgres_reorg_outbox.go`：

- **`EnqueueReorgJob(ctx, job, notifications)`** —— 任务与通知**一个事务**写入。因此可能的状态只有"任务已记录、通知待发"（drainer 会重试）或"两者都已提交"，**永远不会"回滚完成但通知丢失"**。按 `job_id` **幂等**（重复入队不重复）。
- `PendingReorgJobs(chainID, limit)` —— 未完成任务，供**重启后续跑**。
- `PendingReorgNotifications(limit)` / `MarkReorgNotificationPublished(id)` / `RecordReorgNotificationFailure(id)` —— outbox drainer。标记**幂等**（"已发布但崩溃在标记前"可安全重放）；`attempts` 让毒消息**可见**而不是无限重试。
- 通知**携带 `chain_id` 与 `generation`**，消费者无需二次查询即可识别"来自已被取代的 generation"的过期通知。

### 1.3 验收（真实 Postgres，4/4 PASS）

| 测试 | 断言 |
| --- | --- |
| `TestPostgresReorgOutboxIsAtomic` | 任务可发现、2 条通知待发、drain 一条后剩一条、完成任务后移出续跑集 |
| `TestPostgresReorgOutboxEnqueueIsIdempotent` | 重复入队 ⇒ 仍 1 个任务、2 条通知（不重复） |
| `TestPostgresReorgOutboxSurvivesRestart` | 进程 A 记录后退出 ⇒ 进程 B **发现未完成任务与未发通知**并 drain 完 |
| `TestPostgresReorgOutboxMarkIsIdempotent` | 重复标记为 no-op；失败计数可见 |

迁移实测：全新库 **12/12** 通过。

### 1.4 接进 `HandleReorg`（本轮完成）

- **`pkg/services/reorg/reorg_job.go`（新）**：`Job` / `Notification` 类型 + **`JobStore` 端口**（`EnqueueJob` / `PendingJobs` / `CompleteJob`）+ `OutboxDrainer` 端口 + `ReorgJobID(chainID, from, to)`（**确定性**，重试同一重组不会产生第二个任务）。
- **`ReorgHandler.WithJobStore(store)`**；`HandleReorg` 改为：
  - **有 store** ⇒ `EnqueueJob` 把任务与两条通知（`reorg.detected`、`reorg.rollback`）**一个事务**写入；失败**返回错误**而不是吞掉（回滚已提交，吞掉就正好是 F07 那个状态）；
  - **无 store** ⇒ 保留旧的 best-effort publish（**向后兼容**，注释明确标注"NOT crash-safe"）。
- **`pkg/application/bootstrap/reorg_job_store.go`（新）**：`PostgresReorgJobStore` 把 Postgres outbox 适配到 `reorg.JobStore`。放在**组合层**，使 reorg 服务不依赖数据库插件、插件也不依赖服务。

**验收（单元 5 个，全 PASS）**：任务+2 通知一起入队（含稳定 job id、block range）/ **入队失败被上报** / **有 store 时旧 publish 路径不运行**（优先级钉住）/ **无 store 时旧路径仍工作**（向后兼容）/ job id 稳定且按链隔离。

### 1.5 outbox drainer（本轮完成）

`pkg/services/reorg/outbox_drainer.go`（新）：`OutboxDrainerLoop`。

- **投递语义是 at-least-once**：**发布成功之后**才 `MarkPublished`。因此"发布成功但标记前崩溃"会**重投**——这是**刻意**的：反过来（先标记）会把崩溃变成**通知丢失**，正是 F07 要消除的状态。消费者按 `(job id, generation)` 去重。
- `DrainOnce(ctx)` —— 取一批 → 逐条 publish → 成功则标记。**单条毒消息不阻塞其余**（失败记 `RecordFailure` 后继续，返回首个错误供告警）。
- `Run(ctx)` —— 定时 drain，`ctx` 取消返回 `ctx.Err()`、`Stop()` 返回 `nil`（**优雅停机不算失败**）。
- 组合层适配器：`PostgresReorgJobStore` 同时实现 `reorg.JobStore` **与** `reorg.OutboxDrainer`；新增 `EventBusPublisher` 把 `core.EventBus` 适配成 `reorg.NotificationPublisher`。

**验收（单元 7 个，全 PASS）**：发布并标记 / **毒消息不阻塞 + 记为失败 + 仍待发** / **at-least-once 排序**（标记失败 ⇒ 已发布但**仍待发**）/ 空 outbox no-op / 批大小生效 / `Run` 在 `Stop` 时返回 nil / `Run` 在 ctx 取消时返回 `Canceled`。

### 1.6 修复执行与 generation 失效（本轮完成）

`pkg/services/reorg/reorg_repair.go`（新）：`Repair`。

**步骤顺序是被设计过的**（`RepairJob`）：

1. **回滚事件**（`RollbackEvents`）—— 先让数据自洽；
2. **回退 checkpoint** 到 `FromBlock - 1` —— **第一个需要重新索引的块就是 FromBlock**；不后退就会**静默跳过**被重组的区间；
3. **失效派生状态**（`InvalidateRange(chainID, generation, from, to)`）—— 派生数据可从上面两者重建，所以放最后；
4. **标记 job 完成**。

**可续跑**：只有全部步骤成功才 `CompleteJob` ⇒ 任一步失败 job **保持 pending**，下次 `Resume` 会重做。重做是安全的，因为每一步都是**区间作用域幂等**操作。

**generation 是 fence**：`InvalidateRange` 显式接收 generation ⇒ "低 generation 缓存的派生状态不得被服务"这一语义有落点（`Job.Generation` 已就位）。

**真实 checkpoint 回退**：`PostgreSQLPluginAdapter.RewindCheckpoint(chainID, toBlock)` —— `UPDATE indexing_state SET last_indexed_block = $2, last_indexed_hash = (SELECT hash FROM blocks WHERE chain_id=$1 AND number=$2)`；目标块仍在 `blocks` 中则恢复其哈希，否则置空由重新索引填充。

**验收**：

| 测试 | 断言 |
| --- | --- |
| `TestRepairJobRunsStepsInOrder` | 调用顺序恰为 `events:100-110` → `checkpoint:ethereum:99` → `derived:ethereum:gen4:100-110` |
| `TestRepairJobPassesGenerationToInvalidation` | generation=4 与区间原样传给失效器 |
| `TestRepairJobSkipsDoneJob` | 已完成的 job 不再应用 |
| `TestRepairJobLeavesJobPendingOnFailure` | 四步**各自**失败时 job 都**不**被标记完成（可续跑） |
| `TestRepairJobRejectsInvertedRange` / `RejectsMissingChain` | 非法输入不开事务 |
| `TestRepairResumeRepairsEveryPendingJob` | 两个 pending 都被修复 |
| `TestRepairResumeStopsOnFailure` | 失败即停，**没有任何 job 被误标完成** |
| `TestPostgresRewindCheckpoint`（真实 Postgres） | checkpoint 110 → **99**，且哈希从 `blocks` 恢复 |

---

## 2. 未完成（T06 剩余）

- [ ] **启动 drainer / repair 循环**：`OutboxDrainerLoop` 与 `Repair` 均已就绪，但**没有入口把它们 `Run` 起来**（需接进 cmd / runtime 启停）。
- [ ] **generation 的生产者**：`Job.Generation` 目前**无人填充**（reorg 路径尚未从 `AcquireLease` 的 generation 取值）；`DerivedStateInvalidator` 也尚无真实实现（Redis 缓存失效）。
- [ ] **深度 1 / 2 / 12 端到端**：结果需与 `test/acceptance/tools/digest` 的 `CanonicalEvents`/`CanonicalBlocks` 摘要一致。
- [ ] **重放重复 / 修复中崩溃 / outbox 失败后重启** 的组合演练。
- [ ] **超深度 / 祖先未知时停止提交**（现有 `maxRollback` 只做拒绝，未与 checkpoint 回退联动）。

---

## 3. 复现

```bash
export CHAINPULSE_RUN_POSTGRES_INTEGRATION=1
export CHAINPULSE_TEST_DATABASE_URL='postgres://chainpulse:chainpulse_dev@127.0.0.1:15433/chainpulse_t04?sslmode=disable'
go test -tags=integration -count=1 -run 'TestPostgresReorgOutbox' ./test/contracts/
```
