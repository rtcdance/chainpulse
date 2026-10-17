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

## 3.5 强制机制：未限定方法在多链下**拒绝**（本轮完成）

此前"未限定链方法仅允许明确单链上下文"**只在注释里声明**，没有任何机制阻止多链部署调用它们。现在有了：

- **`PostgreSQLPluginAdapter.WithChainScope(chainIDs...)`** —— 调用方**声明**本适配器服务哪些链（空/空白 id 被丢弃）。
- **`requireSingleChain(method)`** —— 声明 **>1 条链**时，未限定方法**直接报错**，错误信息**点名替代方法**：`"GetEventsByBlockRange is not chain-scoped and this adapter serves [ethereum polygon]; use the ...ForChain variant (invariant I01)"`。
- **未声明 scope ⇒ 保持旧行为**（单链部署零破坏）；**声明 1 条链 ⇒ 显式单链上下文，放行**。
- **实现位置**：守卫放在**适配器**（端口表面）上，用**覆盖方法**（`GetEventsByBlockRange`/`GetAllEvents`/`GetAllBlocks`/`GetBlock`/`GetLatestBlock`/`MarkEventsAsReorged`/`DeleteEventsByBlockRange`）委托给 `*PostgreSQLDatabase`。原始类型保持原行为，供直接使用者。

**验收**（`TestPostgresChainScopeEnforcement`，真实 Postgres）：

| 场景 | 结果 |
| --- | --- |
| 未声明 scope | 未限定方法**可用**（legacy） |
| 声明 1 条链 | 未限定方法**可用**（显式单链） |
| 声明 2 条链 | **7 个未限定方法全部报错**，且错误点名 `ForChain` 替代 |
| 声明 2 条链 | **作用域 API 仍可用** |
| 只传空白 id | scope 视为空 ⇒ 不误判为多链 |

---

## 4. 未完成（T03 剩余）

- [ ] **缓存与 metadata 查询的链作用域**（`pkg/services/query`、cache key 生成器）。
- [ ] **重组 / 去重失效接口的链作用域逐层传入**（`core.IdempotencyInvalidator` 等）。
- [ ] **`query` 路径改用作用域访问器**（当前仍走 `eventStore` 的未限定方法）。
- [ ] **`WithChainScope` 的接线**：机制已就绪，但**启动路径尚未按配置声明链集合**（未声明时守卫不生效）。
