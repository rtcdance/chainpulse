# ChainPulse 双模式架构方案

> 首席架构师视角 · Web3 链上数据基建 + Go 企业级架构
> 兼容现有目录结构，零破坏性改动

---

## 目录

1. [双模式架构图](#1-双模式架构图)
2. [共享核心层设计](#2-共享核心层设计)
3. [服务职责与非功能策略](#3-服务职责与非功能策略)
4. [可插拔 Adapters 设计](#4-可插拔-adapters-设计)
5. [Platform 层（可观测性）](#5-platform-层可观测性)
6. [迁移路线与学习步骤](#6-迁移路线与学习步骤)
7. [风险与指标表](#7-风险与指标表)
8. [企业级调优指标清单](#8-企业级调优指标清单)

---

## 1. 双模式架构图

### 1.1 单体模式（本地调试）

```
┌─────────────────────────────────────────────────────────────────┐
│ cmd/monolithic/chainpulse · 单进程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌───────────────────────────────────────────────────────┐    │
│   │ Platform Layer (pkg/observability, core/logger)       │    │
│   │  · zap Logger                                         │    │
│   │  · IndexerMetrics / OTel Tracer                       │    │
│   │  · Health Check / viper Config                        │    │
│   └───────────────────────────────────────────────────────┘    │
│                         ↓ injects                              │
│   ┌───────────────────────────────────────────────────────┐    │
│   │ Shared Core (pkg/core) · PURE DOMAIN                  │    │
│   │  · interfaces: DataPullerPlugin, DatabasePlugin, etc  │    │
│   │  · models: BlockchainEvent, Block, ReorgStats         │    │
│   │  · in-process EventBus                                │    │
│   └───────────────────────────────────────────────────────┘    │
│                         ↑ implements                           │
│   ┌───────────────────────────────────────────────────────┐    │
│   │ In-Process Adapters (pkg/plugins)                     │    │
│   │  · MemoryMQ / MockMQ                                  │    │
│   │  · InMemoryCache                                      │    │
│   │  · SQLiteDB / MockDB                                  │    │
│   │  · GraphQL API Server                                 │    │
│   └───────────────────────────────────────────────────────┘    │
│                         ↑ uses                                 │
│   ┌───────────────────────────────────────────────────────┐    │
│   │ Application Services (pkg/services)                   │    │
│   │  · indexing.MultiChainIndexer                         │    │
│   │  · reorg.ReorgHandler                                 │    │
│   │  · query.QueryService                                 │    │
│   │  · decoder.EventDecoder                               │    │
│   └───────────────────────────────────────────────────────┘    │
│                         ↑ wires                                │
│   ┌───────────────────────────────────────────────────────┐    │
│   │ Composition Root (main.go) · NO business logic        │    │
│   └───────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                    ┌─────────────────┐
                    │ EVM Node / Mock │
                    └─────────────────┘
```

### 1.2 微服务模式（生产部署）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         微服务集群                                           │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐     │
│  │ API Gateway │   │ API Service │   │   Puller    │   │  Indexer    │     │
│  │  (GraphQL)  │──▶│   (Query)   │   │  (Multi-   │──▶│  (Event-    │     │
│  │             │   │             │   │   chain)    │   │  Processor) │     │
│  └─────────────┘   └──────┬──────┘   └──────┬──────┘   └─────────────┘     │
│         │                  │                 │                              │
│         │                  │                 ▼                              │
│         │                  │        ┌──────────────────┐                     │
│         │                  │        │  Kafka Topics    │                     │
│         │                  │        │  · raw_events    │                     │
│         │                  │        │  · indexed_events│                     │
│         │                  │        │  · reorg_events  │                     │
│         │                  │        │  · dlq           │                     │
│         │                  │        └────────┬─────────┘                     │
│         │                  │                 │                              │
│         │                  ▼                 ▼                              │
│         │           ┌────────────────────────────────────┐                  │
│         │           │     PostgreSQL    │     Redis      │                  │
│         │           │  (Event Store)    │   (Cache)      │                  │
│         │           └────────────────────────────────────┘                  │
│         │                                                                    │
│         └──────────────────────────────────────────────────┐                 │
│                                                            ▼                 │
│                                              ┌──────────────────────────┐   │
│                                              │  Consul Service Discovery │   │
│                                              └──────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                     ↓
                    ┌─────────────────────────────────┐
                    │  OTel Collector                 │
                    │  · Prometheus (Metrics)         │
                    │  · Jaeger (Distributed Traces)  │
                    │  · Loki (Logs)                  │
                    └─────────────────────────────────┘
```

### 1.3 共享核心边界

```
┌─────────────────────────────────────────────────────────────┐
│                   pkg/core  (PURE DOMAIN)                   │
│  interfaces: DataPullerPlugin, DatabasePlugin, CachePlugin  │
│             MQPlugin, APIPlugin, EventBus, Logger           │
│  models:     BlockchainEvent, Block, ReorgStats, Config     │
│  NO import of any adapter / infra package                   │
└─────────────────┬───────────────────────────────────────────┘
                  │ implements
        ┌─────────┴──────────┐
        ▼                    ▼
  pkg/plugins/*         pkg/infrastructure/*
  (monolithic impls)    (distributed impls)
```

---

## 2. 共享核心层设计

### 2.1 现有接口对齐

`pkg/core/plugin.go` 已定义完整 domain 接口，两种模式直接复用同一套：

| 接口 | 单体实现 | 微服务实现 |
|------|---------|----------|
| `DataPullerPlugin` | `pkg/plugins/pullers/` MockPuller | `pkg/infrastructure/data/DataPuller`（HTTP-JSONRPC） |
| `DatabasePlugin` | InMemoryDB / MockDB | `pkg/services/query/postgres_adapter.go` |
| `CachePlugin` | InMemoryCache | Redis adapter via `pkg/infrastructure/config/redis_config.go` |
| `MQPlugin` | `pkg/core/mq_plugin.go` MemoryMQ | Kafka via `pkg/infrastructure/config/kafka_config.go` |
| `EventBus` | `pkg/core/eventbus.go` in-process | Kafka topic router（微服务层） |

### 2.2 DDD 分层规则

```
domain/core          → 只定义接口与模型，零依赖外部包
application/services → 用例编排，只依赖 core 接口
adapters/plugins     → 实现 core 接口，依赖外部 SDK
platform/observability → 注入 Logger/Metrics/Trace，不含业务逻辑
cmd/*                → 组合根，只做 wire + lifecycle
```

**强制约束**：`pkg/services/**` 绝不 import `pkg/plugins/**` 或 `pkg/infrastructure/**`。
依赖方向永远向内：`cmd → adapters → services → core`。

### 2.3 多链支持设计决策

`core.Config.Blockchains map[string]BlockchainConfig` 已支持多链配置。每条链拥有独立的 `DefaultChainIndexer` 实例（`pkg/services/indexing/chain_indexer.go`），通过 `MultiChainIndexer` 统一管理。

**为什么这样设计**：不同链的最终性深度不同（Ethereum ~12 blocks，Polygon ~128 blocks，BSC ~15 blocks），必须按链隔离 `reorgThreshold` 与 `maxRollback`，避免误回滚。`ReorgHandler` 每条链独立实例化，互不干扰。

---

## 3. 服务职责与非功能策略

### 3.1 Puller（数据拉取服务）

**职责**：从 EVM/非-EVM 节点拉取原始区块与事件，发布到 MQ（单体为 EventBus，微服务为 Kafka）。

**核心代码**：`pkg/infrastructure/data/data_puller.go`、`block_height_tracker.go`

**依赖接口**：`core.DataPullerPlugin`、`core.MQPlugin`、`core.Logger`

| 维度 | 单体模式 | 微服务模式 |
|------|---------|----------|
| MQ | `core.EventBus`（内存 chan） | Kafka `raw_events` topic |
| RPC | MockPuller / 本地节点 | 多节点轮询 + 故障切换 |
| 扩缩容 | 单 goroutine per chain | 按链水平扩展 Pod |

**容错策略**：
- **RPC 故障切换**：`pkg/infrastructure/blockchain/blockchain_cluster.go` 维护节点池，失败自动切到备用节点
- **指数退避重试**：`pkg/services/resilience/retry_logic.go`，最大重试 3 次，初始间隔 1s，上限 30s
- **拉取进度持久化**：`block_height_tracker.go` checkpoint 落盘，重启后从断点续拉
- **背压控制**：MQ 满时 Puller 暂停拉取，防止内存溢出

**健康与指标**：
- `chainpulse_puller_block_lag{chain_id}` — 当前块 vs 链头差值，告警阈值 > 50
- `chainpulse_puller_rpc_errors_total{chain_id}` — RPC 错误率，告警 > 5%
- `chainpulse_puller_events_per_second{chain_id}` — 拉取吞吐量
- 健康端点：`GET /health/puller` 返回各链拉取状态

---

### 3.2 Indexer（索引服务）

**职责**：消费原始事件，ABI 解码，幂等持久化到数据库，维护缓存。

**核心代码**：`pkg/services/indexing/`、`pkg/services/decoder/`、`pkg/integrations/`

**依赖接口**：`core.DatabasePlugin`、`core.CachePlugin`、`core.MQPlugin`

| 维度 | 单体模式 | 微服务模式 |
|------|---------|----------|
| DB | MockDB / SQLite | PostgreSQL via postgres_adapter |
| Cache | InMemoryCache | Redis |
| 事件来源 | EventBus | Kafka consumer group |

**容错策略**：
- **幂等写入**：`pkg/services/processor/idempotency.go`，基于 `(chain_id, tx_hash, log_index)` 唯一键去重，防止重复消费
- **DLQ**：处理失败事件路由到 Kafka `dlq_events` topic，保留 7 天，支持人工重放
- **批量写入**：`core.DatabasePlugin.BatchStoreEvents()` 减少 DB RTT，批次大小由 `Config.BatchSize` 控制
- **ABI 平滑升级**：`pkg/services/decoder/contract_manager.go` 多版本 ABI 并存，按 block 范围路由解码逻辑

**健康与指标**：
- `chainpulse_indexer_events_processed_total{chain_id}`
- `chainpulse_indexer_error_rate{chain_id}` — 告警 > 1%
- `chainpulse_indexer_dlq_depth` — 告警 > 100
- `chainpulse_indexer_batch_latency_ms` — P99 告警 > 500ms

---

### 3.3 Query（查询服务）

**职责**：对外提供链上数据查询，支持缓存、熔断、降级、一致性验证。

**核心代码**：`pkg/services/query/`（`query_service.go`、`circuit_breaker.go`、`cache_service.go`、`consistency_checker.go`）

**依赖接口**：`core.DatabasePlugin`、`core.CachePlugin`

| 维度 | 单体模式 | 微服务模式 |
|------|---------|----------|
| Cache | InMemoryCache | Redis 分布式缓存 |
| DB | MockDB | PostgreSQL |
| 熔断器 | 内存实现 | Redis 分布式状态共享 |

**容错策略**：
- **熔断**：`circuit_breaker.go`，错误率 > 50% 且请求量 > 10/s 时熔断 30s，直接返回缓存或错误
- **缓存击穿防护**：`cache_warmer.go` 预热热点数据，`cache_middleware.go` 单机锁防并发穿透
- **降级**：DB 不可用时返回缓存数据（带 `X-Cache-Stale` 头），缓存也不可用时返回预设默认值
- **一致性检查**：`consistency_checker.go` 对比 Redis vs PostgreSQL，差异写入修复队列

**健康与指标**：
- `chainpulse_query_latency_ms{percentile}` — P99 告警 > 200ms
- `chainpulse_query_cache_hit_rate` — 目标 > 80%
- `chainpulse_query_circuit_breaker_state` — 0=closed, 1=open
- `chainpulse_query_consistency_mismatches` — 告警 > 10/min

---

### 3.4 API Gateway

**职责**：协议适配（GraphQL / gRPC / HTTP / WebSocket）、认证鉴权、限流、路由。

**核心代码**：`pkg/infrastructure/gateway/`、`pkg/plugins/api/`

| 维度 | 单体模式 | 微服务模式 |
|------|---------|----------|
| 协议 | GraphQL only | GraphQL + gRPC + HTTP + WebSocket |
| 限流 | 内存令牌桶 | Redis 分布式限流 |
| 认证 | mock | JWT + API Key |

**容错策略**：
- **多协议适配**：`multi_protocol_api.go` 同一业务逻辑，不同协议入口
- **连接池管理**：`websocket_subscription.go` 维护 WebSocket 连接池，单 Pod 上限 10000
- **限流**：`api/auth_middleware.go` 限流中间件，按 API Key 限 1000 req/min，按 IP 限 100 req/min

---

### 3.5 Reorg Handler

**职责**：检测链重组，回滚受影响数据，触发下游重索引。

**核心代码**：`pkg/services/reorg/reorg_handler.go`

**容错策略**：
- **最终性确认**：`reorgThreshold` 按链配置（Ethereum 12，Polygon 128），低于该高度的块不对外暴露
- **原子回滚**：`RollbackEvents()` 在事务中删除块范围内事件，成功后发 `reorg_events` 通知 Indexer
- **最大回滚深度**：`maxRollback` 限制，超过阈值人工介入

**健康与指标**：
- `chainpulse_reorg_detected_total{chain_id}`
- `chainpulse_reorg_blocks_rolled_back{chain_id}`
- `chainpulse_reorg_recovery_time_ms` — 告警 > 30s

---

## 4. 可插拔 Adapters 设计

### 4.1 接口定义（已存在于 pkg/core）

```go
// pkg/core/plugin.go

// DataPullerPlugin pulls events from blockchain sources
type DataPullerPlugin interface {
    Plugin
    PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]BlockchainEvent, error)
    GetLatestBlock(ctx context.Context) (uint64, error)
    SubscribeToEvents(ctx context.Context, handler func(BlockchainEvent)) error
    GetStats() map[string]interface{}
}

// DatabasePlugin manages database operations
type DatabasePlugin interface {
    Plugin
    StoreEvent(ctx context.Context, event interface{}) error
    BatchStoreEvents(ctx context.Context, events []interface{}) error
    QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error)
    // ... more methods
}

// MQPlugin manages message queue operations
type MQPlugin interface {
    Plugin
    Publish(ctx context.Context, topic string, message []byte) error
    Subscribe(ctx context.Context, topic string, handler func([]byte)) error
    GetQueueDepth(ctx context.Context, topic string) (int64, error)
}
```

### 4.2 单体实现（内存/Mock）

**位置**：`pkg/plugins/`（新建）或直接使用 `pkg/core/mq_plugin.go` 的 MemoryMQ

```go
// pkg/plugins/mq/memory_mq.go
package mq

import (
    "context"
    "sync"
    "chainpulse/pkg/core"
)

type MemoryMQ struct {
    mu        sync.RWMutex
    topics    map[string][]chan []byte
    logger    core.Logger
}

func NewMemoryMQ(logger core.Logger) *MemoryMQ {
    return &MemoryMQ{
        topics: make(map[string][]chan []byte),
        logger: logger,
    }
}

func (m *MemoryMQ) Name() string { return "memory-mq" }
func (m *MemoryMQ) Version() string { return "1.0.0" }

func (m *MemoryMQ) Initialize(config core.Config) error { return nil }
func (m *MemoryMQ) Start() error { return nil }
func (m *MemoryMQ) Stop() error { return nil }
func (m *MemoryMQ) Health() error { return nil }

func (m *MemoryMQ) Publish(ctx context.Context, topic string, message []byte) error {
    m.mu.RLock()
    defer m.mu.RUnlock()

    for _, ch := range m.topics[topic] {
        select {
        case ch <- message:
        default: // drop if full (背压)
            m.logger.Warn("memory mq channel full, dropping message", "topic", topic)
        }
    }
    return nil
}

func (m *MemoryMQ) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    ch := make(chan []byte, 100)
    m.topics[topic] = append(m.topics[topic], ch)

    go func() {
        for {
            select {
            case msg := <-ch:
                handler(msg)
            case <-ctx.Done():
                return
            }
        }
    }()
    return nil
}

func (m *MemoryMQ) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
    return 0, nil // memory mq has no backlog
}
```

### 4.3 微服务实现（Kafka）

**位置**：`pkg/infrastructure/config/kafka_config.go`（已存在，可扩展为完整 MQPlugin 实现）

```go
// pkg/infrastructure/mq/kafka_mq.go
package mq

import (
    "context"
    "github.com/segmentio/kafka-go"
    "chainpulse/pkg/core"
)

type KafkaMQ struct {
    writers map[string]*kafka.Writer
    readers map[string]*kafka.Reader
    logger  core.Logger
    config  KafkaConfig
}

func NewKafkaMQ(config KafkaConfig, logger core.Logger) *KafkaMQ {
    return &KafkaMQ{
        writers: make(map[string]*kafka.Writer),
        readers: make(map[string]*kafka.Reader),
        logger:  logger,
        config:  config,
    }
}

func (k *KafkaMQ) Name() string { return "kafka-mq" }

func (k *KafkaMQ) Publish(ctx context.Context, topic string, message []byte) error {
    writer, ok := k.writers[topic]
    if !ok {
        writer = &kafka.Writer{
            Addr:     kafka.TCP(k.config.Brokers...),
            Topic:    topic,
            Balancer: &kafka.LeastBytes{},
        }
        k.writers[topic] = writer
    }

    return writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte("chainpulse"),
        Value: message,
    })
}

func (k *KafkaMQ) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: k.config.Brokers,
        Topic:   topic,
        GroupID: "chainpulse-indexer",
    })

    go func() {
        for {
            msg, err := reader.ReadMessage(ctx)
            if err != nil {
                k.logger.Error("kafka read error", "error", err.Error())
                if ctx.Err() != nil {
                    return
                }
                continue
            }
            handler(msg.Value)
        }
    }()
    return nil
}
```

### 4.4 契约测试

```go
// test/integration/mq_contract_test.go
package integration

import (
    "context"
    "testing"
    "time"
    "chainpulse/pkg/core"
    "chainpulse/pkg/plugins/mq"
    inframq "chainpulse/pkg/infrastructure/mq"
)

// MQContractTest 定义 MQ 契约
func MQContractTest(t *testing.T, factory func() core.MQPlugin) {
    ctx := context.Background()
    mq := factory()

    t.Run("publish_and_subscribe", func(t *testing.T) {
        received := make(chan []byte, 1)

        err := mq.Subscribe(ctx, "test-topic", func(msg []byte) {
            received <- msg
        })
        require.NoError(t, err)

        err = mq.Publish(ctx, "test-topic", []byte("hello"))
        require.NoError(t, err)

        select {
        case msg := <-received:
            assert.Equal(t, "hello", string(msg))
        case <-time.After(5 * time.Second):
            t.Fatal("timeout waiting for message")
        }
    })

    t.Run("multiple_subscribers", func(t *testing.T) {
        // 验证多订阅者都收到消息
    })

    t.Run("queue_depth", func(t *testing.T) {
        // 验证队列深度查询
    })
}

func TestMemoryMQ(t *testing.T) {
    MQContractTest(t, func() core.MQPlugin {
        return mq.NewMemoryMQ(core.NewDefaultLogger(core.LogLevelInfo))
    })
}

func TestKafkaMQ(t *testing.T) {
    MQContractTest(t, func() core.MQPlugin {
        // 需要 Docker TestContainer 启动 Kafka
        return inframq.NewKafkaMQ(testKafkaConfig, testLogger)
    })
}
```

---

## 5. Platform 层（可观测性）

### 5.1 统一标签注入

所有指标、日志、trace 必须携带以下标签：

| 标签 | 说明 | 示例 |
|------|------|------|
| `chain_id` | 区块链 ID | ethereum, polygon, bsc |
| `service` | 服务名 | puller, indexer, query, api-gateway |
| `operation` | 操作类型 | pull_events, store_events, query_events |
| `block_height` | 当前区块高度 | 12345678 |

### 5.2 指标定义

```go
// pkg/observability/metrics.go

var (
    // 链级吞吐量
    EventsPerSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "chainpulse_events_per_second",
        Help: "Events indexed per second",
    }, []string{"chain_id", "service"})

    // 延迟分位数
    IndexingLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "chainpulse_indexing_latency_ms",
        Help:    "Event indexing latency in milliseconds",
        Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1, 2, 4, 8, 16, 32, 64, 128, 256, 512
    }, []string{"chain_id"})

    // 错误分类
    ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "chainpulse_errors_total",
        Help: "Total errors by type",
    }, []string{"chain_id", "error_type", "service"}) // error_type: rpc, db, mq, decode, timeout
)
```

### 5.3 分布式追踪

```go
// 在 Puller 中
ctx, span := tracer.Start(ctx, "pull_events",
    trace.WithAttributes(
        attribute.String("chain_id", chainID),
        attribute.Int64("from_block", int64(fromBlock)),
        attribute.Int64("to_block", int64(toBlock)),
    ))
defer span.End()
```

---

## 6. 迁移路线与学习步骤

### 6.1 阶段划分

```
Phase 1: 单体调试（当前状态）
├── 目标：本地单进程调试，快速验证业务逻辑
├── 实现：MemoryMQ + InMemoryCache + MockDB
├── 验证：单元测试 + 集成测试通过
└── 耗时：1-2 天（已完成）

Phase 2: 双模式切换
├── 目标：通过配置切换单体/微服务
├── 实现：
│   ├── 实现 pkg/plugins/* 所有单体 adapters
│   ├── 完善 pkg/infrastructure/* 微服务 adapters
│   └── cmd/ 启动层根据 DEPLOYMENT_MODE 选择 adapters
├── 验证：
│   ├── 契约测试：单体与微服务 MQ/DB/Cache 行为一致
│   └── 集成测试：同一份 service 代码跑两种模式
└── 耗时：3-5 天

Phase 3: 微服务部署
├── 目标：K8s 集群部署
├── 实现：
│   ├── k8s/ 目录完善 Helm charts
│   ├── Kafka / PostgreSQL / Redis 集群部署
│   └── 配置 service mesh (Consul)
├── 验证：
│   ├── E2E 测试：完整链路测试
│   └── 压力测试：模拟多链高并发
└── 耗时：5-7 天

Phase 4: 生产监控闭环
├── 目标：可观测性 + 告警 + 自动恢复
├── 实现：
│   ├── Prometheus + Grafana 看板
│   ├── 关键指标告警（P99 延迟、错误率、reorg）
│   └── DLQ 自动/人工重放流程
├── 验证：
│   ├── 混沌测试：模拟节点故障、网络分区
│   └── 演练：reorg 恢复、服务扩容
└── 耗时：3-5 天
```

### 6.2 学习步骤建议

| 步骤 | 内容 | 代码位置 | 学习目标 |
|------|------|---------|---------|
| 1 | 理解 core 接口 | `pkg/core/plugin.go` | 掌握 DDD 分层，接口定义在中心 |
| 2 | 阅读 chain_indexer | `pkg/services/indexing/chain_indexer.go` | 理解多链索引逻辑 |
| 3 | 阅读 reorg_handler | `pkg/services/reorg/reorg_handler.go` | 理解重组检测与回滚 |
| 4 | 实现 MemoryMQ | `pkg/plugins/mq/memory_mq.go` | 练习 adapter 模式 |
| 5 | 添加契约测试 | `test/integration/mq_contract_test.go` | 掌握契约测试编写 |
| 6 | 配置双模式 | `pkg/infrastructure/deployment/` | 理解部署模式抽象 |
| 7 | 部署到 K8s | `k8s/` | 理解微服务运维 |

---

## 7. 风险与指标表

| 风险场景 | 防范措施 | 关键指标 | 告警阈值 | 恢复路径 |
|---------|---------|---------|---------|---------|
| **Reorg 深度超限** | 配置 maxRollback，超限人工介入 | `reorg_blocks_rolled_back` | > 100 blocks | 暂停服务，人工确认后重置 checkpoint |
| **Cache 不一致** | 一致性检查 + 修复队列 | `consistency_mismatches` | > 10/min | 自动修复队列消费，不一致时读 DB |
| **多链 RPC 故障** | 节点池 + 熔断 + 指数退避 | `rpc_errors_total` / `rpc_available_nodes` | 错误率 > 5% / 可用节点 < 2 | 切换到备用节点，告警通知 |
| **API 限流击穿** | 令牌桶 + 分布式限流 | `rate_limit_hits` | > 100/min | 返回 429，客户端退避重试 |
| **Kafka 消费延迟** | 消费者组扩容 + 背压 | `consumer_lag` | > 1000 messages | 扩容 consumer pod |
| **DLQ 堆积** | 监控 + 自动重试 | `dlq_depth` | > 100 | 人工审查后批量重放 |
| **ABI 解码失败** | 多版本 ABI + 未知事件兜底 | `decode_errors_total` | > 0.1% | 更新 ABI 配置，重放失败事件 |
| **非 EVM 链差异** | 链特定 adapter 实现 | `chain_adapter_errors` | > 0 | 链适配器隔离，不影响其他链 |

---

## 8. 企业级调优指标清单

### 8.1 单体调试模式优化

| 指标 | 目标值 | 调优手段 |
|------|--------|---------|
| 单链吞吐量 | > 100 events/sec | 调整 batch_size, worker_pool_size |
| 内存使用 | < 512MB | 限制 chan buffer, 定期 GC |
| 启动时间 | < 5s | 延迟初始化非关键组件 |
| 测试覆盖率 | > 80% | 契约测试 + 属性测试 |

**监控建议**：
```yaml
# docker-compose.local.yml 中加入
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
```

### 8.2 微服务部署模式优化

| 指标 | 目标值 | 调优手段 | 告警阈值 |
|------|--------|---------|---------|
| 链级 throughput | > 500 events/sec/chain | Partition by chain_id, 水平扩容 | < 300 |
| Reorg 恢复时长 | < 30s | 优化 rollback SQL，加索引 | > 30s |
| Query P99 延迟 | < 100ms | Redis 缓存热点，PostgreSQL 索引 | > 200ms |
| DLQ 消费速率 | = 生产速率 | 独立 consumer 处理 DLQ | 积压 > 100 |
| 缓存命中率 | > 85% | 分析热点 key，调整 TTL | < 70% |
| 服务可用性 | 99.9% | 多副本 + 健康检查 + 自动重启 | < 99.5% |
| Lock contention | < 1% | 减少分布式锁粒度，用乐观锁 | > 5% |

**扩缩容策略**：
```yaml
# HPA 示例
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: chainpulse-indexer
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: indexer
  minReplicas: 2
  maxReplicas: 20
  metrics:
    - type: Pods
      pods:
        metric:
          name: kafka_consumer_lag
        target:
          type: AverageValue
          averageValue: "500"
```

---

## 附录：代码生成命令

```bash
# 1. 生成 protobuf（如果使用 gRPC）
protoc --go_out=. --go-grpc_out=. pkg/plugins/api/proto/*.proto

# 2. 运行契约测试
go test -v ./test/integration/... -run Contract

# 3. 单体模式本地运行
DEPLOYMENT_MODE=monolithic go run cmd/monolithic/chainpulse/main.go

# 4. 微服务模式启动（需 Kafka/PostgreSQL/Redis）
DEPLOYMENT_MODE=microservice go run cmd/microservices/puller/main.go
DEPLOYMENT_MODE=microservice go run cmd/microservices/event-processor/main.go
DEPLOYMENT_MODE=microservice go run cmd/microservices/api-service/main.go
DEPLOYMENT_MODE=microservice go run cmd/microservices/api-gateway/main.go

# 5. 压力测试
go test -bench=. -benchmem ./test/performance/...
```

---

## 9. 代码规范与静态分析

### 9.1 强制工具链

| 工具 | 用途 | 配置文件 | 触发时机 |
|------|------|---------|--------|
| `gofumpt` | 格式化（比 gofmt 更严格） | `.golangci.yml` | `make fmt` / 保存时 |
| `golangci-lint` | 静态分析聚合器 | `.golangci.yml` | `make lint` / CI |
| `go vet` | 标准静态检查 | 内置 | `make vet` / CI |
| `staticcheck` | 深度静态分析 | `.golangci.yml` | `make staticcheck` / CI |
| `gosec` | 安全扫描 | `.golangci.yml` | `make security` / CI |

### 9.2 关键 Lint 规则

- 圈复杂度 ≤ 15（`gocyclo`）
- 函数长度 ≤ 80 行 / 50 语句（`funlen`）
- 所有导出类型必须有注释（`revive: exported`）
- 错误必须处理，不允许 `_ = err`（`errcheck`）
- `pkg/services/**` 禁止 import `pkg/plugins/**` 或 `pkg/infrastructure/**`

### 9.3 依赖方向检查

```
cmd/* → plugins/infrastructure → services → core
                                     ↑
                            platform (logger/metrics/trace)
```

使用 `go-cleanarch` 或 CI 中的 import path 检查脚本验证此约束。

---

## 10. 测试体系

### 10.1 测试金字塔

```
     E2E Tests      · test/e2e/         · -tags=e2e      · nightly / 手动
  ─────────────────
  Integration Tests · test/integration/ · -tags=integration · PR 路径触发
  ─────────────────
  Contract Tests    · test/contracts/   · -tags=integration · PR 路径触发
  ─────────────────
  Unit Tests        · pkg/**/*_test.go  · (默认)           · 每次 push/PR
```

### 10.2 各层测试策略

| 层级 | 目标覆盖率 | 依赖 | Mock 策略 |
|------|-----------|------|----------|
| 单元测试 | > 80% | 无外部依赖 | Mock 所有接口 |
| 契约测试 | 100% 接口方法 | MemoryMQ / MockDB | 两种实现跑同一套断言 |
| 集成测试 | 关键链路 | Docker services（PG/Redis/Kafka） | 真实基础设施 |
| E2E 测试 | 主流程场景 | docker-compose.dev.yml | 完整单体或微服务启动 |

### 10.3 契约测试（Adapter 一致性保障）

核心原则：同一套 `XxxContractTest(t, factory)` 函数，分别用单体实现和微服务实现调用，保证行为一致。

```
test/contracts/
├── mq_contract_test.go      # MQPlugin 契约
└── db_contract_test.go      # DatabasePlugin 契约
```

**新增 Adapter 必须通过对应契约测试才能合并。**

### 10.4 测试辅助工具

| 工具 | 用途 | 位置 |
|------|------|------|
| `testify/assert` | 断言（失败继续） | 集成测试 |
| `testify/require` | 断言（失败停止） | 单元/集成测试 |
| `testify/mock` | Mock 生成 | 单元测试 |
| `testify/suite` | 测试套件 | 集成/E2E |
| `gopter` | 属性测试 | `test/e2e/*_property_test.go` |
| `testcontainers-go` | 真实 Docker 服务 | 集成测试 |
| `test/helpers/testutil.go` | 通用 context/retry 工具 | 所有层 |

### 10.5 常用测试命令

```bash
make test-unit          # 单元测试（-race -short）
make test-integration   # 集成测试（需 docker-up）
make test-e2e           # E2E 测试
make test-coverage      # 覆盖率报告 → build/coverage/coverage.html
make test-bench         # 基准测试

# 调试单个测试
go test -v -run TestIndexEvents ./pkg/services/indexing

# Delve 调试
dlv test ./pkg/services/indexing -- -test.run TestIndexEvents
```

---

## 11. CI/CD 流水线

### 11.1 流水线分层

```
.github/workflows/
├── ci.yml              # 每次 push/PR：lint → unit test → build → security scan
├── ci-integration.yml  # 路径触发 / nightly / 手动：integration → contract → e2e
└── release.yml         # tag 触发：多平台构建 → Docker 镜像 → GitHub Release
```

### 11.2 ci.yml 流程

```
push/PR → main/master/develop
    │
    ├── lint        golangci-lint + go vet（并行）
    ├── test-unit   go test -race -short ./pkg/...（并行）
    ├── build       需 lint + test-unit 通过后执行
    │               monolithic + 4 microservices 入口
    └── security    gosec SARIF 上传（独立，不阻塞 build）
```

### 11.3 ci-integration.yml 触发条件

| 触发方式 | 场景 |
|---------|------|
| `workflow_dispatch` | 手动触发，可选是否运行 E2E |
| `schedule: 0 2 * * *` | 每晚 02:00 UTC 自动运行（含 E2E） |
| PR 路径变更 | `test/**` 或 `pkg/services/**` 有改动时 |

**nightly 失败**自动创建 GitHub Issue（`bug` + `test-failure` + `nightly` 标签）。

### 11.4 Go 版本管理

所有 workflow 使用 `go-version-file: go.mod`，自动读取项目声明版本（当前 `go 1.24.0`），无需在 workflow 中硬编码版本号。

### 11.5 本地 CI 验证

```bash
make ci      # fmt-check + lint + vet + test-unit（等同于 CI 核心检查）
make cd      # build + test-coverage

# 启动本地基础设施
make docker-up   # 启动 PG / Redis / Kafka / Prometheus / Grafana / Jaeger
make docker-down # 停止
```

### 11.6 IDE 调试配置

```
.vscode/
├── launch.json    # Debug Monolithic / Debug Puller / Debug Tests / Attach
└── settings.json  # gofumpt 格式化 / golangci-lint / 覆盖率显示
```

热重载开发：`.air.toml` 配置，`air` 命令启动自动编译。

---

**总结**：本方案在现有 ChainPulse 代码基础上，通过清晰的 DDD 分层、可插拔 adapters、统一的 platform 可观测性，实现"单体调试 + 微服务部署"双模式。核心设计决策是：domain 层纯接口、service 层零外部依赖、启动层只做 wire。这样既保留了本地开发的便利性，又具备生产级扩展能力。
