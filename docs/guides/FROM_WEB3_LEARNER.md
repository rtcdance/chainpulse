# Web3 → Go 转型者学习指南

如果你是 Solidity/JavaScript 背景的开发者，正在学习 Go 并想理解链上数据索引——或者你是 Go 开发者想深入 Web3 事件处理——这份指南帮你建立 ChainPulse 代码库中的概念映射。

---

## 1. 概念映射：Web3 → ChainPulse

### 区块与交易

| Web3 概念 | Go 实现 | 文件 |
|---|---|---|
| `block.number` | `Block.Number uint64` | `pkg/core/blockchain_models.go` |
| `block.hash` | `Block.Hash common.Hash` | 同上 |
| `block.timestamp` | `Block.Timestamp int64` (Unix 秒) | 同上 |
| `tx.hash` | `Transaction.Hash common.Hash` | 同上 |
| `tx.from` / `tx.to` | `Transaction.From/To common.Address` | 同上 |
| `tx.gasPrice` | `Transaction.GasPrice *big.Int` | 同上 |
| EIP-1559 `maxFeePerGas` | `Transaction.MaxFeePerGas *big.Int` | 同上 |
| EIP-4844 Blob Tx | `Transaction.Type == TxBlob` + `BlobSidecar` | 同上 |

**关键学习点**: Go 使用 `common.Hash` (\[32\]byte) 和 `common.Address` (\[20\]byte) 而非字符串。使用 `common.Hash{}` 检查零值而非 `null`。

### 事件日志

```solidity
// Solidity
event Transfer(address indexed from, address indexed to, uint256 value);
```

```go
// ChainPulse Go 表示
type BlockchainEvent struct {
    EventSignature  common.Hash      // keccak256("Transfer(address,address,uint256)")
    ContractAddress common.Address   // 合约地址
    EventTopic      []common.Hash    // topic[0..3]
    EventData       []byte           // 非索引参数的 ABI 编码
    DecodedData     map[string]interface{} // 解码后的键值对
    TypedData      interface{}       // 强类型解码（如 *ERC20Transfer）
    Removed         bool             // 链重组时为 true
}
```

#### 从 Go 中解码事件的完整路径

```
Blockchain RPC
  │ eth_getLogs / eth_getTransactionReceipt
  ▼
pkg/plugins/pullers/https_jsonrpc_puller.go   ← 原始 JSON-RPC 调用
  │ types.Log (go-ethereum)
  ▼
pkg/core/event_decoder.go                      ← ABI 解码
  │ BlockchainEvent
  ▼
pkg/services/decoder/contract_manager.go       ← ABI 管理
pkg/services/decoder/event_decoder.go          ← 事件解码
  │ 解码后的数据
  ▼
pkg/integrations/erc20/erc20_indexer.go        ← ERC-20 业务逻辑
```

### 链重组 (Reorg)

| Web3 概念 | Go 实现 | 文件 |
|---|---|---|
| 区块被回滚 | `ReorgDetectedMessage` + `reorg_handler.go` | `core/blockchain_models.go`、`services/reorg/reorg_handler.go` |
| 确认深度 | `EventStatus` pending→confirmed→finalized | `core/blockchain_models.go` |
| 最终性检查 | `finality_checker.go` | `services/finality/` |
| 幂等处理 | `IdempotencyInvalidator.InvalidateRange` | `core/plugin.go` |

**学习模式**: Go 使用 `enum` 风格的常量 + switch（而非继承），`EventStatus` 定义了清晰的状态机。

### ERC-4337 账户抽象

```go
type UserOperation struct {
    Sender               common.Address
    Nonce                *big.Int
    InitCode             []byte         // 工厂部署代码
    CallData             []byte         // 执行调用数据
    PaymasterAndData     []byte         // 燃料赞助
    Signature            []byte
}
```

v0.7 的 packed gas 字段用辅助方法解包：
```go
func (op *UserOperationV07) DecodeV07GasLimits() (verificationGasLimit, callGasLimit uint64)
```

**关键学习点**: 在 Solidity 中用 `abi.encodePacked` 打包 → 在 Go 中用 `big.Int` + `SetBytes` 解包。

---

## 2. Go DDD 架构——从概念到分层

ChainPulse 使用 Go 实现 DDD：

```
pkg/core/             ← 共享类型与接口（零实现逻辑）
pkg/domain/           ← 纯业务逻辑（依赖 core，不依赖其他层）
pkg/application/      ← 编排层/用例
pkg/adapters/         ← 外部世界适配器
pkg/infrastructure/   ← 具体实现（数据库、RPC 客户端等）
pkg/plugins/          ← 可插拔组件（pullers/database/cache/MQ）
pkg/services/         ← 跨领域业务逻辑
```

**DDD 层间依赖规则**（Go 的 interface 是其核心机制）：
```
core  ←  domain  ←  application  ←  adapters/infrastructure/plugins
                                                    ↑
                                    services ───────┘
```

Go 的 interface 使 DDD 自然实现：`core.Plugin` 是接口，`infrastructure` 里的具体类型实现它。转型者应关注**接口定义在 core，实现在 infrastructure** 的模式。

### Go 泛型的实际应用

```go
// pkg/core/eventbus.go
func SubscribeTyped[T any](bus EventBus, ctx context.Context, topic string, handler func(T)) (uint64, error) {
    return bus.Subscribe(ctx, topic, func(raw interface{}) {
        typed, ok := raw.(T)
        if !ok { return }
        handler(typed)
    })
}
```

使用方式：
```go
SubscribeTyped[*ERC20Transfer](bus, ctx, "erc20_transfer", func(event *ERC20Transfer) {
    // 不需要类型断言，handler 直接得到 *ERC20Transfer
})
```

---

## 3. Go 模式——从 Solidity / JS 来的人需要知道

### 初始化（零值而非 null）

```go
// Solidity 中：
// address addr;       → 0x0
// Go 中：
var addr common.Address  // 零值 = [20]byte{0...}
if addr == (common.Address{}) {  // 检查空地址
    // ...
}
```

### Error 是值（而非 revert）

```go
// 不要：
if somethingBad { return }

// 要：
if somethingBad {
    return fmt.Errorf("failed to process block %d: %w", blockNumber, err)
}
```

### Context 作为第一参数

```go
func Process(ctx context.Context, event *BlockchainEvent) error {
    select {
    case <-ctx.Done():
        return ctx.Err()  // 优雅取消
    default:
    }
    // 正常处理
}
```

### 接口组合

```go
type DatabasePlugin interface {
    Plugin
    EventReader      // 嵌入——DatabasePlugin "是一个" EventReader
    EventWriter
    BlockReader
    ReorgStatsProvider
}
```

---

## 4. 推荐学习路径（按难度排序）

### 第 1 级：运行 + 理解数据模型
1. `go run cmd/monolithic/chainpulse/main.go`（或 playground，见 Phase 4）
2. 看 `pkg/core/blockchain_models.go`——所有链上数据类型的定义位置
3. 理解 `BlockchainEvent` 的字段如何对应 Solidity event

### 第 2 级：追踪一条事件从链到库的路径
4. `pkg/plugins/pullers/https_jsonrpc_puller.go` → `PullEvents()`
5. 进入 `core/event_decoder.go` → `DecodeEvent()`
6. 进入 `services/decoder/event_decoder.go` → 真正的 ABI 解码
7. 进入 `services/indexing/chain_indexer.go` → 编入索引系统

### 第 3 级：理解韧性
8. `services/reorg/reorg_handler.go` → 如何处理链重组
9. `services/consistency/consistency_checker.go` → 数据一致性
10. `services/resilience/retry_logic.go` → 带退避的重试

### 第 4 级：集成与 DeFi
11. `integrations/erc20/erc20_indexer.go` → 代币索引
12. `integrations/uniswap/uniswap_indexer.go` → DEX 索引
13. `core/defi_primitives.go` → AMM 数学（Uniswap v2/v3）

---

## 5. 常见陷阱

| 陷阱 | 解释 | 正确做法 |
|---|---|---|
| 用 `==` 比较 `[]byte` | Go slice 不能直接比较 | `bytes.Equal(a, b)` |
| 忘记检查 `log.Removed` | 重组中的 log 会被标记为移除 | 检查 `BlockchainEvent.Removed` |
| 用字符串处理地址 | `common.Address` 是 `[20]byte` | 用 `.Hex()` 输出字符串 |
| 忽略 zero value | Go 结构体的字段默认零值 | 检查 `BlockNumber == 0` 等条件 |
| 到处用 `interface{}` | 丢失类型安全 | 用泛型 `[T any]` 或明确类型 |
| 阻塞 goroutine 不恢复 | panic 会崩掉整个进程 | 在 goroutine 内加 `recover()` |
| 不通过 `ctx.Done()` 检查 | 无法优雅关闭 | 每个阻塞操作前检查 context |

---

## 6. 推荐的扩展阅读

| 文件 | 为什么读 |
|---|---|
| `pkg/core/plugin.go` | 理解 Go 接口组合 + 可选接口模式 |
| `pkg/core/eventbus.go` | 内存事件总线的 worker pool 实现 |
| `pkg/core/errors.go` | Go 错误分类（transient/permanent/critical） |
| `pkg/infrastructure/processing/idempotency_service.go` | 幂等性的 Go 实现 |
| `pkg/plugins/pullers/data_puller.go` | 基类 + 子类复用的 Go 模式 |
| `cmd/microservices/puller/puller_execution.go` | 微服务启动全流程 |
| `docs/DEBUGGING.md` | 调试入门 + Web3 概念到断点的映射 |

---

## 7. 用调试器学习

只看代码不如单步跟踪一条 Transfer 事件从生成到响应的全过程。
`docs/DEBUGGING.md` 包含完整的**学习向断点教学**：

### 5 条调试路径

| 路径 | 学什么 | 起点断点 |
|---|---|---|
| 事件生命周期 | BlockchainEvent 所有字段 | `playground/main.go:42` |
| ABI 解码 | `[]byte` → `DecodedData` 的转换 | `core/event_decoder.go:50` |
| Reorg + 最终性 | 分叉检测、幂等处理 | `finality/finality_checker.go:30` |
| DeFi + AA | AMM 数学、账户抽象 | `playground/main.go:86`/`:130` |
| 多链路由 | ChainID 调度 | `multi_chain_puller.go:37` |

### 快速开始

```bash
# 一键启动 playground + 载入所有学习断点
dlv debug ./cmd/playground --init /tmp/dlv-learn.txt
```

`docs/DEBUGGING.md` 的 `Delve 断点脚本` 章节提供了可复制的一键断点脚本。

### VS Code 用户

1. 将 playground launch config 加入 `.vscode/launch.json`
2. 选择 `Debug Playground (Learning)` 启动
3. 打开 `cmd/playground/main.go` 手动设断点
4. `curl http://localhost:9099/generate` 触发断点

---

## 8. 全链路调试：从 Solidity 合约到 Go 断点

Playground 用 mock 事件教学，但真实学习路径是：

```
Solidity 合约 → Anvil 节点 → ChainPulse Puller → 解码 → 存储 → API
```

### 全链路调试步骤

```bash
# 1. 启动 Anvil 本地以太坊节点（:8545）
docker compose -f docker/docker-compose.yml up -d anvil-ethereum --wait

# 2. 部署合约并发送事件
bash scripts/deploy-event-emitter.sh
# 输出: Contract: 0x..., 9 events emitted

# 3. 启动 ChainPulse monolithic 模式指向 Anvil
CHAINPULSE_BLOCKCHAIN_NODE_URL=http://localhost:8545 \
CHAINPULSE_DEPLOYMENT_MODE=monolithic \
CHAINPULSE_DATABASE_TYPE=memory \
CHAINPULSE_CACHE_TYPE=memory \
CHAINPULSE_MQ_TYPE=memory \
CHAINPULSE_CHAINS=ethereum \
CHAINPULSE_START_BLOCK=0 \
dlv debug ./cmd/monolithic/chainpulse

# 4. 在调试器中设置断点（看到真实链上事件流过）
#    pkg/plugins/pullers/https_jsonrpc_puller.go:300  ← PullEvents 拉取真实区块
#    pkg/core/event_decoder.go:50                     ← ABI 解码真实事件
#    pkg/adapters/indexing/monolithic_memory_storage.go:93 ← 存储
```

### 两种调试模式对比

| | Playground (mock) | 真实链 (Anvil) |
|---|---|---|
| 启动速度 | 即时 | 需等 Docker |
| 外部依赖 | 无 | Docker |
| 数据 | 模拟的 random 事件 | 你部署的合约 + 你触发的真实事件 |
| 可控制性 | 只能生成固定模式 | 可部署任何合约、发任何交易 |
| 学习目标 | Go 数据流 | Solidity → RPC → Go 全链路 |
| ABI 解码 | 跳过（mock 已有 decoded_data） | 真实解码过程可以看到 |

### 推荐学习顺序

```
第 1 步: mock playground（Go 数据流）     ← 你已经会了
第 2 步: Anvil + real chain（全链路）     ← 现在打通
第 3 步: 修改合约，增加自定义事件        ← 理解 ABI 升级
第 4 步: 生产 RPC 接入（Alchemy/Infura） ← 实战
```
