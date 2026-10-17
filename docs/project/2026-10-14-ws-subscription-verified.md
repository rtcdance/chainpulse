# WebSocket-JSONRPC 采集：从"未验证"变成"已验证"

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 证据：`build/acceptance/<sha>/ws-subscription/{SUMMARY.txt,anvil-tests-all.log}`
- 来源：**支持矩阵的 `unverified` 行**——把它当待办清单逐条处理

## 待办来自哪里

上一轮建的支持矩阵里，这一行是 `unverified`：

| capability | status | 当时写的理由 |
| --- | --- | --- |
| WebSocket-JSONRPC collection | unverified | 同一个 puller 在 node URL 为 `ws://`/`wss://` 时切换到 WS 订阅，带轮询兜底；**没有测试起一个 WS endpoint 去走那条分支** |

而 README 的能力表把它当作**已承诺的采集协议**之一（"Data Collection: Multiple protocols (HTTPS-JSONRPC, **WebSocket-JSONRPC**)"）。
所以这是一条**已对外声明、但从未执行过**的能力。

## 为什么这条分支特别值得测

```go
// https_jsonrpc_puller.go:485
if strings.HasPrefix(p.nodeURL, "ws://") || strings.HasPrefix(p.nodeURL, "wss://") {
    p.LogInfo("Poll: attempting WebSocket subscription mode", "url", p.nodeURL)
    if err := p.pollWithSubscription(ctx); err != nil {
        p.LogWarn("WebSocket subscription failed, falling back to polling", ...)
    } else {
        return nil
    }
}
return p.pollWithTicker(ctx)
```

**失败会静默降级**：订阅建立不起来时，puller **照样工作**，只是变成轮询。
⇒ 一个坏掉的 WS 路径，从外部看**和正常的一模一样，只是慢一点**。
这比"未测试"更糟——**它不会红，只会退化成另一种实现**。

## 先确认前提（Anvil 同时提供 HTTP 与 WS）

```
HTTP  eth_chainId            -> 0x7a69
WS    eth_chainId            -> 0x7a69
WS    eth_subscribe newHeads -> 0x77ee11bf27e54080a8ae7be2e6b2fa08f48f6f45c9660376b0b7e2e3cc41677a
```

## 测试断言三件事

新增 `TestAnvil_WebSocketSubscription`（`test/e2e/anvil_test.go`，`-tags=e2e`）：

1. **订阅确实建立了** —— 用 `th.NewTestLoggerWithCapture()` 捕获日志，断言出现
   `Poll: subscribed to newHeads via WebSocket`。
2. **没有走兜底** —— 捕获的日志里**不得**出现 `WebSocket subscription failed`。
   这一条是关键：只看"能工作"是不够的，因为兜底也能工作。
3. **推送真的送达了** —— 用 `evm_mine` 造一个区块，然后断言 puller 的
   checkpoint（`GetLastBlockNumber()`）推进到该高度。订阅路径只在**处理完一个 head 之后**
   才 `SetLastBlockNumber`，所以这证明的是**推送到达并被处理**，而不只是"subscribe 调用返回成功"。

## 结果

```
--- PASS: TestAnvil_Connect
--- PASS: TestAnvil_GetBlockHeaders
--- PASS: TestAnvil_FilterLogs
--- PASS: TestAnvil_PullerIntegration
--- PASS: TestAnvil_WebSocketSubscription (0.37s)     ← 新增
--- PASS: TestAnvil_ParentHashChainVerification
--- PASS: TestAnvilStartupVerify
ok  	github.com/rtcdance/chainpulse/test/e2e	29.829s
```

⇒ 支持矩阵里这一行已从 **unverified** 改为 **verified**。

## 反例：证明这条测试会红

按本会话的纪律，一条只会绿的测试不算证据。把测试里的 URL 换回 `http://`（即让 `ws://` 分支不成立），
puller 走的是 ticker 路径，测试**必须失败**：

```
--- FAIL: TestAnvil_WebSocketSubscription (10.25s)
    anvil_test.go:268: no WebSocket subscription was established; captured logs:
      [data puller plugin started RPC chain ID verified HTTPS-JSONRPC puller started
       Poll: starting polling loop Poll: fetching latest block... ...]
```

注意捕获到的日志里是 **`Poll: starting polling loop`**——正是兜底路径。
反例跑完已还原（`grep` 确认 `strings.Replace(rpcURL, "http://", "ws://", 1)` 仍在）。

## 诚实边界

1. **只验证了 `ws://`，没验证 `wss://`**：分支判断两者前缀相同，但 TLS 路径需要证书，
   本地没有起。断言的是"`ws://` 会进订阅模式"，不是"TLS 也一定行"。
2. **只验证了订阅建立与 head 送达**，没有验证**断线重连**。代码里有一处
   `newHeads subscription lost` 的处理，但没测。
3. **`evm_mine` 是 Anvil/Hardhat 扩展方法**，非标准 EVM RPC；测试只在 Anvil 下跑，可接受。
4. **这不是 §9 的场景**：E01/E03 仍未验收（需要双 Anvil 与 fixture 部署）。本轮做的是
   **把一条已声明的采集能力从"从未执行"变成"有测试且会红"**。
5. **矩阵仍只是文档**：`verified` 行靠人维护，没有自动一致性检查。

## 复现

```bash
# 前置：Anvil 同时提供 HTTP 与 WS
anvil --port 18545 --silent &
cast rpc --rpc-url ws://127.0.0.1:18545 eth_chainId

# 测试
PATH="$HOME/.foundry/bin:$PATH" \
  go test -count=1 -v -tags=e2e -run TestAnvil_WebSocketSubscription ./test/e2e/

# 全部 7 个带 tag 的 Anvil 测试
PATH="$HOME/.foundry/bin:$PATH" go test -count=1 -v -tags=e2e -run TestAnvil ./test/e2e/
```
