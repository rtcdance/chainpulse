# M2: 双模式切换（严格按 ARCHITECTURE_v1.md 蓝图 Phase 2 执行）

> 这是可以直接发给 GPT 的完整 prompt。
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M2 - 实现双模式切换

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`
- M1 状态: 单体端到端链路已修复（M1-1 完成）

### 蓝图 Phase 2 定义

```
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
```

### 蓝图对双模式的具体要求

#### 单体模式（§1.1）
| 组件 | 实现 | 位置 |
|---|---|---|
| MQ | MemoryMQ（内存 chan） | `pkg/plugins/mq/memory_mq.go` |
| DB | MockDB / SQLite | `pkg/plugins/database/mock_db.go` |
| Cache | InMemoryCache | `pkg/plugins/cache/inmemory_cache.go` |
| API | GraphQL only | `pkg/plugins/api/graphql/` |
| 事件流 | EventBus（内存） | `pkg/core/eventbus.go` |

#### 微服务模式（§1.2）
| 组件 | 实现 | 位置 |
|---|---|---|
| MQ | Kafka（`raw_events`, `indexed_events`, `reorg_events`, `dlq`） | `pkg/plugins/mq/kafka_mq.go` |
| DB | PostgreSQL | `pkg/infrastructure/database/` |
| Cache | Redis | `pkg/plugins/cache/redis_cache.go` |
| API | GraphQL + gRPC + HTTP + WebSocket | `pkg/infrastructure/gateway/` |
| 事件流 | Kafka consumer group | `pkg/infrastructure/processing/` |
| 服务发现 | Consul | `pkg/infrastructure/deployment/consul_config.go` |

#### 共享核心（§1.3）
```
pkg/core (PURE DOMAIN)
  ↑ implements
pkg/plugins/* (单体 adapters)    pkg/infrastructure/* (微服务 adapters)
```

### 当前状态：8 个断裂点

#### 断裂 1: DEPLOYMENT_MODE 环境变量未解析
- 蓝图要求: cmd 启动层根据 `DEPLOYMENT_MODE` 选择 adapters
- 当前: 环境变量存在但未在 main.go 中实际切换
- 修复: 在 cmd 层解析 `DEPLOYMENT_MODE=monolithic|microservice`，选择对应的 adapter 实现

#### 断裂 2: 契约测试无法运行
- 蓝图要求: 契约测试验证单体与微服务 MQ/DB/Cache 行为一致
- 当前: `test/contracts/` 有 3 个文件但 `go test ./test/contracts/...` 匹配不到包
- 原因: 文件有 `//go:build integration` 标签，需要 `go test -tags=integration`
- 修复: 确保契约测试能运行，验证 MemoryMQ vs KafkaMQ、MockDB vs PostgreSQL、InMemoryCache vs RedisCache 行为一致

#### 断裂 3: 4 个微服务入口未验证独立启动
- 蓝图要求: Puller / EventProcessor / APIService / APIGateway 各自可独立启动
- 当前: `cmd/microservices/*/main.go` 有 4 个入口，但未验证独立启动
- 修复: 验证每个微服务入口能独立编译和启动

#### 断裂 4: 集成测试未验证双模式
- 蓝图要求: 同一份 service 代码跑两种模式
- 当前: 有集成测试但未验证双模式一致性
- 修复: 运行集成测试，验证同一份 service 代码在单体和微服务模式下行为一致

#### 断裂 5: Kafka topics 未定义
- 蓝图要求: 微服务模式使用 `raw_events`、`indexed_events`、`reorg_events`、`dlq` 四个 Kafka topic
- 当前: KafkaMQ 实现存在但 topic 配置不完整
- 修复: 完善 Kafka topic 配置和消费者组

#### 断裂 6: Consul 服务发现未接入
- 蓝图要求: 微服务模式配置 Consul service discovery
- 当前: `pkg/infrastructure/deployment/consul_config.go` 有基础实现但未接入
- 修复: 在微服务启动时注册到 Consul

#### 断裂 7: 微服务模式 DB/Cache 未验证
- 蓝图要求: 微服务模式用 PostgreSQL + Redis
- 当前: docker-compose.yml 有 PostgreSQL 和 Redis 服务
- 修复: 验证微服务模式能连接 PostgreSQL 和 Redis

#### 断裂 8: 多协议 API 未验证
- 蓝图要求: 微服务模式支持 GraphQL + gRPC + HTTP + WebSocket
- 当前: `pkg/infrastructure/gateway/multi_protocol_api.go` 有实现
- 修复: 验证多协议 API 在微服务模式下可用

### 完整数据流（双模式）

```
DEPLOYMENT_MODE=monolithic:
  cmd/monolithic/chainpulse
    → MemoryMQ (内存 chan)
    → MockDB (内存)
    → InMemoryCache (内存)
    → EventBus (内存)
    → GraphQL API only

DEPLOYMENT_MODE=microservice:
  cmd/microservices/puller
    → KafkaMQ (raw_events topic)
    → Consul 注册
  cmd/microservices/event-processor
    → KafkaMQ (raw_events → indexed_events/reorg_events/dlq)
    → PostgreSQL
    → Redis Cache
  cmd/microservices/api-service
    → PostgreSQL (查询)
    → Redis Cache
    → CircuitBreaker + Cache + Degradation
  cmd/microservices/api-gateway
    → GraphQL + gRPC + HTTP + WebSocket
    → RateLimiter + AuthMiddleware
    → WebSocket 连接池 (max 10000)
```

### 目标
实现蓝图 Phase 2 的双模式切换，使同一套 service 代码能通过 `DEPLOYMENT_MODE` 环境变量在单体和微服务之间切换。

### 成功标准

#### 基础（必须全部通过）
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过

#### 双模式切换
- [ ] `DEPLOYMENT_MODE=monolithic` 时使用 MemoryMQ + MockDB + InMemoryCache + EventBus
- [ ] `DEPLOYMENT_MODE=microservice` 时使用 KafkaMQ + PostgreSQL + RedisCache + Kafka consumer group
- [ ] 同一份 service 代码（`pkg/services/`）在两种模式下都能正常工作
- [ ] 4 个微服务入口各自可独立编译和启动

#### 契约测试
- [ ] `go test -tags=integration ./test/contracts/...` 通过
- [ ] MemoryMQ 和 KafkaMQ 通过同一套契约测试断言
- [ ] MockDB 和 PostgreSQL 通过同一套契约测试断言
- [ ] InMemoryCache 和 RedisCache 通过同一套契约测试断言

#### 微服务基础设施
- [ ] Kafka topics 配置完整（raw_events, indexed_events, reorg_events, dlq）
- [ ] 微服务启动时注册到 Consul
- [ ] 微服务模式能连接 PostgreSQL 和 Redis
- [ ] 多协议 API（GraphQL + gRPC + HTTP + WebSocket）可用

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`，特别是:
1. 新代码只写入正确的层
2. 不要往 `pkg/domain/`、`pkg/application/`、`pkg/adapters/` 添加新功能
3. 不要修改已有依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
4. 不要重构已工作的代码

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，§1.1 单体 / §1.2 微服务 / Phase 2**
- `cmd/monolithic/chainpulse/main.go` — 单体入口（M1-1 已修复）
- `cmd/microservices/puller/main.go` — Puller 微服务入口
- `cmd/microservices/event-processor/main.go` — EventProcessor 微服务入口
- `cmd/microservices/api-service/main.go` — APIService 微服务入口
- `cmd/microservices/api-gateway/main.go` — APIGateway 微服务入口
- `pkg/plugins/mq/memory_mq.go` — MemoryMQ（单体 MQ）
- `pkg/plugins/mq/kafka_mq.go` — KafkaMQ（微服务 MQ）
- `pkg/plugins/database/mock_db.go` — MockDB（单体 DB）
- `pkg/plugins/cache/inmemory_cache.go` — InMemoryCache（单体 Cache）
- `pkg/plugins/cache/redis_cache.go` — RedisCache（微服务 Cache）
- `pkg/infrastructure/database/` — PostgreSQL（微服务 DB）
- `pkg/infrastructure/gateway/multi_protocol_api.go` — 多协议 API
- `pkg/infrastructure/deployment/consul_config.go` — Consul 服务发现
- `pkg/core/eventbus.go` — EventBus（内存）
- `test/contracts/mq_contract_test.go` — MQ 契约测试
- `test/contracts/db_contract_test.go` — DB 契约测试
- `test/contracts/cache_contract_test.go` — Cache 契约测试
- `docker/docker-compose.yml` — 完整基础设施编排
- `docker/docker-compose.dev.yml` — 开发环境基础设施

### 修复步骤（按顺序）

**Step 1: 解析 DEPLOYMENT_MODE 并选择 adapters**
```
在每个 cmd/*/main.go 中:
1. 读取 DEPLOYMENT_MODE 环境变量（默认 monolithic）
2. 根据模式选择对应的 adapter 实现:
   - monolithic: MemoryMQ, MockDB, InMemoryCache, EventBus
   - microservice: KafkaMQ, PostgreSQL, RedisCache, Kafka consumer group
3. 所有 service 代码（pkg/services/）不感知 deployment mode
```

**Step 2: 修复并运行契约测试**
```
1. 确保契约测试能运行: go test -tags=integration ./test/contracts/...
2. 验证 MemoryMQ 和 KafkaMQ 通过同一套断言
3. 验证 MockDB 和 PostgreSQL 通过同一套断言
4. 验证 InMemoryCache 和 RedisCache 通过同一套断言
```

**Step 3: 验证 4 个微服务独立启动**
```
1. go build ./cmd/microservices/puller/
2. go build ./cmd/microservices/event-processor/
3. go build ./cmd/microservices/api-service/
4. go build ./cmd/microservices/api-gateway/
5. 各自能独立启动（需要 Kafka/PostgreSQL/Redis 运行）
```

**Step 4: 完善 Kafka topics 和 Consul 注册**
```
1. 配置 4 个 Kafka topic: raw_events, indexed_events, reorg_events, dlq
2. 微服务启动时注册到 Consul
3. 验证服务发现可用
```

**Step 5: 验证微服务模式完整链路**
```
1. docker-compose up 启动 Kafka + PostgreSQL + Redis
2. DEPLOYMENT_MODE=microservice 启动 4 个微服务
3. 验证 Puller → Kafka → EventProcessor → PostgreSQL → APIService → APIGateway 完整链路
4. 验证多协议 API（GraphQL + gRPC + HTTP + WebSocket）可用
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）
- **必须与 ARCHITECTURE_v1.md 蓝图 Phase 2 一致**
- **本阶段只做双模式切换和契约测试，不做以下事项**:
  - K8s 部署（M3 做）
  - 压力测试（M3 做）
  - 混沌测试（M3 做）
  - Grafana 看板（M3 做）
  - 告警规则配置（M3 做）
  - DLQ 重放流程（M3 做）
  - 可观测性完善（M1-1c 做）

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 契约测试
go test -tags=integration ./test/contracts/... -v
# 单体模式验证
DEPLOYMENT_MODE=monolithic make run-monolithic
# 微服务模式验证（需要 docker-compose up 先启动基础设施）
docker-compose up -d
DEPLOYMENT_MODE=microservice go run cmd/microservices/puller/
DEPLOYMENT_MODE=microservice go run cmd/microservices/event-processor/
DEPLOYMENT_MODE=microservice go run cmd/microservices/api-service/
DEPLOYMENT_MODE=microservice go run cmd/microservices/api-gateway/
```
