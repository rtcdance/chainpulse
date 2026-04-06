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

### 当前状态：11 个断裂点

#### 断裂 16-19: Query 容错缺失（熔断 + 缓存 + 降级 + 一致性检查）
- 蓝图要求: §3.3 — Query 支持熔断、缓存、降级、一致性检查
- 当前: QueryService 无容错逻辑
- 修复: 在 QueryService wiring 中接入 circuit_breaker、cache_service、degradation_handler、consistency_checker

#### 断裂 20: RPC 故障切换缺失
- 蓝图要求: §3.1 — 多节点池，失败自动切换到备用节点
- 当前: 每条链只用单个 RPC 端点
- 修复: 为每条链配置 2+ 个 RPC 端点，失败时自动切换

#### 断裂 21: API Gateway 限流/认证缺失
- 蓝图要求: §3.4 — 内存令牌桶限流，mock 认证
- 当前: main.go 没有接入限流和认证
- 修复: 创建 RateLimitMiddleware 和 AuthMiddleware，通过 `gateway.SetRateLimitMiddleware()` 和 `gateway.SetAuthMiddleware()` 接入

#### 断裂 22: WebSocket 连接池上限缺失
- 蓝图要求: §3.4 — 单 Pod 上限 10000
- 当前: ConnectionPoolManager 有 maxConns 字段但需要确认配置
- 修复: 确认或添加 WebSocket 连接池上限配置

#### 断裂 23: 统一标签注入缺失
- 蓝图要求: §5 — 所有指标、日志携带 chain_id, service, operation, block_height
- 当前: 指标和日志没有统一携带这些标签
- 修复: 在 IndexerMetrics 和 logger 调用中统一注入这四个标签

#### 断裂 24: Puller 健康端点缺失
- 蓝图要求: §3.1 — `GET /health/puller` 返回各链拉取状态
- 当前: 没有 Puller 健康端点
- 修复: 注册 `/health/puller` HTTP handler

#### 断裂 25: 缓存击穿防护缺失
- 蓝图要求: §3.3 — `cache_warmer.go` 预热 + `cache_middleware.go` 单机锁
- 当前: 两个文件不存在
- 修复: 创建 `cache_warmer.go` 和 `cache_middleware.go`

#### 断裂 26: 分布式追踪缺失
- 蓝图要求: §5.3 — Tracer，span 携带 chain_id, from_block, to_block
- 当前: `pkg/observability/distributed_tracing.go` 有 Tracer 接口和 DefaultTracer 实现，但未被使用
- 修复: 在 Puller/Indexer/Query 的关键操作中注入 Tracer span

### 完整数据流（修复后）

```
API Gateway (蓝图 §3.4)
  → 认证: authMiddleware.Handler()  // mock 认证
  → 限流: gateway.SetRateLimitMiddleware(rateLimitMiddleware)  // 内存令牌桶
  → WebSocket 连接池: ConnectionPoolManager(maxConns=10000)
  → GraphQL 路由

Health Endpoints (蓝图 §3.1)
  → GET /health/puller → 返回各链拉取状态（lastIndexedBlock, blockLag, rpcErrors）
  → GET /health → 总体健康状态

Observability (蓝图 §5)
  → IndexerMetrics: RecordIndexingProgress(currentBlock, latestBlock)
  → IndexerMetrics: RecordEventIndexed(latency)
  → IndexerMetrics: RecordEventFailed(errorType)
  → DefaultTracer: StartSpan(ctx, "pull_events") → AddEvent → EndSpan
  → 所有日志携带: chain_id, service, operation, block_height

QueryService (蓝图 §3.3)
  → 熔断: circuitBreaker.Call(fn)  // 错误率 > 50% 熔断 30s
  → 缓存: cacheService.Get(ctx, key)
  → 缓存击穿防护: cache_middleware.GetOrLock() + cache_warmer.Warm()
  → DB 查询: MockDB.QueryEvents()
    - DB 失败时 → degradationHandler 降级到缓存
  → 一致性检查: consistencyChecker.CheckConsistency(ctx)
  → GraphQL API 返回
```

### 目标
修复上述 11 个断裂，使单体模式具备蓝图要求的完整可观测性、API Gateway 和 Query 容错能力。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性
- [ ] **Query 熔断接入**（蓝图 §3.3: circuitBreaker.Call(fn)，错误率 > 50% 熔断 30s）
- [ ] **Query 缓存接入**（蓝图 §3.3: cacheService.Get(ctx, key)）
- [ ] **Query 降级接入**（蓝图 §3.3: degradationHandler，DB 失败→缓存→默认值）
- [ ] **Query 一致性检查接入**（蓝图 §3.3: consistencyChecker.CheckConsistency(ctx)）
- [ ] **RPC 故障切换**（蓝图 §3.1: 多节点池，失败自动切换）
- [ ] **API Gateway 限流**（蓝图 §3.4: SetRateLimitMiddleware，API Key 1000/min, IP 100/min）
- [ ] **API Gateway mock 认证**（蓝图 §3.4: SetAuthMiddleware）
- [ ] **WebSocket 连接池上限**（蓝图 §3.4: maxConns=10000）
- [ ] **统一标签注入**（蓝图 §5: chain_id, service, operation, block_height）
- [ ] **Puller 健康端点**（蓝图 §3.1: GET /health/puller）
- [ ] **缓存击穿防护**（蓝图 §3.3: cache_warmer + cache_middleware）
- [ ] **分布式追踪**（蓝图 §5.3: DefaultTracer.StartSpan/AddEvent/EndSpan）

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`

### 参考文件（含实际 API 签名）

- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，§3.1 + §3.3 + §3.4 + §5**
- `cmd/monolithic/chainpulse/main.go` — Composition Root（M1-1a/b 已修改）

**API Gateway**:
- `pkg/plugins/api/gateway.go`:
  - `SetAuthMiddleware(middleware *AuthMiddleware)`
  - `SetRateLimitMiddleware(middleware *RateLimitMiddleware)`
- `pkg/plugins/api/rate_limiter.go`:
  - `NewRateLimiter(logger, metrics, config *RateLimitConfig) *RateLimiter`
  - `RateLimitConfig{APIKeyRate: 1000, IPRate: 100}`
  - `NewRateLimitMiddleware(limiter *RateLimiter, logger core.Logger) *RateLimitMiddleware` — 注意: 第一个参数是 *RateLimiter 实例，不是 config
- `pkg/plugins/api/auth_middleware.go`:
  - `NewAuthMiddleware(tokenValidator, rbacChecker, auditLogger, logger, metrics) *AuthMiddleware` — 注意: 5 个参数，mock 时前 3 个传 nil
- `pkg/infrastructure/gateway/websocket_subscription.go`:
  - `NewConnectionPoolManager(maxConns int) *ConnectionPoolManager`

**Query 容错**:
- `pkg/services/query/circuit_breaker.go`:
  - `NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker`
  - `cb.Call(fn func() error) error` — 不是 Allow()!
  - `CircuitBreakerConfig{ErrorThreshold: 0.5, RequestThreshold: 10, SleepWindow: 30*time.Second}`
- `pkg/services/query/cache_service.go`:
  - `NewCacheService(logger, metricsCollector) CacheService` — ⚠️ 注意: **不需要** cache 参数！内部使用 map 缓存
  - `cs.Get(ctx, key string) ([]core.BlockchainEvent, error)`
  - `cs.Set(ctx, key string, value []core.BlockchainEvent, ttl) error`
  - `cs.Initialize(ctx) error`
  - `cs.Start(ctx) error`
- `pkg/services/query/degradation_handler.go`:
  - `NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics) DegradationHandler`
  - ⚠️ 参数顺序: eventStore(EventStore 接口), metadataStore(EventMetadataStore 接口), cacheService(CacheService 接口), logger, metrics
  - `h.GetDegradationMode(ctx) DegradationMode`
  - `h.CanUseCache(ctx) bool`
- `pkg/services/query/consistency_checker.go`:
  - `NewConsistencyChecker(eventStore, metadataStore, logger, metrics) *ConsistencyChecker`
  - `cc.CheckConsistency(ctx) (*ConsistencyCheckResult, error)`

**可观测性**:
- `pkg/observability/indexer_metrics.go`:
  - `NewIndexerMetrics() *IndexerMetrics`
  - `im.RecordIndexingProgress(currentBlock, latestBlock uint64)`
  - `im.RecordEventIndexed(latency time.Duration)`
  - `im.RecordEventFailed(errorType string)`
  - `im.RecordReorg(blocksRolledBack uint64)`
- `pkg/observability/distributed_tracing.go`:
  - `NewDefaultTracer(logger, metrics) *DefaultTracer`
  - `tracer.StartSpan(ctx, name string, kind SpanKind) (context.Context, Span)` — ⚠️ 返回 Span（非指针）
  - `tracer.AddEvent(span *Span, name string, attributes map[string]interface{})`
  - `tracer.EndSpan(span *Span)`

### 修复步骤

**Step 1: Query 熔断 + 缓存接入**
```
文件: cmd/monolithic/chainpulse/main.go
在 QueryService wiring 中:
1. circuitBreaker := query.NewCircuitBreaker(&query.CircuitBreakerConfig{
     ErrorThreshold: 0.5, RequestThreshold: 10, SleepWindow: 30*time.Second,
   })
2. cacheService := query.NewCacheService(logger, metrics)  // ⚠️ 注意: 不需要 cache 参数
   cacheService.Initialize(ctx)
   cacheService.Start(ctx)
```

**Step 2: Query 降级 + 一致性检查接入**
```
文件: cmd/monolithic/chainpulse/main.go

// 注意: NewDegradationHandler 签名是 (eventStore, metadataStore, cacheService, logger, metrics)
// eventStore 和 metadataStore 需要从 indexingDatabase 或 query 层的 store 获取
// 如果当前没有现成的 eventStore 实现，先用 indexingDatabase 包装
degradationHandler := query.NewDegradationHandler(eventStore, metadataStore, cacheService, logger, metrics)
degradationHandler.Initialize(ctx)

// 注意: NewConsistencyChecker 签名是 (eventStore, metadataStore, logger, metrics)
consistencyChecker := query.NewConsistencyChecker(eventStore, metadataStore, logger, metrics)
consistencyChecker.Initialize(ctx)
```

**Step 3: API Gateway 限流 + 认证**
```
文件: cmd/monolithic/chainpulse/main.go

// 先创建 RateLimiter 实例
rateLimiter := api.NewRateLimiter(logger, metrics, &api.RateLimitConfig{
    APIKeyRate: 1000,  // 1000 req/min per API Key
    IPRate:     100,   // 100 req/min per IP
})

// 再创建中间件（注意: NewRateLimitMiddleware 接受 (limiter, logger)）
rateLimitMiddleware := api.NewRateLimitMiddleware(rateLimiter, logger)

// mock 认证（注意: NewAuthMiddleware 需要 5 个参数，mock 时传 nil）
authMiddleware := api.NewAuthMiddleware(nil, nil, nil, logger, metrics)

gateway.SetRateLimitMiddleware(rateLimitMiddleware)
gateway.SetAuthMiddleware(authMiddleware)
```

**Step 4: Puller 健康端点 + WebSocket 连接池**
```
文件: cmd/monolithic/chainpulse/main.go
1. 注册 HTTP handler:
   http.HandleFunc("/health/puller", func(w http.ResponseWriter, r *http.Request) {
     // 返回各链的 lastIndexedBlock, blockLag, rpcErrors
   })
2. WebSocket 连接池:
   connPool := gateway.NewConnectionPoolManager(10000)  // 上限 10000
```

**Step 5: 统一标签注入 + 分布式追踪**
```
文件: cmd/monolithic/chainpulse/main.go
1. IndexerMetrics:
   metrics := observability.NewIndexerMetrics()
   metrics.RecordIndexingProgress(lastBlock, chainHead)
   metrics.RecordEventIndexed(latency)
2. Tracer:
   tracer := observability.NewDefaultTracer(logger, metrics)
   ctx, span := tracer.StartSpan(ctx, "pull_events", observability.SpanKindClient)
   tracer.AddEvent(&span, "pulled", map[string]interface{}{
     "chain_id": chainID, "from_block": fromBlock, "to_block": toBlock,
   })
   defer tracer.EndSpan(&span)
3. Logger:
   logger.Info("pulled events", "chain_id", chainID, "service", "puller",
     "operation", "pull_events", "block_height", toBlock)
```

**Step 6: 缓存击穿防护**
```
新建 pkg/services/query/cache_warmer.go:
  - 定时预热最近 N 个块的事件到缓存

新建 pkg/services/query/cache_middleware.go:
  - 使用 singleflight 防止缓存击穿
  - 缓存未命中时只允许一个请求查 DB，其他等待
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反
- **必须与 ARCHITECTURE_v1.md 蓝图一致**
- **API 使用必须与上述参考文件中的实际签名完全一致**:
  - CircuitBreaker 用 Call(fn)，不是 Allow()
  - CacheService 用 Get(ctx, key)，不是 Get()
  - Gateway 用 SetRateLimitMiddleware(middleware)，不是 SetRateLimiter(limiter)
  - Tracer 用 StartSpan/AddEvent/EndSpan，不是 OTel 的 span 接口
  - 指标用 IndexerMetrics，不是 prometheus.NewGaugeVec

### 验证步骤
```bash
make build
make test-unit
make vet
make run-monolithic &
sleep 60
# 验证限流
curl -s http://localhost:8080/graphql -d '{"query": "{ events(limit: 1) { id } }"}'
# 验证健康端点
curl -s http://localhost:8080/health/puller
curl -s http://localhost:8080/health
# 验证指标
curl -s http://localhost:8080/metrics | grep chainpulse_ | head -10
```
