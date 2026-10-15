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
