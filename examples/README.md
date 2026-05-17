# ChainPulse 最小可运行示例

> 面向 Web3 + Go 转型者的独立学习示例。每个示例 ≤ 100 行代码，可独立运行。

## 使用方式

```bash
# 运行任意示例
go run ./examples/01_event_bus/

# 运行示例并查看输出
go run ./examples/02_event_signature/
```

## 示例列表

| 示例 | 学习重点 | 对应生产代码 |
|------|---------|-------------|
| [01_event_bus](01_event_bus/) | EventBus 发布/订阅模式 | [pkg/core/eventbus.go](file://../../pkg/core/eventbus.go) |
| [02_event_signature](02_event_signature/) | keccak256 事件签名计算 | [pkg/core/event_signature.go](file://../../pkg/core/event_signature.go) |
| [03_error_handling](03_error_handling/) | 错误分类 + 重试逻辑 | [pkg/core/errors.go](file://../../pkg/core/errors.go) |
| [04_reorg_detection](04_reorg_detection/) | 链重组检测与回滚 | [pkg/services/reorg/reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go) |
| [05_context_graceful_shutdown](05_context_graceful_shutdown/) | Context 优雅关闭 | 全局模式 |

## 学习路径

```
第 1 步: 01_event_bus          ← 理解 Go 的发布/订阅
第 2 步: 02_event_signature    ← 理解 ABI 事件签名
第 3 步: 03_error_handling     ← 理解 Go 错误处理
第 4 步: 04_reorg_detection    ← 理解链重组处理
第 5 步: 05_context_graceful_shutdown ← 理解优雅关闭
```

## 扩展练习

每个示例的 `EXERCISE.md` 包含进阶任务：
1. 修改代码并观察行为变化
2. 添加新的测试用例
3. 引入一个 bug 然后用 Delve 调试
