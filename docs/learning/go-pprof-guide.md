# Go pprof 性能分析实战指南

> 面向 Web3 开发者：用 chainpulse 的真实代码，学会定位 Go 程序的性能瓶颈。

---

## 零、快速开始（30 秒上手）

```bash
# 1. 启动 playground（已内置 pprof）
cd chainpulse && go run ./cmd/playground/main.go

# 2. 另开终端，抓取 30 秒 CPU profile
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30

# 3. 在浏览器打开 http://localhost:8080 查看火焰图

# 4. 抓取当前堆内存快照
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap

# 5. 查看 goroutine 数量
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -30
```

> **前提**：playground 的 pprof 端点已内置于 `:6060`，无需额外配置。

---

## 一、pprof 是什么？

`pprof` 是 Go 标准库自带的性能分析工具，无需安装任何第三方依赖。它能分析：

| Profile 类型 | 回答什么问题 | 典型场景 |
|-------------|-------------|---------|
| **CPU profile** | 程序时间花在哪里？ | 某个 RPC 调用慢，哪行代码占 CPU 最多 |
| **Heap profile** | 内存分配在哪里？ | GC 压力大，哪个结构体在堆上分配过多 |
| **Goroutine profile** | 当前有多少 goroutine？ | goroutine 泄漏诊断 |
| **Allocs profile** | 内存分配频率如何？ | 高频分配导致 GC 停顿 |
| **Block profile** | 哪些操作在等锁/IO？ | mutex 竞争分析 |
| **Mutex profile** | 互斥锁竞争程度？ | 并发写入性能瓶颈 |

---

## 二、CPU 性能分析

### 2.1 工作流程

```
运行中程序                pprof 工具               浏览器
    │                       │                      │
    ├─ :6060/debug/pprof/  ─┤                      │
    │  (暴露端点)            │                      │
    │                       ├─ go tool pprof ──────┤
    │                       │  (抓取+解析)           │
    │                       │                      ├─ 火焰图
    │                       │                      ├─ Top 函数列表
    │                       │                      └─ 调用图
```

### 2.2 抓取 CPU profile

```bash
# 方式一：交互模式（推荐）
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 方式二：直接启动 Web UI
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30

# 方式三：先保存再分析（适合生产环境）
curl -o cpu.prof http://localhost:6060/debug/pprof/profile?seconds=60
go tool pprof -http=:8080 cpu.prof
```

**建议**：`seconds` 参数至少设为 30 秒，时间太短采样不充分，火焰图会有偏差。生产环境建议 60 秒。

### 2.3 交互模式常用命令

进入 `go tool pprof` 交互模式后：

```
(pprof) top10          # CPU 占用 Top 10 函数
(pprof) top10 -cum     # 按累积时间排序（含子调用）
(pprof) list findReorg # 查看指定函数的逐行耗时
(pprof) web            # 生成调用图（需要 graphviz）
(pprof) peek malloc    # 查看与 malloc 相关的调用者
```

### 2.4 在代码中编程式生成 CPU profile

```go
import (
    "os"
    "runtime/pprof"
)

func main() {
    f, _ := os.Create("cpu.prof")
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()

    // ... 你的业务逻辑 ...
}
```

---

## 三、内存性能分析

### 3.1 Heap profile vs Allocs profile

| 维度 | `heap` | `allocs` |
|------|--------|----------|
| 含义 | 当前**存活**对象的分配栈 | **所有**内存分配的栈（含已释放） |
| 类比 | 当前余额 | 累计消费 |
| 用途 | 找内存泄漏 | 找高频分配点 |

```bash
# 当前堆上的对象（存活）
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap

# 累计所有分配（包含已 GC 回收的）
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/allocs
```

### 3.2 内存分析关注点

在 pprof Web UI 中切换到 **SAMPLE** 视图，关注四个指标：

```
alloc_space   → 累计分配内存总量（找大头）
alloc_objects → 累计分配对象数（找高频分配）
inuse_space   → 当前正在使用的内存（找泄漏）
inuse_objects → 当前存活对象数（找泄漏）
```

### 3.3 Web3 场景下的典型内存问题

```go
// ❌ 坏实践：循环内动态扩容 slice
events := make([]core.BlockchainEvent, 0)
for block := startBlock; block <= endBlock; block++ {
    batch := rpc.FetchLogs(block)        // 每次返回 100 条
    events = append(events, batch...)     // 频繁触发 slice 扩容 + 拷贝
}

// ✅ 好实践：预分配容量
totalEstimate := (endBlock - startBlock + 1) * avgLogsPerBlock
events := make([]core.BlockchainEvent, 0, totalEstimate)
```

```go
// ❌ 坏实践：hex 字符串频繁拼接
event.ID = "0x" + fmt.Sprintf("%x", hash) + "_" + fmt.Sprintf("%d", index)

// ✅ 好实践：使用 strings.Builder 或预分配
var sb strings.Builder
sb.Grow(66 + len(chainID))
sb.WriteString("0x")
sb.WriteString(hex.EncodeToString(hash[:]))
```

---

## 四、Goroutine 分析

### 4.1 检测 goroutine 泄漏

```bash
# 查看所有 goroutine 的栈信息
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutine.txt

# Web UI 可视化
go tool pprof -http=:8082 http://localhost:6060/debug/pprof/goroutine
```

### 4.2 goroutine 泄漏的常见元凶

```go
// ❌ goroutine 泄漏：channel 永远等不到数据
func watchBlocks(ctx context.Context, ch <-chan *core.Block) {
    go func() {
        for {
            select {
            case block := <-ch:        // 如果 ch 永远不会 close 或发送
                process(block)
            case <-ctx.Done():
                return                 // ✅ 有退出路径，但依赖 ctx 取消
            }
        }
    }()
}

// ✅ 防御式写法：加上超时
go func() {
    for {
        select {
        case block := <-ch:
            process(block)
        case <-ctx.Done():
            return
        case <-time.After(30 * time.Second):
            log.Println("[warn] no blocks received in 30s")
            return
        }
    }
}()
```

### 4.3 诊断命令速查

```bash
# 查看 goroutine 数量变化趋势（每秒采样）
watch -n1 'curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 | head -1'

# 输出 goroutine 数量随时间变化
for i in $(seq 1 60); do
  count=$(curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 | head -1 | awk '{print $1}')
  echo "$(date +%H:%M:%S) goroutines=$count"
  sleep 1
done
```

### 4.4 编程式 goroutine profile

```go
import "runtime/pprof"

func dumpGoroutines(path string) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()
    return pprof.Lookup("goroutine").WriteTo(f, 2)
}
```

---

## 五、chainpulse 实战：reorg 二分查找 vs 线性扫描

### 5.1 背景

chainpulse 的 [reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go) 在检测链重组时，需要找到分叉点。

**线性扫描**（[linearScanReorg](file://../../pkg/services/reorg/reorg_handler.go#L312-L345)）：从当前块号递减扫描，O(n) 复杂度。

**二分查找**（[binarySearchReorg](file://../../pkg/services/reorg/reorg_handler.go#L287-L310)）：每次对半查找，O(log n) 复杂度。

### 5.2 性能基准测试

[benchmark 代码](file://../../pkg/services/reorg/reorg_handler_bench_test.go) 对 1000 个区块的 reorg 检测进行了对比：

```bash
# 运行 benchmark
cd chainpulse
go test -bench=. -benchmem -benchtime=3s ./pkg/services/reorg/

# 同时生成 CPU profile
go test -bench=BenchmarkBinarySearchReorg -benchmem -cpuprofile=cpu.prof ./pkg/services/reorg/
go tool pprof -http=:8080 cpu.prof

# 同时生成内存 profile
go test -bench=BenchmarkBinarySearchReorg -benchmem -memprofile=mem.prof ./pkg/services/reorg/
go tool pprof -http=:8081 mem.prof
```

### 5.3 预期效果

| 算法 | 时间复杂度 | 1000 个区块的迭代次数 | RPC 调用次数 |
|------|-----------|---------------------|------------|
| 线性扫描 | O(n) | ≤ 1000 | ≤ 1000 |
| 二分查找 | O(log n) | ≈ 10 | ≈ 10 |

在链上有 1000 个区块需要验证时，二分查找比线性扫描 **少约 100 倍的 RPC 调用**。

### 5.4 从 profile 中学到什么

运行 benchmark 分析后重点看：

1. **火焰图中最宽的"平顶"**：那是 CPU 时间占比最高的函数
2. `binarySearchReorg` 中 `GetBlockHash` 的调用次数：如果 RPC 调用比预期多，说明二分查找的条件判断有问题
3. 内存分配热点：`make()` 或 `new()` 在热路径上

---

## 六、如何阅读火焰图

### 6.1 基本概念

```
     main()
    ┌──┴──┐
  foo()  bar()
  ┌─┴─┐  ┌─┴─┐
 baz() qux() q() r()
```

- **X 轴（宽度）**：该函数占 CPU 时间的比例，**越宽越慢**
- **Y 轴（高度）**：调用栈深度，从下往上是调用链
- **颜色**：通常随机，**不代表性能好坏**

### 6.2 三步诊断法

```
第一步：找"平顶"
  → 最宽的横向矩形 = CPU 时间占比最高的函数
  → 优化这个函数，收益最大

第二步：找"瘦高塔"
  → 垂直方向很窄但很高的栈 = 调用链很深但耗时很少
  → 通常是递归或过度封装，可忽略或用内联优化

第三步：点击函数
  → 火焰图会放大该函数及其子调用
  → 用 top / list 查看逐行耗时
```

### 6.3 常见模式识别

| 火焰图形状 | 可能的含义 | 行动 |
|-----------|-----------|-----|
| 单个宽平顶 | 某个函数占用大量 CPU | 优化算法复杂度 |
| 大量等宽小方块 | 大量函数调用开销 | 减少间接调用 |
| 锯齿状 | 递归调用 | 检查递归终止条件 |
| 有 `runtime.mallocgc` 宽块 | 频繁内存分配 | 预分配、对象池 |
| 有 `runtime.gcBgMarkWorker` | GC 占 CPU | 减少堆分配 |
| 有 `syscall.Syscall` 宽块 | 系统调用瓶颈 | 批量化、缓存 |

---

## 七、Web3 Go 代码的常见性能瓶颈

### 7.1 RPC 调用过多

```go
// ❌ N+1 查询：每条事件独立查一次 receipt
for _, txHash := range txHashes {
    receipt, _ := client.TransactionReceipt(ctx, txHash)  // 一次网络往返
    events = append(events, receipt.Logs...)
}

// ✅ 批量查询（eth_getLogs 一次返回多个区块的日志）
filter := ethereum.FilterQuery{
    FromBlock: big.NewInt(int64(startBlock)),
    ToBlock:   big.NewInt(int64(endBlock)),
}
logs, _ := client.FilterLogs(ctx, filter)
```

### 7.2 ABI 解码开销

```go
// ❌ 每条事件都重新解析 ABI
for _, log := range logs {
    parsedABI, _ := abi.JSON(strings.NewReader(contractABI))
    _ = parsedABI.UnpackIntoInterface(&event, eventName, log.Data)
}

// ✅ 启动时解析一次，运行时复用
var parsedABI abi.ABI  // 包级变量，init() 中初始化
func init() {
    parsedABI, _ = abi.JSON(strings.NewReader(contractABI))
}
// 热路径直接使用 parsedABI
```

### 7.3 big.Int 分配开销

```go
// ❌ 每次循环 new 一个新的 big.Int
for i := 0; i < 10000; i++ {
    val := new(big.Int).SetUint64(i)  // 堆分配
    process(val)
}

// ✅ 复用 big.Int
val := new(big.Int)
for i := 0; i < 10000; i++ {
    val.SetUint64(i)
    process(val)
}
```

### 7.4 hex/bytes 转换开销

```go
// ❌ 二次转换
raw := common.FromHex(hexStr)         // hex → []byte
hash := common.BytesToHash(raw)       // []byte → Hash

// ✅ 直接转换
hash := common.HexToHash(hexStr)       // hex → Hash，一次完成
```

### 7.5 JSON 序列化瓶颈

```go
// ❌ 默认的 json.Marshal 使用反射
data, _ := json.Marshal(event)         // 大量反射开销

// ✅ 热路径使用预分配的 encoder + 复用 buffer
var buf bytes.Buffer
enc := json.NewEncoder(&buf)
for _, event := range events {
    buf.Reset()
    enc.Encode(event)
    write(buf.Bytes())
}

// ✅ 更极致的方案：使用 easyjson / jsoniter
```

---

## 八、性能分析工具速查表

```bash
# === CPU ===
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30
go test -bench=. -cpuprofile=cpu.prof ./pkg/... && go tool pprof -http=:8080 cpu.prof

# === 内存 ===
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap
go test -bench=. -memprofile=mem.prof ./pkg/... && go tool pprof -http=:8081 mem.prof

# === Goroutine ===
go tool pprof -http=:8082 http://localhost:6060/debug/pprof/goroutine
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutine.txt

# === 阻塞分析（需要先启用） ===
curl http://localhost:6060/debug/pprof/block?seconds=30 > block.prof
# 注意：需要在代码中调用 runtime.SetBlockProfileRate(1) 才能采样

# === Mutex 竞争 ===
curl http://localhost:6060/debug/pprof/mutex?seconds=30 > mutex.prof
# 注意：需要 runtime.SetMutexProfileFraction(1)

# === 对比两次 profile（找回归） ===
go tool pprof -http=:8083 -base=before.prof after.prof

# === 下载 profile 原始文件 ===
curl -o cpu.prof 'http://localhost:6060/debug/pprof/profile?seconds=30'
curl -o heap.prof http://localhost:6060/debug/pprof/heap

# === 全量 30 秒分析 ===
go tool pprof -http=:8080 \
  http://localhost:6060/debug/pprof/profile?seconds=30
```

---

## 九、进阶技巧

### 9.1 对比分析（找回归）

```bash
# 改动前抓一份
curl -o before.prof 'http://localhost:6060/debug/pprof/profile?seconds=30'

# 改动后抓一份
curl -o after.prof 'http://localhost:6060/debug/pprof/profile?seconds=30'

# 对比：红色 = 变慢了，绿色 = 变快了
go tool pprof -http=:8083 -base=before.prof after.prof
```

### 9.2 自定义 profile 端点

```go
import (
    "net/http"
    "net/http/pprof"
    "runtime"
)

func registerDebugHandlers(mux *http.ServeMux) {
    // 启用 block 和 mutex profile
    runtime.SetBlockProfileRate(1)
    runtime.SetMutexProfileFraction(1)

    mux.HandleFunc("/debug/pprof/", pprof.Index)
    mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
    mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
```

### 9.3 生产环境安全

```go
// 生产环境：pprof 应该只监听 localhost，不对外暴露
go func() {
    log.Fatal(http.ListenAndServe("localhost:6060", nil))
}()

// 绝不要这样做：
// log.Fatal(http.ListenAndServe("0.0.0.0:6060", nil))  <- 危险！
```

### 9.4 Go execution tracer（深入分析）

当 pprof 不够用时，使用 execution tracer 看 goroutine 调度、GC 事件、阻塞事件：

```bash
curl -o trace.out 'http://localhost:6060/debug/pprof/trace?seconds=5'
go tool trace -http=:8084 trace.out
```

---

## 十、chainpulse 项目中的相关资源

| 资源 | 路径 |
|------|------|
| pprof 端点（playground） | [cmd/playground/main.go](file://../../cmd/playground/main.go) — 端口 `:6060` |
| reorg 二分查找实现 | [pkg/services/reorg/reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go#L287-L310) |
| reorg 线性扫描实现 | [pkg/services/reorg/reorg_handler.go](file://../../pkg/services/reorg/reorg_handler.go#L312-L345) |
| reorg benchmark 测试 | [pkg/services/reorg/reorg_handler_bench_test.go](file://../../pkg/services/reorg/reorg_handler_bench_test.go) |
| Web3 → Go 概念映射 | [docs/learning/web3-to-go-mapping.md](file://../web3-to-go-mapping.md) |