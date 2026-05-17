# 废弃 API 迁移指南

> 本文档列出 chainpulse 中标记为 `// Deprecated:` 的 API，提供迁移路径和替代方案。

---

## 废弃 API 总览

| # | 废弃项 | 文件 | 替代方案 |
|---|--------|------|---------|
| 1 | `NewDefaultLogger` | [logger.go](file://../../pkg/core/logger.go#L43) | `NewSlogLogger` |
| 2 | `EventFilter.BuildLegacy()` | [event_filter.go](file://../../pkg/core/event_filter.go#L267) | `Build()` + 显式错误处理 |
| 3 | `DefaultBlobGasLimit` | [gas_estimator.go](file://../../pkg/core/gas_estimator.go#L254) | `BlobParamsForFork()` |
| 4 | 旧版 Retry 函数 | [errors.go](file://../../pkg/core/errors.go#L199) | `services/resilience.RetryExecutor` |
| 5 | `NewShadowWriteTracker` | [shadow_write_tracker.go](file://../../pkg/services/indexing/shadow_write_tracker.go#L59) | 依赖注入 |
| 6 | `circuit_breaker.Call()` | [circuit_breaker.go](file://../../pkg/services/query/circuit_breaker.go#L95) | `CallWithContext(ctx, fn)` |
| 7 | `retry_logic.Call()` | [retry_logic.go](file://../../pkg/infrastructure/processing/retry_logic.go#L186) | `CallWithContext(ctx, op)` |
| 8 | `NewDefaultTracer` | [distributed_tracing.go](file://../../pkg/observability/distributed_tracing.go#L144) | `NewDefaultTracerWithProvider` |

---

## 迁移趋势

### 趋势 1：无 Context → 有 Context

```go
// ❌ Deprecated: 无超时控制
result, err := breaker.Call(func() error {
    return doWork()
})

// ✅ 迁移到
result, err := breaker.CallWithContext(ctx, func() error {
    return doWork()
})
```

**为什么？** 没有 context 的调用无法被取消、无法设置超时、无法传播 trace。这是 Go 生态最核心的演进方向。

### 趋势 2：全局单例 → 依赖注入

```go
// ❌ Deprecated: 全局单例
tracker := indexing.NewShadowWriteTracker()
value := indexing.GetShadowWriteTracker()

// ✅ 迁移到
type MyService struct {
    tracker *indexing.ShadowWriteTracker
}
// 由构造函数注入
```

**为什么？** 全局单例难以测试、难以替换实现（如 mock）、违反了依赖反转原则。

### 趋势 3：魔法常量 → 配置化

```go
// ❌ Deprecated: 硬编码常量
blobLimit := DefaultBlobGasLimit

// ✅ 迁移到
params := BlobParamsForFork(fork)
blobLimit := params.MaxBlobGas
```

**为什么？** EIP-4844 的 blob gas 限制会随 fork 版本变化，硬编码无法适应网络升级。

---

## 迁移检查清单

- [ ] `NewDefaultLogger` → `NewSlogLogger`（日志标准化）
- [ ] `EventFilter.BuildLegacy()` → `Build()`（显式错误处理）
- [ ] 旧版 Retry → `RetryExecutor`（统一重试策略）
- [ ] 全局 Trackers → 依赖注入（可测试性）
- [ ] `Call()` → `CallWithContext()`（上下文传播）
- [ ] `NewDefaultTracer` → `NewDefaultTracerWithProvider`（可观测性）

---

## 迁移方法

### 渐进式迁移（推荐）

1. **标记阶段**：添加 `// Deprecated:` 注释（已完成）
2. **内部迁移**：将项目内部调用替换为新 API
3. **兼容期**：保留废弃 API 至少一个版本，提供平滑过渡
4. **移除**：下个主要版本中移除

### 工具辅助

```bash
# 查找所有废弃 API 的使用
grep -r "Deprecated:" pkg/ --include="*.go"

# 使用 staticcheck 检测废弃 API
staticcheck ./...

# 使用 golangci-lint 中的 gocritic（已启用 deprecatedComment 检查）
golangci-lint run --enable gocritic
```