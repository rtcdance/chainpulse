# 待决事项（2026-10-13）

> 用途：把计划剩余工作**收敛成几个必须由作者拍板的决策**。
> 背景：本轮逐个核查了交接文档 §6 的剩余项，结论是**它们都不缺实现，缺的是装配**；而每一处装配都卡在一个设计/策略决策或一条缺失的依赖链上。逐条列出，便于一次拍完。
> 相关证据：`2026-10-13-handover.md`（§6 剩余清单）、`2026-10-13-write-path-findings.md`（§4 栈选择、§10 F05）、`2026-10-13-unwired-components-audit.md`（163 个未接线构造器）。

---

## D1. 生产栈到底是哪一套？（最高优先，阻塞 D2）

- **问题**：端口栈（`core.DatabasePlugin` + PG 适配器 + `BatchCommitter`/fenced lease）与 service-store 栈（Mongo 事件 + PG 元数据，四服务实际使用）**哪一套是生产栈**？
- **为什么阻塞**：write-path-findings §4 给了 A/B/C 三选一。选 A 要把四服务的 processor 装配重写；选 B 要在 service-store 上重做 T05 的原子提交语义（跨库需 outbox/saga）；选 C 要下修 §8/§10 的多项承诺。
- **不决策的后果**：T02/T03/T05/F05 永远停在"端口级已验证"，而部署路径的半提交窗口一直开着。

## D2. F05 的接口怎么补 chain 维度？（依赖 D1）

- **问题**：`AtomicBatchCommitter.CommitBlock` 无 `chainID`，而 `BatchCommitter` 按单链构造、lease 用构造时 chainID；`RuntimeDeps.Committer` 只有一个，运行时却服务多链。
- **选项**：(a) 给 `CommitBlock` 加 `chainID` 形参（`BatchCommitter` 改为按调用解析 lease）；(b) 把 `RuntimeDeps.Committer` 改为工厂 `func(chainID) (AtomicBatchCommitter, error)` 或加按链分发的薄 dispatcher。
- **我的倾向**：**(b)** —— `BatchCommitter` 及其 10 个测试与 fenced 语义都不用动。
- **证据**：write-path-findings §10。

## D3. `DrainController`（F24）放在哪一层？

- **问题**：`DrainController` 在 `pkg/application/bootstrap/drain.go`，而要用它的批次处理在 `pkg/services/indexing/chain_indexer.go`。services 直接 import application 是分层倒置。
- **选项**：(a) 把 `DrainController` 下移到 `pkg/services` 或 `pkg/core`（一个接口 `WorkGate`）；(b) 在 `pkg/application/indexing` 定义接口、由 bootstrap 注入实现。
- **注意**：F24 的语义已定（SIGTERM → 停止接新活 → 等 in-flight 批次提交完 → 超时要**报出来**而不是吞掉）。只差落位与启动/停止接线。

## D4. 单体路径要不要引入 event bus？（阻塞 T06 的 outbox drainer）

- **问题**：`OutboxDrainerLoop` 需要一个 `NotificationPublisher`，现成实现是 `EventBusPublisher`（适配 `core.EventBus`）。但**单体 main.go 里根本没有 event bus**。
- **选项**：(a) 给单体路径引入 `core.EventBus`（新子系统）；(b) outbox 通知改为直接投递到 MQ（`core.MessageQueuePlugin` 已存在）；(c) 承认 outbox 只在四服务路径有意义，单体不接。
- **现状**：`NewPostgresReorgJobStore` 与 `EventBusPublisher` 都**已实现且从未被构造**——它们的依赖链在部署路径上不存在。

## D5. CORS 策略

- **问题**：`CORSMiddleware` 已实现且从未接线；**全仓库没有任何 CORS 配置项**，其默认是 `allowedOrigins: ["*"]`。
- **选项**：接线前必须先定"允许哪些 origin"（以及是否允许 credentials）。直接接默认值等于把 API 对任意站点开放。
- **注意**：前端目录 `frontend/` 存在，浏览器客户端需要 CORS 才可用。

## D6. 告警往哪发？

- **问题**：`NewAlertManager`（14 处测试引用）从未被构造；T13 剩余项里的"告警↔runbook 对应""故障注入下指标与告警按预期出现"都依赖它。
- **需要**：一个告警出口（webhook / 日志 / Prometheus Alertmanager）与阈值来源。

## D7. 每条链用哪个 finality 策略？

- **问题**：`pkg/services/indexing/finality_adapter.go` 里 `NewFinalityStrategyRegistry`、`NewEth2FinalityStrategy`、`NewL2RollupFinalityStrategy`、`NewProbabilisticFinalityStrategy` **四个构造器全部未接线**。
- **需要**：链 → 策略的映射（按 chain ID 还是按配置项），以及默认策略。

## D8. Kafka 验收环境（只需一行，但需提权）

- **问题**：`TestKafkaRedeliveryKeepsEffectSingle` 因 broker 广播地址 `kafka:9092` 在本机不可解析而 **SKIP**。
- **选项**：把 `kafka → 127.0.0.1` 加进 `/etc/hosts`（需 sudo），或改 broker 的 `KAFKA_ADVERTISED_LISTENERS`（需重启服务，会影响并行会话的 `cp-t12` 等容器）。
- **不做的后果**：I06 的修复只有库语义 + 回归测试背书，**没有真实重投递跑通**。

## D9. 吞吐 property test 怎么处置？（`test/e2e`）

- **问题**：`TestPropertyThroughputConsistency`（`test/e2e/performance_manager_property_test.go:196`）断言 10 轮**墙钟吞吐**各轮与均值偏差 ≤50%。机器有负载时某一轮被抢占即假失败。实测：单独跑 6/6 PASS；全量 `./...` 第 1 次 FAIL（`-75.4%`）、第 2 次 PASS；当时 load 7.4–9.2。
- **为什么需要决策**：它断言的是**调度器**而非代码性质；调整阈值/断言接近"降低指标目标"，计划对此有明确纪律，故不擅改。
- **选项**：(a) `-short` 下跳过（`-short` 语义正是跳过慢/负载敏感用例）；(b) 改断言为代码可控的性质（吞吐为正 + 各轮事件数达标），去掉墙钟比较；(c) 保留现状，但把 §10 该条明确为"逐模块"而非根 `./...`。
- **证据**：`2026-10-13-workspace-modules-verification.md` §3。

## D10. 嵌套模块是否要纳入常规门禁？

- **问题**：`go.work` 有 7 个模块，但 `./...` **不进入**嵌套模块（根目录 89 个包，6 个子模块贡献 0）。于是 `make test-short`、`go build ./...`、`go vet ./...`、以及 PR 模板要求的 `make build`/`make test-unit`/`make vet` **全都漏掉** `pkg/{blockchain,chainid,configmodel,histogram,logkeys,runtimetypes}`。
- **选项**：(a) 在各 Makefile 目标里显式遍历 `go.work` 的模块；(b) 加一个 CI job 逐模块 build/vet/test；(c) 维持现状，接受这 6 个模块只在被 import 时才编译。
- **证据**：`2026-10-13-workspace-modules-verification.md` §1、§4.3。

## D11. 71 个 skip 怎么处置？（计划要求"未知skip=0"）

- **现状**：unit lane 实测 `skip=71`（`pass=6589 fail=0`）。分类：**42 个需要外部依赖**（PG/Redis/Kafka，env 门未开）、**26 个 `-short` 模式跳过**、3 个其他。原因都合理，但**不等于 0**。
- **而且未被强制**：检验器有 `--fail-on-unknown-skip`，但**没有任何 CI 工作流使用它**（CI 只传 `--jsonl … --min-tests 1`）。
- **已补的机制**：该开关的报错文案原本写着 "not allow-listed"，**但代码里没有 allow-list**（一个工具兑现不了的承诺）。已实现 `--skip-allowlist <json>`（声明的放行、未声明的失败）+ 自测。
- **需要决策**：(a) 把 71 个声明为已知可接受（填 allow-list）并在 CI 打开 `--fail-on-unknown-skip`；(b) 消除它们（补依赖或删失效用例）；(c) 维持现状。
- **提醒**：`AGENTS.md` 指出约 89 处 `t.Skip("... pre-existing ...")` 中 **42 处是 "undefined at HEAD"**（生产函数已删）——**不要为了把数字变绿而批量删除**，先分类。
- **证据**：`2026-10-13-coverage-and-skip-verification.md`。

---

## D12. CI 门禁载体（`acceptance-gate` + ruleset + 分支过滤）

- **现状（`gh api` 实测）**：
  - **`acceptance-gate` job 不存在**；计划 §7.3 的 lane 名（`quality`/`unit-race`/`contracts`/`evm-e2e`/`frontend-api`/`security`）**一个都不是 job 名**。最接近的 `notify` 只在夜间失败时开 issue，**不汇总 required checks**。
  - `gh api repos/:owner/:repo/rulesets` → **`[]`**（无 ruleset，且有读权限、确为空）；`branches/main/protection` → **404**（未配置）；`permissions.admin` → **false**（活动账号 `codex689` 仅 `repo` scope，无 admin）。
  - **默认分支是 `master`**，但 `test.yml`/`coverage.yml`/`metrics.yml`/`ci.yml` 的分支过滤是 `[main, develop]`（不含 `master`）⇒ **主测试门禁在默认分支上不触发**。
- **需要决策/权限**：(a) 谁来给 admin（或换 `rtcdance` 账号）；(b) 默认分支改成 `main` 还是给 4 个 workflow 补 `master`；(c) lane 名是否统一到计划 §7.3 的词汇。
- **证据**：`2026-10-13-ci-gate-audit.md`（含全部命令与原始输出）。

## D13. `test/acceptance` 没有任何 CI lane 会跑

- **现状（实测）**：`grep -rn "test/acceptance" .github/workflows/` → **空**；`Makefile` 也没有 `test-acceptance` target。CI 只跑 `./test/integration/...`、`./test/contracts/...`、`./test/e2e/...`。
- **盲区里有什么**：F24 的 SIGTERM + kill 崩溃测试、`TestCanonicalDigestsAcrossRollbackAndRedelivery`、E09 的 DB/Redis 故障测试、Kafka 重投递测试等 —— **全是计划 §9 的验收场景**。它们本会话手工跑全绿，但**没有自动化持续跑**，可以静默腐坏。
- **注**：F25 的 provider 受限测试在 `pkg/plugins/pullers`（无 build tag），**属 unit lane，会被跑**。
- **选项**：(a) 加一个 `acceptance` lane（需 Postgres/Redis service 容器 + `-tags=integration`）；(b) 明确声明它只在发布前手工运行，并写进 DoD；(c) 把其中的关键用例迁到 `test/integration`（已有 lane）。
- **证据**：`2026-10-14-f24-f25-acceptance.md` §4。

## 决策后我可以立刻执行的事

| 决策 | 立刻可做 |
| --- | --- |
| D1 = A 或 B | 按选定栈接线 F05 的 committer（D2 选 b 的话约 3 处改动） |
| D2 | 加 committer 工厂 + 接线 starter + 把 `chain_indexer.go:168` 切到 `ProcessBatchAtomic`，随后跑 T12 端到端 |
| D3 | `DrainController` 落位 + starter 启动/停止 + SIGTERM 路径验收 |
| D4 | outbox drainer 接线（选 b 的话不需要新子系统） |
| D5/D6/D7 | 各自接线 + 验收测试 |
| D8 | 跑 `TestKafkaRedeliveryKeepsEffectSingle`，把 I06 从"库语义背书"升级为"实测通过" |
| D9 | 按选定方案改该 property test，并连续跑 N 次证明不再间歇失败 |
| D10 | 改 Makefile 目标 + 加 CI job，逐模块 build/vet/test 并留原始输出 |
| D11 | 填 allow-list（或消除 skip）+ 在 CI 打开 `--fail-on-unknown-skip`，并留原始输出 |
| D12 | 拿到 admin 后：建 `acceptance-gate` job（`needs` 全部 required + `always()` + 逐条断言 success）+ 配 ruleset + 对齐分支过滤 |
| D13 | 加 `acceptance` lane（或把关键用例迁进 `test/integration`），并留原始输出 |

> 说明：本文件只记录**需要拍板的问题**，不含任何未决的代码改动。以上每一项我都可以在得到答复后立即执行并给出实测证据。
