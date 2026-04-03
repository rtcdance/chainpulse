# M1-1b: 修复单体容错层（严格按 ARCHITECTURE_v1.md 蓝图）

> 这是 M1-1 的第二阶段。**前提: M1-1a 已完成且验证通过。**
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M1-1b - 修复单体容错层

### 背景
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- M1-1a 状态: 基础数据链路已通（Puller → EventBus → Indexer → DB → Query → API）

### 蓝图对容错的要求

| 蓝图章节 | 蓝图要求 |
|---|---|
| §3.1 Puller 容错 | 指数退避重试（max 3, 1s-30s）+ checkpoint 落盘 + 背压控制 |
| §3.2 Indexer 容错 | **幂等写入**（唯一键去重）+ 批量写入 + **ABI 平滑升级**（多版本 ABI） |
| §3.5 Reorg 容错 | 最终性确认 + 原子回滚 + **reorg_events 通知** |

### 当前状态：6 个断裂点

#### 断裂 10: Checkpoint 未落盘
- 蓝图要求: §3.1 — `block_height_tracker.go` checkpoint **落盘**，重启后从断点续拉
- 当前: 无 checkpoint 机制
- 修复: checkpoint 写入 `.chainpulse/checkpoints/{chainID}.json` 文件，启动时加载

#### 断裂 11: 没有指数退避重试
- 蓝图要求: §3.1 — `pkg/services/resilience/retry_logic.go`，最大重试 3 次，初始间隔 1s，上限 30s
- 当前: RPC 错误直接返回，没有重试
- 修复: 在拉取循环中集成 RetryExecutor + DefaultRetryPolicy

#### 断裂 12: ABI 解码未接入 Indexer
- 蓝图要求: §3.2 — Indexer 职责包含 **ABI 解码**
- 当前: 事件以原始 bytes 存储，没有 ABI 解码
- 修复: 在 Puller 的 logToEvent 中接入 EventDecoder，使用 ContractManager 按 block 范围选择正确 ABI 解码
- **注意**: 当前 Puller.PullEvents 返回 `[]core.BlockchainEvent`（已通过 logToEvent 转换）。EventDecoder.DecodeEventBatch 需要 `[]*types.Log`。因此 ABI 解码应该在 Puller 内部的 getLogs→logToEvent 流程中接入，而不是在 main.go 的循环中

#### 断裂 13: ABI 平滑升级未接入
- 蓝图要求: §3.2 — `contract_manager.go` 多版本 ABI 并存，按 block 范围路由解码逻辑
- 当前: Puller 的 logToEvent 没有使用 ContractManager
- 修复: 在 Puller 中创建 ContractManager，注册多版本 ABI，logToEvent 时根据 contract address 和 block number 选择正确 ABI

#### 断裂 14: 幂等写入缺失
- 蓝图要求: §3.2 — 基于 `(chain_id, tx_hash, log_index)` 唯一键去重
- 当前: 重复拉取会导致重复数据
- 修复: 在 ProcessBatch 中使用 DefaultIdempotencyService 的 GenerateHash + IsDuplicate + MarkProcessed 循环去重

#### 断裂 15: 背压控制缺失 + Reorg 通知缺失
- 蓝图要求: §3.1 — MQ 满时 Puller 暂停拉取；§3.5 — RollbackEvents 成功后发 `reorg_events` 通知
- 当前: 无背压控制，无 reorg 通知
- 修复: EventBus 添加 `IsBackpressured()` 方法；RollbackEvents 成功后通过 EventBus 发布 `reorg_events`

### 完整数据流（修复后）

```
for each chain:
  go func() {
    // 蓝图 §3.1: checkpoint 从文件加载 (os.ReadFile + json.Unmarshal)
    checkpoint := loadCheckpointFromFile(checkpointFile)

    // 蓝图 §3.1: 重试策略 + 执行器
    retryPolicy := resilience.NewDefaultRetryPolicy(
      &resilience.RetryConfig{MaxRetries: 3, InitialBackoff: time.Second, MaxBackoff: 30*time.Second},
      resilience.NewDefaultErrorHandler(logger),
    )
    retryExecutor := resilience.NewRetryExecutor(retryPolicy, logger, metrics)

    for {
      // 蓝图 §3.1: 背压控制
      if eventBus.IsBackpressured("blockchain-events") {
        sleep(5s)
        continue
      }

      // 蓝图 §3.1: 指数退避重试
      var events []core.BlockchainEvent
      err := retryExecutor.Execute(ctx, func() error {
        var err error
        events, err = puller.PullEvents(ctx, fromBlock, toBlock)
        return err
      }, "pull_events")

      if err != nil || len(events) == 0 {
        sleep(10s)
        continue
      }

      // 蓝图 §3.5: Reorg 检测 + 通知
      if events[0].BlockHash != lastBlockHash[chainID] && lastBlock > 0 {
        reorgHandler.DetectReorg(ctx, chainID, fromBlock, ...)
        reorgHandler.HandleReorg(ctx, ...)
        reorgHandler.RollbackEvents(ctx, fromBlock)
        eventBus.Publish("reorg_events", ReorgInfo{ChainID: chainID, FromBlock: fromBlock})
      }

      // 蓝图 §3.2: 幂等检查
      idempotencySvc := processor.NewDefaultIdempotencyService(logger, metrics)
      uniqueEvents := make([]core.BlockchainEvent, 0, len(events))
      for _, event := range events {
        hash, _ := idempotencySvc.GenerateHash(&event)
        isDup, _ := idempotencySvc.IsDuplicate(hash)
        if !isDup {
          idempotencySvc.MarkProcessed(hash)
          uniqueEvents = append(uniqueEvents, event)
        }
      }

      eventBus.Publish("blockchain-events", uniqueEvents)

      // 蓝图 §3.2: Indexer 消费 + 批量写入
      envelopes := toEventEnvelope(&uniqueEvents[i])  // 使用 chain_indexer.go 已有的 toEventEnvelope 函数
      chainIndexer.ProcessBatch(ctx, chainID, envelopes)

      // 蓝图 §3.1: checkpoint 落盘到文件
      checkpoint = toBlock
      saveCheckpointToFile(checkpointFile, checkpoint)
      lastBlockHash[chainID] = events[len(events)-1].BlockHash

      sleep(10s)
    }
  }
```

### 目标
修复上述 6 个断裂，使单体模式具备以下容错能力：
1. **Puller**: 指数退避重试（max 3, 1s-30s）+ checkpoint 文件落盘 + 背压控制
2. **Indexer**: 幂等写入（唯一键去重）+ 批量写入
3. **Reorg**: RollbackEvents 成功后发 reorg_events 通知
4. **ABI 平滑升级**: Puller 的 logToEvent 接入 ContractManager 多版本 ABI

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性
- [ ] **指数退避重试**（蓝图 §3.1: max 3 retries, 1s-30s backoff，使用 RetryExecutor）
- [ ] **Checkpoint 落盘**（蓝图 §3.1: 写入 JSON 文件，重启后从断点续拉）
- [ ] **背压控制**（蓝图 §3.1: EventBus.IsBackpressured 满时暂停拉取）
- [ ] **幂等写入**（蓝图 §3.2: GenerateHash + IsDuplicate + MarkProcessed 循环去重）
- [ ] **批量写入**（蓝图 §3.2: MockDB.BatchStoreEvents 链路通）
- [ ] **Reorg 通知**（蓝图 §3.5: RollbackEvents 成功后 eventBus.Publish("reorg_events")）
- [ ] **ABI 平滑升级**（蓝图 §3.2: Puller 的 logToEvent 接入 ContractManager）

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，§3.1 + §3.2 + §3.5**
- `cmd/monolithic/chainpulse/main.go` — Composition Root（M1-1a 已修改）
- `pkg/services/resilience/retry_logic.go` — 重试 API:
  - `NewDefaultRetryPolicy(config *RetryConfig, errorHandler *ErrorHandler) *DefaultRetryPolicy`
  - `NewRetryExecutor(policy RetryPolicy, logger core.Logger, metrics core.MetricsCollector) *RetryExecutor`
  - `executor.Execute(ctx context.Context, operation func() error, source string) error` — 返回 error，结果通过闭包变量捕获
- `pkg/services/resilience/error_handler.go` — ErrorHandler:
  - `NewErrorHandler(logger core.Logger, metricsCollector core.MetricsCollector) *ErrorHandler`
- `pkg/services/processor/idempotency.go` — 幂等 API:
  - `NewDefaultIdempotencyService(logger, metrics) *DefaultIdempotencyService`
  - `svc.GenerateHash(event *core.BlockchainEvent) (string, error)`
  - `svc.IsDuplicate(hash string) (bool, error)`
  - `svc.MarkProcessed(hash string) error`
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler
- `pkg/services/decoder/contract_manager.go` — ContractManager（多版本 ABI）:
  - `NewContractManager(logger) *ContractManager`
  - `cm.LoadContractABI(name, address, abi) error`
  - `cm.GetABI(name) (abi.ABI, error)`
- `pkg/services/decoder/event_decoder.go` — EventDecoder:
  - `NewEventDecoder(contractManager, logger) *EventDecoder`
  - `ed.DecodeEvent(rawEvent *types.Log, contractABI abi.ABI) (*DecodedEvent, error)`
  - `ed.DecodeEventBatch(rawEvents []*types.Log, contractABI abi.ABI) ([]*DecodedEvent, error)`
- `pkg/plugins/pullers/https_jsonrpc_puller.go` — Puller（需修改 logToEvent 接入 ABI 解码）
- `pkg/core/eventbus.go` — EventBus
- `pkg/infrastructure/data/block_height_tracker.go` — Checkpoint 追踪

### 修复步骤

**Step 1: 创建 EventBus 背压方法**
```
文件: pkg/core/eventbus.go
添加 IsBackpressured(topic string) bool 方法:
  遍历该 topic 的所有订阅者 chan，检查 len(ch)/cap(ch) > 0.8 返回 true
```

**Step 2: 修改 Puller 的 logToEvent 接入 ContractManager（ABI 平滑升级）**
```
文件: pkg/plugins/pullers/https_jsonrpc_puller.go
1. 在 HTTPSJSONRPCPuller 结构体中添加:
   contractManager *decoder.ContractManager
   eventDecoder    *decoder.EventDecoder
2. 在 NewHTTPSJSONRPCPuller 中初始化:
   cm := decoder.NewContractManager(logger)
   // 注册已知合约 ABI（ERC20 等）
   cm.LoadContractABI("erc20", "0x...", erc20ABI)
   p.contractManager = cm
   p.eventDecoder = decoder.NewEventDecoder(cm, logger)
3. 修改 logToEvent 方法:
   - 根据 log.Address 查找对应的 contract ABI
   - 如果有匹配的 ABI，使用 eventDecoder.DecodeEvent(log, abi) 解码
   - 将解码后的 eventName、EventData 填入 BlockchainEvent
   - 如果没有匹配的 ABI，保持现有的简单转换逻辑
```

**Step 3: 在拉取循环中集成 Puller 容错（重试 + 背压 + checkpoint）**
```
文件: cmd/monolithic/chainpulse/main.go
在 M1-1a 的 goroutine 循环中修改:

循环前初始化:
  checkpointFile := fmt.Sprintf(".chainpulse/checkpoints/%s.json", chainID)
  var checkpoint uint64
  if data, err := os.ReadFile(checkpointFile); err == nil {
    var cp struct{ LastBlock uint64 `json:"lastBlock"` }
    json.Unmarshal(data, &cp)
    checkpoint = cp.LastBlock
  }

  retryPolicy := resilience.NewDefaultRetryPolicy(
    &resilience.RetryConfig{MaxRetries: 3, InitialBackoff: time.Second, MaxBackoff: 30*time.Second},
    resilience.NewErrorHandler(logger, metrics),  // 注意: NewErrorHandler，不是 NewDefaultErrorHandler
  )
  retryExecutor := resilience.NewRetryExecutor(retryPolicy, logger, metrics)

循环内:
  1. 背压控制:
     if eventBus.IsBackpressured("blockchain-events") { time.Sleep(5*time.Second); continue }

  2. 指数退避重试:
     var events []core.BlockchainEvent
     err := retryExecutor.Execute(ctx, func() error {
       var err error
       events, err = puller.PullEvents(ctx, fromBlock, toBlock)
       return err
     }, "pull_events")
     if err != nil {
       time.Sleep(10 * time.Second)
       continue  // 重试耗尽，跳过本轮
     }

  3. 循环末尾保存 checkpoint:
     checkpointData, _ := json.Marshal(map[string]uint64{"lastBlock": toBlock})
     os.MkdirAll(filepath.Dir(checkpointFile), 0755)
     os.WriteFile(checkpointFile, checkpointData, 0644)
```

**Step 4: 创建 checkpoint 文件读写**
```
不需要单独创建函数。直接在 Step 3 中使用内联代码:

加载:
  var checkpoint uint64
  if data, err := os.ReadFile(checkpointFile); err == nil {
    var cp struct{ LastBlock uint64 `json:"lastBlock"` }
    json.Unmarshal(data, &cp)
    checkpoint = cp.LastBlock
  }

保存:
  checkpointData, _ := json.Marshal(map[string]uint64{"lastBlock": toBlock})
  os.MkdirAll(filepath.Dir(checkpointFile), 0755)
  os.WriteFile(checkpointFile, checkpointData, 0644)
```

**Step 5: 在拉取循环中集成幂等写入**
```
文件: cmd/monolithic/chainpulse/main.go
在 eventBus.Publish 之前加入:

  idempotencySvc := processor.NewDefaultIdempotencyService(logger, metrics)
  uniqueEvents := make([]core.BlockchainEvent, 0, len(events))
  for _, event := range events {
    hash, err := idempotencySvc.GenerateHash(&event)
    if err != nil { continue }
    isDup, err := idempotencySvc.IsDuplicate(hash)
    if err != nil || isDup { continue }
    idempotencySvc.MarkProcessed(hash)
    uniqueEvents = append(uniqueEvents, event)
  }
  // 使用 uniqueEvents 替代 events 进行后续处理
```

**Step 6: 在拉取循环中集成 Reorg 通知**
```
文件: cmd/monolithic/chainpulse/main.go
在 M1-1a 的 goroutine 循环中，reorgHandler.RollbackEvents 之后加入:
  eventBus.Publish("reorg_events", map[string]interface{}{
    "chain_id":   chainID,
    "from_block": fromBlock,
    "timestamp":  time.Now().Unix(),
  })
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- 不要试图修复 16 处依赖违反
- **必须与 ARCHITECTURE_v1.md 蓝图一致**
- **本阶段只做容错，不做可观测性、API 限流、缓存击穿防护、分布式追踪**
- **API 使用必须与上述参考文件中的实际签名完全一致，不要假设不存在的方法**

### 验证步骤
```bash
make build
make test-unit
make vet
# 验证 checkpoint 落盘
make run-monolithic &
sleep 60
ls .chainpulse/checkpoints/  # 应该有 checkpoint 文件
# 验证重启后从断点续拉
kill %1
make run-monolithic &
# 日志应显示从上次 checkpoint 继续，不是从头开始
# 验证幂等: 重复拉取同一区块，不应产生重复数据
# 验证重试: 临时停止 RPC 节点，Puller 应重试 3 次后失败，恢复后继续
```
