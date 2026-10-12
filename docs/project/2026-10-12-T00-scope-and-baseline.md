# T00 交付：范围、规则与基线（唯一口径）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T00 / P0 / 5h**。
> 状态：**In Progress**（本文为 T00 的可执行产物；待人工确认后转 Verified）。
> 日期：2026-10-12，Asia/Singapore。所有数字均来自**当日命令实测**，不沿用计划正文的初值（F27）。

---

## 0. 元信息（实测）

| 项 | 值 | 命令 |
| --- | --- | --- |
| HEAD | `d364744e68ad41a81fe0be5c9809af3b9feced4f`（`main`） | `git rev-parse HEAD` |
| 工作区 | **dirty**：88 `M` + 1 `R` + 6 `??` = **95** 条 | `git status --porcelain=v1 \| wc -l` |
| stash | **4** 个（`stash@{0..3}`） | `git stash list` |
| Go | `go1.26.0 darwin/arm64` | `go version` |
| linter | `golangci-lint 2.12.2 built with go1.26.0` | `make lint` 前置 |
| Foundry | `forge 1.8.5`（`~/.foundry/bin`） | `forge --version` |
| Docker | 运行中 | `docker info` |
| gh 账号 | active=`codex689`（scopes: `repo`,`read:org`,`gist`,`admin:public_key`）；另有 `rtcdance`（inactive，PAT） | `gh auth status` |
| workspace 模块 | **7**：`.` + `pkg/{blockchain,chainid,configmodel,histogram,logkeys,runtimetypes}`；`go.work` go 指令 = `1.26.0` | `go work edit -json` |

> ⚠️ **F14 前置风险**：active 账号 `codex689` **只有 `repo` scope，没有 `admin`**。GitHub ruleset / branch protection 需要仓库 **admin** 权限。T01 配置 ruleset 前必须先解决账号授权（切到 `rtcdance` 或补 scope），否则 §7.3 的 required checks 无法落地。**这是 T01 的已知阻塞点，必须在开工前确认。**

---

## 1. 规则源统一（F12）

### 1.1 指针文件（已核对，无需改）

`CLAUDE.md`、`.claude/rules.md`、`.github/copilot-instructions.md` 三者**均已**改为指向 `AGENTS.md` 的指针，且明确写"**no `pkg/adapters`**"。✅

### 1.2 与 AGENTS 冲突的旧规则页（**需处理**）

`docs/project/ARCHITECTURE_RULES.md`（2026-04-03）**规则 5：禁止创建 spec 文件** —— 与 `AGENTS.md`/本项目 spec 流程直接冲突。

- **动作**：该页改为指向 `AGENTS.md` 的指针（保留历史链接可读性，删除自相矛盾的规则正文）。
- **状态**：见本文 §7 的落地批次 `B0`。

### 1.3 已不存在的 `pkg/adapters` 引用（F12）

全仓 **74 处**（33 个 `.md` + 3 个 `.sh` + 其余在技能/归档）。分布：

| 类别 | 处置 |
| --- | --- |
| `docs/specs/archive/**`（历史归档） | **不动**（历史记录，改反而失真） |
| `.codex/archive/**`（历史归档） | **不动** |
| **活跃技能** `.codex/skills/{code-organization-placement,smart-contract-integration-safety,multi-chain-consistency,mempool-pending-tx-handling}/SKILL.md`、`.codex/CONVENTIONS.md` | **改**：删除/改写 `adapters/` 示例（技能会被 Agent 当规则读取，是真实污染源） |
| 活跃文档 `docs/{IMPLEMENTATION_STATUS,CHANGELOG,OPTIMIZATION_REPORT,DEBUGGING}.md`、`docs/architecture/*`、`docs/guides/*` | **改**：替换为实际目录 |
| `scripts/deploy-event-emitter.sh` | 核查后改（若为死路径则标注） |
| `optimization-plan/MASTER_PLAN.md`（gitignored） | 改 |

> 判据（T00 验收）：活跃文档/技能中 `pkg/adapters` 命中数 → **0**；归档目录允许保留。

---

## 2. skip 分类表（唯一口径，F19）

三处口径互不一致（AGENTS "~89"、基线 JSON "69 skip actions"、源码 `t.Skip(` "72 处/29 文件"）。**以命令实测为唯一口径：**

```text
grep -rn "t\.Skip(" --include="*_test.go" .            → 72 行 / 29 文件
grep -rn "pre-existing" --include="*_test.go" .        → 3 行
go test -json -short ...（基线）                        → 69 个 skip 测试动作
```

**口径定义**：
- **72** = 源码中 `t.Skip(` 的**出现次数**（含同一测试内的分支 skip、表驱动 skip）。
- **69** = 短测试运行时**实际触发**的 skip 动作数（部分 skip 因前置条件未达而未走到）。
- 二者不等，属正常：**源码点是上界，运行动作数是子集**。T00 起统一以"**运行动作数 + 源码点**"双列呈现，禁止只报一个数。

### 2.1 按 reason 分类（源码点，实测）

| 类别 | 计数 | 代表 reason | 去往哪个 lane | 处置 |
| --- | --- | --- | --- | --- |
| A. 依赖缺失（Anvil） | 16 | `"Anvil not available"`×14、`"...multi-chain test"`×2 | `evm-e2e`（必须执行） | **不可删**；lane 里必须真跑，缺依赖即 fail（§7.4） |
| B. 短模式主动排除 | ~20 | `"skipping ... in short mode"`、`"Skipping stress test in short mode"` | `unit-race`/`contracts` 短跑 skip；**full lane 必须执行** | 保留，映射到 full job |
| C. 真实服务缺失（PG/Redis/Kafka/Mongo） | ~10 | `"PostgreSQL database not available"`、`"set DATABASE_URL..."`、`"Requires Redis server..."` | `contracts`（必须执行） | 保留，`contracts` lane 起依赖后必须跑 |
| D. fixture/ABI 缺失 | 6 | `"ABI not available for Swap"`×2、`"Requires fixtures package"`×4 | `unit-race`/`evm-e2e` | **T02/T09 补齐 fixture 后必须执行**（F15/F16） |
| E. 历史残余 | 3 | `"regression"`×3 | 逐项判定 | T00 分类后归入 full job 或删（需人工确认） |
| F. 其他（并发/网络门控） | 其余 | `"skipping session manager concurrency test"`、`CHAINPULSE_RUN_HTTP_INTEGRATION` | 按依赖归 lane | 保留 |

> **铁律**：`t.Skip` **不是 pass**（AGENTS 防假通过规则 3）。T00 不删任何 skip；只建映射。删除仅在"确认为空壳且无人依赖"时**人工确认后**进行（A-3 在第二十轮已处理过一轮）。

### 2.2 测试规模（实测，F19）

| suite | Test 数 | 命令 |
| --- | --- | --- |
| `test/integration`（`-tags=integration`） | **147** | `go test -tags=integration ./test/integration/... -list '.*'` |
| `test/e2e`（`-tags=e2e`） | **349** | `go test -tags=e2e ./test/e2e/... -list '.*'` |
| `test/contracts`（`-tags=integration`） | **0** ⚠️ | `... -list '.*'` → 无输出（**空 suite，F02 确认**） |

`test/contracts` 为 0 是 **P0 缺陷**（PR 上跑空 suite 必然假绿），T02 负责补 `Test...` runner。

---

## 3. 支持矩阵（T00 冻结）

| 维度 | 值 | 证据等级 |
| --- | --- | --- |
| 生产验收组合 | PostgreSQL + Kafka + Redis + EVM | 目标（待 T02–T12 验收） |
| 零依赖学习路径 | 单体 + Playground（in-memory） | 已有，保留 |
| 基准链语义 | Ethereum/Sepolia JSON-RPC 标准标签 | 目标 |
| 本地确定性链 | 双 Anvil（隔离/故障测试） | 目标（T12） |
| 已验证链（本轮承诺） | **仅** EVM 双链 + 本地 Anvil | — |
| 保留但**未验证** | Solana / Cosmos 等历史能力 | **不得列入"已验证"** |
| 外部测试网 | 小范围烟测（证明真实接入），**不取代** CI | Pending |

---

## 4. 任务 → 证据映射（T00 冻结，T18 消费）

| 任务 | 必需工件 | lane |
| --- | --- | --- |
| T00 | 本文 + skip 分类表 + dirty→commit 计划 | — |
| T01 | 阳性对照 6 类失败 run URL；ruleset `gh api` 反证；lane 实测时长 | `quality`/`acceptance-gate` |
| T02 | `forge build/test` 输出；契约 suite `-list`；oracle 单测 | `contracts`/`evm-e2e` |
| T03–T07 | 各不变量（I01–I08）对应测试 + 摘要 | `contracts`/`evm-e2e` |
| T08–T11 | E07/E08/E10/E11/E12 场景测试 | `unit`/`contracts`/`frontend-api` |
| T12 | 一条启动入口 + oracle 摘要一致 + 证据收集器 | `evm-e2e` |
| T13–T14 | metrics/trace 证据；§8 原始数据 + 摘要 | `chaos-soak-perf` |
| T15–T16 | 同 digest 提升；回滚演练；§9 矩阵 | `release-candidate`/CD |
| T17 | README/图/ADR/录屏/报告 | — |
| T18 | `manifest.json` 全清单（收集器生成） | `acceptance-gate` |

---

## 5. dirty → commit 批次计划（F18，**关键**）

### 5.1 事实

- 关键正确性修复 **只在工作区，不在 HEAD**（实测）：
  - `git show HEAD:pkg/siwe/siwe.go | grep -c VerifySIWEWithOrigin` → **0**（工作区 = 2）
  - `git show HEAD:pkg/evm/typed_events.go | grep -c decodeSignedBigInt` → **0**（工作区 = 5）
  - `git show HEAD:pkg/services/processor/idempotency.go | grep -c CheckAndMark` → **0**（工作区 = 1）
- ⇒ 若不先落 commit，T18"从最终 SHA 新检出"验收会**静默丢失**这些修复。

### 5.2 批次划分（建议）

| 批次 | 主题 | 文件（示例） | 归属 |
| --- | --- | --- | --- |
| **B0** | 规则/文档指针 | `docs/project/ARCHITECTURE_RULES.md`、活跃技能 `adapters` 清理 | 本项目 T00 |
| **B1** | A-4a SIWE 严格校验 | `pkg/siwe/siwe.go`、`pkg/siwe/siwe_test.go`、`pkg/plugins/api/siwe_handler.go`、`siwe_handler_test.go` | 本项目 A-4a |
| **B2** | A-4b signed int 解码 | `pkg/evm/typed_events.go`、`typed_events_test.go`、`pkg/evm/chained_decoder.go`、`chained_decoder_test.go` | 本项目 A-4b |
| **B3** | A-4c~f 其他缺陷 | `pkg/plugins/api/dlq_handler.go`、`pkg/services/reorg/reorg_handler.go`、`pkg/services/processor/idempotency.go` + 3 个新 `*_test.go`（untracked） | 本项目 A-4c~f |
| **B4** | 版本/工具链 | `go.mod`/`go.sum`/`go.work`/6 个 `pkg/*/go.mod`、`Makefile`、`.golangci.yml` | 他人（版本升级） |
| **B5** | CI/workflow 微调 | `.github/workflows/{ci,test}.yml`、`.github/copilot-instructions.md` | 他人 |
| **B6** | 文档/README/QUICKSTART | `README.md`、`docs/**`、`cmd/**/QUICKSTART.md` | 他人 |
| **B7** | 死代码/脚手架清理 | `pkg/plugins/{database,pullers,mq}/**`、`pkg/services/query/qindex/**`、`test/integration/**` | 他人（A-1/A-2 系列） |
| **B8** | 工具配置（本机） | `package.json`、`playwright.config.ts`、`frontend/package.json` | 本项目（第二十轮） |

### 5.3 stash 处置（**需人工确认**）

| stash | 内容 | 建议 |
| --- | --- | --- |
| `stash@{0}` `my-cleanup-final` | 删 `pkg/generics`（-2026 行）+ `scripts/check-file-organization.sh` | **保留**（删除动作，丢弃需确认） |
| `stash@{1}` / `stash@{2}` | `.codex/active-skills.md` 小改 | 可丢弃 |
| `stash@{3}` `wip: local tooling` | `.codex/active-skills.md` + **`chainpulse` 二进制 blob（17M→29M）** + `scripts/codex-provider.sh` + `verify-docker-compose-stack.sh` | **保留**（含本机工具脚本；**二进制不应入库**） |

> **T00 不执行任何 commit / stash 操作**（除 B0 文档）。批次落地需**用户确认**后由 T01 之前完成——否则最终 SHA 会丢修复。
> **落地顺序建议**：B1→B2→B3（本项目修复，最高价值，先落）→ B0 → B4/B5 → B6/B7/B8。
> **禁止** `git add .`；每个批次用显式路径。

---

## 6. 验收工具链设计（F21，落点冻结）

计划 §8/§10 反复依赖三个工具，此前**无任务归属**。T00 冻结其落点与接口：

| 工具 | 落点 | 职责 | 归属任务 |
| --- | --- | --- | --- |
| **canonical digest oracle** | `test/acceptance/tools/digest/`（Go 包 + `cmd`） | 按 I02/I03 元组 `(chain,block_hash,tx_hash,log_index)` stable sort 聚合摘要；乱序输入摘要一致；自带测试 | **T02** |
| **tests.jsonl 检验器 + empty-run 检测** | `scripts/acceptance/verify-tests.py`（CI 直接消费） | 统计有 `Test` 字段的 pass/fail/skip、顶层 Test 名、package 成败、无测试包；检出 empty-run | **T01** |
| **证据收集器** | `scripts/acceptance/collect-evidence.sh` | 一次运行产出 §10 清单（`manifest.json`、`tests.jsonl`、`skip-report`、摘要、profile、run URL）；缺工件即失败；自测 | **T12** |

**T18 只消费，不手工拼装。**

---

## 7. 部署目标冻结（F22）

计划 §7.5 只写"一台已有可用 Linux/Docker 环境"，未指名。T00 需冻结：

| 选项 | 可得性（本机实测） | 判断 |
| --- | --- | --- |
| **本地 Docker 演示（推荐默认）** | Docker 运行中 ✅ | 可立即演练；CD 结论**不得**夸大为多环境部署 |
| 云 VM | 未知（需用户提供） | 需用户提供凭证 |
| 已有服务器 | 未知 | 需用户提供 |

**T00 建议**：冻结为 **本地 Docker 演示**（计划的明确 fallback）。若用户有云 VM/服务器，请提供连通性后升级目标。

---

## 8. 残余风险与阻塞

| # | 风险 | 影响 | 缓解 |
| --- | --- | --- | --- |
| R1 | **gh 无 admin scope**（F14） | T01 ruleset 无法配置 | 用户切换 `rtcdance` 账号或补 `admin:org`/repo admin |
| R2 | **关键修复未落 commit**（F18） | T18 静默丢修复 | 先执行 §5.2 批次 B1–B3 |
| R3 | **forge-std 缺失**（F15） | `evm-e2e` 无前置 | T02 vendoring |
| R4 | **Swap fixture 无来源**（F16） | T09 解码验收缺数据 | T02/T09 落实来源 |
| R5 | 规模 = 199h / 4–6 周 | 单会话无法完成 | 按 §6 节奏，任务级交付 |
| R6 | 全局 `~/.gitignore_global` 忽略 `docs` | 文档需 `git add -f` | 见 §9 |

---

## 9. 产物入库（本机全局 ignore 陷阱）

`~/.gitignore_global` 忽略整个 `docs/`。本轮文档需显式纳入：

```bash
git add -f docs/specs/2026-10-12-web3-go-backend-delivery-plan.md
git add -f docs/project/2026-10-12-web3-go-baseline.md
git add -f docs/project/2026-10-12-T00-scope-and-baseline.md
```

**禁止** `git add .`。

---

## 10. T00 验收清单

- [x] 一份支持矩阵（§3）
- [x] 一份 skip 分类表（唯一口径，§2）
- [x] 一份任务到证据映射（§4）
- [x] 一份 dirty→commit 批次计划（§5，F18）
- [x] 部署目标冻结记录（§7，F22 = 本地 Docker 演示）
- [x] 验收工具链清单与落点确认（§6，F21）
- [x] 规则源冲突清单（§1，F12）
- [x] **`git show HEAD:` 抽查确认关键修复已在提交历史中（F18 达成）**
  - `git show HEAD:pkg/siwe/siwe.go | grep -c VerifySIWEWithOrigin` → **2**
  - `git show HEAD:pkg/evm/typed_events.go | grep -c decodeSignedBigInt` → **5**
  - `git show HEAD:pkg/services/processor/idempotency.go | grep -c CheckAndMark` → **4**
  - `git status --porcelain` → **0**（工作区干净）；`git rev-list --count origin/main..HEAD` → **11**
- [ ] 部署目标连通性验证记录（本地 Docker 已运行 ✅；E15 时补真实验证记录）
- [x] 非本任务 dirty 文件未被重置（全部按批次 commit，未做 `reset`/`checkout`）

### 10.1 本会话落地的 11 个 commit（F18）

```text
aa641b9 chore(scripts): fix deploy-event-emitter debug hint path
e9df742 refactor: remove dead code, unused scaffolding and expired test skips
de3db0b chore(tooling): align frontend React types; add root Playwright config
1ab3118 chore(deps): align shared dependencies and toolchain to Go 1.26
206c8a4 docs(spec): add delivery plan, baseline and T00 scope artifacts
14f4736 docs: update README, guides and quickstarts; drop stale pkg/adapters refs
637c146 chore(ci): align workflow Go version to 1.26
99cf9f5 chore(rules): consolidate agent rules into AGENTS.md
a5fce33 fix(core): harden DLQ replay, reorg detection and idempotency
f914c71 fix(evm): decode signed int256/int24 via two's complement
a4f9d46 fix(siwe): validate domain, URI and chainID in SIWE verification
d364744 (origin/main) …
```

### 10.2 提交后门禁复跑（committed state）

| 门禁 | 结果 |
| --- | --- |
| `go build ./...` | exit 0 |
| `make check` | exit 0 |
| `make lint` | **0 issues** |
| `make test-short` | exit 0 / 66 ok / 0 FAIL |

### 10.3 提交钩子（新增环境事实）

`.git/hooks/pre-commit` → `scripts/pre-coding-checklist.sh` → `scripts/check-file-organization.sh`：对每个 tracked 文件跑 `file`（≈1min/commit），暂存 `.go` 时追加全量 `go build`+`go vet`。**提交必须后台执行**，否则撞工具 120s 超时。根目录白名单已补 `AGENTS.md`/`playwright.config.ts`；`requirements` 已移入 `docs/specs/`。
