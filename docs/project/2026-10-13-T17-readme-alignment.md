# T17 交付：README 与真实结构对齐（F20）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T17 / P1 / 8h**。
> 状态：**In Progress** —— **F20 的 README 对齐已完成并机器验证**；
> **F26 两份叙事文档已产出并机器验证链接（§4）**；
> **README 架构图（Mermaid）已加（§5）**；
> **三份核心 ADR + ADR-004 编号修复已完成（§6）**；
> **20 个面试问题已重写并校验路径（§7）**；
> 录屏、性能/故障报告、英文摘要、新机器 happy path 未做。
> 日期：2026-10-13，Asia/Singapore。

---

## 1. F20 的三处问题（逐条实测）

F20 原文："**README 结构树缺 `contracts/` 等真实目录，badge 只指向 ci/test 两个 workflow；与目标 lane 表和最终交付形态不一致**"。实测后比 F20 描述的更多一处：

| # | 问题 | 实测证据 |
| --- | --- | --- |
| 1 | 结构树缺真实目录 | 真实顶层有 `contracts/ migrations/ examples/`，树里**都没有**；`pkg/` 与 `docs/` 子树严重不全（真实 24 个 `pkg/` 子目录，树里列了 11 个） |
| 2 | 结构树**描述过时** | 树里把 `test/acceptance/` 写成 "Playwright UI acceptance tests" —— 它现在同时承载 **Go 验收测试**与 `tools/`；`test/` 真实有 8 个子目录，树里只有 3 个 |
| 3 | badge 只覆盖 2 个 workflow | 实际存在 **7 个** workflow（`ci`、`test`、`coverage`、`release`、`deploy`、`metrics`、`ci-integration`），README 只 badge 了 `ci` 与 `test` |
| 4 | **一处失效链接**（F20 未提到） | `[scripts/README.md](scripts/README.md)` —— **该文件不存在**。T17 验收明写"README 无过时路径" |

## 2. 修复

### 2.1 结构树按真实布局重写

- 补 `contracts/`（Foundry + vendored forge-std）、`migrations/`、`examples/`、`cmd/playground/`；
- `pkg/` 子树补 `ports/`、`domain/`、`evm/`、`siwe/`、`runtimetypes/` 等，并给 `plugins/` 与 `services/` 加上真实说明；
- `docs/` 子树按真实目录列出（`adr/`、`architecture/`、`deployment/`、`guides/`、`operations/`、`project/`、`specs/`）；
- `test/` 子树补全 8 个真实子目录，并修正 `acceptance/` 的过时描述；
- 补齐 `scripts/acceptance/` 一行（四个验收工具）。

**刻意不列入**：`build/`（生成物）与 `optimization-plan/`（被 gitignore，且按项目规则保持本地）——列进结构树会让树描述不存在于仓库的东西。

### 2.2 badge 与真实 workflow 对齐

补 `coverage.yml`、`release.yml`、`deploy.yml` 三个 badge（**文件确实存在**）。

**刻意不加** `ci-integration.yml` 的 badge：T01 已确认它是"**空 suite 假绿**"的来源，给它挂 badge 等于把已知不可信的 check 当成门面。

### 2.3 补齐失效链接指向的文件

新建 **`scripts/README.md`**：说明四个验收工具（各自的职责与调用方式）、顶层 25 个脚本的分组、以及五个子目录。它同时兑现了 README 里"50+ automation and verification scripts"的说法（实测 `scripts/` 共 78 个文件）。

## 3. 验收（**机器验证**，非目测）

用一段脚本抽取结构树里的**每一条路径**与**每一个 badge 的 workflow 文件**并核对存在性：

```
tree paths checked: 51; MISSING: 0
badges checked: 5; broken: 0
badged workflows: ci.yml, coverage.yml, deploy.yml, release.yml, test.yml
```

相对链接扫描：

```
relative links checked: 18; BROKEN: 0      # 修复前为 1（scripts/README.md）
```

⇒ 满足 T17 验收的"**README 无过时路径/失效 badge**"。

> 写这个检查器时我第一版**算错了树的深度**（把 `└── test/` 的子项当成根级），结果 51 条路径全报缺失。修正深度公式后才得到 0 missing —— 这也是为什么"检查器本身也要被检查"。

## 4. F26：两份轻量叙事文档（已完成，机器验证链接）

T17 验收对 F26 的原文要求：**"补两份轻量叙事文档（F26）：10x/50 链扩展设计（只设计不实现）、Web3 安全案例讨论提纲（桥攻击/MEV/账户抽象的防线取舍），作为面试深挖储备"**，且**"10x 设计与本项目 ADR 相互引用，不暗示未实现能力"**。

| 文档 | 覆盖 F26 要求 | 与 ADR 相互引用 |
| --- | --- | --- |
| [`2026-10-13-10x-chain-scaling-design.md`](2026-10-13-10x-chain-scaling-design.md) | §1 现状表、§2 三处失速点、§3 设计（5 小节，逐条 extends ADR-001/002/003）、§4 待测量清单、§5 明确不声称的能力 | ADR-001、ADR-002、ADR-003 |
| [`2026-10-13-web3-security-discussion.md`](2026-10-13-web3-security-discussion.md) | 桥攻击 / MEV / 账户抽象 各含"攻击形态 + 防线取舍表 + 值得争论的取舍 + 本项目相邻与不相邻" | ADR-003、ADR-005 |

**"不暗示未实现能力"是如何机械保证的**：两份文档都在开头用引用块声明"设计文档/讨论提纲，不是实现，也不是覆盖声明"，并在结尾各有一段"**deliberately does not claim**" / "**does not implement**" 清单；每个安全议题都显式写出"本项目与此相邻之处"与"**它不是**（桥防御 / MEV 缓解 / 账户抽象实现）"。

链接验证（相对链接解析，非目测）：

```
2026-10-13-10x-chain-scaling-design.md:   3 link(s), broken 0   # 修复前 3（../../adr/ 深度写错）
2026-10-13-web3-security-discussion.md:   2 link(s), broken 0   # 修复前 2
```

> 首次产出时把 ADR 链接写成 `../../adr/...`（从 `docs/project/` 出发应是 `../adr/...`，只上一级）。检查器抓到 5 条全断，修正后归零 —— 与 §3 的"检查器本身也要被检查"同一个教训。

## 5. 架构图（已完成）

T17 要求"一张实际架构图"，而 README 原本只有文字表格。已加一张 **Mermaid flowchart** 到 README 的 `## 📊 Architecture` → `### System Overview`：

- 节点名用**真实包名**（`plugins/pullers`、`services/decoder`、`services/processor`、`plugins/database`、`plugins/api`、`siwe`、`services/reorg`、`services/finality`、`observability`），不虚构组件；
- 画的是**已实现的索引路径**：puller → decoder → processor → **fenced batch commit** → PostgreSQL（事件 + checkpoint 原子提交），旁挂 Kafka 重投、Redis 版本化缓存、`DegradingReader`、reorg 修复、finality、lag 指标；
- **不暗示未实现能力**：图注明写"今天真正压过的是双链隔离测试与单链负载路径，不是多链生产部署"；
- ADR 段落指向**确实存在**的 ADR（ADR-001 plugin architecture / ADR-002 event bus / ADR-003 reorg）。三份核心 ADR 见 §6。

README 相对链接复检：**18 条，BROKEN 0**（新增的 ADR 链接均存在）。

## 6. 三份核心 ADR + 编号修复（已完成）

plan §11 要求"三份核心 ADR：事件身份/canonical 模型；DB 提交与 Kafka 进度协调；finality 与重组降级策略"，每份说明备选方案、选型理由、失败语义、测试和代价。已产出：

| ADR | 主题 | 关键内容 | 关联 finding |
| --- | --- | --- | --- |
| [ADR-008](../adr/ADR-008-event-identity-canonical-digest.md) | Event Identity and the Canonical Digest | 物理身份 `(chain_id, block_hash, tx_hash, log_index)`；canonical 摘要有序无关 + 幂等；同 ID 不同 payload = 冲突报错 | F06、F21 |
| [ADR-009](../adr/ADR-009-db-commit-kafka-progress.md) | Database Commit and Kafka Progress Coordination | events + checkpoint 单事务 fenced 提交；partition key = chain key；at-least-once + 去重；outbox | F04、F05、F28 |
| [ADR-010](../adr/ADR-010-finality-reorg-degradation.md) | Finality and Reorg Degradation | finality 是置信度非布尔；只有 finalized 可提升；reorg 修复有序且 generation 作用域 | F08 |

**编号冲突修复**：`docs/adr/` 原有**两个 ADR-004**。保留较早的 `ADR-004-env-var-naming-convention`（2026-05-05），把 `ADR-004-ethclient-migration`（2026-05-13）重编号为 **ADR-007**（`git mv` 保留历史，标题加注说明）。新增 [`docs/adr/README.md`](../adr/README.md) 作为索引。

每份 ADR 的"看测试"段引用的测试名均经 `grep '^func Test'` 核对存在。

## 7. 20 个面试问题（已完成）

plan §11 要求"20 个面试问题按四组各 5 题整理，每题能指向代码与测试"。已重写 [`docs/guides/INTERVIEW_MAP.md`](../guides/INTERVIEW_MAP.md)：

- 按 plan 的四组各 5 题（Go / Web3 / 后端 / 架构与交付）；
- **旧版有 12 条失效路径**（`pkg/core/` 下 bloom_filter、siwe、mev_flashbots、event_signature、event_decoder、ecdsa_verify、aa_mempool、l2_bridge、l2_events、mev_builder，`multi_rpc_puller.go`，以及不存在的 `docs/adr/ADR-003.md`）；
- 新版每题指向**已存在**的代码 + 测试；抽路径脚本核对：**47 条路径引用，MISSING 0**；
- 失效路径的教训保留在文末注记，不抹掉。

## 8. 未完成（T17 剩余）

- [ ] **3–5 分钟录屏**（需可编排环境 + 显示）。
- [ ] **性能报告**、**故障报告**（需真实发压/故障运行）。
- [ ] **英文项目摘要**。
- [ ] **新机器 5 分钟 happy path / 15 分钟复现故障**（需可编排环境）。
