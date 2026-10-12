# T02 交付：真实适配器与 schema

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T02 / P0 / 14h**。
> 状态：**In Progress**（本地可验证部分完成；Postgres/Redis/Kafka 契约 runner 受阻于适配器不满足端口，见 §3）。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 已完成

### 1.1 canonical digest oracle（F21）—— 已实现并测试

落点：`test/acceptance/tools/digest/`（Go 包 + 测试）。

按 **I02/I03** 定义规范形式：

| 不变量 | oracle 中的编码 |
| --- | --- |
| I02 physical event ID = (chain key, block_hash, tx_hash, log_index) | `Event.ID()`；哈希统一小写 |
| I03 每链每高度至多一个 canonical 块 | `CanonicalBlocks()`：同 (chain,height) 异哈希 → **报错** |
| I01 chain key 必须显式 | 空 chain key → **报错** |

保证：
- **顺序无关**：stable sort，乱序输入摘要一致；
- **幂等**：同一 ID 重复投递 100 次摘要不变（`duplicate effect = 0`）；
- **严格**：同 ID 不同 payload → 报错，不静默取舍；
- **无歧义**：字段长度前缀编码（避免 `"ab"+"c"` 与 `"a"+"bc"` 碰撞）。

测试：`test/acceptance/tools/digest/digest_test.go`，**14 个用例 `-race` 全过**。

### 1.2 契约 suite 真正执行（F02）—— 已修复

`test/contracts` 此前只有 helper 函数（`DatabaseContractTest`/`CacheContractTest`），**没有任何 `Test...`**，`go test` 报 `[no tests to run]` 并 **exit 0** —— 即 T01 消除的那个假绿来源。

新增 `test/contracts/runners_test.go`，绑定真实工厂并驱动插件生命周期（`Initialize` + `Start` + `t.Cleanup(Stop)`）：

| Runner | 绑定实现 |
| --- | --- |
| `TestMonolithicMemoryDatabaseContract` | `bootstrap.NewMonolithicMemoryDatabase` |
| `TestMonolithicMemoryCacheContract` | `bootstrap.NewMonolithicMemoryCache` |
| `TestInMemoryCacheContract` | `cache.NewDefaultInMemoryCachePlugin` |
| `TestPortConformanceGaps` | tripwire（见 §3） |

**实测**：`go test -tags=integration -race ./test/contracts/...` → **exit 0**；经检验器：`pass=21 fail=0 skip=0 top_level=4`。

### 1.3 契约暴露的真实缺陷（已修）

跑起来后立刻发现两处实现不合契约（这正是"契约测试不能被 mock 遮蔽"的价值）：

**A. `MonolithicMemoryCache` 忽略 TTL、无统计**（`CachePlugin` 端口显式要求 `Set(..., ttl int)` 与 `GetStats()`）：
- `Get` 不过期 → `ttl_expiration` 失败；
- `GetStats()` 恒返回零值 → `stats_tracking` 失败。
→ 已实现到期表 + hit/miss 计数（`pkg/application/bootstrap/monolithic_memory_storage.go`）。

**B. `GetEvent` 缺失语义未定义，导致孤儿检测失效**：
- 契约断言 missing → error，但适配器层（`event_retrieval_service.go:91`、`graphql_handler.go:148`）一致地按 `event == nil` 分支 ⇒ **约定是 `(nil, nil)`**；
- `consistency_checker.go:180` 却用 `if err != nil` 判孤儿 ⇒ **恒为 0，静默失效**（真 bug）。
→ 已：`GetEvent` 明确 `(nil, nil)` 约定并补注释；**修 consistency checker 为 `err != nil || event == nil`**；契约测试对齐该约定。

### 1.4 Foundry 资产可编译（F15）—— 已修复

- `contracts/lib/forge-std` **vendored 到 v1.9.6**（`package.json` 佐证），安装命令与版本记录在 §5；
- `contracts/test/EventEmitter.t.sol` 原本**无法编译**（`emit EventEmitter.Transfer(...)` —— Solidity 不允许用限定名 emit 别的合约的事件）→ 改为在测试合约内**本地重声明**同签名事件配合 `vm.expectEmit`；
- **实测**：`forge build` → exit 0；`forge test` → **5 passed / 0 failed / 0 skipped**。

---

## 2. 证据（原始输出摘要）

| 命令 | 结果 |
| --- | --- |
| `go test -race ./test/acceptance/tools/digest/...` | ok，14 用例通过 |
| `go test -json -tags=integration -race ./test/contracts/...` + 检验器 | exit 0；`pass=21 fail=0 skip=0 top_level=4` |
| `forge build` | exit 0 |
| `forge test` | 5 passed / 0 failed |
| `go build ./...` / `make check` / `make lint` | 0 / 0 / **0 issues** |
| `make test-short` | exit 0（连跑 2 次，0 FAIL） |

> 说明：`make test-short` 曾偶发 `pkg/plugins/api TestDifferentExpirationTimes` 失败。**隔离复跑 3/3 通过**，根因是该测试用 `1 * time.Second` 过期 + JWT 秒级 `exp`，在满载并行下会踩边界 —— **既有 flaky 测试**，与本轮改动无关，未修改。

---

## 3. 阻塞 / 未完成（T02 未 Verified）

**端口一致性缺口（重要发现）**：生产适配器**不满足**系统依赖的端口接口：

| 适配器 | 缺口 |
| --- | --- |
| `database.DefaultInMemoryDatabasePlugin` | **缺 `BatchStoreEvents`** |
| `database.PostgreSQLDatabase` | 签名不符：`WriteEvent(ctx,*Event)` / `GetEventByHash(hash)` vs 端口要求 `StoreEvent(ctx,any)` / `GetEvent(ctx,id)` |
| `cache.RedisCachePlugin` | 签名不符：`Get(key) (*CacheEntry,error)` / `Set(*CacheEntry)` vs 端口要求 `Get(ctx,key) ([]byte,error)` / `Set(ctx,key,value,ttl)` |

后果：契约 suite **只能绑定 monolithic in-memory 实现**；§7.3 要求的"每个必需生产 factory executed>0"**对 Postgres/Redis 无法达成**。已用 `TestPortConformanceGaps` 做成 **tripwire**：任一套接器开始合规时该测试立即失败，强制同批补 runner。

未完成项：
- [ ] Postgres / Redis 契约 runner（**受阻于上述端口缺口**，需先做适配层或对齐签名）
- [ ] Kafka 契约（publish / consume / ack / redelivery）
- [ ] 迁移 fresh / upgrade runner（需真实 Postgres；从空库跑迁移并读写）
- [ ] `manager`/`checkpoint_store` 具体 driver 核对（`pkg/infrastructure/database/checkpoint_store.go` 的结构性类型断言）

---

### 3.5 🔴 P0：Postgres schema 与代码「三方分裂」（实测证据，F04）

在按"适配层"路线推进前，先用**真实 Postgres（本机 `chainpulse-postgres`，5432）**核对，发现比端口缺口更严重的问题。

**证据 1 — 全新库跑迁移在 000004 直接失败（修复前）**

```text
ok   000001_init_schema.up.sql
ok   000002_api_keys.up.sql
ok   000003_webhooks.up.sql
FAIL migrations/000004_event_natural_key_constraint.up.sql (rc=3):
ERROR:  relation "events_metadata" does not exist
```

`000004` 对 `events_metadata` 加约束，而**没有任何迁移创建过这张表**（000001 建的是 `blockchain_events`）。

**证据 2 — 代码的 INSERT 在真实 schema 上必然失败**

`postgres_database.go:158` 生成：
```sql
INSERT INTO blockchain_events (event_hash, chain_id, block_number, transaction_hash,
  log_index, contract_address, event_name, event_data, timestamp) VALUES (...)
```
在迁移产物上实测：
```text
ERROR:  column "event_hash" of relation "blockchain_events" does not exist
```

**三方对照**：

| 来源 | 表 | 关键列 |
| --- | --- | --- |
| 迁移 000001 | `blockchain_events` | `id`(PK)、`block_hash`(NOT NULL)、`event_data` |
| 本机 live 库 | **`events`** | `id`(PK)、`block_hash`、`event_data`（**无 `blockchain_events`**） |
| 代码 `postgres_database.go` | `blockchain_events` | 期望 `event_hash`、`data`、`indexed_params` |

实测列清单（全新库）：`id chain_id block_number block_hash transaction_hash log_index contract_address event_name event_data timestamp created_at status correlation_id`。
代码额外引用且**任何迁移都不存在**的列：`event_hash`（6 处）、`data`、`indexed_params`（各 2 处）。
代码自身还不自洽：INSERT 用 `event_hash`，而 `:557` 的 DELETE 用 `WHERE id = $1`。

**结论**：Postgres 读写路径**对任何真实 schema 都无法工作**。这不是"缺一个方法"，而是**表名 + 主键 + 列集合三方不一致**。因此：

- **适配层无法建立**：在把适配器包成 `core.DatabasePlugin` 之前，必须先决定"事实来源 schema"（T02 明令：`禁止为迎合旧文档创建第二套表`）。
- **这正是 T04 的职责**（事件身份与迁移：`修改 event hash 与相关 DB 映射`）。建议**把 T04 提到 Postgres 适配层之前**。

**本轮已做的无争议修复**：`000004` 改为对 `blockchain_events` 加约束（约束名 `uq_blockchain_events_natural_key`），与代码 `ON CONFLICT (chain_id, block_number, transaction_hash, log_index)` 一致。修复后全新库**迁移 8/8 全过**。
**未做（属 T04）**：`event_hash`/`data`/`indexed_params` → 真实列的映射；`id` vs `event_hash` 主键统一；live 库 `events` 与迁移 `blockchain_events` 的取舍。

## 4. 对后续任务的影响

- **T03+ 可直接复用 digest oracle** 做链隔离与重组摘要比对（`CanonicalEvents` / `CanonicalBlocks`）。
- 端口缺口是 **T05（原子提交）** 的前置：事务提交能力要落在"真实契约边界"，而当前 Postgres 适配器连读端口都不满足。建议在 T05 之前先决定：**加适配层**（推荐，不动现有适配器签名）还是**对齐签名**。
- `contracts` lane 的"必需 skip=0 / executed>0"要等上述缺口关闭才能真正满足。

---

## 5. 复现命令

```bash
export GOPROXY=https://goproxy.cn,direct

# digest oracle
go test -race -count=1 ./test/acceptance/tools/digest/...

# contract suite（内存实现）
go test -json -tags=integration -race -count=1 ./test/contracts/... | tee /tmp/contracts.jsonl
python3 scripts/acceptance/verify-tests.py --jsonl /tmp/contracts.jsonl --min-tests 4

# Foundry
cd contracts && forge install foundry-rs/forge-std@v1.9.6 --no-git && forge build && forge test
```
