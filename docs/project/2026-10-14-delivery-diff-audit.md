# 交付差异核查（计划 §10 清单项，2026-10-14）

> 对应计划 §10："**实际文件差异只包含各任务允许的变更；无吞并其他人的dirty changes。**"
>
> **结论：通过。** 工作树干净、无未跟踪文件、本会话 15 个提交共 27 个文件全部为预期变更、**无删除**、与他人在途工作**无交叉吞并**。

---

## 1. 核查项与结果

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 工作树是否干净 | `git status --short` | **空**（无未提交改动） |
| 是否有未跟踪文件 | `git status --porcelain -uall` | **空** |
| 本会话是否删除过文件 | `git log --diff-filter=D --name-only 76a6df8~1..HEAD` | **空**（无删除） |
| 本会话是否改过重命名 | `git log --diff-filter=R …` | 无 |
| 本会话改动文件清单 | `git log --name-only 76a6df8~1..HEAD` | **27 个，逐一核对全部为预期** |

> 范围用 `76a6df8~1..HEAD`（含本会话第一个提交本身）。用 `76a6df8..HEAD` 会漏掉该提交自己的文件，数出来是 26 —— 我第一版就写错了这个数字，特此更正。

## 2. 本会话的 27 个文件（按类别）

| 类别 | 文件 |
| --- | --- |
| **代码（生产）** | `pkg/plugins/mq/kafka_mq.go`、`pkg/plugins/pullers/https_jsonrpc_puller.go` |
| **代码（测试）** | `pkg/plugins/mq/kafka_consumer_commit_test.go`、`pkg/plugins/pullers/verify_chain_id_test.go` |
| **工具** | `scripts/acceptance/verify-tests.py`、`scripts/acceptance/acceptance-gate.py`、`scripts/audit/unwired-components.py` |
| **依赖声明** | `go.mod`、`go.work`、`pkg/{blockchain,chainid,configmodel,histogram,logkeys,runtimetypes}/go.mod`、`docs/project/DEPENDENCY_APPROVAL.md` |
| **文档** | `docs/project/2026-10-1{3,4}-*.md` 共 11 篇、`docs/project/2026-10-12-T13-slo-diagnostics.md`、`docs/project/2026-10-13-write-path-findings.md` |

**没有一个是计划外文件**；尤其**没有**把并行会话的产物（`docs/project/2026-10-13-handover.md`、`write-path-findings.md` 的他人段落、`docker-compose.*` 等）混进我的提交——其中 `write-path-findings.md` 我只**追加**了一节（§10），未改动他人内容。

## 3. 一个需要说明的观察：本仓库同时有**多条工作流在提交**

最近 40 个提交**全部署名为 `av`**（本环境的 git 身份），因此**无法按作者区分**谁是谁。本会话的 15 个提交是我按**内容**辨识出来的（`76a6df8~1..HEAD` 恰好 15 个，与我的操作一一对应）。

同期还有**另一条会话**的提交（`9139631` handover、`f464df2` I04 findings、`061b58b` I04 实现、`e3c9643`/`2d46c57`/`c927bc5` T12 与 F26/F27 等），以及一次**第三方清理**（`8a1374d av chore: clean up tracked IDE/AI config`，删除了 `.codex/`、`.claude/` 等）。

**这正是本条要防的风险**（多人/多会话共享工作树，容易互相吞并在途改动）。核查结论是**未发生**：每条提交的文件集都与它的信息自洽，工作树始终干净。

## 4. 边界（如实）

- 本核查只覆盖**本会话的 15 个提交**；他人提交的"是否只含该任务允许的变更"需要各自的作者确认（我无权替他们判定）。
- 计划里"各任务允许的变更"是按 T 任务定义的文件白名单；本会话的工作**大部分不在原计划的 T 任务范围内**（它们是 T18 预检、缺陷修复、审计与决策梳理），因此我按"**提交信息与文件集是否自洽**"来判定，而不是对照某个 T 任务的白名单。
- 所有提交共用同一 git 身份 `av`，所以**审计轨迹无法区分贡献者**——这本身是一个值得注意的仓库配置问题。
