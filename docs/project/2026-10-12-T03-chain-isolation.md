# T03 交付：链隔离（I01）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T03 / P0 / 6h**。
> 状态：**In Progress** —— Postgres 侧的链作用域访问器与 A/B 隔离验收已完成；
> 缓存与 metadata 查询的链作用域、以及"旧未限定链方法仅允许明确单链上下文"的**强制**尚未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 缺陷（读代码确认）

`core.DatabasePlugin` 端口的多个方法**不带 chain key**：

| 方法 | 问题 |
| --- | --- |
| `GetBlock(ctx, blockNumber)` | 两链同高 ⇒ 返回任意一条 |
| `GetLatestBlock(ctx)` | 跨链取最大高度 |
| `GetEventsByBlockRange(ctx, from, to)` | **两链事件混在一起** |
| `MarkEventsAsReorged(ctx, from, to)` | **`UPDATE … WHERE block_number BETWEEN …`，无 chain 过滤** ⇒ 回滚 A 会把 B 的行也标成 reorged |
| `GetAllBlocks` / `GetAllEvents` | 全库不分链 |

这正是 I01 禁止的跨链泄漏。

## 2. 实现

`pkg/plugins/database/postgres_chain_scope.go`（新）—— **链作用域访问器**，多链代码必须使用：

| 方法 | 说明 |
| --- | --- |
| `GetEventsByBlockRangeForChain(chainID, from, to)` | `WHERE chain_id = $1 AND …` |
| `GetBlockForChain(chainID, blockNumber)` | 按链取块 |
| `GetLatestBlockForChain(chainID)` | 按链取最大高度 |
| `GetAllBlocksForChain(chainID)` | 按链列块 |
| `MarkEventsAsReorgedForChain(chainID, from, to)` | **按链**软删；返回受影响行数 |
| `GetCheckpointForChain(chainID)` | 按链读 checkpoint |

**空 chain key 一律拒绝**（`chainID is required`），绝不退化成"匹配所有链"。

## 3. 验收（真实 Postgres，2/2 PASS）

`TestPostgresChainIsolation` —— 两条链在**同一高度**、**同一 tx hash**、**同一 log index** 各放一条事件（刻意制造最易混淆的场景），然后只回滚 A：

| 断言 | 结果 |
| --- | --- |
| `MarkEventsAsReorgedForChain(A, …)` 只影响 **1** 行 | ✅ |
| A 的事件 `status = reorged` | ✅ |
| **B 的事件 `status = pending`**（未被误标） | ✅ |
| **B 的 rows/blocks/checkpoint 摘要与回滚前逐字一致**（用 T02 的 `CanonicalEvents`/`CanonicalBlocks` oracle 计算） | ✅ |
| 链作用域读**从不混链** | ✅ |
| **未限定链的 `GetEventsByBlockRange` 确实看到 2 条**（反向断言：证明作用域 API 的必要性） | ✅ |
| `GetBlockForChain` 两链分别返回各自 block hash | ✅ |
| `GetCheckpointForChain` 两链分别返回各自 checkpoint | ✅ |

`TestPostgresChainScopedAccessorsRejectEmptyChain` —— 6 个作用域访问器在空 chain key 下**全部报错**。

## 4. 未完成（T03 剩余）

- [ ] **缓存与 metadata 查询的链作用域**（`pkg/services/query`、cache key 生成器）。
- [ ] **"旧未限定链方法仅允许明确单链上下文"的强制**：目前只在文档与注释里声明；尚无机制（如运行时校验或 lint）阻止多链部署调用未限定方法。
- [ ] **重组/去重失效接口的链作用域逐层传入**（`core.IdempotencyInvalidator` 等）。
- [ ] `query` 路径改用作用域访问器（当前仍走 `eventStore` 的未限定方法）。
