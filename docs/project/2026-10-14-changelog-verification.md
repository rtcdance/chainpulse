# CHANGELOG 声明 vs 代码：逐条核查（计划 §10，2026-10-14）

> 起因：`2026-10-14-silent-revert-kafka.md` 发现 CHANGELOG 的一条声明（Kafka at-least-once）与代码不符。于是**把 "Unreleased" 段里可核查的声明逐条对照代码**。
> **结果：7 条中 5 条有据、1 条已失效（stale）、1 条被静默回退。**

---

## 1. 逐条结果

| # | CHANGELOG 声明 | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | API gateway upstream HTTP client: added missing **30s timeout** | ✅ **有据** | `pkg/plugins/api/request_router.go:17-18` `sharedHTTPClient{Timeout: core.DefaultTimeout}`，而 `core.DefaultTimeout = 30 * time.Second` |
| 2 | All puller goroutines: added **panic recovery** | ✅ **有据** | `pkg/plugins/pullers/data_puller.go:322` `if r := recover(); r != nil` |
| 3 | Event processor consume context: added **5-minute deadline** (was unbounded) | ⚠️ **已失效** | 见 §3：**曾被加上，后被 `c731756` 移除**；现在 `main.go:301` 是 `context.WithCancel`（无 deadline） |
| 4 | AuthAPIKeys migrated from `[]string` to **`[]core.SecretString`** | ✅ **有据** | `pkg/core/config.go` 用 `configmodel.SecretString`；`pkg/core/secret.go` 注释指向该类型 |
| 5 | **pprof** accessible via `CHAINPULSE_PPROF_ENABLED=true` | ✅ **有据** | `pkg/plugins/api/gateway.go:532` `if os.Getenv("CHAINPULSE_PPROF_ENABLED") != "true"` |
| 6 | **Per-endpoint latency scoring** for RPC failover routing | ✅ **有据** | `pkg/plugins/pullers/https_jsonrpc_puller.go:816` `EndpointHealthScores()` |
| 7 | Kafka delivery: at-most-once → **at-least-once** (manual offset commit) | ❌ **被静默回退** | 见 `2026-10-14-silent-revert-kafka.md` |

（"RED metrics wired into HTTP API gateway" 本会话早前已核实为**有据**——`observability.NewREDRecorder` 在 `https_jsonrpc_puller.go` 与 `gateway_router_integration.go` 中接线。）

## 2. 我自己的一个假阴性（值得记）

第一次核查第 1 条时，我 grep 的是：

```bash
grep -rn "30 \* time.Second\|Timeout: 30" --include="*.go" pkg/plugins/api/
→ (空)
```

于是差点判它"无据"。**错**：那个 30s 是**具名常量** `core.DefaultTimeout`，不是字面量。

⇒ **又一次"零命中型结论必须用第二种方法复核"**。本仓库的 AGENTS.md 与交接文档都写过这条，我仍然踩了。**负结论（"找不到"）比正结论更需要复核**——正结论有证据，负结论只是"我没找到"。

## 3. 第 3 条：不是"没做"，而是"做了又（正确地）去掉"

时间线：

| 日期 | 提交 | 事件 |
| --- | --- | --- |
| — | `8dc093b` `fix: runtime resilience — timeouts, panic recovery, context deadlines` | **加上** 5 分钟 consume deadline（CHANGELOG 说的就是这次） |
| **2026-05-28** | `c731756` `fix: kafka consumer context timeout in event-processor` | **移除**它：`- context.WithTimeout(..., 5*time.Minute)` → `+ context.WithCancel(...)` |

**关键点**：移除它的提交标题叫 "**fix** ... timeout"。而这一次**很可能确实是对的** —— 给一个**长期运行的消费者**加 5 分钟 deadline，意味着**每 5 分钟消费者就被掐断一次**，那是个 bug。

所以第 3 条的性质与第 7 条**不同**：

| | 第 7 条（Kafka） | 第 3 条（deadline） |
| --- | --- | --- |
| 怎么没的 | **被一个完全不提它的提交覆盖**（静默） | **被一个专门修它的提交移除**（有意） |
| 移除是否正确 | 否（回退了一个正确性属性） | **很可能正确**（5 分钟掐断长期消费者是 bug） |
| 问题在哪 | **代码**丢了修复 | **文档**没跟着撤声明 |

⇒ 第 3 条**不是代码缺陷**，是 **CHANGELOG 陈旧**：它在广告一个**已经被有意去掉**的行为。读者会以为有 5 分钟上限，实际没有。

## 4. 结论

1. **CHANGELOG 的 "Unreleased" 段有 2/7 条与代码不符**（1 条静默回退、1 条陈旧）。命中率足以说明：**这份 CHANGELOG 不能当作"代码现状"来读**。
2. 两份文件的写法差异很说明问题：`test/benchmarks/micro_baseline.json` 的 provenance **精确标注了主机、命令、样本数与"非参照环境"**（见 `2026-10-14-bench-gate.md` §3），而 CHANGELOG 是**从提交信息整理**的（`635d6a8` 自述 "Create CHANGELOG.md from git history"）——**没有回代码核对**。
3. **建议**（并入 D15）：CHANGELOG 每条"修复"类声明应能指向**断言它的测试**，而不是指向提交信息；无测试可指的，标注为"未验证"。

## 5. 未做

- 只核查了 "Unreleased" 段里**可机械核查**的条目；`[2026-04-09] Initial Release` 段的声明（如"7-chain support"、"Kubernetes manifests for production"、"Alertmanager configuration with Slack/PagerDuty templates"）**未逐条核查** —— 其中部分本会话已有旁证存疑（如告警未接线，见 T13 §4.1）。
