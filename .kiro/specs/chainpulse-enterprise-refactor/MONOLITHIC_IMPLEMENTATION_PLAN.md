# 单体服务实现计划：Phase 3/4/12 优先级

**日期**: 2026年1月11日  
**目标**: 作为单体服务，优先完整实现 Phase 3、4、12，快速达到生产就绪  
**策略**: 聚焦核心功能，完整实现消息队列、事件处理、文档

---

## 总体策略

### 为什么调整优先级？

1. **Phase 3 (消息队列)**: 是事件处理的基础，必须完整实现
2. **Phase 4 (事件处理)**: 核心业务逻辑，决定系统可靠性
3. **Phase 12 (文档)**: 生产就绪的必要条件

### 实现顺序

```
Phase 1-2 (已完成)
    ↓
Phase 3 (消息队列) ← 优先完整实现
    ↓
Phase 4 (事件处理) ← 优先完整实现
    ↓
Phase 5-11 (并行实现)
    ↓
Phase 12 (文档) ← 优先完整实现
```

---

## Phase 3: 消息队列插件 (完整实现)

### 目标
完整实现所有消息队列插件，支持 Kafka、Redis、ZeroMQ

### 任务列表

#### 3.1 消息队列插件接口 (已完成)
- [x] 定义 MQPlugin 接口
- [x] 创建基础实现
- [x] 实现死信队列支持

#### 3.2 Kafka 消息队列插件 (需完整实现)
**文件**: `pkg/plugins/mq/kafka_mq.go`

```go
type KafkaMessageQueue struct {
    producer sarama.SyncProducer
    consumer sarama.ConsumerGroup
    config   *KafkaConfig
    logger   Logger
    metrics  MetricsCollector
}

// 需要实现的方法:
- Publish(ctx context.Context, topic string, message []byte) error
- Subscribe(ctx context.Context, topic string, handler func([]byte)) error
- GetQueueDepth(ctx context.Context, topic string) (int64, error)
- PublishToDLQ(ctx context.Context, message []byte, reason string) error
- GetDLQMessages(ctx context.Context, limit int) ([][]byte, error)
```

**子任务**:
- [ ] 3.2.1 实现 Kafka 生产者
  - 连接池管理
  - 批量发送优化
  - 错误处理和重试
  - _Requirements: 2.1, 2.8_

- [ ] 3.2.2 实现 Kafka 消费者
  - 消费者组管理
  - 偏移量追踪
  - 优雅关闭
  - _Requirements: 2.1, 2.8_

- [ ] 3.2.3 实现死信队列
  - DLQ 主题管理
  - 失败消息存储
  - DLQ 消息查询
  - _Requirements: 2.4_

- [ ]* 3.2.4 Kafka 插件属性测试
  - **Property 46: Multi-MQ Support**
  - **Validates: Requirements 2.8**

- [ ]* 3.2.5 Kafka 集成测试
  - 测试消息发布和消费
  - 测试偏移量追踪
  - 测试死信队列

#### 3.3 Redis 消息队列插件 (需完整实现)
**文件**: `pkg/plugins/mq/redis_mq.go`

```go
type RedisMessageQueue struct {
    client   redis.Client
    config   *RedisConfig
    logger   Logger
    metrics  MetricsCollector
}

// 需要实现的方法:
- Publish(ctx context.Context, topic string, message []byte) error
- Subscribe(ctx context.Context, topic string, handler func([]byte)) error
- GetQueueDepth(ctx context.Context, topic string) (int64, error)
- PublishToDLQ(ctx context.Context, message []byte, reason string) error
```

**子任务**:
- [ ] 3.3.1 实现 Redis Stream 生产者
  - 使用 Redis Stream 数据结构
  - 自动 ID 生成
  - 批量操作
  - _Requirements: 2.1, 2.8_

- [ ] 3.3.2 实现 Redis Stream 消费者
  - 消费者组管理
  - 消息确认
  - 待处理列表 (PEL) 管理
  - _Requirements: 2.1, 2.8_

- [ ] 3.3.3 实现死信队列
  - DLQ Stream 管理
  - 失败消息存储
  - _Requirements: 2.4_

- [ ]* 3.3.4 Redis MQ 属性测试
  - **Property 46: Multi-MQ Support**
  - **Validates: Requirements 2.8**

- [ ]* 3.3.5 Redis 集成测试
  - 测试消息发布和消费
  - 测试消费者组
  - 测试死信队列

#### 3.4 ZeroMQ 消息队列插件 (需完整实现)
**文件**: `pkg/plugins/mq/zeromq_mq.go`

```go
type ZeroMQMessageQueue struct {
    publisher *zmq.Socket
    subscriber *zmq.Socket
    config    *ZeroMQConfig
    logger    Logger
    metrics   MetricsCollector
}

// 需要实现的方法:
- Publish(ctx context.Context, topic string, message []byte) error
- Subscribe(ctx context.Context, topic string, handler func([]byte)) error
- GetQueueDepth(ctx context.Context, topic string) (int64, error)
```

**子任务**:
- [ ] 3.4.1 实现 ZeroMQ 发布者
  - PUB 套接字
  - 主题路由
  - 消息序列化
  - _Requirements: 2.1, 2.8_

- [ ] 3.4.2 实现 ZeroMQ 订阅者
  - SUB 套接字
  - 主题过滤
  - 消息反序列化
  - _Requirements: 2.1, 2.8_

- [ ] 3.4.3 实现死信队列
  - 失败消息处理
  - DLQ 存储
  - _Requirements: 2.4_

- [ ]* 3.4.4 ZeroMQ 属性测试
  - **Property 46: Multi-MQ Support**
  - **Validates: Requirements 2.8**

- [ ]* 3.4.5 ZeroMQ 集成测试
  - 测试消息发布和消费
  - 测试主题路由
  - 测试死信队列

#### 3.5 检查点 - 消息队列完成
- [ ] 所有 MQ 插件实现完成
- [ ] 所有测试通过
- [ ] 死信队列功能验证
- [ ] 性能基准测试

---

## Phase 4: 事件处理核心 (完整实现)

### 目标
完整实现幂等性服务、事件处理器、缓存更新逻辑

### 任务列表

#### 4.1 幂等性服务 (需完整实现)
**文件**: `pkg/services/idempotency/idempotency_service.go`

```go
type IdempotencyService interface {
    GenerateHash(event BlockchainEvent) string
    IsDuplicate(ctx context.Context, hash string) (bool, error)
    MarkProcessed(ctx context.Context, hash string) error
    GetProcessedCount(ctx context.Context) (int64, error)
}

type IdempotencyServiceImpl struct {
    db      DatabasePlugin
    cache   CachePlugin
    logger  Logger
    metrics MetricsCollector
}
```

**子任务**:
- [ ] 4.1.1 实现事件哈希生成
  - SHA-256 哈希算法
  - 确定性输入
  - 性能优化
  - _Requirements: 2.2, 7.1_

- [ ] 4.1.2 实现重复检测
  - 数据库查询
  - 缓存查询
  - 缓存更新
  - _Requirements: 2.2, 7.2, 7.3_

- [ ] 4.1.3 实现处理标记
  - 数据库写入
  - 缓存更新
  - 事务管理
  - _Requirements: 2.2, 7.1_

- [ ]* 4.1.4 幂等性属性测试
  - **Property 5: Idempotency Hash Consistency**
  - **Validates: Requirements 2.2, 7.1**

- [ ]* 4.1.5 重复检测属性测试
  - **Property 6: Duplicate Detection**
  - **Validates: Requirements 2.2, 7.2, 7.3**

- [ ]* 4.1.6 幂等性集成测试
  - 测试哈希一致性
  - 测试重复检测
  - 测试处理标记

#### 4.2 事件处理器 (需完整实现)
**文件**: `pkg/services/processing/event_processor.go`

```go
type EventProcessor interface {
    ProcessEvent(ctx context.Context, event BlockchainEvent) error
    ProcessBatch(ctx context.Context, events []BlockchainEvent) error
    GetProcessingStats() ProcessingStats
}

type EventProcessorImpl struct {
    mq           MQPlugin
    db           DatabasePlugin
    cache        CachePlugin
    idempotency  IdempotencyService
    validator    EventValidator
    logger       Logger
    metrics      MetricsCollector
    workerPool   *WorkerPool
}
```

**子任务**:
- [ ] 4.2.1 实现事件验证
  - 结构验证
  - 数据验证
  - 错误处理
  - _Requirements: 2.1_

- [ ] 4.2.2 实现事件消费
  - 从 MQ 消费
  - 批量处理
  - 错误处理
  - _Requirements: 2.1_

- [ ] 4.2.3 实现事件存储
  - 数据库写入
  - 事务管理
  - 批量操作
  - _Requirements: 2.3_

- [ ] 4.2.4 实现死信队列处理
  - 验证失败处理
  - 错误日志
  - DLQ 发布
  - _Requirements: 2.4_

- [ ] 4.2.5 实现缓存更新
  - 处理后更新缓存
  - 缓存失效
  - 错误处理
  - _Requirements: 2.5_

- [ ] 4.2.6 实现数据库错误恢复
  - 事务回滚
  - 重试逻辑
  - 错误分类
  - _Requirements: 2.6, 4.5_

- [ ]* 4.2.7 事件验证属性测试
  - **Property 4: Event Validation**
  - **Validates: Requirements 2.1**

- [ ]* 4.2.8 事件存储属性测试
  - **Property 7: Event Storage**
  - **Validates: Requirements 2.3**

- [ ]* 4.2.9 数据库错误恢复属性测试
  - **Property 10: Database Error Recovery**
  - **Validates: Requirements 2.6, 4.5**

- [ ]* 4.2.10 事件处理集成测试
  - 测试完整处理流程
  - 测试错误处理
  - 测试性能

#### 4.3 缓存更新逻辑 (需完整实现)
**文件**: `pkg/services/processing/cache_updater.go`

```go
type CacheUpdater interface {
    UpdateCache(ctx context.Context, events []BlockchainEvent) error
    InvalidateCache(ctx context.Context, keys []string) error
    GetCacheStats() CacheStats
}

type CacheUpdaterImpl struct {
    cache   CachePlugin
    logger  Logger
    metrics MetricsCollector
}
```

**子任务**:
- [ ] 4.3.1 实现缓存更新
  - 批量更新
  - TTL 管理
  - 错误处理
  - _Requirements: 2.5, 3.4_

- [ ] 4.3.2 实现缓存失效
  - 选择性失效
  - 级联失效
  - 错误处理
  - _Requirements: 3.4_

- [ ] 4.3.3 实现缓存统计
  - 命中率追踪
  - 更新计数
  - 性能指标
  - _Requirements: 5.1_

- [ ]* 4.3.4 缓存更新属性测试
  - **Property 9: Cache Update After Processing**
  - **Validates: Requirements 2.5**

- [ ]* 4.3.5 缓存集成测试
  - 测试缓存更新
  - 测试缓存失效
  - 测试统计

#### 4.4 检查点 - 事件处理完成
- [ ] 幂等性服务实现完成
- [ ] 事件处理器实现完成
- [ ] 缓存更新逻辑实现完成
- [ ] 所有测试通过
- [ ] 性能基准测试

---

## Phase 12: 文档和最终化 (完整实现)

### 目标
完整实现所有文档，确保生产就绪

### 任务列表

#### 12.1 API 文档 (已完成)
- [x] OpenAPI/Swagger 文档
- [x] API 使用示例
- [x] 错误代码文档

#### 12.2 部署文档 (已完成)
- [x] 部署流程文档
- [x] 配置选项文档
- [x] 故障排除指南

#### 12.3 开发者指南 (已完成)
- [x] 插件开发文档
- [x] 测试流程文档
- [x] 代码组织文档

#### 12.4 运维指南 (需完整实现)
**文件**: `docs/guides/OPERATIONS_GUIDE.md`

**子任务**:
- [ ] 12.4.1 监控和告警指南
  - Prometheus 指标说明
  - 告警规则配置
  - 仪表板设置
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

- [ ] 12.4.2 扩展和性能调优指南
  - 水平扩展步骤
  - 性能优化建议
  - 容量规划
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6_

- [ ] 12.4.3 备份和恢复指南
  - 备份策略
  - 恢复流程
  - 灾难恢复计划
  - _Requirements: 4.5, 7.6_

- [ ] 12.4.4 故障排除指南
  - 常见问题
  - 调试技巧
  - 日志分析
  - _Requirements: 4.1, 4.2, 4.3_

#### 12.5 最终集成测试 (需完整实现)
**文件**: `test/integration/final_integration_test.go`

**子任务**:
- [ ] 12.5.1 完整工作流测试
  - 事件拉取到 API 查询
  - 错误处理和恢复
  - 多插件场景
  - _Requirements: 9.3_

- [ ] 12.5.2 性能测试
  - 吞吐量基准
  - 延迟基准
  - 资源使用基准
  - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.6, 16.7_

- [ ] 12.5.3 兼容性测试
  - 向后兼容性
  - 多版本 API 支持
  - 多格式输出支持
  - _Requirements: 13.2, 13.3, 13.4_

- [ ] 12.5.4 多客户端测试
  - 并发请求
  - 客户端断开处理
  - 不同平台支持
  - _Requirements: 14.1, 14.2, 14.7_

#### 12.6 最终检查点 - 项目完成
- [ ] 所有任务完成
- [ ] 所有测试通过
- [ ] 所有文档完成
- [ ] 代码覆盖率 > 80%
- [ ] 性能目标达成
- [ ] 生产就绪

---

## 实现时间表

### Phase 3: 消息队列 (2-3 周)
- 第 1 周: Kafka 实现
- 第 2 周: Redis 实现
- 第 3 周: ZeroMQ 实现 + 测试

### Phase 4: 事件处理 (2-3 周)
- 第 1 周: 幂等性服务
- 第 2 周: 事件处理器
- 第 3 周: 缓存更新 + 测试

### Phase 12: 文档 (1-2 周)
- 第 1 周: 运维指南 + 最终集成测试
- 第 2 周: 文档审查 + 最终检查

**总计**: 5-8 周完整实现

---

## 质量标准

### 代码质量
- [ ] 代码覆盖率 > 80%
- [ ] 无关键 linting 错误
- [ ] 所有测试通过
- [ ] 代码审查通过
- [ ] 文档更新

### 测试标准
- [ ] 单元测试覆盖所有业务逻辑
- [ ] 集成测试覆盖组件交互
- [ ] 属性测试验证正确性
- [ ] 性能测试验证基准
- [ ] 端到端测试验证工作流

### 文档标准
- [ ] 代码注释清晰
- [ ] 函数文档完整 (godoc)
- [ ] 架构文档准确
- [ ] API 文档完整
- [ ] 部署文档清晰

---

## 成功标准

### Phase 3 完成时
- ✅ 所有 MQ 插件实现完成
- ✅ 所有测试通过
- ✅ 死信队列功能验证
- ✅ 性能基准测试

### Phase 4 完成时
- ✅ 幂等性服务实现完成
- ✅ 事件处理器实现完成
- ✅ 缓存更新逻辑实现完成
- ✅ 所有测试通过
- ✅ 性能基准测试

### Phase 12 完成时
- ✅ 所有文档完成
- ✅ 最终集成测试通过
- ✅ 性能目标达成
- ✅ 生产就绪

---

## 下一步

1. **确认范围**: 确认 Phase 3/4/12 的完整实现范围
2. **设置环境**: 准备开发环境和测试基础设施
3. **开始 Phase 3**: 从 Kafka 实现开始
4. **跟踪进度**: 定期检查进度和质量指标
5. **完成后评估**: Phase 3/4/12 完成后评估是否需要实现其他 Phase

---

**计划状态**: ✅ 准备就绪  
**预计时间**: 5-8 周  
**推荐团队**: 2-3 名开发者  
**下一步**: 开始 Phase 3 实现

