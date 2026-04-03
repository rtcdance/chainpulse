# ChainPulse 实现状态地图

> 生成时间: 2026-04-03
> 基准架构: docs/archive/ARCHITECTURE_v1.md
> 用途: 让 AI 快速了解什么已实现、什么缺失，避免重复工作或跑偏

---

## 总览

| 层级 | 代码量 | 测试数 | 状态 |
|---|---|---|---|
| `pkg/core` | 4,892 LOC | 21 | ✅ 完成 |
| `pkg/services` | 11,412 LOC | 34 | ✅ 完成 |
| `pkg/plugins` | 28,180 LOC | 85 | ✅ 完成 |
| `pkg/infrastructure` | 12,481 LOC | 48 | ✅ 完成 |
| `pkg/observability` | 1,168 LOC | 3 | ⚠️ 基础完成 |
| `pkg/integrations` | 1,227 LOC | 3 | ✅ 完成 |
| `cmd/monolithic` | 2,618 LOC | 4 | ✅ 完成 |
| `cmd/microservices/*` | 7,260 LOC | 21 | ✅ 完成 |
| `test/integration` | 3,419 LOC | 13 | ✅ 完成 |
| `test/e2e` | 6,447 LOC | 38 | ✅ 完成 |
| `test/contracts` | 3 个契约测试文件 | — | ✅ 完成 |

**总计**: ~79K LOC 源码 + 277 个测试文件

---

## 按 ARCHITECTURE_v1.md 组件逐项对照

### 1. Shared Core (`pkg/core`)

| 组件 | 文件 | 行数 | 状态 | 备注 |
|---|---|---|---|---|
| Plugin 接口 | `plugin.go` | — | ✅ | Name/Version/Initialize/Start/Stop/Health |
| DataPullerPlugin | `plugin.go` | — | ✅ | PullEvents/GetLatestBlock/SubscribeToEvents/GetStats |
| DatabasePlugin | `plugin.go` | — | ✅ | StoreEvent/BatchStoreEvents/QueryEvents |
| CachePlugin | `plugin.go` | — | ✅ | Get/Set/Delete/Clear |
| MQPlugin | `mq_plugin.go` | 845 | ✅ | Publish/Subscribe/GetQueueDepth |
| EventBus | `eventbus.go` | — | ✅ | in-process chan 实现 |
| Logger | `logger.go` | — | ✅ | 结构化日志接口 + 默认实现 |
| Metrics | `metrics.go` | — | ✅ | Prometheus 指标接口 |
| Config | `config.go` | 710 | ✅ | 多链配置，含 BlockchainConfig |
| Models | `blockchain_models.go` | — | ✅ | Block/Transaction/Event/Log |
| Errors | `errors.go` | — | ✅ | 自定义错误类型 |
| Registry | `registry.go` | — | ✅ | 组件注册表 |
| Health | `health.go` | — | ✅ | 健康检查接口 |

### 2. Application Services (`pkg/services`)

| 组件 | 文件 | 行数 | 状态 | 备注 |
|---|---|---|---|---|
| MultiChainIndexer | `indexing/multi_chain_indexer.go` | — | ✅ | 多链索引协调器 |
| ChainIndexer | `indexing/chain_indexer.go` | 271 | ✅ | 单链索引器 |
| QueryService | `query/query_service.go` | 499 | ✅ | 查询服务 |
| CircuitBreaker | `query/circuit_breaker.go` | — | ✅ | 熔断器 |
| CacheService | `query/cache_service.go` | — | ✅ | 缓存服务 |
| ConsistencyChecker | `query/consistency_checker.go` | — | ✅ | 一致性检查 |
| EventDecoder | `decoder/event_decoder.go` | — | ✅ | ABI 解码 |
| ContractManager | `decoder/contract_manager.go` | — | ✅ | 多版本 ABI 管理 |
| ReorgHandler | `reorg/reorg_handler.go` | — | ✅ | 链重组处理 |
| RetryLogic | `resilience/retry_logic.go` | — | ✅ | 指数退避重试 |
| GracefulShutdown | `resilience/graceful_shutdown.go` | — | ✅ | 优雅关闭 |
| FailureRecovery | `resilience/failure_recovery.go` | 534 | ✅ | 故障恢复 |
| ErrorHandler | `resilience/error_handler.go` | — | ✅ | 错误处理 |
| EventProcessor | `processor/event_processor.go` | — | ✅ | 事件处理 |
| Idempotency | `processor/idempotency.go` | — | ✅ | 幂等写入 |

### 3. Monolithic Adapters (`pkg/plugins`)

| 组件 | 文件 | 行数 | 状态 | 备注 |
|---|---|---|---|---|
| MemoryMQ | `mq/memory_mq.go` | — | ✅ | in-process chan MQ |
| KafkaMQ | `mq/kafka_mq.go` | 1,131 | ✅ | Kafka 实现 |
| RedisMQ | `mq/redis_mq.go` | 527 | ✅ | Redis 实现 |
| ZeroMQMQ | `mq/zeromq_mq.go` | 521 | ✅ | ZeroMQ 实现 |
| InMemoryCache | `cache/` | — | ✅ | 内存缓存 |
| RedisCache | `cache/redis_cache.go` | ~400 | ✅ | Redis 缓存 |
| PostgresDatabase | `database/postgres_database.go` | 879 | ✅ | PostgreSQL 实现 |
| MockPuller | `pullers/` | — | ✅ | Mock 数据拉取 |
| GraphQL API | `api/graphql/` | — | ✅ | GraphQL 查询 |
| gRPC API | `api/grpc/` | — | ✅ | gRPC 服务 |
| HTTP API | `api/http/` | — | ✅ | HTTP REST |
| WebSocket API | `api/websocket/` | — | ✅ | WebSocket 订阅 |
| RateLimiter | `api/rate_limiter.go` | 502 | ✅ | 令牌桶限流 |

### 4. Microservice Adapters (`pkg/infrastructure`)

| 组件 | 文件 | 行数 | 状态 | 备注 |
|---|---|---|---|---|
| DataPuller | `data/data_puller.go` | — | ✅ | 生产 Puller |
| BlockHeightTracker | `data/block_height_tracker.go` | — | ✅ | 拉取进度持久化 |
| APIGateway | `gateway/api_gateway.go` | — | ✅ | 生产 Gateway |
| MultiProtocolAPI | `gateway/multi_protocol_api.go` | — | ✅ | 多协议适配 |
| WebSocketSubscription | `gateway/websocket_subscription.go` | — | ✅ | WS 连接池 |
| KafkaConfig | `config/kafka_config.go` | — | ✅ | Kafka 配置 |
| RedisConfig | `config/redis_config.go` | — | ✅ | Redis 配置 |
| ConsulConfig | `deployment/consul_config.go` | 55 | ✅ | Consul 服务发现 |
| EventProcessor | `processing/event_processor.go` | — | ✅ | 事件处理 |
| IdempotencyService | `processing/idempotency_service.go` | — | ✅ | 幂等服务 |
| RetryLogic | `processing/retry_logic.go` | — | ✅ | 重试逻辑 |
| FailureDetection | `reliability/failure_detection.go` | — | ✅ | 故障检测 |
| HorizontalScaling | `reliability/horizontal_scaling.go` | — | ✅ | 水平扩缩容 |
| GracefulShutdown | `reliability/graceful_shutdown.go` | — | ✅ | 优雅关闭 |
| StatelessService | `reliability/stateless_service.go` | — | ⚠️ | 含 placeholder |
| BlockchainCluster | `blockchain/blockchain_cluster.go` | 376 | ✅ | 节点池+故障切换 |

### 5. Entry Points (`cmd/`)

| 组件 | 文件 | 行数 | 状态 | 备注 |
|---|---|---|---|---|
| Monolithic | `cmd/monolithic/chainpulse/main.go` | 516 | ✅ | 单进程入口 |
| Puller | `cmd/microservices/puller/main.go` | 484 | ✅ | Puller 微服务 |
| EventProcessor | `cmd/microservices/event-processor/main.go` | 465 | ✅ | Indexer 微服务 |
| APIService | `cmd/microservices/api-service/main.go` | 402 | ✅ | Query 微服务 |
| APIGateway | `cmd/microservices/api-gateway/main.go` | 323 | ✅ | Gateway 微服务 |

### 6. Observability (`pkg/observability`)

| 组件 | 状态 | 备注 |
|---|---|---|
| Prometheus Metrics | ✅ | 指标定义完整 |
| Zap Logger | ✅ | 结构化日志 |
| OTel Tracer | ⚠️ | 基础实现，需完善 |
| Health Check | ✅ | 各服务都有 /health |

### 7. Integrations (`pkg/integrations`)

| 组件 | 状态 | 备注 |
|---|---|---|
| ERC20 | ✅ | ERC20 事件解码 |
| Uniswap | ✅ | Uniswap 事件解码 |
| Generic | ✅ | 通用事件解码 |

### 8. Testing

| 层级 | 状态 | 备注 |
|---|---|---|
| 单元测试 | ✅ 203 个测试文件 | 覆盖所有核心包 |
| 集成测试 | ✅ 13 个测试文件 | 3,419 LOC |
| E2E 测试 | ✅ 38 个测试文件 | 6,447 LOC |
| 契约测试 | ✅ 3 个文件 | mq/db/cache_contract_test.go |
| 属性测试 | ✅ gopter + rapid | 核心逻辑有 property tests |

### 9. Infrastructure

| 组件 | 状态 | 备注 |
|---|---|---|
| Docker Compose | ✅ | docker-compose.yml + docker-compose.dev.yml |
| Kubernetes | ✅ | k8s/ 目录有部署文件 |
| CI/CD | ✅ | .github/workflows/ 有多个 workflow |
| Makefile | ✅ | 完整的构建/测试/部署命令 |

---

## ❌ 缺失 / 需补充

| 缺失项 | 优先级 | 说明 | 对应里程碑 |
|---|---|---|---|
| **双模式切换机制** | 🔴 高 | `DEPLOYMENT_MODE` 环境变量未在 cmd 层实际切换 adapters | M2 |
| **契约测试执行框架** | 🟡 中 | 3 个契约测试文件存在，但需确认能跑通 | M2 |
| **Grafana 看板** | 🟡 中 | Prometheus 指标有，但 Grafana dashboard JSON 缺失 | M3 |
| **MockDB 实现** | 🟡 中 | `pkg/plugins/database/` 有 PostgreSQL，需确认 MockDB/SQLite | M1 |
| **完整 Docker Compose 编排** | 🟡 中 | 需验证 docker-compose.yml 能否一键启动完整链路 | M3 |

---

## ⚠️ 迁移中间态包（理解其真实角色）

以下包不是"空壳"，而是迁移过程中创建的中间层。**不要向它们添加新功能**，但已有代码暂时保留。

| 包 | 代码量 | 真实角色 | 最终目标 |
|---|---|---|---|
| `pkg/domain/query/` | 53 LOC | Query 领域接口（Request, Result, Service） | 接口移到 `pkg/core/` |
| `pkg/application/indexing/` | 427 LOC | Indexing 运行时接口（EventEnvelope, SharedRuntime） | 接口移到 `pkg/core/` |
| `pkg/application/bootstrap/` | 574 LOC | 单体 wiring 逻辑（composition root 辅助） | 移到 `cmd/` |
| `pkg/application/query/` | 77 LOC | Query 遗留 facade | 删除 |
| `pkg/adapters/` | 250 LOC | 遗留兼容层 | 删除 |

**规则**: 新代码不要向这些包添加功能。接口定义在 `pkg/core/`，wiring 逻辑在 `cmd/`。

---

## 给 AI 的快速指引

1. **写接口** → `pkg/core/`
2. **写业务逻辑** → `pkg/services/{子目录}/`
3. **写单体适配器** → `pkg/plugins/{类型}/`
4. **写微服务适配器** → `pkg/infrastructure/{子目录}/`
5. **写入口** → `cmd/monolithic/` 或 `cmd/microservices/{服务}/`
6. **写测试** → 同目录下 `*_test.go` 或 `test/integration/`、`test/e2e/`
7. **依赖方向**: `cmd → plugins/infrastructure → services → core`（永远向内）

### 迁移中间态包（暂时可用，但不要新增）

- `pkg/domain/query/` — Query 领域接口，被 `pkg/plugins/api` 使用
- `pkg/application/indexing/` — Indexing 运行时，被 `pkg/services/indexing` 使用
- `pkg/application/bootstrap/` — wiring 逻辑，被所有 cmd 入口使用
- `pkg/adapters/` — 遗留兼容层

### 已知依赖违反（不要修复，除非任务明确要求）

详见 `docs/DEPENDENCY_GRAPH.md`。12 处违反是已知技术债，混入功能开发中修复会导致测试失败和编译错误。
