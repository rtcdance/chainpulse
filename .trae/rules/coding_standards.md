# ChainPulse 编码规范

本文档定义了 Go 代码的具体编码规范。

## 代码风格

### 导入顺序
```go
import (
    // 1. 标准库
    "context"
    "fmt"
    
    // 2. 第三方库
    "github.com/ethereum/go-ethereum"
    "go.uber.org/zap"
    
    // 3. 项目内部包
    "chainpulse/pkg/core"
    "chainpulse/pkg/plugins"
)
```

### 错误处理
```go
// ✅ 正确：包装错误并添加上下文
if err != nil {
    return fmt.Errorf("failed to process block %d: %w", blockNum, err)
}

// ❌ 错误：直接返回
if err != nil {
    return err
}
```

### Context 使用
```go
// ✅ 正确：context 作为第一个参数
func ProcessBlock(ctx context.Context, blockNum uint64) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // 处理逻辑
}

// ❌ 错误：context 不是第一个参数
func ProcessBlock(blockNum uint64, ctx context.Context) error {
    // ...
}
```

### 接口定义
```go
// ✅ 正确：接口在消费者端定义
type BlockProcessor interface {
    Process(ctx context.Context, block *Block) error
}

// ✅ 正确：小接口
type Reader interface {
    Read(p []byte) (n int, err error)
}

// ❌ 错误：大而全的接口
type Service interface {
    DoA() error
    DoB() error
    DoC() error
    // ... 太多方法
}
```

### 结构体初始化
```go
// ✅ 正确：使用字段名
svc := &Service{
    client:    client,
    db:        db,
    cache:     cache,
}

// ❌ 错误：位置参数
svc := &Service{client, db, cache}
```

## 并发模式

### Goroutine 启动
```go
// ✅ 正确：传递 context
go func(ctx context.Context) {
    select {
    case <-ctx.Done():
        return
    case <-ticker.C:
        // 处理
    }
}(ctx)

// ❌ 错误：无取消机制
go func() {
    for {
        // 无限循环
    }
}()
```

### Channel 使用
```go
// ✅ 正确：发送方关闭
func producer(ctx context.Context, ch chan<- Event) {
    defer close(ch)
    for {
        select {
        case <-ctx.Done():
            return
        case ch <- event:
        }
    }
}

// ✅ 正确：接收方检查关闭
for event := range ch {
    process(event)
}
```

## 测试规范

### 表驱动测试
```go
func TestService_Process(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "success",
            input:   "valid",
            want:    "result",
            wantErr: false,
        },
        {
            name:    "error",
            input:   "invalid",
            want:    "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // 测试逻辑
        })
    }
}
```

### Mock 使用
```go
// 使用 testify/mock
type MockDB struct {
    mock.Mock
}

func (m *MockDB) Get(key string) (string, error) {
    args := m.Called(key)
    return args.String(0), args.Error(1)
}
```

## 日志规范

### 结构化日志
```go
// ✅ 正确：结构化字段
logger.Info("block processed",
    zap.Uint64("block", blockNum),
    zap.String("chain", chainID),
    zap.Duration("duration", elapsed),
)

// ❌ 错误：字符串拼接
logger.Info(fmt.Sprintf("processed block %d", blockNum))
```

### 日志级别
- `DEBUG`: 详细调试信息（生产环境关闭）
- `INFO`: 关键业务事件
- `WARN`: 可恢复的异常
- `ERROR`: 需要关注的错误

## 配置管理

### 环境变量
```go
// ✅ 正确：使用 viper
type Config struct {
    DatabaseURL string `mapstructure:"DATABASE_URL"`
    RedisURL    string `mapstructure:"REDIS_URL"`
}

func LoadConfig() (*Config, error) {
    viper.SetEnvPrefix("CHAINPULSE")
    viper.AutomaticEnv()
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

## 代码注释

### 何时添加注释
- 复杂算法的解释
- 非显而易见的设计决策
- 公共 API 的文档

### 何时不添加注释
- 显而易见的代码
- 可以通过命名表达的意图
- 重复代码逻辑的说明

```go
// ✅ 正确：解释为什么
// We use a 12-block confirmation depth to match Ethereum's 
// finality guarantee for high-value transactions.
const Confirmations = 12

// ❌ 错误：描述是什么
// This is a constant for confirmations
const Confirmations = 12
```
