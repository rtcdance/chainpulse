# CI 门禁现状核查（计划 §10 清单项，2026-10-13/14）

> 对应计划 §10 两条：
> - "**CI反例全红、正常路径全绿；acceptance-gate阻断skipped/neutral/missing。**"
> - "**GitHub ruleset/branch protection 已配置且可 `gh api` 复核；acceptance-gate 为 main required check，缺 check 的 PR 合并被拒（F14）。**"
>
> **结论：两条都无法满足** —— `acceptance-gate` 这个 job **不存在**，ruleset **为空**，且分支过滤与默认分支不匹配。以下为可复核的实测证据。

---

## 1. `acceptance-gate` 不存在；计划 §7.3 的 lane 名一个都没实现

计划 §7.3 定义了一组 lane 名并要求它们成为 required check。实际 workflow 里的 **job 名**是：

| workflow | jobs |
| --- | --- |
| `ci-integration.yml` | `integration`、`contract`、`e2e`、`notify` |
| `ci.yml` | `build` |
| `coverage.yml` | `coverage`、`coverage-diff` |
| `deploy.yml` | `verify`、`build`、`deploy-staging`、`deploy-production`、`rollback` |
| `metrics.yml` | `collect-metrics`、`performance-regression-check` |
| `release.yml` | `test`、`build`、`docker`、`release` |
| `test.yml` | `test`、`lint`、`build` |

逐一核对计划里的 lane 名：

| 计划 lane | 是否为 job | 说明 |
| --- | --- | --- |
| `quality` | ❌ | 只在 `deploy.yml` 里作为**步骤名** "Check code quality" 出现 |
| `unit-race` | ❌ | 0 处 |
| `contracts` | ❌ | 只在**输出文件名**（`contracts-output.jsonl`）里出现；job 叫 `contract`（单数） |
| `evm-e2e` | ❌ | 0 处 |
| `frontend-api` | ❌ | 0 处 |
| `security` | ❌ | 0 处 |
| **`acceptance-gate`** | ❌ | **0 处** |

⇒ 计划 §7.3 的 lane 词汇**从未落地**；`acceptance-gate` 这个"汇总并阻断 skipped/neutral/missing"的 job **不存在**。T01 文档也自述其为"**待建（F14 阻塞）**"。

最接近的 `notify`（`ci-integration.yml`）**不是** gate：它 `if: failure() && github.event_name == 'schedule'`，只在**夜间**失败时开一个 issue，**不汇总 required checks，也不会因 skipped/neutral/missing 而失败**。

## 2. ruleset 为空、分支保护未配置、账号无 admin（`gh api` 实测）

```
$ gh api repos/:owner/:repo/rulesets
[]

$ gh api repos/:owner/:repo --jq '.full_name, .default_branch, .permissions.admin'
rtcdance/chainpulse
master
false

$ gh api repos/:owner/:repo/branches/main/protection
{"message":"Not Found", ... "status":"404"}

$ gh api repos/:owner/:repo/branches/main/protection/required_status_checks
{"message":"Not Found", ... "status":"404"}
```

- **ruleset 列表为空** ⇒ 没有任何 ruleset（F14 的"无门禁载体"得到复核）。
- **`permissions.admin = false`** ⇒ 当前活动账号（`codex689`，token scopes 仅 `admin:public_key, gist, read:org, repo`）**不是仓库管理员**，无法配置 ruleset/branch protection —— 与 T01 §76 行的记录一致。
- **`main` 的 protection 返回 404** ⇒ 未配置分支保护；required status checks 同样 404。
- 注意：**`rulesets` 返回 `[]` 而不是 403/404**，说明**有读权限且确实为空**，不是权限不足导致的假象。

⇒ 计划 §10 的 "ruleset 已配置且可 `gh api` 复核" 被 `gh api` **直接证伪**。

## 3. 默认分支是 `master`，但 4 个 workflow 不监听它

```
$ gh api repos/:owner/:repo/branches --jq '.[].name'
codex/acceptance-wrapup-2026-04-09
codex-docker-acceptance-runtime-wiring
main
master
```

**`default_branch = master`**。而各 workflow 的分支过滤：

| workflow | `push`/`pull_request` 分支过滤 | 含 `master`？ |
| --- | --- | --- |
| `ci-integration.yml` | `[ main, master, develop ]` | ✅ |
| `test.yml` | `[ main, develop ]` | ❌ |
| `coverage.yml` | `[ main, develop ]` | ❌ |
| `metrics.yml` | `[ main, develop ]` | ❌ |
| `ci.yml` | push `[main, codex/acceptance-wrapup-2026-04-09]`；PR `[main]` | ❌ |
| `deploy.yml` / `release.yml` | 仅 tag `v*` | n/a |

⇒ **`test.yml`（主测试门禁）、`coverage.yml`、`metrics.yml`、`ci.yml` 都不会在默认分支 `master` 上触发**。若把 PR 的目标分支设为默认的 `master`，主测试门禁**不会跑**，而 GitHub 仍会显示"可合并"（因为没有 ruleset 要求 check）。

## 4. 对计划 §10 的影响

| §10 条目 | 判定 |
| --- | --- |
| CI 反例全红、正常路径全绿；`acceptance-gate` 阻断 skipped/neutral/missing | **无法满足** —— gate 不存在 |
| ruleset/branch protection 已配置且可 `gh api` 复核 | **被证伪** —— `rulesets` 为空、protection 404 |
| `acceptance-gate` 为 main required check，缺 check 的 PR 合并被拒 | **无法满足** —— 无 ruleset，且账号无 admin |

## 5. 需要什么

1. **仓库 admin 权限**（当前活动账号没有）—— 这是 F14 的硬前提；T01 已记录 `codex689` 仅 `repo` scope，仓库另有 `rtcdance` 账号可试。
2. **建 `acceptance-gate` job**：`needs` 全部 required jobs，`if: always()`，逐条断言结论为 `success`，把 `skipped/neutral/cancelled/missing` 一律判失败。
3. **对齐分支**：要么把默认分支改为 `main`，要么把 4 个 workflow 的分支过滤补上 `master`。二者择一，否则主门禁在默认分支上不跑。
4. **统一 lane 名**：计划 §7.3 的名字与现有 job 名不同；ruleset 依赖 check 名，改名必须同步 ruleset（计划 §7.3 已声明此纪律）。

> 本文件只记录核查结果与证据，未改动任何 workflow。相关决策已并入 `2026-10-13-decision-requests.md`（D12）。
