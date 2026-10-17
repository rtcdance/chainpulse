# 实际接线与验收记录（2026-10-14）

这是已批准交付方案的执行记录，不是新的优化方案。全量交付仍未完成，原方案保持 Approved；下列局部结果不能替代 T18。

## 已实施的生产接线

- 单体 PostgreSQL 模式按完整 RPC 区间分组提交整个区块，空块也保存持久 checkpoint。原子提交失败不回退到旧写入路径。
- 每轮拉取检查持久 checkpoint 的 canonical hash，包括进程重启和链头未增长时的同高分支替换。限定深度内找不到已知祖先时停止推进。
- 重组事务同时完成 writer fencing、事件撤销、block snapshot 清理、checkpoint 回退、generation 更新、repair job 和 outbox。任务只在新 canonical 重放追上之后完成；拒绝撤销 finalized 行。
- Kafka 使用 FetchMessage，handler 成功之后才提交 offset。订阅入口复用同一消费配置；删除了多余 reader 创建及其嵌套锁问题。
- PostgreSQL event_data 使用版本化二进制载荷，保存 raw bytes、topics、交易序号和解码结果。真实 ABI 零字节可持久化，金额以字符串保留；puller physical ID 包含分支身份。
- 实际列表 API 直接对 PostgreSQL canonical 行过滤、计数和分页，在同一 repeatable-read snapshot 中完成。多链查询要求明确 chain filter，默认排除 orphan，空结果和后端失败不切到旧数据源。移除了可能返回旧重组状态的无限增长内存缓存。
- /readyz、/livez 实际经过网关分发，/health/ready 增加处理可用性检查。进度采样在 RPC 初始化之后启动，RPC 未初始化时返回错误而非 panic。
- 单体和 API Gateway 使用共享 Redis nonce store，明确 SIWE chain ID，可配置 origin/URI。真实 Redis 请求不降级为本地 nonce；缺失或不可用时失败。
- 修复 Redis factory 的旧指针 Initialize 接线、原始 host:port 解析、连接日志脱敏和连接池关闭。请求 Redis 而实际未连接时拒绝启动。
- 受影响的兼容类型引用迁移到现有 ports 定义。纠正 ports 包文档错误地把权威定义标为 deprecated 的矛盾。组件复用进程 tracer provider。

## 当前已取得的证据

原始目录：`build/acceptance/20261014-working/`。这是 dirty 工作树的本地执行证据，不是干净 candidate SHA、远端 CI 或最终 release manifest。

| 项目 | 原始工件及结果 |
| --- | --- |
| 实际区块重组、停机期间重组后恢复 | deployed-reorg-final.jsonl；深度 1/2/12，SQL 与 RPC canonical tuple 摘要一致。随后 full/contracts.jsonl 对更新后的实际入口重新运行 |
| 同交易重新入块 | 进程级测试明确比较新旧 transaction hash 相同、block hash 不同；新旧 physical effects 分离 |
| 真实依赖与故障 | full/contracts.jsonl：124 个 test pass actions，54 个顶级 Test，0 fail、0 skip；包含 PostgreSQL/Redis 故障、Kafka 重投、单体/四服务重组、双链模式 oracle 和 E06 repair crash |
| SIWE 跨实例 HTTP 重放 | siwe-http-replicas.jsonl：两个实例，100 请求恰好 1 个成功；使用生产 bootstrap Redis store factory |
| 不安全 nonce 的反例可复现性 | nonce-negative-deterministic.log：通道屏障固定检查与删除之间的交错，GOMAXPROCS=1、race、100 次均通过。修复了依赖调度碰巧出现多个赢家的 flaky 控制测试 |
| 实际 storage factory | production-factories.jsonl：真实 PostgreSQL/Redis 构造、二进制读写、关闭 Redis pool |
| workspace 完整性 | workspace.log：动态发现的 7 个模块 build/vet/test 全通过。后续变更仍需最终 candidate 上复验 |
| schema | full/migrations.jsonl：14 个迁移，fresh/upgrade/full-down/reapply；7 pass actions，0 skip。修复测试错连固定数据库的问题 |
| 门禁反例 | full/negative-controls.json、negative-controls.log：29/29 符合预期；每个失败输入都有通过的阳性对照 |
| Workflow 静态验证 | actionlint-fixed.log：通过。Actions 锁到 upstream 已解析的 commit SHA，服务容器上下文和重复 env 已修正 |
| OpenAPI | full/openapi-report.json：drift 检查通过 |
| 标准 DoD | dod-results.json 和五份 *-final.log。每次源代码变更后只接受最后一轮实际结果 |

完整 micro-loop 已替换两处失效门禁：旧指标脚本匹配零 Test，旧 migration-governance 路径已归档。新 full gate 必须实际执行指标、真实依赖故障、进程级重组、schema 迁移与门禁反例，缺少隔离依赖即失败。

## 仍需完成的原方案范围

- T00：分批落地并建立干净 candidate；本轮未重置、丢弃或清理原有 dirty/stash。
- T01/T15：同 SHA 的远端 CI、反例 run URL、ruleset/required checks、完整 release/CD 门禁和不可变镜像 digest。
- T05/T12：完整区块事务、实际四服务查询、双链正常模式 oracle 和 1/2/12 重组已通过；仍需 Kafka rebalance/revocation 所有权生命周期、双链故障等价和更严格的已排队旧分支恢复。
- T11：generation-safe 热缓存、并发 cursor 与订阅恢复的完整验收。
- T12/E01/E13：双链正常模式 oracle 与 Swap fixture 已通过；完整双链故障矩阵、浏览器/订阅流程和一条命令的镜像部署仍需完成。
- T14/E14：原方案指定的固定语料、250 eps/600s、100 req/s/600s、积压恢复、资源/RPC 成本及 30min soak。没有取得这些实测结果。
- T16/E15：E06 repair-crash 本地已通过；剩余完整双链故障矩阵、镜像版本回滚与数据恢复。
- T17/T18：对应验收 tag 的录屏和完整证据包、最终 fresh checkout 与作者验收。

最后一次 GitHub 权限核验：CLI 账号 codex689 对 rtcdance/chainpulse 只有 pull，push/admin 均 false，默认分支 master。用户已表示会切换账号；此记录不把该意图当作权限已改变，也不把它作为缩小交付范围的理由。

后续继续顺序：统一四服务完整区块消费与存储权限边界；运行双链模式等价及原规模性能/soak；完成候选提交、远端 CI/CD、回滚和封版。没有将这些剩余项改为可选。

## 四服务投递边界追加验证

- Event processor 删除异步入队即成功的 ack 路径，Kafka callback 等待处理与输出发布完成；处理失败、输出发布失败均返回给 Kafka，不推进消费 offset。暂停/恢复广播修正为每个周期有效。
- Puller 首个 encode/publish 错误立即结束本轮，不推进 intake checkpoint，不执行成功 shadow；重试完整范围。全部成功后推进到 RPC 区间末尾，包括末尾空块。此 intake cursor 仍不是 PostgreSQL 的持久事实 checkpoint。
- Puller 对生产 Kafka 使用 NewChainMessage；writer 改为 Hash 和 RequireAll。此前三分区测试暴露默认 RequireNone 导致 PublishMessage 成功却无法观察到全部已发布记录；修复后 24 条记录完整、两条链分别保持单分区与发布顺序。
- 原始追加工件：message-boundaries-100.log（race、100 次）；message-boundaries-staticcheck.log（0）；message-path-race.log（三个受影响 package 通过）；kafka-chain-partition.log（真实三分区 broker、race、通过）。组合 full gate 和标准 DoD 已在这些最新变更上重跑通过，原始记录见下文。
- 此处的处理完成边界不表示已把 Mongo/PG 分裂存储统一为块事务，也不表示 rebalance 所有权、两模式等价或 T18 已通过。

- 追加 PostgreSQL fenced checkpoint 边界：拒绝跨过未完成块和未经 durable reorg 的同高换分支；旧 canonical 消息重投不回退进度。checkpoint-order-red.log 先复现 gap 被错误接受，checkpoint-order.log 真实 PostgreSQL race 测试通过。旧 unfenced legacy 适配器行为保留，完整区块生产路径使用 fenced transaction。

## 最新工作树门禁结果

`micro-full-message-boundaries.log` 已结束（exit 0）：指标 14 pass/0 skip；真实 adapter、故障和实际入口重组 114 pass actions、50 顶级 Test、0 fail/0 skip；迁移 7 pass/0 skip；29/29 门禁反例通过。OpenAPI drift、自检、staticcheck、lint 与编译均通过。

标准 DoD 按顺序重新执行，`dod-results.json` 五项 exit_code 均为 0。原始输出文件保留为 `go-build-final.log`、`make-check-final.log`、`make-lint-final.log`、`make-test-short-final.log`、`git-diff-stat-final.log`。`make lint` 的 0 issues 仍是仓库规定的 diff scope，不表示历史 lint backlog 清零。

短测试同范围追加 JSON 验证结果（`test-short-final-report.json`）：7088 pass actions、0 fail、71 skip actions，5272 个顶级 Test。71 个 skip 全部匹配既有声明的 allowlist；保留这些例外，未把它们改写成“全部测试都实际通过”。真实依赖 contracts 的 0 skip 与此短测试统计分开。

仍为 dirty 工作树证据。当前没有 final candidate SHA、远端 CI/CD 成功或完整 T00–T18 封版结论。

## 完整区块生产协议追加实施

已增加 `chainpulse.block.v1` 的 header + 全块 logs frame，共享业务校验归入 application/indexing。单体与 native PostgreSQL processor 使用同一 `ProcessCompleteBlock` → `CommitCompleteBlock` → fenced PG transaction。header 的 parent/time 不再作为单体事务之后的独立 side effect 写入。

native puller 的 PostgreSQL 配置按 bounded RPC range 发布 complete frames（含空块），使用数字 chain key 的 Kafka Hash/RequireAll。processor PostgreSQL 配置不启动 Mongo event/metadata 双写，通过现有消费控制、HTTP 和安全 middleware 接入 shared runtime。完整事务失败不 ack；schema/链键错误失败；大整数 JSON 解码用 Number 保留精度。停机先关闭事务 admission、等待 in-flight commit，再关闭消费并释放所有权。

实测 `native-block-pipeline.jsonl`：真实 Anvil + 三分区 Kafka + native puller/processor + isolated PG，1 pass、0 fail/0 skip，含 8-log transaction、原始 bytes 对照、消费进程停机积压/重启、空块 hash/parent 元数据和两个进程 SIGTERM。日志在 native-block-pipeline/；fixture 采用 1-block confirmation，发交易之后显式挖确认块，未绕过配置验证。源码/真实数据库辅助证据见 complete-header.log、complete-block-boundaries-100.log、completed-block-final-100.log、durable-block-staticcheck.log。

CI 已增加 native binaries 与此真实场景；contracts job 不拥有 Anvil 的两项 lane 例外均明确列入 skip allowlist，evm-e2e 实际入口验证不允许 skip。evm-e2e 的 E01/E13 仍为空待完成，故当前全 lane required gate 仍不能作为最终通过结果。缺少 native binaries/isolated DB 的 full micro-loop 必须失败。

仍需：两独立 EVM 链及全四服务 mode oracle；完整重组/repair/outbox 与 Kafka generation/rebalance 接线；订阅、性能长跑、release/CD 和最终 SHA 证据。旧非 PG legacy 路径没有作为这些必做项的替代。

最新组合门禁 `micro-full-durable-blocks.log` 已完成（exit 0）：114 pass actions、50 顶级 Test、0 fail/0 skip，包含 native complete-block pipeline 与更新后的单体 1/2/12 深度真实重组。PG complete header rollback、全 schema fresh/down/reapply、29/29 gate controls、staticcheck 和 Actionlint 通过。标准 DoD 重新按序全部 exit 0；同范围短测试 JSON 是 7077 pass/0 fail/71 已声明 skip。尚无远端 lane green、clean candidate 或最终 release 结论。


## 2026-10-15：四服务查询与版本化 cursor

PostgreSQL API-service 的真实 factory 已改为 canonical query runtime，并为列表、详情、统计和 reader 分页复用 PostgreSQL 数据。补齐已存在前端发出的 chain_id 参数；保留原参数优先级并接受 camelCase aliases。修复孤块详情仍可见、统计加载全表、查询失败/空结果回退等路径。统计全部计数来自同一数据库 snapshot；adapter 声明的 chain scope 不能被请求扩大。工厂构造失败会释放已初始化的数据库，关闭错误会向上返回并继续清理其他资源。

新增 v1 游标绑定链、过滤摘要、canonical generation 与排序位置。generation/anchor/page 使用同一 repeatable-read snapshot；REST 返回 nextCursor，GraphQL 支持 after 和真实 edge/page cursors。不同 chain/filter 拒绝请求，generation 变更返回 CURSOR_STALE 并要求重新分页。已通过真实 PostgreSQL 的新增头部数据不重复分页、孤块 anchor 拒绝和 generation 失效测试；尚未把缓存或订阅恢复标为完成。

native-block-pipeline.jsonl 已升级为真实四个 native 进程，直接访问 API-service 与 API Gateway：验证 8 条真实链日志、逐条详情、空结果、与 SQL 相符的统计、REST/GraphQL 连续 cursor、过滤变化错误、processor 停机积压后恢复及四进程 SIGTERM。此查询正确性 fixture 明确关闭中间件限流；限流行为由单独场景验收，不能用此结果声称生产限流验证完成。

当前补充证据：canonical-reader-cursor.log、versioned-cursor.log、canonical-api-failclosed.log、filter-alias.log、runtime-cleanup.log、canonical-api-staticcheck.log、canonical-api-actionlint.log。workspace-canonical-api.log 已确认动态发现的 7 个模块 build/vet/test 通过。最新组合 full gate/标准 DoD 仍以完成后的原始记录为准。

已准备第二个独立 Anvil（31338，18566）和 bytecode 与编译工件一致的 EventEmitter/SwapEmitter；dual-chain-fixtures.json 仅是环境与 fixture 证明，尚不是 E01/E13 双链模式等价验收。


2026-10-15 最新验收收尾：完整 micro-loop 已 exit 0。full/contracts-report.json 为 114 pass actions、50 顶级 Test、0 fail/0 skip；指标 14 pass、迁移 7 pass、29/29 反例、OpenAPI drift、自检和 staticcheck 均通过。标准 DoD 五项按序 exit 0；新短测试 JSON 为 7088 pass、0 fail、71 已声明 skip（5272 顶级 Test）。七模块完整检查通过。新增四服务 API 与 REST/GraphQL cursor 已进入实际进程场景，不能将这些结果替代仍待执行的双链 oracle、四服务重组/rebalance、generation-safe 热缓存、订阅恢复、原规模性能/soak、真实版本回滚和同 SHA 远端 CI/CD。GitHub CLI 实测账号仍为 codex689，rtcdance/chainpulse push/admin=false。


## 2026-10-15：E01/E13 双独立链与两模式 oracle

新增 TestDeployedDualChainModesMatchRPCOracle，独立链 31337/31338、独立 mode 数据库，先发 uint256 最大值 Transfer 和真实 Swap，再分别启动单体、四个 native 服务。RPC eth_getLogs 为 oracle；比较全部原始 topics/data 的 canonical physical tuple 摘要，以及每条链每一已提交高度的 hash/parent。API 逐链核对统计量和金额字符串，并明确要求 Swap 在实际 API 中出现。

第二次独立执行 dual-mode.jsonl 已通过：3 pass actions（父 Test + 两 mode 子 Test），0 fail/0 skip。dual-mode/dual-mode-oracle.json 记录 345 条日志、163 个完整 header，monolithic / microservices / oracle 三个摘要均为 6eed108ca3d4ff14505aacee96c9344054435861c470487410d87e5419ce253c。每次运行在真实链上新增事件，所以未来执行数量和摘要可能变化；同次运行的三方一致性才是断言。

E01/E13 已映射到此真实 Test，不再保留空 tests 占位。CI 安装并启动第二个 Anvil、部署两个 fixture、创建两个 mode DB 并迁移，evm-e2e 需要三个实际入口场景全部运行；contracts lane 的例外只说明它没有双 Anvil fixture，不能代替 evm-e2e。full micro-loop 也强制要求双链输入并验证 evm-e2e 场景。Actionlint 与本地 fast gate 通过；当前完整组合门禁和标准 DoD 正在复验。

这些是正常链路的模式等价证据。四服务 reorg/rebalance、repair 途中再次崩溃、热缓存/订阅恢复、性能长跑、release rollback 和同 SHA 的远端 CI/CD 仍是必做范围，不以正常摘要一致代替。

## 2026-10-15：四服务真实重组闭环

上一轮双链组合 full gate 实际结果为 117 pass actions、51 顶级 Test、0 fail/0 skip；标准 DoD 全部 exit 0。该结果已完成，不再记为正在运行。

新增四服务同高重组回归先在旧二进制复现失败（native-reorg-before-fix.jsonl：旧分支未撤销、新分支未重放），再完成生产接线：共享 bounded canonical ancestor scan，puller 每轮检查 PostgreSQL checkpoint 与已发布 intake tip，重启从 durable checkpoint 恢复；rollback 增加 expected checkpoint hash，防止陈旧监督决策撤销已修复分支。processor 在持有写租约的事务内部检查 RPC canonical header，过期原 owner 不能再开启事务。孤块帧先 rollback，再将完整 payload 持久化为 terminal orphaned audit；审计失败不 ack，成功审计不发布 committed-block 通知。

原生 puller 现在按 chain key 发布版本化持久 outbox envelope，携带 notification/job/chain/generation/kind，成功 RequireAll 后才标记。缺失 topic/发布失败保留待投递记录。processor 追上原重组高水位之后才完成 repair job。两个新原生环境变量与迁移/恢复步骤见 docs/architecture/multi-chain.md。

native-block-pipeline.jsonl 已实际通过深度 1、2、12 三个子场景：同交易重新入块、新旧 block hash 不同、旧 physical effects 保留为 orphan、滞留旧分支帧被审计且后续 canonical 消息继续、repair job 完成、outbox 清空。canonical-fence-contracts-report.json 为真实 PostgreSQL 3 pass、0 fail/skip；覆盖过期 owner、陈旧 rollback 决策、100 次孤块审计幂等。native-canonical-unit.log、native-intake-unit.log 和 orphan-ack-unit.log 为 race+100 次重复通过。fast micro-loop 与 Actionlint 通过。

当前最新组合 full gate 还在运行，新增整条 canonical history oracle 与真实 reorg topic envelope 核对以其完成后工件为准。本轮没有将 Kafka rebalance/revocation fencing、双链故障隔离、repair 途中再次 crash、热缓存/订阅、性能/soak、release rollback 或同 SHA 远端 CI/CD 标为完成。


2026-10-15 本轮重组组合验收已收尾：micro-full-durable-blocks.log exit 0，full/contracts-report.json 为 123 pass actions、54 顶级 Test、0 fail/0 skip；指标14、迁移7、29/29 gate controls 均通过。full-deployed-reorg/native-reorg-oracle.json 记录 620 条 canonical 日志，SQL/RPC 摘要均为 59aa1f1592ad53ab81f65477b7c0bebeafc550907f6b9c0b3fa3a5c09063dd01，并实际消费核验 6 条带 job/generation 的 Kafka reorg envelope。当前双链正常模式工件有629条日志、287个 header，三方摘要均为 e7375fe1294f6bac815f99450ef230d70138a36462660dcdbdaeb686a437aa68。计数只属于同一次运行。

标准 DoD 五项按序 exit 0；最新 test-short-final-report.json 为7099 pass、0 fail、71已声明skip、5277顶级Test。38个基线dirty文件字节未变、36个后续有变、0个缺失，见 dirty-inventory.json；未提交或清理原有修改。后续 E06 演练与其他剩余任务仍需单独补齐，123动作不包含尚未实施的 repair-crash 场景。


## 2026-10-15：E06 与已排队孤块缺口

严格扩展 E06 时发现并复现另一实际缺陷：重组回退后，队列中高于新 checkpoint 的旧分支帧先触发顺序缺口检查，反复重试同 offset，阻塞正确分支。repair-crash-backlog-before-fix.jsonl 记录失败与完整 oracle 的 803/795 差异。修复保留原连续性检查，新增 typed BlockGapError；只有这类拒绝才额外查询 RPC，确认孤块后仍须完整持久化审计。canonical 真缺块、RPC 失败和审计失败均不能误确认。

canonical-gap-unit.log 是 race+100 次重复，包含 canonical/orphan gap、RPC/audit/store 失败的正反例。严格 native 回归修复后通过；E06 已映射到真实 repair-crash 子 Test 并由 evm-e2e/full gate 强制执行，不再以 resume 单元测试替代故障演练。

最新组合运行中的 E06 已完成：full-deployed-reorg/repair-crash-before-restart.json 记录已提交回退与实际通知失败次数；repair-crash-after-restart.json 记录重启后 2 条真实 Kafka 通知、1 条带8个log的 queued orphan 审计、8条 canonical replacement 以及8条 retained orphan。native-reorg-oracle.json 为963条 canonical 日志，SQL/RPC摘要同为8e4b1ff65e69590af8432a02b7070e75de5c30316d3f2e70edda7f88b3b6af2c。其余组合测试仍以整轮结束后的报告为准。

最新标准 DoD 五项按序 exit0，test-short-final-report.json 为7105 pass、0 fail、71已声明skip、5278顶级Test。repair-crash-fast.log 已通过。仍未完成实际 Kafka rebalance/revocation fencing、全双链故障隔离、独立事件状态语义、hot cache/订阅、完整SLO/性能/soak、镜像版本回滚和同SHA远端CI/CD。E06 的本地成功不缩减这些原方案任务。


2026-10-15 最新 E06 组合收尾：完整 micro-loop exit0，full/contracts-report.json 为124 pass actions、54顶级Test、0 fail/0 skip；required E06 子Test实际通过。指标14、迁移7、29/29门禁反例及OpenAPI/staticcheck通过，repair-crash-actionlint.log为空且exit0。标准DoD五项按序exit0；最新短测试为7105 pass、0 fail、71已声明skip、5278顶级Test。

本次双链正常模式 oracle/indexer 摘要为 deaf4d4aa5acd0693416e1d5722c758966707212734fb937bc78e9cac965a1fb，日志974条、header 444个。源码工作清单 working-source-manifest.json 已刷新；另冻结 repair-crash-source-manifest.json，并用 repair-crash-binaries.json 保存本轮五个实际 native race 二进制的 SHA256/Go build metadata。它们是 dirty native 证据，不是不可变 Linux image release，也不构成T18同SHA远端验收。


## 2026-10-15：T05 rebalance 租约隔离证据与 T11b 订阅恢复

T05 已从“机制存在、无证据”变为已执行验收。既有 test/acceptance/kafka_generation_fencing_test.go 在真实 Kafka(127.0.0.1:19092)+PostgreSQL(127.0.0.1:15433) 下实跑 PASS（2.63s，exit 0）。新增 test/acceptance/kafka_generation_fencing_ordering_test.go（integration tag，缺 env 即 t.Fatalf，零 t.Skip）断言：Revoked 在下一代 Assigned 之前完成；撤销后 chain_leases 行仍在且 expires_at<=now、owner 仍为旧 token；旧 owner 的 BeginFencedTx 返回 ErrLeaseLost；新代接管后可提交一次。count=3 重复运行 3×PASS。新增 pkg/services/processor/ownership_revoke_test.go 为不可跳过单元门，固定 9 步效果序（release 先于 owner swap）。consumer_runtime.go 的 legacy fallback 未改码：t05-fallback-inventory.md 证明 durable 模式下 PostgreSQL 写者受 main.go 启动门控不可达该 fallback，legacy PG 触点仅为 append-only 的 event_metadata/dlq_events。CI 判定：contracts lane job env 已含 Kafka+PG 集成 flag，fail-hard 测试不会误红，required-cases 无需变更。证据：build/acceptance/20261014-working/t05-generation-fencing.log、t05-generation-ordering.log、t05-generation-ordering-count3.log、t05-generation-combined.log、t05-fallback-inventory.md；gates build/vet/make lint/make check/test-short 全部 exit 0（PASS=119 SKIP=0 FAIL=0）。

T11b 订阅恢复在生产路径 EventSubscriptionHandler（全模式同时服务 /ws 与 /events/subscribe*）落地：BufferEvent 与订阅快照同临界区、单调 seq；cursor 优先级 Last-Event-ID header > ?last_event_id= > 无 cursor 仅实时（向后兼容），malformed cursor=400；新增 BroadcastReorg 与 monolith main.go 订阅 reorg-rollback 发出 type=reorg 更正。RED→GREEN 日志见 t11b-sub-red-resume.log、t11b-sub-red-reorg.log、t11b-sub-green.log、t11b-sub-green-allhandlers.log、t11b-sub-gates.log；独立 -race 复跑两测试 exit 0；gates build/vet/lint/test-short exit 0。已知边界：api-service 模式无 EventBus，重组更正仅单体模式生效；resume 两种模式均可用；前端未改（seq 为附加字段）。

仍未完成（不得以本节缩减）：全双链故障隔离（E04 进行中）、热缓存 generation 接线（T11a 进行中）、独立事件状态语义、完整 SLO/性能/soak、镜像版本回滚、同 SHA 远端 CI/CD。

## 2026-10-15：T11a 版本化缓存键与查询路径参数收紧

热缓存 generation 接线已落地并复验。新增 pkg/ports/cache_scope.go、pkg/plugins/cache/cache_scope_resolver.go（LiveScopeResolver）、pkg/plugins/database/postgres_cache_generation.go（ReadCacheGeneration 读 reorg_generations 聚合）；query_service.go 新增 SetCacheScopeVersioner，Query/QueryByHash 缓存键变为 v{ver}:c{chain}:g{gen} 形式，versioner 读取失败时 fail-closed 绕过缓存（绝不回退裸键）；runtime_wiring.go 在 mongo 分支注入 newCacheScopeVersioner。canonical PostgreSQL 查询路径保持无缓存，未新增缓存。chainId 决定作用域，network 忽略（跨链 puller 默认同名 network，不能证明单链）。同批收紧：list/correlated 端点对畸形数值参数（limit=abc、溢出、to_block=1.5 等）返回 400，见 event_query_handler_params_test.go（9 子测）。

迁移 000015_live_chain_contract_index 新增部分索引 idx_events_live_chain_contract (chain_id, lower(contract_address), block_number DESC) WHERE status<>'reorged'，附 EXPLAIN (ANALYZE, BUFFERS) 证据（t11-cache-explain.log，50 万行/4 链）：count 默认谓词 362ms→23ms（Index Only Scan，Heap Fetches 0）；contract 分页 461–529ms 全表→0.11–0.32ms 索引扫描。迁移幂等测试（15 个迁移、4 阶段 up/down/up）通过；up/down 文件齐备。

证据：t11-cache-red.log、t11-cache-green.log、t11-cache-params-red.log、t11-cache-params-green.log、t11-cache-explain.log、t11-cache-migration-test.log、t11-cache-gates.log、build/acceptance/30c45b05485b/migration-report.json。gates：build/make check/make lint(0 issues)/make test-short 全 exit 0；lsp 干净 ×9。独立复验：services/query+plugins/cache 与 api Params 测试 -race 均 exit 0，新文件 0 t.Skip。已知边界：查询缓存仍为进程内 map（约 5 分钟 TTL），Redis 仅用于索引幂等，查询路径 Redis 故障为 no-op；版本化仅在 mongo 分支生效。

## 2026-10-15：E04 双链故障隔离验收落地

E04 从“无部署级测试”变为已执行并通过。新增 test/acceptance/deployed_dual_fault_test.go（integration tag，约380行，门控 CHAINPULSE_RUN_DEPLOYED_DUAL_FAULT=1，独立端口 20482-20485）：将两条 Anvil 链等高到共享高度后启动四服务栈，仅对链 A 执行 evm_snapshot→evm_revert→evm_setNextBlockTimestamp(oldHeader.Time+30)→重发同一交易（同 tx hash、新 block hash）→anvil_mine 的同高度替换；断言链 B(31338) 前后深度相等（stateBlock 519、canonical 11、reorged 0、reorg_jobs 0、gateway pageTotal 11），链 A 的 FilterLogs(1..H+1) oracle 摘要与库内摘要一致（88cc6df1…，1171 条 oracle 事件、orphanReorged 1），产物 dual-fault-oracle.json。

基线运行 PASS（38.14s，exit 0）：此前怀疑的 DerivedStateInvalidator 为空 / 未加链范围回滚缺陷未显现，生产代码零修改。证据：build/acceptance/20261014-working/dual-fault.jsonl、dual-fault-oracle.json、dual-fault-{processor,puller,api-service,gateway}-*.log。

CI 接线：ci-integration.yml 增加 CHAINPULSE_RUN_DEPLOYED_DUAL_FAULT:'1'，evm-e2e -run 正则加入 TestDeployedDualChainFaultIsolation，--min-tests 由 3 升为 4；required-cases.json 在 contracts E04 之前插入 lane=evm-e2e 的 E04（verify-tests.py 先按 lane 过滤再以 id 字典覆盖，按 lane 安全）；skip-allowlist.json 增加 dependency_gated 允许项，精确匹配缺失 env 时的 skip 文案，到期 2026-11-14。JSON/YAML 解析、go build、make check、make lint(0 issues)、make test-short 全部 exit 0。边界：涉及的 3 个文件同时含并行在途 hunk（属他人改动）；本轮二进制由 dirty 树构建，同 SHA 远端 CI 仍是最终验收；本地复跑配方与二进制绝对路径注意事项见 agent 报告（go test cwd 为包目录，CI 用 $PWD/build/deployed-* 绝对路径）。

仍未完成（不得以本节缩减）：独立事件状态语义、完整 SLO/性能/soak（T14）、镜像版本回滚与 SBOM/security lane（T15/T16）、同 SHA 远端 CI/CD 与 ruleset（需管理员）、T00 干净候选与 T17/T18 封存。

## 2026-10-15：Kafka generation / revocation fencing

已实现 native PostgreSQL 专用的 generation-owned 消费路径。processor 仅接受一个权威 block input topic，并拒绝 legacy consumer 回退；broker generation 的撤销传到 handler、RPC 和 SQL context。旧分区处理结束后先释放 DB owner，再允许新 generation 开始效果。每个 generation 轮换独立 owner token；pgTx/committer 显式拒绝已取消上下文。释放租约不再删除 fencing 历史，避免 generation 从1重置为1（ownership-generation-red.jsonl 先复现，ownership-generation.jsonl 修复后通过）。

资源约束为最多16个已分配分区、每reader队列1、fetch不超过10MB。失败消息在当前分区原offset重试，不读取后续offset；正常revocation不记为持续readiness故障。旧adapter继续保留原接口，只有durable路径强制新接口。

kafka-generation-fencing-report.json 为真实 broker+PG 五轮通过、0 fail/skip：两个live组成员造成真实rebalance；旧未提交事务取消、旧token被拒、offset未提前推进，新代际只有1份效果。ownership-lease-contracts-report.json 为6个真实PG契约通过；ownership-unit-100.log 的取消/身份/轮换/native dispatch单元测试在race下重复100次通过。

实际native pipeline已新增live-rebalance子场景：启动第二个processor，旧token在TTL前失效且BeginFencedTx被拒；加入/离开两次generation变化后均继续索引。它和深度1/2/12、E06旧分支积压/repair-crash一起通过。E05必需manifest已纳入真实broker重组成员验收。ownership-fast.log通过；当前完整micro-loop和标准DoD正在复验，不能沿用上一轮124动作当作新实现的最终结果。

### 当前工作树合并复验：严格门禁与确定性 fixture

合并已有 T11/E04 改动后，完整门禁首先在 staticcheck 失败：新增触达的 query/subscription 仍引用 deprecated core 别名。按报告迁移到 ports 的同一类型，Mongo query tracer 改用共享 provider，并修复 S1021；没有关闭诊断或更改测试范围。ownership-alias-staticcheck.log 已通过。额外修复 required-cases 验证器两处假绿色：同ID多lane必须合并测试，而非覆盖；某个必需场景为空时，即使别的场景已通过也必须失败。RED/GREEN 自检与真实 CLI 反例均保留，反例矩阵升级为31/31。

第一次合并完整运行实际为128 pass、1 fail、0 skip，故障在 E04 初始追平、尚未注入重组；多轮复用 Anvil 累积历史使原有60s初始追平期限不足。失败原始流和失败report归档为 ownership-stale-history-contracts.jsonl / ownership-stale-history-report.json，未删除失败。随后使用全新独占 Anvil（固定1700000000时间戳、相同锁定imageID、三份部署runtime bytecode核验）模拟CI的新建环境；不放宽E04时限。fresh-fixture-1792058749/fixture-proof.json 记录fixture，当前新运行E04已经9.04s通过，完整suite仍在执行，结束后以新report为准。临时启动器只在退出时清理本次创建的两个 --rm 容器，原18565/18566节点未重置。

最新根DoD五项按序exit0；短测试7137 pass、0 fail、71已声明skip、5296顶级Test。ownership-workspace.log 记录七模块build/vet与六个嵌套模块short通过，根模块由DoD及当前race门禁覆盖。

独立的新数据库 chainpulse_delivery_state 上，T04/T06补充回归还复现 A→B→原始A 的canonical返回缺陷：当前DO NOTHING去重让原始A事件继续保持orphan状态。original-branch-return-red.jsonl 为0.09s真实PG失败，original-branch-return-repro_test.go 保留可执行复现。这是未修复P0，不被现有1/2/12重组绿色掩盖；后续需修复fenced重放恢复以及独立事件状态语义，整体方案仍未完成。


## 2026-10-15：原分支返回修复与凭据复核

真实 PostgreSQL 的 A→B→原始A 回归先红后绿：canonical-return-contract-red.jsonl 复现 canonical 查询缺失；修复仅允许匹配链的 fenced 事务恢复已有孤块状态，保留原 physical identity 和 raw payload。未受 fence 保护的重复插入仍不恢复孤块；未经 durable rollback 的冲突分支恢复随同事务回滚。canonical-return-contract-green.jsonl 为 race、count=5，额外断言替代分支仍为孤块、物理行2/canonical行1、跨链事件写入拒绝。required E02 新增实际回归名。独立 canonical/processing/finality 状态仍未完成，不能仅凭本修复把 T04 标为 Verified。

fresh-chain-old-checkpoint 组合运行失败后，已修正单体测试的初始化隔离：仅在专用测试数据库、第一次启动之前清空旧链状态，测试内部 restart 保留数据，不绕过 canonical 校验。原始失败日志保存为 ownership-fresh-chain-old-checkpoint.jsonl/.log；旧历史节点超长初始追赶失败另存 ownership-stale-history-*。最新重跑使用全新两条固定时间戳 Anvil、实际单体与四服务入口二进制，整轮状态以进程退出及报告为准。

当前 gh 账号已变更为 rtcdance，仓库 API 确认 push/admin=true，默认分支仍为 master。创建分支 ruleset 的 POST 返回 HTTP403 Resource not accessible by personal access token；尚未创建或修改门禁，github-ruleset-created.json 保存服务端错误。用户已收到补充 token 的 Administration:write 权限请求；不再把旧账号 pull-only 当成现状。真正门禁 check 名还须与 workflow 的最终显示名一致，未把 ruleset 当作已通过。四个历史 stash 的逐项 stat/path/sha 归档于 stash-inventory-current.json，全部保留，未 apply/pop/drop。


## 2026-10-15：T04 additive state 与 T01 汇总门禁接线

上一轮 canonical-return full 已真实 exit0：130 pass actions、59 顶级 Test、0 fail/skip。冻结副本 canonical-return-full/ 保留当轮完整报告；不把随后修改的代码冒充该轮源码。标准 DoD 的 canonical-return-dod-results.json 五项 exit0。

迁移16新增 event_state（canonicality/processing/finality）。旧 status 字段与原ID/原始载荷保留；兼容 trigger 不从 status=finalized 推导 RPC proof；orphan 回退保留 processed，finality 失效。fenced INSERT 在同一区块事务中直接写 processed，避免为每个正常新事件增加一次 UPDATE；仅已有孤块返回时恢复状态。PG所有共享行读者及实际REST响应增加独立 state，canonical查询检查该维度，重组边界保护显式 finalized evidence。已批准方案追加实现细化与应用先回退、schema后down的顺序，未改变 Approved 状态。

新状态真实PG提交/分叉契约当前4 pass/0 skip；API兼容测试通过；event-state-migrations.jsonl 为8 pass、0 skip，包含15→16→15→16的原行/载荷不变检查及完整fresh/down/reapply。前端31个Test及build通过，显示明确Finality标签；缺失/降级/过期/错误来源/孤块证据都不能显示Finalized。此处未声称生产RPC finality观察器已接入，实际双模式测试明确要求fixture为unknown；最终T07仍需实际provider失败/过期及观察写入闭环。

旧 runtime fixture 无迁移ledger，尝试普通升级在迁移4失败（自然键定义已为block_hash；保留旧库未force版本、未删数据）。为防止掩盖状态，以实际migrator从空库建立5个chainpulse_delivery_s16_*独立库并应用1–16；event-state-fresh-fixtures.log记录。新full正在这些fresh库与两条fresh Anvil上运行，包含实际API state断言。

CI汇总由只检查3个check改为 test/acceptance/required-checks.json 的10个必需check：计划7个lane加lint、integration、build-binaries。显示名固定为quality/unit-race/contracts/evm-e2e/frontend-api/security/coverage，汇总名acceptance-gate与拟配置ruleset一致。真实依赖workflow对全部PR及main/master push触发，不再靠paths过滤隐去必需lane。CLI支持versioned policy，空/重复/畸形policy拒绝；重名success不能覆盖failure。25/25实际CLI反例、31/31既有反例、自测及Actionlint通过，尚无远端run或ruleset成功结论。token权限追加后首次重试仍403，服务端响应明确administration=write；待当前凭据授权真实生效后再执行。


## 2026-10-15：权限恢复、真实 CI 修复及 finality RPC 校验

当前凭据已实际创建并读回 active ruleset `25154687`，默认分支 master/main 要求 PR 和 GitHub Actions `acceptance-gate`，无 bypass actor；负向验证 PR #2 因缺 required check 被 HTTP405 拒绝合并，随后关闭并删除专用分支。上述事实取代此前 403 的当前状态描述，历史失败证据保留。交付候选在独立 worktree 的 `codex/web3-go-backend-delivery`，草稿 PR #1；原工作区与四个历史 stash 保留。

候选 a36501c7ccc09cf83464821903cc8b1143be8041 的第三轮远端运行：Integration 38756973579 中 integration/contracts 已成功，真实 `Deployed canonical reorg and restart` 步骤成功（整个 evm-e2e job 尚在执行）；Performance Metrics 38756973418 的采集与回归检查均成功；Security 38756973391、Frontend API 38756973469 成功。unit-race 38756973318 仍失败：安装 Anvil 后原来的缺工具跳过转为实际运行，17 个 fixture 需 localhost:8545 节点，其中13项日志失败源为节点未启动；继续补齐进程启动、chain ID 探测与拥有者清理，不能扩大 skip allowlist。quality Docker 构建仍在执行，未将整个 CI 标为通过。

本地 readiness 修复后完整 micro-loop exit0：131 pass actions、60 顶层测试、0 fail/0 skip；31/31 门禁反例通过。此前 full 证据复制到 pre-readiness-full，保留运行边界。新增 finality RPC 反例已用旧候选源 overlay 真实复现：HTTP错误、非法hex/overflow、非规范quantity、错误JSON-RPC ID/version、尾部响应和切换provider旧缓存。当前修复拒绝这些响应，限制响应体2MiB，provider重新注册清缓存并拒绝旧in-flight结果覆盖；状态接口的降级结果 IsFinalized=false，height与元数据由同一次快照返回。新增回归100次race与整个finality包race通过。此项只证明 checker 改进，尚不能替代生产 finality observer 的 chain/genesis/header/freshness 全链路接线验收。

T14 微基准工具新增 fail-closed 校验：每个样本必须有有限正 ns/op 和非负有限 B/op，缺/部分指标不能通过；零分配基线合法，但从0增加分配失败。15%阈值未改变；旧代码自测反例真实失败，修复后自测及31/31外部CLI反例通过。正式100k双链正确性、持续摄取、查询、积压、30min soak、不可变镜像发布/回滚与最终作者签字仍未完成，不将本轮局部结果当作T00–T18全部完成。

第三轮终态补充：真实部署四场景步骤已成功，后续 legacy E2E 因同一 localhost:8545 fixture 缺失而失败；两个调用该套件的 workflow 均补固定 Anvil 进程和探测。18 项相关并发测试（两源文件全量）本地 race 复核为18pass/0fail/0skip。初次本地筛选只包含13项而误设min17，检验器正确拒绝空覆盖；已改为从两文件函数名生成精确18项筛选，原始记录保留，不把数量不足当通过。quality 本轮最终成功，但超过方案5min预算，仍需优化和复测；acceptance-gate因unit-race/legacy E2E失败正确阻断。


## 2026-10-15：T01 lane 时间与覆盖拆分

为修复 quality 超过5min，把五目标 Docker 构建移到独立必需 `image-build`（20min timeout），required-checks.json由10项增加到11项，外层 ruleset 仍要求同名 acceptance-gate。quality 保留每个workspace模块build/vet和格式、manifest校验；重复govulncheck交给已必需的security。原root短测覆盖迁至unit-race：pkg、cmd、integration、e2e均显式-race，动态枚举6个嵌套module执行-race，动态补齐剩余root package。quality 5min、unit-race 10min预算直接用job timeout执行，仍须远端验证冷缓存预算，未宣称已达标。

check-workspace-modules.sh --no-test 的旧输出错误地带“and tested”，已修为明确“tests not requested”。本地7模块build/vet通过；剩余root包14pass/0fail/2skip，两项原有 MemoryMQ stress 的testing.Short主动跳过已逐项分类为GAP，既非删除测试也非正式T14验收证明。image-build缺失/失败/skipped均被汇总拒绝，success正向控制通过（4/4）。

新增六类真实CI故障注入入口，仅workflow_dispatch显式选择启用。每次在临时零依赖Go module先运行真实正向fixture，再注入编译/断言/timeout/空跑/必需skip/缺coverage，保存Go JSONL、退出码与检验器输出；本地6/6得到预期拒绝，基础设施失败单独exit2。远端将使用拥有独立marker commit的专用分支，避免故障check污染业务候选SHA；仍待保留六条远端失败run URL，不能把本地控制当成远端验收。

T01 新分层首轮 e7ab3a2：远端CI 38760066608的quality于15:24:40Z–15:25:23Z成功（43s），独立image-build成功；这证明该次运行在预算内，冷缓存专项仍未验证。六类故障专用SHA为b613d8507ac81cec4f9860d7b5c5717465d90027，dispatch run依次为compile 38760458524、assertion 38760473196、timeout 38760483351、empty 38760493439、required-skip 38760503820、coverage 38760515542。前五个已按预期失败，其中四份下载report已确认positive_control_passed与injection_rejected均为true；timeout工件遇一次TLS下载超时待重试，coverage仍排队。不能把下载失败算为注入验收通过。

第四轮quality 38758799184另有真实旧测试时序误报：TestDrainControllerWaitsForInFlightWork在End()之后close(finished)，正确Drain可能抢先返回。已使用testing/synctest在所有等待者阻塞时检查Drain尚未返回，再End并验证释放；100次race通过。跳过实际Wait的错误版本overlay会被新测试稳定拒绝。候选ab40d59仅改此测试；DoD的build/check/lint/test-short日志均通过，lint0issues。cmd覆盖337pass actions/233顶层/0skip，六嵌套module race通过；本次没有生产Drain行为变化。

T01本轮收尾证据：e7ab3a2的quality=43s、unit-race=422s、image-build=214s，均success；cmd与nested race和剩余包范围已接线，两个既有short stress跳过明确登记GAP而非正式T14证据。六个专用远端故障运行全部failure，六份report均已下载核验positive_control_passed=true且injection_rejected=true，完整映射见ci-negative-remote/verified-matrix.json；因此可确认失败来自正确注入而非基础设施不可用。临时故障分支已请求删除，managed worktree已请求归档，源commit和run工件保留。候选原命令make build/make test-unit/make vet全部exit0，原始日志candidate-pr-make-*.log；不将这些局部结果替代最终T18。

## 2026-10-15：Coverage Diff 独立旧故障

e7ab3a2的coverage主job成功，但Coverage Diff的Compare coverage因heredoc缩进导致bash语法错误而失败；不是40%阈值失败，不能误归因为新的drain测试。保留remote-fifth-coverage-failed.log。修复独立比较报告：base checkout固定PR base SHA，与候选同口径-short；profile下载、base测试与JSON检验均不再忽略错误；新coverage-diff.py校验原始profile、合并重复block并拒绝缺失/空/格式错误/零statement输入。比较百分比取自两个匹配checkout各自go tool cover原生报告，不填虚假0。数值与语义有自测；summary替代自动PR评论，MIN_COVERAGE=40保持不变。该修复仍待远端验证，不把工具自测当整条CI通过。

真实profile交叉核验纠正：e7ab3a2原始block总计20904/30084=69.4854%，但go tool cover函数汇总为69.6%；两种口径不能混用。首次工具版本错误地把raw block比例用于比较，已保留失败校验记录并改为要求两个原生functions报告，工作流在各自源码目录执行go tool cover；raw block统计仅作diagnostic。使用同一真实profile/原生summary的工具正向验证返回69.6%和0差值，只证明解析器一致，不宣称base→candidate覆盖率变化。


## 2026-10-16：冻结候选通过与T07持久证据接线

7636755a67ea57e42ac59f63208cc1c37363d1e7已保持稳定等待，Integration 38762492657的integration/contracts/evm-e2e全部success，CI 38762492719的quality/image-build/acceptance-gate全部success。11个required checks已真实汇总通过。附加Coverage Diff报告失败原因已缩小至master基线自身5个SIWE测试动作失败（父Test含4个子Test），不是候选coverage失败。40%主gate成功；旧基线失败的部分profile不得用作比较。后续报告明确比较不可用、保留基线JSON和原始退出码，无差值/基线0；有效候选profile仍先严格验证。该比较是信息报告，未给现有40%门禁新增基线成功条件。

T07新代码仅在原工作区，尚未进入7636755。migration17为chain/tag观察表，16→17→16两轮真实PG测试通过，raw/ID/state原样保持；schema主测试也补入此前遗漏的reorg_generations表以覆盖真实回滚。独立chainpulse_delivery_s17_contracts/runtime/micro/dual_mono/dual_micro均由CLI正常迁到17，不改16版历史验收库。

真实PG finality合同6pass/0fail/0skip：只有声明链和safe/finalized可写；验证提交checkpoint/header、genesis pin、时间、generation；拒绝ahead/stale/future/wrong header/genesis及finalized回退；provider故障保留已验证finalized边界；其上的reorg后fresh proof可绑定新generation。query在事件相同repeatable-read事务中批量投影，原事件State不批量改写。固定clock投影100次race通过。

RPC观察在tag/canonical header两侧校验chain/genesis，latest不回退；wrong chain/genesis/header/mid-probe切换与原canonical RPC合同联合100次race通过。共享observer只对显式tag策略启动，默认确认数/最新头不冒充finality；每链一个5s轮询与5s请求界限，取消后join再关闭客户端，单体和native processor均已接入。别名策略冲突由registry拒绝。observer生命周期测试自身最初计数器共享访问被race发现，已改atomic并保存原始失败后100次通过。build/check/lint均通过，lint0issues。

E07已追加到两种现有真实进程fixture的尾部，受控RPC仅替换safe/finalized标签和错误chain，header/事件仍取真实Anvil；查询实际HTTP API断言safe/finalized/unknown-degraded/expiry/recovery，并保存各mode的finality-api.json。当前正在运行full-finality完整回归，尚无完成结论；不得把单元/PG通过代替该实际部署结果。


## 2026-10-16：full-finality 收尾与 monolithic 链别名重复缺陷修复

上一节的full-finality完整回归已实际结束：`micro-full-finality.log` 与随后在全新双链fixture上重跑的 `micro-full-finality-guard.log` 均 exit 0，`full-finality-guard/contracts-report.json` 为139 pass actions、61 顶级 Test、0 fail、0 skip，E07_finality_observation 子场景在两个进程 fixture 中真实通过（52s/52.9s）；`full-finality-guard/monolithic-finality-api.json` 与 `microservices-finality-api.json` 记录 safe/unavailable-degraded/wrong-chain/expiry/recovery 全序列，29→31/31 门禁反例矩阵通过。原始工件保留在 `finality-guard-deployed/` 与 `finality-guard-fixture-1792081291/`。

E07 收尾期间上一轮在 00:28 落下的 RED 测试 `chain_identity_alias_test.go`（`TestMonolithicChainsRejectDuplicateNetworkAliases`）一直没有转绿：monolithic 的 `parseChains` 只按字面逗号切分，`CHAINS=ethereum,1` 会把同一网络索引两次，而 durable processor 已拒绝同身份重复（`durable_runtime.go`），finality registry 也拒绝 alias 冲突——单体入口是这条防线的缺口。本轮修复：`parseChains` 现在按 `chainid.ResolveChainID` 解析出的数字身份去重，解析失败的条目按小写字面去重；`ethereum,1`、`31337,31337`、`polygon,137` 被拒绝，`ethereum,polygon` 保持原样返回。RED→GREEN 见 `finality-mono-alias-red.log`（原始失败）与 `finality-mono-alias-green.log`（race + count=100 通过）。

源码变更后标准 DoD 按序重新执行（`alias-fix-build.log` 等五份原始输出）：build/check/lint(0 issues)/test-short/diff-stat 五项 exit 0，短测试 0 fail、0 skip。随后在全新双 Anvil fixture（固定1700000000时间戳、锁定 imageID、EventEmitter/SwapEmitter runtime bytecode 核验，`alias-fix-fixture-1792112595/fixture-proof.json`）上重建全部五个 -race 入口二进制并重跑完整 micro-loop：`micro-full-alias-fix.log` exit 0，`full-alias-fix/contracts-report.json` 为139 pass actions、61 顶级 Test、0 fail、0 skip（含 1/2/12 重组、四服务 live-rebalance、E06 repair-crash、双链模式 oracle、E04 故障隔离、E07 finality），迁移与指标步骤 pass、31/31 门禁反例、OpenAPI drift、staticcheck 均通过。

过程记录：本轮首次 full 运行在 staticcheck 步骤失败，原因是机器上的 staticcheck 二进制为 go1.24 构建（`module requires at least go1.26.0, but Staticcheck was built with go1.24.13`）；按 Makefile 既有方式 `GOROOT= go install honnef.co/go/tools/cmd/staticcheck@latest` 重建（2026.2.2/go1.26）后通过。第二次失败为环境变量缺失（未带隔离 DB 配置直接调脚本），改用带完整 env 的 fixture 包装脚本（`/tmp/chainpulse-run-alias-fix-fresh-full-20261016.sh`）后通过。两次失败输出都被同一路径的最终成功运行覆盖，仅存于当轮会话记录，没有独立工件——这是本文件已知的一处证据缺口，如实登记而不补造日志。

仍为 dirty 工作树证据：342 dirty 文件本轮未提交、未清理。仍未完成（不得以本节缩减）：独立事件状态语义、完整 SLO/性能/soak（T14：250eps/600s、100 req/s/600s、积压恢复、30min soak 实测）、镜像版本回滚与 SBOM/security lane 完整范围、T00 干净候选与同 SHA 远端 CI/CD、T17/T18 封版。

## 2026-10-16：T14 查询失败审计与负载驱动器归档

权限已恢复，master ruleset 25154687 与 required acceptance-gate 已验证；此前7636755的11项必需检查成功只对应当时源码，不能代替后续T07/T14改动的CI。

T14本机探索性测量：持续摄取记录250eps/600s、150000窗口内事件、poll P99=1852.2ms；查询记录60000计划请求中15个503，P95=585991.99ms/P99=614973.2ms，未达标。原始 query_raw.jsonl 为append模式，含192000行、60000不同idx、最多重复5次，跨多次attempt，不能作为独立运行证据。积压记录34.5s恢复，但暂停期采样峰值14600低于15000目标，精确暂停边界与输入计数仍需复核；30min soak未完成。这些均为Mac本地dirty源码探索数据，不是计划约定Linux8-vCPU正式结果。

新增可版本化 scripts/bench/query-load.py：新建输出目录及独占原始文件、有界队列、固定种子、monotonic计划到达时间、包括失败请求的端到端延迟、保留503状态、过载显式失败、每计划请求唯一终态核验。输出明确只验证HTTP/响应结构及性能，e14_verified=false，不将结构检查冒充完整canonical正确性。9项回归测试通过；已接入quality workflow和full micro-loop。布尔total反例先失败后修复；首次RED日志同时有沙箱HTTP监听拒绝，GREEN在授权本机监听环境重跑全部9项通过，未把环境拒绝计入代码反例。原始日志见query-load-boolean-red.log、query-load-regression-green.log。

下一步：候选分支归档既有T07和查询索引改动并重新执行门禁；定位重复COUNT/统计聚合开销，使用全新工件目录复测。T00–T18整体保持未完成；此前“独立事件状态未完成”的概述不能覆盖已存在的T04/T07局部证据。

## 2026-10-16：候选 d00ad28 远端核验与 T11 计数缓存基础

已按主题提交并推送 c29dd21（finality/查询索引）、fef98d2（隔离压测驱动器）、d00ad28（CI与必需case）；候选worktree干净。11项required checks及acceptance-gate全部success。原始check快照为candidate-d00ad28-check-runs-final.json；远端CI38879254062、Integration38879254068。PR实际checkout为synthetic merge aa22306abdb34f12a62e7855363e3bcbca8d4eea，其父提交由GitHub git API核验为master fa80b83和候选d00ad28；因此这是对应head的PR集成验证，不冒称T18最终tag SHA执行。实际部署EVM工件deployed-reorg-report为16 pass actions/4顶层/0fail/0skip，两种mode的E07_finality_observation均pass。旧E2E报告371pass/32skip，32个已分类历史skip没有被称为已执行。coverage必需gate成功；附加比较报告成功生成但comparison_status=unavailable_base_tests_failed，候选Go-native为69.4%，旧基线失败，不虚构差值。

候选本地原样make build/make test-unit/make vet全部通过；build/check/lint/test-short/diff顺序门禁通过，提交业务后再次lint0issues。前端缺node_modules首跑未执行，npm ci后32测试+build成功。迁移5顶层/10pass/0fail/0skip。16份原始本地日志归档candidate-d00ad28-local-gates/，PR#1仍为draft。

随后仅原工作区继续T11：migration20增加同写事务的chain mutation revision/epoch与INSERT/UPDATE/DELETE/TRUNCATE触发器，包含未推进checkpoint的兼容写者；schema重建epoch改变，避免旧Redis键复用。测试先因无migration20失败；首次实现测试错误地将重插created_at与旧值比较，已用固定fixture时间并把完整行保留在down/up期间后修复，原始失败没有覆盖。全部迁移6顶层/11pass/0fail/0skip；新增迁移100次race为100pass/0fail/0skip。

精确count缓存已进入QueryDomainEvents：同一repeatable-read snapshot读取chain revision向量+filter构造键，页与finality证明仍读PG；异常或非整数/负数载荷回源。配置未启用时保持原路径。专用Redis后端无本地map回退，50ms操作期限、8条活跃连接上限、GETRANGE限制64KiB、TTL上限300s。两项真实PG/Redis合同连续100次为200pass/0fail/0skip，覆盖实际命中、兼容SQL重组/删除后的失效、缓存故障与畸形载荷回源；Redis边界单元100次race通过。新DB为chainpulse_delivery_s20_cache_contracts，由CLI正常up20创建，不force旧库。当前尚未接入实际入口与统计聚合缓存；不能据此宣布性能或T11完成。

压测驱动器进一步修复写盘失败时往满队列塞shutdown sentinel造成的死锁。初次简单负控未复现，原始失败保留query-load-writer-negative-control.log；改为事件屏障使worker先进入、排队项滞留，再触发写盘错误，冻结d00ad28版本被父进程3s超时杀掉，修复版本明确失败退出，证据query-load-writer-controlled-negative.log。全部10项回归通过。新增计数缓存后的micro-fast通过；标准DoD正在单独运行。新增缓存/迁移/生命周期改动尚未提交到候选，不属于d00ad28已绿范围。

整体交付仍未完成：实际缓存/原生订阅接线、严格100k oracle/250eps/冷热查询/积压边界/30min soak及资源、完整镜像版本回滚和最终证据仍需继续，保持原Approved范围。

T11计数缓存基础的标准DoD已结束：query-count-cache-build/check/lint/test-short/diff-stat五份日志对应exit0，lint原始输出0issues；query-count-cache-micro-fast.log也以[micro-loop] done结束。required-cases追加E12的contracts/migration/unit三lane，迁移必需case检验通过。仍未执行新的全栈full回归、未接实际入口，不将该基础改动标为整体完成。

## 2026-10-16：T11 实际缓存接线、启动负控与 T14 短诊断

已将revision绑定的精确计数与dashboard聚合缓存接到实际单体和canonical API-service。GROUPING SETS在一次canonical扫描中取total/by-chain/by-name，另计reorged；缓存payload检查分组总和、scope、非负数与容量。Redis只保存计数/聚合，页面和finality每次读同一PG snapshot；冷/故障缓存回源。单体storage wrapper拥有独立Redis client并在Stop关闭，API runtime关闭前detach；关闭出错也继续DB cleanup。缓存启用前校验clean schema≥20和两条启用的mutation triggers；链范围在一次查询中只读取一次，SQL与cache key一致。

真实负控发现并修正API-service异步Query.Start失败只打印错误、主进程仍宣告Running。改为在HTTP启动前同步启动非阻塞query runtime，拥有独立lifetime context，失败关闭wiring并exit1。最初外部负控20s超时未通过，随后固化进程测试RED为12s超时，均保留；GREEN实际schema19以exit1拒绝且无Running消息。新测试创建/删除独立schema19库，随机空闲端口，产生日志/JSON各自唯一文件，不以killed/timeout算拒绝。

full-cache-runtime-v1实际失败，原因是native fixture只向gateway传Redis地址，API缓存回到默认端口；新增E12进程检查发现指定Redis的GETRANGE增量为0。保留v1原始JSONL、failure-report和mono成功样本。补齐native common Redis环境后，在全新双Anvil重跑full-cache-runtime-v2：145 pass actions/65顶层/0fail/0skip；两mode E12都真实通过（mono canonical128、micro96，均GETRANGE=4/hits=2，API与PG一致）；schema19启动负控通过。迁移、静态检查、OpenAPI、14项指标合同、31/31门禁反例和full结尾均通过。当前生产二进制由/tmp/chainpulse-run-cache-runtime-full-v2-20261016.sh重建，full日志和fresh fixture proof均保留；这仍是dirty源码本地证明。

T14工具可版本化：query-oracle-export.py参数化psql，在repeatable-read快照导出每条canonical ID/chain/block/log/address/tx/name及reorged统计；QueryOracle用索引/bisect检查有序offset/range/cursor页、total与统计分组。输入snapshot不是RPC独立canonical oracle，结果明确E14=false。固定seed mix在测前细化为30%offset/25%cursor/15%stats/20%contract/10%range；oracle/hash/原始记录独占存储，失败响应与队列等待均计入，错误2xx容忍0；加入健康预检与URL凭据拒绝。18项工具控制包含真实HTTP错误响应的CLI反例。不得将应用cache-off称为磁盘冷缓存。

实际硬件补录：Mac物理10核心，Docker Desktop VM6CPU/约8GB，任务Postgres限制2CPU/1GiB（shared_buffers128MB、work_mem4MB）；不是计划Linux8CPU/16GB参考数字。短诊断基于274900条静态事件，前后revision相同。第一轮独立启动脚本返回后所有后台进程终结，3000请求URLError，明确FAIL；保存旧栈日志并确认所有PID已死后清理，未当成成功。第二轮用同一持续会话包住启动/测量/清理：30s/100rps/3000条终态，2888个200全部精确oracle通过，112次generator_overload、0响应正确性错误，P95=1127.55ms/P99=2419.63ms，仍FAIL。过载集中首5秒；后25秒无过载，不能裁掉前段宣称验收通过。当前缓存冷启动重复聚合并发是下一项待验证假设；不修改现有门槛。

SQL诊断保留：offset宽行67.7ms，先ID再回表28.5ms；contract宽行本身0.639ms，事务内临时覆盖索引实验8.68ms更慢，已ROLLBACK，未擅自留下新索引或修改历史migration19。规定600s冷热查询、100k独立RPC oracle、250eps/600s复测、积压精确边界与30min soak尚未闭环。全计划仍Approved，原生committed/reorg订阅桥及其他T00–T18残项未缩减。

当前缓存候选的标准DoD与18项工具控制均exit0；lint0issues。完整计数/统计真实合同另经100轮race：300pass/0fail/0skip。真实API schema19启动负控另经100轮race：100pass/0fail/0skip，100个独立数据库均正常清理，100份唯一进程工件保留。该结果没有把30秒负载FAIL改成PASS。后续需带对照验证冷启动重复聚合、按完整600秒/soak合同复测。

CI接线核对另补：evm-e2e选择正则由4个顶层进程case扩大到5个，明确包含schema19启动拒绝，min-tests同步5。contracts lane没有部署二进制时该新增case实测1个dependency-gated skip，逐项分类并给出到期日；单独manifest反控确认即使该skip已分类，required evm-e2e仍明确拒绝“was SKIPPED (required skips must be 0)”，不是放宽必需case。原始skip流与反控报告已保留。

## 2026-10-16：T14 六次 attempt 的查询/积压验收全过程

T14 固定语料与 250eps/600s 摄取在 attempt 2 已 PASS（ingest_result.json 保留）。查询与积压 phase 之后经历六次 attempt，全部原始证据按 attempt1…attempt6 目录保留，失败工件一律未删除：

- **attempt 1**（查询失败）：t14_stack.sh 的 Kafka topic 只开 3 分区，fnv1a("31337")%3 == fnv1a("31338")%3 == 2，两条 fixture 链全部落到同一分区，单 worker 串行消费。改为 4 分区（31337->0、31338->1）并在 t14_stack.sh 注释里记录该分区契约（partition_contract.go F28）。
- **attempt 3**（诊断）：串行慢查询（log_min_duration_statement=100）暴露 page 查询在 lib/pq 预处理语句下走 147MB external merge 排序，而 psql 字面量 EXPLAIN 为 6.6ms 无 Sort——通用计划（custom→generic）切换放大了链别名展开的缺陷。
- **attempt 4**（查询+积压 FAIL，三层根因）：① harness 5s 客户端超时在 handler 30s 预算内砍断在途请求（1267 次 context canceled），连续 5 次失败打开网关熔断（5/30s），后续 50804 个请求瞬时 502；② `sem.acquire()` 阻塞在派发循环内，60000 个请求用了 4105s 而非 600s，rate 门被“通过”；③ 产品侧 `chain_id IN ('31337','31337','31337')`（event_query_handler.go 链别名展开三值全相同）使 PG 拒绝 idx_events_canonical_chain_page 有序索引扫描，全量 Index Only Scan 123675 行 + Disk=147424kB 排序 = 762ms，而单值等式 6.6ms。积压 FAIL 判据也根因明确：resume 过渡样本（16150）仍在增长，旧判据拿它对 paused peak（15100）+550 比较必然失败。
- **attempt 5**（积压 4/4 PASS，查询延迟 FAIL）：修复后 rate/error/correctness 全 PASS（dispatch 599.99s、错误 0.025%、正确性 0），但 100req/s 冷路径稳态完成能力只有约 50rps：service p50 411ms（contract 1385ms、stats 911ms），queue_wait p50 311s，p95=585992ms。RSS 峰值 617MB<1GiB。这轮诚实判定为 FAIL，未移动任何门槛。
- **attempt 6**（查询 4/4 PASS + 积压 4/4 PASS）：数据库侧 revision 绑定的聚合缓存（schema 000020 canonical_query_revisions + Redis canonical-aggregate）接线后，60000/60000 请求全部 200：P95=33.73ms（门槛100）、P99=97.39ms（门槛250）、错误率 0%、correctness 0；queue_wait p50=0.04ms 说明队列不积压。积压 4/4：峰值 14500、恢复后 64s 内清零（门槛120s）、输入全程未停、转移样本之后单调收缩。RSS 四进程峰值 112MB。attempt6/rpc_cost_result.json 记录 RPC 成本：两链 71,530 次调用、291MB、429=0、5xx=0，每 10,000 块 57,741 次调用、每 1,000 事件 230.9 次调用；anvil 无定价，USD 标 unknown（§8 禁止杜撰统一单价）。

冷热缓存分别报告（cache_cold_hot.json）：stats 冷 446.3ms → 热 3.4–4.9ms（Redis revision 缓存）；list 无响应缓存冷热同源（38.5 → 8–16ms，属 buffer 热度差异）；contract 首次 1040ms、随后 8–12ms。产品 HTTP CacheMiddleware 仍未接线（health 仍报 cache-plugin-not-wired），本项按"冷热缓存分别报告"以实测冷/热数值交付，不虚报接线状态。

伴随修复：postgres_domain_query.go `$in` 分支对绑定值去重（重复 IN 值不再摧毁有序索引扫描，注释里记录 762ms→6.6ms 实测）；迁移 000019（canonical contract 部分索引，contract count 400ms→118ms）与 000020；t14_query.py 超时 5s→10s + 队列化派发（dispatch_seconds 门）；t14_backlog.py 峰值取 resume 瞬间、转移样本之后才要求单调收缩。

30min soak：soak-check.sh 断言生产安全姿态，栈原先关闭 auth/rate-limit 导致 sample 1 即失败；已为栈启用 auth+rate-limit（上限 600 万/分钟，不整形 §8 负载）并让 harness 带 X-API-Key。期间发现 event-processor 的 durable 路径 /runtime/summary 返回 "microservices"/"durable-block-v1"，与 rollout_report_parity.go 及其余服务的 canonical 值（core.DeploymentModeMicroservice="microservice"/"runtime-wired"）不一致、也与 soak 契约不符，已修正为 canonical 值（gofmt/build/vet 通过）。

## 2026-10-16：metrics 导出修复与 30min soak 两次 attempt

**metrics 修复**：soak 期间发现四服务 /metrics 缺 histogram/go_goroutines/go_memstats（soak 资源证据依赖 go_memstats_alloc_bytes）。根因是 MetricLabelGuard 未实现 core.PrometheusMetricsExporter 接口，core.ExportMetricsPrometheus 走 fallback 分支。修复：pkg/observability/metric_labels.go 增加 ExportPrometheus() 委托 core.ExportMetricsPrometheus(g.inner)。修复前后对照证据存 metrics-before/*.txt 与 metrics-after/*.txt：gateway 由 1835B（go_goroutines=0 go_memstats=0 buckets=0）变为 4414B（go_goroutines=1 go_memstats=2 buckets=22，go_memstats_alloc_bytes=2383240）。注意 /metrics 需带 X-API-Key（栈开 auth，401 属预期）。修复后四部署二进制于 21:42 重建（不带 -race）、21:43 重启 T14 栈（RUN_ID=1792158178）。采样器 t14_resource.py 同步增加 goroutines/heap_alloc/heap_sys 采样（含 scrape 容错，单点失败不中断 RSS 序列）。

**attempt7（FAIL，完整归档 soak30-attempt7-fail/）**：soak-check 30/30 通过、correctness 0，但查询门 P99=876.61ms > 250ms，诚实判 FAIL，门槛未移动。诊断：两个整机 stall 窗口（offset 511–541s、971–1081s），842 条 >1s 请求中 793 条集中在 1 分钟；排除窗口后 p95=29.8ms p99=74.4ms；gateway 无 ERROR；PG 日志在 13:59:55→14:00:05 有 10s 断口；资源采样器自身在同窗口卡 12.3s/14.6s（名义 5s）；`log show` 证实两窗口均有 jetsam memorystatus killing（第二窗 174 条），swap 8.3/9.2GB。结论为 host 级内存压力 stall（macOS 本机非 §8 参考环境），非产品缺陷，按 FAIL 归档后轮转压缩 54MB 日志重跑。

**attempt8（PASS，归档 soak30/）**：supervisor 用 Python double-fork + setsid 脱离 job teardown（macOS 无 setsid），同窗执行 30min soak-check（DURATION_SECONDS=1800）+ t14_query.py --rate 50 --duration 1800（90000 请求）。窗口 2026-10-16T14:54:45Z→15:24:55Z，soak_rc=0 query_rc=0，Soak check passed 30/30。查询 4/4：90000/90000 全 200，P95=30.1ms（门槛100）、P99=62.17ms（门槛250）、错误率 0%（门槛0.1%）、correctness 0。资源（234 个 5s 采样）：四进程总 RSS 峰值 126.9MB ≤ 1GiB；heap_alloc 斜率 -0.013MiB/min、末段/首段比 1.005 < 1.25 → 30min 结束 GC 后 heap 不持续增长；goroutines 67–102 稳定。verdict 全绿见 soak_verdict.json（all_ok=true）。attempt7 的失败证据（含 jetsam 日志）保留在 soak30-attempt7-fail/ 未删除；修复前旧 soak 产物保留在 soak30-pre-metrics-fix/。

本轮 metric_labels.go 变更后 DoD 五门禁重跑全部通过：go build ./... exit 0；make check（vet+gofumpt）exit 0 "All checks passed"；make lint exit 0 "0 issues"；make test-short exit 0（66 ok / 0 FAIL）；git diff --stat 仅含工作区既有 dirty 文件（201 files，无计划外新增）。


## 2026-10-17：验收范围复核与冷启动配对诊断（进行中）

远端候选9cd933e1897d76217b25fc6b184046adfd1d0d85的11个required checks及acceptance-gate已重新读取为success；active ruleset25154687可读取。PR实际merge SHA为29e91762fe095fcbdb2ea67556c64d1d3753cc92，这些结果不替代最终tag SHA验收。

复核上一节的证据范围：attempt6确为600s/100rps的延迟/状态码样本，但mix没有range、correctness只检查响应形状。soak30确有1800s/50rps和30次soak-check，尚未证明末尾显式GC，也没有独立RPC canonical oracle。因此上一节“PASS”和“GC后”只保留作当轮harness结论，不能据此将完整E14标Verified。required-cases中PerformanceManager的10项测试只证明测量/判定工具；已明确另列pending release-candidate E14，禁止将工具单测视为实际负载验收。全部失败工件保留。

查询聚合增加最多16个同修订in-flight键、inline owner、取消/异常清理和一次共享取消重试，测试使用synctest/barrier。aggregate-flight-unit-all-race100：800pass/0fail/0skip；aggregate-flight-concurrent-contract-race100-v2：300pass/0fail/0skip；aggregate-flight-contracts-race100：300pass/0fail/0skip。它们是dirty工作树证明，尚未提交候选。真实分页排序/过滤/尾部合同offset-ids-contract-race100-v2：100pass/0fail/0skip；首次测试uint/uint64编译失败原始JSONL保留。

独立端口24084/24085、RedisDB14配对诊断，不重启或修改既有T14四服务。race构建v3两组都失败；普通构建v4 baseline：510error、P95 2958.69ms/P99 4735.51ms；coalesced：317error、P95 2198.52ms/P99 3985.98ms，全部FAIL。每组30s/100rps/3000条终态，全部成功响应精确PG snapshot校验无错误，前后revision一致；PG oracle不等于独立RPC truth。这些失败不能裁掉冷启动窗口；也不冒充600秒正式验收。正在以offset IDs后回表方案继续配对；不放宽门槛、不新增索引。


冷启动v5普通构建配对仍FAIL：baseline645错误、P95 3215.49ms/P99 6060.79ms；offset+coalescing414错误、P95 2528.67ms/P99 4587.18ms。全部成功响应精确PG oracle错误0，原始3000条请求及快照/metrics/GC/退出码完整保留。数据库EXPLAIN证明contract count旧索引为heap Index Scan，154825行/31306 buffers约617.93ms；事务内count-only覆盖索引约61.99ms，但会引入页面排序风险，未采用；带完整block/log/id顺序的覆盖索引count为Index Only Scan约132.24ms，page约0.11ms，实测65MiB。两次实验均ROLLBACK，无诊断索引残留。

新增migration21仅增加该覆盖索引，保留18/19/20，事务内lock_timeout3s/statement_timeout5min；上/下迁移需维护窗口，写阻塞与空间要求已写入批准spec。先写测试RED（没有migration21真实失败），再实现；contract-cover-migration-race100：100pass/0fail/0skip，20→21→20两轮且全部event/revision/epoch摘要一致。fast candidate gate通过。18项查询工具控制首次因沙箱禁止本地HTTP监听失败，使用允许本地socket的环境重跑18/18通过，失败日志保留。


隔离候选工作树本轮完整micro-loop（full-cold-query-v1）exit0：真实PG/Redis/Kafka/故障/双部署/双链/raw RPC摘要/finality/cache共149pass actions、67顶层、0fail/0skip；migration12pass actions、7顶层、0fail/0skip；指标合同14/14、查询工具18/18、门禁反控31/31。full unit历史skip实际71项，未将其记为0，也不将非-v test-short不显示SKIP当作没有跳过。顺序DoD build/check/lint/test-short/diff-stat均exit0，lint0issues；原始日志在candidate-cold-query-local-gates。候选变更仅显式copy清单，包含新文件的范围另由git status/staged stat核对，未使用只含tracked文件的diff-stat隐瞒新文件。既有四服务T14栈和4个stash均保留。

独占chainpulse_t14性能库已通过真实migrate up到clean21（version21/dirty=false），新索引indisvalid/indisready=true，65MiB。当前继续同seed/rate/冷Redis范围诊断，不将新索引SQL实验或数据库升级本身视为§8性能通过；完整E14、原生订阅桥、CD/真实digest回滚及最终封版仍待完成。


schema21冷缓存v6诊断仍FAIL：3000终态、333错误（326 generator_overload、7个stats503）、P95 1201.56ms/P99 4568.49ms、成功响应精确PG校验错误0。数据库慢日志显示首个stats聚合因5s上下文取消，约7个共享等待请求返回503；同期contract count在实际并发下3.1–4.46s，串行EXPLAIN改善未转化为冷启动负载通过。已保存aggregate-cover-v6及pg-startup.log，不将SQL实验或完整功能门禁替代性能验收。下一步检查参数化真实计划、冷启动并发下资源/等待与发压器自身开销；禁止裁掉首段或提高性能门槛。


候选业务提交a415001（13个显式文件）及配置提交89af157已推送，隔离工作树clean。提交后权威make lint仍0issues，make build实跑全部二进制exit0。新CI地址已归档candidate-89af157-run-urls.json，当前只确认frontend-api success，其余尚在运行；上一候选全绿不能代替新候选CI结论。

auto_explain仅通过本次连接URL options开启，未修改数据库全局设置；v7诊断已退出并清理自有API/gateway。实际参数化contract count使用schema21 Index Only Scan，无全量heap退化；两链冷count约1.25s/0.95s，stats在canonical page index上Index Only Scan约1.91s（并发下；含分析仪开销），无大排序。v7全部3000请求200且精确PG校验错误0，但P95=539.22ms/P99=1045.12ms，仍FAIL，且instrumented诊断不能替代正式性能。需要继续区分工作集/PG缓存与并发资源影响，并以同参数完整600s冷热复测；仍禁止修改门槛或裁掉首段。


## 2026-10-17：T13 指标、摘要与诊断日志增量（进行中）

上一goal turn完成查询业务a415001与验收配置89af157的实际提交/推送及full/DoD证据，判为progress。当前89af157远端contracts/integration已实读success，evm-e2e仍在运行；其余结论按实际run/check逐项核对，不借上一候选结果替代。

本轮把夜间dirty工作树已有的MetricLabelGuard.ExportPrometheus接入隔离候选，补bucket与敏感高基数标签不泄漏断言。durable processor真实摘要provider统一canonical字段，同时保留indexing/consumer细节；stats失败仅对操作者日志输出可读字符串，HTTP503仍为脱敏STATS_FAILED。新增用例先在旧候选RED：缺Exporter能力、缺durable helper产生编译失败，stats日志字段原错误对象断言失败；再实现。telemetry-closure-race100：4顶层/400pass/0fail/0skip；fast通过；顺序DoD五步均exit0、lint0issues。完整真实部署回归运行中，未提前标记通过。

T11接线复核仍有实质缺口：API-service硬编码processed-events并按单BlockchainEvent解码，而durable processor实际输出chainpulse.block.v1完整frame到committed topic；gateway的订阅路由仅升级本地无MQ输入的handler；numeric replay ring没有epoch/过期范围裁决。puller的durable_outbox确实发布chainpulse.reorg.v1封装，携带id/job_id/chain_id/generation/kind/payload，需要消费者正确接入；不能把“WebSocket能upgrade”当成emit→committed/reorg→gateway订阅的证明。后续必须实现并通过真实进程链路、去重/重组更正、重连缺口与过期游标负控。


T13本轮完整门禁已exit0（full-telemetry-closure-v1）：67顶层/149pass/0fail/0skip真实adapter/deployed；migration7顶层/12pass/0fail/0skip；14指标合同、18查询工具控制、31门禁反例均通过。历史unit跳过动作71仍如实披露。telemetry-runtime-v1只读采样实际processor/puller/API/gateway/monolithic五入口，逐个绑定lsof监听PID、ps命令行及二进制SHA256，全部含go_goroutines/go_memstats_alloc_bytes；sampler正常退出，不是30min资源验收。新增四个回归用例均列入unit/command必需清单，原生订阅完整链路另列release-candidate pending。该增量补齐指标可见性、摘要一致性与错误诊断，不宣称T13跨MQ trace/告警故障矩阵已全量闭环。


本轮首次业务提交被目录检查拒绝：micro-loop对单main包执行go build，在候选根目录生成36MiB的event-processor（已核对Mach-O/arm64、mtime与.gitignore规则，非tracked文件）。未绕过hook，已将该自有产物移入候选build/acceptance/20261014-working/telemetry-compile-artifact归档；正常hook随后通过，业务提交为c796290，真实提交后lint0issues。此前失败调用后的同名lint日志不能视作提交成功证明，v2日志才对应实际提交后检查。

scripts/dev-micro-loop.sh编译输出改为/dev/null，单main与./pkg/...多包实测均exit0且根目录无event-processor；fast --base HEAD~1也通过。独立二进制命名的full-telemetry-closure-v2正在真实重跑，不能提前认定通过；尚待该轮退出、迁移/反控结果与根目录无产物检查，再提交独立配置批次。

候选89af157远端11required checks与acceptance-gate已全部success，集成/EVM原始工件及真实PR head/merge SHA已归档。4stash逐一show统计与分类已补齐并保持原SHA，不apply/pop/drop；历史diff-audit文档前置说明其2026-10-14结论不代表当前完整封版。


full-telemetry-closure-v2现已exit0，编译改为/dev/null后的真实部署/adapter回归仍149pass/67顶层/0fail/0skip；迁移与31反控均通过，结束后明确test ! -e event-processor通过。v1原始证据/生成二进制归档均保留，v2使用独立二进制命名没有覆盖其哈希绑定。该结果闭环本次构建产物缺陷。

原始dirty源码中另已开始T11共享reorg wire增量，先RED再实现；subscription-wire-race100：1500pass actions/4顶层/0fail/0skip，覆盖生产puller编码器与共享解析、识别detected通知、不合法schema/id/job/chain/generation/kind/range/time/超大及尾随JSON。该增量尚未同步候选，更未接入API消费者，不能将此单元合同记为整个订阅链路已完成。


可观测性业务c796290与独立配置/工具批次edd7006现均已推送。候选工作树clean；实际make build按edd7006元信息构建全部二进制exit0，新CI正在运行，旧89af157全绿仅绑定旧候选。T11共享wire代码仍只在原始dirty工作树，尚未进入候选或实际API消费路径；不可把源码race100当作远端/端到端证明。


## 2026-10-17：订阅消费者状态机 RED→GREEN 与 edd7006 CI 收口

并行执行于10:49落下订阅消费者就绪状态机的RED测试（cmd/microservices/api-service/subscription_consumer_test.go）后停滞，实现缺失使make check与make test-short因vet: undefined: newSubscriptionConsumerState连续失败——这是"左侧任务卡住"的直接证据。当日19:53接管补完实现：subscriptionConsumerState按分区记录effect成败、单分区成功不得掩盖另一分区失败、revocation清空旧generation失败、ready观察ctx取消；新增validSubscriptionTopic在启动worker前拒绝非法topic（含长度249、字符集、"."/".."边界）；core.MessageQueueMessage新增Partition字段并由kafka owned consumer以broker权威值填充，MQ hook签名对照ConsumerOwnershipHooks实际定义（Revoked为单ctx）修正。同轮gofumpt修复并行在途的canonical_query_runtime.go格式。

验证：go test -race -count=3 ./cmd/microservices/api-service/ -run 'TestSubscription|TestAPIServiceUsesConfigured'全PASS；顺序DoD dod-20261017-195656-*五步exit code均0（build空输出、check 78B、lint 0 issues、test-short 66 ok/0 FAIL），api-service整包-race -count=1复核ok。全部原始日志在build/acceptance/20261014-working/。该状态机与Partition字段仍只在原始dirty工作树，与T11共享wire同状态，未同步候选、未接端到端链路，不记为T11完成。

候选edd7006的远端CI已实读收口：PR head与远端分支一致（edd70060d7cec23cd1c2f2360eb771d443640828），statusCheckRollup中16项全部SUCCESS（quality、coverage、unit-race、frontend-api、integration、security、image-build、lint、contracts、acceptance-gate、build-binaries、evm-e2e、Collect Performance Metrics、Check Performance Regression、Coverage comparison report、Evidence Pack D22），1项Notify on failure为SKIPPED，PR state为OPEN。上一节"新CI正在运行"至此闭环；仍不替代最终tag SHA验收与release/CD门禁。


## 2026-10-17 22:34：T11 订阅链路同步候选 909169e

左侧执行自10:49落RED测试后停滞约11小时（执行文档停在10:30），本轮直接接管推进T00分批落地的第一批：把订阅链路从原始dirty工作树同步进隔离候选工作树（~/.codex/worktrees/web3-go-delivery/chainpulse）。同步前逐一diff核实每个既存文件的差异仅为订阅相关（main.go仅消费接线替换、mq_plugin.go仅+Partition、kafka_owned_consumer.go仅Partition回填、canonical_query_runtime.go仅+SubscriptionStore/SubscriptionChains两方法），新文件依赖闭包全部核对为候选已有（dlq_events表、migration清单两侧42文件IDENTICAL、ChainScope/requireDB/SetReadinessCheck/Health/ReorgRollbackEvent/core topics均在候选中存在），共13个文件（4 modified + 9 new，untracked新文件经git status独立核对，未用tracked-only diff-stat隐瞒）。

候选内顺序门禁：go build exit0；make check（vet+gofumpt）exit0；make lint --new-from-rev=HEAD~1 0 issues（提交前后各一次）；make test-short exit0全绿；订阅三包-race -count=1全ok（api-service/indexing/mq）。业务提交909169e（13 files, +1176/-76）经pre-commit build+vet后推送，远端实读909169e0159f449680c8d60e00aa87aa63893fd1与本地一致。该批包含：API-service就绪状态机消费者替换processed-events旧路径、共享reorg wire解析、PG权威delivery（block/reorg/durability校验+quarantine）、Partition broker权威身份、就绪探针接入健康检查。远端CI对该SHA运行中，结论未出前不记通过；dirty工作树原状保留未提交部分不动。


## 2026-10-17 22:55：909169e contracts 失败与 2071ca1 修复推送

909169e的远端CI实读结论：CI=failure、Integration & E2E=cancelled（被后续push取消）、Security/Frontend API/Performance Metrics/E2E Tests/Code Coverage=success。CI失败根因在acceptance-gate的4条issue，其中硬失败为contracts：TestPostgresSubscriptionDeliveryUsesCommittedAuthority在64行断言max-uint256（78位精确值）回读失败——该测试（属第一批）依赖postgres_event_payload.go的UseNumber解码修复（属第二批），我将本应原子的单元拆成两批导致。诊断未猜测：拉取job 117421263994原始日志定位断言与expected json.Number值。

修复与本地复现：第二批（postgres_event_payload.go UseNumber+尾随JSON拒绝、puller durable_outbox改用共享appindexing.ReorgNotice编码器+新增durable_outbox_test、计划spec与执行文档增量）在候选内已过build/check/lint（0 issues）/test-short/两包-race门禁；另建独立库chainpulse_sub于cp-pg-fwd:15433（migrate up至head），CHAINPULSE_RUN_POSTGRES_INTEGRATION=1实跑该contracts用例PASS（0.36s），确认修复对症。提交2071ca1（5 files, +83/-10）经pre-commit build+vet、提交后make lint 0 issues后推送，远端实读2071ca1502bd487096359f2d095c8487b2a9c4ba一致，其CI运行中，结论未出不记通过。

提交后做了双向全树审计（不只git status）：候选tracked全量对主树cmp 0差异0缺失；主树tracked全量对候选0差异0缺失；主树全部untracked（含被全局gitignore隐藏的docs/project执行文档本身、migrations、workflows、frontend、bench脚本）逐一cmp均为内容一致或候选已tracked；例外仅pkg/plugins/database/zz_nul_repro_test.go（前缀zz的临时NUL复现，实测PASS，payload raw为base64无NUL，判定为草稿不入候选）。即除该草稿外，dirty工作树与候选已完全同步，T00分批落地的同步面清零，待2071ca1 CI结论。


## 2026-10-17 23:05：2071ca1 CI 七项全绿收口

2071ca1502bd487096359f2d095c8487b2a9c4ba 的远端CI实读结论（轮询至全部completed）：CI、Integration & E2E Tests（含contracts）、Code Coverage、E2E Tests、Security、Frontend API、Performance Metrics 共7个workflow全部success——909169e失败的contracts用例（TestPostgresSubscriptionDeliveryUsesCommittedAuthority的max-uint256精确回读）经UseNumber修复后闭环。至此T00分批同步面（909169e+2071ca1）远端全绿，dirty工作树与候选完全同步。下一步推进T11遗留：gateway订阅路由（现仅本地无MQ输入的handler升级）、numeric replay ring的epoch/gap恢复裁决，以及emit→committed/reorg→gateway的真实进程链路负控。


## 2026-10-18 06:44：T15/T16 E15 发布/回滚验收与 release-candidate lane 接线

按 spec `docs/specs/2026-10-17-e15-release-rollback.md`（Approved）落地 §9 的 E15，全部改动在隔离候选工作树内同批进行。

新增 `test/acceptance/deployed_rollback_test.go`（`//go:build integration`，单顶层用例 `TestDeployedReleaseRollback`，门控 `CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1`，否则 skip 文案与 allowlist 登记逐字一致）。prev 侧用 `git archive CHAINPULSE_DEPLOYED_ROLLBACK_PREV_REF`（RC workflow 默认 `7636755`）解到 TempDir 后编本机二进制，并断言该修订恰有16个 up 迁移（否则早失败）；HEAD 编当前二进制；两侧记 sha256（`291c28e9739a…` / `94607b97969c…`），保证16→21 expand 真实发生。三个隔离库：主库（prev 16 → expand 21）、探针 C 的16、探针 D 的 **21+dirty**（比原型16+dirty 更精确地把失败单独归因到迁移中断）。prev/cur 均为本机进程（SIGTERM + 30s 上限，进程提前退出即判失败并打印日志尾部），docker 仅用于坏镜像探针；digest 复用 `chainDigest`（events+blocks+checkpoint 折叠）且采样前先停写。

实测（候选工作树 + 本机 e15 栈 PG15.19@15440/Anvil31337@18576/Redis@6379/Mongo@27017/Kafka@9092，emitter `0xe7f1725E…0512`）：`go test -tags=integration -run '^TestDeployedReleaseRollback$'` → `--- PASS: TestDeployedReleaseRollback (37.36s)`，`-json` 复跑 `Elapsed":31.79` 且包 pass。过程行：prev 在 schema16 索引后 `schema=16 rows=13`；探针 config `exit=1 message="API_PORT must be 1-65535"`；停 prev 后 expand 并断言 `version=21 dirty=false`；探针 identity `exit=1 "RPC chain ID mismatch"`（刻意放在 schema21 之后，否则会先撞 schema20 缓存门而测错拒绝原因）；探针 stale-schema `exit=1 "requires clean schema20"`（独立16库）；探针 dirty-ledger `exit=1` 同消息（独立21+dirty库）；探针 corrupt-image `exit=126`（docker build 非可执行体 + docker run）；cur 起于 schema21 后 `schema=21 rows=14`；回滚 prev 起于同一库后 `/health` 200、`/api/v1/events` 200、digest 与 cur 逐字一致（`89cadde0b75d…|7ff6d65d37ad…`）且 rows=14；回滚后 emit#3 `rows=15` 记为**观察项**（不 gate，§9 只承诺"旧版恢复+摘要一致"）。每个探针另断言 ctx 未超时（被杀/超时不算 fail-closed）且输出不出现 `Status: Running`；docker 缺失时 `t.Fatal` 不 skip。artifact：`build/acceptance/e15-local/rollback-report.json`（五探针 `startup_rejected: true`、`digest_identical: true`）。

`scripts/acceptance/verify-tests.py` 新增可重复 `--require-scenario ID`（`scenario_ids` + `resolve_required_cases`）：未知ID→exit 2；显式选择解析为空即失败，不可静默放行；lane 判定/空场景/placeholder 规则一律未动。真实 manifest 演练（非仅夹具）：正向 `--require-scenario E15 --min-tests 1 --fail-on-unknown-skip --skip-allowlist …` → `✓ verification passed`（pass=1/fail=0/skip=0，报告 `"ok": true`）；`--require-scenario E99` → exit 2；`--required-lane release-candidate` → `required case E12/E14: no executable tests declared`（**实测印证决议前提**：该 lane 直接接线必红）；构造 E15 skip 的 jsonl → `was SKIPPED (required skips must be 0)`；`--self-test` 21 项 `[PASS]` + `self-test: OK`。`negative-cases.py` 新增4条（未知ID、被点名 scenario skip、lane 过滤后为空、正向对照）→ 矩阵 **35/35** as expected。

`required-cases.json` E15 填 `TestDeployedReleaseRollback` 并删 `pending`；`skip-allowlist.json` 登记该用例 `dependency_gated`/lane contracts/scenario E15/expires 2026-11-18 —— 该登记是必需的（contracts lane 跑 `./test/acceptance` 且 `--fail-on-unknown-skip`，不登记会把既有 lane 变红），skip 文案与 `t.Skip` 逐字一致。

CI/CD 接线：新增 `.github/workflows/release-candidate.yml`（`workflow_dispatch`，`prev_ref` 默认`7636755`；`push` tag `v*`），services PG15/Redis7/Mongo7/Kafka，Foundry v1.8.5 + Anvil31337@18576 + `forge create` emitter，跑 `go test -json -tags=integration -run '^TestDeployedReleaseRollback$'` 后以 `--required-cases … --require-scenario E15 --fail-on-unknown-skip` 强制该场景，artifact `release-candidate-evidence`（`if-no-files-found: error`）；另有 Anvil/Docker 守卫步骤，缺任一即红不 skip。`.github/workflows/ci.yml` quality 新增 dep-approval 步骤（先 `--self-test`，再取 base：PR=`pull_request.base.sha`、push=`event.before`、dispatch/全零→`git fetch --depth=1 origin master`；仓库 pack 221MiB 故定点抓取 base 而非拉全史）。`.github/workflows/coverage.yml` 把 `gocovmerge@latest` 钉到与 test.yml 相同的 `b5bfa59e…`（全仓唯一未固定工具）。`scripts/dev-micro-loop.sh` full 门控新增 `CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1` 与四个 rollback env 的 `:?` 检查（保持 full 模式零 skip 不变量，仿 DUAL_FAULT 先例）、step8 `-timeout` 10m→20m、新增 step11 单跑 E15 并以 `--require-scenario` 门禁。

候选内顺序门禁原始结论：`gofumpt -l` 对新增文件为空；`go vet -tags=integration ./test/acceptance` 无输出 exit0；`go build ./...` exit0；`make check` → `All checks passed.`；`make lint` → `0 issues.`（linter `2.12.2 built with go1.26.0`）；`make test-short` exit0（0 个 FAIL）；`scripts/dev-micro-loop.sh --mode fast` exit0 —— 该模式原来在"仅有 integration-only 包变更"时 `changed_pkgs` 为空，step3/step4 以无参数的 `go vet`/`staticcheck` 报 `no Go files in …` 而失败，本批新增 integration-only 文件把它暴露出来，已补守卫（如实跳过并注明依赖 full 门）；`actionlint` 对 release-candidate/ci/coverage 三份 workflow 全通过；`bash -n scripts/dev-micro-loop.sh` 通过；两个 JSON 策略文件 `json.load` 通过；`dep-approval-gate.py --self-test passed (9 cases + manifest naming)`、`--base origin/master --head HEAD` exit0（本批未改任何依赖清单）。`git diff --stat`：7 files changed, +242/−13，另有2个未跟踪新文件（`release-candidate.yml`、`deployed_rollback_test.go`）。

SBOM 证据（真实 run/artifact，非仅 workflow 文本）：security run `39075941287`（`headSha=ddf94f9…`，与本批候选 HEAD **同 SHA**，success）的 artifact `security-report`（121474B）下载实测含 `sbom/govulncheck-sbom.json`（`"schema":"chainpulse.sbom.v1"`、`produced_by: govulncheck v1.8.0`、`go_version: go1.26.9`）、`govulncheck-root.json`、`trivy-fs.json`、`security/summary.json`；`release.yml` 存在五次 `anchore/sbom-action` + `upload-artifact … name: sbom-images, path: sbom/`（含 `sbom/**/*.spdx.json`）接线，最近一次 success 的 release run `26755463489`（2026-06-01，push）其工件已过保留期不可下载，如实记为"接线存在、工件已过期"，不记为可引用工件。

仍未完成、本节不记为通过（属 T00/T18 封版项）：推送后手动触发 release-candidate 远端运行并保存 run URL；本批新 SHA 的同 SHA 11 项 required checks 与 acceptance-gate 远端结论；`v*` tag 触发的 release SBOM 工件引用；T15 CD 半边 `deploy-rehearsal.sh` 的两 digest 回滚扩展（另项推进）。
