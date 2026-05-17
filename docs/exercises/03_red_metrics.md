# Exercise 3: 为 RPC 调用添加 RED 指标

## 目标

理解 RED (Rate/Errors/Duration) 可观测性，为 `getBlockTimestamp` 方法添加指标记录。

## 前置知识

- `red_metrics.go` — REDRecorder, metric 常量
- `https_jsonrpc_puller.go` — getLogs 中的 RED 用法

## 背景

RED 指标是生产系统的三大支柱：

| 指标 | 问题 | 意义 |
|------|------|------|
| Rate | "有多频繁？" | 容量规划 |
| Errors | "失败了多少？" | 可靠性 |
| Duration | "有多慢？" | 延迟分析 |

## 任务

### 任务 1: 理解现有的 RED 集成

在 `https_jsonrpc_puller.go` 中找到 `getLogs` 方法，观察如何用 `redRecorder` 记录指标：

```go
if p.redRecorder != nil {
    if err != nil {
        p.redRecorder.RecordRPCError("eth_getLogs", p.ChainID(), core.ClassifyErrorCode(err), elapsed)
    } else {
        p.redRecorder.RecordRPCCall("eth_getLogs", p.ChainID(), elapsed)
    }
}
```

### 任务 2: 为 getBlockTimestamp 添加 RED

找到 `fetchBlockTimestamps` 或 `getBlockTimestamp` 方法，添加 RED 指标记录：

- 成功时: `RecordRPCCall("eth_getBlockByNumber", chainID, elapsed)`
- 失败时: `RecordRPCError("eth_getBlockByNumber", chainID, errorCode, elapsed)`

### 任务 3: 验证

运行测试验证指标被正确记录:

```bash
go test -v -run "TestREDRecorderRecordRPCCall|TestREDRecorderRecordRPCError" ./pkg/observability/
```

### 任务 4: 在运行时验证

启动 dev 环境，发送一些请求，检查 `/metrics` 输出:

```bash
bash scripts/dev/dev.sh start:real
sleep 15
curl -s http://localhost:8081/metrics | grep -E "chainpulse_rpc"
```

## 预期输出

```
chainpulse_rpc_calls_total{method="eth_getLogs",chain_id="sepolia",status="ok"} 5
chainpulse_rpc_duration_seconds{method="eth_getLogs",chain_id="sepolia"} ...
```

如果添加了 `getBlockByNumber` 的 RED，还应该看到：

```
chainpulse_rpc_calls_total{method="eth_getBlockByNumber",chain_id="sepolia",status="ok"} 1
```

## 加分题

为 `getLatestBlockNumber` 方法也添加 RED 指标记录（方法名: "eth_blockNumber"）。
