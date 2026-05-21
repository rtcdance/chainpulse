# 练习: Context 与优雅关闭

## 任务 1: 添加超时控制

修改 main 函数，使用 `context.WithTimeout` 替代固定的 10 秒睡眠:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

观察: 超时后 worker 和 indexer 是否正确清理?

## 任务 2: 实现级联取消

添加第二个 indexer，使其依赖第一个 indexer 的 context:

```go
ctx2, cancel2 := context.WithCancel(ctx) // ctx 取消时 ctx2 也取消
```

问题: 如果父 context 取消，子 context 会怎样?

## 任务 3: 处理 panic 恢复

在 worker 的 goroutine 中添加 `recover()`:

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Recovered from panic: %v\n", r)
        }
    }()
    // ... existing code
}()
```

然后模拟一个 panic，验证不会导致整个进程崩溃。

## 参考答案方向

查看:
- `pkg/services/resilience/graceful_shutdown.go` - 优雅关闭实现
- `pkg/core/eventbus.go` - goroutine panic 恢复
- 整个代码库中 `ctx.Done()` 的使用模式
