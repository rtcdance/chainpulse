# Context 模式实战：从 Web3 到 Go

> 本文档覆盖 Go `context.Context` 的七大实战模式，每个模式配 chainpulse 真实代码示例。

---

## 零、Web3 概念对照

| Web3 概念 | Go Context |
|-----------|-----------|
| `eth_call` 的 timeout 参数 | `context.WithTimeout(ctx, 30*time.Second)` |
| AbortController / signal.aborted | `ctx.Done()` channel |
| Promise 链中的取消传播 | 所有子 goroutine 共享同一个 ctx |
| `setTimeout(fn, ms)` | `time.After(d)` + `select { case <-ctx.Done() }` |
| `Promise.race([p1, p2])` | `select { case <-ch1: ... case <-ch2: ... }` |

---

## 一、超时控制（WithTimeout）

**场景**：RPC 调用、数据库查询、HTTP 请求都需要超时。

```go
// 正确：5 秒后自动取消
func fetchBlock(ctx context.Context, number uint64) (*Block, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel() // 避免 goroutine 泄漏

    return rpcClient.BlockByNumber(ctx, big.NewInt(int64(number)))
}
```

**chainpulse 实例**：[mongodb_adapter.go](file://../../pkg/services/query/mongodb_adapter.go) — MongoDB 查询统一 5 秒超时。

**常见错误**：
```go
// ❌ 忘记 defer cancel() → goroutine 泄漏
ctx, _ := context.WithTimeout(ctx, 5*time.Second)

// ✅ 正确
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

---

## 二、取消传播（WithCancel）

**场景**：父任务取消时，所有子 goroutine 自动收到通知。

```go
func indexAllChains(ctx context.Context) error {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    g, ctx := errgroup.WithContext(ctx)
    for _, chain := range chains {
        chain := chain
        g.Go(func() error {
            return indexChain(ctx, chain)
        })
    }
    return g.Wait() // 任一 goroutine 出错，其他都会收到 ctx.Done()
}
```

**chainpulse 实例**：[multi_chain_indexer.go](file://../../pkg/services/indexing/multi_chain_indexer.go) — 多链并行索引。

---

## 三、优雅关闭（signal.NotifyContext）

**场景**：接收 SIGINT/SIGTERM 后，给服务一个清理窗口。

```go
// Go 1.16+ 的标准模式
func run() error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // ... 启动服务 ...

    <-ctx.Done() // 等待信号

    // 30 秒内完成清理
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return server.Shutdown(shutdownCtx)
}
```

**chainpulse 实例**：playground 的 [main.go](file://../../cmd/playground/main.go) — 刚重构为此模式。

---

## 四、请求级 Context（WithValue）

**场景**：在请求链路中传递 trace ID、用户 ID 等元数据。

```go
type contextKey string

const (
    ctxKeyRequestID contextKey = "request_id"
    ctxKeyUserID    contextKey = "user_id"
)

// 中间件注入
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := uuid.New().String()
        ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 业务层提取
func processEvent(ctx context.Context, event *Event) error {
    reqID, _ := ctx.Value(ctxKeyRequestID).(string)
    logger.Info("processing event", "request_id", reqID)
    // ...
}
```

**重要**：`context.WithValue` 只用于请求范围内的元数据，不要用来传业务参数。

---

## 五、Select 多路复用

**场景**：同时等待 channel 数据和 context 取消。

```go
func subscribeEvents(ctx context.Context, ch <-chan Event) {
    for {
        select {
        case evt, ok := <-ch:
            if !ok {
                return // channel 已关闭
            }
            handle(evt)
        case <-ctx.Done():
            logger.Warn("subscription cancelled", "reason", ctx.Err())
            return
        }
    }
}
```

**chainpulse 实例**：[channel_eventbus.go](file://../../pkg/core/channel_eventbus.go) — ChannelEventBus 的订阅 goroutine。

---

## 六、组合模式（Context 链）

**场景**：同时需要超时 + 取消 + 元数据。

```go
// Base context (from main)
ctx := context.Background()

// Layer 1: 添加超时
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

// Layer 2: 添加请求 ID
ctx = context.WithValue(ctx, ctxKeyRequestID, reqID)

// Layer 3: 创建子作用域
childCtx, childCancel := context.WithCancel(ctx)
defer childCancel()

// 此时 childCtx 继承所有上层设置：
//   - 10 秒后自动超时
//   - 携带 request_id
//   - 可独立取消
```

---

## 七、Deadline vs Timeout

```go
// 相对时间（推荐用于具体操作）
ctx, cancel := context.WithTimeout(parent, 5*time.Second)

// 绝对时间（用于需要精确截止时间的场景）
deadline := time.Now().Add(30 * time.Second)
ctx, cancel := context.WithDeadline(parent, deadline)

// 检查剩余的截止时间
if d, ok := ctx.Deadline(); ok {
    log.Printf("time remaining: %v", time.Until(d))
}
```

**chainpulse 实例**：整个项目统一使用 `WithTimeout`，约 100 处调用。5 秒是网络操作的默认超时。

---

## 八、关键最佳实践

### ✅ DO
- 总是 `defer cancel()` — 避免 goroutine 泄漏
- Context 作为第一个参数 — `func(ctx context.Context, ...)`
- 使用 `context.TODO()` 标记将来需要补充 context 的地方
- 传给下游的所有函数都接受 context

### ❌ DON'T
- 不要将 context 存储在 struct 中 — 它是请求级的
- 不要用 `context.WithValue` 传递业务参数
- 不要创建过长的超时（生产环境建议 ≤ 30s）
- 不要忽略 `ctx.Done()` 的返回值 — 它是取消的唯一信号

---

## 九、chainpulse 中的 Context 使用统计

| 模式 | 使用次数 | 典型场景 |
|------|---------|---------|
| `context.WithTimeout` | ~100 | 数据库 Ping、RPC 调用、HTTP 请求 |
| `context.WithCancel` | ~20 | 多链索引、订阅管理、优雅关闭 |
| `context.WithValue` | 少量 | 请求 ID、trace 传播 |
| `context.Background()` | 全局 | main 函数、测试入口 |

---

## 十、延伸阅读

- [Go Blog: Context](https://go.dev/blog/context)
- [Go Concurrency Patterns: Context](https://go.dev/blog/context)
- chainpulse 优雅关闭重构：`cmd/playground/main.go`
- chainpulse ChannelEventBus：`pkg/core/channel_eventbus.go`