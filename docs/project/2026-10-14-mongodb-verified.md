# MongoDB：13 个测试只测了守卫条件，从没碰过真服务器

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 证据：`build/acceptance/<sha>/mongodb-real-server/{SUMMARY.txt,run-with-server.log}`
- 来源：**支持矩阵的 `unverified` 行**（把它当待办清单）

## 待办来自哪里

矩阵里这一行是 `unverified`：

| capability | status | 当时写的理由 |
| --- | --- | --- |
| MongoDB persistence | unverified | `go.mongodb.org/mongo-driver` 是直接依赖且有真实非测试用法；**没有 lane 起 MongoDB** |

而 README 的能力表把它当**已承诺的持久化后端**（"Persistence: PostgreSQL and **MongoDB** support"）。

## 真实规模：306 处引用，四个生产文件

```
pkg/services/query/mongodb_event_store.go   126
pkg/infrastructure/database/manager.go      111
pkg/services/query/mongodb_adapter.go        69
pkg/services/query/query_service.go          45
```

这是一条**很厚的生产路径**（索引创建、单条/批量写入、按 ID/链/合约/事件名/关联 ID 查询、
分页、按区块范围删除、reorg 标记、TTL 清理、健康检查）。

## 但 13 个测试只断言守卫条件

`pkg/services/query/mongodb_event_store_test.go` 有 13 个测试、全部通过。它们跑在**一个 nil mock** 上：

```go
func TestMongoDBEventStoreInitialize(t *testing.T) {
	dbManager := &mockDatabaseManager{mongoClient: nil, postgresDB: nil}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)
	err := store.Initialize(ctx)
	if err != nil {
		t.Logf("Initialize error (expected with mock): %v", err)   // ← 记一条日志，不失败
	}
}
```

其余断言的是："未初始化时应失败"、"nil event 时应失败"、"空批次应成功"。
**没有一条真的插入过一个文档，也没有一条读回过一个文档。**

⇒ `createIndexes`、插入路径、以及**每一个查询过滤器**，**从未在真实 MongoDB 上执行过**。
"有 13 个测试且全绿"与"这个后端能用"是**两个不同的主张**。

## 修复

**1. 新增 `TestMongoDBEventStoreAgainstRealServer`**（`pkg/services/query/mongodb_event_store_integration_test.go`，`-tags=integration`）。
按仓库既有约定用环境变量门控（`CHAINPULSE_RUN_MONGODB_INTEGRATION` + `CHAINPULSE_TEST_MONGODB_URI`），
用**独立集合名**（`events_itest_<纳秒>`）以免碰到真实数据。它做的是**真实往返**：

- `InsertEvent` 一条 → `GetEvent(id)` 读回，断言 `EventName` 与 `BlockNumber` **逐字段一致**
  （只断言"没报错"是不够的：静默不写入也会返回 nil）
- `InsertEventBatch` 两条 → `CountEvents` **精确 +2**
- `GetEventsByEventName("Transfer")` → **三条都被找到**（走索引过滤器，不只是点查）
- `Health()` → 断言 `status == "healthy"`（它内部真的 `Ping`，未初始化或 ping 失败都会返回 `"unhealthy"`，
  所以这个断言不是常量）

**2. 接进 `contracts` lane**：给该 job 加 `mongo:7` 服务（沿用 compose 里的 healthcheck 约定）
与两个环境变量。**这很关键**：该 job 用 `--fail-on-unknown-skip`，
所以如果 MongoDB 没被 provision，这个测试会 **skip 并让 job 变红**——不会静默跳过。

## 结果

```
# 对着运行中的 mongo:7（27017）
--- PASS: TestMongoDBEventStoreAgainstRealServer (0.39s)
    MongoDB round trip verified: 1 single + 2 batch inserts, count 3, 3 found by event name

# 门控关闭时（开发者机器没有 MongoDB）
--- SKIP: TestMongoDBEventStoreAgainstRealServer (0.00s)

# 跑完不留垃圾集合
db.getSiblingDB("chainpulse").getCollectionNames().filter(n => n.startsWith("events_itest_"))
→ []
```

矩阵该行 unverified → **verified**。8 个 workflow YAML 全部可解析。

## 顺带修掉的一处自己的错

第一版清理用的是 `defer manager.Close(...)` + `t.Cleanup(drop)`。
**defer 先于 Cleanup 执行** ⇒ 客户端已断开 ⇒ 集合**没被删掉**，
留下一个 `events_itest_1791943892914993000`（运行日志里那句
`warning: could not drop ...: client is disconnected` 就是它）。
已把 drop 与 close 都放进同一个 `t.Cleanup`、按正确顺序执行，
并**手工删掉了我自己留下的那个集合**；随后再跑两次，`[]`。

## 诚实边界

1. **只覆盖了写入、点查、按事件名查询、计数、健康检查。** 未覆盖：`createIndexes` 的**结果**（只间接跑过）、
   `GetEventsByChain`/`ByContract`/`ByCorrelationID`、`GetEventsPaginated` 的游标、
   `DeleteEventsByBlockRange`、`MarkEventsAsReorged`、`DeleteExpiredEvents` 的 TTL。
   这些仍是 unverified——**矩阵该行现在是 verified，但只对上面列出的那几项**。
2. **`wss://` 式的 TLS、鉴权、副本集都未测**：本地是单节点、无鉴权。
3. **workflow 未在 GitHub 上跑过**：验的是 YAML 可解析 + 那条命令在本机逐字执行。
4. **E01/E03 等 §9 场景仍未验收**，与本轮无关。
5. **矩阵仍靠人维护**，没有自动一致性检查——这一行的状态靠我这次手动改。

## 复现

```bash
# 需要一台 MongoDB（本机 chainpulse-mongodb 在 27017）
CHAINPULSE_RUN_MONGODB_INTEGRATION=1 \
CHAINPULSE_TEST_MONGODB_URI='mongodb://127.0.0.1:27017' \
  go test -count=1 -v -tags=integration -run TestMongoDBEventStoreAgainstRealServer ./pkg/services/query/

# 门控关闭时应 SKIP
go test -count=1 -v -tags=integration -run TestMongoDBEventStoreAgainstRealServer ./pkg/services/query/
```
