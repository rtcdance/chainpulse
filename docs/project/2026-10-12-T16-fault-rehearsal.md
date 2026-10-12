# T16 交付：故障演练（§9 E09 DB 故障，真实执行）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T16 / P1 / 6h**。
> 状态：**In Progress** —— §9 **E09 的 DB 故障**已真实执行并通过；
> MQ/RPC/Redis 故障、kill 矩阵、版本回滚与数据恢复未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 为什么这一条能真跑

计划要求"**§9 故障矩阵全部有运行证据**"。本环境的能力边界（实测）：

| 依赖 | 可达性 | 原因 |
| --- | --- | --- |
| Postgres / Redis / Kafka | ✅ **裸 TCP 可用** | 沙箱放行裸 TCP |
| Anvil / 外部测试网 RPC | ❌ **HTTP 被中介** | `cast`/`curl`/Go `net/http` 到 `8545` 全部被 reset |

所以 **E09（DB/MQ/Redis 故障）与 E05（提交崩溃）可以真跑**，而 E01/E03/E13（需要 Anvil）不能。本轮先把 **DB 故障**做成真实证据。

**注入方式**：不碰数据库本身，而是 `docker stop` 掉 **socat 转发容器**（`cp-pg-fwd`），让宿主到 Postgres 的连接被拒绝。这既是真实故障，也**不破坏数据**，恢复只需 `docker start`。

## 2. 实现

`test/acceptance/postgres_fault_test.go`（新）：`TestDatabaseDisconnectIsBackpressure`。

序列（**顺序本身是被设计过的**）：

1. 健康态提交块 0..2 ⇒ checkpoint = 2、事件 3 条；
2. `docker stop` 转发容器 ⇒ 数据库不可达；
3. 提交块 3 ⇒ **必须失败**（背压，而不是丢数据）；
4. **先恢复数据库**再断言 —— 在故障中断言"块 3 没留下东西"只能证明"故障是真的"，证明不了"没有半提交"；
5. 恢复后：checkpoint **仍是 2**、事件**仍是 3 条** ⇒ **无半提交**；
6. 用一个**全新的 adapter**（模拟进程重启）从 checkpoint 续跑块 3 ⇒ checkpoint = 3、事件 4 条 ⇒ **无漏项、无重复**。

## 3. 验收（真实执行）

```
$ CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 \
  CHAINPULSE_TEST_DATABASE_URL='postgres://chainpulse:chainpulse_dev@127.0.0.1:15433/chainpulse_t04?sslmode=disable' \
  CHAINPULSE_PG_FORWARDER=cp-pg-fwd \
  go test -tags=integration -count=1 -run TestDatabaseDisconnectIsBackpressure -v ./test/acceptance/

--- PASS: TestDatabaseDisconnectIsBackpressure (0.73s)
```

断言逐条：

| 断言 | 计划原文对应 |
| --- | --- |
| 数据库不可达时批次**失败**而非静默成功 | "DB/MQ暂停有背压" |
| 恢复后 checkpoint 仍为 2、事件仍为 3 | "**无漏项、无重复效果**；checkpoint 与 durable effects 一致" |
| 新 adapter 从 checkpoint 续跑成功、事件恰为 4 | 同上（续跑不跳过、不重复） |

测试在 `t.Cleanup` 里**无条件恢复转发容器**，断言失败也不会把环境留在故障态。

## 4. 未完成（T16 剩余）

- [ ] **E09 的 MQ 部分**：Kafka 重投与 rebalance（Kafka 已起，9092 可用；需先接消费路径）。
- [ ] **E09 的 Redis 部分**：Redis 停止后的降级查询（Redis 6379 可用）。
- [ ] **E05 的 kill 矩阵**：persist / DB commit / mark / checkpoint / ack **四处** kill 各 10 次（目前只有 SIGTERM 一处，见 T05）。
- [ ] **E06 修复任务崩溃**：outbox/重采中停止 ⇒ restart 续跑（持久任务与 outbox 已就绪，缺编排）。
- [ ] **E15 版本回滚**：错误镜像/配置/迁移中断，前版 digest 回滚，旧版恢复后数据摘要一致。
- [ ] **RPO/RTO 报告**：按本地受控范围给出，**不冒充跨机房保证**（计划原文）。
