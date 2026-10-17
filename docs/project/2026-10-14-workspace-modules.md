# §7.1 workspace 完整性：6 个嵌套模块从未被 build/vet/test

- 日期：2026-10-14
- 对应 §10 行：*"PR template的build/test-unit/vet实际通过；**workspace所有module已build/vet/test**"*
- 直接对应 **§7.1**：*"根目录 `./...` **不等于**整个 Go workspace。从 `go work edit -json` **动态枚举**每个 `.Use[].DiskPath`，对每个 module 执行 build/vet/test；根模块另外执行 cmd 测试。CI 检查 go.work 的 go 指令不低于任一模块"*，以及 *"全量 lint 要覆盖实际 workspace 模块（先动态枚举再在模块执行）……**不能靠根命令未进入子模块来完成**"*
- 证据：`build/acceptance/<sha>/workspace-modules/`

## 结论

`go.work` 里有 **7 个模块**（根 + 6 个嵌套）。但**没有任何东西枚举过它们**：

```
grep -rn "go work edit" --include="*.sh" --include="*.yml" --include=Makefile   →  空
```

而 CI 里**每一条** build/vet/test 用的都是 `./...` 或 `./pkg/...`——**两者都在当前模块边界停下**：

| CI 步骤 | 命令 | 实际覆盖 |
| --- | --- | --- |
| Format check | `gofmt -l .` | 全树（按文件系统遍历）✓ |
| Vet | `go vet ./...` | **只有根模块** |
| Test (short) | `go test -race -short ./...` | **只有根模块** |
| `make check` | `vet fmt-check` | **只有根模块** |
| `make test-short` | `./pkg/... ./cmd/...` | **只有根模块** |

⇒ `pkg/blockchain`、`pkg/chainid`、`pkg/configmodel`、`pkg/histogram`、`pkg/logkeys`、`pkg/runtimetypes`
**从未在 CI 里被 build、vet 或 test 过**。（我上一轮给它们加了 govulncheck 扫描——所以"被引用过一次"，
但那是漏洞扫描，不是 build/vet/test。）

这**看起来**是覆盖的，因为读者检查的正是那条根命令——§7.1 那句"**不能靠根命令未进入子模块来完成**"
说的就是这个。

（6 个模块的测试本地跑都是绿的：`ok ... 0.37s~1.05s`。所以这不是"坏了"，是"没人跑"。）

## 修复

新增 **`scripts/check-workspace-modules.sh`**，按 §7.1 的做法实现：

1. **动态枚举**：`go work edit -json` → `.Use[].DiskPath`。**不用硬编码列表**——
   硬编码会在有人新增模块的那一刻静默漏掉它，而那正是这个检查存在的理由。
2. **逐模块** `go build ./...`、`go vet ./...`、`go test -short -count=1 ./...`。
3. **`go.work` 的 go 指令不得低于任一模块**。
4. 根模块的 `./cmd/...` 由 `./...` 覆盖，脚本里注明而不是重复跑。
5. 汇总所有失败并 **exit 1**；`--no-test` 只跑 build+vet。

接线：`make workspace-check` + `ci.yml` 的 build job 新增一步（§7.3 的 `quality` lane 就是
"workspace build、gofumpt、vet"）。**根模块的标准口径结果被保留**——脚本把 `.` 当作模块之一，
在里面跑的就是同一条 `go vet ./...`。

**实测**（真实仓库）：
```
workspace go directive: 1.26.9
modules discovered: 7
  . pkg/blockchain pkg/chainid pkg/configmodel pkg/histogram pkg/logkeys pkg/runtimetypes
...
workspace check passed: 7 modules built, vetted and tested
exit=0  elapsed=172s
```

**反例路径**（在 /tmp 的沙箱 workspace 里验，不动仓库）：
```
clean, consistent workspace      -> exit 0
go.work 1.26.0 < module 1.26.9   -> exit 1  "go.work go 1.26.0 is lower than ./a's go 1.26.9"
one module fails to build        -> exit 1  "./b: go build ./... failed"
```

## 我自己写错并被反例抓住的一处

第一版的方向比较是反的：

```bash
printf '%s\n%s\n' "$work_go" "$mod_go" | sort -V | head -1   # ← 取最小的那个
```

它会把 **`1.26.9` 报成"低于" `1.26.0`**。改成取 `tail -1`（升序排完最后一个才是较大的那个）。
**如果我只跑正向路径（仓库里指令本来就一致），这个 bug 会一直藏着**——它是被"造一个不一致的沙箱"
逼出来的。这也是本会话反复出现的同一件事：**正向结果不构成证据**。

## 诚实边界

1. **`go.work` 指令这条要求，Go 工具链本身已经在强制**：把 `go.work` 调低会让**每个**模块的构建
   都以 `module X listed in go.work file requires go >= Y` 失败。所以脚本里那条显式检查提供的是
   **清晰的诊断**（指出是哪个模块、差多少），**不是新的强制力**。我把它留着是因为
   "所有模块都构建失败"与"go.work 指令落后"是两种截然不同的读者体验。
2. **耗时 172s**：§7.3 的 `quality` lane 预算 300s，它在预算内，但占了**一半以上**。
   其中绝大部分是根模块的 `go test -short`。若将来超预算，第一选择是把根模块的测试交给
   `unit-race` lane（§7.3 本来就是这么分的），只在这里留 build+vet。
3. **lint 未按模块跑**：§7.1 说"全量 lint 也要先动态枚举再在模块执行"。本轮只做了 build/vet/test；
   `golangci-lint` 的逐模块执行**未做**（§7.1 提到 `--tests=false` 全量口径，属封版动作）。
4. **workflow 未在 GitHub 上跑过**：验的是 YAML 可解析 + 脚本在真实仓库与沙箱中按预期退出。
5. **`make check` 本身没改**：它仍是 `vet fmt-check`（根模块）。我把新检查做成独立目标
   `workspace-check`，因为把 172s 塞进 `check` 会让 §7.1 要求"按顺序跑"的那组本地门禁变慢。

## 复现

```bash
# 完整（build+vet+test，7 个模块）
make workspace-check

# 只要 build+vet
bash scripts/check-workspace-modules.sh --no-test

# 证明根命令够不到子模块
go list ./... | grep -c pkg/histogram        # 0
(cd pkg/histogram && go test -short ./...)   # 这里才有
```
