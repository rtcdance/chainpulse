# 系统性搜索"被静默回退的修复"：方法与它的盲区（计划 §10，2026-10-14）

> 起因：`2026-10-14-silent-revert-kafka.md` 发现的静默回退是**手工撞见的**。既然回退能藏 146 天，自然要问：**还有多少？**
> 本轮做了这个搜索，结论有两层：**没有找到第二例同类回退；但搜索方法本身有一个已知盲区，而且盲区正好覆盖了那一例。**

---

## 1. 方法：提交级"存活率"

工具 `scripts/audit/reverted-fixes.py`：

1. 取**声称修复**的提交（消息匹配 `fix|security|concurren|resilien|correctness|P0|P1|race|leak`）；
2. 只看**改动过生产 Go 代码**（`pkg/` 或 `cmd/` 下的非测试文件）的提交——否则报告会被文档/prompt/CI 的频繁改写淹没；
3. 取该提交**新增的行**，过滤掉空行/注释/过短行；
4. 与**HEAD 中这些文件仍存在的行**比对，算出**存活率**；
5. 报告存活率 ≤50% 的提交。

```bash
python3 scripts/audit/reverted-fixes.py --limit 500 --json out.json
```

## 2. 结果：233 个修复类提交 → 10 个候选

| 存活率 | 提交 | 日期 | 主题 |
| --- | --- | --- | --- |
| 0% | `0bb8db03` | 2026-05-18 | fix: gofmt playground/main.go |
| 0% | `1d4310a0` | 2026-05-18 | fix: correct db.Initialize call in playground |
| **13%** | **`8dc093b`** | 2026-05-20 | fix: runtime resilience — timeouts, panic recovery, context deadlines |
| 14% | `4e29936` | 2026-05-22 | fix: data race in SetPublishTimeout + nil guard |
| 25% | `737895e5` | 2026-06-03 | fix: update monolithic wiring and event processor |
| 26% | `dba17e1f` | 2026-04-09 | Fix H5 demo acceptance and probes |
| 26% | `56cb3334` | 2026-04-08 | Fix Prometheus readiness acceptance checks |
| 34% | `8e707c39` | 2026-04-07 | fix: WebSocket 404, PostgreSQL SSL, GraphQL |
| 40% | `e5122e67` | 2026-05-17 | fix: GetMetadataBatch implementations |
| 44% | `6fee91be` | 2026-05-18 | fix: resolve all remaining CI test failures |

## 3. 人工分诊：**没有找到第二例同类回退**

| 候选 | 分诊结果 |
| --- | --- |
| `4e29936`（**data race** 修复，14%） | ✅ **修复仍在**：`pkg/core/eventbus.go:40` 仍是 `publishTimeout atomic.Int64`，`Store`/`Load` 都在。存活率低是因为**同提交里的其它新增行**（metric 常量等）被重构掉了。→ **假阳性** |
| 同上，另一半"`writeCacheEntry` nil guard" | 函数 `writeCacheEntry` **整个已被删除** ⇒ 该 guard 不是被回退，而是**随函数一起重构掉了**。→ 非回退 |
| `8dc093b`（13%） | 它新增的行里：`Timeout: 30 * time.Second` ✅ 仍在、`recover()`/`handlePullerPanic` ✅ 仍在、`WithTimeout(..., 5*time.Minute)` ❌ **已被有意移除**（即 `2026-10-14-changelog-verification.md` §3 那条）。→ 已知、且移除**正确** |
| 其余 7 个 | 主题均为 CI / demo / 探针 / wiring 的演进，属正常重构范围，**未发现正确性属性丢失** |

⇒ **未发现第二例"正确性修复被静默回退"。**

## 4. ⚠️ 但方法有盲区，而且盲区正好覆盖了那一例

**工具没能找到已知的 Kafka 回退**（`8f3c475` 与 `cf345a2` **都不在候选列表里**）。原因：

- `8f3c475` 共新增 **292 行**（239 行非平凡），**横跨 12 个文件**；
- 其中**只有 2 行**是那次回退（`CommitInterval: p.getCommitInterval(),`）；
- 于是它的**提交级存活率约 99%**，**远高于 50% 阈值** ⇒ 永远进不了列表。

**即：提交级存活率看不见"大提交里的外科手术式回退"——而这恰好是本案的形态。**

| 工具擅长 | 工具不擅长 |
| --- | --- |
| 整批被撤销（整分支回退、大块删除） | **两行正确性修复被无关大提交覆盖**（本案） |

## 5. 真正有效的方法：**从"声明"出发**

回头看得出的结论是：**找到 Kafka 回退的不是"扫行"，而是"扫声明"**。

1. 从 **CHANGELOG 的一条声明**出发；
2. **回代码核对**（发现不符）；
3. 用 `git log -S "<关键行>"` **回溯**，区分"从未做对"与"做对后被推翻"。

这个方法之所以有效，是因为**声明是有限的、可枚举的**（CHANGELOG + README + 关键注释 + 提交信息），而**行是无界的**。本会话按这个顺序连续挖出三件事：

| 顺序 | 产出 |
| --- | --- |
| 声明 → 代码 | README 的 gRPC 声明**无实现**（`2026-10-14-readme-claims-audit.md`） |
| 声明 → 代码 | CHANGELOG 7 条中 **2 条不符**（`2026-10-14-changelog-verification.md`） |
| 声明 → 回溯 | Kafka at-least-once **被静默回退 146 天**（`2026-10-14-silent-revert-kafka.md`） |

## 6. 结论与建议

1. **未发现第二例同类静默回退**——这是一个**有信息量的否定结论**，前提是方法与盲区都讲清楚。
2. **不要把这个工具当成"回退扫描器"**：它只能发现**整批消失**；对"大提交里的小回退"无效。这一点已写进脚本的 docstring。
3. **要继续覆盖盲区，应当扩展"声明清单"而不是"行清单"**：例如把每个 ADR 的"决定"、每个 `F*` 缺陷的修复、以及 README/CHANGELOG 的每条能力声明，都要求能指向**断言它的测试**。**有测试守着的东西，回退会让它变红**——这正是 Kafka 那次所缺的（本会话已补 `TestConsumerReaderConfigDisablesAutoCommit`）。
4. **已并入 D15**：为已修复缺陷补"断言不变式"的测试；CHANGELOG 条目须指向测试而非提交信息。

## 7. 证据

`build/acceptance/<sha>/reverted-fixes/`：`report.txt`、`candidates.json`、`candidates-v1.json`（未加生产代码过滤的版本，用于说明过滤的必要性：45 → 10 个候选）。
