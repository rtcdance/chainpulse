# Reorg — 区块链重组处理

## Web3 概念

**区块链重组（Reorg）** 是区块链网络中的一种现象：矿工/验证者同时产生两个有效区块，一部分节点先看到区块 A，另一部分先看到区块 B。当其中一条链累积更多工作量/质押时，较短链上的区块被"重组"掉，其上记录的所有交易和事件也随之失效。

重组是 Web3 开发的核心概念，面试高频题：
- 重组深度受什么影响？（以太坊 ~1-7 块，Polygon ~128 块，BTC ~1 块）
- 如何处理重组中的数据一致性？
- 最终性（Finality）与重组概率的关系

## 检测算法

`ReorgHandler` 使用**二分查找 + 线性回退**的双阶段检测：

1. **二分查找**（`binarySearchReorg`）：在 `[lowerBound, currentBlock]` 区间内二分查找哈希分歧点，O(log n)
2. **线性回退**（`linearScanReorg`）：如果二分查找因 RPC 错误失败，从 currentBlock 向后线性扫描，O(n)
3. **安全限制**：`maxScanDepth=256` 防止无限扫描

```
reorg_handler.go:DetectReorg()
  → 对比 storedHash vs canonicalHash
  → 不一致 → findReorgBlock() 二分查找分歧点
  → 找到分歧点 → HandleReorg() 回滚 + 发布事件
```

## 事件流

```
observeEvent()                  ← puller 收到新区块
  → DetectReorg()               ← 比较哈希
  → [可选] HandleReorg()        ← 回滚事件 + 清除缓存
    → Publish("reorg-detected") ← 通知监控系统
    → Publish("reorg-rollback") ← 通知 puller 重新索引
```

## Go 要点

| 模式 | 位置 | 说明 |
|------|------|------|
| 接口依赖 | `reorg_handler.go:18-36` | `core.DatabasePlugin` + `core.EventBus` + `core.BlockHashProvider` |
| 二分搜索 | `reorg_handler.go:318-361` | 标准二分查找 + context cancellation 检查 |
| 可选依赖 | `SetBlockHashProvider()` | 默认查数据库，生产注入 RPC 提供者 |
| 幂等性清除 | `RollbackEvents()` | 回滚时清除幂等性记录，防止重复判定 |

## 学习路径

1. 读 `DetectReorg()` 理解哈希比较逻辑
2. 读 `findReorgBlock()` → `binarySearchReorg()` 理解二分查找算法
3. 读 `HandleReorg()` → `RollbackEvents()` 理解回滚流程
4. 看 `m1a_runtime_wiring.go:125-137` 了解如何注入 BlockHashProvider

## 关键文件

| 文件 | 功能 |
|------|------|
| `reorg_handler.go` | 核心检测 + 回滚（521 行） |
| `reorg_handler_test.go` | 单元测试 |
| `reorg_handler_bench_test.go` | 基准测试 |