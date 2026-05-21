# 练习: 链重组处理

## 任务 1: 理解重组检测逻辑

运行示例观察:
```
Old chain: 100(0xaaa) → 101(0xbbb) → 102(0xccc) → 103(0xddd)
New chain: 100(0xaaa) → 101(0xbbb) → 102(0xccc) → 103(0xDDD) ← 重组点
```

问题: 为什么需要检查 block hash 而非只是 block number?

## 任务 2: 添加幂等性保护

修改 `HandleReorg`，添加幂等性键 `(chain_id, block_number, log_index)` 的去重逻辑。

提示:
```go
type IdempotencyKey struct {
    ChainID     string
    BlockNumber uint64
    LogIndex    uint32
}

var processed = make(map[IdempotencyKey]bool)
```

## 任务 3: 测试深度重组

模拟超过 `maxRollback` 的深度重组:
- 当前 checkpoint: 1000
- 重组点: 500 (需要回滚 500 块)
- maxRollback: 120

验证: 应该拒绝回滚并返回错误。

## 参考答案方向

查看 `pkg/services/reorg/reorg_handler.go` 的 `HandleReorg` 方法，
特别注意 `maxRollback` 保护和 `IdempotencyInvalidator` 的使用。
