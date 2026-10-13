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

---

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

> 说明：本文件只记录**需要拍板的问题**，不含任何未决的代码改动。以上每一项我都可以在得到答复后立即执行并给出实测证据。
