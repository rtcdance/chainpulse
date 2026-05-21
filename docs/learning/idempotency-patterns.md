# 区块链事件幂等性 (Idempotency) — Go 实现模式

## Web3 背景

在区块链索引系统中，**幂等性**是最关键的安全保证之一。由于以下原因，同一个事件可能被多次处理：

1. **节点重连** — RPC 断开后重连，可能重新拉取已处理过的区块
2. **链重组 (reorg)** — 区块被回滚后重新索引，产生相同的事件
3. **消息队列 at-least-once 语义** — Kafka/Redis 可能重复投递
4. **进程重启** — puller 重启后从 checkpoint 恢复，边界事件可能重复

## Go 实现要点

### 1. 确定性哈希 (Deterministic Hash)

事件的自然唯一键由 `(chain_id, block_number, tx_hash, log_index)` 组成。
这些字段在链上是不可变的，任何派生字段（如 EventName、ContractAddress）都不参与哈希计算。

```go
// pkg/core/event_hash.go
func ComputeEventHash(event *BlockchainEvent) string {
    hashInput := fmt.Sprintf("%s:%d:%s:%d",
        event.ChainID,
        event.BlockNumber,
        event.TransactionHash.Hex(),
        event.LogIndex,
    )
    hash := sha256.Sum256([]byte(hashInput))
    return hex.EncodeToString(hash[:])
}
```

### 2. sync.RWMutex — 读多写少的并发优化

`sync.RWMutex` 允许**多个 goroutine 同时读取** `processedHashes` map，
只有在写入（标记已处理）时才需要独占锁。这对于读密集的 `IsDuplicate` 调用特别重要。

```go
type DefaultIdempotencyService struct {
    mu              sync.RWMutex
    processedHashes map[string]bool
    processedCount  int64
    duplicateCount  int64
}

func (s *DefaultIdempotencyService) IsDuplicate(ctx context.Context, hash string) (bool, error) {
    s.mu.RLock()            // 读锁 — 允许多个并发读取
    defer s.mu.RUnlock()
    return s.processedHashes[hash], nil
}

func (s *DefaultIdempotencyService) MarkProcessed(ctx context.Context, hash string) error {
    s.mu.Lock()             // 写锁 — 独占
    defer s.mu.Unlock()
    if s.processedHashes[hash] {
        s.duplicateCount++
        return nil
    }
    s.processedHashes[hash] = true
    s.processedCount++
    return nil
}
```

### 3. 不需要 TTL/过期

与传统缓存系统不同，区块链事件的幂等性存储**不需要 TTL**：

- 每个事件的唯一键永不会再出现
- 链上事件是永久唯一的
- 数据库层唯一约束作为最终保障

### 4. 测试要点

幂等性测试需要覆盖以下场景：

| 测试场景 | 验证内容 |
|---------|---------|
| 基本去重 | IsDuplicate → MarkProcessed → IsDuplicate 返回 true |
| 哈希确定性 | 同一事件多次 GenerateHash 返回相同哈希 |
| 跨链隔离 | ChainID 不同的相同 block/tx/log 产生不同哈希 |
| 并发安全 | 多个 goroutine 同时 MarkProcessed，无竞态 |
| 清空后重用 | Clear 后可重新处理先前的事件 |
| 运行状态检查 | 未 start 时操作返回错误 |

## Web3 → Go 概念对照

| 概念 | Solidity / EVM | Go |
|------|---------------|-----|
| 事件唯一键 | `keccak256(chain_id, block, txhash, logIndex)` | `sha256.Sum256([]byte(...))` |
| 不可变哈希 | `bytes32` | `[32]byte` / hex string |
| 线程安全 | 单线程 EVM，无此概念 | `sync.RWMutex` |
| 并发去重 | 不需要（EVM 顺序执行） | goroutine 安全是核心需求 |
| 幂等性保证 | 链上天然幂等 | 需要显式实现 |

## 学习路径建议

1. 先阅读 `pkg/core/event_hash.go` — 理解哈希算法
2. 然后阅读 `pkg/services/processor/idempotency.go` — 理解服务实现
3. 运行测试：`cd pkg/services/processor && go test -v -run TestIdempotency`
4. 用 race detector 验证：`go test -race -run TestIdempotency`