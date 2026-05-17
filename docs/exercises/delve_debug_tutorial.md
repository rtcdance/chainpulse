# Delve 分步调试教程：Web3+Go 转型者专用

> 通过 5 个渐进式调试课程，理解 ChainPulse 的核心数据流。每课 15-30 分钟。

## 前置条件

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 确保项目依赖已安装
go mod download
```

---

## 课程 1: 事件总线 (EventBus) 分发流程

**学习重点**: goroutine 调度、channel 通信、sync.RWMutex 使用

**调试目标**: 追踪一个事件从 Publish 到所有 subscriber 接收的完整过程。

### 步骤 1: 启动调试会话

```bash
dlv debug ./examples/01_event_bus/ -- --listen=:2345 --headless=false
```

### 步骤 2: 设置断点

```
(dlv) break main.go:55      # EventBus.Publish
(dlv) break main.go:30      # 订阅者 1 的 handler
(dlv) break main.go:39      # 订阅者 2 的 handler (大额转账告警)
```

### 步骤 3: 观察执行流

```
(dlv) continue    # 运行到第一个断点
(dlv) print event # 查看当前发布的事件
(dlv) goroutines  # 查看活跃 goroutine
(dlv) stack       # 查看调用栈
```

### 步骤 4: 思考题

- Publish 是同步还是异步的？为什么？
- 如果 channel buffer 满了会发生什么？
- RWMutex 在 Publish 和 Subscribe 中的使用差异？

**参考答案**: Publish 是同步的——它遍历所有 subscriber channel 并等待写入完成。这是**背压**机制：如果 subscriber 处理慢，publisher 会被阻塞。

---

## 课程 2: ABI 事件签名与解码

**学习重点**: keccak256 哈希、ABI 编码规则、[]byte 处理

**调试目标**: 观察 topic0 计算和 data 字段解码过程。

### 步骤 1: 启动调试会话

```bash
dlv debug ./examples/02_event_signature/
```

### 步骤 2: 设置断点

```
(dlv) break main.go:20      # EventSignature 函数入口
(dlv) break main.go:25      # keccak256Hash 调用前
(dlv) break main.go:43      # Transfer 签名验证点
```

### 步骤 3: 观察变量

```
(dlv) continue
(dlv) print sig             # 查看签名字符串 "Transfer(address,address,uint256)"
(dlv) print len(data)       # 查看哈希输出长度 (应为 32 字节)
(dlv) print hash.Hex()      # 查看十六进制输出
```

### 步骤 4: 验证已知哈希

Transfer 事件的已知哈希:
```
0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
```

在调试器中验证:
```
(dlv) print transferSig.Hex() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
```

### 步骤 5: 思考题

- 为什么 indexed string 只存储哈希而非原始值？
- bytes32 类型的 indexed 参数为什么可以直接从 topic 读取？
- 如果事件参数超过 3 个 indexed 会怎样？

---

## 课程 3: 错误处理与重试逻辑

**学习重点**: errors.Is/As、错误包装链、指数退避

**调试目标**: 观察不同类型错误的处理路径。

### 步骤 1: 启动调试会话

```bash
dlv debug ./examples/03_error_handling/
```

### 步骤 2: 设置断点

```
(dlv) break main.go:60      # CallWithRetry 函数入口
(dlv) break main.go:72      # ClassifyError 调用点
(dlv) break main.go:75      # switch 错误类型分支
```

### 步骤 3: 观察错误分类

```
(dlv) continue              # 运行到第一次调用
(dlv) print err             # 查看返回的错误
(dlv) print errType         # 查看分类结果 (0=Transient, 1=Permanent, 2=Critical)
(dlv) print attempt         # 查看当前重试次数
```

### 步骤 4: 手动触发不同错误路径

在调试器中修改变量测试不同路径:

```
(dlv) call caller.callCount = 1   # 强制产生 Transient 错误
(dlv) continue
(dlv) call caller.callCount = 0   # 强制产生 Permanent 错误 (blockNum=0)
(dlv) continue
```

### 步骤 5: 思考题

- `errors.As(err, &rpcErr)` 和 `errors.Is(err, sentinel)` 的区别？
- 指数退避的公式是什么？为什么不用固定间隔？
- context.Done() 在重试循环中起什么作用？

---

## 课程 4: 链重组 (Reorg) 处理

**学习重点**: 状态回滚、一致性保护、maxRollback 防护

**调试目标**: 观察重组检测、回滚和重新索引的完整流程。

### 步骤 1: 启动调试会话

```bash
dlv debug ./examples/04_reorg_detection/
```

### 步骤 2: 设置断点

```
(dlv) break main.go:53      # DetectReorg 调用
(dlv) break main.go:58      # HandleReorg 调用
(dlv) break main.go:35      # 回滚循环 (delete block hashes)
(dlv) break main.go:44      # 重新索引循环
```

### 步骤 3: 观察重组流程

```
(dlv) continue              # 索引 blocks 100-105
(dlv) print rh.checkpoint   # 应为 105
(dlv) print rh.blockHashes  # 查看所有已索引的块

(dlv) continue              # 检测到重组
(dlv) print rollbackDepth   # 回滚深度 (应为 2)
(dlv) print rh.blockHashes  # 回滚后的状态
```

### 步骤 4: 验证 maxRollback 保护

```
(dlv) continue              # 运行到 Step 4
(dlv) print rollbackDepth   # 应为 20 (超过 maxRollback=5)
(dlv) print err             # 查看保护错误信息
```

### 步骤 5: 思考题

- 为什么需要 maxRollback 保护？什么场景会触发？
- 回滚后为什么要删除 blockHashes？
- 如果多个重组同时发生，如何处理？

**参考生产代码**: [pkg/services/reorg/reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go)

---

## 课程 5: Context 优雅关闭

**学习重点**: context 传播、goroutine 生命周期管理、signal 处理

**调试目标**: 观察从 SIGINT 到所有组件停止的完整流程。

### 步骤 1: 启动调试会话 (支持信号)

```bash
dlv debug ./examples/05_context_graceful_shutdown/
```

### 步骤 2: 设置断点

```
(dlv) break main.go:85      # signal 接收点
(dlv) break main.go:33      # Worker 的 ctx.Done() 分支
(dlv) break main.go:66      # Indexer 的 ctx.Done() 分支
```

### 步骤 3: 观察启动流程

```
(dlv) continue              # 运行到 goroutine 启动
(dlv) goroutines            # 查看创建的 goroutine
(dlv) print worker.done     # 查看 channel 状态
```

### 步骤 4: 触发关闭 (新终端)

打开另一个终端，发送 SIGINT:
```bash
kill -SIGINT <pid>
```

或在 Delve 中手动调用 cancel:
```
(dlv) call cancel()
(dlv) continue
```

### 步骤 5: 观察关闭流程

```
(dlv) print ctx.Err()       # 应为 context.Canceled
(dlv) print worker.done     # 检查 channel 是否关闭
(dlv) goroutines            # 确认 goroutine 已退出
```

### 步骤 6: 思考题

- 为什么需要 `signal.Notify` 和 `context.Cancel` 两个机制？
- 如果 worker 的 cleanup() 耗时很长会怎样？
- 如何设置关闭超时时间？

---

## Delve 常用命令速查

| 命令 | 缩写 | 作用 |
|------|------|------|
| `break <loc>` | `b` | 设置断点 |
| `continue` | `c` | 继续执行到下一个断点 |
| `next` | `n` | 执行下一行 (不进入函数) |
| `step` | `s` | 执行下一行 (进入函数) |
| `print <expr>` | `p` | 打印表达式值 |
| `goroutines` | `goroutines` | 列出所有 goroutine |
| `stack` | `bt` | 显示调用栈 |
| `locals` | `locals` | 显示局部变量 |
| `whatas <expr>` | `wh` | 显示表达式的类型 |
| `exit` | `q` | 退出调试器 |

### 高级调试技巧

```bash
# 条件断点 (只在特定条件下触发)
(dlv) break main.go:50 if attempt > 2

# 追踪点 (不暂停执行，只打印)
(dlv) trace main.go:50 "event=%v", event

# 数据断点 (变量被修改时暂停)
(dlv) watch rh.checkpoint

# 查看结构体所有字段
(dlv) print -multiline rh

# 查看函数调用历史
(dlv) frame 2
```

---

## 一键启动脚本

```bash
#!/bin/bash
# scripts/dlv-lesson-1.sh
LESSON=${1:-1}

case $LESSON in
  1) dlv debug ./examples/01_event_bus/ ;;
  2) dlv debug ./examples/02_event_signature/ ;;
  3) dlv debug ./examples/03_error_handling/ ;;
  4) dlv debug ./examples/04_reorg_detection/ ;;
  5) dlv debug ./examples/05_context_graceful_shutdown/ ;;
  *) echo "Usage: $0 {1|2|3|4|5}" ;;
esac
```

使用方法:
```bash
chmod +x scripts/dlv-lesson-*.sh
./scripts/dlv-lesson-1.sh  # 启动课程 1
```

---

## VS Code 配置

在 `.vscode/launch.json` 中添加:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Lesson 1: EventBus",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/examples/01_event_bus/",
      "showLog": true
    },
    {
      "name": "Debug Lesson 2: Event Signature",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/examples/02_event_signature/",
      "showLog": true
    }
  ]
}
```

然后选择 `Debug Lesson 1: EventBus` 启动调试，设置断点后点击运行即可。

---

## 进阶：调试生产代码

完成 5 个示例课程后，可以调试真实的生产代码:

```bash
# 调试事件解码器
dlv debug ./cmd/monolithic/chainpulse/ \
  -- -listen=:8080

# 在关键位置设置断点
(dlv) break pkg/core/event_decoder.go:50
(dlv) break pkg/services/reorg/reorg_handler.go:100
(dlv) break pkg/services/indexing/chain_indexer.go:200
```

或使用 playground 模式 (零外部依赖):
```bash
dlv debug ./cmd/playground/
```
