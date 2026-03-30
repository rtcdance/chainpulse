# ChainPulse 代码规范指南

## 1. 代码风格

### 1.1 格式化

- 使用 `gofumpt` 进行代码格式化（比 `gofmt` 更严格）
- 行长度限制：120 字符
- 导入分组顺序：标准库 → 第三方 → 项目内部

```go
import (
    "context"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/stretchr/testify/assert"

    "chainpulse/pkg/core"
    "chainpulse/pkg/services/indexing"
)
```

### 1.2 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写，无下划线 | `indexing`, `eventbus` |
| 文件名 | 小写，下划线分隔 | `chain_indexer.go` |
| 接口名 | 动词 + er/or | `Indexer`, `Puller` |
| 结构体 | 名词 | `BlockchainEvent` |
| 方法 | 动词/动词短语 | `IndexEvents`, `PullData` |
| 常量 | PascalCase | `MaxRetries` |
| 变量 | camelCase | `eventCount` |
| 私有 | 小写开头 | `internalCache` |

### 1.3 注释规范

- 所有导出类型必须添加注释
- 注释以被注释对象的名称开头
- 使用完整句子，以句号结尾

```go
// ChainIndexer indexes blockchain events for a specific chain.
type ChainIndexer interface {
    // IndexEvents indexes the given events and returns any errors encountered.
    IndexEvents(ctx context.Context, events []*BlockchainEvent) error
}
```

## 2. 代码结构

### 2.1 包组织

```
pkg/
├── core/              # 领域接口与模型（零外部依赖）
├── services/          # 应用服务（仅依赖 core）
├── plugins/           # 单体模式适配器
├── infrastructure/    # 微服务模式适配器
└── observability/     # 可观测性工具
```

### 2.2 依赖规则

```
cmd/* → adapters → services → core
            ↓
      platform (logger, metrics, trace)
```

**禁止**：
- `pkg/services/**` 导入 `pkg/plugins/**`
- `pkg/core/**` 导入任何外部包
- 循环依赖

### 2.3 接口定义

- 在 `pkg/core` 中定义接口
- 接口名以功能命名，不以实现命名
- 保持接口精简，接口隔离原则

```go
// 好：功能命名
type EventStore interface {
    Store(ctx context.Context, event *BlockchainEvent) error
    Get(ctx context.Context, id string) (*BlockchainEvent, error)
}

// 不好：实现命名
type PostgresStore interface { ... }
```

## 3. 错误处理

### 3.1 错误包装

使用 `fmt.Errorf` 包装错误，添加上下文：

```go
if err != nil {
    return fmt.Errorf("failed to store event %s: %w", event.ID, err)
}
```

### 3.2 错误类型

定义领域错误类型：

```go
var (
    ErrEventNotFound = errors.New("event not found")
    ErrInvalidBlock  = errors.New("invalid block number")
)

// 使用 errors.Is 检查
if errors.Is(err, ErrEventNotFound) {
    // 处理未找到
}
```

### 3.3 错误检查

- 不要忽略错误
- 使用 `require.NoError(t, err)` 在测试中

```go
// 好
if err := doSomething(); err != nil {
    return err
}

// 不好
_ = doSomething()  // 忽略错误
```

## 4. 并发安全

### 4.1 互斥锁

- 使用 `sync.RWMutex` 区分读写
- 锁粒度要小
- 避免在持有锁时进行 I/O

```go
type Cache struct {
    mu    sync.RWMutex
    data  map[string]interface{}
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}
```

### 4.2 Context 使用

- 函数第一个参数使用 `ctx context.Context`
- 传递 context，不要存储在结构体中
- 使用 `ctx.Done()` 处理取消

```go
func (s *Service) Process(ctx context.Context, event *Event) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // 处理事件
    }
}
```

## 5. 测试规范

### 5.1 测试结构

```go
func TestService_Method(t *testing.T) {
    // Arrange
    mockDB := new(MockDatabase)
    svc := NewService(mockDB)

    // Act
    result, err := svc.Method(ctx, input)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 5.2 表驱动测试

```go
func TestIndexer(t *testing.T) {
    tests := []struct {
        name    string
        input   *BlockchainEvent
        wantErr bool
    }{
        {
            name:    "valid event",
            input:   &BlockchainEvent{ID: "1", BlockNumber: 100},
            wantErr: false,
        },
        {
            name:    "nil event",
            input:   nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := indexer.Index(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 5.3 Mock 使用

```go
// 定义 Mock
type MockDatabase struct {
    mock.Mock
}

func (m *MockDatabase) StoreEvent(ctx context.Context, event interface{}) error {
    args := m.Called(ctx, event)
    return args.Error(0)
}

// 使用
mockDB.On("StoreEvent", mock.Anything, mock.Anything).Return(nil)
```

## 6. 日志规范

### 6.1 日志级别

- `Debug`: 调试信息，生产环境关闭
- `Info`: 关键流程节点
- `Warn`: 可恢复错误，需要注意
- `Error`: 错误，需要处理
- `Fatal`: 致命错误，程序退出

### 6.2 结构化日志

```go
logger.Info("event indexed",
    "chain_id", event.ChainID,
    "block_number", event.BlockNumber,
    "tx_hash", event.TransactionHash.Hex(),
)

logger.Error("failed to index event",
    "error", err.Error(),
    "chain_id", chainID,
)
```

## 7. 性能优化

### 7.1 避免内存分配

```go
// 好：预分配 slice
events := make([]*BlockchainEvent, 0, expectedCount)

// 不好：动态扩容
events := []*BlockchainEvent{}
```

### 7.2 使用对象池

```go
var eventPool = sync.Pool{
    New: func() interface{} {
        return &BlockchainEvent{}
    },
}

func getEvent() *BlockchainEvent {
    return eventPool.Get().(*BlockchainEvent)
}

func putEvent(e *BlockchainEvent) {
    e.Reset()
    eventPool.Put(e)
}
```

## 8. 文档规范

### 8.1 README

每个包应包含 README.md：

```markdown
# 包名

简要描述包的功能。

## 使用示例

\`\`\`go
// 示例代码
\`\`\`

## 接口说明

| 接口 | 说明 |
|------|------|
| Interface | 描述 |
```

### 8.2 架构决策记录 (ADR)

重大决策记录在 `docs/adr/`：

```markdown
# ADR-001: 使用 Kafka 作为消息队列

## 状态
已接受

## 上下文
需要解耦 Puller 和 Indexer

## 决策
使用 Kafka 作为消息队列

## 后果
- 正：高吞吐、持久化
- 负：增加运维复杂度
```

## 9. 代码审查清单

- [ ] 代码符合本规范
- [ ] 所有测试通过
- [ ] 新增代码有测试覆盖
- [ ] 无明显的性能问题
- [ ] 错误处理完善
- [ ] 并发安全（如有需要）
- [ ] 文档已更新
- [ ] 无安全漏洞

## 10. 工具配置

### 10.1 提交前检查

```bash
# 安装 pre-commit hook
cp scripts/pre-commit.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### 10.2 常用命令

```bash
# 格式化
make fmt

# 检查
make lint

# 测试
make test

# 全部检查
make ci
```
