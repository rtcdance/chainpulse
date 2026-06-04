# Docker 部署

## 一键部署

支持两种模式：**单体（monolith）** 包含所有路由，**微服务（microservices）** 4 个独立服务。

### 单体模式

```bash
bash docker/deploy-monolith.sh
```

部署后访问：
- H5 前端：[http://localhost:3000](http://localhost:3000)
- REST API：[http://localhost:8080](http://localhost:8080)

| 服务名 | 端口 | 说明 |
|--------|------|------|
| `chainpulse-app` | 8080, 50051 | 单体应用（REST + WS + gRPC） |
| `chainpulse-frontend` | 3000 | H5 前端 |
| `chainpulse-anvil-ethereum` | 8545 | Anvil Ethereum 链 |
| `chainpulse-anvil-polygon` | 8547 | Anvil Polygon 链 |
| `chainpulse-anvil-bsc` | 8546 | Anvil BSC 链 |
| `chainpulse-anvil-optimism` | 8549 | Anvil Optimism 链 |
| `chainpulse-anvil-arbitrum` | 8548 | Anvil Arbitrum 链 |
| `chainpulse-anvil-base` | 8550 | Anvil Base 链 |
| `chainpulse-anvil-avalanche` | 8551 | Anvil Avalanche 链 |
| `chainpulse-postgres` | 5432 | PostgreSQL |
| `chainpulse-redis` | 6379 | Redis |
| `chainpulse-mongodb` | 27017 | MongoDB |
| `chainpulse-kafka` | 9092 | Kafka |
| `chainpulse-prometheus` | 9090 | Prometheus |
| `chainpulse-grafana` | 3001 | Grafana |

### 微服务模式

```bash
bash docker/deploy-microservices.sh
```

部署后访问：
- H5 前端：[http://localhost:13000](http://localhost:13000)
- API 网关：[http://localhost:8080](http://localhost:8080)

| 服务名 | 端口 | 说明 |
|--------|------|------|
| `chainpulse-api-gateway` | 8080 | API 网关（入口） |
| `chainpulse-api-service` | 8081 | 事件查询 / 统计 |
| `chainpulse-event-processor` | 8082 | Kafka 消费 + 存储 |
| `chainpulse-puller` | 8083 | 链事件主动拉取 |
| `chainpulse-ms-frontend` | 13000 | H5 前端 |
| `chainpulse-anvil` | 8545 | Anvil（多链合用一个节点） |
| `chainpulse-solana-validator` | 8899, 8900 | Solana 验证节点 |
| `chainpulse-postgres` | 5432 | PostgreSQL |
| `chainpulse-redis` | 6379 | Redis |
| `chainpulse-kafka` | 9092 | Kafka |
| `chainpulse-prometheus` | 9090 | Prometheus |
| `chainpulse-grafana` | 3000 | Grafana |

### 调度入口

```bash
bash docker/deploy-and-simulate.sh monolith       # 单体模式
bash docker/deploy-and-simulate.sh microservices  # 微服务模式（默认）
bash docker/deploy-and-simulate.sh status         # 查看状态
bash docker/deploy-and-simulate.sh stop           # 停止
```

## 能力对比

| 特性 | 单体 | 微服务 |
|------|:----:|:------:|
| REST API | ✅ | ✅ |
| WebSocket 实时监听 | ✅ | ✅ |
| SIWE 认证 | ✅ | ✅ |
| 事件名解析 | ✅ | ✅ |
| Rate Limiter | ✅ | ✅ |
| Admin API Key CRUD | ✅ | ❌ |
| Webhook 注册 | ✅ | ❌ |
| Export 导出 | ✅ | ❌ |
| Solana 拉取 | ❌ | ✅ |
| 可伸缩部署 | ❌ | ✅ |

## 架构

```
单体模式                        微服务模式
┌──────────────┐               ┌──────────────┐  8080
│ chainpulse   │ 8080          │ api-gateway  │──────▶ H5
│ (所有功能)    │──────▶ H5     │ (路由/鉴权)  │
│              │               └──────┬───────┘
│ REST API     │                      │
│ WS           │              ┌───────┴────────┐  8081
│ Admin Key    │              │ api-service     │
│ Webhook      │              │ 事件查询/统计    │
│ Export       │              └───────┬────────┘
│ Reorg        │                      │
└──────────────┘              ┌───────┴────────┐  8082
                              │ event-processor│
                              │ Kafka 消费      │
                              └───────┬────────┘
                                      │
                              ┌───────┴────────┐  8083
                              │ puller          │
                              │ 链事件拉取       │
                              └────────────────┘
```

## 部署脚本

| 脚本 | 用途 |
|------|------|
| `deploy-monolith.sh` | 单体构建 + 部署 + 模拟 + 验证 |
| `deploy-microservices.sh` | 微服务构建 + 部署 + 模拟 + 验证 |
| `deploy-and-simulate.sh` | 调度入口 |
| `acceptance.sh` | 单体镜像构建 |
