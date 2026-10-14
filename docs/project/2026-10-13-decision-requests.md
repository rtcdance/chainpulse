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

## D14. `make test-unit` 的稳定性（PR 必需命令）

- **现状**：PR 模板要求 `make test-unit`，而它实际是 **`go test -v -race -timeout 5m -short ./pkg/...`**（`TESTFLAGS := -v -race -timeout 5m`）—— **带 `-race`**，我此前所有"短测全绿"都**没开 race**。
- **观察到一次失败**：exit=2（82s）；随后用同一命令**累计重复 17 次全部通过**（7 次 load≈14、10 次 load≈9–13）⇒ 失败率 **1/18 ≈ ≤5.5%**，**未复现**，只能判定为**罕见**，**不能**判定是"偶发"还是"低概率稳定竞态"。**失败的包名未知**（当时未保存完整日志）。
- **已排除的假设**：固定端口冲突 —— 测试里**没有**固定端口绑定（唯一的 `net.Listen` 用 `:0`）。
- **需要决定/执行**：(a) 约定"失败必须留存完整日志"（本轮唯一线索就是这样丢的）；(b) 在**安静机器**上重复更多次以定性（本机 load 长期 9–15，不适合）；(c) 若确认偶发，是否允许重试，还是必须修到稳定。
- **为什么重要**：它是 PR 门禁的必需项，**它的稳定性就是门禁的稳定性**；建议**先跑到稳定再设成 required check**。
- **证据**：`2026-10-14-pr-template-commands.md`；`build/acceptance/<sha>/race-repetition/`、`race-flake/`。

## D15. 如何防止"修复被静默回退"

- **触发事件（本会话最重要发现）**：Kafka at-least-once 在 **2026-05-19**（`cf345a2`）被正确实现，**次日 2026-05-20** 被一个**完全不提 Kafka** 的提交（`8f3c475`，主题是 CI / ZeroMQ 死代码 / API 文档）静默回退；同日的 CHANGELOG 提交（`635d6a8`）却**声称该修复存在**。**146 天后**（本会话 `7c05be4`）才被恢复。
- **为什么能藏 146 天**：① 文档说它是好的；② **没有测试守住该不变式**（所以回退不会让任何东西变红）；③ 回退提交的信息里不提 Kafka，review 时不会看到那段 diff。
- **选项**：(a) 对**已修复的缺陷**强制留下"断言该不变式的测试"（本会话已为 I06 补上 `TestConsumerReaderConfigDisablesAutoCommit`）；(b) CHANGELOG 条目必须能指向**断言它的测试**，而不是指向某条提交信息；(c) 对 `git log -S "<关键行>"` 命中"关键行消失"的回退类改动单独标注。
- **未做**：**没有**系统排查"还有多少其他修复被静默回退"——那需要按关键行逐个 `-S` 排查，是一个可做但未做的更大工程。
- **证据**：`2026-10-14-silent-revert-kafka.md`。

## D16. `evm-e2e` lane 没有验收"真实发日志"与"reorg"

- **现状**：CI 跑的 Go `test/e2e` lane 里，**29 个用例在任何环境下都必然 skip**（本地门全开、Anvil 在 PATH 上，仍是 315 pass / 29 skip），因为 `NewTestOrchestrator()`（`test/e2e/orchestrator.go:27`）把 DSN 硬编码成占位值 `postgres://user:password@localhost:5432/test`，且 `test/e2e` 包**不读任何**数据库环境变量——CI 设的 `TEST_DATABASE_URL` / `POSTGRES_*` 全被忽略。被 skip 的 29 个里就包含 `TestDataPullerEventCollection` 与 `TestDataPullerReorgHandling`。
- **`anvil_test.go` 在两个 CI lane 里都不执行**：`test.yml` 装了并起了 Anvil 却**不带 `-tags=e2e`**（文件不参与编译）；`ci-integration.yml` 带 tag 但**不装 Anvil**（6 个全 skip、exit 0）。正确的命令已在 Makefile 的 `test-anvil`（带 tag + 缺依赖即失败），**但无人调用**。
- **lane 内"真实事件"是伪造**：`DeployContract` 用时钟造地址与 tx hash、从不发交易；`EmitEvent` 返回造的 struct；`TestDataPullerEventCollection` 再用它逐字段构造"已采集"列表去和自己比对（同义反复），`RawData` 为空字节。`SimulateReorg` 全程不碰链，全仓库无 `evm_snapshot`/`evm_revert`。
- **真实路径存在但没接 CI**：`docker/verify-e2e-reorg.sh`（真 `cast rpc evm_snapshot` → 发真实事件 → `evm_revert` → 核对索引数）与 `docker/verify-e2e-indexing.sh`（真部署 EventEmitter）只在 `docker/ACCEPTANCE.md` 里被引用；`scripts/multi-chain-e2e.sh` 有 Makefile 目标但无 CI。
- **需要决定/执行**：(a) 占位 DSN 改为读 `TEST_DATABASE_URL`（保留可用默认），还是删除这批用例、承认它们不属于本 lane；(b) `anvil_test.go` 并入 `test.yml`（去 tag 或给该步加 tag），还是给 `ci-integration` 的 e2e job 补 Foundry——两者都影响 CI 时长与额度（关联 F23）；(c) `evm-e2e` lane 究竟跑哪一套（Go 套件 / docker 脚本 / `make test-anvil`），需与 §7 lane 表对齐；(d) 两个 e2e job 的 verifier 是否补 `--fail-on-unknown-skip`。
- **为什么重要**：§7 把 `evm-e2e` 定为每个 PR 都跑的 lane，验收是"真实发日志 + reorg/restart"。当前它既没有真日志、也没有真 reorg，核心场景还 skip——**lane 的绿灯不代表它宣称验收过的东西被验收过**。
- **证据**：`2026-10-14-evm-e2e-lane.md`；`build/acceptance/<sha>/e2e-lane/{e2e-full-run.jsonl,ci-anvil-simulated-absent.jsonl,SUMMARY.txt}`。

## D17. 安全 lane 缺"已审定基线"机制，且仍不是独立 required check

- **现状**：`ci.yml` 的 build job 里原先 `Security scan` 带 **`continue-on-error: true`**（**已修**：§7.3 line 311 明确禁止；改为按退出码失败 job），版本由 `@latest` 固定为 **`v1.8.0`**，并补扫 **6 个嵌套模块**（根 `./...` 够不到 `pkg/{configmodel,logkeys,blockchain,runtimetypes,histogram,chainid}`，已用 `go list` 逐个确认），同时接入 `govulncheck-summary.py` 产出 §10 工件。修后本地实测 **7 个模块全 exit 0 / 0 callable**。
- **需要决定/执行**：(a) §7.3 写的是"按**已审定基线**"——即允许把已知可调用漏洞登记为**已处置**。目前 `govulncheck` **没有基线机制**（`bench-gate` 有）。不建基线，则任何新可调用漏洞（含低危）都会直接阻断 PR，且第三方库新披露会让仓库"无故"变红。(b) `security` 在 §7.3 是**独立 lane / required check**，现在只是 build job 的一个 step。新建 lane 涉及 **T15 job 命名冻结**与 **F14 ruleset（需 admin）**，不宜单方面改动。(c) secret scan / gosec / 镜像扫描在 §7.3 也属 security lane，当前**均不存在**。
- **为什么重要**：安全门禁的价值在于"新严重项阻断"。去掉 `continue-on-error` 已让 govulncheck 真正具备阻断力，但缺基线会让它变成"噪音源"，而缺独立 lane 则让它在 §7.3 的表里**名不副实**。
- **证据**：`2026-10-14-ci-negative-cases.md`；`build/acceptance/<sha>/security/SUMMARY.txt`。

## D18. release 流程的镜像 digest 与 SBOM 未审

- **现状**：§7.5 第 3 条要求"生成覆盖**所有二进制/镜像**的 checksum、**SBOM** 与扫描报告……**镜像使用不可变 digest，禁止重新 build 后直接提升**"。本轮**已修**二进制 checksum 循环覆盖写文件问题（`release.yml` 的 `Create checksums`，25 个二进制只登记 5 个；已抽成 `scripts/acceptance/checksums.sh` 并纳入 25 例反例矩阵）。
- **未做**：(a) **镜像 digest**：`docker` job 的 push/提升逻辑**未审查**，是否存在"重新 build 后直接提升"未知。(b) **SBOM**：`govulncheck-summary.py` 有 `--sbom-dir`，但 release 流程里**没有 SBOM 步骤**。(c) **`checksums.txt` 是否随 release 发布未确认**——release 用 `files: dist/**/*`，按 minimatch 语义应匹配深度 1 的文件，但那是 action 自己的 glob 方言，本机无法验证；我**刻意没有**加显式一行，因为若本已匹配会导致资产重名冲突。
- **需要决定/执行**：在有 GitHub 的环境确认 `checksums.txt` 是否被发布；审查 `docker` job 的 digest 策略；把 SBOM 接进 release。
- **证据**：`2026-10-14-release-checksums.md`；`build/acceptance/<sha>/release-checksums/`。

## D19. `deploy.yml` 现在会红，需要一个决定；`metrics.yml` 同类缺陷未修

- **背景**：§7.5 第 8 条要求"验证失败保留失败状态；不能用 `|| true` 使 `if: failure()` 无法触发；缺 kubeconfig 时不能报告 K8s 部署成功"；§7.3 line 311 要求"测试步骤不使用 `continue-on-error`"。`deploy.yml` 原本 **14/21 个 run-step 被掩蔽**、引用的**三个验证脚本都不存在**、并**生成一份硬编码四个 ✓ 的就绪报告**（且 `$(date)` 因 quoted heredoc 未展开）。**已修**：掩蔽清零（0/21）、缺脚本与缺 kubeconfig 改为 fail-closed、报告改由 `steps.<id>.outcome` 生成、21 个 run 块全部 `bash -n` 通过。
- **需要决定（二选一）**：`scripts/verify-production-readiness.sh` **不存在**，而 `verify` job 现在会因此失败。**(a)** 写出该脚本（需要定义"生产就绪"检查什么——这是设计决策）；**(b)** 删掉这一步（承认该 job 只做 tests/lint/docs）。**在此之前 `deploy.yml` 会红**——这是有意的（§7.5"保留失败状态"）。触发面仅限 **tag push 与手动 dispatch，PR 不受影响**。
- **同类未修**：`metrics.yml` 仍有 **6/9** 个 run-step 被掩蔽，**含 "Check for regressions"**——那是 **§8 性能回归门禁**，掩蔽意味着它不可能失败。同类问题，需同样处理。
- **未接**：§7.5 说"不采用 K8s 的版本应明确调用已选 Docker 部署路径"。本仓库 CD 是纯 K8s；`docker/deploy-monolith.sh` 存在但未被 workflow 调用。选哪条路径是决策。
- **E15 仍未执行**：§10 要求"CD 部署目标已在 T00 冻结且 **E15 在该目标真实执行**"。本轮是**让门禁不再说谎**，不是**完成一次真实部署**（仍需环境，见 F22）。
- **证据**：`2026-10-14-cd-deploy-gate.md`；`build/acceptance/<sha>/cd-deploy-gate/SUMMARY.txt`。

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
| D14 | 在 `-race` 下重复 10–20 次判定 `make test-unit` 的失败是偶发还是稳定竞态，并加"失败留存完整日志"约定 |
| D15 | 为已修复缺陷补"断言不变式"的测试；要求 CHANGELOG 条目指向测试；对 `-S` 命中"关键行消失"的回退单独标注 |
| D16 | 定 `evm-e2e` lane 跑哪一套；修占位 DSN 与 `anvil_test.go` 的 tag 错配；两个 e2e job 补 `--fail-on-unknown-skip` |
| D17 | 给 govulncheck 建"已审定基线"机制；把 `security` 提为独立 required check（依赖 T15 命名冻结与 F14 ruleset）；补 secret/gosec/镜像扫描 |
| D18 | 在有 GitHub 的环境确认 `checksums.txt` 是否随 release 发布；审查 `docker` job 的镜像 digest 策略；把 SBOM 接进 release |
| D19 | 决定 `scripts/verify-production-readiness.sh` 是**写出来**还是**删掉这一步**（`deploy.yml` 在此之前会有意变红）；用同样手法修 `metrics.yml` 的 6 处掩蔽（含 §8 性能回归门禁）；决定 CD 走 K8s 还是 Docker 路径 |

> 说明：本文件只记录**需要拍板的问题**，不含任何未决的代码改动。以上每一项我都可以在得到答复后立即执行并给出实测证据。
