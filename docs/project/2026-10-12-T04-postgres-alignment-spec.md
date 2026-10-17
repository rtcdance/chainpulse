# T04 前置：Postgres schema ↔ 代码 对齐规格（可机械执行）

> 来源：T02 实测发现的 P0（见 [T02 状态文档](./2026-10-12-T02-adapters-and-schema.md) §3.5）。
> 用途：**T04（事件身份与迁移）** 的输入。本页把每一处不一致列成"现状 → 目标"，
> 使 T04 无需重新发现即可执行。**本页不是已完成记录。**
> 日期：2026-10-12，Asia/Singapore。

---

## 0.1 执行记录（本轮**已执行**，非仅规格）

在建立可验证回路（socat 端口转发 + 迁移过的测试库）后，直接改了生产代码并用**真实 Postgres**验证。

**先发现的更根本 bug**：`postgres_database.go` 的 `Initialize` 打开连接后**从不保存**（`grep "p\.db ="` 零命中）⇒ 所有方法走 `p.db == nil` 返回 "database not initialized"，`Start()` 还 nil-panic。**该适配器此前完全不可用**。已修。

**已对齐（`pkg/plugins/database/postgres_database.go`）**：

| 位置 | 修复 |
| --- | --- |
| `Initialize` | ping 成功后 `p.db = db`（失败 `Close` + `RecordError`） |
| `WriteEvent` / `WriteEvents` / `WriteBatch` | `event_hash` → **`id`**；补 `block_hash`；哈希/地址 `.Hex()`；`event_data` 经 `eventDataJSON` |
| `QueryEvents` | SELECT 改真实列；扫描经 string 中转 + `common.HexToHash/HexToAddress` |
| `GetEventByHash` | `WHERE id = $1`（原 `event_hash`）；同样扫描转换 |
| `GetAllEvents` / `GetEventsByBlockRange` | 删 `event_signature`/`event_hash`/`data`/`indexed_params`，改真实列 |
| `GetAllBlocks` / `GetBlock` | 删 `miner`/`difficulty`/`gas_used`/`gas_limit`，改 `number, hash, parent_hash, timestamp` |

**验收（真实 Postgres，非 mock）**：`TestPostgresEventRoundTrip` **PASS**，覆盖
`WriteEvent → GetEventsByBlockRange / GetEventByHash / GetAllEvents / QueryEvents / GetAllBlocks / GetBlock / DeleteEvent`；
`block_hash` 与 `event_data` 往返一致。门禁：`go build` 0 / `make check` 0 / `make lint` 0 issues / `make test-short` 66 ok。

**仍未做（见 §2、§3）**：F06 身份（`ComputeEventHash` 加 `block_hash`）、`status`/`correlation_id` 写入、block 写入路径、live `events` 表处置、T02 的 PG 契约 runner + 适配层。

### 0.2 后续轮次补齐（全部完成）

| 项 | 落地 | 验证 |
| --- | --- | --- |
| **F06 身份** | `ComputeEventHash` 输入改为 `chain:block_hash:tx:log_index`；迁移 **000009** 把唯一约束加宽为含 `block_hash`；`ON CONFLICT` 同步 | `TestPostgresBlockHashIdentity`：同高两分支**并存两行**、重复投递**折叠** |
| **`status` / `correlation_id` 持久化** | 三个 INSERT 合并为共享常量 `insertEventSQL`（12 列）；四个读路径合并为 `selectEventColumns` + `scanEvent` | `TestPostgresEventRoundTrip` 断言 `status` 与 `correlation_id` 往返一致 |
| **block 写入路径** | 新增 `WriteBlock(ctx, chainID, *Block)`：`id = chain_id:number` 作为 PK **在结构上强制 I03**，同高替换走 `ON CONFLICT (id) DO UPDATE` | `TestPostgresBlockUpsert`：同高写两次 → `GetBlock` 返回替换后的 hash，且该高度**只有一行** |
| **legacy `events` 表** | 迁移 **000010**：若存在 `public.events` 则**重命名为 `events_legacy`**（可逆、不丢数据）；down 恢复原名 | 实测：含 `events` 的库迁移后为 `events_legacy`；down 后恢复 `events` |
| **T02 适配层 + PG 契约 runner** | `PostgreSQLPluginAdapter`；`TestPostgresDatabaseContract` 用**同一套契约**跑 Postgres | 五个子用例全 PASS |

迁移总量：全新库 **10/10** 通过。

**清理**：`pkg/plugins/database/postgres_database.go` 的 12 处 SQL 收敛为 2 个常量 + 1 个 `scanEvent`，杜绝"三处 INSERT 各自漂移"（正是最初那类 bug 的成因）。

---

## 0. 事实来源（T02 已核）

**迁移产物（唯一事实来源）** —— `migrations/000001_init_schema.up.sql` + 000005/000007：

```
blockchain_events(
  id VARCHAR(255) PK,
  chain_id VARCHAR(64) NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash VARCHAR(128) NOT NULL,
  transaction_hash VARCHAR(128) NOT NULL,
  log_index INTEGER NOT NULL,
  contract_address VARCHAR(128) NOT NULL,
  event_name VARCHAR(255) NOT NULL,
  event_data JSONB,
  timestamp BIGINT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  status VARCHAR(50) NOT NULL DEFAULT 'pending',   -- 000005
  correlation_id VARCHAR(128)                       -- 000007
)
约束：uq_blockchain_events_natural_key (chain_id, block_number, transaction_hash, log_index)  -- 000004（本轮修复）

blocks(
  id VARCHAR(255) PK, chain_id VARCHAR(64) NOT NULL, number BIGINT NOT NULL,
  hash VARCHAR(128) NOT NULL, parent_hash VARCHAR(128), timestamp BIGINT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

**本机 live 库**（`chainpulse-postgres`）另有一张 **`events`**（同列、空表）——**没有任何迁移创建它**。T04 必须二选一：把 live `events` 视为历史遗留并在 T04 迁移中显式删除/改名，或让迁移创建它。**当前选择：以迁移为准（`blockchain_events`）**，`events` 标为遗留。

---

## 1. 不一致清单（`pkg/plugins/database/postgres_database.go`）

### 1.1 列/表名不存在

| 行 | 现状 | 目标 |
| --- | --- | --- |
| 158, 224, 312 | `INSERT INTO blockchain_events (event_hash, …)` | 列 `event_hash` → **`id`**；并补 `block_hash` |
| 394 | `SELECT event_hash, …` | `event_hash` → **`id`** |
| 507 | `WHERE event_hash = $1` | → **`WHERE id = $1`** |
| 557 | `WHERE id = $1` | ✅ 已正确（与 INSERT 不自洽，见 §1.3） |
| 651, 748 | `SELECT …, event_signature, …, event_hash, data, indexed_params, created_at` | 删除不存在的 `event_signature`/`event_hash`/`data`/`indexed_params`；改为真实列：`id, chain_id, block_number, block_hash, transaction_hash, log_index, contract_address, event_name, event_data, timestamp, created_at` |
| 702, 796 | `SELECT number, hash, parent_hash, timestamp, miner, difficulty, gas_used, gas_limit FROM blocks` | 删除不存在的 `miner`/`difficulty`/`gas_used`/`gas_limit`；改为 `id, chain_id, number, hash, parent_hash, timestamp` |

### 1.2 类型映射

| Go 侧 | 现状 | 问题 | 目标 |
| --- | --- | --- | --- |
| `event.TransactionHash` (`common.Hash`) | 直接作为 `$n` 传入 / `&event.TransactionHash` 扫描 | `[32]byte` 不实现 `driver.Valuer`/`sql.Scanner`，lib/pq 无法处理 | 写：`.Hex()`；读：先扫进 `string`，再 `common.HexToHash` |
| `event.BlockHash`, `event.ContractAddress` | 同上 | 同上 | 同上（`.Hex()` / `common.HexToAddress`） |
| `event.EventData` (`[]byte`) | 直接传入 `event_data`（JSONB） | 原始字节通常不是合法 JSON → 插入失败 | 写：空则 `NULL`，否则 `json.Marshal(string(data))`；读：`json.Unmarshal` 回 `[]byte` |
| `event.LogIndex`, `event.BlockNumber` (`uint64`) | 直接传入 | 表为 `INTEGER`/`BIGINT`；`log_index INTEGER` 与 `uint64` 有窄化风险 | 写：`int64`；读：`uint64` |
| `event.BlockTimestamp` (`int64`) | 传入 `timestamp` | ✅ 匹配 | 保持 |
| `status` / `correlation_id` | 代码未写 | 列有 `DEFAULT`/可空 → 插入可行，但**读回丢失** | T04：写入 `status`、`correlation_id`（`BlockchainEvent` 已有字段） |

### 1.3 代码自身不自洽

- INSERT 用 `event_hash`，`DeleteEvent`（:557）用 `WHERE id = $1` ⇒ **同一文件两套主键**。T04 统一为 `id`。
- `QueryEvents`（:394）不 SELECT `chain_id`/`block_hash`，读回的事件缺这两个字段。

---

## 2. 事件身份（I02 / F06）

现状 `pkg/core/event_hash.go`：

```go
hashInput = chain_id : block_number : tx_hash : log_index   // 缺 block_hash
```

**I02 要求 physical event ID = (chain key, block_hash, tx_hash, log_index)**，故 F06 成立：
- 同高替换分支（不同 `block_hash`、同 `tx_hash`/`log_index`）会算出**相同** hash ⇒ 无法区分。
- 唯一约束 `uq_blockchain_events_natural_key` 同样缺 `block_hash`。

**T04 动作**：
1. `ComputeEventHash` 输入加入 `event.BlockHash.Hex()`；保留 `EventNaturalKey` 的人类可读形式并同步。
2. 唯一约束改为 `(chain_id, block_hash, transaction_hash, log_index)`（新迁移 `000009`，expand→backfill→verify→switch）。
3. 保留旧 ID 映射（`BlockchainEvent.ID` 与 `EventHash` 的兼容读取）。

---

## 3. 验收（T04 DoD）

- [ ] 全新库 `migrations/*.up.sql` 8/8 通过（**已达成**，见 T02 §3.5）。
- [ ] **真实 Postgres 事件写读往返**：`WriteEvent` → `GetAllEvents`/`GetEventByHash` 字段逐一相等（`test/contracts/postgres_roundtrip_test.go`，当前 **RED**）。
- [ ] **真实 Postgres 区块写读往返**：`blocks` 路径同样通过。
- [ ] 相同消息重复 100 次 → 单份效果（`ON CONFLICT` + 唯一约束）。
- [ ] 同高异 `block_hash` → 两条可审计记录（F06 修复后）。
- [ ] 迁移前后摘要一致（用 `test/acceptance/tools/digest` 的 `CanonicalEvents`）。
- [ ] 完成后回填 T02：PG 契约 runner + 适配层（`core.DatabasePlugin`）。

## 4. 风险

- `event_data JSONB` 与 `[]byte` 的语义映射是**权宜之计**；若 T04 决定改列为 `BYTEA`/`TEXT`，需独立迁移并说明。
- live 库的 `events` 表来源不明；T04 迁移须显式处理（不可静默忽略）。
- 本机 **Kafka 9092 未起**，T05 的 Kafka 部分需先起依赖。
