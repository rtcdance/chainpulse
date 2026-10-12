# T17 交付：README 与真实结构对齐（F20）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T17 / P1 / 8h**。
> 状态：**In Progress** —— **F20 的 README 对齐已完成并机器验证**；
> 架构图、三份 ADR、录屏、性能/故障报告、英文摘要、F26 两份叙事文档未做。
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

## 4. 未完成（T17 剩余）

- [ ] **一张实际架构图**（当前 README 只有文字表格）。
- [ ] **三份核心 ADR**（`docs/adr/` 已存在，需确认是否有三份可交付）。
- [ ] **3–5 分钟录屏**、**性能报告**、**故障报告**、**英文项目摘要**。
- [ ] **F26 两份轻量叙事文档**：10x/50 链扩展设计（**只设计不实现**）、Web3 安全案例讨论提纲（桥攻击 / MEV / 账户抽象的防线取舍）；要求与项目 ADR **相互引用**且**不暗示未实现能力**。
- [ ] **20 个问题有代码/测试答案**的整理。
- [ ] **新机器 5 分钟 happy path / 15 分钟复现故障**（需可编排环境）。
