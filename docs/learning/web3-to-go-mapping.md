# Web3 → Go 概念映射指南

> 本文档面向有 Web3 开发经验（Solidity / ethers.js / web3.py）的 Go 转型者，
> 用可运行的 chainpulse 代码示例说明 Web3 概念如何在 Go 中实现。

---

## 一、核心概念映射总表

| Web3 概念 | Go 对应 | chainpulse 位置 |
|-----------|---------|-----------------|
| **合约事件 → struct** | `type Xxx struct { ... }` + json tag | [pkg/core/blockchain_models.go](file://../../pkg/core/blockchain_models.go) |
| **keccak256 事件签名** | `crypto.Keccak256Hash([]byte(sig))` | [pkg/core/event_signature.go](file://../../pkg/core/event_signature.go) |
| **ABI 解码** | `abi.UnpackIntoInterface()` | [pkg/services/decoder/event_decoder.go](file://../../pkg/services/decoder/event_decoder.go) |
| **RPC 批量调用** | `errgroup.WithContext` 并发 goroutine | [pkg/services/indexing/multi_chain_indexer.go](file://../../pkg/services/indexing/multi_chain_indexer.go) |
| **Event Listener (web3.js)** | `channel` + goroutine | EventBus subscribe |
| **`eth_call` 超时** | `context.WithTimeout` | [pkg/infrastructure/rpc/failover_client.go](file://../../pkg/infrastructure/rpc/failover_client.go) |
| **Solidity `require`** | `if !errors.Is(err, ErrUnauthorized)` | [pkg/core/errors.go](file://../../pkg/core/errors.go) |
| **event emitter 订阅** | `EventBus.Subscribe()` 接口 | [pkg/core/plugin.go](file://../../pkg/core/plugin.go) |
| **链重组的 finality 等待** | `time.Ticker` + `select/case` | [pkg/services/reorg/reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go) |
| **Gas profiling** | `go test -bench=. -benchmem` | benchmark test files |
| **debug_traceTransaction** | `net/http/pprof` + `go tool pprof` | 内置 `runtime/pprof` |

---

## 二、数据结构映射

### 2.1 Solidity event → Go struct

**Solidity:**
```solidity
event Transfer(address indexed from, address indexed to, uint256 value);
```

**Go 等价（在 chainpulse 中的实现）：**
```go
// pkg/core/blockchain_models.go
type BlockchainEvent struct {
    ID               string         `json:"id"`
    ChainID          string         `json:"chain_id"`
    BlockNumber      uint64         `json:"block_number"`
    EventSignature   string         `json:"event_signature"`
    ContractAddress  common.Address `json:"contract_address"`
    TransactionHash  common.Hash    `json:"transaction_hash"`
    DecodedData      map[string]any `json:"decoded_data"`
    // ...
}
```

**学习要点：**
- `indexed` 参数 = Go 中作为独立字段存储
- Solidity 的 `address` = Go 的 `common.Address`（go-ethereum 库）
- `uint256` 范围超过 Go 的 `uint64`，需用 `*big.Int`
- Go 的 struct tag (`json:"..."`) 对应 Solidity ABI 的字段名

### 2.2 keccak256 事件签名计算

**JS/ethers.js:**
```js
const sig = ethers.keccak256(
  ethers.toUtf8Bytes("Transfer(address,address,uint256)")
);
```

**Go:**
```go
// pkg/core/event_signature.go
import "github.com/ethereum/go-ethereum/crypto"

func GetEventSignature(eventDef string) common.Hash {
    return crypto.Keccak256Hash([]byte(eventDef))
}

// 使用
sig := GetEventSignature("Transfer(address,address,uint256)")
```

**对比：** 完全等价——都是对签名字符串做 keccak256。

---

## 三、并发模型映射

### 3.1 RPC 批量请求：Promise.all → errgroup

**JS/ethers.js:**
```js
const results = await Promise.all(
  blockNumbers.map(n => provider.getBlock(n))
);
```

**Go（chainpulse 实测）：**
```go
// pkg/services/indexing/multi_chain_indexer.go
import "golang.org/x/sync/errgroup"

func indexMultipleChains(ctx context.Context, chains []core.BlockchainConfig) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(3) // 最多 3 个并发

    for _, chain := range chains {
        chain := chain // Go 闭包陷阱：必须重新绑定
        g.Go(func() error {
            return indexChain(ctx, chain)
        })
    }
    return g.Wait() // 等价 Promise.all()
}
```

**核心差异：**
| 特性 | JS Promise.all | Go errgroup |
|------|---------------|-------------|
| 错误语义 | 第一个 reject 立即抛异常 | 第一个 error 取消其他 goroutine |
| 并发数控制 | `p-limit` 库 | `g.SetLimit(n)`（Go 1.22+） |
| 取消传播 | AbortController | `ctx.Done()` channel |
| 闭包陷阱 | 无（let 块级作用域） | 必须 `chain := chain` 重新绑定 |

### 3.2 Event Listener → Channel + Goroutine

**JS:**
```js
contract.on("Transfer", (from, to, value, event) => {
    console.log(from, to, value);
});
```

**Go chainpulse:**
```go
// EventBus 基于 channel 发布订阅
eventBus.Subscribe(ctx, "Transfer", func(event any) {
    evt := event.(core.BlockchainEvent)
    logger.Info("received transfer",
        core.LogKeySender, evt.DecodedData["from"],
        core.LogKeyRecipient, evt.DecodedData["to"],
    )
})
```

**学习要点：**
- JS 的回调是异步单线程，Go 的 goroutine 是多线程并发的
- 需要 `sync.RWMutex` 保护共享状态（见 [reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go)）
- `chan` 需要选择缓冲大小（无缓冲 = 同步、有缓冲 = 异步队列）

### 3.3 setTimeout → time.After + select

**JS:**
```js
setTimeout(() => {
    retryRequest();
}, 5000);
```

**Go:**
```go
// 带 context 取消的重试等待
select {
case <-time.After(5 * time.Second):
    retryRequest()
case <-ctx.Done():
    return ctx.Err() // 支持取消
}
```

**优势：** Go 的 `select` + `<-ctx.Done()` 天然支持优雅退出，无需手动 `clearTimeout`。

---

## 四、错误处理映射

### 4.1 Solidity require → Go error + sentinel errors

**Solidity:**
```solidity
require(msg.sender == owner, "Not authorized");
require(block.timestamp < deadline, "Expired");
```

**Go chainpulse:**
```go
// pkg/core/errors.go - 预定义 sentinel errors
var (
    ErrUnauthorized    = NewSystemError(ErrorTypePermanent, ErrorCodeValidation, "unauthorized", nil)
    ErrDeadlineExceeded = NewSystemError(ErrorTypeTransient, ErrorCodeTimeout, "deadline exceeded", nil)
)

func checkAuth(ctx context.Context, user string) error {
    if user != owner {
        return ErrUnauthorized
    }
    if time.Now().After(deadline) {
        return ErrDeadlineExceeded
    }
    return nil
}
```

### 4.2 错误分类（Transient / Permanent / Critical）

chainpulse 实现了三级错误分类，对应 Web3 中不同类型的失败：

```go
// pkg/core/errors.go - ClassifyError 函数
func ClassifyError(err error) ErrorType {
    // 1. SystemError（所有预定义 sentinel）
    var sysErr *SystemError
    if errors.As(err, &sysErr) {
        return sysErr.Type
    }
    // 2. net.Error（网络重试）
    var netErr net.Error
    if errors.As(err, &netErr) {
        return ErrorTypeTransient
    }
    // 3. context.DeadlineExceeded
    if errors.Is(err, context.DeadlineExceeded) {
        return ErrorTypeTransient
    }
    // 默认永久
    return ErrorTypePermanent
}
```

| 分类 | 含义 | 处理 | Web3 类比 |
|------|------|------|-----------|
| Transient | 可重试 | 指数退避 + 熔断 | RPC 429 限流、网络超时 |
| Permanent | 不可重试 | 记录日志 + 跳过 | `eth_call` revert、ABI 不匹配 |
| Critical | 需立即关注 | 告警 + 退出 | 数据损坏、共识失败 |

### 4.3 try-catch → error wrapping

**JS:**
```js
try {
    await fetchBlock(blockNumber);
} catch (err) {
    throw new Error(`fetch failed at ${blockNumber}: ${err.message}`);
}
```

**Go:**
```go
// 使用 %w 包装错误，保留原始链
func processBlock(ctx context.Context, num uint64) error {
    block, err := fetchBlock(ctx, num)
    if err != nil {
        return fmt.Errorf("process block %d: %w", num, err)
    }
    // 调用者可以用 errors.Is 检查原始错误
    if errors.Is(err, ErrBlockNotFound) {
        // 特殊处理
    }
    return nil
}
```

**学习要点：**
- Go 无异常机制，`error` 是普通返回值
- `%w` 保留错误链，`errors.Is/As` 解包检查
- 链上每个环节都应该包装错误添加上下文

---

## 五、接口 / 插件系统映射

### 5.1 Solidity interface → Go interface + DI

**Solidity:**
```solidity
interface IPlugin {
    function name() external view returns (string memory);
    function start() external;
    function stop() external;
}
```

**Go chainpulse:**
```go
// pkg/core/plugin.go
type Plugin interface {
    Name() string
    Version() string
    Initialize(config Config) error
    Start() error
    Stop() error
    Health() error
}

// 依赖注入（编译时选择实现）
type ChainIndexer struct {
    db      DatabasePlugin   // 接口类型，运行时注入具体实现
    puller  DataPullerPlugin  // 接口类型
    eventBus EventBus         // 接口类型
}
```

**核心差异：**
- Solidity interface 用于合约互操作，Go interface 用于依赖反转
- Go 可以在编译时用 Wire DI 自动组装依赖
- Solidity 没有"构造函数注入"概念，Go 是主流模式

---

## 六、内存管理映射

### 6.1 Gas 优化 → 堆/栈分配 + Benchmark

**Solidity 的 gas 优化：**
```solidity
// 使用 memory 而非 storage
uint256[] memory tempArray = new uint256[](10);
```

**Go 的性能分析：**
```bash
# 内存分配分析
go test -bench=. -benchmem ./pkg/services/indexing/

# 输出示例
BenchmarkIndexBlock-8    500    2.1ms/op    1024 B/op    5 allocs/op
#                                          ^^^^^^^^      ^^^^^^^^^^
#                                          每次操作分配   分配次数
```

**学习要点：**
- Go 的栈分配（零开销）对应 Solidity 的 `memory`
- Go 的堆分配对应 Solidity 的 `storage`（持久化）
- `-benchmem` 是 Go 的 gas profiler

---

## 七、测试映射

### 7.1 Hardhat 测试 → go test + testify

**Hardhat/JS:**
```js
describe("Transfer", () => {
    it("should emit Transfer event", async () => {
        await expect(tx).to.emit(contract, "Transfer")
            .withArgs(alice, bob, 100);
    });
});
```

**Go chainpulse:**
```go
// pkg/services/processor/event_processor_test.go 风格示例
func TestProcessEvent(t *testing.T) {
    tests := []struct {
        name    string
        event   *core.BlockchainEvent
        wantErr bool
    }{
        {"valid transfer", mockTransfer("alice", "bob", 100), false},
        {"nil event", nil, true},
        {"empty network", &core.BlockchainEvent{}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := processor.ProcessEvent(ctx, tt.event)
            if (err != nil) != tt.wantErr {
                t.Errorf("ProcessEvent() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 八、部署 / 运维映射

### 8.1 .env → pkg/env + struct

**JS:**
```js
require('dotenv').config();
const rpcUrl = process.env.RPC_URL || 'http://localhost:8545';
```

**Go chainpulse:**
```go
// pkg/env/env.go - 统一环境变量读取（支持 CHAINPULSE_ 前缀回退）
import "chainpulse/pkg/env"

rpcURL := env.Get("RPC_URL", "http://localhost:8545")
port := env.GetInt("PORT", 8080)
debug := env.GetBool("DEBUG", false)
```

### 8.2 docker-compose → 单体 / 微服务双模部署

chainpulse 同一套代码支持两种模式：

```bash
# 单体模式（开发）
DEPLOYMENT_MODE=monolithic ./chainpulse run

# 微服务模式（生产）
DEPLOYMENT_MODE=microservice ./chainpulse run --service=event-processor
```

这展示了 Go 的一个核心优势：**编译一次，多种部署方式**。

---

## 九、学习路径建议

1. **先跑 playground**（零依赖）→ 感受 Go 的事件处理流程
   ```bash
   go run ./cmd/playground/
   curl http://localhost:9099/generate
   curl http://localhost:9099/events
   ```

2. **读 `pkg/core/plugin.go`** → 理解 Go interface（对应 Solidity interface）

3. **读 `pkg/core/errors.go`** → 理解 Go error 处理（对应 Solidity require/revert）

4. **读 `pkg/services/reorg/reorg_handler.go`** → 理解并发 + 互斥锁（对应链重组的竞争条件处理）

5. **读 `pkg/services/indexing/multi_chain_indexer.go`** → 理解 errgroup 并发（对应 js Promise.all）

6. **写一个 integration** → 在 `pkg/integrations/` 创建自己的事件索引器

---

## 十、常见陷阱

| 陷阱 | 说明 | 解决 |
|------|------|------|
| **闭包捕获循环变量** | `for _, chain := range chains { go func() { use(chain) }() }` | `chain := chain` 重新绑定 |
| **nil map 写入 panic** | `var m map[string]int; m["key"] = 1` | `make(map[string]int)` |
| **defer 在循环中** | 循环中 defer 只在函数结束时执行 | 提取为子函数 |
| **goroutine 泄漏** | 启动后永不退出的 goroutine | 用 `ctx.Done()` + `select` |
| **Big.Int 比较** | `==` 不适用于 `*big.Int` | 用 `Cmp()` 方法 |
| **锁拷贝** | `sync.Mutex` 值拷贝会导致死锁 | 总是用指针接收者 |