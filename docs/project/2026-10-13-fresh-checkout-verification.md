# 新检出验证（计划 §10 清单项，2026-10-13）

> 对应计划 §10："**从最终SHA新checkout的build/check/lint/test-short按序通过，原始输出完整保存。**"
> 本轮执行了该条，并因此**发现并修复了一个我上一轮引入的真实缺陷**（见 §4）。

---

## 1. 执行条件

| 项 | 值 |
| --- | --- |
| 检出方式 | `git clone --local . /tmp/cp-fresh`（HEAD 的干净副本） |
| SHA | `d6b163ddf32f3b6054e4fd4674102e5d8300364f` |
| 工作树 | **0 个未提交改动**（与开发工作区的脏状态隔离） |
| Go | `go1.26.6 darwin/arm64` |
| `GOCACHE` | `/tmp/cp-fresh-gocache` —— **刻意用冷缓存**，避免开发机上残留的产物掩盖问题 |
| `GOPROXY` | `https://goproxy.cn,direct`（AGENTS.md 要求；`sum.golang.org` 本机不可达） |

脚本与原始日志：`build/acceptance/d6b163ddf32f/fresh-checkout/run.sh`、`verify.log`（`build/` 被 gitignore，属仓库既有约定）。

## 2. 结果（原始输出见 verify.log）

| 步骤 | 命令 | 退出码 | 耗时 |
| --- | --- | --- | --- |
| build | `go build ./...` | **0** | 33s |
| check | `make check`（vet + gofumpt） | **0** | 11s |
| lint | `make lint` | **0** | 4s |
| test-short | `make test-short` | **0** | 44s |
| | **OVERALL** | **0** | ~92s |

⇒ 计划 §10 该条**在 d6b163d 上通过**。

## 3. 但 `make lint` 的 "0 issues" 几乎什么都没证明

`make lint` 只用了 **4 秒**，这不合常理。原因：它带 `--new-from-rev=HEAD~1`，**只检查上一个提交的 diff**；而 `d6b163d` 的 diff 是一个 **Markdown 文档**，没有 Go 文件 ⇒ **一个 Go 文件都没检查**。这正是 AGENTS.md 反假绿规则 4 描述的情形。

于是按 AGENTS.md 给的**未限定范围**命令重跑：

```bash
"$(go env GOBIN)/golangci-lint" run --tests=false --timeout=10m ./pkg/... ./cmd/...
```

**它报出了一个真实缺陷**：

```
pkg/plugins/mq/kafka_mq.go:62:2: field commitInterval is unused (unused)
	commitInterval   time.Duration
1 issues:
* unused: 1
```

## 4. 该缺陷：我上一轮的提交信息说了假话

`commitInterval` 字段本应在 I06 修复（`7c05be4`）中删除，提交信息也写了 "Drop the now-meaningless commit-interval knob (…, **the commitInterval field**, …)"。**但它从未被删掉**——`git show` 显示该字段在 `7c05be4`、`674f47f`、`d6b163d` 三个提交里都还在。

**为什么没被发现**：`pre-commit` 钩子跑的是 build / vet / file-organization / 依赖审批，**不跑 golangci-lint**；而我当时只跑了 `go vet` 与测试，**没有跑 `make lint`**。一个未被引用的结构体字段不会让 build 或 vet 失败。

**修复**：删除该字段（本次提交）。修复后：

```
go build ./...            -> exit 0
make check                -> exit 0  (All checks passed)
golangci-lint (unscoped)  -> 0 issues
make test-short           -> 见本次提交
```

## 5. 结论与教训

1. **§10 该条：在 `d6b163d` 上通过**（冷缓存、干净检出、四条命令按序全绿）。
2. **"make lint 绿"不等于"仓库干净"** —— 它是 diff 限定的。要判断整树，必须跑 §3 的未限定命令。本次若只信 `make lint`，就会把一个未使用字段留在主干。
3. **pre-commit 不含 linter** ⇒ 改 Go 代码后必须**显式**跑 `make lint`（或未限定版）。这条我上一轮漏了，导致提交信息与实际改动不符。
4. 提交信息属于**证据**的一部分；说"删了"就必须用 `git show` 复核，不能凭编辑工具的回显。
