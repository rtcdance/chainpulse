# Solana Puller — 架构差异说明

## 背景

`SolanaPuller` 是 Solana 区块链数据拉取器，遵循与 EVM puller 相同的 `DataPullerPlugin` 接口，但底层数据模型有根本差异。

## EVM vs Solana 核心差异

| 维度 | EVM (HTTPS/WS/gRPC Puller) | Solana Puller |
|---|---|---|
| 单位 | Block（区块） | Slot（时隙） |
| 最终性 | PoS 最终性（~2 epochs） | PoH 最终性（~32 slots） |
| 交易模型 | 账户余额模型 + `from/to` | UTXO 式 + 程序指令 |
| 事件 | `event`/`log` 通过 `eth_getLogs` | `Program` 日志通过 `getProgramAccounts` |
| 地址格式 | `common.Address` (20 bytes hex) | `common.Address` (32 bytes base58) |
| Token 标准 | ERC-20/721/1155 | SPL Token / SPL NFT |
| 重组检测 | 区块哈希比较 | 区块哈希 + 银行哈希 |

## 索引器设计影响

### 1. 程序（Program）取代智能合约

Solana 使用"程序"而非"智能合约"。事件不是独立的 log 对象，而是程序执行日志的一部分。索引器需要：

```
Solana 交易 → 指令 → 程序日志 → 日志解析
```

而 EVM 是：

```
EVM 交易 → Receipt → Log → ABI 解码
```

### 2. Slot 而非 BlockNumber

Solana `Slot` 是时间单位，但交易可以跨 slot 处理。当前实作使用 `currentSlot` 而非 `blockNumber`。调用者在 TranslateToBlockchainEvent 中将 slotNumber 映射到 `BlockchainEvent.BlockNumber`。

### 3. 重组范围更大

Solana 的 PoH 可能在分叉时回滚 1000+ slot。当前 `BaseDataPullerPlugin` 的 checkpoint/idempotency 机制足够处理，但 `confirmation depth` 建议从 EVM 的 12 区块改为 Solana 的 32 slot。

### 4. 地址长度

go-ethereum 的 `common.Address` 是 [20]byte（EVM 地址）。
Solana 地址是 32 字节的 ed25519 公钥。

当前 `SolanaPuller` 将 Solana 地址填充/截断为 `common.Address` 以复用现有数据结构。
**这不是长远的解决方案**——生产系统应该区分这两种地址类型。

## 当前限制

1. **SPL Token 支持**: 当前没有解析 SPL Token 的 Transfer/Approve 事件。需要实现 Solana 程序的 CPI (Cross-Program Invocation) 日志解析。
2. **geyser 插件**: 生产 Solana 索引通常使用 Geyser 插件（gRPC streaming）而非轮询 RPC。当前实现的 HTTP 轮询模式在延迟和吞吐量上都不适合生产环境。
3. **版本化交易**: Solana v0 版本化交易（地址查找表）当前未处理。

## 参考实现

Solana puller 提供了一个最小可行实现，展示了：

- `Core.DataPullerPlugin` 接口的非 EVM 实现
- SlotNumber → BlockNumber 的转换映射
- Solana 特有 JSON-RPC 方法的封装（`getSlot`、`getBlock`、`getProgramAccounts`）

完整生产级实现需要补全上述限制中的项目。
