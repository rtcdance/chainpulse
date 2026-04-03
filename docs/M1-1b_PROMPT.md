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
- 修复: 在拉取循环中集成 `RetryLogic`

#### 断裂 12: ABI 解码未接入 Indexer
- 蓝图要求: §3.2 — Indexer 职责包含 **ABI 解码**
- 当前: 事件以原始 bytes 存储，没有解码
- 修复: 在 ProcessBatch 中接入 `decoder.EventDecoder`

#### 断裂 13: ABI 平滑升级未接入
- 蓝图要求: §3.2 — `contract_manager.go` 多版本 ABI 并存，按 block 范围路由解码逻辑
- 当前: EventDecoder 没有接入 ContractManager
- 修复: 使用 `ContractManager` 管理多版本 ABI，按 block 范围选择正确的 ABI 解码

#### 断裂 14: 幂等写入缺失
- 蓝图要求: §3.2 — 基于 `(chain_id, tx_hash, log_index)` 唯一键去重
- 当前: 重复拉取会导致重复数据
- 修复: 在 EventSink.Persist 中接入 `processor.IdempotencyService`

#### 断裂 15: 背压控制缺失 + Reorg 通知缺失
- 蓝图要求: §3.1 — MQ 满时 Puller 暂停拉取；§3.5 — RollbackEvents 成功后发 `reorg_events` 通知
- 当前: 无背压控制，无 reorg 通知
- 修复: EventBus 添加 `IsBackpressured()` 方法；RollbackEvents 成功后通过 EventBus 发布 `reorg_events`

### 完整数据流（修复后）

```
for each chain:
  go func() {
    // 蓝图 §3.1: checkpoint 从文件加载
    checkpoint := loadCheckpointFromFile(checkpointFile)

    // 蓝图 §3.1: 指数退避重试
    retry := resilience.NewRetryLogic(3, 1*time.Second, 30*time.Second, logger)

    for {
      // 蓝图 §3.1: 背压控制
      if eventBus.IsBackpressured("blockchain-events") {
        sleep(5s)
        continue
      }

      // 蓝图 §3.1: 指数退避重试
      events, err := retry.ExecuteWithRetry(func() ([]core.BlockchainEvent, error) {
        return puller.PullEvents(ctx, fromBlock, toBlock)
      })

      // 蓝图 §3.5: Reorg 检测 + 通知
      if len(events) > 0 && events[0].BlockHash != lastBlockHash[chainID] {
        reorgHandler.DetectReorg(ctx, chainID, fromBlock, ...)
        reorgHandler.HandleReorg(ctx, ...)
        reorgHandler.RollbackEvents(ctx, fromBlock)
        eventBus.Publish("reorg_events", ReorgInfo{ChainID: chainID, FromBlock: fromBlock})
      }

      eventBus.Publish("blockchain-events", events)

      // 蓝图 §3.2: ABI 解码 + 平滑升级
      decodedEvents := contractManager.DecodeBatch(events)  // 按 block 选 ABI

      // 蓝图 §3.2: 幂等检查
      uniqueEvents := idempotencyService.FilterDuplicates(decodedEvents)

      // 蓝图 §3.2: Indexer 消费 + 批量写入
      envelopes := toEventEnvelopes(uniqueEvents)
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
修复上述 6 个断裂，使单体模式具备蓝图要求的全部容错能力。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（35 个包全部 PASS）
- [ ] `make vet` 通过
- [ ] `make run-monolithic` 启动后不 panic

#### 蓝图一致性
- [ ] **指数退避重试**（蓝图 §3.1: max 3 retries, 1s-30s backoff）
- [ ] **Checkpoint 落盘**（蓝图 §3.1: 写入文件，重启后从断点续拉）
- [ ] **背压控制**（蓝图 §3.1: MQ 满时暂停拉取）
- [ ] **ABI 解码 + 平滑升级**（蓝图 §3.2: EventDecoder + ContractManager 多版本 ABI）
- [ ] **幂等写入**（蓝图 §3.2: 基于唯一键去重）
- [ ] **批量写入**（蓝图 §3.2: BatchStoreEvents）
- [ ] **Reorg 通知**（蓝图 §3.5: RollbackEvents 成功后发 reorg_events）

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，§3.1 + §3.2 + §3.5**
- `cmd/monolithic/chainpulse/main.go` — Composition Root（M1-1a 已修改）
- `pkg/services/resilience/retry_logic.go` — 指数退避重试
- `pkg/services/decoder/event_decoder.go` — EventDecoder
- `pkg/services/decoder/contract_manager.go` — ContractManager（多版本 ABI）
- `pkg/services/processor/idempotency.go` — IdempotencyService
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler
- `pkg/infrastructure/data/block_height_tracker.go` — Checkpoint 追踪
- `pkg/core/eventbus.go` — EventBus

### 修复步骤

**Step 1: 创建 EventBus 背压方法**
```
文件: pkg/core/eventbus.go
添加 IsBackpressured(topic string) bool 方法:
  检查订阅者 chan 缓冲区使用率，超过 80% 返回 true
```

**Step 2: 在拉取循环中集成容错**
```
文件: cmd/monolithic/chainpulse/main.go
在 M1-1a 的 goroutine 循环中加入:
  1. 指数退避重试包装 PullEvents
  2. 背压控制: if eventBus.IsBackpressured() { sleep; continue }
  3. Checkpoint 从文件加载
  4. ABI 解码: contractManager.DecodeBatch(events)
  5. 幂等检查: idempotencyService.FilterDuplicates(decodedEvents)
  6. Reorg 通知: eventBus.Publish("reorg_events", reorgInfo)
  7. Checkpoint 保存到文件
```

**Step 3: 创建 checkpoint 文件读写**
```
新建 .chainpulse/checkpoints/{chainID}.json:
  { "chainID": "ethereum", "lastBlock": 18923456, "lastBlockHash": "0x...", "updatedAt": "..." }
启动时加载，每次拉取后保存。
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
```
