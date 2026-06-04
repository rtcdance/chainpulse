# Docker 部署

## 前提条件

- **Docker Desktop** 已启动（macOS: 菜单栏有 Docker 图标即正常）
- 至少 **8GB 可用内存**（推荐 16GB，因会启动 7 条链 + 5 个中间件 + 应用）
- 端口 **8080、3000、5432、6379、8545-8551 等**未被占用（首次运行脚本会自动检测冲突）

## 快速选择

| 场景 | 推荐模式 | 命令 |
|------|---------|------|
| 第一次体验 / 玩所有功能 | 单体 | `bash docker/deploy-monolith.sh` |
| 展示微服务架构 / 带 Solana | 微服务 | `bash docker/deploy-microservices.sh` |

---

## 单体模式（推荐入门）

```bash
bash docker/deploy-monolith.sh
```

脚本会：构建镜像 → 启动全部容器 → 部署合约 → 开始事件模拟 → 等待 30 秒 → 自动验证所有功能是否正常。

### 部署后体验

```
H5 前端:   http://localhost:3000
REST API:  http://localhost:8080
```

### 服务列表

| 服务名 | 端口 | 说明 |
|--------|------|------|
| `chainpulse-app` | 8080, 50051 | 单体应用（REST + WebSocket + gRPC + Admin API） |
| `chainpulse-frontend` | 3000 | H5 前端 |
| `chainpulse-anvil-ethereum` | 8545 | Ethereum Anvil |
| `chainpulse-anvil-polygon` | 8547 | Polygon Anvil |
| `chainpulse-anvil-bsc` | 8546 | BSC Anvil |
| `chainpulse-anvil-arbitrum` | 8548 | Arbitrum Anvil |
| `chainpulse-anvil-optimism` | 8549 | Optimism Anvil |
| `chainpulse-anvil-base` | 8550 | Base Anvil |
| `chainpulse-anvil-avalanche` | 8551 | Avalanche Anvil |
| `chainpulse-postgres` | 5432 | PostgreSQL |
| `chainpulse-redis` | 6379 | Redis |
| `chainpulse-mongodb` | 27017 | MongoDB |
| `chainpulse-kafka` | 9092 | Kafka |
| `chainpulse-prometheus` | 9090 | Prometheus |
| `chainpulse-grafana` | 3001 | Grafana |

### H5 前端能干的事

打开 http://localhost:3000 后：

| 页面 | 功能 |
|------|------|
| Dashboard | 查看实时事件统计、链状态、最近事件列表 |
| Events | 按链/合约/事件名/区块范围过滤事件，查看详情 |
| WebSocket | 连接 `/ws` 或 `/events/subscribe`，设置过滤器，实时接收事件推送 |
| Admin | 服务健康探测矩阵、运行时摘要、事件名称分布 |
| GraphQL | 预设查询模板，交互式 GraphQL 探索器 |

### 验证输出示例

一键部署后控制台会自动输出验证结果：

```
[1/8] WebSocket upgrade test...
  OK  /ws → 101 Switching Protocols
  OK  /events/subscribe → 101
[2/8] SIWE auth test...
  OK  nonce=5d9836c8f733d210d9c65c2ebb9743d0
[3/8] Event name resolution test...
  OK  All event names are human-readable
[4/8] Admin API key CRUD test...
  OK  Admin API key created
[5/8] Webhook endpoint test...
  OK  Webhook created
[6/8] Rate limiter stress test...
  OK  Rate limiter triggered after 18 requests
[7/8] Reorg handler check...
  OK  Reorg handler wired
[8/8] Solana check...
  SKIP  Solana not deployed (use microservices mode for Solana)
```

---

## 微服务模式

```bash
bash docker/deploy-microservices.sh
```

### 部署后体验

```
H5 前端:   http://localhost:13000
API 网关:  http://localhost:8080  （微服务入口）
```

### 服务列表

| 服务名 | 端口 | 说明 |
|--------|------|------|
| `chainpulse-api-gateway` | 8080 | API 网关（所有请求的入口） |
| `chainpulse-api-service` | 8081 | 事件查询 / 统计（内网服务） |
| `chainpulse-event-processor` | 8082 | Kafka 消费 + 写入数据库 |
| `chainpulse-puller` | 8083 | 链事件主动拉取 + 发布到 Kafka |
| `chainpulse-ms-frontend` | 13000 | H5 前端 |
| `chainpulse-anvil` | 8545 | Anvil（6 条 EVM 链共用一个节点，通过 chain-id 切换） |
| `chainpulse-solana-validator` | 8899, 8900 | Solana 验证节点 |
| `chainpulse-postgres` | 5432 | PostgreSQL |
| `chainpulse-redis` | 6379 | Redis |
| `chainpulse-kafka` | 9092 | Kafka |
| `chainpulse-prometheus` | 9090 | Prometheus |
| `chainpulse-grafana` | 3000 | Grafana |

### 微服务 vs 单体差异

微服务模式下，4 个服务独立运行，但 **Admin API Key / Webhook / Export** 路由仅在单体模式可用（这些功能不在 API 网关暴露）。

---

## 调度入口

```bash
bash docker/deploy-and-simulate.sh monolith       # 单体模式
bash docker/deploy-and-simulate.sh microservices  # 微服务模式
bash docker/deploy-and-simulate.sh status         # 查看状态
bash docker/deploy-and-simulate.sh stop           # 停止全部
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

## 架构对比

```
单体模式                        微服务模式
┌──────────────┐               ┌──────────────┐  8080
│ chainpulse   │ 8080          │ api-gateway  │──────▶ H5
│ (全部功能)    │──────▶ H5     │ (路由/鉴权)  │
│              │               └──────┬───────┘
│ REST / WS    │                      │
│ Admin / WH   │              ┌───────┴────────┐  8081
│ Export / DLQ │              │ api-service     │
│ Reorg 检测   │              │ 事件查询/统计    │
└──────────────┘              └───────┬────────┘
                                      │
                              ┌───────┴────────┐  8082
                              │ event-processor│
                              │ Kafka 消费      │
                              └───────┬────────┘
                                      │
                              ┌───────┴────────┐  8083
                              │ puller          │
                              │ 链事件拉取       │
                              └────────────────┘
```

## 常见问题

**Q: `port is already allocated` 错误**
A: 运行 `lsof -i :PORT` 查看占用进程，关闭冲突服务或修改 compose 端口映射。

**Q: Docker Desktop 不在运行**
A: 确保 Docker Desktop 已启动（菜单栏有图标），首次启动可能需要等 30 秒。

**Q: 需要跑多久才能看到事件？**
A: 一键部署后约 30 秒事件开始产生，控制台会显示 `Events indexed by ChainPulse: N`。

**Q: 跑完怎么关？**
A: `bash docker/deploy-and-simulate.sh stop` 会停止所有容器，加 `-v` 会删除数据卷。

## 部署脚本

| 脚本 | 用途 |
|------|------|
| `deploy-monolith.sh` | 单体构建 → 部署 → 模拟 → 验证 |
| `deploy-microservices.sh` | 微服务构建 → 部署 → 模拟 → 验证 |
| `deploy-and-simulate.sh` | 调度入口 |
