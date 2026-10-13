# workspace 全模块验证（计划 §10 清单项，2026-10-13）

> 对应计划 §10 两条：
> - "**workspace所有module已build/vet/test**"
> - "**PR template的build/test-unit/vet实际通过**"（PR 模板要求 `make build` / `make test-unit` / `make vet`）
>
> 执行后得到两个结论：一个**结构性缺口**（标准命令根本不覆盖嵌套模块），一个**间歇性失败**（`test/e2e` 里一个对机器负载敏感的 property test）。
> 原始输出：`build/acceptance/e0da112dc2b8/workspace-modules/`（`modules.log`、`root-all-rerun.log`、`run.sh`）。

---

## 1. 结构性缺口：`./...` **不会**进入嵌套模块

`go.work` 声明 7 个模块：

```
.  pkg/blockchain  pkg/chainid  pkg/configmodel  pkg/histogram  pkg/logkeys  pkg/runtimetypes
```

但在仓库根目录执行 `go list ./...` 只得到 **89 个包**，其中来自 6 个子模块的包数是 **0**——因为含 `go.mod` 的目录会被 `./...` 模式排除。

**后果**：本仓库的常规命令

- `make test-short` → `go test -short ./pkg/... ./cmd/...`
- `go build ./...` / `go vet ./...`

**都不会编译或测试那 6 个模块**。它们的模块路径是独立的（如 `github.com/rtcdance/chainpulse/pkg/blockchain`），只能进入各自目录单独构建。本会话此前所有 `go build ./...` 也都漏掉了它们。

## 2. 逐模块结果（`e0da112`）

| 模块 | 包数 | build | vet | test (`-short`) |
| --- | --- | --- | --- | --- |
| `.`（根） | 89 | ✅ 0 | ✅ 0 | ❌ **1**（见 §3） |
| `pkg/blockchain` | 1 | ✅ 0 | ✅ 0 | ✅ 0 |
| `pkg/chainid` | 1 | ✅ 0 | ✅ 0 | ✅ 0 |
| `pkg/configmodel` | 1 | ✅ 0 | ✅ 0 | ✅ 0 |
| `pkg/histogram` | 1 | ✅ 0 | ✅ 0 | ✅ 0 |
| `pkg/logkeys` | 1 | ✅ 0 | ✅ 0 | ✅ 0 |
| `pkg/runtimetypes` | 1 | ✅ 0 | ✅ 0 | ✅ 0 |

⇒ **6 个子模块全部通过 build/vet/test**；根模块的失败来自 `test/e2e`（下节）。

## 3. 间歇性失败：`test/e2e` 的吞吐 property test

根目录 `go test -short ./...` 会把 **`test/e2e`** 也纳入（而 `make test-short` 只跑 `./pkg/... ./cmd/...`，**不含**它）。第一次全量运行失败：

```
--- FAIL: TestPropertyThroughputConsistency (0.22s)
    performance_manager_property_test.go:224: run 4: throughput 1477.38 deviates -75.4% from average 6007.88
FAIL	github.com/rtcdance/chainpulse/test/e2e	102.432s
```

**该测试做什么**（`test/e2e/performance_manager_property_test.go:196`）：跑 10 轮 `BenchmarkEventProcessing`，取每轮的**墙钟吞吐**，要求每轮与均值的偏差在 **±50%** 以内。

**为什么它会假失败**：它断言的是**调度器的行为**，不是代码的性质。机器有负载时某一轮被抢占，吞吐就会掉到均值的一半以下。实测：

| 观测 | 结果 |
| --- | --- |
| 单独运行该测试 | **6/6 PASS** |
| 全量 `./...` 第 1 次（并行跑各包） | **FAIL**（`-75.4%`） |
| 全量 `./...` 第 2 次 | **PASS** |
| 当时的 load average | **7.39 / 7.78 / 9.15** |

⇒ 间歇性、负载敏感，**不是被测代码的缺陷**。注意它是"假失败"而非"假通过"，所以与仓库既有的 F26/F27（假通过）不同类；但它让 `go test ./...` 不可复现。

**它不在标准门禁里**：`make test-short` 不含 `test/e2e`，CI 的 e2e 是独立 job（`make test-e2e`）。所以这个 flake 只在有人用根 `./...` 时才咬人——而计划 §10 的这条恰恰要求"所有 module"。

## 4. 结论与建议

1. **§10 "workspace所有module已build/vet/test"**：**部分满足**——7 个模块的 build/vet 全绿、6 个子模块的 test 全绿；根模块 test 因 §3 的 flake 间歇失败。**若按字面用 `go test ./...` 判定，该条无法稳定通过。**
2. **建议（需决策）**：`TestPropertyThroughputConsistency` 断言的是机器负载而非代码性质。可选：
   - (a) 在 `-short` 下跳过（`-short` 的语义正是"跳过慢/负载敏感用例"），全量模式保留；
   - (b) 改为对**代码可控的性质**断言（例如吞吐为正、且各轮都完成了预期的事件数），去掉墙钟比较；
   - (c) 保留现状，但把 §10 该条明确为"逐模块"而非"根 `./...`"。
   - 我**没有**擅自改它：调整测试阈值/断言属于"降低指标目标"的邻近操作，计划对此有明确纪律（"没有人为提高/降低指标目标"），应由作者定。
3. **PR 模板的 `make build` / `make test-unit` / `make vet`**：三者覆盖的是根模块（`make test-unit` = `./pkg/...`），**同样不含** 6 个嵌套模块——与 §1 是同一个缺口。
