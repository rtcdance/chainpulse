# M1-1c: 修复单体可观测性 + API Gateway（严格按 ARCHITECTURE_v1.md 蓝图）

> 这是 M1-1 的第三阶段。**前提: M1-1a + M1-1b 已完成且验证通过。**
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M1-1c - 修复单体可观测性 + API Gateway

### 背景
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- M1-1a 状态: 基础数据链路已通
- M1-1b 状态: 容错层已完成

### 蓝图对可观测性和 API Gateway 的要求

| 蓝图章节 | 蓝图要求 |
|---|---|
| §3.4 API Gateway | GraphQL only + 内存令牌桶限流（API Key 1000/min, IP 100/min）+ mock 认证 + WebSocket 连接池（上限 10000） |
| §3.1 健康端点 | `GET /health/puller` 返回各链拉取状态 |
| §5 Platform 层 | 统一标签注入（chain_id, service, operation, block_height）+ 指标定义 + 分布式追踪 |
| §3.3 缓存击穿防护 | `cache_warmer.go` 预热 + `cache_middleware.go` 单机锁 |

### 当前状态：7 个断裂点

#### 断裂 16: RPC 故障切换缺失
- 蓝图要求: §3.1 — 多节点池，失败自动切换到备用节点
- 当前: 每条链只用单个 RPC 端点
- 修复: 为每条链配置 2+ 个 RPC 端点，失败时自动切换

#### 断裂 17: API Gateway 限流/认证缺失
- 蓝图要求: §3.4 — 内存令牌桶，API Key 1000/min，IP 100/min；mock 认证
- 当前: main.go 没有接入限流和认证
- 修复: 接入 `rate_limiter.go` 和 `auth_middleware.go`（mock 认证）

#### 断裂 18: WebSocket 连接池上限缺失
- 蓝图要求: §3.4 — 单 Pod 上限 10000
- 当前: 需要确认连接池有 maxConns 限制
- 修复: 确认或添加 WebSocket 连接池上限配置

#### 断裂 19: 统一标签注入缺失
- 蓝图要求: §5 — 所有指标、日志、trace 携带 chain_id, service, operation, block_height
- 当前: 指标和日志没有统一携带这些标签
- 修复: 在 metrics/log 调用中统一注入这四个标签

#### 断裂 20: Puller 健康端点缺失
- 蓝图要求: §3.1 — `GET /health/puller` 返回各链拉取状态
- 当前: 没有 Puller 健康端点
- 修复: 注册 `/health/puller` 端点，返回各链的 lastIndexedBlock、blockLag、rpcErrors

#### 断裂 21: 缓存击穿防护缺失
- 蓝图要求: §3.3 — `cache_warmer.go` 预热 + `cache_middleware.go` 单机锁
- 当前: 两个文件不存在
- 修复: 创建 `cache_warmer.go` 和 `cache_middleware.go`

#### 断裂 22: 分布式追踪缺失
- 蓝图要求: §5.3 — OTel Tracer，span 携带 chain_id, from_block, to_block
- 当前: 没有 OTel Tracing
- 修复: 在 Puller/Indexer/Query 的关键操作中注入 OTel span

### 完整数据流（修复后）

```
API Gateway (蓝图 §3.4)
  → 认证: auth_middleware.MockAuth()
  → 限流: rate_limiter.TokenBucket(API Key 1000/min, IP 100/min)
  → WebSocket 连接池: maxConns=10000
  → GraphQL 路由

Health Endpoints (蓝图 §3.1)
  → GET /health/puller → 返回各链拉取状态
  → GET /health → 总体健康状态

Observability (蓝图 §5)
  → 所有指标/日志/trace 携带: chain_id, service, operation, block_height
  → OTel Tracer span: pull_events(chain_id, from_block, to_block), process_batch, query_events

QueryService (蓝图 §3.3)
  → 缓存击穿防护: cache_middleware.GetOrLock() + cache_warmer.Warm()
```

### 目标
修复上述 7 个断裂，使单体模式具备蓝图要求的完整可观测性和 API Gateway 能力。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性
- [ ] **RPC 故障切换**（蓝图 §3.1: 多节点池，失败自动切换）
- [ ] **API Gateway 限流**（蓝图 §3.4: 内存令牌桶，API Key 1000/min, IP 100/min）
- [ ] **API Gateway mock 认证**（蓝图 §3.4）
- [ ] **WebSocket 连接池上限**（蓝图 §3.4: 10000）
- [ ] **统一标签注入**（蓝图 §5: chain_id, service, operation, block_height）
- [ ] **Puller 健康端点**（蓝图 §3.1: GET /health/puller）
- [ ] **缓存击穿防护**（蓝图 §3.3: cache_warmer + cache_middleware）
- [ ] **分布式追踪**（蓝图 §5.3: OTel Tracer span）

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，§3.1 + §3.3 + §3.4 + §5**
- `cmd/monolithic/chainpulse/main.go` — Composition Root（M1-1a/b 已修改）
- `pkg/plugins/api/rate_limiter.go` — 内存令牌桶限流
- `pkg/plugins/api/auth_middleware.go` — mock 认证中间件
- `pkg/plugins/api/websocket_subscription.go` — WebSocket 连接池
- `pkg/observability/metrics.go` — Prometheus 指标定义
- `pkg/observability/tracer.go` — OTel Tracer
- `pkg/services/query/cache_service.go` — 缓存服务

### 修复步骤

**Step 1: API Gateway 限流 + 认证 + WebSocket 连接池**
```
文件: cmd/monolithic/chainpulse/main.go
rateLimiter := api.NewRateLimiter(api.RateLimitConfig{
    APIKeyRate: 1000,  // 1000 req/min per API Key
    IPRate:     100,   // 100 req/min per IP
})
authMiddleware := api.NewMockAuth()
gateway.SetRateLimiter(rateLimiter)
gateway.SetAuthMiddleware(authMiddleware)
websocketPool := api.NewWebSocketPool(10000)
gateway.SetWebSocketPool(websocketPool)
```

**Step 2: Puller 健康端点**
```
注册 GET /health/puller 端点，返回各链的:
  - lastIndexedBlock
  - blockLag (chainHead - lastIndexedBlock)
  - rpcErrors (错误计数)
  - isRunning
```

**Step 3: 统一标签注入**
```
在 metrics/log 调用中统一注入:
  metrics.WithLabels("chain_id", chainID, "service", "puller", "operation", "pull_events")
  logger.Info("pulled events", "chain_id", chainID, "service", "puller", "operation", "pull_events", "block_height", toBlock)
```

**Step 4: 缓存击穿防护**
```
新建 pkg/services/query/cache_warmer.go:
  - 定时预热最近 N 个块的事件到缓存

新建 pkg/services/query/cache_middleware.go:
  - 使用 singleflight 防止缓存击穿
  - 缓存未命中时只允许一个请求查 DB，其他等待
```

**Step 5: 分布式追踪**
```
在关键操作中注入 OTel span:
  - Puller: span = tracer.Start(ctx, "pull_events", chain_id, from_block, to_block)
  - Indexer: span = tracer.Start(ctx, "process_batch", chain_id, event_count)
  - Query: span = tracer.Start(ctx, "query_events", chain_id, query_type)
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反
- **必须与 ARCHITECTURE_v1.md 蓝图一致**

### 验证步骤
```bash
make build
make test-unit
make vet
make run-monolithic &
sleep 60
# 验证限流
curl -s http://localhost:8080/graphql -H "X-API-Key: test" -d '{"query": "{ events(limit: 1) { id } }"}'
# 验证健康端点
curl -s http://localhost:8080/health/puller
curl -s http://localhost:8080/health
# 验证指标标签
curl -s http://localhost:8080/metrics | grep chainpulse_ | head -10
```
