# §10 清单：F18（关键修复在提交历史中）与 F21（验收工具链有代码与测试）（2026-10-14）

> 这两条是本会话此前**漏掉**的 §10 条目——之前的提取只覆盖了清单的一部分。补做如下。

---

## 1. F18：关键正确性修复已包含在最终 SHA 的提交历史中

计划原文："**关键正确性修复（SIWE/signed-int/幂等提交）已包含在最终 SHA 的提交历史中，非仅存在于工作区。**"

**方法**：用 `git show <sha>:<file>` 读**已提交树**（而非工作区）。工作树当前是干净的，但这个检查的意义在于**证明修复在提交历史里**。

| 修复 | 证据（读自已提交树） | 判定 |
| --- | --- | --- |
| **SIWE** domain/URI/chainID 校验 | `a4f9d46 fix(siwe): validate domain, URI and chainID in SIWE verification`；该提交在 `pkg/siwe/siwe.go` 新增 `VerifyOptions{Domain,URI,ChainID}` 与 `VerifySIWEWithOrigin`，注释明写这是 **EIP-4361 反钓鱼**检查（"VerifySIWE alone … accepts a message issued for *any* domain"） | ✅ |
| **signed-int** 解码 | `pkg/evm/typed_events.go` 有 `decodeSignedBigInt`（两补码），并被 `Amount0`/`Amount1`（int256）与 `Tick`（int24）调用；注释直指陷阱："Using `new(big.Int).SetBytes` here would silently turn -1 into 2^256-1" | ✅ |
| **幂等提交** | `pkg/services/processor/event_processor.go` 的 `idempotencyService` 字段、构造注入与 `Initialize` 调用均在已提交树中 | ✅ |

⇒ **F18 通过**：三项修复都在提交历史中。

## 2. F21：验收工具链有实际代码与测试

计划原文要求"**canonical digest oracle、tests.jsonl 检验器、证据收集器有实际代码与测试**"，且"**§10 证据包由收集器生成而非手工拼装**"。

### 2.1 逐工具自测（实际运行，非 grep）

| 工具 | `--self-test` |
| --- | --- |
| `verify-tests.py`（tests.jsonl 检验器 + empty-run 检测） | ✅ `self-test: OK` |
| `collect-evidence.py`（证据收集器） | ✅ `passed (15 required artifact specs)` |
| `bench-gate.py` | ✅ `passed (threshold 15%, min samples 5, 10 cases)` |
| `dep-approval-gate.py` | ✅ `passed (9 cases + manifest naming)` |
| `acceptance-gate.py`（本会话新增） | ✅ `self-test: OK` |
| `skip-report.py` | ✅ `all self-tests passed` |
| `chaos-report.py` | ✅ `all self-tests passed` |
| `openapi-drift.py` | ✅ `all self-tests passed` |
| **`govulncheck-summary.py`** | ❌ **原本没有自测** → **本轮补上（见 §2.3）** |

### 2.2 canonical digest oracle

`test/acceptance/tools/digest/`：`digest.go` + `digest_test.go`；`go test ./test/acceptance/tools/digest/` → **ok**。✅

### 2.3 补上缺的那个自测（`govulncheck-summary.py`）

9 个工具里只有它没有 `--self-test`。而它的 docstring 自己写着最危险的失效模式：

> "govulncheck emits a STREAM of pretty-printed JSON objects, not JSONL … Reading it line by line silently yields nothing, which is how a broken parser reports **"no vulnerabilities" from a scan that found many**."

**即：这个工具"坏掉"的表现是"报告干净"** —— 正是最需要自测的一类。已补 `self_test()`，10 项断言：

- **核心一条**：一段**多行 pretty-printed** 的假 govulncheck 流能解出 **5 个对象**（逐行解析会得到 0 个，正是 docstring 描述的陷阱）；
- callable 判定：**只有** trace 里带 `function` 的 finding 算 callable（2 条 finding → 1 条 callable）；
- 非 callable 的模块**不出现在** `by_module`；
- `toolchain_go_version` 从 `config` 正确读取；
- **空流解析为 0 条**（对应 `main()` 里"解析到 0 条即返回 1、拒绝报告干净"的 fail-closed 规则）。

实测：**`--self-test` 10/10 PASS**。

### 2.4 顺带的一次交叉验证

用改后的工具跑**真实** `govulncheck -json ./...`：

```
advisories considered : 295
callable advisories   : 0
callable findings     : 0
```

⇒ 仓库**自己的工具**独立确认了本会话的安全修复结果（可调用漏洞为 0），与 `govulncheck` 的结论一致。

## 3. 结论

| 条目 | 判定 |
| --- | --- |
| **F18** 关键修复在提交历史中 | ✅ 通过（SIWE / signed-int / 幂等，均读自已提交树） |
| **F21** 验收工具链有代码与测试 | ✅ 通过（9 个工具全部有可运行自测；oracle 有 Go 测试） |

## 4. 一个观察

计划把 F21 的验收写成"工具链**有实际代码与测试**"，而不是"有工具"。这条区分是对的：本会话已经反复证明**工具的失效方向是"报绿"**——空转的 required-cases 门禁、diff 限定的 `make lint`、承诺了不存在的 allow-list 的 skip 开关、以及这个"逐行解析会静默报告干净"的扫描汇总器。**要求每个门禁自带自测，是唯一能对抗这一点的做法。**

## 5. 仍未做的 §10 条目（本轮之外）

- **F15/F16**：`forge build` / `forge test` 与可追溯来源的 Swap fixture（需 Foundry）；
- **F13**：`ci-integration.yml` 归并或退役；
- **F14**：GitHub ruleset（需 admin）；
- **F22**：CD 部署目标冻结与 E15 实跑；
- 以及"同一最终 SHA 的 CI run URL / RC performance / 真实 CD 回滚"（远端）。
