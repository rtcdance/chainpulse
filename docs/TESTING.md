# Testing Best Practices

ChainPulse 测试体系包含三层：单元测试、集成测试、E2E 测试。

## 测试金字塔

```
    /\
   /  \     E2E Tests (少量)
  /----\    - 完整链路验证
 /      \   - 使用 docker-compose
/--------\
   ||||
  /----\    Integration Tests (中等)
 /      \   - 组件交互验证
/--------\  - 使用 TestContainers
   ||||
  /----\    Unit Tests (大量)
 /      \   - 纯内存，快速执行
/--------\  - Mock 外部依赖
```

> **注意**: 当前集成测试使用 `docker-compose` 管理外部依赖。
> `testcontainers-go` 已提出依赖审批（见 `docs/project/DEPENDENCY_APPROVAL.md`），
> 审批通过后将迁移到原生 Go 容器管理，消除 `docker-compose` 的预启动需求。

## 目录结构

```
test/
├── unit/              # 单元测试（与源码同目录的 _test.go）
├── integration/       # 集成测试
│   ├── fixtures/      # 测试数据
│   ├── mocks/         # Mock 实现
│   └── helpers/       # 测试辅助函数
├── e2e/               # 端到端测试
│   ├── scenarios/     # 测试场景定义
│   └── data/          # E2E 测试数据
└── contracts/         # 契约测试
```

## 标签使用

```bash
# 只运行单元测试（默认）
go test ./...

# 运行集成测试
go test -tags=integration ./test/integration/...

# 运行 E2E 测试
go test -tags=e2e ./test/e2e/...

# 运行所有测试
go test -tags="integration e2e" ./...
```

## 测试辅助工具

### testify

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
    "github.com/stretchr/testify/mock"
)

// assert - 失败继续
assert.Equal(t, expected, actual)

// require - 失败停止
require.NoError(t, err)

// suite - 测试套件
type IndexerTestSuite struct {
    suite.Suite
    db *sql.DB
}

func (s *IndexerTestSuite) SetupTest() {
    // 每个测试前执行
}
```

### TestContainers

```go
import "github.com/testcontainers/testcontainers-go/modules/postgres"

func TestWithPostgres(t *testing.T) {
    ctx := context.Background()

    container, err := postgres.Run(ctx,
        "postgres:15-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)
    defer container.Terminate(ctx)

    connStr, err := container.ConnectionString(ctx)
    // 使用真实数据库测试
}
```

### 属性测试 (Property-based Testing)

```go
import "github.com/leanovate/gopter"

func TestEventProperties(t *testing.T) {
    parameters := gopter.DefaultTestParameters()
    properties := gopter.NewProperties(parameters)

    properties.Property("event ID is unique", prop.ForAll(
        func(blockNum uint64, logIdx uint) bool {
            event := generateEvent(blockNum, logIdx)
            // 验证属性
            return len(event.ID) > 0
        },
        gen.UInt64(),
        gen.UInt(),
    ))

    properties.TestingRun(t)
}
```

## Mock 最佳实践

```go
// 1. 定义 Mock

type MockDatabase struct {
    mock.Mock
}

func (m *MockDatabase) StoreEvent(ctx context.Context, event interface{}) error {
    args := m.Called(ctx, event)
    return args.Error(0)
}

// 2. 在测试中使用
func TestIndexer(t *testing.T) {
    mockDB := new(MockDatabase)
    mockDB.On("StoreEvent", mock.Anything, mock.Anything).Return(nil)

    indexer := NewIndexer(mockDB)
    err := indexer.Index(ctx, event)

    assert.NoError(t, err)
    mockDB.AssertExpectations(t)
}
```

## 契约测试

```go
// test/contracts/mq_contract_test.go
func MQContractTest(t *testing.T, mq core.MQPlugin) {
    t.Run("publish and subscribe", func(t *testing.T) {
        ctx := context.Background()
        received := make(chan []byte, 1)

        err := mq.Subscribe(ctx, "test", func(msg []byte) {
            received <- msg
        })
        require.NoError(t, err)

        err = mq.Publish(ctx, "test", []byte("hello"))
        require.NoError(t, err)

        select {
        case msg := <-received:
            assert.Equal(t, "hello", string(msg))
        case <-time.After(time.Second):
            t.Fatal("timeout")
        }
    })
}

// 验证不同实现行为一致
func TestMemoryMQContract(t *testing.T) {
    MQContractTest(t, NewMemoryMQ())
}

func TestKafkaMQContract(t *testing.T) {
    // 使用 TestContainer 启动 Kafka
    MQContractTest(t, NewKafkaMQ(container.Addr()))
}
```

## 调试技巧

```bash
# 单个测试详细输出
go test -v -run TestName ./...

# 失败的测试详细输出
go test -v -run TestName ./... 2>&1 | less

# 使用 delve 调试
dlv test ./pkg/services/indexing -- -test.run TestIndexEvents

# 在 VS Code 中设置断点调试
# .vscode/launch.json 已配置
```

## 覆盖率报告

```bash
# 生成覆盖率报告
make test-coverage

# 查看 HTML 报告
open build/coverage/coverage.html

# 查看覆盖率最低的包
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | sort -k3 -n | head -20
```
