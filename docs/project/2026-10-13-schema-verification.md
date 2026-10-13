# schema fresh / upgrade / rollback 验证（计划 §10 清单项，2026-10-13）

> 对应计划 §10："**schema fresh/upgrade/rollback兼容，DB/API/checkpoint/canonical摘要通过。**"
> 本轮在**本地真实 PostgreSQL** 上跑完整循环，并跑通 canonical digest 验收测试。**结论：schema 与 canonical 摘要两半均通过。**
> 原始证据：`build/acceptance/<sha>/schema/`（`schema-verify.log`、`canonical-digests.log`、`run.sh`）。

---

## 1. 方法

- **用一次性 scratch 库**：`chainpulse_schema_verify`，跑完**删除**。全程不触碰 `chainpulse_dev` / `chainpulse_t04` 等既有库（并行会话的容器在用的那个）。
- 迁移工具：仓库自带 `go run ./cmd/migrate/ -db <dsn> -path migrations {up|down|version}`（`cmd/migrate/main.go`，基于 golang-migrate）。
- 迁移集：`migrations/000001…000013`，共 **13 对**（26 个 SQL）。

## 2. 结果

### 2.1 FRESH（空库 → 全量迁移）

```
up exit=0
version: Current version: 13 (dirty: false)
```

建表 **10 张**（按名排序）：

```
api_keys  blockchain_events  blocks  chain_leases  dlq_events
indexing_state  reorg_jobs  reorg_outbox  schema_migrations  webhooks
```

关键表都在：`blockchain_events`（事件）、`indexing_state`（I04 的 checkpoint 读模型）、`chain_leases`（fenced lease）、`reorg_outbox`（T06 事务性 outbox）、`blocks`。

### 2.2 ROLLBACK（逐步回退）

- 逐步 `down` 直到无迁移可退；
- 回退后 `public` 模式剩 **1 张表**：`schema_migrations` —— 这是 golang-migrate 自己的版本记账表，**按设计保留**，属预期。
- ⇒ 13 个迁移的 `down` 脚本**全部可用**（没有"能 up 不能 down"的迁移）。

### 2.3 UPGRADE（回退后再升）

```
up exit=0
version: Current version: 13 (dirty: false)
tables after re-upgrade: 10
```

⇒ 回退到空后再升回，**表数与版本都回到 13 / 10 张**，且 `dirty: false`（没有半途失败的迁移）。

### 2.4 收尾

scratch 库已删除（`0 remaining`），未留残留。

## 3. canonical 摘要

计划同一条还要求 "canonical摘要通过"。本地跑：

```
go test -count=1 -tags=integration -run TestCanonical ./test/acceptance/
--- PASS: TestCanonicalDigestsAcrossRollbackAndRedelivery (0.54s)
ok  	github.com/rtcdance/chainpulse/test/acceptance	6.130s
```

`TestCanonicalDigestsAcrossRollbackAndRedelivery` **通过** —— 即"回滚 + 重投递之后 canonical 摘要仍然一致"，正是 ADR-008 要保护的性质。

## 4. 覆盖边界（如实说明）

| 计划措辞 | 本轮状态 |
| --- | --- |
| schema **fresh** | ✅ 空库全量迁移到 v13 |
| schema **upgrade** | ✅ 回退后再升回 v13，`dirty: false` |
| schema **rollback** | ✅ 13 个 down 全部可用，只剩记账表 |
| **canonical摘要** | ✅ 回滚+重投递摘要一致性测试通过 |
| **DB** | ✅ 上述均在真实 PostgreSQL 上执行 |
| **checkpoint** | 由 I04（并行会话）已闭环：迁移 000013 + `PostgresCheckpointStore` 接线，实测 `indexing_state` 有真实行 |
| **API** | 未在本轮覆盖 —— API 层的摘要/契约需要跑起 API（属 D5/D6 相邻的装配工作） |

## 5. 结论

**计划该条的 schema 与 canonical 两半：通过**，且证据可复现（脚本在 `build/acceptance/<sha>/schema/run.sh`，用一次性库，跑完自删）。
唯一未覆盖的是 "API" 那半，原因是 API 的装配本身尚未完成（见决策备忘录 D5/D6）。
