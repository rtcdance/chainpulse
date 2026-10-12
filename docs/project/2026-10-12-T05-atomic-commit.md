# T05 交付：提交与 checkpoint（原子提交）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T05 / P0 / 18h**。
> 状态：**In Progress**（事务提交能力与 F28 契约已完成并用真实 Postgres 验证；F24 优雅停机与 runtime/processor 接线未做）。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 已完成

### 1.1 事务提交能力（F05）—— 已实现并验证

**发现**：`pkg/ports/plugin.go` 里 `Transactioner` / `Tx` 端口**已声明但无任何实现**（`grep "func (.*) BeginTx("` 零命中）。因此"写事件"与"推进 checkpoint"只能是**分步提交**——这正是 F05 描述的缺口：两步之间崩溃会让 checkpoint **超前于数据**（事件永久丢失）或**落后于数据**（重启整批重放）。

**实现**：

- `pkg/ports/plugin.go`：`Tx` 端口扩展 `SaveCheckpoint(ctx, chainID, blockNumber, blockHash) error`，使"事件批次 + 其 checkpoint 推进"成为**同一个事务**。
- `pkg/plugins/database/postgres_tx.go`（新）：
  - `PostgreSQLPluginAdapter.BeginTx(ctx) (core.Tx, error)`；
  - `pgTx` 实现 `StoreEvent` / `SaveCheckpoint` / `Commit` / `Rollback`；
  - `StoreEvent` 复用 `insertEventSQL`（与 `WriteEvent` **同一份** SQL，杜绝两套写法漂移）；
  - `Rollback` **幂等**（`done` 标志 + 忽略 `sql.ErrTxDone`），支持 `defer tx.Rollback(ctx)` 惯用法；重复 `Commit` 报错而非静默再提交。
- 编译期断言：`var _ core.Transactioner = (*PostgreSQLPluginAdapter)(nil)`、`var _ core.Tx = (*pgTx)(nil)`。

**验收（真实 Postgres）**：

| 测试 | 断言 |
| --- | --- |
| `TestPostgresAtomicCommit` | `BeginTx → StoreEvent + SaveCheckpoint → Commit` ⇒ 事件与 checkpoint **都**可见 |
| `TestPostgresAtomicRollback` | `… → Rollback` ⇒ 事件与 checkpoint **都不可见**（**无半提交**） |
| `TestPostgresRollbackIsIdempotent` | Commit 后 `defer Rollback` 为无害 no-op；已提交数据存活；二次 `Commit` 报错 |

### 1.2 F28：冻结 Kafka 分区键 = chain key

`pkg/plugins/mq/partition_contract.go`（新）：

- `PartitionKeyForChain(chainID) string` —— 契约：**分区键就是 chain key**。kafka-go 按 record key 哈希选分区 ⇒ 同一链的所有消息落在同一分区、按 offset 有序消费（**链内有序不依赖分区数**）；不同链分散到不同分区、**并行消费**。
- `NewChainMessage(chainID, topic, payload)` —— 让发布方**无法**遗漏或臆造分区键。
- 部署参数（非消息内容）在注释中冻结：分区数是 topic 属性；**每条链的并发提交者上限 = 1**（DB generation/owner token 是第二道防线，不是第一道）。
- `partition_contract_test.go`：`TestPartitionKeyContract` 逐字冻结"键 == chain key / 同链同键 / 异链异键"；`TestNewChainMessageSetsPartitionKey` 保证构造即带键。

### 1.3 F24：SIGTERM 优雅停机 —— 已实现并验证（10 次）

`pkg/application/bootstrap/drain.go`（新）：`DrainController`。

- `Begin()` 在停机开始后**拒绝新批次**（返回 false），`End()` 配对；
- `Drain(timeout)` 翻转 draining 并等待 in-flight 批次完成，**超时如实上报**（超时意味着 checkpoint 可能落后、批次将被重放，不能隐藏）；
- 提供 `InFlight()` / `Rejected()` 便于断言。

**为什么必须这样**：`kill -9` 覆盖的是崩溃场景，而 Docker stop / 滚动部署走 **SIGTERM**。没有 drain 就会**死在批次中间**——要么 checkpoint 已推进而事件丢失，要么整批重放。

**验收（真实进程 + 真实 Postgres，10 次）**：`test/acceptance/tools/graceful` 是一个真实工作负载（每批 3 个事件 + 同事务推进 checkpoint，循环提交），`TestSigtermGracefulShutdown` 每次启动它、等它提交若干批、发 **SIGTERM**，然后断言：

1. 退出码 **0**（干净退出，不是被 kill）；
2. checkpoint 仍落在批次边界；
3. **事件数恰好等于 checkpoint 推算值** ⇒ **无半提交、无缺口**。

实测：`run_00` … `run_09` **10/10 PASS**。

> 过程中的一个自证：工作负载最初忘了调用 `tx.Commit`，结果 108 个批次全部回滚、库中 0 行——这同时验证了 `Rollback` 确实生效。

### 1.4 Owner fencing（世代 / owner token）—— 已实现并验证

**为什么需要**：Kafka consumer group 的 rebalance 会把分区在 worker 之间搬走，而**旧 owner 可能正处在批次中间**。没有 fencing token，它会继续提交，导致**一条链有两个写入者**。

- **迁移 `000011`**：`chain_leases(chain_id PK, owner_token, generation, acquired_at, expires_at)`。
- **`postgres_lease.go`（新）**：
  - `AcquireLease(ctx, chainID, ownerToken, ttl)` —— upsert，且**带条件**：只有"已过期"或"还是自己"才允许接管，每次获取 `generation + 1`。竞争者拿不到行 ⇒ `ErrLeaseLost`。**两条链不会同时以为自己是 owner。**
  - `ReleaseLease` / `CurrentLease`；`ReleaseLease` 只删自己的租约（非 owner 调用是 no-op）。
  - **`BeginFencedTx(ctx, chainID, ownerToken)`** —— 用 `SELECT … FOR UPDATE` 在事务内校验并**锁定租约行**，锁持续到事务结束。这让语义精确可用：
    - **在途批次允许完成**（不产生半提交）；
    - **接管之后才开始的批次被拒**（`ErrLeaseLost`）。
- `pgTx` 增加 `chainID / ownerToken / generation` 字段记录该事务开启时所属的 owner 与世代。

**验收（真实 Postgres）**：

| 测试 | 断言 |
| --- | --- |
| `TestPostgresLeaseTakeoverRejectsOldOwner` | A 持租约 → 过期 → B 接管（generation 1→2）⇒ **A 的新事务被拒**、B 正常提交 |
| `TestPostgresLeaseCannotBeStolenWhileLive` | 租约未过期时 B **抢不走**；同 owner 续租幂等且 generation 递增 |
| `TestPostgresLeaseReleaseAllowsHandover` | 优雅释放后立即可交接；**非 owner 释放是 no-op**（B 的租约不被 A 误删） |

**顺带修复**：迁移 `000004` 改为先 `DROP CONSTRAINT IF EXISTS` 再 `ADD`，使其**可重复执行**（此前二次运行会失败）。

### 1.5 共享提交契约 `BatchCommitter` —— 已实现并验证

**为什么**：此前"写事件"与"推进 checkpoint"是**两步**，且 processor 里 `CheckAndMark` 是**内存 map**、只有测试调用。计划要求的"同一提交契约"并不存在。

- **`pkg/services/processor/batch_committer.go`（新）**：`BatchCommitter.CommitBlock(ctx, blockNumber, blockHash, events)` —— **一个 fenced 事务**内完成"写事件 + 推进 checkpoint"。
  - `FencedStore` 接口（`AcquireLease` / `ReleaseLease` / `BeginFencedTx`）由 `PostgreSQLPluginAdapter` 直接满足；
  - `WorkGate` 接口（`Begin`/`End`）由 `bootstrap.DrainController` 满足 —— **不引入 services→application 的反向依赖**；
  - `EnsureLease` **惰性续租**（过半 TTL 才续），避免每批churn 租约。
- **三条性质**：① **原子**（事件与 checkpoint 同生共死）② **fenced**（接管后的事务被拒）③ **可 drain**（gate 关闭后不再开新批，在途批允许完成）。
- **不做重复预检**：去重是数据库自然键 + `ON CONFLICT DO NOTHING` 的职责。预检会**重新引入 check-then-write 窗口**（正是 F05 的病灶）。

**验收（真实 Postgres）** `TestPostgresBatchCommitter`：

| 场景 | 断言 |
| --- | --- |
| 普通块 | 事件落库 + checkpoint 前进 |
| **空块** | 0 事件，**checkpoint 仍前进**（否则该高度永久卡死） |
| **全重复块** | 事件数不变，**checkpoint 仍前进** |
| 正在 drain | 新批被拒（`ErrDraining`），**checkpoint 不动** |
| 被 fence 掉的 worker | 提交失败，**checkpoint 不动** |

单元测试 10 个（fake store）+ 集成 1 个，全 PASS。

### 1.6 接进 processor：`ProcessBlock` —— 已实现并验证

`DefaultEventProcessor` 原本只有**逐事件**路径：`ProcessEvent` 做"生成 hash → **查重** → 写 → 标记"，且 `CheckAndMark` 是内存 map。这正是 F05 的"分步提交"。

- **`BlockCommitter` 接口**（`CommitBlock`）+ `WithCommitter(...)`：processor 依赖**提交契约**而非具体数据库。
- **`ProcessBlock(ctx, blockNumber, blockHash, events)`**：整块走**一个事务**。
  - **先校验再开事务**：非法事件让整块失败且**不碰数据库**；
  - **空块合法**（提交它正是"推进 checkpoint 越过该高度"的手段）；
  - **不碰 idempotency service**：去重交给数据库自然键，从而**消除 check-then-write 窗口**；
  - fenced + drain-aware（继承 `BatchCommitter`）。

**验收**（单元 7 个，全 PASS）：无 committer 报错 / 未 running 报错 / 正常委派并计数 / **空块合法** / **非法事件不进 committer** / 提交失败如实计数且不计入 processed / **同一块提交两次会调用 committer 两次**（证明**没有**预检）。

### 1.7 接进 shared runtime：`ProcessBatchAtomic` —— 已实现并验证

`SharedRuntime.ProcessBatch` 的注释本身就写着 **"dedupe -> persist -> mark processed -> checkpoint"** —— **四步独立提交**，正是 F05。

- **`AtomicBatchCommitter` 接口**（`CommitBlock`）+ `RuntimeDeps.Committer`（**可选**，不设则只有四步路径）。
- **`ProcessBatchAtomic(ctx, chainID, events)`**：
  - **先全部转换校验**（`Payload` 必须是 `*blockchain.BlockchainEvent`）——任何非法信封都会**在提交前**失败整批；
  - **按块分组、升序提交**，失败**停在块边界**：checkpoint 仍指向上一个**完整提交**的块 ⇒ **无缺口、无半提交**；
  - **不碰 idempotency store**（去重交给 DB 自然键）。

**验收**（单元 7 个，全 PASS）：无 committer 报错 / **按块分组**（3 事件 2 块 ⇒ 2 次提交、块序正确）/ **非法 payload 不进 committer** / 链不匹配报错 / 空批 no-op / **块 11 失败时 checkpoint 停在 `block:10` 且只计 1 个 processed** / **同一块提交两次 ⇒ committer 调用两次**（证明无预检）。

---

## 2. 未完成（T05 剩余）

- [ ] **接入具体入口**：`cmd/microservices/event-processor` 与 monolithic 启动路径**注入** `Committer`（本轮交付的是 runtime 的原子路径与注入点；`RuntimeDeps.Committer` 目前无人设置）。
- [ ] **Kafka 消费进度与 ack 对齐**：`chainpulse-kafka` 已启动（9092 healthy），消费/ack 路径尚未接线与验收。
- [ ] commit 前后 / ack 前后**四处 kill/restart 各 10 次**（§5 验收原文；目前只覆盖了 SIGTERM 一处）。

---

## 3. 复现

```bash
# 需要一个可达的 Postgres（本机 5432 被原生 postgres 遮蔽，故用 socat 转发 + 迁移过的库）
docker run -d --name cp-pg-fwd --network docker_chainpulse-network -p 15433:5432 \
  alpine/socat tcp-listen:5432,fork,reuseaddr tcp:chainpulse-postgres:5432

export CHAINPULSE_RUN_POSTGRES_INTEGRATION=1
export CHAINPULSE_TEST_DATABASE_URL='postgres://chainpulse:chainpulse_dev@127.0.0.1:15433/chainpulse_t04?sslmode=disable'

go test -tags=integration -count=1 -run 'TestPostgresAtomic|TestPostgresRollback' ./test/contracts/...
go test -race -count=1 ./pkg/plugins/mq/ -run 'TestPartitionKeyContract|TestNewChainMessage'
```
