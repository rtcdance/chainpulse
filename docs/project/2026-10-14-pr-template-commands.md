# PR 模板要求的三条命令 + 一次未复现的 race 失败（计划 §10 清单项，2026-10-14）

> 对应计划 §10："**PR template的build/test-unit/vet实际通过；workspace所有module已build/vet/test。**"
> （module 部分已在 `2026-10-13-workspace-modules-verification.md` 单独覆盖。）

---

## 1. 三条命令的结果

PR 模板（`.github/PULL_REQUEST_TEMPLATE.md`）要求勾选：

| 命令 | 第一次（循环内） | 后续 |
| --- | --- | --- |
| `make build` | ✅ exit 0（22s） | — |
| **`make test-unit`** | ❌ **exit 2（82s）** | ✅ 重跑 exit 0（8s，走缓存） |
| `make vet` | ✅ exit 0（2s） | — |

**关键：`make test-unit` 并不是我以为的那条命令。**

```
Makefile:27   TESTFLAGS := -v -race -timeout 5m
Makefile:      test-unit:  $(GO) test $(TESTFLAGS) -short ./pkg/...
```

⇒ 它实际跑的是 **`go test -v -race -timeout 5m -short ./pkg/...`** —— **带 `-race`**。

我此前所有 `go test -short ./pkg/...` 的绿，都**没有**开 race。这是一个真实的覆盖差异：**race 模式下的行为与普通模式不同**。

## 2. 那次失败：观察到一次，但**无法复现**

- **观察到**：循环里第一次 `make test-unit` 在 82s 后 exit=2。当时的输出尾部是 `pkg/testhelpers` **PASS**，随后单独的 `FAIL`（`go test` 在某个包失败时打印的汇总行）+ `make: *** [test-unit] Error 1`。
- **⚠️ 但那次运行我没有保存完整日志**，所以**失败的包名未知**。这是本轮操作上的疏漏——发现失败时应当立刻留存完整输出。
- **复现尝试**：用**完全相同的命令**（`go test -v -race -timeout 5m -short -count=1 ./pkg/...`）连续跑了 **7 次**，**7 次全绿**（exit 0，无 `FAIL` 行）。证据：`build/acceptance/<sha>/race-flake/rerun-{1..7}.log`。
- 复现时的负载：14.30 / 14.95 / 12.13（交接文档提到 load≈15 时性能类测量会假阻塞）。

## 3. 怎么理解

**结论：`make test-unit` 存在一次未复现的失败。** 按证据强弱区分：

| 事实 | 强度 |
| --- | --- |
| `make test-unit` 跑的是 `-race` 模式 | **确证**（Makefile 原文） |
| 它失败过一次（exit=2，82s） | **确证**（循环输出） |
| 失败的是哪个包/哪个测试 | **未知**（未留完整日志） |
| 它是可复现的缺陷还是偶发 | **未知**（7 次未复现） |

**两种可能**，本文件**不做断言**：

1. **偶发**（load 相关或 race 时序敏感）—— 与仓库里另一个已记录的同类现象一致（`test/e2e` 的 `TestPropertyThroughputConsistency` 也是负载敏感的偶发失败）；
2. **稳定但低概率的竞态** —— 若如此，它会在 CI 里随机变红，正是 PR 门禁最怕的形态。

## 4. 建议（并入决策备忘录 D14）

- **优先**：给 CI 与本地都加一条**失败时留存完整日志**的约定（本轮就是因为没留而丢失了唯一的线索）。
- 在 `-race` 模式下**重复跑 N 次**（例如 10–20 次）以判定它是偶发还是低概率稳定竞态；必要时用 `GORACE` 提高检测灵敏度。
- 明确 `make test-unit` 是 PR 必需项 —— 那么它的稳定性就是**门禁的稳定性**，不能当作"偶尔红一下无所谓"。

## 5. 顺带确认

- **`make build` / `make vet` 均通过**（exit 0）。
- **ADR 索引一致**：`docs/adr/README.md` 列出 ADR-001…010，与目录里的 10 个文件逐一对应（文件名、日期、状态都对），并且**记录**了 ADR-007 原为第二个 ADR-004 的编号冲突已修正。
