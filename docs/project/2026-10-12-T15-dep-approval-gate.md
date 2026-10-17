# T15 交付：依赖审批门禁（F17）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T15 / P1 / 8h**。
> 状态：**In Progress** —— F17 的**依赖审批门禁**已实现并有自测 + 真实历史验证；
> release/deploy workflow、不可变镜像 digest、SBOM 与漏洞扫描、checksum 覆盖、同 digest 提升未做（多数依赖 F22 的部署目标）。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. F17：一条没有强制力的规则

`AGENTS.md:111`：

> Dependency changes also require updating `docs/project/DEPENDENCY_APPROVAL.md`.

F17 记录："**该义务未出现在任何任务、§7 或 §10 DoD 中**"。T15 的验收把它变成了硬条件：

> **依赖变更而 `DEPENDENCY_APPROVAL.md` 未更新时 §10 DoD 不通过**

## 2. 实现

`scripts/acceptance/dep-approval-gate.py`（新，仅 Python 3 标准库 + **`--self-test`**）：

- **监控的清单**：`go.mod`、`go.sum`、`package.json`、`package-lock.json`、`frontend/{package.json,package-lock.json,pnpm-lock.yaml}`、**`contracts/foundry.toml`**（forge-std 与 solc 在此固定）；
- **判定**：有清单变更 **且** `docs/project/DEPENDENCY_APPROVAL.md` 未变更 ⇒ **FAIL**，并**点名具体清单**；
- **有清单变更且审批文档已更新** ⇒ PASS；
- **无清单变更** ⇒ PASS（"什么都没碰"不是失败）；
- 两种驱动方式：`--base/--head`（走 `git diff --name-only`）或 `--changed-files`（CI 可直接喂列表）。

**范围刻意收窄**：它**不判断**某个依赖是否可接受，只保证**有人记录了决定**。

## 3. 验收

### 3.1 自测

```
$ python3 scripts/acceptance/dep-approval-gate.py --self-test
dep-approval-gate self-test passed (9 cases + manifest naming)
```

9 个用例：无变更 / 无关文件 / **go.mod 无审批** / **go.sum 无审批** / **前端清单无审批** / **foundry 固定无审批** / go.mod + 审批 / 仅审批 / 无关文件 + 审批；另加"失败必须**点名**清单"的断言。

### 3.2 🔴 真实历史验证：**门禁抓到了一个真实的 F17 违规**

| 范围 | 结果 |
| --- | --- |
| `c2e3687..HEAD`（本会话 30+ 个提交） | **PASS** —— `no dependency manifest changed; nothing to approve`（与会话中每个提交的 "No dependency changes" 一致） |
| `206c8a4..1ab3118`（提交信息："chore(deps): align shared dependencies and toolchain to Go 1.26"） | **BLOCK** ↓ |

```
DEPENDENCY APPROVAL GATE FAILED (F17 / §10):
  docs/project/DEPENDENCY_APPROVAL.md was not updated, but these dependency manifests changed: go.mod, go.sum

Record the decision in docs/project/DEPENDENCY_APPROVAL.md (or revert the dependency change).
```

⇒ **`1ab3118` 实际违反了 F17**：改了 `go.mod`/`go.sum` 却没更新审批文档。门禁不是纸面规则——它在真实历史上就抓到了违规。

> 该历史提交**不做追溯性修改**（历史不可改写，且当时确实没有门禁）。门禁从**现在起**生效。

## 4. 未完成（T15 剩余）

- [ ] **release / deploy workflow**：同 SHA 全门禁 → 不可变镜像 digest → 暂存环境 → 真实 smoke → 提升。
- [ ] **单体与四服务两个目标**。
- [ ] **修 checksum 覆盖**：每个发布文件有正确 checksum。
- [ ] **缺服务镜像 / secret / 工件即失败**。
- [ ] **部署失败不报告成功**。
- [ ] **同 digest 提升**。
- [ ] **SBOM 与漏洞扫描**；"无可调用漏洞未处置"。
- [ ] **固定 action SHA、tool/image 版本**。
- [ ] **门禁接进 CI**：`dep-approval-gate.py` 目前**无人调用**。
- [ ] **F22 部署目标仍未冻结**（T00 遗留），这是上面多项的前置。
