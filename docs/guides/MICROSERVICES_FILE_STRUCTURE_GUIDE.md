# 微服务文件结构完整指南

**Date**: January 12, 2026  
**Purpose**: 详细说明微服务相关的所有文件位置和用途

---

## 📁 完整文件结构

```
chainpulse/
│
├── cmd/                                    ← 微服务程序入口
│   ├── chainpulse/                         ← 单体服务 (原有)
│   │   ├── main.go                         ← 单体服务主程序
│   │   ├── Makefile                        ← 构建脚本
│   │   ├── QUICKSTART.md                   ← 快速开始
│   │   ├── INTEGRATION_GUIDE.md            ← 集成指南
│   │   └── README.md                       ← 说明文档
│   │
│   ├── chainpulse-api-gateway/             ← API 网关微服务
│   │   ├── main.go                         ← 程序入口
│   │   ├── Makefile                        ← 构建脚本
│   │   ├── QUICKSTART.md                   ← 快速开始
│   │   └── README.md                       ← 说明文档
│   │
│   ├── chainpulse-api-service/             ← API 服务微服务
│   │   ├── main.go                         ← 程序入口
│   │   ├── Makefile                        ← 构建脚本
│   │   ├── QUICKSTART.md                   ← 快速开始
│   │   └── README.md                       ← 说明文档
│   │
│   ├── chainpulse-event-processor/         ← 事件处理器微服务
│   │   ├── main.go                         ← 程序入口
│   │   ├── Makefile                        ← 构建脚本
│   │   ├── QUICKSTART.md                   ← 快速开始
│   │   └── README.md                       ← 说明文档
│   │
│   └── chainpulse-puller/                  ← 数据拉取器微服务
│       ├── main.go                         ← 程序入口
│       ├── Makefile                        ← 构建脚本
│       ├── QUICKSTART.md                   ← 快速开始
│       └── README.md                       ← 说明文档
│
├── pkg/                                    ← 核心包
│   ├── core/                               ← 核心服务
│   │   ├── logger.go                       ← 日志服务
│   │   ├── metrics.go                      ← 指标收集
│   │   ├── registry.go                     ← 插件注册表
│   │   ├── config.go                       ← 配置管理
│   │   └── ...
│   │
│   ├── infrastructure/                     ← 基础设施
│   │   ├── deployment/                     ← 部署相关
│   │   │   ├── deployment_mode.go          ← 部署模式管理
│   │   │   ├── monolithic_deployment.go    ← 单体部署
│   │   │   ├── microservice_deployment.go  ← 微服务部署
│   │   │   ├── api_gateway_cluster.go      ← API 网关集群
│   │   │   ├── data_puller_cluster.go      ← 数据拉取集群
│   │   │   ├── event_processor_cluster.go  ← 事件处理集群
│   │   │   ├── service_registry.go         ← 服务注册
│   │   │   ├── service_discovery_advanced.go ← 服务发现
│   │   │   └── health_check.go             ← 健康检查
│   │   │
│   │   └── ...
│   │
│   ├── plugins/                            ← 插件系统
│   │   ├── api/                            ← API 插件
│   │   │   ├── gateway.go                  ← API 网关
│   │   │   ├── http/                       ← HTTP 支持
│   │   │   ├── graphql/                    ← GraphQL 支持
│   │   │   ├── grpc/                       ← gRPC 支持
│   │   │   └── websocket/                  ← WebSocket 支持
│   │   │
│   │   ├── database/                       ← 数据库插件
│   │   │   ├── postgres_database.go        ← PostgreSQL
│   │   │   ├── mongodb_database.go         ← MongoDB
│   │   │   └── database_plugin.go          ← 数据库接口
│   │   │
│   │   ├── cache/                          ← 缓存插件
│   │   │   ├── redis_cache.go              ← Redis 缓存
│   │   │   ├── inmemory_cache.go           ← 内存缓存
│   │   │   └── cache_plugin.go             ← 缓存接口
│   │   │
│   │   ├── mq/                             ← 消息队列插件
│   │   │   ├── kafka_mq.go                 ← Kafka 支持
│   │   │   ├── redis_mq.go                 ← Redis MQ
│   │   │   ├── zeromq_mq.go                ← ZeroMQ 支持
│   │   │   └── mq_plugin.go                ← MQ 接口
│   │   │
│   │   └── pullers/                        ← 数据拉取插件
│   │       ├── multi_chain_puller.go       ← 多链拉取
│   │       ├── https_jsonrpc_puller.go     ← HTTPS JSON-RPC
│   │       ├── websocket_jsonrpc_puller.go ← WebSocket JSON-RPC
│   │       ├── grpc_puller.go              ← gRPC 拉取
│   │       └── data_puller.go              ← 拉取接口
│   │
│   ├── services/                           ← 业务服务
│   │   ├── indexing/                       ← 索引服务
│   │   │   ├── multi_chain_indexer.go      ← 多链索引
│   │   │   ├── chain_indexer.go            ← 单链索引
│   │   │   └── ...
│   │   │
│   │   ├── query/                          ← 查询服务
│   │   │   ├── query_service.go            ← 查询实现
│   │   │   └── database_indexes.sql        ← 数据库索引
│   │   │
│   │   ├── decoder/                        ← 解码服务
│   │   │   ├── event_decoder.go            ← 事件解码
│   │   │   ├── contract_manager.go         ← 合约管理
│   │   │   └── ...
│   │   │
│   │   ├── consistency/                    ← 一致性服务
│   │   │   ├── consistency_checker.go      ← 一致性检查
│   │   │   └── ...
│   │   │
│   │   └── reorg/                          ← 重组处理
│   │       ├── reorg_handler.go            ← 重组处理
│   │       └── ...
│   │
│   ├── integrations/                       ← 区块链集成
│   │   ├── erc20/                          ← ERC20 支持
│   │   │   ├── erc20_indexer.go            ← ERC20 索引
│   │   │   └── ...
│   │   │
│   │   ├── uniswap/                        ← Uniswap 支持
│   │   │   ├── uniswap_indexer.go          ← Uniswap 索引
│   │   │   └── ...
│   │   │
│   │   └── generic/                        ← 通用支持
│   │       ├── generic_indexer.go          ← 通用索引
│   │       └── ...
│   │
│   └── observability/                      ← 可观测性
│       ├── indexer_metrics.go              ← 索引指标
│       ├── indexer_health.go               ← 索引健康
│       ├── distributed_tracing.go          ← 分布式追踪
│       └── ...
│
├── docker/                                 ← Docker 配置
│   ├── docker-compose.yml                  ← Docker Compose 配置
│   ├── Dockerfile                          ← 通用 Dockerfile
│   ├── api-gateway.Dockerfile              ← API 网关 Dockerfile
│   ├── api-service.Dockerfile              ← API 服务 Dockerfile
│   ├── event-processor.Dockerfile          ← 事件处理器 Dockerfile
│   ├── puller.Dockerfile                   ← 数据拉取器 Dockerfile
│   ├── .env.example                        ← 环境变量示例
│   ├── .dockerignore                       ← Docker 忽略文件
│   ├── Makefile                            ← Docker 构建脚本
│   └── README.md                           ← Docker 说明
│
├── k8s/                                    ← Kubernetes 配置
│   ├── namespace.yaml                      ← 命名空间
│   ├── configmap.yaml                      ← 配置映射
│   ├── chainpulse-microservice-deployment.yaml ← 微服务部署
│   ├── chainpulse-monolithic-deployment.yaml   ← 单体部署
│   ├── api-gateway-deployment.yaml         ← API 网关部署
│   ├── api-service-deployment.yaml         ← API 服务部署
│   ├── event-processor-deployment.yaml     ← 事件处理器部署
│   ├── puller-deployment.yaml              ← 数据拉取器部署
│   ├── kafka-deployment.yaml               ← Kafka 部署
│   ├── postgres-deployment.yaml            ← PostgreSQL 部署
│   ├── redis-deployment.yaml               ← Redis 部署
│   ├── hpa-api-service.yaml                ← API 服务 HPA
│   ├── hpa-event-processor.yaml            ← 事件处理器 HPA
│   └── hpa-puller.yaml                     ← 数据拉取器 HPA
│
├── docs/                                   ← 文档
│   ├── guides/                             ← 指南
│   │   ├── MICROSERVICES_IMPLEMENTATION_GUIDE.md      ← 实现指南
│   │   ├── MICROSERVICES_QUICK_REFERENCE.md           ← 快速参考
│   │   ├── MICROSERVICES_DEPLOYMENT_QUICK_CARD.md     ← 快速卡片
│   │   ├── MICROSERVICES_FILE_STRUCTURE_GUIDE.md      ← 文件结构 (本文件)
│   │   ├── DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md   ← 完整部署指南
│   │   ├── DISTRIBUTED_ARCHITECTURE_PRODUCTION_DEPLOYMENT.md ← 生产部署
│   │   ├── DISTRIBUTED_ARCHITECTURE_STAGING_DEPLOYMENT.md   ← 测试部署
│   │   ├── DISTRIBUTED_ARCHITECTURE_OPERATIONS_GUIDE.md     ← 操作指南
│   │   ├── DISTRIBUTED_ARCHITECTURE_MONITORING_ALERTING.md  ← 监控告警
│   │   ├── DISTRIBUTED_ARCHITECTURE_MIGRATION_GUIDE.md      ← 迁移指南
│   │   └── ...
│   │
│   ├── progress/                           ← 进度文档
│   │   ├── MICROSERVICES_ARCHITECTURE_ANALYSIS_ENTERPRISE_WEB3.md
│   │   ├── PHASE_4_MICROSERVICES_ARCHITECTURE_COMPLETE.md
│   │   ├── TASK_6_MICROSERVICES_ANALYSIS_COMPLETE.md
│   │   ├── JANUARY_12_2026_MICROSERVICES_SESSION_SUMMARY.md
│   │   ├── MICROSERVICES_DOCUMENTATION_INDEX.md
│   │   └── ...
│   │
│   └── architecture/                       ← 架构文档
│       ├── DIRECTORY_STRUCTURE.md
│       ├── CODE_STRUCTURE_ANALYSIS.md
│       └── ...
│
├── test/                                   ← 测试
│   ├── e2e/                                ← 端到端测试
│   │   └── e2e_test.go
│   │
│   ├── integration/                        ← 集成测试
│   │   ├── deployment_mode_integration_test.go
│   │   ├── multi_chain_integration_test.go
│   │   ├── data_puller_integration_example_test.go
│   │   └── ...
│   │
│   └── fixtures/                           ← 测试数据
│       ├── multi_chain_fixtures.go
│       └── ...
│
├── .kiro/                                  ← Kiro 配置
│   ├── specs/                              ← 规范
│   │   ├── chainpulse-distributed-architecture/
│   │   │   ├── requirements.md
│   │   │   ├── design.md
│   │   │   └── tasks.md
│   │   │
│   │   └── ...
│   │
│   └── settings/                           ← 设置
│       └── mcp.json
│
├── go.mod                                  ← Go 模块定义
├── go.sum                                  ← Go 模块校验
├── Makefile                                ← 根 Makefile
├── README.md                               ← 项目说明
├── MICROSERVICES_ARCHITECTURE_START_HERE.md ← 微服务开始指南
├── INDEX.md                                ← 索引
├── LICENSE                                 ← 许可证
└── .env.example                            ← 环境变量示例
```

---

## 🎯 关键文件说明

### 1. 微服务程序入口

| 文件 | 说明 | 端口 |
|------|------|------|
| `cmd/chainpulse-api-gateway/main.go` | API 网关入口 | 8080 |
| `cmd/chainpulse-api-service/main.go` | API 服务入口 | 8081 |
| `cmd/chainpulse-event-processor/main.go` | 事件处理器入口 | 8082 |
| `cmd/chainpulse-puller/main.go` | 数据拉取器入口 | 8083 |

### 2. 部署配置文件

| 文件 | 说明 | 用途 |
|------|------|------|
| `k8s/chainpulse-microservice-deployment.yaml` | 微服务部署 | Kubernetes 部署 |
| `docker/docker-compose.yml` | Docker Compose | 本地开发 |
| `docker/Dockerfile` | 通用 Dockerfile | Docker 镜像构建 |

### 3. 基础设施部署

| 文件 | 说明 | 服务 |
|------|------|------|
| `k8s/postgres-deployment.yaml` | PostgreSQL 部署 | 数据库 |
| `k8s/redis-deployment.yaml` | Redis 部署 | 缓存 |
| `k8s/kafka-deployment.yaml` | Kafka 部署 | 消息队列 |

### 4. 核心业务代码

| 目录 | 说明 | 功能 |
|------|------|------|
| `pkg/infrastructure/deployment/` | 部署管理 | 微服务部署 |
| `pkg/plugins/api/` | API 插件 | 多协议支持 |
| `pkg/services/indexing/` | 索引服务 | 多链索引 |
| `pkg/plugins/mq/` | 消息队列 | Kafka 支持 |

### 5. 文档

| 文件 | 说明 | 用途 |
|------|------|------|
| `MICROSERVICES_ARCHITECTURE_START_HERE.md` | 开始指南 | 快速入门 |
| `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md` | 完整部署指南 | 详细部署 |
| `docs/guides/MICROSERVICES_DEPLOYMENT_QUICK_CARD.md` | 快速卡片 | 快速查找 |
| `docs/archive/MICROSERVICES_IMPLEMENTATION_GUIDE.md` | 实现指南（归档） | 开发参考 |

---

## 🔄 文件关系图

```
cmd/ (微服务程序)
  ├── chainpulse-api-gateway/
  │   └── main.go → pkg/infrastructure/deployment/api_gateway_deployment.go
  │
  ├── chainpulse-api-service/
  │   └── main.go → pkg/plugins/api/gateway.go
  │
  ├── chainpulse-event-processor/
  │   └── main.go → pkg/plugins/mq/kafka_mq.go
  │
  └── chainpulse-puller/
      └── main.go → pkg/plugins/pullers/multi_chain_puller.go

docker/ (Docker 配置)
  ├── docker-compose.yml → 启动所有服务
  └── Dockerfile → 构建镜像

k8s/ (Kubernetes 配置)
  ├── chainpulse-microservice-deployment.yaml → 部署所有微服务
  ├── postgres-deployment.yaml → 部署数据库
  ├── redis-deployment.yaml → 部署缓存
  └── kafka-deployment.yaml → 部署消息队列

docs/ (文档)
  ├── guides/
  │   ├── DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md → 完整指南
  │   ├── MICROSERVICES_DEPLOYMENT_QUICK_CARD.md → 快速卡片
  │   └── MICROSERVICES_IMPLEMENTATION_GUIDE.md → 实现指南
  │
  └── progress/
      ├── MICROSERVICES_ARCHITECTURE_ANALYSIS_ENTERPRISE_WEB3.md
      └── PHASE_4_MICROSERVICES_ARCHITECTURE_COMPLETE.md
```

---

## 📊 部署流程

```
1. 选择部署方式
   ├── Kubernetes (生产)
   │   └── 使用 k8s/ 目录下的 YAML 文件
   │
   ├── Docker Compose (开发)
   │   └── 使用 docker/docker-compose.yml
   │
   └── 手动部署 (调试)
       └── 直接运行 cmd/ 下的程序

2. 配置环境
   ├── 设置环境变量
   ├── 配置数据库连接
   ├── 配置缓存连接
   └── 配置消息队列连接

3. 启动基础设施
   ├── PostgreSQL
   ├── Redis
   ├── Kafka
   └── Consul (可选)

4. 启动微服务
   ├── API 网关
   ├── API 服务
   ├── 事件处理器
   └── 数据拉取器

5. 验证部署
   ├── 检查 Pod 状态
   ├── 检查服务连接
   ├── 测试 API 端点
   └── 查看日志和指标
```

---

## 🔍 快速查找

### 我想找...

**API 网关程序**
→ `cmd/chainpulse-api-gateway/main.go`

**API 服务程序**
→ `cmd/chainpulse-api-service/main.go`

**事件处理器程序**
→ `cmd/chainpulse-event-processor/main.go`

**数据拉取器程序**
→ `cmd/chainpulse-puller/main.go`

**Kubernetes 部署文件**
→ `k8s/chainpulse-microservice-deployment.yaml`

**Docker Compose 配置**
→ `docker/docker-compose.yml`

**部署指南**
→ `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md`

**快速参考**
→ `docs/guides/MICROSERVICES_DEPLOYMENT_QUICK_CARD.md`

**实现指南**
→ `docs/archive/MICROSERVICES_IMPLEMENTATION_GUIDE.md`

**架构分析**
→ `docs/progress/MICROSERVICES_ARCHITECTURE_ANALYSIS_ENTERPRISE_WEB3.md`

---

## 📝 文件大小统计

| 类别 | 文件数 | 总大小 |
|------|--------|--------|
| 微服务程序 | 4 | ~2KB |
| Docker 配置 | 6 | ~5KB |
| Kubernetes 配置 | 12 | ~15KB |
| 核心代码 | 50+ | ~500KB |
| 文档 | 20+ | ~200KB |
| 测试 | 30+ | ~100KB |

---

## ✅ 检查清单

- [ ] 了解 4 个微服务程序的位置
- [ ] 了解 3 种部署方式
- [ ] 了解环境变量配置
- [ ] 了解 Kubernetes 部署文件
- [ ] 了解 Docker Compose 配置
- [ ] 了解基础设施部署
- [ ] 了解服务发现机制
- [ ] 了解监控和日志
- [ ] 了解扩展和优化
- [ ] 了解故障排查

---

## 📚 相关文档

- **完整部署指南**: `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md`
- **快速参考卡片**: `docs/guides/MICROSERVICES_DEPLOYMENT_QUICK_CARD.md`
- **实现指南**: `docs/archive/MICROSERVICES_IMPLEMENTATION_GUIDE.md`
- **快速参考**: `docs/guides/MICROSERVICES_QUICK_REFERENCE.md`
- **架构分析**: `docs/progress/MICROSERVICES_ARCHITECTURE_ANALYSIS_ENTERPRISE_WEB3.md`
- **开始指南**: `MICROSERVICES_ARCHITECTURE_START_HERE.md`

---

**Status**: ✅ COMPLETE  
**Last Updated**: January 12, 2026  
**Version**: 1.0
