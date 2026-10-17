# lane 实测时长 vs §7.3 预算（计划 §10 清单项，2026-10-14）

> 对应计划 §10："**各 lane 实测时长在 §7.3 预算内；cache 未命中仍在预算内完成。**"
> 预算原文（§7.3）：`quality` ≤5min、`unit-race` ≤10min、`contracts` ≤15min、`evm-e2e` ≤25min、`frontend-api` ≤10min、`security` ≤15min、`coverage` ≤15min。
>
> **结论：所有可测 lane 均在预算内，且留有大量余量。** 证据：`build/acceptance/<sha>/lane-durations/`。

---

## 1. 实测结果

| lane | 预算 | 实测 | 判定 | 命令 |
| --- | --- | --- | --- | --- |
| `quality` | 300s | **4s** | ✅ | `make check`（vet + gofumpt） |
| `unit-race` | 600s | **74s** | ✅ | `go test -race -short -count=1 ./pkg/...` |
| `contracts` | 900s | **12s** | ✅ | `go test -tags=integration -count=1 ./test/contracts/...`（PG/Redis 门开） |
| `coverage` | 900s | **48s** | ✅ | CI 原命令 + `-count=1`（见 §2） |
| `security` | 900s | **14s** | ✅（时长） | `govulncheck ./...` |
| `evm-e2e` | 1500s | ~**102s**（代理） | ✅ | 本地无 Anvil；用 `test/e2e` 全包时长代理 |
| `frontend-api` | 600s | **未测** | — | 需已构建的前端 + 运行中的 API（装配未完成，见 D5/D6） |

余量最小的 `unit-race` 也只用了预算的 **12%**（74s / 600s）。

## 2. 我自己测量时踩的一个坑（值得记）

第一次跑 `coverage` 只用了 **4 秒**——因为 `go test` 会**缓存结果**，而我那条命令**没加 `-count=1`**，命中缓存直接返回。同一命令加 `-count=1` 后是 **48 秒**。

⇒ **"cache 未命中仍在预算内"这条不能靠开发机的缓存结果来证明**：开发机通常热缓存，会给出远低于 CI 的数字。要证明预算，必须 `-count=1`（或冷缓存）。这与 AGENTS.md 里"热模块缓存会掩盖 GOPROXY 问题"是同一类陷阱：**本机绿不等于 CI 绿**。

## 3. `govulncheck` 的退出码语义

`security` lane 的退出码是 **3**，但那**不是命令失败**：govulncheck 用 3 表示"**发现了漏洞**"。

- **时长**：14s，远在 15min 预算内 ✅
- **结论**：该 lane 目前会**因发现而红** —— 实测 **14 条**（stdlib 已为 0，剩余全为第三方：x/net、otel、mongo-driver、pgx）。
- 计划 §10 对 security 的要求是"**可调用漏洞及新严重发现无未处置项**"，即这些需要**逐条处置**（升级或书面接受），而不是把 lane 判成"命令失败"。

## 4. 未覆盖与边界

| 项 | 状态 |
| --- | --- |
| `frontend-api` | **未测**：需前端构建产物 + 运行中的 API，装配未完成（D5/D6） |
| `evm-e2e` | **代理测量**：计划要求双 Anvil 链；本地用 `test/e2e` 全包时长（~102s）作上限代理 |
| 计划 lane 名 vs 实际 job 名 | **不一致**（见 `2026-10-13-ci-gate-audit.md`）：这些名字目前都不是 CI job，所以本表是"**按等价命令**"的测量，不是"按 job"的测量 |

## 5. 结论

- **§10 "各 lane 实测时长在 §7.3 预算内"：满足**（就所有可测 lane 而言，且余量充足）。
- **"cache 未命中仍在预算内完成"：本机无法完全证明** —— 开发机是热缓存；本表已用 `-count=1` 消除测试结果缓存的影响，但 Go **构建缓存**（`GOCACHE`）仍是热的。严格证明需要在 CI（冷缓存）或冷 `GOCACHE` 上复测；本会话的"新检出验证"已用冷 `GOCACHE` 跑过 build/check/lint/test-short，可作旁证。
- **security lane 会红**：不是时长问题，是 14 条第三方漏洞待处置。
