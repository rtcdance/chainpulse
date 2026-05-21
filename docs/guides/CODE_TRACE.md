# ChainPulse 代码追踪：一笔事件的生命周期

**目标**: 跟踪一笔 Transfer 事件从 RPC 响应到 API 可查询的完整代码路径。

---

## 追踪概述

```
RPC 节点                    ChainPulse 进程
┌────────┐   eth_getLogs    ┌──────────────────────────────────────┐
│ Anvil  │ ◄──────────────► │ puller.Poll → getLogs → ethLogToEvent│
│ Sepolia│    5s 间隔       │      ↓                              │
└────────┘                  │ eventBus.Publish("blockchain-events")│
                            │      ↓                              │
                            │ event_processor.Subscribe → Process  │
                            │      ↓                              │
                            │ SharedRuntime.ProcessBatch           │
                            │      ↓                              │
                            │ 存储 (MongoDB/PostgreSQL)            │
                            │      ↓                              │
                            │ GraphQL 查询 → API 响应              │
                            └──────────────────────────────────────┘
```

---

## Step 1: Puller 启动

### 入口点

**文件**: `cmd/monolithic/chainpulse/main.go:333`

```go
monolithicPullerRuntime, err := newMonolithicPullerRuntime(
    ctx, *coreConfig, config.BlockchainNodeURLs, chains, 
    logger, metrics, indexingDatabase, multiChainIndexer,
)
```

这里做了什么：
1. 读取 `config.BlockchainNodeURLs`（如 `http://localhost:8545`）
2. 为每条链创建一个 `HTTPSJSONRPCPuller` 实例
3. 注册 EventBus 订阅者（收到事件后转发给 processor）

### Puller 结构

**文件**: `pkg/plugins/pullers/https_jsonrpc_puller.go:23`

```
HTTPSJSONRPCPuller
  ├── ethClient *ethclient.Client    ← go-ethereum 的以太坊客户端
  ├── currentBlock uint64            ← 当前索引进度
  ├── lifecycleCtx                   ← 生命周期上下文 (R4 修复)
  ├── redRecorder *REDRecorder       ← RED 指标记录 (Phase 2)
  └── BaseDataPullerPlugin (嵌入)
       ├── eventBus core.EventBus    ← 发布事件
       ├── metricsCollector          ← 指标收集
       └── config core.Config        ← 配置
```

---

## Step 2: Poll 循环

**文件**: `pkg/plugins/pullers/https_jsonrpc_puller.go:299`

```go
func (p *HTTPSJSONRPCPuller) Poll(ctx context.Context) error {
    // 每 5 秒执行一次
    ticker := time.NewTicker(p.pollInterval)
    for {
        select {
        case <-ticker.C:
            // 1. 获取最新块号
            latestBlock, err := p.getLatestBlockNumber(ctx)
            
            // 2. 计算差异
            fromBlock := atomic.LoadUint64(&p.currentBlock) + 1
            if latestBlock <= fromBlock { continue }  // 没有新区块
            
            // 3. 分批拉取事件
            events, err := p.pullEvents(ctx, fromBlock, latestBlock)
            
            // 4. 发布到 EventBus
            for _, event := range events {
                p.eventBus.Publish(ctx, "blockchain-events", event)
            }
            
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

**关键设计决策**:
- 用 `select + ticker` 而不是 `time.Sleep` → 可以监听 `ctx.Done()` 实现优雅停止
- `currentBlock` 用 `atomic.LoadUint64` → 并发安全
- `fromBlock` 用 `lastCheckpoint + 1` → 不会重复索引已处理的块

### 2.1 getLatestBlockNumber

**文件**: `pkg/plugins/pullers/https_jsonrpc_puller.go:483`

```go
func (p *HTTPSJSONRPCPuller) getLatestBlockNumber(ctx context.Context) (uint64, error) {
    header, err := p.ethClient.HeaderByNumber(ctx, nil)  // nil = latest
    // RED 指标自动记录 (eth_blockNumber, duration)
    return header.Number.Uint64(), nil
}
```

RPC 调用: `eth_blockNumber`
返回值: 当前区块号（如 `10845346`）

### 2.2 pullEvents

**文件**: `pkg/plugins/pullers/https_jsonrpc_puller.go:199`

```go
func (p *HTTPSJSONRPCPuller) pullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
    chunkSize := uint64(1000)  // 每批最多 1000 块
    // 分批查询
    for chunkFrom < toBlock {
        chunkTo := min(chunkFrom+chunkSize-1, toBlock)
        logs, err := p.getLogs(ctx, chunkFrom, chunkTo)
        // ...
    }
}
```

**为什么分批**: 公共 RPC 节点限制单次 `eth_getLogs` 的范围。1000 块是保守值。

---

## Step 3: eth_getLogs → 原始日志

**文件**: `pkg/plugins/pullers/https_jsonrpc_puller.go:499`

```go
func (p *HTTPSJSONRPCPuller) getLogs(ctx context.Context, fromBlock, toBlock uint64) ([]types.Log, error) {
    query := ethereum.FilterQuery{
        FromBlock: big.NewInt(int64(fromBlock)),
        ToBlock:   big.NewInt(int64(toBlock)),
    }
    // 如果配置了合约地址过滤
    if addrs := p.GetConfig().ContractAddresses; len(addrs) > 0 {
        query.Addresses = addrs  // ← 这是 P2.1 新增的
    }
    // RED 指标
    start := time.Now()
    logs, err := p.ethClient.FilterLogs(ctx, query)
    if p.redRecorder != nil {
        p.redRecorder.RecordRPCCall("eth_getLogs", p.ChainID(), time.Since(start))
    }
    return logs, nil
}
```

RPC 调用: `eth_getLogs`
返回值示例:
```json
{
  "address": "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
  "topics": [
    "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
    "0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266",
    "0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8"
  ],
  "data": "0x0000000000000000000000000000000000000000000000000000000000000064",
  "blockNumber": "0x4",
  "transactionHash": "0x71a8485817f7cd8957b67de066b9d334d995900f9b42968123d82962080fc517"
}
```

---

## Step 4: 日志 → BlockchainEvent 转换

**文件**: `pkg/plugins/pullers/https_jsonrpc_puller.go:568`

```go
func (p *HTTPSJSONRPCPuller) ethLogToEvent(log types.Log) (core.BlockchainEvent, error) {
    // 1. 提取 event signature
    topic0 := log.Topics[0]
    
    // 2. 计算 event 名称 (从 ABI 或 known signatures)
    eventName := resolveEventName(topic0)
    
    // 3. 解码参数 (ChainedDecoder)
    decoder := core.NewChainedDecoder()
    decodedData := decoder.Decode(eventName, log.Topics, log.Data)
    
    // 4. 构造 BlockchainEvent
    return core.BlockchainEvent{
        EventHash:      topic0.Hex(),
        EventName:      eventName,
        BlockNumber:    log.BlockNumber,
        TxHash:         log.TxHash,
        ContractAddress: log.Address,
        DecodedData:    decodedData,
        ChainID:        p.ChainID(),
        Status:         core.EventStatusConfirmed,
    }, nil
}
```

### ChainedDecoder 三阶段解码

**文件**: `pkg/core/chained_decoder.go:44`

```go
func (d *ChainedDecoder) Decode(eventName string, topics []common.Hash, data []byte) map[string]any {
    // Strategy 1: Runtime ABIs (通过 RegisterABI 注册的)
    // → 精确解码，知道每个参数的类型
    if result := d.decodeFromRegistered(eventName, topics, data); result != nil {
        return result
    }
    // Strategy 2: Known ABIs (event_abi_defs.go 中的标准事件)
    if result := DecodeEventData(eventName, topics, data); result != nil {
        return result
    }
    // Strategy 3: Raw hex fallback（所有数据保持 hex 字符串）
    return d.rawHexFallback(eventName, topics, data)
}
```

**关键设计**: 永不 panic。即使 data 是随机的畸形字节，也返回 raw hex fallback。

---

## Step 5: EventBus 发布

**文件**: `pkg/plugins/pullers/data_puller.go:270`

```go
func (p *BaseDataPullerPlugin) PublishEvent(event core.BlockchainEvent) {
    if err := p.eventBus.Publish(ctx, core.TopicBlockchainEvents, event); err != nil {
        p.LogError("failed to publish event", "error", err)
    }
}
```

### EventBus 内部

**文件**: `pkg/core/eventbus.go:87`

```go
func (b *DefaultEventBus) Publish(ctx context.Context, topic string, event any) error {
    // 1. 从 workerPool 获取一个 slot（16 个 slot 的 semaphore）
    b.workerPool <- struct{}{}
    
    // 2. 异步派发给所有 subscriber
    b.wg.Add(1)
    go func() {
        defer b.wg.Done()
        defer func() { <-b.workerPool }()
        handler(event)
    }()
    
    // Publish 立即返回，不等待 handler 执行完毕
}
```

**重要**: `Publish` 是同步派发+异步执行——它获取 worker slot 后启动 goroutine 立即返回。所有 16 个 slot 满时 Publish 阻塞（背压）。

---

## Step 6: 事件处理

**文件**: `pkg/services/processor/event_processor.go`

```go
func (p *EventProcessor) ProcessEvent(ctx context.Context, event core.BlockchainEvent) error {
    // 1. 幂等性检查: 是否已处理过？
    key := idempotencyKey(event)
    if p.idempotency.IsProcessed(key) {
        return nil  // 已处理，跳过
    }
    
    // 2. 存储事件
    if err := p.store.StoreEvent(ctx, event); err != nil {
        return err
    }
    
    // 3. 标记已处理
    p.idempotency.MarkProcessed(key)
    return nil
}
```

### 幂等性

**文件**: `pkg/services/processor/idempotency.go`

**去重键**: `chain_id:block_number:tx_hash:log_index`

如果同一笔事件被拉了两次（如重启后 checkpoint 回退），第二次会被幂等性检查跳过。

---

## Step 7: Shared Runtime (Shadow Mode)

**文件**: `pkg/services/indexing/chain_indexer.go:134`

```go
func (dci *DefaultChainIndexer) IndexBlocks(ctx context.Context, events []BlockchainEvent) error {
    // Legacy path: 直接写入存储
    dci.legacyOwnedEvents += len(events)
    
    // Shadow path: 转发到 Shared Runtime
    if dci.sharedRuntime != nil {
        dci.sharedRuntime.ProcessBatch(ctx, dci.chainID, events)
        dci.shadowOwnedEvents += int64(len(events))
        dci.legacyOwnedEvents -= int64(len(events))  // 所有权转移
    }
}
```

**Shadow Mode 目的**: 在不中断现有数据管道的情况下验证新 runtime 的正确性。指标中同时暴露 `shadow_owned_events` 和 `legacy_owned_events`。

---

## Step 8: API 查询

**文件**: `pkg/plugins/api/event_query_handler.go`

```go
func (h *EventQueryHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
    // 1. 解析过滤器 (chain, contract, eventName, fromBlock, toBlock, limit)
    filter := parseQueryFilter(r)
    
    // 2. 查询事件存储
    events, err := h.eventStore.QueryEvents(ctx, filter)
    if err != nil {
        WriteErrorEnvelope(w, MapErrorToAPIError(err))
        return
    }
    
    // 3. 响应
    WriteEnvelope(w, http.StatusOK, QueryResponse{Events: events})
}
```

API 响应格式:
```json
{
  "data": { "events": [...] },
  "meta": { "timestamp": 1778679315 }
}
```

---

## 完整文件顺序

要在 delve 中追踪完整路径，按此顺序设置断点：

```
1. cmd/monolithic/chainpulse/main.go:333    → puller runtime 创建
2. pkg/plugins/pullers/https_jsonrpc_puller.go:299  → Poll 循环进入
3. pkg/plugins/pullers/https_jsonrpc_puller.go:483  → getLatestBlockNumber
4. pkg/plugins/pullers/https_jsonrpc_puller.go:499  → getLogs (eth_getLogs)
5. pkg/plugins/pullers/https_jsonrpc_puller.go:568  → ethLogToEvent
6. pkg/core/eventbus.go:87                          → Publish
7. pkg/services/processor/event_processor.go        → ProcessEvent
8. pkg/plugins/api/event_query_handler.go            → API 查询
```

## 知识检查

看完这段追踪后，你应该能回答：

1. Poll 循环的间隔是多少？为什么用 select + ticker 而不是 time.Sleep？
2. eth_getLogs 为什么分批查询？每批多大？
3. ChainedDecoder 的三个解码策略是什么？顺序如何？
4. EventBus 的 Publish 是同步还是异步？16 个 slot 满了会发生什么？
5. 幂等性用什么做去重键？
6. Shadow Mode 中 shadow_owned_events 和 legacy_owned_events 的区别是什么？
7. API 响应的标准格式是什么？
