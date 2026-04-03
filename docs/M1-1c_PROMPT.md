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
| §3.3 Query 容错 | **熔断** + **缓存** + **降级** + **一致性检查** + 缓存击穿防护 |
| §3.4 API Gateway | GraphQL only + 内存令牌桶限流（API Key 1000/min, IP 100/min）+ mock 认证 + WebSocket 连接池（上限 10000） |
| §3.1 健康端点 | `GET /health/puller` 返回各链拉取状态 |
| §5 Platform 层 | 统一标签注入（chain_id, service, operation, block_height）+ 指标定义 + 分布式追踪 |

### 当前状态：11 个断裂点

#### 断裂 16: Query 熔断缺失
- 蓝图要求: §3.3 — `circuit_breaker.go`，错误率 > 50% 且请求量 > 10/s 时熔断 30s，直接返回缓存或错误
- 当前: QueryService 无熔断逻辑
- 修复: 在 QueryService wiring 中接入 `circuit_breaker.go`

#### 断裂 17: Query 缓存缺失
- 蓝图要求: §3.3 — `cache_service.go` 缓存查询结果
- 当前: QueryService 无缓存
- 修复: 在 QueryService wiring 中接入 `cache_service.go`

#### 断裂 18: Query 降级缺失
- 蓝图要求: §3.3 — DB 不可用时返回缓存数据（带 `X-Cache-Stale` 头），缓存也不可用时返回预设默认值
- 当前: QueryService 无降级逻辑
- 修复: 在 QueryService wiring 中接入 `degradation_handler.go`

#### 断裂 19: Query 一致性检查缺失
- 蓝图要求: §3.3 — `consistency_checker.go` 对比缓存 vs DB，差异写入修复队列
- 当前: QueryService 无一致性检查
- 修复: 在 QueryService wiring 中接入 `consistency_checker.go`

#### 断裂 20: RPC 故障切换缺失
- 蓝图要求: §3.1 — 多节点池，失败自动切换到备用节点
- 当前: 每条链只用单个 RPC 端点
- 修复: 为每条链配置 2+ 个 RPC 端点，失败时自动切换

#### 断裂 21: API Gateway 限流/认证缺失
- 蓝图要求: §3.4 — 内存令牌桶，API Key 1000/min，IP 100/min；mock 认证
- 当前: main.go 没有接入限流和认证
- 修复: 接入 `rate_limiter.go` 和 `auth_middleware.go`（mock 认证）

#### 断裂 22: WebSocket 连接池上限缺失
- 蓝图要求: §3.4 — 单 Pod 上限 10000
- 当前: 需要确认连接池有 maxConns 限制
- 修复: 确认或添加 WebSocket 连接池上限配置

#### 断裂 23: 统一标签注入缺失
- 蓝图要求: §5 — 所有指标、日志、trace 携带 chain_id, service, operation, block_height
- 当前: 指标和日志没有统一携带这些标签
- 修复: 在 metrics/log 调用中统一注入这四个标签

#### 断裂 24: Puller 健康端点缺失
- 蓝图要求: §3.1 — `GET /health/puller` 返回各链拉取状态
- 当前: 没有 Puller 健康端点
- 修复: 注册 `/health/puller` 端点，返回各链的 lastIndexedBlock、blockLag、rpcErrors

#### 断裂 25: 缓存击穿防护缺失
- 蓝图要求: §3.3 — `cache_warmer.go` 预热 + `cache_middleware.go` 单机锁
- 当前: 两个文件不存在
- 修复: 创建 `cache_warmer.go` 和 `cache_middleware.go`

#### 断裂 26: 分布式追踪缺失
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
  → 熔断检查: circuit_breaker.Allow()  // 蓝图 §3.3: 错误率 > 50% 熔断 30s
  → 缓存检查: cache_service.Get()  // 蓝图 §3.3: 缓存查询结果
  → 缓存击穿防护: cache_middleware.GetOrLock() + cache_warmer.Warm()  // 蓝图 §3.3
  → DB 查询: MockDB.QueryEvents()
    - DB 失败时 → 降级到缓存（X-Cache-Stale 头）  // 蓝图 §3.3: 降级
    - 缓存也不可用 → 返回预设默认值
  → 一致性检查: consistency_checker.Verify()  // 蓝图 §3.3: 缓存 vs DB 对比
  → GraphQL API 返回
```

### 目标
修复上述 11 个断裂，使单体模式具备以下能力：
1. **Query 容错**: 熔断（circuit_breaker.go）+ 缓存（cache_service.go）+ 降级（DB→缓存→默认值）+ 一致性检查（consistency_checker.go）+ 缓存击穿防护（cache_warmer + cache_middleware）
2. **API Gateway**: 内存令牌桶限流（API Key 1000/min, IP 100/min）+ mock 认证 + WebSocket 连接池（上限 10000）
3. **可观测性**: 统一标签注入（chain_id, service, operation, block_height）+ Puller 健康端点 + 分布式追踪（OTel Tracer）
4. **Puller 可靠性**: RPC 故障切换（多节点池）

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性
- [ ] **Query 熔断接入**（蓝图 §3.3: circuit_breaker.go，错误率 > 50% 熔断 30s）
- [ ] **Query 缓存接入**（蓝图 §3.3: cache_service.go）
- [ ] **Query 降级接入**（蓝图 §3.3: DB→缓存→默认值，X-Cache-Stale 头）
- [ ] **Query 一致性检查接入**（蓝图 §3.3: consistency_checker.go）
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
- `pkg/services/query/circuit_breaker.go` — 熔断器
- `pkg/services/query/consistency_checker.go` — 一致性检查
- `pkg/services/query/degradation_handler.go` — 降级策略

### 修复步骤

**Step 1: Query 熔断 + 缓存接入**
```
文件: cmd/monolithic/chainpulse/main.go
在 QueryService wiring 中接入:
1. circuit_breaker.go — 熔断器（错误率 > 50% 且 > 10/s 时熔断 30s，返回缓存或错误）
2. cache_service.go — 缓存查询结果
```

**Step 2: Query 降级 + 一致性检查接入**
```
文件: cmd/monolithic/chainpulse/main.go
在 QueryService wiring 中接入:
3. degradation_handler.go — 降级（DB 失败→返回缓存带 X-Cache-Stale 头→缓存也不可用→返回预设默认值）
4. consistency_checker.go — 一致性检查（对比缓存 vs DB，差异写入修复队列）
```

**Step 3: API Gateway 限流 + 认证 + WebSocket 连接池**
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

**Step 4: Puller 健康端点**
```
注册 GET /health/puller 端点，返回各链的:
  - lastIndexedBlock
  - blockLag (chainHead - lastIndexedBlock)
  - rpcErrors (错误计数)
  - isRunning
```

**Step 5: 统一标签注入**
```
在 metrics/log 调用中统一注入:
  metrics.WithLabels("chain_id", chainID, "service", "puller", "operation", "pull_events")
  logger.Info("pulled events", "chain_id", chainID, "service", "puller", "operation", "pull_events", "block_height", toBlock)
```

**Step 6: 缓存击穿防护**
```
新建 pkg/services/query/cache_warmer.go:
  - 定时预热最近 N 个块的事件到缓存

新建 pkg/services/query/cache_middleware.go:
  - 使用 singleflight 防止缓存击穿
  - 缓存未命中时只允许一个请求查 DB，其他等待
```

**Step 7: 分布式追踪**
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
- **蓝图要求的以下功能全部必须实现，不可跳过**:
  - Query 熔断（§3.3: circuit_breaker.go）
  - Query 缓存（§3.3: cache_service.go）
  - Query 降级（§3.3: DB→缓存→默认值）
  - Query 一致性检查（§3.3: consistency_checker.go）
  - 缓存击穿防护（§3.3: cache_warmer + cache_middleware）
  - API Gateway 限流 + 认证（§3.4）
  - WebSocket 连接池上限（§3.4）
  - 统一标签注入（§5）
  - Puller 健康端点（§3.1）
  - 分布式追踪（§5.3）

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
