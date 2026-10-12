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

---

## 2. 未完成（T05 剩余）

- [ ] **接线**：把 `BeginTx` / `DrainController` 接进 shared runtime 与 processor 的提交路径（目前 processor 仍先查重再写再标记，`CheckAndMark` 只有测试调用 —— 见 F05）。
- [ ] **世代与 owner fencing**：rebalance 后旧 worker 的提交必须被拒（需要 DB 侧 generation/owner token）。
- [ ] **空块 / 全重复批次**：`ON CONFLICT DO NOTHING` 下 checkpoint 仍须推进（需测试固定）。
- [ ] **Kafka 消费进度与 ack 对齐**：`chainpulse-kafka` 已启动（9092 healthy），但消费/ack 路径尚未接线与验收。
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
