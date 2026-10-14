# CI 反例验收（§7.4）：门禁是否真的会红

- 日期：2026-10-14
- 对应 §10 行：*"CI反例全红、正常路径全绿；acceptance-gate阻断skipped/neutral/missing"*
- 对应 §7.4 表（7 行注入）
- 证据：`build/acceptance/<sha>/negative-cases/{matrix.json,matrix.txt}`；`build/acceptance/<sha>/security/`
- 工具：`scripts/acceptance/negative-cases.py`（新增，21 个用例）

## 做了什么

§7.4 要求"注入故障 → 对应 job 必须红"，并保存失败 run URL。run URL 需要 GitHub；**判定逻辑不需要**。
所以我把能在本地验的那一半做实了：对每一行注入，给仓库自己的验收工具喂**对抗输入**，断言它 **exit≠0**；
再喂对应**干净输入**，断言它 **exit=0**。

**结果：21/21 全部符合预期。**

```
result  expect  case                                   tool
ok      fail    assertion-failure                      verify-tests.py
ok      fail    race-detector-failure                  verify-tests.py
ok      fail    package-fail-without-FAIL-line         verify-tests.py
ok      fail    run-matches-zero-tests                 verify-tests.py
ok      fail    required-case-never-ran                verify-tests.py
ok      fail    required-skip                          verify-tests.py
ok      pass    control-happy-path                     verify-tests.py
ok      pass    control-required-case-present          verify-tests.py
ok      pass    control-no-test-package               verify-tests.py
ok      fail    prior-check-skipped                    acceptance-gate.py
ok      fail    prior-check-neutral                    acceptance-gate.py
ok      fail    prior-check-missing                    acceptance-gate.py
ok      pass    control-all-checks-success             acceptance-gate.py
ok      fail    perf-artifact-empty                    bench-gate.py
ok      fail    perf-artifact-nan                      bench-gate.py
ok      fail    perf-regression-over-budget            bench-gate.py
ok      pass    control-perf-within-budget             bench-gate.py
ok      fail    dependency-changed-without-approval    dep-approval-gate.py
ok      pass    control-dependency-approved            dep-approval-gate.py
ok      fail    security-artifact-undecodable          govulncheck-summary.py
ok      fail    real-build-failure-no-FAIL-line        verify-tests.py
        detail: go test exit=1, verifier exit=1
```

**为什么每个反例都配一个正向对照**：一个"永远失败"的工具能满足所有 `expected: failure` 行却毫无用处。
对照必须通过，否则矩阵不成立。

**为什么不直接信工具自带的 `--self-test`**：因为自测可能与工具**错在同一处**。
本会话已经多次撞到这类问题（一个门禁的测试恰好断言了它已有的行为）。所以本脚本只从**外部**走 CLI，只读 exit code。

## 矩阵发现的缺陷：包级失败被忽略（已修）

跑矩阵时，`package-fail-without-FAIL-line` 这一格**不符合预期** —— 它没有红。这正是 §7.4 单独点名的那一行：

> | timeout / panic / go test exit 1但没有 `--- FAIL` | **failure，不依赖文本 grep** |

`verify-tests.py` 的 `compute_failures` 只检查**带 `Test` 字段**的失败事件。
而"包编译失败 / 测试外的 panic / TestMain 非零退出 / 超时被杀"这四种情况，
`go test` 退出码非 0、JSONL 里只有**包级** `fail`，**没有任何 `--- FAIL`**。
包级结果其实**已经被解析并打印**（"no-test packages"），只是**从未被用于判定**。

**真实复现**（不是构造 JSONL）：两个包，一个正常、一个不编译：

```
$ go test -json -count=1 ./...
go test exit=1
{"Action":"build-fail"}                                   ← 编译失败
{"Action":"fail","Package":"negprobe/broken","FailedBuild":"..."}   ← 包级 fail，无 Test 字段
{"Action":"pass","Package":"negprobe/good","Test":"TestGood"}       ← 另一个包有 Test 事件
```

**修复前**：

```
tests: pass=1 fail=0 skip=0 top_level=1 packages=2
no-test packages (1):
  - negprobe/broken
✓ verification passed          ← exit 0
```

**修复后**：

```
✗ verification FAILED (1 issue(s)):
  - package failed: negprobe/broken
verifier exit=1
```

**修复**：`compute_failures` 新增一步，任一包的包级结论为 `fail` 即判失败；并补两条自测
（包级失败必须红 / 包级 pass 与无测试包**不得**红）。

**为什么这个修复是安全的**（我先验证再改）：`go test ./...` 对"被 build tag 完全排除"的包是
**整体省略**而不是报失败：

```
go list ./...                      → 89 个包（test/contracts、test/acceptance 根、test/migration 不在内）
go list -tags=integration ./...    → 91 个包（多出 test/contracts、test/acceptance）
```

所以任何 CI 调用都不会因为 tag 而误触发。**回归**：把 `build/acceptance/` 下**全部约 60 份归档 JSONL**
逐份重跑（含 `unit.jsonl`：5170 个顶层用例、**74 个包**）—— **全部 exit 0**，无一次误报。

## 矩阵同时发现的缺陷：安全扫描**不可能失败**（已修）

查 `govulncheck` 在 CI 里的位置时发现，`ci.yml` 的 build job 里：

```yaml
      - name: Security scan
        continue-on-error: true      # ← 扫出任何漏洞都无法让 job 红
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
```

§7.3（line 311）明确写着：

> 测试步骤**不使用 `continue-on-error`**；必要的诊断可 `always()` 上传，但不能改变 gate 结果。

即：**这一条被直接违反**。同一段还有两个问题：

1. **只扫根模块。** 仓库有 **6 个嵌套模块**（`pkg/configmodel`、`pkg/logkeys`、`pkg/blockchain`、
   `pkg/runtimetypes`、`pkg/histogram`、`pkg/chainid`，各有自己的 `go.mod`），
   根目录 `./...` **够不到它们**（已用 `go list` 逐个确认）。§7.3 要求的是"**各模块** govulncheck"。
2. **`@latest` 未固定版本**，与"工具版本…可追溯"不符。
3. **`govulncheck-summary.py`（§10 安全工件的生产者，本轮还给它补了自测）从未被任何地方调用。**

**修复**：去掉 `continue-on-error`（改为捕获退出码 → 先产出 §10 工件 → 再按退出码失败 job）；
版本固定为**本机已验证的 `v1.8.0`**；循环扫描 6 个嵌套模块；接入 summary 工具产出工件。

**本地逐条验证修复后的序列**（`set -euo pipefail` 下）：

```
root govulncheck exit=0
wrote build/acceptance/security/security/summary.json
advisories considered : 295
callable advisories   : 0
--- submodules ---
  pkg/configmodel: exit=0 | No vulnerabilities found.
  pkg/logkeys:     exit=0 | No vulnerabilities found.
  pkg/blockchain:  exit=0 | No vulnerabilities found.
  pkg/runtimetypes: exit=0 | No vulnerabilities found.
  pkg/histogram:   exit=0 | No vulnerabilities found.
  pkg/chainid:     exit=0 | No vulnerabilities found.
```

⇒ **7 个模块全干净**，所以去掉 `continue-on-error` **今天不会让 CI 变红**；它只在**将来出现新可调用漏洞时**才红——
这正是 §7.3"新严重项阻断"想要的行为。7 个 workflow 的 YAML 全部仍可解析。

## 诚实边界

- **run URL 未取得**：本机无 GitHub，§7.4 要求的"保存失败 run URL"这一半未做。
  本地做的是**判定逻辑**那一半（21/21）。
- **§7.4 的 release/deploy 行只覆盖了判定逻辑**：`acceptance-gate` 的 skipped/neutral/missing 三例已验，
  但"CD credential 缺失 / smoke 失败"需要部署目标。
- **"已审定基线"机制未做**：§7.3 的 security lane 写的是"按**已审定基线**"，
  即允许把已知可调用漏洞登记为已处置。目前 `govulncheck` 没有基线机制（`bench-gate` 有）。
  去掉 `continue-on-error` 后，任何新可调用漏洞都会直接阻断——这是计划字面要求，但"基线"这层
  是设计决策，记为 **D17**。
- **security lane 仍是 ci.yml 里的一个 step，不是独立 required check**。§7.3 把 `security` 列为独立 lane。
  新建 lane 涉及 job 命名冻结（T15）与 ruleset（F14，需 admin），不在本轮单方面改动。

## 复现

```bash
# 反例矩阵（21 个用例）
python3 scripts/acceptance/negative-cases.py

# 单独看那一格
python3 scripts/acceptance/negative-cases.py --list | grep package-fail

# 安全扫描全模块
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
govulncheck -json ./... > /tmp/gv.json
python3 scripts/acceptance/govulncheck-summary.py --input /tmp/gv.json
for m in pkg/configmodel pkg/logkeys pkg/blockchain pkg/runtimetypes pkg/histogram pkg/chainid; do
  (cd "$m" && govulncheck ./...)
done
```
