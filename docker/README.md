# ChainPulse 一键部署指南

> 从零开始，5 分钟内看到 7 条 EVM 链 + Solana 的实时区块链事件。

---

## 第一步：安装前提

你只需要装两样东西：

| 工具 | 最低版本 | 检查命令 | 安装方式 |
|------|---------|---------|---------|
| **Go** | 1.24+ | `go version` | [go.dev/dl](https://go.dev/dl/) |
| **Docker Desktop** | 最新 | `docker version` | [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop) |

> macOS 用户：Docker Desktop 安装后，菜单栏出现鲸鱼图标即表示运行中。
> 首次启动 Docker Desktop 需等待约 30 秒。

确认就绪：

```bash
go version      # 应输出 go1.24.x 或更高
docker version  # 应输出 Client + Server 信息
```

**硬件要求**：至少 8GB 可用内存（推荐 16GB，因为会启动 20+ 容器）。

---

## 第二步：选模式，一条命令启动

在项目根目录执行：

| 场景 | 命令 | 说明 |
|------|------|------|
| **第一次体验** | `bash docker/deploy-monolith.sh` | 单体模式，1 个应用容器，简单直接 |
| **体验微服务架构** | `bash docker/deploy-microservices.sh` | 4 个服务容器 + Solana，展示完整架构 |
| **统一入口** | `bash docker/deploy-and-simulate.sh monolith` | 自动调度到上面两个脚本 |

> 两种模式都支持 7 条 EVM 链 + Solana + H5 前端 + 事件模拟。区别在于应用容器数量。

脚本会自动完成：**编译 Go → 构建 Docker 镜像 → 启动 20+ 容器 → 部署智能合约 → 开始模拟事件 → 验证功能**。

整个过程约 3-5 分钟（取决于网速和机器性能），请耐心等待。

---

## 第三步：打开 H5 前端

| 模式 | 前端地址 |
|------|---------|
| 单体 | http://localhost:13000 |
| 微服务 | http://localhost:13000 |

打开后你会看到：

| 页面 | 能做什么 |
|------|---------|
| **Dashboard** | 实时事件统计、7+1 条链的状态、最近事件列表 |
| **Events** | 按链/事件名过滤，查看 Transfer、Swap、Borrow 等事件详情 |
| **Admin** | 服务健康探测矩阵、运行时摘要、事件名称分布图 |
| **WebSocket** | 连接实时事件推送，设置过滤器 |

> 首次打开时如果看到 "API Error"，请等待 30 秒后刷新（后端正在索引事件）。

---

## 你会看到什么

部署完成后，控制台输出类似：

```
╔══════════════════════════════════════════════════════════════╗
║    ChainPulse Monolith  —  One-Click Deploy & Simulate      ║
╠══════════════════════════════════════════════════════════════╣
║  Mode:       MONOLITHIC (single binary)                     ║
║  Chains:     7 EVM + Solana                                 ║
║  Burst: ON (15-50 TPS)   Reorg: depth 2-12                 ║
║                                                              ║
║  API:            http://localhost:8080                       ║
║  Frontend UI:    http://localhost:13000                      ║
║  Grafana:        http://localhost:3001                       ║
╚══════════════════════════════════════════════════════════════╝
```

模拟器持续产生的事件类型：

| 事件 | 来源 | 说明 |
|------|------|------|
| Transfer / Approval | ERC-20 Token | 代币转账和授权 |
| Swap | Uniswap V2/V3 | DEX 交易 |
| Supply / Borrow / Withdraw / Repay | Aave V3 | 借贷协议 |
| LiquidationCall | Aave V3 | 清算事件 |
| Supply / Withdraw / Borrow | Compound V3 | 借贷协议 |
| VoteCast / ProposalCreated | Governance | 治理投票 |
| Bridge | 跨链桥 | L1↔L2 跨链 |
| Transfer (ERC-721) | NFT | NFT 转移 |
| TransferSingle (ERC-1155) | ERC-1155 | 多代币标准 |
| SPL Transfer / Mint / Burn | Solana | Solana 代币操作 |

还有特殊场景模拟：
- **Burst**: 15-50 TPS 突发流量
- **Reorg**: 2-12 区块深度的链重组
- **Causal Chain**: Supply → Borrow → Liquidation 因果链
- **Timestamp Anomaly**: 时间戳异常
- **Duplicate**: 重复事件检测

还有线上异常场景模拟：
- **Gas Spike** (8%): Gas 价格突然飙升到 500 gwei
- **Dropped Transaction** (5%): Gas 不足导致交易失败/丢失
- **Stale Block** (5%): 查询历史旧区块数据
- **MEV Sandwich** (7%): 前置交易 + 受害者交易 + 后置交易的夹击模式
- **Contract Revert** (8%): 合约 require() 失败（如未授权的 transferFrom）
- **Nonce Gap** (4%): 跳跃 nonce 导致交易顺序错乱
- **Large Block** (5%): 单区块打包 10-30 笔交易

还有高优先级线上场景模拟：
- **Cross-chain Bridge** (6%): 链 A 锁定 → 链 B 铸造，模拟跨链桥联动
- **Flash Loan** (5%): 借款 → Swap → 还款，模拟闪电贷套利
- **Liquidation Cascade** (4%): 价格下跌 → 连环清算 2-5 个账户
- **DEX Aggregator** (6%): 3 跳路由 TokenA→WETH→USDC→TokenB
- **Solana Vote/Stake** (10%): 验证者投票和质押委托

还有中优先级线上场景模拟：
- **Cross-chain Arbitrage** (3%): 同一交易对在两条链上价差套利
- **Solana Slot Skip** (4%): 验证者跳过 slot，产生区块间隔
- **Solana Compute Limit** (3%): 计算预算超限导致交易失败
- **Causal Block** (7%): 同区块内 Transfer→Swap→Transfer 因果序

---

## API 快速体验

```bash
# 健康检查
curl http://localhost:8080/health

# 查看事件统计
curl http://localhost:8080/events/stats

# 查询最近 10 条事件
curl http://localhost:8080/events?limit=10

# 只看 Solana 事件
curl http://localhost:8080/events?network=solana&limit=5

# 只看以太坊 Transfer 事件
curl http://localhost:8080/events?chainId=1&eventName=Transfer&limit=5

# WebSocket 实时监听
wscat -c ws://localhost:8080/ws

# 运行时摘要
curl http://localhost:8080/runtime/summary

# Prometheus 指标
curl http://localhost:8080/metrics
```

---

## 可观测性

| 工具 | 地址 | 用途 |
|------|------|------|
| **Grafana** | http://localhost:3001 | 仪表盘（admin / 你设置的密码） |
| **Prometheus** | http://localhost:9090 | 指标查询 |
| **Jaeger** | http://localhost:16686 | 分布式追踪 |

---

## 停止和清理

```bash
# 停止所有容器（保留数据）
bash docker/deploy-and-simulate.sh stop

# 或者直接用 docker compose
cd docker
docker compose -f docker-compose.monolith.yml down       # 单体
docker compose -f docker-compose.microservices.yml down   # 微服务

# 彻底清理（删除数据卷）
docker compose -f docker-compose.monolith.yml down -v
```

---

## 两种模式对比

### 架构图

```
单体模式                              微服务模式
┌──────────────────┐                 ┌──────────────────┐
│  chainpulse-app  │ 8080            │  api-gateway     │ 8080
│  ┌────────────┐  │                 │  (路由/鉴权)      │
│  │ Puller     │  │                 └────────┬─────────┘
│  │ Processor  │  │──────▶ H5               │
│  │ API        │  │ 13000          ┌─────────┴─────────┐
│  │ Admin/WH   │  │                │  api-service      │ 8081
│  │ Reorg/DLQ  │  │                │  (事件查询/统计)   │
│  └────────────┘  │                └─────────┬─────────┘
└──────────────────┘                         │
                                   ┌─────────┴─────────┐
                                   │  event-processor  │ 8082
                                   │  (Kafka 消费)      │
                                   └─────────┬─────────┘
                                             │
                                   ┌─────────┴─────────┐
                                   │  puller           │ 8083
                                   │  (链事件拉取)      │
                                   └───────────────────┘
```

### 功能对比

| 特性 | 单体 | 微服务 |
|------|:----:|:------:|
| REST API | ✅ | ✅ |
| WebSocket 实时监听 | ✅ | ✅ |
| 事件名解析（人类可读） | ✅ | ✅ |
| 7 条 EVM 链模拟 | ✅ | ✅ |
| Solana 模拟 | ✅ | ✅ |
| H5 前端 | ✅ | ✅ |
| Prometheus + Grafana | ✅ | ✅ |
| Jaeger 追踪 | ✅ | ✅ |
| Admin API Key CRUD | ✅ | ❌ |
| Webhook 注册 | ✅ | ❌ |
| Export 导出 | ✅ | ❌ |
| DLQ 回放 | ✅ | ❌ |
| 水平扩展 | ❌ | ✅ |
| 独立扩缩容 | ❌ | ✅ |

### 容器数量

| | 单体 | 微服务 |
|---|:----:|:------:|
| 应用容器 | 1 | 4 |
| 基础设施 | 5 (PG + Redis + MongoDB + Kafka + ZK) | 5 |
| 区块链节点 | 8 (7 Anvil + Solana) | 8 (7 Anvil + Solana) |
| 前端 | 1 | 1 |
| 可观测性 | 3 (Prometheus + Grafana + Jaeger) | 3 |
| **总计** | **18** | **21** |

---

## 常见问题

### 部署相关

**Q: `port is already allocated` 错误**
```bash
# 查看哪个进程占用了端口
lsof -i :8080
lsof -i :13000
# 关闭冲突进程，或修改 docker-compose 端口映射
```

**Q: Docker Desktop 不在运行**
确保菜单栏有 Docker 鲸鱼图标。首次启动需等 30 秒。

**Q: Kafka 容器一直 restarting**
Kafka 启动较慢，等待 60-90 秒。如果持续失败，尝试完全重启：
```bash
bash docker/deploy-and-simulate.sh stop
bash docker/deploy-monolith.sh   # 或 deploy-microservices.sh
```

**Q: 编译报错 `go: command not found`**
安装 Go 1.24+：https://go.dev/dl/

**Q: 前端页面显示 API Error**
后端可能还在启动中，等待 30 秒后 Cmd+Shift+R 硬刷新浏览器。

### 事件相关

**Q: 需要跑多久才能看到事件？**
一键部署后约 30 秒事件开始产生。控制台会显示 `Events indexed by ChainPulse: N`。

**Q: 为什么有些事件名是 `0x133f...` 这样的哈希？**
事件签名解析需要匹配 ABI 注册表。如果看到未解析的哈希，说明该事件签名尚未注册。

**Q: Solana 事件名显示 `Vote1111111...` 这样的 Program ID？**
这是 Solana 原生程序的地址格式，系统会自动映射为 "Vote Program" 等可读名称。

### 清理相关

**Q: 跑完怎么关？**
```bash
bash docker/deploy-and-simulate.sh stop
```

**Q: 如何彻底清理包括数据？**
```bash
cd docker
docker compose -f docker-compose.monolith.yml down -v      # 单体
docker compose -f docker-compose.microservices.yml down -v  # 微服务
```

**Q: 如何查看模拟器状态？**
```bash
bash docker/deploy-and-simulate.sh status
```

---

## 完整服务列表

### 单体模式

| 服务 | 端口 | 说明 |
|------|------|------|
| chainpulse-app | 8080 | 单体应用（REST + WebSocket + gRPC + Admin） |
| chainpulse-mono-frontend | 12000 | H5 前端 |
| chainpulse-anvil-ethereum | 8545 | Ethereum 本地链 |
| chainpulse-anvil-polygon | 8546 | Polygon 本地链 |
| chainpulse-anvil-bsc | 8547 | BSC 本地链 |
| chainpulse-anvil-arbitrum | 8548 | Arbitrum 本地链 |
| chainpulse-anvil-optimism | 8549 | Optimism 本地链 |
| chainpulse-anvil-base | 8550 | Base 本地链 |
| chainpulse-anvil-avalanche | 8551 | Avalanche 本地链 |
| chainpulse-solana | 8899 | Solana 测试验证器 |
| chainpulse-postgres | 5432 | PostgreSQL |
| chainpulse-redis | 6379 | Redis |
| chainpulse-mongodb | 27017 | MongoDB |
| chainpulse-kafka | 9092 | Kafka |
| chainpulse-zookeeper | 2181 | Zookeeper |
| chainpulse-prometheus | 9090 | Prometheus |
| chainpulse-grafana | 3001 | Grafana |
| chainpulse-jaeger | 16686 | Jaeger |

### 微服务模式

| 服务 | 端口 | 说明 |
|------|------|------|
| chainpulse-api-gateway | 8080 | API 网关（统一入口） |
| chainpulse-api-service | 8081 | 事件查询/统计 |
| chainpulse-event-processor | 8082 | Kafka 消费 + 写入 |
| chainpulse-puller | 8083 | 链事件拉取 + 发布 |
| chainpulse-ms-frontend | 13000 | H5 前端 |
| 其余同上 | — | 区块链节点 + 基础设施 + 可观测性 |

---

## 部署脚本一览

| 脚本 | 用途 |
|------|------|
| `deploy-monolith.sh` | 单体：编译 → 镜像 → 部署 → 合约 → 模拟 → 验证 |
| `deploy-microservices.sh` | 微服务：编译 → 镜像 → 部署 → 合约 → 模拟 → 验证 |
| `deploy-and-simulate.sh` | 统一调度入口，支持 `monolith` / `microservices` / `stop` / `status` |

---

## 下一步

- 阅读 [API 文档](../docs/guides/API_DOCUMENTATION.md) 了解完整 API
- 阅读 [部署指南](../docs/guides/DEPLOYMENT_GUIDE.md) 了解生产环境部署
- 运行 `make playground` 体验零依赖内存模式
