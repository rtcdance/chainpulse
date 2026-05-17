# Processor — 事件处理引擎

## Web3 概念

ChainPulse 从区块链节点拉取到原始事件日志后，需要经过一系列处理才能持久化到数据库供查询：

1. **验证**：检查事件结构的完整性（network、block number、tx hash、contract address）
2. **幂等性检查**：防止同一事件被重复处理（区块链可能因重组产生重复事件）
3. **持久化**：写入数据库
4. **缓存更新**：写入缓存加速后续查询

## 处理流程

```
ProcessEvent(event)
  ├── validateEvent()          ← 结构验证
  ├── idempotencyService.GenerateHash()   ← 生成唯一哈希
  ├── idempotencyService.IsDuplicate()    ← 检查是否已处理
  ├── storeEventWithRetry()    ← 写入数据库（最多 3 次重试）
  ├── idempotencyService.MarkProcessed()  ← 标记已处理
  └── cachePlugin.Set()        ← 更新缓存
```

## Go 要点

| 模式 | 位置 | 说明 |
|------|------|------|
| 指数退避重试 | `storeEventWithRetry():450-478` | `boundedRetryMultiplier()` 防止退避溢出 |
| 幂等性 | `idempotency.go` | `GenerateHash()` + `IsDuplicate()` + `MarkProcessed()` |
| 接口隔离 | `event_processor.go:17-26` | 依赖 `EventStorage` 和 `CacheWriter` 小接口，而非完整 `DatabasePlugin` |
| Context 传播 | 所有方法 | 每个公共方法接受 `context.Context`，支持优雅关闭 |
| 健康检查 | `event_processor.go:178-200` | 自定义 health check 暴露运行状态 |

## 幂等性设计

幂等性是 Web3 索引器的关键需求：
- 区块链重组可能导致同一事件被多次拉取
- 重试机制可能重复执行 `WriteEvent`
- 解决方案：`GenerateHash(event)` → 存入幂等性存储 → 后续事件先查再写

## 学习路径

1. 读 `ProcessEvent()` 理解全流程
2. 读 `storeEventWithRetry()` 理解重试 + context-aware backoff
3. 读 `boundedRetryMultiplier()` 理解防溢出退避算法
4. 读 `idempotency.go` 理解幂等性存储接口和实现

## 关键文件

| 文件 | 功能 |
|------|------|
| `event_processor.go` | 核心处理引擎（493 行） |
| `idempotency.go` | 幂等性服务接口和实现 |