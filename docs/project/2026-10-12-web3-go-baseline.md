# ChainPulse Web3 + Go 后端评审：2026-10-12 基线

本报告记录本轮实际读取与命令结果；与 [v7 交付计划](../specs/2026-10-12-web3-go-backend-delivery-plan.md) 配套。日期采用 Asia/Singapore。它证明当前本地工作区的质量基线，不证明优化任务已实施、生产依赖已通过或远端 CI/CD 已通过。

## 1. 检查范围与结论

- 本地 HEAD：`d364744e68ad41a81fe0be5c9809af3b9feced4f`；工作区 dirty，开始前 `git status --porcelain=v1` 有 94 条记录。
- 根模块：Go 1.26.0；golangci-lint 2.12.2 built with go1.26.0。所有本轮 Go 命令先设置 `GOPROXY=https://goproxy.cn,direct`。
- 按仓库顺序执行 build → check → lint → test-short → diff-stat，各命令 exit 0；额外跑 root `--tests=false` 全量 lint，输出 `0 issues.`。
- 从 `go work edit -json` 枚举六个子模块，逐个 build/vet/test-short，18 个命令均 exit 0。根目录 `./...` 的结果与这些结果分开记录。
- 对根 `pkg`/`cmd` 补跑 JSON 短测试：6674 个 pass 测试动作、69 个 skip 测试动作、0 个 fail 测试动作；66 个 package pass，8 个 package skip（无测试包）。pass 包含父子 subtest，不是独立顶层测试数。
- `go test -tags=integration ./test/contracts/... -list .` 没有列出任何 Test；现有契约 workflow 可能空跑。
- 读取代码发现 chain 作用域、schema、事务/checkpoint、finality、SIWE nonce 和 CI/CD 退出码问题，尚未进行对应故障复现或修复。
- 本轮未执行真实 Postgres/Kafka/Redis 全链路、Anvil E2E、frontend build、性能、部署或回滚；这些都是 v7 的明确待验收项。

## 2. 具体代码证据

| 发现 | 位置 | 证据与限制 |
| --- | --- | --- |
| CI 捕获退出码但不用于最终判断 | `.github/workflows/test.yml:113–164,176–205,246–275` | 保存 PIPESTATUS 后按日志 grep 判失败，timeout 分支可继续、末尾 exit 0。部分异常可能被 shell 默认行为拦截，但显式逻辑仍不可信，需阳性对照检验 |
| 宽松部署验证 | `.github/workflows/deploy.yml:72–108,209–256,353–381` | 验证/测试/lint 有忽略失败；生产部署有 continue-on-error，不能证明部署成功 |
| 空契约 suite | `test/contracts/db_contract_test.go:17`、`cache_contract_test.go:17` | 只有辅助函数，无 `Test...` runner；实际 list 输出附后 |
| 链隔离缺口 | `pkg/plugins/database/postgres_database.go:795–880` | GetBlock 按 number、GetLatestBlock 全表MAX，Delete/Mark reorg 按范围不含chain；共享库的跨链影响要用真实数据验证 |
| 表模型分裂 | `migrations/000001_init_schema.up.sql`、`000004_event_natural_key_constraint.up.sql`、`pkg/plugins/database/postgres_database.go:158` | 初始表id/block_hash与插入event_hash不一致；自然键迁移指向events_metadata；未在本轮跑fresh DB，不能声称已经运行报错 |
| 幂等修复未接生产入口 | `pkg/services/processor/idempotency.go:343`、`event_processor.go:281,321` | CheckAndMark只有测试调用；processor仍分步查重/写/标记。直接先mark后write也不正确，计划采用durable事务效果 |
| 非原子checkpoint | `pkg/application/indexing/runtime.go:463–503` | Persist、MarkProcessed、Save分步；全重复批次直接返回；缺口/空块/恢复语义需真实测试 |
| checkpoint DB方法签名风险 | `pkg/infrastructure/database/checkpoint_store.go:30–36,62–68` | 使用返回匿名接口或any的结构性类型断言；需与manager返回的实际driver方法签名核对，mock不能替代真实契约 |
| 事件身份不足 | `pkg/core/event_hash.go:21–40` | chain_id/block_number/tx_hash/log_index，未含block_hash；分支历史与重新入块身份需明确 |
| 重组发布未可靠闭环 | `pkg/services/reorg/reorg_handler.go:285–287` | rollback event发布失败只记日志后继续；是否有其他可靠修复入口需沿wiring动态确认 |
| finality误用风险 | `pkg/services/finality/finality_checker.go:95–155,164–184` | L2 safe近似、latest fallback；bool封装返回IsFinalized，不能保证降级调用方不误标 |
| SIWE nonce与生命周期 | `pkg/plugins/api/siwe_handler.go:141–159,186–233` | 地址未比对、Get/Delete分离；cleanupLoop没有取消。origin修复已在当前handler，不重复当未实现任务 |
| 性能门禁与发压无效 | `.github/workflows/metrics.yml:334–359`、`scripts/bench/ingest.sh` | 回归TODO；发压可忽略失败或skip。旧计划§11.3也明确摄取吞吐未验收 |
| 文档冲突 | `docs/project/ARCHITECTURE_RULES.md`、`.codex/skills/code-organization-placement/SKILL.md` | 旧页禁止spec，示例含已不存在adapters；本轮以用户AGENTS与代码为准，纳入T00 |

静态发现不是“所有实际部署都会触发”的结论。它确定了需要补齐的契约与故障测试；T02/T05/T06须定位真实调用路径并做动态证伪。

## 3. 原始质量门禁输出

以下顺序与实际执行一致；各块标注命令及exit。空输出表示该命令没有stdout/stderr，不代表没执行。

### 3.1 工具链

```text
go version go1.26.0 darwin/arm64
golangci-lint has version 2.12.2 built with go1.26.0 from (unknown, modified: ?, mod sum: "h1:7+d1uY0bq1MU2UV3R5pW5Q7QWdcoq4naMRXM+gsJKrs=") on (unknown)
```

### 3.2 go build ./...

exit_code=0；原始stdout/stderr为空。

### 3.3 make check

exit_code=0。

```text
Running go vet...
go vet ./...
Checking code formatting...
All checks passed.
```

### 3.4 make lint

exit_code=0；这是标准diff scope，不单独证明全仓库干净。

```text
Running linter...
0 issues.
```

补充命令：`"$(go env GOBIN)/golangci-lint" run --tests=false --timeout=10m ./pkg/... ./cmd/...`，exit_code=0。

```text
0 issues.
```

### 3.5 make test-short

exit_code=0。

```text
Running short tests (no integration/e2e)...
go test -short -count=1 ./pkg/... ./cmd/...
?   	github.com/rtcdance/chainpulse/pkg/application	[no test files]
ok  	github.com/rtcdance/chainpulse/pkg/application/bootstrap	2.958s
ok  	github.com/rtcdance/chainpulse/pkg/application/indexing	3.282s
ok  	github.com/rtcdance/chainpulse/pkg/codegen	2.447s
ok  	github.com/rtcdance/chainpulse/pkg/consensus	3.066s
ok  	github.com/rtcdance/chainpulse/pkg/core	13.186s
ok  	github.com/rtcdance/chainpulse/pkg/core/batch	2.674s
ok  	github.com/rtcdance/chainpulse/pkg/core/bloom	2.549s
ok  	github.com/rtcdance/chainpulse/pkg/core/correlation	2.797s
ok  	github.com/rtcdance/chainpulse/pkg/core/crypto	2.863s
ok  	github.com/rtcdance/chainpulse/pkg/core/dedup	2.320s
ok  	github.com/rtcdance/chainpulse/pkg/core/eventsig	0.926s
ok  	github.com/rtcdance/chainpulse/pkg/core/reorgprofile	0.807s
ok  	github.com/rtcdance/chainpulse/pkg/core/replay	0.621s
?   	github.com/rtcdance/chainpulse/pkg/core/topics	[no test files]
ok  	github.com/rtcdance/chainpulse/pkg/defi	0.675s
ok  	github.com/rtcdance/chainpulse/pkg/domain	0.588s
ok  	github.com/rtcdance/chainpulse/pkg/domain/query	0.605s
ok  	github.com/rtcdance/chainpulse/pkg/env	0.880s
ok  	github.com/rtcdance/chainpulse/pkg/evm	0.805s
ok  	github.com/rtcdance/chainpulse/pkg/gas	0.603s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/blockchain	0.539s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/config	1.865s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/data	0.546s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/database	10.811s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/deployment	0.845s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/discovery	0.794s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/gateway	12.162s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/health	0.497s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/http	1.235s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/reliability	30.621s
ok  	github.com/rtcdance/chainpulse/pkg/infrastructure/rpc	5.119s
ok  	github.com/rtcdance/chainpulse/pkg/integrations/erc20	0.591s
ok  	github.com/rtcdance/chainpulse/pkg/integrations/generic	0.482s
ok  	github.com/rtcdance/chainpulse/pkg/integrations/uniswap	0.465s
ok  	github.com/rtcdance/chainpulse/pkg/mempool	0.428s
ok  	github.com/rtcdance/chainpulse/pkg/observability	0.543s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api	2.114s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/apierrors	0.449s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/business	0.479s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/core	0.751s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/graphql	0.524s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/http	0.640s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/rollout	0.430s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/shared	1.241s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/api/websocket	0.459s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/cache	6.724s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/database	0.474s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/mq	0.963s
ok  	github.com/rtcdance/chainpulse/pkg/plugins/pullers	0.562s
?   	github.com/rtcdance/chainpulse/pkg/ports	[no test files]
ok  	github.com/rtcdance/chainpulse/pkg/services/decoder	0.439s
ok  	github.com/rtcdance/chainpulse/pkg/services/finality	5.453s
ok  	github.com/rtcdance/chainpulse/pkg/services/indexing	0.477s
ok  	github.com/rtcdance/chainpulse/pkg/services/processor	3.826s
ok  	github.com/rtcdance/chainpulse/pkg/services/query	0.999s
ok  	github.com/rtcdance/chainpulse/pkg/services/query/circuitbreaker	0.498s
ok  	github.com/rtcdance/chainpulse/pkg/services/query/qerrors	0.754s
ok  	github.com/rtcdance/chainpulse/pkg/services/query/qindex	0.437s
ok  	github.com/rtcdance/chainpulse/pkg/services/reorg	0.454s
ok  	github.com/rtcdance/chainpulse/pkg/siwe	0.422s
ok  	github.com/rtcdance/chainpulse/pkg/testhelpers	0.744s
?   	github.com/rtcdance/chainpulse/cmd/chainpulse	[no test files]
ok  	github.com/rtcdance/chainpulse/cmd/exercises/01_decode_event	0.438s
ok  	github.com/rtcdance/chainpulse/cmd/microservices/api-gateway	0.598s
ok  	github.com/rtcdance/chainpulse/cmd/microservices/api-service	0.864s
ok  	github.com/rtcdance/chainpulse/cmd/microservices/event-processor	1.126s
ok  	github.com/rtcdance/chainpulse/cmd/microservices/puller	0.641s
?   	github.com/rtcdance/chainpulse/cmd/migrate	[no test files]
ok  	github.com/rtcdance/chainpulse/cmd/monolithic/chainpulse	0.736s
ok  	github.com/rtcdance/chainpulse/cmd/playground	0.433s
?   	github.com/rtcdance/chainpulse/cmd/tools/chainprobe	[no test files]
?   	github.com/rtcdance/chainpulse/cmd/tools/chainpulse	[no test files]
?   	github.com/rtcdance/chainpulse/cmd/tools/deployed-real-event-acceptance	[no test files]
```

### 3.6 git diff --stat

exit_code=0；以下是整个工作区的既存修改，**不是本轮创建的代码差异**。新文档未暂存，不会自动出现在该命令；ignored MASTER_PLAN也不会出现。任务实际范围在§6单独列明。

```text
 .github/copilot-instructions.md                    | 125 +------------
 .github/workflows/ci.yml                           |   2 +-
 .github/workflows/test.yml                         |   2 +-
 .gitignore                                         |   4 +
 .golangci.yml                                      |  33 +++-
 CLAUDE.md                                          |  42 +----
 Makefile                                           |  10 +-
 README.md                                          |  22 ++-
 cmd/microservices/api-gateway/QUICKSTART.md        |   2 +-
 .../api-gateway/rollout_report_producer_test.go    |   3 -
 .../api-gateway/rollout_runtime_support_test.go    |   1 -
 cmd/microservices/api-service/QUICKSTART.md        |   2 +-
 .../api-service/rollout_report_integration_test.go |   1 -
 .../api-service/rollout_report_producer_test.go    |   2 -
 cmd/microservices/event-processor/QUICKSTART.md    |   2 +-
 .../event-processor/processor_runtime.go           |   1 +
 cmd/microservices/puller/QUICKSTART.md             |   2 +-
 .../chainpulse/m1a_runtime_wiring_test.go          |  16 ++
 cmd/monolithic/chainpulse/main.go                  |   9 +-
 .../ownership_rollout_report_body_test.go          |   3 -
 cmd/playground/main.go                             |   6 +-
 cmd/playground/static.go                           |   4 +-
 docker/Dockerfile                                  |   2 +-
 docker/Dockerfile.microservices                    |   2 +-
 docker/README.md                                   |   4 +-
 docs/e2e-testing/README.md                         |   2 +-
 docs/e2e-testing/faq.md                            |   2 +-
 docs/e2e-testing/testing-guide.md                  |   4 +-
 docs/guides/DEPLOYMENT_GUIDE.md                    |   2 +-
 docs/guides/DEVELOPER_GUIDE.md                     |   6 +-
 frontend/package.json                              |   3 +-
 go.mod                                             |  49 ++---
 go.sum                                             | 200 ++++++++++++++++-----
 go.work                                            |   2 +-
 package.json                                       |  27 ---
 pkg/blockchain/go.mod                              |   2 +-
 pkg/chainid/go.mod                                 |   2 +-
 pkg/configmodel/go.mod                             |   2 +-
 pkg/core/config_extensions.go                      |   1 +
 pkg/core/config_validation.go                      |   1 +
 pkg/core/event_filter.go                           |   1 -
 pkg/core/metrics.go                                |   2 +
 pkg/evm/chained_decoder.go                         |   3 +-
 pkg/evm/chained_decoder_test.go                    |  25 +++
 pkg/evm/typed_events.go                            |  23 ++-
 pkg/evm/typed_events_test.go                       |  56 ++++++
 pkg/histogram/go.mod                               |   2 +-
 .../config/advanced_config_manager_test.go         |   5 +-
 pkg/infrastructure/config/redis_config_test.go     |   9 +-
 pkg/infrastructure/database/manager_test.go        |  12 +-
 .../deployment/deployment_config_property_test.go  |  13 +-
 .../deployment/deployment_config_test.go           |   8 -
 .../deployment/deployment_mode_test.go             |  12 --
 pkg/logkeys/go.mod                                 |   2 +-
 pkg/plugins/api/dlq_handler.go                     |  17 +-
 pkg/plugins/api/event_query_handler.go             |   5 +-
 pkg/plugins/api/gateway_test.go                    |   2 +-
 pkg/plugins/api/http/plugin.go                     |   2 -
 pkg/plugins/api/pprof_handler.go                   |   1 +
 pkg/plugins/api/siwe_handler.go                    |   8 +-
 pkg/plugins/api/siwe_handler_test.go               |  56 ++++++
 pkg/plugins/cache/cache_plugin.go                  |   3 +
 pkg/plugins/database/database_plugin_base_test.go  |  24 ---
 pkg/plugins/database/postgres_database.go          |  38 ----
 pkg/plugins/database/postgres_database_health.go   |   2 -
 pkg/plugins/mq/kafka_mq.go                         |   6 +-
 pkg/plugins/mq/kafka_state.go                      |   1 +
 pkg/plugins/pullers/https_jsonrpc_puller.go        |   8 -
 pkg/plugins/pullers/jsonrpc_helpers.go             |   4 +
 pkg/plugins/pullers/puller_helpers_test.go         |   3 -
 pkg/plugins/pullers/solana_puller.go               |  25 ---
 pkg/runtimetypes/go.mod                            |   2 +-
 pkg/services/processor/event_processor_test.go     |   3 -
 pkg/services/processor/idempotency.go              |  78 ++++++--
 pkg/services/query/error_classifier_test.go        |  10 --
 pkg/services/query/qerrors/classifier.go           |  19 +-
 pkg/services/query/qindex/index.go                 |   2 -
 pkg/services/query/qindex/optimizer.go             |   2 -
 pkg/services/query/qindex/stats.go                 |   2 -
 pkg/services/query/runtime_summary_test.go         |   9 +-
 pkg/services/reorg/reorg_handler.go                |  13 +-
 pkg/siwe/siwe.go                                   |  33 ++++
 pkg/siwe/siwe_test.go                              |  81 +++++++++
 test/integration/container_helper.go               |   1 -
 .../event_store_error_scenarios_test.go            |   2 -
 .../multi_chain_indexing_property_test.go          |   7 -
 test/integration/multi_chain_indexing_test.go      |  11 --
 test/integration/query_examples_test.go            |  10 --
 88 files changed, 719 insertions(+), 551 deletions(-)
```

## 4. 补充执行与 skip 原始清单

JSON补跑命令：`go test -json -short -count=1 ./pkg/... ./cmd/...`，exit_code=0。仅为解决短测试输出不展示skip数量的问题而补跑。

解析结果：

```text
test_actions={"pass": 6674, "skip": 69}
package_actions={"pass": 66, "skip": 8}
```

69个skip测试动作的package/Test原始列表如下；T00实施时需要进一步读取reason分类，不能仅凭名字认定全部合理。

```text
github.com/rtcdance/chainpulse/pkg/infrastructure/blockchain TestConcurrentProcessing
github.com/rtcdance/chainpulse/pkg/infrastructure/config TestWatchConfigDecryptFailure
github.com/rtcdance/chainpulse/pkg/infrastructure/config TestNotifyWatchersMultiple
github.com/rtcdance/chainpulse/pkg/infrastructure/config TestWatchConfigWithSensitiveKey
github.com/rtcdance/chainpulse/pkg/infrastructure/config TestWatchConfig
github.com/rtcdance/chainpulse/pkg/infrastructure/config TestNotifyWatchersPanicRecovery
github.com/rtcdance/chainpulse/pkg/infrastructure/database TestDatabaseManagerInitializeEmptyMongoURI
github.com/rtcdance/chainpulse/pkg/infrastructure/database TestDatabaseManagerInitializeEmptyPostgresURL
github.com/rtcdance/chainpulse/pkg/infrastructure/database TestDatabaseManagerInitializeAlreadyInitialized
github.com/rtcdance/chainpulse/pkg/infrastructure/discovery TestSessionConcurrentUpdate
github.com/rtcdance/chainpulse/pkg/infrastructure/deployment TestConfigurationValidation
github.com/rtcdance/chainpulse/pkg/plugins/api TestMultipleWarmingCycles
github.com/rtcdance/chainpulse/pkg/plugins/api TestLastWarmingTime
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingDataWithDifferentStatusCodes
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingFailureMetrics
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingMetricsRecording
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingDataWithDifferentTTLs
github.com/rtcdance/chainpulse/pkg/plugins/api TestContextCancellation
github.com/rtcdance/chainpulse/pkg/plugins/api TestConcurrentWarmingOperations
github.com/rtcdance/chainpulse/pkg/plugins/api TestIsRunning
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingData
github.com/rtcdance/chainpulse/pkg/plugins/api TestStopSuccess
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingWithError
github.com/rtcdance/chainpulse/pkg/plugins/api TestStartAlreadyRunning
github.com/rtcdance/chainpulse/pkg/plugins/api TestCacheWarmerGetStats
github.com/rtcdance/chainpulse/pkg/plugins/api TestStartSuccess
github.com/rtcdance/chainpulse/pkg/plugins/api TestWarmingWithBatchSize
github.com/rtcdance/chainpulse/pkg/plugins/api TestHandlerForbidden
github.com/rtcdance/chainpulse/pkg/plugins/api TestGatewayRouterIntegrationConcurrentRequests
github.com/rtcdance/chainpulse/pkg/plugins/api TestTokenBucketConcurrency
github.com/rtcdance/chainpulse/pkg/plugins/api/graphql TestGraphQLPluginLifecycleManagement
github.com/rtcdance/chainpulse/pkg/plugins/api/http TestHTTPAndHTTPSConcurrent
github.com/rtcdance/chainpulse/pkg/plugins/api/http TestHTTPSPortConfiguration
github.com/rtcdance/chainpulse/pkg/plugins/api/http TestHTTPSConnection
github.com/rtcdance/chainpulse/pkg/plugins/api/http TestHTTPPluginWithoutTLS
github.com/rtcdance/chainpulse/pkg/plugins/api/http TestTLSMetricsCollection
github.com/rtcdance/chainpulse/pkg/plugins/api/shared TestRequestBatcherContextCancellation
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWebSocketPluginIsRunning
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWSAndWSSConcurrent
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWebSocketPluginStop
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWSSMetricsCollection
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWSSPortConfiguration
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWebSocketPluginStart
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWebSocketPluginStartAlreadyRunning
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWebSocketPluginConcurrentOperations
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWebSocketPluginWithoutTLS
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWSSClientCount
github.com/rtcdance/chainpulse/pkg/plugins/api/websocket TestWSSConnection
github.com/rtcdance/chainpulse/pkg/plugins/database TestBatchInsertPerformance
github.com/rtcdance/chainpulse/pkg/plugins/database TestBatchInsertVariousSizes
github.com/rtcdance/chainpulse/pkg/plugins/database TestSingleEventPerformance
github.com/rtcdance/chainpulse/pkg/plugins/database TestQueryPerformance
github.com/rtcdance/chainpulse/pkg/plugins/database TestDatabaseIntegrationSuite
github.com/rtcdance/chainpulse/pkg/plugins/database TestDatabaseIntegrationWithErrors
github.com/rtcdance/chainpulse/pkg/plugins/database TestDatabaseIntegrationConcurrency
github.com/rtcdance/chainpulse/pkg/plugins/database TestPostgreSQLRealConnection
github.com/rtcdance/chainpulse/pkg/plugins/database TestTransactionIsolation
github.com/rtcdance/chainpulse/pkg/plugins/database TestRollbackOnError
github.com/rtcdance/chainpulse/pkg/plugins/database TestConcurrentTransactions
github.com/rtcdance/chainpulse/pkg/plugins/database TestTransactionConsistency
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationPublishAndConsume
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationPerformance
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationHealthCheck
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationDeadLetterQueue
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationErrorHandling
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationMultipleConsumers
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationBatchPublish
github.com/rtcdance/chainpulse/pkg/plugins/mq TestKafkaIntegrationOffsetTracking
github.com/rtcdance/chainpulse/pkg/plugins/cache TestRedisCacheDelete
```

子模块结果（从go.work动态枚举，未硬编码执行清单）：

| module | build | vet | short test |
| --- | --- | --- | --- |
| pkg/blockchain | exit 0 | exit 0 | exit 0 |
| pkg/chainid | exit 0 | exit 0 | exit 0 |
| pkg/configmodel | exit 0 | exit 0 | exit 0 |
| pkg/histogram | exit 0 | exit 0 | exit 0 |
| pkg/logkeys | exit 0 | exit 0 | exit 0 |
| pkg/runtimetypes | exit 0 | exit 0 | exit 0 |

子模块没有补跑JSON skip报告，也没有在本轮跑完整race/真实服务测试；不能据表声称skip=0或完整workspace验收。

契约测试列举命令：`go test -tags=integration ./test/contracts/... -list .`，exit_code=0。

```text
ok  	github.com/rtcdance/chainpulse/test/contracts	0.470s
```

此块没有Test名称，是空suite证据。

## 5. GitHub状态：有旧结果，没有本轮结果

实际执行 `gh run list --limit 8 --json workflowName,headSha,status,conclusion,url,createdAt`，exit_code=0。最近返回的任务日期为2026-08-10，SHA为`fa80b8346343a6d76f23c4b7bc3dd71c50da0627`，与当前HEAD不同。

- [Performance Metrics：旧SHA的success](https://github.com/rtcdance/chainpulse/actions/runs/31353700077)，createdAt=`2026-08-10T03:50:06Z`。
- [Integration & E2E Tests：旧SHA的success](https://github.com/rtcdance/chainpulse/actions/runs/31353324184)，createdAt=`2026-08-10T03:42:06Z`。

只读取了任务摘要，没有下载并复核旧任务全部日志；不能由success反推用例真执行。本轮未push、触发workflow或部署，所以当前优化实施的远端CI/CD状态为**未验证**。

## 6. 本轮产物与残余工作

本轮只新增本基线报告与v7 spec，并在gitignored的`optimization-plan/MASTER_PLAN.md`头部建立v7入口及历史范围说明。应用代码、工作流、依赖、AGENTS及其他进行中的文件没有由本轮修改。

后续T00–T18全部仍待实施/验收。发现本机 `/Users/mingo/.gitignore_global:138` 的 `docs` 规则忽略了整个文档目录。两个新文档与MASTER_PLAN均是ignored，任务前后普通status没有新增记录；这不能解释为没有生成文件。本轮未修改全局配置、暂存或提交。

后续需要提交时，显式只纳入这三个产物：

```bash
git add -f docs/specs/2026-10-12-web3-go-backend-delivery-plan.md
git add -f docs/project/2026-10-12-web3-go-baseline.md
git add -f optimization-plan/MASTER_PLAN.md
```

不能用 `git add .` 代替以上范围核对，也不能把整个dirty工作区当成本轮变更。仓库最终diff验收应比较任务前快照，并分别报告已有修改、ignored/untracked产物与任务新增差异。

报告中的原始结果属于dirty工作区。最终release必须从已纳入全部必要变更的同一SHA重新新检出验证，并附CI/CD、真实依赖、性能和回滚证据。

