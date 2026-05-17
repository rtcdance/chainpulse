# 练习: 扩展 EventBus

## 任务 1: 添加取消订阅功能

为 `EventBus` 添加 `Unsubscribe(topic string, ch chan any)` 方法。

提示:
- 使用 `eb.mu.Lock()` 保护 subscribers map
- 遍历并删除匹配的 channel

## 任务 2: 实现背压保护

当 channel 满时，当前实现会阻塞发布者。修改 `Publish` 方法：
- 如果 channel 满，跳过该订阅者并记录警告
- 使用 `select` 的 `default` 分支实现

```go
select {
case ch <- event:
default:
    fmt.Println("subscriber slow, skipping")
}
```

## 任务 3: 测试 goroutine 泄漏

验证订阅者退出后 channel 是否被正确清理：
1. 订阅并启动 goroutine
2. 调用 Unsubscribe
3. 等待后检查 goroutine 数量是否减少

## 参考答案方向

查看 `pkg/core/eventbus.go` 中的 `Unsubscribe()` 方法实现。
